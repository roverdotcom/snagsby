package resolvers

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// Concurrency work
type smMessage struct {
	Source      *config.Source
	Name        *string
	Result      string
	Error       error
	IsRecursive bool
}

func smWorker(jobs <-chan *smMessage, results chan<- *smMessage, svc *secretsmanager.Client) {
	for job := range jobs {
		sourceURL := job.Source.URL
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		input := &secretsmanager.GetSecretValueInput{
			SecretId: job.Name,
		}
		versionStage := sourceURL.Query().Get("version-stage")
		if versionStage != "" {
			input.VersionStage = aws.String(versionStage)
		}
		versionID := sourceURL.Query().Get("version-id")
		if versionID != "" {
			input.VersionId = aws.String(versionID)
		}
		getSecret, err := svc.GetSecretValue(ctx, input)
		if err != nil {
			job.Error = err
		} else {
			job.Result = *getSecret.SecretString
		}
		results <- job
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

	jobs := make(chan *smMessage, len(secretKeys))
	results := make(chan *smMessage, len(secretKeys))

	// Determine concurrency level defaulting to number of secrets to pull
	var numWorkers int
	if smConcurrency > 0 {
		numWorkers = smConcurrency
	} else {
		numWorkers = len(secretKeys)
	}

	// Boot up workers
	for w := 1; w <= numWorkers; w++ {
		go smWorker(jobs, results, svc)
	}

	// Publish to workers
	for _, name := range secretKeys {
		jobs <- &smMessage{Source: source, Name: name}
	}
	close(jobs)

	// Loop through results
	for a := 1; a <= len(secretKeys); a++ {
		res := <-results
		if res.Error != nil {
			result.AppendError(res.Error)
		} else {
			result.AppendItem(s.keyNameFromPrefix(prefix, *res.Name), res.Result)
		}
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
