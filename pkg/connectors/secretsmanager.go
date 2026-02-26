package connectors

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/roverdotcom/snagsby/pkg/clients"
	"github.com/roverdotcom/snagsby/pkg/config"
)

type ListSecretsAPIClient interface {
	ListSecrets(context.Context, *secretsmanager.ListSecretsInput, ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
}

type GetSecretValueAPIClient interface {
	GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type SecretsManagerAPIClient interface {
	ListSecretsAPIClient
	GetSecretValueAPIClient
}

type SecretsManagerConnector struct {
	secretsmanagerClient SecretsManagerAPIClient
	source               *config.Source
}

func NewSecretsManagerConnector(source *config.Source) (*SecretsManagerConnector, error) {
	secretsManagerClient, err := clients.NewSecretsManagerClient(source.URL)
	if err != nil {
		return nil, err
	}
	return &SecretsManagerConnector{secretsmanagerClient: secretsManagerClient, source: source}, nil
}

func getConcurrencyOrDefault(keyLength int) int {
	// Pull concurrency settings
	getConcurrency, hasSetting := os.LookupEnv("SNAGSBY_SM_CONCURRENCY")
	if hasSetting {
		i, err := strconv.Atoi(getConcurrency)
		// Concurrency should never be 0 or it will cause the process to deadlock (no background workers)
		if err == nil && i > 0 {
			return i
		}
	}
	return keyLength
}

// fetchSecretValue retrieves a single secret value with version control
func (sm *SecretsManagerConnector) fetchSecretValue(secretName *string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	input := &secretsmanager.GetSecretValueInput{
		SecretId: secretName,
	}

	sourceURL := sm.source.URL
	if versionStage := sourceURL.Query().Get("version-stage"); versionStage != "" {
		input.VersionStage = aws.String(versionStage)
	}
	if versionID := sourceURL.Query().Get("version-id"); versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	getSecret, err := sm.secretsmanagerClient.GetSecretValue(ctx, input)
	if err != nil {
		return "", err
	}

	return *getSecret.SecretString, nil
}

type secretResult struct {
	name  string
	value string
	err   error
}

// GetSecrets handles concurrent retrieval of secrets from secrets manager
func (sm *SecretsManagerConnector) GetSecrets(keys []*string) (map[string]string, []error) {
	keysLength := len(keys)

	if keysLength == 0 {
		return map[string]string{}, nil
	}

	numWorkers := getConcurrencyOrDefault(keysLength)

	// Semaphore to limit concurrency
	sem := make(chan struct{}, numWorkers)

	resultChan := make(chan secretResult, keysLength)
	var wg sync.WaitGroup

	// Launch a goroutine for each key
	for _, key := range keys {
		wg.Add(1)
		go func(k *string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			val, err := sm.fetchSecretValue(k)
			resultChan <- secretResult{name: *k, value: val, err: err}
		}(key)
	}

	// Close results when all goroutines finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	secrets := make(map[string]string)
	var errors []error
	for r := range resultChan {
		if r.err != nil {
			errors = append(errors, r.err)
		} else {
			secrets[r.name] = r.value
		}
	}

	return secrets, errors
}

func (s *SecretsManagerConnector) ListSecrets(prefix string) ([]*string, error) {
	// List secrets that begin with our prefix
	params := &secretsmanager.ListSecretsInput{
		Filters: []types.Filter{
			{
				Key: "name",
				Values: []string{
					prefix,
				},
			},
		},
	}
	secretKeys := []*string{}
	paginator := secretsmanager.NewListSecretsPaginator(s.secretsmanagerClient, params)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return secretKeys, err
		}
		for _, secret := range output.SecretList {
			secretKeys = append(secretKeys, secret.Name)
		}

	}

	return secretKeys, nil
}

// GetSecret retrieves a single secret value
func (sm *SecretsManagerConnector) GetSecret(secretName string) (string, error) {
	return sm.fetchSecretValue(aws.String(secretName))
}
