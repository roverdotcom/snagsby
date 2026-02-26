package resolvers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/roverdotcom/snagsby/pkg/config"
	"github.com/roverdotcom/snagsby/pkg/parsers"
)

type secretsManagerConnector interface {
	ListSecrets(prefix string) ([]*string, error)
	GetSecret(secretName string) (string, error)
	GetSecrets(keys []*string) (map[string]string, []error)
}

type SecretsManagerResolver struct {
	connector secretsManagerConnector
}

func (s *SecretsManagerResolver) keyNameFromPrefix(prefix, name string) string {
	key := strings.TrimPrefix(name, prefix)
	key = KeyRegexp.ReplaceAllString(key, "_")
	key = strings.ToUpper(key)
	return key
}

func (s *SecretsManagerResolver) resolveRecursive(source *config.Source) *Result {
	result := &Result{Source: source}
	sourceURL := source.URL
	prefix := strings.TrimSuffix(fmt.Sprintf("%s%s", sourceURL.Host, sourceURL.Path), "*")

	secretKeys, err := s.connector.ListSecrets(prefix)
	if err != nil {
		result.AppendError(err)
		return result
	}
	secrets, errors := s.connector.GetSecrets(secretKeys)
	for _, err := range errors {
		result.AppendError(err)
	}

	for key, value := range secrets {
		result.AppendItem(s.keyNameFromPrefix(prefix, key), value)
	}

	return result
}

func (s *SecretsManagerResolver) resolveSingle(source *config.Source) *Result {
	result := &Result{Source: source}
	sourceURL := source.URL

	secretName := strings.Join([]string{sourceURL.Host, sourceURL.Path}, "")
	secretString, err := s.connector.GetSecret(secretName)
	if err != nil {
		result.AppendError(err)
		return result
	}

	out, err := parsers.ReadJSONString(secretString)
	if err != nil {
		result.AppendError(err)
		return result
	}

	result.AppendItems(out)

	return result
}

func (s *SecretsManagerResolver) isRecursive(source *config.Source) bool {
	re := regexp.MustCompile(`.*\/\*$`)
	sourceURL := source.URL
	return re.MatchString(strings.Join([]string{sourceURL.Host, sourceURL.Path}, ""))
}

// Resolve returns results
func (s *SecretsManagerResolver) Resolve(source *config.Source) *Result {
	// Recursive will behave differently
	if s.isRecursive(source) {
		return s.resolveRecursive(source)
	}
	return s.resolveSingle(source)
}
