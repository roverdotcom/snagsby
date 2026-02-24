package resolvers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	snagsbyConfig "github.com/roverdotcom/snagsby/pkg/config"
)

func getAwsConfig(optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
	// If the SNAGSBY_LOG_AWS_RETRIES environment variable is truthy
	// we will log retries
	if snagsbyConfig.EnvBool("SNAGSBY_LOG_AWS_RETRIES") {
		optFns = append(optFns, config.WithClientLogMode(aws.LogRetries))
	}
	return config.LoadDefaultConfig(context.TODO(), optFns...)
}

func readJSONString(input string) (map[string]string, error) {
	var f map[string]any
	out := map[string]string{}
	if err := json.Unmarshal([]byte(input), &f); err != nil {
		return out, err
	}
	for k, v := range f {
		k = strings.ToUpper(k)
		switch vv := v.(type) {
		case string:
			out[k] = vv
		case float64:
			out[k] = strconv.FormatFloat(vv, 'f', -1, 64)
		case bool:
			var b string
			if vv {
				b = "1"
			} else {
				b = "0"
			}
			out[k] = b
		}
	}
	return out, nil
}

// secretFetchJob represents a secret to fetch
type secretFetchJob struct {
	SecretID string
}

// secretFetchResult represents the result of fetching a secret
type secretFetchResult struct {
	SecretID string
	Value    string
	Error    error
}

// secretFetchWorker fetches secrets from AWS Secrets Manager
func secretFetchWorker(svc *secretsmanager.Client, jobs <-chan *secretFetchJob, results chan<- *secretFetchResult) {
	for job := range jobs {
		result := &secretFetchResult{SecretID: job.SecretID}

		input := &secretsmanager.GetSecretValueInput{
			SecretId: &job.SecretID,
		}

		getSecret, err := svc.GetSecretValue(context.TODO(), input)
		if err != nil {
			result.Error = err
		} else {
			result.Value = *getSecret.SecretString
		}

		results <- result
	}
}

// BatchFetchSecrets fetches multiple secrets from AWS Secrets Manager concurrently.
// It returns a map of secretID -> secretValue and a slice of errors encountered.
// The concurrency parameter controls the number of concurrent workers (0 = number of secrets).
func BatchFetchSecrets(secretIDs []string, concurrency int) (map[string]string, []error) {
	if len(secretIDs) == 0 {
		return map[string]string{}, nil
	}

	// Get AWS config with retries
	cfg, err := getAwsConfig(config.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
	}))
	if err != nil {
		return nil, []error{err}
	}

	svc := secretsmanager.NewFromConfig(cfg)

	// Determine number of workers
	numWorkers := concurrency
	if numWorkers <= 0 {
		numWorkers = 20 // Default to 20 workers
	}

	// Create channels
	jobs := make(chan *secretFetchJob, len(secretIDs))
	results := make(chan *secretFetchResult, len(secretIDs))

	// Start workers
	for range numWorkers {
		go secretFetchWorker(svc, jobs, results)
	}

	// Queue jobs
	for _, secretID := range secretIDs {
		jobs <- &secretFetchJob{SecretID: secretID}
	}
	close(jobs)

	// Collect results
	secretValues := make(map[string]string)
	var errors []error

	for range len(secretIDs) {
		result := <-results
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else {
			secretValues[result.SecretID] = result.Value
		}
	}
	close(results)

	return secretValues, errors
}
