# Refactoring: Shared BatchFetchSecrets Function

## Summary

Consolidated duplicate secret-fetching logic across three resolvers (`file.go`, `manifest.go`, `secretsmanager.go`) into a single shared function `BatchFetchSecrets` in `aws.go`.

## Problem

Three different resolvers had nearly identical code for:
- Creating AWS Secrets Manager client
- Using worker pools to fetch secrets concurrently
- Collecting results and errors

This duplication led to:
- More code to maintain
- Inconsistent error handling
- Harder to modify concurrent fetching behavior

## Solution

Created `BatchFetchSecrets(secretIDs []string, concurrency int) (map[string]string, []error)` in `pkg/resolvers/aws.go` that:
1. Takes a list of secret IDs (paths)
2. Fetches them concurrently using worker pools
3. Returns a map of secretID -> secretValue and any errors

## Changes

### New Code

**`pkg/resolvers/aws.go`**
- Added `BatchFetchSecrets` function (shared secret fetching logic)
- Added `secretFetchJob` struct (job definition)
- Added `secretFetchResult` struct (result definition)
- Added `secretFetchWorker` function (worker implementation)

### Refactored Code

**`pkg/resolvers/file.go`**
- **Removed**: `envFileResult` struct (no longer needed)
- **Removed**: `envFileWorker` function (~20 lines)
- **Simplified**: `resolveEnvFileItems` function
  - Now uses `BatchFetchSecrets` instead of custom worker pool
  - Went from ~50 lines to ~25 lines

**`pkg/resolvers/manifest.go`**
- **Removed**: `manifestResult` struct (no longer needed)
- **Removed**: `manifestWorker` function (~10 lines)
- **Removed**: `getSecretValue` function (~10 lines)
- **Simplified**: `resolveManifestItems` function
  - Now uses `BatchFetchSecrets` instead of custom worker pool
  - Went from ~35 lines to ~20 lines

**`pkg/resolvers/secretsmanager.go`**
- **Removed**: `smMessage` struct (no longer needed)
- **Removed**: `smWorker` function (~30 lines)
- **Removed**: `time` import (no longer needed)
- **Simplified**: `resolveRecursive` function
  - Now uses `BatchFetchSecrets` instead of custom worker pool
  - Went from ~80 lines to ~60 lines

### Code Reduction

- **Total lines removed**: ~140 lines of duplicated logic
- **Total lines added**: ~60 lines (shared implementation)
- **Net reduction**: ~80 lines
- **Duplicated patterns eliminated**: 3

## API

```go
// BatchFetchSecrets fetches multiple secrets from AWS Secrets Manager concurrently.
// It returns a map of secretID -> secretValue and a slice of errors encountered.
// The concurrency parameter controls the number of concurrent workers (0 = default of 20).
func BatchFetchSecrets(secretIDs []string, concurrency int) (map[string]string, []error)
```

### Parameters
- `secretIDs []string` - List of AWS secret IDs/paths to fetch
- `concurrency int` - Number of concurrent workers (0 or negative = default of 20)

### Returns
- `map[string]string` - Mapping of secretID → secretValue
- `[]error` - Any errors encountered during fetching

### Example Usage

```go
// Fetch multiple secrets
secretIDs := []string{
    "/apps/webapp/db-password",
    "/apps/webapp/api-token",
    "production/cache-key",
}

secretValues, errors := BatchFetchSecrets(secretIDs, 20)

// Handle errors
for _, err := range errors {
    log.Printf("Error fetching secret: %v", err)
}

// Use secret values
if password, ok := secretValues["/apps/webapp/db-password"]; ok {
    // Use password
}
```

## Usage in Resolvers

### file.go (`.env` file resolver)
```go
// Extract secret IDs from sm:// references
secretIDs := []string{}
for _, item := range items {
    if strings.HasPrefix(item.Value, "sm://") {
        secretID := strings.TrimPrefix(item.Value, "sm://")
        secretIDs = append(secretIDs, secretID)
    }
}

// Fetch all secrets at once
secretValues, errors := BatchFetchSecrets(secretIDs, 20)
```

### manifest.go (YAML manifest resolver)
```go
// Build list from manifest items
secretIDs := []string{}
for _, item := range manifestItems.Items {
    secretIDs = append(secretIDs, item.Name)
}

// Fetch all secrets at once
secretValues, errors := BatchFetchSecrets(secretIDs, 20)
```

### secretsmanager.go (sm:// resolver)
```go
// After ListSecrets pagination
secretIDs := []string{}
for _, secret := range secretList {
    secretIDs = append(secretIDs, *secret.Name)
}

// Fetch all secrets at once
secretValues, errors := BatchFetchSecrets(secretIDs, smConcurrency)
```

## Benefits

### 1. **Code Reusability**
- Single implementation of secret fetching logic
- Changes to fetching behavior only need to be made in one place

### 2. **Consistency**
- All resolvers use the same AWS client configuration
- Same retry logic (10 retries with backoff)
- Same error handling approach

### 3. **Maintainability**
- Less code to maintain (~80 lines less)
- Easier to add features (e.g., caching, rate limiting)
- Clearer separation of concerns

### 4. **Testability**
- Single function to test for batch fetching
- Easier to mock or stub for unit tests

### 5. **Performance**
- Consistent concurrency handling across all resolvers
- Easy to tune concurrency in one place

## Testing

### New Tests
- `TestBatchFetchSecretsEmpty` - Tests with no secrets
- `TestBatchFetchSecretsWithoutAWS` - Tests error handling without AWS
- `TestBatchFetchSecretsConcurrency` - Tests concurrency parameter

### Existing Tests
All existing tests for file.go, manifest.go, and secretsmanager.go pass without modification:
- `TestParseEnvFile` ✓
- `TestMultiSourceOrderPreservation` ✓
- All other resolver tests ✓

### Coverage
- **Before**: pkg/resolvers 41.7% coverage
- **After**: pkg/resolvers 46.1% coverage (+4.4%)

## Migration Guide

No migration needed - this is a purely internal refactoring. The public API of all resolvers remains unchanged.

## Future Enhancements

With this shared function, it's now easier to add:

1. **Caching** - Cache fetched secrets to avoid repeated AWS calls
2. **Rate Limiting** - Add rate limiting to respect AWS quotas
3. **Metrics** - Add metrics/logging for secret fetching
4. **Retries** - Enhance retry logic in one place
5. **Batch Size Limits** - Automatically split large batches

## Related Files

- `pkg/resolvers/aws.go` - Shared function implementation
- `pkg/resolvers/file.go` - Uses BatchFetchSecrets
- `pkg/resolvers/manifest.go` - Uses BatchFetchSecrets
- `pkg/resolvers/secretsmanager.go` - Uses BatchFetchSecrets
- `pkg/resolvers/aws_batch_test.go` - Tests for BatchFetchSecrets
