#!/bin/bash

# Ensure that the tools needed to build locally are present
set -xeuo pipefail

export CURPATH=`pwd`
export BIN_DIR=$CURPATH/bin
export GO111MODULE=on
export PATH="${BIN_DIR}:${PATH}"

go build -o ${BIN_DIR}/golint golang.org/x/lint/golint
