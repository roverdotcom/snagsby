package resolvers

import (
	"fmt"
	"os"

	"github.com/roverdotcom/snagsby/pkg/config"

	"sigs.k8s.io/yaml"
)

type manifestSecretsConnector interface {
	GetSecrets(keys []*string) (map[string]string, []error)
}

type ManifestItems struct {
	Items []*ManifestItem
}

type ManifestItem struct {
	Name string `json:"name"`
	Env  string `json:"env"`
}

type ManifestResolver struct {
	connector manifestSecretsConnector
}

func (m *ManifestResolver) resolveManifestItems(manifestItems *ManifestItems, result *Result) {

	numItems := len(manifestItems.Items)
	secretKeys := make([]*string, numItems)
	envVarSecretMap := make(map[string]string)

	for i, item := range manifestItems.Items {
		secretKeys[i] = &item.Name
		envVarSecretMap[item.Name] = item.Env
	}

	secrets, errors := m.connector.GetSecrets(secretKeys)
	for _, err := range errors {
		result.AppendError(err)
	}

	for key, value := range secrets {
		if envVar, ok := envVarSecretMap[key]; ok {
			result.AppendItem(envVar, value)
		}
	}
}

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

	s.resolveManifestItems(&manifestItems, result)

	return result
}
