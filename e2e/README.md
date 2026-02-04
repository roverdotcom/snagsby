# Snagsby E2E Tests

This directory contains end-to-end (e2e) tests for Snagsby that verify functionality requiring actual AWS resources.

## Overview

The e2e tests cover critical code paths that cannot be tested with unit tests:
- **S3 Resolver**: Reading JSON configuration from S3 buckets
- **Secrets Manager Resolver**: Fetching secrets (both single and recursive patterns)
- **Manifest Resolver**: Processing YAML manifests with secret references
- **CLI Functionality**: Command-line flags and output formats
- **Error Handling**: AWS errors, invalid inputs, etc.

## Test Files

### Core Tests
- **`e2e.sh`**: Original e2e test script (basic Secrets Manager tests)
- **`e2e.py`**: Python tests for environment variable loading and CLI functionality
- **`e2e_extended.py`**: Extended tests for S3, Manifest, and integration scenarios
- **`e2e_comprehensive.sh`**: Comprehensive test runner for all e2e tests

### Running Tests

#### Prerequisites
1. Build snagsby: `make dist`
2. Configure AWS credentials (IAM role, environment variables, or AWS config file)
3. Set up test AWS resources (see below)

#### Basic Tests (Secrets Manager only)
```bash
# Uses default secrets: sm://snagsby/acceptance and sm:///snagsby/app/acceptance/*
./e2e/e2e.sh
```

#### Comprehensive Tests
```bash
# Run all configured tests
./e2e/e2e_comprehensive.sh
```

#### Individual Test Suites
```bash
# Run specific test classes
python3 ./e2e/e2e.py SnagsbyCliTests
python3 ./e2e/e2e_extended.py SnagsbyS3Tests
python3 ./e2e/e2e_extended.py SnagsbyManifestTests
```

## Required AWS Resources

### Secrets Manager (Required)

The basic e2e tests require these secrets in AWS Secrets Manager:

1. **`snagsby/acceptance`** - Single secret with JSON value:
```json
{
  "tricky_characters": "@^*309_!~``:*/\\{}%()>$t'",
  "starts_with_hash": "#hello?world"
}
```

2. **`snagsby/app/acceptance/recursive_tricky_characters`** - For recursive pattern testing:
```
Value: @^*309_!~``:*/\{}%()>$t'
```

### S3 (Optional)

For S3 resolver tests, create an S3 bucket with a test configuration file:

**Example: `s3://my-test-bucket/snagsby-test.json`**
```json
{
  "test_key": "test_value",
  "number": 123,
  "boolean": true
}
```

Then set:
```bash
export SNAGSBY_E2E_S3_SOURCE="s3://my-test-bucket/snagsby-test.json?region=us-west-2"
```

### Manifest (Optional)

For Manifest resolver tests, create a YAML manifest file:

**Example: `/tmp/test-manifest.yaml`**
```yaml
items:
  - name: snagsby/acceptance
    env: TEST_SECRET
  - name: snagsby/app/acceptance/recursive_tricky_characters
    env: ANOTHER_SECRET
```

Then set:
```bash
export SNAGSBY_E2E_MANIFEST_SOURCE="manifest:///tmp/test-manifest.yaml"
```

## Environment Variables

### Required
- **`SNAGSBY_E2E_SOURCE`**: Secrets Manager sources (default: `"sm://snagsby/acceptance sm:///snagsby/app/acceptance/*"`)

### Optional
- **`SNAGSBY_E2E_S3_SOURCE`**: S3 source URL for S3 tests
- **`SNAGSBY_E2E_MANIFEST_SOURCE`**: Manifest file path for manifest tests
- **`SNAGSBY_E2E_S3_INVALID_JSON`**: S3 source with invalid JSON for error testing
- **`SNAGSBY_E2E_OVERRIDE_TEST`**: Space-separated sources for testing merge behavior

### AWS Configuration
- **`AWS_REGION`**: Default AWS region
- **`AWS_ACCESS_KEY_ID`** / **`AWS_SECRET_ACCESS_KEY`**: AWS credentials (if not using IAM roles)

## Test Coverage

These e2e tests provide coverage for:

### S3 Resolver (`pkg/resolvers/s3.go`)
- ✅ `Resolve()` method - AWS S3 integration
- ✅ Region parameter handling
- ✅ JSON parsing from S3 objects
- ✅ Error handling (nonexistent buckets, invalid JSON)

### Secrets Manager Resolver (`pkg/resolvers/secretsmanager.go`)
- ✅ `smWorker()` - Concurrent secret fetching
- ✅ `resolveRecursive()` - Wildcard pattern matching
- ✅ `resolveSingle()` - Single secret retrieval
- ✅ Version stage and version ID parameters
- ✅ Error handling

### Manifest Resolver (`pkg/resolvers/manifest.go`)
- ✅ `manifestWorker()` - Concurrent manifest processing
- ✅ `getSecretValue()` - Secret fetching from manifest
- ✅ `resolveManifestItems()` - Bulk secret processing
- ✅ `Resolve()` - YAML parsing and processing

### Main CLI (`main.go`)
- ✅ `-v` flag (version information)
- ✅ `-e` flag (fail on errors)
- ✅ `-show-summary` flag (display summary)
- ✅ `-o` / `-output` flag (output formats)
- ✅ `env` formatter (default)
- ✅ `envfile` formatter (no export prefix)
- ✅ `json` formatter
- ✅ Multiple source handling
- ✅ Error handling and exit codes
- ✅ `SNAGSBY_SOURCE` environment variable

## CI/CD Integration

### GitHub Actions Example
```yaml
- name: Run E2E Tests
  env:
    AWS_REGION: us-west-2
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    SNAGSBY_E2E_SOURCE: "sm://snagsby/acceptance sm:///snagsby/app/acceptance/*"
  run: |
    make dist
    ./e2e/e2e_comprehensive.sh
```

### Local Development
```bash
# Build binary
make dist

# Run with default Secrets Manager sources
./e2e/e2e_comprehensive.sh

# Run with all sources configured
export SNAGSBY_E2E_S3_SOURCE="s3://my-bucket/config.json?region=us-west-2"
export SNAGSBY_E2E_MANIFEST_SOURCE="manifest:///tmp/manifest.yaml"
./e2e/e2e_comprehensive.sh
```

## Troubleshooting

### "Binary not found" error
Run `make dist` to build the distribution binaries.

### AWS authentication errors
- Verify AWS credentials are configured
- Check IAM permissions for Secrets Manager, S3, etc.
- Ensure the AWS region is set correctly

### Test secrets not found
- Create the required secrets in AWS Secrets Manager
- Verify secret names match the expected format
- Check the region configuration

### Python test failures
- Ensure Python 3 is installed
- Required modules are in the standard library (no extra deps needed)

## Contributing

When adding new resolvers or significant features:
1. Add corresponding e2e tests to `e2e_extended.py`
2. Update this README with new requirements
3. Document any new environment variables
4. Update the comprehensive test runner if needed
