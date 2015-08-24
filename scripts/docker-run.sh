#!/bin/bash
set -e

if [ ! -f $CONFIG ]; then
	echo "\"$CONFIG\" file missing"
	exit 1
fi

sudo docker run -d \
	-p $HOST_PORT:$CONTAINER_PORT \
	-v $CONFIG:/etc/imgry.conf \
	-v /data:/data \
	--memory-swappiness=0 \
	--restart=always \
	--name $NAME $IMAGE
