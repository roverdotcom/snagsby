package resolvers

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/roverdotcom/snagsby/pkg/config"
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

	svc, err := NewSecretsManager(sourceURL)
	if err != nil {
		result.AppendError(err)
		return result
	}

	// List secrets that begin with our prefix
	params := &secretsmanager.ListSecretsInput{
		Filters: []types.Filter{
			{
				Key: "name",
				Values: []string{
					prefix,
				},
			},
		},
	}
	secretKeys := []*string{}
	paginator := secretsmanager.NewListSecretsPaginator(svc, params)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			result.AppendError(err)
			return result
		}
		for _, secret := range output.SecretList {
			secretKeys = append(secretKeys, secret.Name)
		}

	}

	secrets, errors := getSecrets(source, svc, secretKeys)
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

	cfg, err := getAwsConfig()

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
	out, err := readJSONString(*res.SecretString)
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
