package resolvers

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
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

func smWorker(jobs <-chan *smMessage, results chan<- *smMessage, svc *secretsmanager.SecretsManager) {
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
		getSecret, err := svc.GetSecretValueWithContext(ctx, input)
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
	sess, sessionError := getAwsSession()

	if sessionError != nil {
		result.AppendError(sessionError)
		return result
	}

	region := sourceURL.Query().Get("region")
	config := aws.Config{}
	if region != "" {
		config.Region = aws.String(region)
	}
	svc := secretsmanager.New(sess, &config)

	// List secrets that begin with our prefix
	params := &secretsmanager.ListSecretsInput{
		Filters: []*secretsmanager.Filter{
			{
				Key: aws.String("name"),
				Values: []*string{
					aws.String(prefix),
				},
			},
		},
	}
	out := map[string]string{}
	secretKeys := []*string{}
	err := svc.ListSecretsPages(params,
		func(page *secretsmanager.ListSecretsOutput, lastPage bool) bool {
			for _, secret := range page.SecretList {
				secretKeys = append(secretKeys, secret.Name)
			}
			return true
		})
	if err != nil {
		result.AppendError(err)
		return result
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

	result.AppendItems(out)
	return result
}

func (s *SecretsManagerResolver) resolveSingle(source *config.Source) *Result {
	result := &Result{Source: source}
	sourceURL := source.URL

	sess, sessionError := getAwsSession()

	if sessionError != nil {
		result.AppendError(sessionError)
		return result
	}

	region := sourceURL.Query().Get("region")
	config := aws.Config{}
	if region != "" {
		config.Region = aws.String(region)
	}
	svc := secretsmanager.New(sess, &config)

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
	res, err := svc.GetSecretValue(input)
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
