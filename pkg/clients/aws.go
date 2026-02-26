package clients

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	snagsbyConfig "github.com/roverdotcom/snagsby/pkg/config"
)

func GetAwsConfig(optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
	// If the SNAGSBY_LOG_AWS_RETRIES environment variable is truthy
	// we will log retries
	if snagsbyConfig.EnvBool("SNAGSBY_LOG_AWS_RETRIES") {
		optFns = append(optFns, config.WithClientLogMode(aws.LogRetries))
	}
	return config.LoadDefaultConfig(context.TODO(), optFns...)
}

// TODO -0 Is this part of AWS?
func ReadJSONString(input string) (map[string]string, error) {
	var f map[string]any
	out := map[string]string{}
	if err := json.Unmarshal([]byte(input), &f); err != nil {
		return out, err
	}
	for k, v := range f {
		k = strings.ToUpper(k)
		switch vv := v.(type) {
		case string:
			out[k] = vv
		case float64:
			out[k] = strconv.FormatFloat(vv, 'f', -1, 64)
		case bool:
			var b string
			if vv {
				b = "1"
			} else {
				b = "0"
			}
			out[k] = b
		}
	}
	return out, nil
}

func NewSecretsManager(sourceURL *url.URL) (*secretsmanager.Client, error) {

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
