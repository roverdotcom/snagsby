package resolvers

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/roverdotcom/snagsby/pkg/clients"
	"github.com/roverdotcom/snagsby/pkg/config"
	"github.com/roverdotcom/snagsby/pkg/connectors"
)

// SecretsManagerResolver handles secrets manager resolution
type SecretsManagerResolver struct{}

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

	smConnector, err := connectors.NewSecretsManagerConnector(source)
	if err != nil {
		result.AppendError(err)
		return result
	}
	secretKeys, err := smConnector.ListSecrets(prefix)
	if err != nil {
		result.AppendError(err)
		return result
	}
	secrets, errors := smConnector.GetSecrets(secretKeys)
	for _, err := range errors {
		result.AppendError(err)
	}

	for key, value := range secrets {
		result.AppendItem(s.keyNameFromPrefix(prefix, key), value)
	}

	return result
}

// TODO - See how to clean this up
func (s *SecretsManagerResolver) resolveSingle(source *config.Source) *Result {
	result := &Result{Source: source}
	sourceURL := source.URL

	cfg, err := clients.GetAwsConfig()

	if err != nil {
		result.AppendError(err)
		return result
	}

	region := sourceURL.Query().Get("region")
	if region != "" {
		cfg.Region = region
	}
	svc := secretsmanager.NewFromConfig(cfg)

	secretName := strings.Join([]string{sourceURL.Host, sourceURL.Path}, "")
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
	res, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		result.AppendError(err)
		return result
	}

	// TODO - ReadJSONString does not belong to AWS clients
	out, err := clients.ReadJSONString(*res.SecretString)
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
