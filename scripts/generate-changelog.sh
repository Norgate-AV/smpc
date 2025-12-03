#!/bin/bash

set -e

if [ "$CI" != "true" ]; then
    echo "Not running in CI, skipping changelog generation"
    exit 0
fi

git-cliff --output CHANGELOG.md --verbose

# Verify the changelog was generated
if [ ! -s CHANGELOG.md ]; then
    echo "Changelog generation failed or produced an empty file"
    exit 1
fi
