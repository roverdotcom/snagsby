package resolvers

import (
	"fmt"
	"os"

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

func resolveManifestItems(manifestItems *ManifestItems, result *Result) {
	// Build list of secret IDs and mapping to env var names
	secretKeys := make([]*string, 0, len(manifestItems.Items))
	secretIDToEnv := make(map[string]string)

	for _, item := range manifestItems.Items {
		secretKeys = append(secretKeys, &item.Name)
		secretIDToEnv[item.Name] = item.Env
	}

	svc, err := NewSecretsManager(result.Source.URL)
	if err != nil {
		result.AppendError(err)
		return
	}

	// Fetch all secrets using shared batch function
	result = getSecrets(result.Source, svc, secretKeys)

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
