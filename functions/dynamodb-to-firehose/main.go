package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/altairsix/eventsource"
	"github.com/altairsix/eventsource-lambda"
	"github.com/altairsix/eventsource/dynamodbstore"
	"github.com/apex/go-apex"
	"github.com/apex/go-apex/dynamo"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	// Size contains max number of byte to store per firehose record
	Size = 999
)

// Config contains the configuration object
type Config struct {
	DeliveryStream string `yaml:"delivery_stream"`
}

type Publisher func(deliveryStream string, data []byte) error

func (fn Publisher) Publish(deliveryStream string, data []byte) error {
	return fn(deliveryStream, data)
}

func newFirehose(region string) *firehose.Firehose {
	s, err := session.NewSession(aws.NewConfig().WithRegion(region))
	if err != nil {
		log.Fatalln(err)
	}

	return firehose.New(s)
}

func newPublisher(fh *firehose.Firehose) Publisher {
	return func(deliveryStream string, data []byte) error {
		if len(data) == 0 {
			return nil
		}

		_, err := fh.PutRecord(&firehose.PutRecordInput{
			DeliveryStreamName: aws.String(deliveryStream),
			Record: &firehose.Record{
				Data: data,
			},
		})

		return err
	}
}

func findDeliveryStream(region, eventsourceARN string) (string, error) {
	s, err := session.NewSession(aws.NewConfig().WithRegion(region))
	if err != nil {
		return "", err
	}
	s3api := s3.New(s)

	configs := map[string]Config{}
	s3uri := os.Getenv("S3_CONFIG")
	if err := eventsource_lambda.Unmarshal(s3api, s3uri, &configs); err != nil {
		return "", err
	}

	tableName, err := dynamodbstore.TableName(eventsourceARN)
	if err != nil {
		return "", err
	}

	c, ok := configs[tableName]
	if !ok {
		return "", fmt.Errorf("no delivery stream defined for table, %v", tableName)
	}

	return c.DeliveryStream, nil
}

func start(ch <-chan eventsource.Record, deliveryStream string, publisher Publisher) <-chan error {
	errs := make(chan error, 1)

	go func() {
		defer close(errs)

		content := make([]byte, 0, Size)
		buf := bytes.NewBuffer(content)

		for record := range ch {
			if v := base64.StdEncoding.EncodedLen(len(record.Data)); v+buf.Len() > Size {
				if err := publisher.Publish(deliveryStream, buf.Bytes()); err != nil {
					errs <- err
					return
				}
				buf.Reset()
			}

			buf.WriteString(base64.StdEncoding.EncodeToString(record.Data))
			buf.WriteString("\n")
		}

		errs <- publisher.Publish(deliveryStream, buf.Bytes())
	}()

	return errs
}

func newHandler(region string, publisher Publisher) dynamo.HandlerFunc {
	return func(event *dynamo.Event, c *apex.Context) error {
		if len(event.Records) == 0 {
			return nil
		}

		deliveryStream, err := findDeliveryStream(region, event.Records[0].EventSourceARN)
		if err != nil {
			return err
		}

		ch := make(chan eventsource.Record, 32)
		errs := start(ch, deliveryStream, publisher)

		count := 0
		for _, item := range event.Records {
			records, err := dynamodbstore.Changes(item)
			if err != nil {
				return err
			}

			for _, record := range records {
				ch <- record
				count++
			}
		}
		close(ch)

		for err := range errs {
			if err != nil {
				return err
			}
		}

		fmt.Fprintf(os.Stderr, "Successfully published %v events\n", count)
		return nil
	}
}

func main() {
	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		region = "us-west-2"
	}

	publisher := newPublisher(newFirehose(region))
	h := newHandler(region, publisher)

	dynamo.HandleFunc(h)
}
