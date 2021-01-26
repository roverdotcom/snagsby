package resolvers

import (
	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/roverdotcom/snagsby/pkg/config"
)

// S3ManagerResolver handles s3 resolution
type S3ManagerResolver struct{}

// Resolve returns results
func (s *S3ManagerResolver) Resolve(source *config.Source) *Result {
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
	svc := s3.New(sess, &config)
	res, s3err := svc.GetObject(&s3.GetObjectInput{
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
