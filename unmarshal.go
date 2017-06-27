package eventsource_lambda

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	s3re = regexp.MustCompile(`s3://([^/]+)/(.*)`)
)

func bucketAndKey(s3uri string) (string, string, bool) {
	matches := s3re.FindStringSubmatch(s3uri)
	if len(matches) != 3 {
		return "", "", false
	}

	return matches[1], matches[2], true
}

func Unmarshal(s3api s3iface.S3API, s3uri string, v interface{}) error {
	bucket, key, ok := bucketAndKey(s3uri)
	if !ok {
		return fmt.Errorf("invalid s3 uri, %v.  expected s3://bucket/path", s3uri)
	}

	out, err := s3api.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return errors.Wrapf(err, "unable to retrieve content from s3 uri, %v", s3uri)
	}
	defer out.Body.Close()

	data, err := ioutil.ReadAll(out.Body)
	if err != nil {
		return errors.Wrapf(err, "unable to read content from s3 uri, %v", s3uri)
	}

	if err := yaml.Unmarshal(data, v); err != nil {
		return errors.Wrapf(err, "unable read config from s3 uri, %v", s3uri)
	}

	return nil
}
