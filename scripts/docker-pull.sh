#!/bin/bash

# Clone repo for the first time
if [[ ! -d /tmp/$IMAGE || ! -d /tmp/$IMAGE/.git ]]; then
	mkdir -p /tmp/$IMAGE
	git clone -b $BRANCH $REPO /tmp/$IMAGE
fi

cd /tmp/$IMAGE

sudo chown -R $USER:$USER ./
git reset --hard
git clean -f -d
git checkout $BRANCH || git checkout -b $BRANCH origin/$BRANCH
git pull
