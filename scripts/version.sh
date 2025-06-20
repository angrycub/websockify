#!/usr/bin/env bash

# This script generates the version string for websockify.
# Similar to coder/coder's version.sh but simplified for this project.

set -euo pipefail

# Change to the root of the git repository
cd "$(dirname "${BASH_SOURCE[0]}")/.."

# If WEBSOCKIFY_FORCE_VERSION is set, use that
if [[ -n "${WEBSOCKIFY_FORCE_VERSION:-}" ]]; then
    echo "${WEBSOCKIFY_FORCE_VERSION}"
    exit 0
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    echo "0.1.0-devel+unknown"
    exit 0
fi

# Get the current commit
current_commit=$(git rev-parse HEAD)

# Try to find tags
tag_list=$(git tag --sort=version:refname 2>/dev/null || echo "")

if [[ -z "${tag_list}" ]]; then
    # No tags found, use default version with commit
    short_commit=$(git rev-parse --short HEAD)
    echo "0.1.0-devel+${short_commit}"
    exit 0
fi

# Get the latest tag
last_tag=$(git tag --sort=version:refname | tail -n 1)

# Check if we're exactly on a tag
if git describe --exact-match --tags HEAD >/dev/null 2>&1; then
    # We're on a tag, check if this is a release build
    if [[ "${WEBSOCKIFY_RELEASE:-}" == "true" ]]; then
        echo "${last_tag#v}"  # Remove 'v' prefix if present
    else
        # Development build even on tag
        short_commit=$(git rev-parse --short HEAD)
        echo "${last_tag#v}-devel+${short_commit}"
    fi
else
    # We're not on a tag, this is a development version
    short_commit=$(git rev-parse --short HEAD)
    echo "${last_tag#v}-devel+${short_commit}"
fi