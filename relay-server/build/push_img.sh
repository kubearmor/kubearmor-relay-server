#!/bin/bash

BASE_DIR=`dirname $(realpath "$0")`/../..
cd $BASE_DIR
[[ "$REPO" == "" ]] && REPO="sibashi/kubearmor-relay-server"
[[ "$PLATFORMS" == "" ]] && PLATFORMS="linux/amd64,linux/arm64/v8"

# check version
VERSION=latest

if [ ! -z $1 ]; then
    VERSION=$1
fi

echo "[INFO] Pushing $REPO:$VERSION"
docker buildx build --platform $PLATFORMS --push -t $REPO:$VERSION -f $BASE_DIR/relay-server/Dockerfile .

if [ $? != 0 ]; then
    echo "[FAILED] Failed to push $REPO:$VERSION"
    exit 1
fi
echo "[PASSED] Pushed $REPO:$VERSION"
