package helper

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
)

type DummyS3ObjectPutterService struct {
	ExpectedBucket        string
	ExpectedKey           string
	ExpectedBody          string
	ExpectedContentType   string
	ExpectedContentLength int64
}

func (s3Svc DummyS3ObjectPutterService) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {

	if s3Svc.ExpectedContentLength != *input.ContentLength {
		return nil, fmt.Errorf(
			"expected content length does not match supplied content length\nexpected=%v, supplied=%v",
			s3Svc.ExpectedContentLength,
			input.ContentLength,
		)
	}

	if s3Svc.ExpectedBucket != *input.Bucket {
		return nil, fmt.Errorf(
			"expected bucket does not match supplied bucket\nexpected=%v, supplied=%v",
			s3Svc.ExpectedBucket,
			input.Bucket,
		)
	}

	if s3Svc.ExpectedKey != *input.Key {
		return nil, fmt.Errorf(
			"expected key does not match supplied key\nexpected=%v, supplied=%v",
			s3Svc.ExpectedKey,
			*input.Key,
		)
	}

	if s3Svc.ExpectedContentType != *input.ContentType {
		return nil, fmt.Errorf(
			"expected content type does not match supplied content type\nexpected=%v, supplied=%v",
			s3Svc.ExpectedContentType,
			input.ContentType,
		)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(input.Body)
	suppliedBody := buf.String()

	if s3Svc.ExpectedBody != suppliedBody {
		return nil, fmt.Errorf(
			"expected body does not match supplied body\nexpected=%v, supplied=%v",
			s3Svc.ExpectedBody,
			suppliedBody,
		)
	}

	resp := &s3.PutObjectOutput{}

	return resp, nil
}
