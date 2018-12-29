package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/minio/minio/pkg/wildcard"
)

// Default expander
func expand(source *url.URL) ([]*url.URL, error) {
	if source.Scheme == "sm" {
		return expandSM(source)
	}
	return []*url.URL{source}, nil
}

func expandSM(source *url.URL) ([]*url.URL, error) {
	secretName := fmt.Sprintf("%s%s", source.Host, source.Path)

	// If we're not splatting just return
	if !strings.Contains(secretName, "*") {
		return []*url.URL{source}, nil
	}

	out := []*url.URL{}
	sess, sessionError := getAwsSession()

	if sessionError != nil {
		return out, sessionError
	}

	region := source.Query().Get("region")
	config := aws.Config{}
	if region != "" {
		config.Region = aws.String(region)
	}
	svc := secretsmanager.New(sess, &config)
	svc.ListSecretsPages(&secretsmanager.ListSecretsInput{}, func(page *secretsmanager.ListSecretsOutput, lastPage bool) bool {
		for _, p := range page.SecretList {
			if wildcard.MatchSimple(secretName, *p.Name) {
				url, err := url.Parse(fmt.Sprintf("sm://%s", *p.Name))
				if err == nil {
					out = append(out, url)
				}
				url.RawQuery = source.RawQuery
			}
		}
		return true
	})
	return out, nil
}
