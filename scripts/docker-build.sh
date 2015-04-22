#!/bin/bash

cd /tmp/$IMAGE || exit 1

# Cleanup.
sudo rm -rf bin

# Build our image
sudo docker build -t $IMAGE .
