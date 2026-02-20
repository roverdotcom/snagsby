# AGENTS.md - Guide for AI Agents Working on Snagsby

## Project Overview

**Snagsby** is a Go-based command-line tool that reads configuration from AWS S3 buckets or AWS Secrets Manager and outputs environment variable exports suitable for shell evaluation. It's commonly used in Docker container workflows to inject configuration and secrets from AWS services into containers at runtime.

### Key Features
- Reads configuration from S3 (JSON objects) and AWS Secrets Manager (secret values, optionally JSON)
- Converts configuration to shell environment variable exports
- Supports multiple configuration sources with merge capability
- Works with AWS IAM roles, instance profiles, and task roles
- Handles multiple output formats (env, json)
- Can process multiple sources in parallel

### Technology Stack
- **Language**: Go 1.16+
- **AWS SDK**: aws-sdk-go-v2 (v1.17.3)
- **Key Dependencies**: 
  - AWS S3 SDK
  - AWS Secrets Manager SDK
  - sigs.k8s.io/yaml for YAML parsing
- **Build Tool**: Make with Goreleaser
- **Testing**: Go standard testing library
- **CI/CD**: GitHub Actions

## Repository Structure

```
snagsby/
├── main.go                 # Entry point - CLI flag parsing and orchestration
├── pkg/
│   ├── version.go         # Version information
│   ├── app/
│   │   └── app.go         # Main application logic (parallel source resolution)
│   ├── config/
│   │   ├── config.go      # Configuration parsing and source management
│   │   ├── config_test.go # Configuration tests
│   │   └── utils.go       # Configuration utilities
│   ├── formatters/
│   │   ├── format.go      # Output formatters (env, json)
│   │   └── format_test.go # Formatter tests
│   └── resolvers/
│       ├── aws.go         # AWS configuration setup
│       ├── resolvers.go   # Core resolver logic
│       ├── s3.go          # S3-specific resolver
│       ├── secretsmanager.go # Secrets Manager resolver with wildcard support
│       ├── manifest.go    # JSON manifest parsing
│       └── *_test.go      # Resolver tests
├── e2e/
│   ├── e2e.sh            # E2E test runner (requires AWS credentials)
│   └── e2e.py            # E2E test validation script
├── scripts/
│   ├── fpm.sh            # Legacy package creation script
│   └── DockerfileFpm     # FPM packaging Dockerfile
├── .goreleaser.yaml      # Goreleaser configuration
├── Makefile              # Build and test targets
├── Dockerfile            # Multi-stage Docker build
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
└── vendor/               # Vendored dependencies

```

## Building and Testing

### Prerequisites
- Go 1.16 or higher
- Make
- Goreleaser (for building distribution artifacts)
- Docker (optional, for containerized builds)

### Common Commands

#### Build
```bash
# Build binary locally
make build

# Build distribution binaries for all platforms (requires goreleaser)
make dist

# Build a snapshot release (test release build without publishing)
make release-snapshot

# Build in Docker container
make docker-build-images

# Install to $GOPATH/bin
make install

# Clean build artifacts
make clean
```

#### Testing
```bash
# Run all unit tests
make test

# Run tests in Docker
make docker-test

# Run e2e tests (requires AWS credentials and resources)
make e2e

# Quick e2e test (uses existing dist/ binaries)
make e2e-quick
```

#### Distribution
```bash
# Build distribution binaries for multiple platforms
make dist

# Build a snapshot release locally
make release-snapshot

# Actual releases are created automatically via GitHub Actions
# when a new tag is pushed (e.g., git tag v0.6.2 && git push --tags)
```

### Build Configuration
- **Version**: Managed via Git tags (e.g., `v0.6.1`) - no VERSION file needed
- **Build flags**: `-ldflags "-X github.com/roverdotcom/snagsby/pkg.Version={{.Version}}"`
- **CGO**: Disabled (`CGO_ENABLED=0`) for static binary compilation
- **Platforms**: Supports Linux and macOS, both amd64 and arm64
- **Release Tool**: Goreleaser handles builds, archives, and GitHub releases

## Code Architecture

### Main Components

1. **main.go**: Entry point
   - Parses CLI flags (`-v` version, `-e` fail on error, `-o` output format)
   - Initializes configuration from args and SNAGSBY_SOURCE env var
   - Coordinates resolution and formatting
   - Handles errors and exit codes

2. **pkg/config**: Configuration Management
   - `Config`: Manages list of sources to resolve
   - `Source`: Represents a single configuration source (URL + options)
   - Parses source URLs with query parameters (e.g., `?region=us-west-2`)
   - Supports comma-delimited source lists

3. **pkg/app**: Application Logic
   - `ResolveConfigSources()`: Resolves multiple sources in parallel using goroutines
   - Collects results from all sources
   - Returns array of results (successful or with errors)

4. **pkg/resolvers**: Source Resolution
   - `ResolveSource()`: Main entry point for resolving a source
   - **S3 Resolver**: Reads JSON objects from S3 buckets
     - URL format: `s3://bucket-name/path/to/config.json?region=us-west-2`
     - S3 object must contain JSON
   - **Secrets Manager Resolver**: Reads secrets from AWS Secrets Manager
     - Single secret: `sm://secret-name` - secret value contains JSON that gets expanded into multiple env vars
     - Wildcard: `sm:///path/prefix/*` - each secret value is used directly as a single env var value
     - AWS SM stores secret name/value pairs; single secrets can contain JSON values that snagsby expands
   - `readJSONString()`: Parses JSON into key-value pairs (used by S3 and single SM secrets)
     - Sanitizes keys (uppercase, replace special chars with underscore)
     - Handles string, number, boolean types
     - Converts multiline strings, booleans to shell format

5. **pkg/formatters**: Output Formatting
   - `env`: Shell export format (default)
   - `json`: JSON output format
   - `Merge()`: Merges multiple result maps (later sources override earlier)

### Data Flow
```
CLI Args/Env Var → Config → App (parallel resolution) → Resolvers (S3/SM) → 
  JSON Parser (for S3 and single SM secrets) → Merge → Formatter → Output
```

## Development Workflow

### Making Code Changes

1. **Understand the component**: Identify which package needs changes
   - Config parsing: `pkg/config/`
   - Resolution logic: `pkg/resolvers/`
   - Output formatting: `pkg/formatters/`
   - CLI/orchestration: `main.go` or `pkg/app/`

2. **Write tests first**: Add tests in corresponding `*_test.go` files
   - Follow existing test patterns
   - Use table-driven tests where appropriate
   - Mock AWS calls when needed

3. **Make minimal changes**: Keep changes focused and surgical
   - Preserve existing behavior unless explicitly changing it
   - Don't refactor unrelated code

4. **Test locally**:
   ```bash
   # Run unit tests
   make test
   
   # Build and test manually
   make build
   ./snagsby -v
   ```

5. **Verify build in Docker** (optional):
   ```bash
   make docker-test
   ```

### Code Conventions

- **Naming**: Follow Go standard conventions
  - Exported functions: PascalCase
  - Unexported functions: camelCase
  - Acronyms: Keep uppercase (e.g., `AWS`, `URL`, `JSON`)

- **Error Handling**: Return errors, don't panic
  - Use `fmt.Errorf()` for error wrapping
  - Return multiple errors where appropriate (see `Result.Errors`)

- **Testing**: Use Go standard testing
  - Test files: `*_test.go`
  - Table-driven tests for multiple scenarios
  - No external test frameworks beyond standard library

- **Comments**: Add comments for exported functions and complex logic
  - Follow godoc conventions
  - Explain "why" not "what" for non-obvious code

- **Dependencies**: Use Go modules with vendoring
  - Run `go mod vendor` after adding dependencies
  - Check vendored code into git

## AWS Configuration

Snagsby relies on AWS SDK v2 for credential resolution:
- IAM instance profiles (EC2)
- ECS task roles
- Shared credentials file (`~/.aws/credentials`)
- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- Shared config file (`~/.aws/config`)

Region can be specified:
- In source URL query parameter: `?region=us-west-2`
- Via `AWS_REGION` environment variable
- From AWS config files

## Testing Strategy

### Unit Tests
- Located in `*_test.go` files alongside source
- Test individual functions and components
- Mock AWS interactions (test files contain simple validation)
- Run with: `make test`

### E2E Tests
- Located in `e2e/` directory
- Require actual AWS resources and credentials
- Environment variable: `SNAGSBY_E2E_SOURCE` specifies test sources
- Validates actual AWS integration
- Run with: `make e2e` (requires AWS setup)

### CI/CD
- GitHub Actions workflow: `.github/workflows/main.yaml`
- Runs on push to master and PRs
- Executes `make docker-test` for consistent environment
- Uses Docker with Go 1.16.3 image

## Common Tasks

### Adding a New Resolver Type

1. Create new file in `pkg/resolvers/` (e.g., `newtype.go`)
2. Implement resolver function following pattern in `s3.go` or `secretsmanager.go`
3. Update `ResolveSource()` in `resolvers.go` to handle new URL scheme
4. Add tests in `newtype_test.go`
5. Update documentation in README.md

### Adding a New Output Format

1. Add formatter function in `pkg/formatters/format.go`
2. Register in `Formatters` map
3. Add tests in `format_test.go`
4. Update CLI help text in `main.go`

### Updating Dependencies

1. Update `go.mod`: `go get github.com/package/name@version`
2. Run `go mod tidy`
3. Run `go mod vendor` to update vendored dependencies
4. Test thoroughly: `make test`
5. Commit changes to `go.mod`, `go.sum`, and `vendor/`

### Debugging

- Use `-show-summary` flag to see what sources are being processed
- Use `-e` flag to fail on first error (helps identify issues)
- Check stderr for error messages (stdout is for exports only)
- For AWS issues, enable AWS SDK logging if needed

## Key Files Reference

- **Makefile**: All build targets and commands
- **.goreleaser.yaml**: Goreleaser configuration for builds and releases
- **go.mod**: Go module dependencies
- **main.go**: CLI entry point and flag handling
- **pkg/app/app.go**: Parallel source resolution orchestration
- **pkg/config/config.go**: Source configuration parsing
- **pkg/resolvers/resolvers.go**: Main resolver dispatcher
- **pkg/resolvers/s3.go**: S3 bucket resolution
- **pkg/resolvers/secretsmanager.go**: AWS Secrets Manager resolution (supports wildcards; single secrets contain JSON expanded into multiple env vars)
- **pkg/resolvers/aws.go**: AWS configuration and JSON parsing utilities
- **pkg/formatters/format.go**: Output format implementations
- **.github/workflows/release.yaml**: GitHub Actions workflow for automated releases

## Security Considerations

- **Credentials**: Never commit AWS credentials
- **Secrets**: Tool is designed to handle secrets - ensure secure handling
- **IAM Permissions**: Tool needs appropriate S3 and Secrets Manager read permissions
- **Error Messages**: Be careful not to leak sensitive data in error messages
- **Vendored Dependencies**: Keep dependencies updated for security patches

## Troubleshooting

### Build Failures
- Ensure Go 1.16+ is installed: `go version`
- Check for missing dependencies: `go mod download`
- Clean and rebuild: `make clean && make build`

### Test Failures
- Unit tests should not require AWS credentials
- E2E tests require AWS resources - check `SNAGSBY_E2E_SOURCE`
- Check Go version compatibility

### Runtime Issues
- Verify AWS credentials are configured
- Check IAM permissions for S3/Secrets Manager
- Verify source URL format and region
- Use `-e` flag to see detailed errors
- Check stderr for error output

## Tips for AI Agents

1. **Make minimal changes**: This is a stable, production tool - avoid unnecessary refactoring
2. **Test thoroughly**: Always run `make test` after code changes
3. **Follow existing patterns**: Look at similar code in the same package
4. **Preserve backward compatibility**: Don't break existing URL formats or CLI flags
5. **Update tests**: Add/update tests for any new functionality
6. **Check vendored deps**: Remember to vendor after dependency changes
7. **Consider AWS**: Most functionality involves AWS SDK - understand async/concurrent patterns
8. **Review resolvers**: Most changes will be in `pkg/resolvers/` - understand the resolver pattern
9. **Shell output format**: Be careful with env format - it must be valid shell syntax
10. **Concurrent processing**: The app resolves sources in parallel - watch for race conditions

## Getting Help

- **Repository**: https://github.com/roverdotcom/snagsby
- **Issues**: Check existing issues for similar problems
- **README.md**: User-facing documentation and examples
- **Code Comments**: Most exported functions have documentation comments
- **AWS Blogs**: Original use case documented in AWS security blog (see README)
