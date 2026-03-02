package clients

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	snagsbyConfig "github.com/roverdotcom/snagsby/pkg/config"
)

func GetAwsConfig(optFns ...func(*awsConfig.LoadOptions) error) (aws.Config, error) {
	// If the SNAGSBY_LOG_AWS_RETRIES environment variable is truthy
	// we will log retries
	if snagsbyConfig.EnvBool("SNAGSBY_LOG_AWS_RETRIES") {
		optFns = append(optFns, awsConfig.WithClientLogMode(aws.LogRetries))
	}
	return awsConfig.LoadDefaultConfig(context.TODO(), optFns...)
}

func NewSecretsManagerClient(sourceURL *url.URL) (*secretsmanager.Client, error) {

	cfg, err := GetAwsConfig(awsConfig.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
	}))

	if err != nil {
		return nil, err
	}

	region := sourceURL.Query().Get("region")
	if region != "" {
		cfg.Region = region
	}
	svc := secretsmanager.NewFromConfig(cfg)

	return svc, nil
}
