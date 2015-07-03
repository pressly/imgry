#!/bin/bash
set -e

cd $WORKDIR

# Cleanup.
sudo rm -rf bin

# Build the resulting image. Tag it with version, then retag the latest.
VERSION=$(scripts/version.sh --long)
docker build --rm --no-cache -t $IMAGE:$VERSION .
docker tag -f $IMAGE:$VERSION $IMAGE:latest
