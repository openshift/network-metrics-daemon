#!/bin/bash

# check_gofmt.sh
# Fail if a .go file hasn't been formatted with gofmt
set -euo pipefail
modified=$(go fmt $(go list ./... | grep -v /vendor/))
echo $modified
test -z $modified
