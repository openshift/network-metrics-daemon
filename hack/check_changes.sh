#!/bin/bash

if [[ -n "$(git status --porcelain)" ]]; then
        echo "uncommitted generated files."
        exit 1
fi
