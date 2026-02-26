package resolvers

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/roverdotcom/snagsby/pkg/config"
	"github.com/roverdotcom/snagsby/pkg/connectors"

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

func resolveManifestItems(source *config.Source, manifestItems *ManifestItems, result *Result) {
	smConnector, err := connectors.NewSecretsManagerConnector(source)
	if err != nil {
		result.AppendError(err)
		return
	}

	numItems := len(manifestItems.Items)
	secretKeys := make([]*string, numItems)
	envVarSecretMap := make(map[string]string)

	for i, item := range manifestItems.Items {
		secretKeys[i] = &item.Name
		envVarSecretMap[item.Name] = item.Env
	}

	secrets, errors := smConnector.GetSecrets(secretKeys)
	for _, err := range errors {
		result.AppendError(err)
	}

	for key, value := range secrets {
		if envVar, ok := envVarSecretMap[key]; ok {
			result.AppendItem(envVar, value)
		}
	}
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

	resolveManifestItems(source, &manifestItems, result)

	return result
}
