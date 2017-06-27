dynamodb-to-firehose
-------------

Projects eventsource data from dynamodb to firehose via dynamodb streams and lambda.  
  
Data stored to S3 will be encoded as:

```
[base64-encoded-event][newline]
```

### Configuration

Like all eventsource-lambda functions, this one is also configured via the ```S3_CONFIG``` 
environment variable.  In this case ```S3_CONFIG``` should point to a yaml file that contains
the name of the table and the delivery stream the content should be sent to.

```yaml
my-table:
  delivery_stream: "my-delivery-stream"
```

Any number of tables may be configured this way.  

If dynamodb-to-firehose comes across a table not listed in its config, it will return an error
under the assumption that this is a misconfiguration.

### AWS Policies

The role executing dynamodb-to-firehose will need permissions to:

* execute lambda function to dynamodb
* access the S3 bucket where the config is stored
* write to the firehose delivery stream
