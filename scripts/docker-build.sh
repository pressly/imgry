#!/bin/bash
set -e

cd $WORKDIR

# Cleanup.
sudo rm -rf bin

# Build the resulting image. Tag it with version, then retag the latest.
VERSION=$(scripts/version.sh --long)
sudo docker build --rm -t $IMAGE:$VERSION .
sudo docker tag $IMAGE:$VERSION $IMAGE:latest
