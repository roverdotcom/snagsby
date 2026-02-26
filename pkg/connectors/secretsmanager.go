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
	secretsManagerClient, err := clients.NewSecretsManager(source.URL)
	if err != nil {
		return nil, err
	}
	return &SecretsManagerConnector{secretsmanagerClient: secretsManagerClient, source: source}, nil
}

func GetConcurrencyOrDefault(keyLength int) int {
	// Pull concurrency settings
	getConcurrency, hasSetting := os.LookupEnv("SNAGSBY_SM_CONCURRENCY")
	if hasSetting {
		i, err := strconv.Atoi(getConcurrency)
		if err == nil && i >= 0 {
			return i
		}
	}
	return keyLength
}

// Concurrency work
type smMessage struct {
	Name        *string
	Result      string
	Error       error
	IsRecursive bool
}

func (sm *SecretsManagerConnector) smWorker(jobs <-chan *smMessage, results chan<- *smMessage) {
	for job := range jobs {
		sourceURL := sm.source.URL
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		input := &secretsmanager.GetSecretValueInput{
			SecretId: job.Name,
		}
		versionStage := sourceURL.Query().Get("version-stage")
		if versionStage != "" {
			input.VersionStage = aws.String(versionStage)
		}
		versionID := sourceURL.Query().Get("version-id")
		if versionID != "" {
			input.VersionId = aws.String(versionID)
		}
		getSecret, err := sm.secretsmanagerClient.GetSecretValue(ctx, input)
		if err != nil {
			job.Error = err
		} else {
			job.Result = *getSecret.SecretString
		}
		results <- job
	}
}

// getSecrets handles concurrent retrieval of secrets from secrets manager
func (sm *SecretsManagerConnector) GetSecrets(keys []*string) (map[string]string, []error) {
	jobs := make(chan *smMessage, len(keys))
	results := make(chan *smMessage, len(keys))

	numWorkers := GetConcurrencyOrDefault(len(keys))

	// Start workers
	for w := 0; w < numWorkers; w++ {
		go sm.smWorker(jobs, results)
	}

	// Send jobs
	// smMessage might not need source because it is part of the struct already
	for _, key := range keys {
		jobs <- &smMessage{Name: key}
	}
	close(jobs)

	secrets := make(map[string]string)
	var errors []error

	for i := 0; i < len(keys); i++ {
		result := <-results
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else {
			secrets[*result.Name] = result.Result
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
	sourceURL := sm.source.URL
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	versionStage := sourceURL.Query().Get("version-stage")
	if versionStage != "" {
		input.VersionStage = aws.String(versionStage)
	}

	versionID := sourceURL.Query().Get("version-id")
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	res, err := sm.secretsmanagerClient.GetSecretValue(ctx, input)
	if err != nil {
		return "", err
	}

	return *res.SecretString, nil
}
