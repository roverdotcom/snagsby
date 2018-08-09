#!/bin/bash

set -euf -o pipefail

SNAGSBY_E2E_SOURCE=${SNAGSBY_E2E_SOURCE:-sm://snagsby/acceptance}
os_name=$(uname -s | tr '[:upper:]' '[:lower:]')

# Evaluate snagsby
snagsby=$(./dist/$os_name/snagsby -e $SNAGSBY_E2E_SOURCE)
eval $snagsby

python ./e2e/e2e.py
