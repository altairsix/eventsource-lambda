package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/altairsix/eventsource-lambda"
	"github.com/altairsix/eventsource/dynamodbstore"
	"github.com/apex/go-apex"
	"github.com/apex/go-apex/dynamo"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
)

var (
	urlRE = regexp.MustCompile(`(\S+)://((\S+):(\S+)@)?(\S+):(\d+)`)
)

// Config contains the configuration object
type Nats struct {
	ClusterID string `yaml:"cluster_id"`
	Url       string `yaml:"url"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

type Stream struct {
	Subject string `yaml:"subject"`
}

type Config struct {
	Nats    Nats              `yaml:"nats"`
	Streams map[string]Stream `yaml:"streams"`
}

func Url(c Nats) string {
	urls := []string{}

	for _, segment := range strings.Split(c.Url, ",") {
		match := urlRE.FindStringSubmatch(strings.TrimSpace(segment))
		if len(match) != 7 {
			log.Fatalln("invalid nats url")
		}

		proto, username, password, host, port := match[1], match[3], match[4], match[5], match[6]
		if c.Username != "" {
			username = c.Username
		}
		if c.Password != "" {
			password = c.Password
		}

		credentials := ""
		if username != "" && password != "" {
			credentials = fmt.Sprintf("%s:%s@", username, password)
		}

		urls = append(urls, fmt.Sprintf("%s://%s%s:%s", proto, credentials, host, port))
	}

	return strings.Join(urls, ", ")
}

func FetchRegion(region string) (*Config, error) {
	s, err := session.NewSession(aws.NewConfig().WithRegion(region))
	if err != nil {
		return nil, err
	}
	s3api := s3.New(s)

	config := &Config{}
	if err := eventsource_lambda.Unmarshal(s3api, os.Getenv("S3_CONFIG"), config); err != nil {
		return nil, err
	}

	if config.Streams == nil {
		config.Streams = map[string]Stream{}
	}

	return config, nil
}

func Connect(config Nats) (stan.Conn, error) {
	nc, err := nats.Connect(Url(config))
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "Attempting to connect to stan cluster, %v\n", config.ClusterID)
	conn, err := stan.Connect(config.ClusterID, ksuid.New().String(), stan.NatsConn(nc))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to connect to stan cluster, %v", config.ClusterID)
	}

	fmt.Fprintf(os.Stderr, "Successfully connected to stan cluster, %v\n", config.ClusterID)
	return conn, nil
}

func newHandler(region string) (dynamo.HandlerFunc, error) {
	config, err := FetchRegion(region)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch config from region, %v", region)
	}

	sc, err := Connect(config.Nats)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to connect to nats, %v", config.Nats.Url)
	}

	return func(event *dynamo.Event, c *apex.Context) error {
		if len(event.Records) == 0 {
			return nil
		}

		eventSourceARN := event.Records[0].EventSourceARN
		tableName, err := dynamodbstore.TableName(eventSourceARN)
		if err != nil {
			return fmt.Errorf("unable to determine table name from event source arn, %v", eventSourceARN)
		}

		stream, ok := config.Streams[tableName]
		if !ok {
			return fmt.Errorf("no stream mapping for table, %v", tableName)
		}

		count := 0
		for _, item := range event.Records {
			records, err := dynamodbstore.Changes(item)
			if err != nil {
				return errors.New("unable to detect changes from item")
			}

			for _, record := range records {
				if err := sc.Publish(stream.Subject, record.Data); err != nil {
					return errors.Wrapf(err, "unable to publish content to stream, %v", stream.Subject)
				}
			}
		}

		fmt.Fprintf(os.Stderr, "Successfully published %v events\n", count)
		return nil
	}, nil
}

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		region = "us-west-2"
	}

	h, err := newHandler(region)
	check(err)

	dynamo.HandleFunc(h)
}
