#!/usr/bin/env bash
set -euo pipefail

# Usage: ./build.sh 1.2.3
APP_VERSION=$1

# get the current git commit
GIT_COMMIT=$(git rev-parse --short HEAD)
# ISO8601 UTC timestamp
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Building r53q v${APP_VERSION} (commit ${GIT_COMMIT}, date ${BUILD_DATE})"

go build -ldflags "\
  -X 'main.appVersion=${APP_VERSION}' \
  -X 'main.gitCommit=${GIT_COMMIT}' \
  -X 'main.buildDate=${BUILD_DATE}'" \
  -o r53q main.go
