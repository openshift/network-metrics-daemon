#!/bin/bash

# Ensure that the tools needed to build locally are present
set -xeuo pipefail

export CURPATH=`pwd`
export BIN_DIR=$CURPATH/bin
export GO111MODULE=on

# DEBUG: show environment
echo "=== DEBUG ENV ==="
echo "PATH=$PATH"
echo "GOPATH=$GOPATH"
echo "GOBIN=${GOBIN:-not set}"
which golint 2>/dev/null && echo "golint pre-installed at: $(which golint)" || echo "golint not pre-installed"
echo "=== END DEBUG ==="

GOBIN=${BIN_DIR} go install -mod=mod golang.org/x/lint/golint@latest
echo "golint installed to: $(ls -la ${BIN_DIR}/golint 2>/dev/null || echo 'NOT FOUND')"
