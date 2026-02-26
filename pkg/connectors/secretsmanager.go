package connectors

import (
	"context"
	"os"
	"strconv"
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

// worker processes secret fetch requests from the jobs channel
func (sm *SecretsManagerConnector) worker(jobs <-chan string, results chan<- secretResult) {
	for secretName := range jobs {
		value, err := sm.fetchSecretValue(&secretName)
		results <- secretResult{
			name:  secretName,
			value: value,
			err:   err,
		}
	}
}

// GetSecrets handles concurrent retrieval of secrets from secrets manager
func (sm *SecretsManagerConnector) GetSecrets(keys []*string) (map[string]string, []error) {
	keysLength := len(keys)

	if keysLength == 0 {
		return map[string]string{}, nil
	}

	numWorkers := getConcurrencyOrDefault(keysLength)

	jobs := make(chan string, keysLength)
	results := make(chan secretResult, keysLength)

	// Start worker goroutines
	for w := 0; w < numWorkers; w++ {
		go sm.worker(jobs, results)
	}

	// Send jobs
	for _, key := range keys {
		jobs <- *key
	}
	close(jobs)

	// Collect results
	secrets := make(map[string]string)
	var errors []error
	for i := 0; i < keysLength; i++ {
		result := <-results
		if result.err != nil {
			errors = append(errors, result.err)
		} else {
			secrets[result.name] = result.value
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
