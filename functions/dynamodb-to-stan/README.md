dynamodb-to-stan
-------------

Projects eventsource data from dynamodb to nats streaming servers via dynamodb and lambda.  
  
Raw events will be published to nats.

### Configuration

Like all eventsource-lambda functions, this one is also configured via the ```S3_CONFIG``` 
environment variable.  In this case ```S3_CONFIG``` should point to a yaml file that contains
the information on how to connect to nats and which cluster / subject to send the events to.

```yaml
nats:
  url: "nats://your-nats-server:4222"
  username: "optional-username"
  password: "optional-password"
  cluster_id: "stan-cluster-id"
  
streams:
  your-table-name:
    subject: "subject.to.publish.events.to"
```

Any number of tables may be configured this way.  

If dynamodb-to-firehose comes across a table not listed in its config, it will return an error
under the assumption that this is a misconfiguration.

### AWS Policies

The role executing dynamodb-to-firehose will need permissions to:

* execute lambda function to dynamodb
* access the S3 bucket where the config is stored

### Networking

Since the lambda function needs access to the nats servers, if you host nats servers configured
in a VPC, you will need to start the lambda function in that VPC
