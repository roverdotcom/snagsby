package resolvers

import (
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/roverdotcom/snagsby/pkg/config"
)

// S3ManagerResolver handles s3 resolution
type S3ManagerResolver struct{}

// Resolve returns results
func (s *S3ManagerResolver) Resolve(source *config.Source) *Result {
	result := &Result{Source: source}
	sourceURL := source.URL

	cfg, err := getAwsConfig()

	if err != nil {
		result.AppendError(err)
		return result
	}

	region := sourceURL.Query().Get("region")

	if region != "" {
		// config.Region = aws.String(region)
		cfg.Region = region
	}
	svc := s3.NewFromConfig(cfg)
	res, s3err := svc.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(sourceURL.Host),
		Key:    aws.String(sourceURL.Path),
	})

	if s3err != nil {
		result.AppendError(s3err)
		return result
	}
	defer res.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	bodyStr := buf.String()
	out, err := readJSONString(bodyStr)
	if err != nil {
		result.AppendError(err)
		return result
	}
	result.AppendItems(out)
	return result
}
