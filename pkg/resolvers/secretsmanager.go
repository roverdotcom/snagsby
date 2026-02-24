package resolvers

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/roverdotcom/snagsby/pkg/config"
)

var smConcurrency int

func init() {
	// Pull concurrency settings
	getConcurrency, hasSetting := os.LookupEnv("SNAGSBY_SM_CONCURRENCY")
	if hasSetting {
		i, err := strconv.Atoi(getConcurrency)
		if err == nil && i >= 0 {
			smConcurrency = i
		}
	}
}

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
	cfg, err := getAwsConfig(awsConfig.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
	}))

	if err != nil {
		result.AppendError(err)
		return result
	}

	region := sourceURL.Query().Get("region")
	if region != "" {
		cfg.Region = region
	}
	svc := secretsmanager.NewFromConfig(cfg)

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

	// Collect all secret names from paginated results
	secretIDs := []string{}
	paginator := secretsmanager.NewListSecretsPaginator(svc, params)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			result.AppendError(err)
			return result
		}
		for _, secret := range output.SecretList {
			secretIDs = append(secretIDs, *secret.Name)
		}
	}

	// Determine concurrency level
	concurrency := smConcurrency
	if concurrency <= 0 {
		concurrency = len(secretIDs)
		if concurrency > 20 {
			concurrency = 20 // Cap at 20 workers by default
		}
	}

	// Fetch all secrets using shared batch function
	secretValues, errors := BatchFetchSecrets(secretIDs, concurrency)

	// Add errors to result
	for _, err := range errors {
		result.AppendError(err)
	}

	// Add fetched secrets with key name transformation
	for secretID, value := range secretValues {
		keyName := s.keyNameFromPrefix(prefix, secretID)
		result.AppendItem(keyName, value)
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
