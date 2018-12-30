package main

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/minio/minio/pkg/wildcard"
)

// ExpandResult will store the expanded sources and any errors to be used in a channel
type ExpandResult struct {
	Sources []*url.URL
	Error   error
}

// Default expander
func expand(source *url.URL) *ExpandResult {
	if source.Scheme == "sm" {
		return expandSM(source)
	}
	return &ExpandResult{[]*url.URL{source}, nil}
}

func expandSM(source *url.URL) *ExpandResult {
	secretName := fmt.Sprintf("%s%s", source.Host, source.Path)

	// If we're not splatting just return
	if !strings.Contains(secretName, "*") {
		return &ExpandResult{[]*url.URL{source}, nil}
	}

	out := []*url.URL{}
	sess, sessionError := getAwsSession()

	if sessionError != nil {
		return &ExpandResult{out, sessionError}
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
	// Sort by secret name
	sort.Slice(out, func(i, j int) bool {
		return out[i].Host+out[i].Path < out[j].Host+out[j].Path
	})
	return &ExpandResult{out, sessionError}
}
