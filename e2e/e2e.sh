#!/bin/bash

set -euf -o pipefail

os_name=$(uname -s | tr '[:upper:]' '[:lower:]')
export SNAGSBY_E2E_SOURCE=${SNAGSBY_E2E_SOURCE:-sm://snagsby/acceptance}
export SNAGSBY_BIN=${SNAGSBY_BIN:-./dist/$os_name/snagsby}


# Evaluate snagsby
snagsby=$($SNAGSBY_BIN -e $SNAGSBY_E2E_SOURCE)
eval $snagsby

python ./e2e/e2e.py
