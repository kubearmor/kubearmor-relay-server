#!/bin/bash

set -euo pipefail

BASE_DIR=$(dirname "$(realpath "$0")")/../..
cd "$BASE_DIR"
REPO=${REPO:-kubearmor/kubearmor-relay-server}
# Keep this list consistent with ci-latest-release.yml.
PLATFORMS=${PLATFORMS:-linux/amd64,linux/arm64/v8}
VERSION=${1:-latest}

echo "[INFO] Pushing $REPO:$VERSION"
if ! docker buildx build --metadata-file kubearmor-relay-server.json \
    --platform "$PLATFORMS" --build-arg "VERSION=$VERSION" --push \
    -t "$REPO:$VERSION" -f "$BASE_DIR/relay-server/Dockerfile" .; then
    echo "[FAILED] Failed to push $REPO:$VERSION"
    exit 1
fi
echo "[PASSED] Pushed $REPO:$VERSION"
