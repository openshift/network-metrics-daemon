#!/bin/bash

# Ensure that the tools needed to build locally are present
set -xeuo pipefail

GO111MODULE=off go get golang.org/x/lint/golint
