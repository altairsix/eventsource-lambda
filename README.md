# eventsource-lambda

Transports for publishing eventsource events using AWS lambda.

This library makes use of the excellent ```apex``` library to manage lambda functions.

## Functions

### dynamodb-to-firehose

Publishes events from dynamodb to kinesis firehose for ultimate storage on S3.  Very 
useful if you intend on using S3 / Cloud Front for event replay and backup.  

[More Info](functions/dynamodb-to-firehose/README.md)

### dynamodb-to-stan

Publishes events from dynamodb to nats stream server for real time event processing.

[More Info](functions/dynamodb-to-stan/README.md)

## Deployment

As each function requires its own configuration, functions should be deployed independently.
For example, to deploy ```dynamodb-to-firehose``` using a config stored in 
```s3://my-config/production/config.yaml``` with an iam role of
```arn:aws:iam::872981728712:role/eventsource_lambda_function```, you would write:

```bash
apex deploy dynamodb-to-firehose \
    --iamrole arn:aws:iam::872981728712:role/eventsource_lambda_function \
    --set "S3_CONFIG=s3://my-config/production/config.yaml"
```
