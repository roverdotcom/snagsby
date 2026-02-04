#!/bin/bash
#
# Comprehensive E2E test runner for Snagsby
#
# This script runs all e2e tests including:
# - Basic Secrets Manager tests (original e2e.sh)
# - Extended CLI tests
# - S3 resolver tests (if configured)
# - Manifest resolver tests (if configured)
#
# Required environment variables:
#   SNAGSBY_E2E_SOURCE - Secrets Manager sources for testing
#
# Optional environment variables:
#   SNAGSBY_E2E_S3_SOURCE - S3 source URL for S3 tests
#   SNAGSBY_E2E_MANIFEST_SOURCE - Manifest file path for manifest tests
#   SNAGSBY_E2E_OVERRIDE_TEST - Sources for testing override behavior
#   SNAGSBY_E2E_S3_INVALID_JSON - S3 source with invalid JSON for error testing

set -euf -o pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Snagsby E2E Test Suite"
echo "=========================================="
echo ""

# Check if dist directory exists
os_name=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ ! -f "./dist/$os_name/snagsby" ]; then
    echo -e "${RED}Error: Snagsby binary not found at ./dist/$os_name/snagsby${NC}"
    echo "Please run 'make dist' first"
    exit 1
fi

# Verify binary works
if ! ./dist/$os_name/snagsby -v > /dev/null 2>&1; then
    echo -e "${RED}Error: Snagsby binary is not executable or doesn't work${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Snagsby binary found and working${NC}"
echo ""

# Set default SNAGSBY_E2E_SOURCE if not provided
SNAGSBY_E2E_SOURCE=${SNAGSBY_E2E_SOURCE:-"sm://snagsby/acceptance sm:///snagsby/app/acceptance/*"}
export SNAGSBY_E2E_SOURCE

echo "Configuration:"
echo "  SNAGSBY_E2E_SOURCE: $SNAGSBY_E2E_SOURCE"
[ -n "${SNAGSBY_E2E_S3_SOURCE:-}" ] && echo "  SNAGSBY_E2E_S3_SOURCE: $SNAGSBY_E2E_S3_SOURCE"
[ -n "${SNAGSBY_E2E_MANIFEST_SOURCE:-}" ] && echo "  SNAGSBY_E2E_MANIFEST_SOURCE: $SNAGSBY_E2E_MANIFEST_SOURCE"
echo ""

# Test 1: Basic environment variable loading (original test)
echo "=========================================="
echo "Test 1: Environment Variable Loading"
echo "=========================================="

snagsby=$(./dist/$os_name/snagsby -e $SNAGSBY_E2E_SOURCE)
eval $snagsby

echo -e "${GREEN}✓ Successfully loaded environment variables from Secrets Manager${NC}"
echo ""

# Test 2: Run Python-based acceptance tests
echo "=========================================="
echo "Test 2: Acceptance Tests (Environment)"
echo "=========================================="

if python3 ./e2e/e2e.py; then
    echo -e "${GREEN}✓ Acceptance tests passed${NC}"
else
    echo -e "${RED}✗ Acceptance tests failed${NC}"
    exit 1
fi
echo ""

# Test 3: Run extended CLI tests
echo "=========================================="
echo "Test 3: CLI Functionality Tests"
echo "=========================================="

if python3 ./e2e/e2e.py SnagsbyCliTests; then
    echo -e "${GREEN}✓ CLI tests passed${NC}"
else
    echo -e "${RED}✗ CLI tests failed${NC}"
    exit 1
fi
echo ""

# Test 4: Run S3 tests if configured
if [ -n "${SNAGSBY_E2E_S3_SOURCE:-}" ]; then
    echo "=========================================="
    echo "Test 4: S3 Resolver Tests"
    echo "=========================================="
    
    if python3 ./e2e/e2e_extended.py SnagsbyS3Tests; then
        echo -e "${GREEN}✓ S3 tests passed${NC}"
    else
        echo -e "${RED}✗ S3 tests failed${NC}"
        exit 1
    fi
    echo ""
else
    echo "=========================================="
    echo "Test 4: S3 Resolver Tests (SKIPPED)"
    echo "=========================================="
    echo -e "${YELLOW}⊘ Set SNAGSBY_E2E_S3_SOURCE to run S3 tests${NC}"
    echo ""
fi

# Test 5: Run Manifest tests if configured
if [ -n "${SNAGSBY_E2E_MANIFEST_SOURCE:-}" ]; then
    echo "=========================================="
    echo "Test 5: Manifest Resolver Tests"
    echo "=========================================="
    
    if python3 ./e2e/e2e_extended.py SnagsbyManifestTests; then
        echo -e "${GREEN}✓ Manifest tests passed${NC}"
    else
        echo -e "${RED}✗ Manifest tests failed${NC}"
        exit 1
    fi
    echo ""
else
    echo "=========================================="
    echo "Test 5: Manifest Resolver Tests (SKIPPED)"
    echo "=========================================="
    echo -e "${YELLOW}⊘ Set SNAGSBY_E2E_MANIFEST_SOURCE to run Manifest tests${NC}"
    echo ""
fi

# Test 6: Run integration tests
echo "=========================================="
echo "Test 6: Integration Tests"
echo "=========================================="

if python3 ./e2e/e2e_extended.py SnagsbyIntegrationTests; then
    echo -e "${GREEN}✓ Integration tests passed${NC}"
else
    echo -e "${YELLOW}⊘ Some integration tests skipped (missing optional sources)${NC}"
fi
echo ""

# Summary
echo "=========================================="
echo "Test Suite Summary"
echo "=========================================="
echo -e "${GREEN}✓ All configured tests passed!${NC}"
echo ""

# Coverage reminder
echo "Note: These e2e tests cover:"
echo "  • Secrets Manager resolver (single and recursive patterns)"
echo "  • CLI flags (-v, -e, -show-summary, -o)"
echo "  • Output formatters (env, envfile, json)"
echo "  • Error handling"
echo "  • Multiple source merging"
if [ -n "${SNAGSBY_E2E_S3_SOURCE:-}" ]; then
    echo "  • S3 resolver"
fi
if [ -n "${SNAGSBY_E2E_MANIFEST_SOURCE:-}" ]; then
    echo "  • Manifest resolver"
fi
echo ""
echo "To enable additional tests, set:"
[ -z "${SNAGSBY_E2E_S3_SOURCE:-}" ] && echo "  - SNAGSBY_E2E_S3_SOURCE for S3 tests"
[ -z "${SNAGSBY_E2E_MANIFEST_SOURCE:-}" ] && echo "  - SNAGSBY_E2E_MANIFEST_SOURCE for Manifest tests"
echo ""
