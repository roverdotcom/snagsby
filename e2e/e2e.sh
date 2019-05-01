#!/bin/bash

set -euf -o pipefail

SNAGSBY_E2E_SOURCE=${SNAGSBY_E2E_SOURCE:-sm://snagsby/acceptance}
os_name=$(uname -s | tr '[:upper:]' '[:lower:]')

# Evaluate snagsby
export SNAGSBY_BIN=${SNAGSBY_BIN:-./dist/$os_name/snagsby}

snagsby=$($SNAGSBY_BIN -e $SNAGSBY_E2E_SOURCE)
eval $snagsby

python ./e2e/e2e.py -v
