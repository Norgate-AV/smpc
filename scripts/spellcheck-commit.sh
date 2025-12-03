#!/bin/bash
set -e

# This script checks a commit message for spelling errors.
# It checks for tools in an order of preference: `typos`, then `cspell`.
# If no spell checker is found, it exits gracefully.

COMMIT_MSG_FILE=$1

if ! [ -f "$COMMIT_MSG_FILE" ]; then
    echo "Commit message file not found: $COMMIT_MSG_FILE"
    exit 1
fi

# Check for spell checkers in order of preference
if command -v typos >/dev/null 2>&1; then
    typos "$COMMIT_MSG_FILE"
elif command -v cspell >/dev/null 2>&1; then
    cspell lint --no-summary --no-progress "$COMMIT_MSG_FILE"
fi
