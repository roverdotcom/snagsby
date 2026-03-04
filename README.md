# Snagsby [![Build Status](https://travis-ci.org/roverdotcom/snagsby.svg?branch=master)](https://travis-ci.org/roverdotcom/snagsby)

Snagsby reads configuration and secrets from multiple sources and outputs them
as environment variables in a format that can be evaluated by a shell.

**Supported sources:**
- Local env files (`file://`) with dotenv format
- AWS S3 JSON objects (`s3://`)
- AWS Secrets Manager (`sm://`)

It's useful for reading configuration and secrets into environment variables in
Docker containers and other deployment scenarios.

It can help with workflows like this one: https://blogs.aws.amazon.com/security/post/Tx2B3QUWAA7KOU/How-to-Manage-Secrets-for-Amazon-EC2-Container-Service-Based-Applications-by-Usi

## Installation

Linux and OSX 64 bit binaries are available on Github

```bash
curl -L https://github.com/roverdotcom/snagsby/releases/download/v0.1.6/snagsby-0.1.6.linux-amd64.tar.gz \
    | tar zxf - -C /usr/local/bin
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
SNAGSBY_SOURCE="file://base.snagsby,  s3://my-bucket/secrets.json" ./bin/snagsby

# -e will fail on errors and exit 1
./bin/snagsby -e \
  file://production.snagsby \
  s3://my-bucket/config.json \
  s3://my-bucket/config2.json
```

An example docker entrypoint may look like:

```bash
#!/bin/sh

set -e

# Combine local config with remote secrets
eval $(./bin/snagsby \
  file://config/base.snagsby \
  file://config/production.snagsby \
  s3://my-bucket/config.json?region=us-west-2)

exec "$@"
```

## Env File Format

Snagsby supports reading environment variables from local files using the `file://` scheme with standard dotenv format.

### Basic Usage

```bash
snagsby file://local.snagsby
```

### File Format

Env files use the standard dotenv format (`KEY=VALUE`):

```bash
# Comments are supported
DATABASE_URL=postgres://localhost:5432/mydb
API_KEY=abc123
DEBUG=true

# Values can be quoted to preserve special characters
MESSAGE="Hello # this is not a comment"
PATH_WITH_SPACES="  /path/with/spaces  "

# Empty values are allowed
OPTIONAL_KEY=
```

### Secret References

Values can reference AWS Secrets Manager using the `sm://` prefix:

```bash
# Direct values
DATABASE_HOST=localhost
DATABASE_PORT=5432

# References to secrets in AWS Secrets Manager
DATABASE_PASSWORD=sm://production/db/password
API_SECRET=sm://production/api/secret
```

Snagsby will automatically fetch the secrets from AWS Secrets Manager and populate the environment variables with the actual values.

### File Naming Conventions

While Snagsby accepts any file extension, we recommend using extensions that clearly indicate the file contains **secret references**, not actual secrets:

- `.snagsby` - Clear about the tool used to resolve the secrets
- `.env.vault` - Suggests secrets/vault references
- `.env.ref` - Short for "references"
- `.envmap` - Conveys "mapping to secrets"

It is recommended to **avoid using `.env`** for files with secret references, as it may give developers a false sense that the file is safe to commit with actual secrets or that the file will not be commited to the repository.

### Multiple Sources

You can combine multiple source types:

```bash
snagsby \
  file://base.snagsby \
  file://production.snagsby \
  s3://my-bucket/config.json?region=us-west-2
```

### Validation and Key Handling

**Env files require strict POSIX-compliant variable names:**

Environment variable names in env files must follow shell naming conventions:
- Start with a letter or underscore (`[a-zA-Z_]`)
- Contain only letters, digits, and underscores (`[a-zA-Z0-9_]`)

Invalid keys (e.g., `my-key` with dashes, `my.key` with dots, or `123key` starting with a digit) will be rejected with a clear error message.

**Why strict validation?**

Unlike other Snagsby resolvers (S3, Secrets Manager) that normalize arbitrary keys (e.g., converting `my-key` to `MY_KEY`), env files are meant to define actual shell environment variables. Since shells only accept POSIX-compliant names, we validate at parse time to catch errors early.

**Key preservation:**

Keys in env files are preserved exactly as written, including their case. For example:
- `DATABASE_URL=...` stays as `DATABASE_URL` (not normalized to uppercase)
- `lowercase_var=...` stays as `lowercase_var`

This matches standard `.env` file behavior and ensures variables are set exactly as intended.

## AWS Configuration

You can configure AWS any way the golang sdk supports:
https://github.com/aws/aws-sdk-go#configuring-credentials

Snagsby enables support for the shared configuration file (~/.aws/config) in
the golang aws sdk.

The preferred method when in ec2 is to rely on instance profiles. When running
in aws ecs snagsby will use the task iam role.

You can configure the default region by setting the `AWS_REGION` environment
variable. It's recommended you set the region on each source:
`s3://my-bucket/snagsby-config.json?region=us-west-2`

## Releasing

From the `main` branch create a new SemVer tag:

```bash
git tag -a v0.7.0 -m "Release v0.7.0"
```

If you are looking to create a pre-release, ensure that you use a SemVer with a pre-release suffix i.e. SemVer with a dash and your pre-release suffix. For example `v0.7.0-alpha`.

When you have your tag created, push to the remote with `git push --tags` and the release job will be created.

