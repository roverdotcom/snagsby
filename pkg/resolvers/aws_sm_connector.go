package resolvers

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/roverdotcom/snagsby/pkg/config"
)

// SecretValueGetter defines the interface for getting secret values
type SecretValueGetter interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

var smConcurrency int

func init() {
	// Pull concurrency settings
	getConcurrency, hasSetting := os.LookupEnv("SNAGSBY_SM_CONCURRENCY")
	if hasSetting {
		i, err := strconv.Atoi(getConcurrency)
		if err == nil && i >= 0 {
			smConcurrency = i
		}
	}
}

// Concurrency work
type smMessage struct {
	Source      *config.Source
	Name        *string
	Result      string
	Error       error
	IsRecursive bool
}

func NewSecretsManager(sourceURL *url.URL) (*secretsmanager.Client, error) {

	cfg, err := getAwsConfig(awsConfig.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
	}))

	if err != nil {
		return nil, err
	}

	region := sourceURL.Query().Get("region")
	if region != "" {
		cfg.Region = region
	}
	svc := secretsmanager.NewFromConfig(cfg)

	return svc, nil
}

func smWorker(jobs <-chan *smMessage, results chan<- *smMessage, svc SecretValueGetter) {
	for job := range jobs {
		sourceURL := job.Source.URL
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
		getSecret, err := svc.GetSecretValue(ctx, input)
		if err != nil {
			job.Error = err
		} else {
			job.Result = *getSecret.SecretString
		}
		results <- job
	}
}

// getSecrets handles concurrent retrieval of secrets from secrets manager
func getSecrets(source *config.Source, svc SecretValueGetter, keys []*string) (map[string]string, []error) {
	jobs := make(chan *smMessage, len(keys))
	results := make(chan *smMessage, len(keys))

	numWorkers := smConcurrency
	if numWorkers <= 0 {
		numWorkers = len(keys)
	}

	// Start workers
	for w := 0; w < numWorkers; w++ {
		go smWorker(jobs, results, svc)
	}

	// Send jobs
	for _, key := range keys {
		jobs <- &smMessage{Source: source, Name: key}
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
