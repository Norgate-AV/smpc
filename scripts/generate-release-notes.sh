#!/bin/bash

set -e

if [ "$CI" != "true" ]; then
    echo "Not running in CI, skipping release notes generation"
    exit 0
fi

git-cliff --latest --verbose | sed -n '/^###/,$p' | sed -e '${/^$/d}' >RELEASE_NOTES.md

# Verify the release notes were generated
if [ ! -s RELEASE_NOTES.md ]; then
    echo "Release notes generation failed or produced an empty file"
    exit 1
fi
