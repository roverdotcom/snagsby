package resolvers

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/roverdotcom/snagsby/pkg/config"

	"sigs.k8s.io/yaml"
)

type ManifestItems struct {
	Items []*ManifestItem
}

type ManifestItem struct {
	Name string `json:"name"`
	Env  string `json:"env"`
}

type manifestResult struct {
	Value string
	Error error
	Item  *ManifestItem
}

func manifestWorker(svc *secretsmanager.Client, jobs <-chan *ManifestItem, resultChan chan<- *manifestResult) {
	for manifestItem := range jobs {
		result := &manifestResult{Item: manifestItem}
		value, err := getSecretValue(svc, manifestItem)
		result.Error = err
		result.Value = value
		resultChan <- result
	}
}

func getSecretValue(svc *secretsmanager.Client, manifestItem *ManifestItem) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: &manifestItem.Name,
	}
	getSecret, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		return "", err
	}
	return *getSecret.SecretString, nil
}

func resolveManifestItems(manifestItems *ManifestItems, result *Result) {
	cfg, err := getAwsConfig(awsConfig.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
	}))
	if err != nil {
		result.AppendError(err)
		return
	}
	svc := secretsmanager.NewFromConfig(cfg)
	numItems := len(manifestItems.Items)
	resultsChan := make(chan *manifestResult, numItems)
	jobsChan := make(chan *ManifestItem, numItems)
	for _, item := range manifestItems.Items {
		jobsChan <- item
	}

	// Boot up 20 workers
	numWorkers := 20
	for i := 0; i < numWorkers; i++ {
		go manifestWorker(svc, jobsChan, resultsChan)
	}
	close(jobsChan)

	for i := 0; i < numItems; i++ {
		getResult := <-resultsChan
		if getResult.Error != nil {
			result.AppendError(getResult.Error)
		} else {
			result.AppendItem(getResult.Item.Env, getResult.Value)
		}
	}
	close(resultsChan)
}

type ManifestResolver struct{}

func (s *ManifestResolver) Resolve(source *config.Source) *Result {
	result := &Result{Source: source}
	filePath := fmt.Sprintf("%s%s", source.URL.Host, source.URL.Path)
	f, err := os.ReadFile(filePath)
	if err != nil {
		result.AppendError(err)
		return result
	}
	var manifestItems ManifestItems
	err = yaml.Unmarshal(f, &manifestItems)
	if err != nil {
		result.AppendError(err)
		return result
	}

	resolveManifestItems(&manifestItems, result)

	return result
}
