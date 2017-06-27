package eventsource_lambda_test

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/altairsix/eventsource-lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
)

type Mock struct {
	s3iface.S3API
	Content string
}

func (m *Mock) GetObject(*s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{
		Body: ioutil.NopCloser(strings.NewReader(m.Content)),
	}, nil
}

func TestUnmarshal(t *testing.T) {
	content := map[string]string{}
	m := &Mock{Content: `{"hello":"world"}`}
	err := eventsource_lambda.Unmarshal(m, "s3://sample/go", &content)
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"hello": "world"}, content)
}
