# Snagsby [![Build Status](https://travis-ci.org/roverdotcom/snagsby.svg?branch=master)](https://travis-ci.org/roverdotcom/snagsby)

Snagsby reads a JSON object from an S3 bucket and outputs the keys and values
in a format that can be evaluated by a shell to set environment variables.

It's useful for reading configuration into environment variables from S3
objects in Docker containers.

It can help with workflows like this one: https://blogs.aws.amazon.com/security/post/Tx2B3QUWAA7KOU/How-to-Manage-Secrets-for-Amazon-EC2-Container-Service-Based-Applications-by-Usi

## Installation

Linux and OSX 64 bit binaries are available on Github

```bash
curl -L \
    https://github.com/roverdotcom/snagsby/releases/download/v0.1.5/snagsby-0.1.5.darwin-amd64.tar.gz \
    | tar zxf - ./snagsby
```

## JSON Format

The s3 object should contain a single JSON object:

```javascript
// s3://my-bucket/config.json
{
  "processes": 2,
  "multiline_config": "123\n456\n789",
  "api_key": "abc123",
  "yes": true,
  "no": false,
  "float_like": 7.777
}
```

Upload to `s3://my-bucket/config.json` with server side encryption and tight bucket access restrictions/policies.

Snagsby can be configured with the `SNAGSBY_SOURCE` env var or you can pass the source url on the command line.

```bash
snagsby s3://my-bucket/config.json?region=us-west-2
```

Would render:

```bash
export API_KEY="abc123"
export FLOAT_LIKE="7.777"
export MULTILINE_CONFIG="123\n456\n789"
export NO="0"
export PROCESSES="2"
export YES="1"
```

You can supply sources in a comma delimited `SNAGSBY_SOURCE` environment variable:

```bash
SNAGSBY_SOURCE="s3://my-bucket/secrets1.json, s3://my-bucket/secrets2.json" ./bin/snagsby

# -e will fail on errors and exit 1
./bin/snagsby -e \
  s3://my-bucket/config.json \
  s3://my-bucket/config2.json
```

An example docker entrypoint may look like:

```bash
#!/bin/sh

set -e

eval $(./bin/snagsby \
  s3://my-bucket/config.json?region=us-west-2 \
  s3://my-bucket/config-production.json?region-us-west-1)

exec "$@"
```

## AWS Configuration

You can configure AWS any way the golang sdk supports:
https://github.com/aws/aws-sdk-go#configuring-credentials

The preferred method when inside ec2 is to rely instance profile IAM roles.

You can configure the default region by setting the `AWS_REGION` environment
variable. It's recommended you set the region on each source:
`s3://my-bucket/snagsby-config.json?region=us-west-2`
