#!/bin/bash
set -e

if [ ! -f $CONFIG ]; then
	echo "\"$CONFIG\" file missing"
	exit 1
fi

sudo docker daemon \
	-p $HOST_PORT:$CONTAINER_PORT \
	-v $CONFIG:/etc/imgry.conf \
	-v /data:/data \
	--memory-swappiness=0 \
	--restart=always \
	--log-opt max-size=100m \
	--log-opt max-file=5 \
	--name $NAME $IMAGE
