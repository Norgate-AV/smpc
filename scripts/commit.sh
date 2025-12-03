#!/bin/bash

set -e

if [ "$CI" != "true" ]; then
    echo "Not running in CI, skipping commit"
    exit 0
fi

TAG="${1}"
if [ -z "$TAG" ]; then
    echo "Usage: $0 <tag>"
    echo "Example: $0 v0.6.0"
    exit 1
fi

git config user.name "github-actions[bot]" || {
    echo "Failed to configure git user.name"
    exit 1
}
git config user.email "github-actions[bot]@users.noreply.github.com" || {
    echo "Failed to configure git user.email"
    exit 1
}

# Determine the default branch
# In CI with detached HEAD, get it from the remote
DEFAULT_BRANCH=$(git remote show origin | grep 'HEAD branch' | cut -d' ' -f5)

if [ -z "$DEFAULT_BRANCH" ]; then
    echo "Could not determine default branch, falling back to 'master'"
    DEFAULT_BRANCH="master"
fi

echo "Target branch: $DEFAULT_BRANCH"

# Files to check and stage
FILES=("CHANGELOG.md")

for file in "${FILES[@]}"; do
    if [ -n "$(git status --porcelain "$file")" ]; then
        echo "Staging $file"
        git add "$file" || {
            echo "Failed to stage $file"
            exit 1
        }
    else
        echo "No changes to $file to commit"
    fi
done

git commit -m "chore: bump version to $TAG [skip ci]" || {
    echo "Failed to commit changes"
    exit 1
}
echo "All changes committed to $DEFAULT_BRANCH"

git push origin HEAD:"$DEFAULT_BRANCH" || {
    echo "Failed to push changes to $DEFAULT_BRANCH"
    exit 1
}
echo "Changes pushed to $DEFAULT_BRANCH"
