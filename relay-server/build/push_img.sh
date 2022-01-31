#!/bin/bash

[[ "$REPO" == "" ]] && REPO="kubearmor/kubearmor-relay-server"

# check version
VERSION=latest

if [ ! -z $1 ]; then
    VERSION=$1
fi

echo "[INFO] Pushing $REPO:$VERSION"
docker push $REPO:$VERSION

if [ $? != 0 ]; then
    echo "[FAILED] Failed to push $REPO:$VERSION"
    exit 1
fi
echo "[PASSED] Pushed $REPO:$VERSION"
