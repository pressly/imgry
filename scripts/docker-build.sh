#!/bin/bash
set -e

cd /tmp/$IMAGE

# Cleanup.
sudo rm -rf bin

# Build our image
sudo docker build -t $IMAGE .
