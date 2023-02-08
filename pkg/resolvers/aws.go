package resolvers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	snagsbyConfig "github.com/roverdotcom/snagsby/pkg/config"
)

func getAwsConfig(optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
	// If the SNAGSBY_LOG_AWS_RETRIES environment variable is truthy
	// we will log retries
	if snagsbyConfig.EnvBool("SNAGSBY_LOG_AWS_RETRIES") {
		optFns = append(optFns, config.WithClientLogMode(aws.LogRetries))
	}
	return config.LoadDefaultConfig(context.TODO(), optFns...)
}

func readJSONString(input string) (map[string]string, error) {
	var f map[string]interface{}
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
