#!/bin/bash

# Ensure that the tools needed to build locally are present
set -xeuo pipefail

export CURPATH=`pwd`
export BIN_DIR=$CURPATH/bin
export GO111MODULE=on

GOBIN=${BIN_DIR} go install -mod=mod golang.org/x/lint/golint@latest
