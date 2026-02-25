# Testing Documentation

This document describes the test coverage for the Snagsby project, particularly for the multi-source resolution feature.

## Test Structure

### Unit Tests

#### `pkg/resolvers/file_test.go`
Tests for the `file://` resolver:
- `TestParseEnvFile` - Parses .env files with various formats (quoted values, comments, sm:// references)
- `TestParseEnvFileWithMalformedLines` - Handles malformed lines gracefully
- `TestParseEnvFileNotFound` - Error handling for missing files
- `TestParseEnvFileEmpty` - Handles empty files
- `TestParseEnvFileOnlyComments` - Handles files with only comments
- `TestParseEnvFileEqualsInValue` - Handles equals signs in values

#### `pkg/resolvers/resolvers_test.go`
Tests for the resolver infrastructure:
- `TestKeyRegexp` - Key sanitization (uppercase, special chars to underscores)
- `TestAppendItems` - Adding items to results
- `TestResolveSource` - Router function for all resolver types (sm, s3, manifest, file)

#### `pkg/formatters/format_test.go`
Tests for output formatting:
- `TestMerge` - **Critical**: Verifies that later maps overwrite earlier ones

### Integration Tests

#### `pkg/app/app_integration_test.go`
Tests for single and multiple file:// sources:
- `TestMultipleSourcesWithOverwriting` - Single file:// source resolution
- `TestMergingMultipleResults` - Multiple file:// sources with overwriting behavior
- `TestParallelResolution` - Concurrent resolution of multiple sources

#### `pkg/app/app_multi_source_test.go`
**Multi-source integration tests with mocks**:

##### `TestMultiSourceIntegrationWithMocks`
**Most important test** - Demonstrates the full integration flow:
```
file://./base.env  (DATABASE_HOST, DATABASE_PORT, API_KEY, SHARED_SECRET)
    ↓
sm://production/*  (DATABASE_PASSWORD, DATABASE_USER, SHARED_SECRET - overwrites file)
    ↓
s3://bucket/config (CACHE_ENDPOINT, CACHE_PORT, API_KEY, SHARED_SECRET - overwrites both)
```

Verifies:
- All three resolver types work together
- Later sources overwrite earlier ones
- Keys from all sources are merged correctly
- The overwrite chain: file → sm → s3

##### `TestRealWorldScenario`
Simulates a real-world usage pattern:
```
.env.defaults      (default values, committed to repo)
    ↓
.env.local         (local overrides, not committed)
    ↓
sm://production/*  (production secrets from AWS)
```

##### `TestSourceOrderMatters`
Verifies that source order determines precedence with the same key in multiple sources.

## Mock Strategy

### Why Mocking?

The AWS resolvers (sm:// and s3://) require:
- AWS credentials
- Network access
- Actual AWS resources

To test the integration without these dependencies, we use **mock resolvers**:

```go
type mockResolver struct {
    items map[string]string
    err   error
}

func (m *mockResolver) Resolve(source *config.Source) *resolvers.Result {
    result := &resolvers.Result{Source: source}
    if m.err != nil {
        result.AppendError(m.err)
        return result
    }
    result.AppendItems(m.items)
    return result
}
```

This allows us to:
1. Test the full integration flow without AWS
2. Control the exact values returned by each source
3. Verify the merge and overwrite behavior
4. Test error scenarios

## Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run specific integration test
go test ./pkg/app/... -v -run TestMultiSourceIntegrationWithMocks

# Run with verbose output
go test ./... -v
```

## Coverage Summary

Current test coverage:
- `pkg/app`: 100% (full integration coverage)
- `pkg/config`: 100%
- `pkg/formatters`: 96.4%
- `pkg/resolvers`: 41.7% (unit tests only, AWS parts untested without credentials)

## Testing Multiple Sources End-to-End

To test with real AWS services:

```bash
# Ensure AWS credentials are configured
export AWS_PROFILE=your-profile

# Test with multiple real sources
snagsby file://./example.env sm://production/secrets/* s3://my-bucket/config.json

# The order matters - later sources overwrite earlier ones
```

## Key Testing Insights

1. **Merge Behavior**: The `formatters.Merge()` function uses `maps.Copy()` which ensures later maps overwrite earlier ones (tested in `TestMerge`)

2. **Parallel Resolution**: Sources are resolved concurrently via goroutines (tested in `TestParallelResolution`)

3. **Error Handling**: Errors from one source don't prevent other sources from resolving (tested in various error scenarios)

4. **Key Normalization**: All keys are converted to UPPERCASE with special characters replaced by underscores (tested in `TestKeyRegexp`)

## Future Testing Improvements

Consider adding:
1. **End-to-end tests with LocalStack** for AWS service mocking
2. **Benchmark tests** for large numbers of secrets
3. **Concurrency stress tests** with many sources
4. **Integration tests with actual AWS resources** (in a CI environment with proper credentials)
