package resolvers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/roverdotcom/snagsby/pkg/config"
)

type EnvFileItem struct {
	Key             string
	Value           string
	NeedsResolution bool
}

type envFileResult struct {
	Value string
	Error error
	Item  *EnvFileItem
}

func envFileWorker(svc *secretsmanager.Client, jobs <-chan *EnvFileItem, resultChan chan<- *envFileResult) {
	for item := range jobs {
		result := &envFileResult{Item: item}

		// Extract the secret name from sm:// reference
		secretName := strings.TrimPrefix(item.Value, "sm://")

		input := &secretsmanager.GetSecretValueInput{
			SecretId: &secretName,
		}
		getSecret, err := svc.GetSecretValue(context.TODO(), input)
		if err != nil {
			result.Error = err
		} else {
			result.Value = *getSecret.SecretString
		}
		resultChan <- result
	}
}

func parseEnvFile(filePath string) ([]*EnvFileItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []*EnvFileItem
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Skip if key is empty
		if key == "" {
			continue
		}

		// Remove surrounding quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
		                        (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		item := &EnvFileItem{
			Key:             key,
			Value:           value,
			NeedsResolution: strings.HasPrefix(value, "sm://"),
		}

		items = append(items, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func resolveEnvFileItems(items []*EnvFileItem, result *Result) {
	// Separate items that need AWS resolution from direct values
	var awsItems []*EnvFileItem
	for _, item := range items {
		if item.NeedsResolution {
			awsItems = append(awsItems, item)
		} else {
			// Direct value - add immediately
			result.AppendItem(item.Key, item.Value)
		}
	}

	// If no AWS items, we're done
	if len(awsItems) == 0 {
		return
	}

	// Initialize AWS client for items that need resolution
	cfg, err := getAwsConfig(awsConfig.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
	}))
	if err != nil {
		result.AppendError(err)
		return
	}
	svc := secretsmanager.NewFromConfig(cfg)

	numAwsItems := len(awsItems)
	resultsChan := make(chan *envFileResult, numAwsItems)
	jobsChan := make(chan *EnvFileItem, numAwsItems)

	for _, item := range awsItems {
		jobsChan <- item
	}

	// Boot up 20 workers
	numWorkers := 20
	for range numWorkers {
		go envFileWorker(svc, jobsChan, resultsChan)
	}
	close(jobsChan)

	// Collect results
	for range numAwsItems {
		getResult := <-resultsChan
		if getResult.Error != nil {
			result.AppendError(getResult.Error)
		} else {
			result.AppendItem(getResult.Item.Key, getResult.Value)
		}
	}
	close(resultsChan)
}

type FileResolver struct{}

func (f *FileResolver) Resolve(source *config.Source) *Result {
	result := &Result{Source: source}

	// Construct file path from URL
	filePath := fmt.Sprintf("%s%s", source.URL.Host, source.URL.Path)

	// Parse the env file
	items, err := parseEnvFile(filePath)
	if err != nil {
		result.AppendError(err)
		return result
	}

	// Resolve items (both direct values and AWS secrets)
	resolveEnvFileItems(items, result)

	return result
}
