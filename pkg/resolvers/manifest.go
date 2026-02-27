package resolvers

import (
	"fmt"
	"os"

	"github.com/roverdotcom/snagsby/pkg/config"

	"sigs.k8s.io/yaml"
)

type manifestSecretsConnector interface {
	GetSecrets(keys []string) (map[string]string, []error)
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

func NewManifestResolver(connector manifestSecretsConnector) *ManifestResolver {
	return &ManifestResolver{connector: connector}
}

func (m *ManifestResolver) resolveManifestItems(manifestItems *ManifestItems, result *Result) {

	numItems := len(manifestItems.Items)
	secretKeys := make([]string, numItems)

	for i, item := range manifestItems.Items {
		secretKeys[i] = item.Name
	}

	secrets, errors := m.connector.GetSecrets(secretKeys)
	for _, err := range errors {
		result.AppendError(err)
	}

	// Iterate over manifestItems.Items in order to ensure deterministic behavior
	// when multiple items map to the same env var (last one wins)
	for _, item := range manifestItems.Items {
		if value, ok := secrets[item.Name]; ok {
			result.AppendItem(item.Env, value)
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
