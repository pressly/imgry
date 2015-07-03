#!/bin/bash

if [ "$1" == "--long" ]; then
	git describe --tags --long --dirty
else
	version=$(git describe --tags --abbrev=0)
	echo ${version:1} # removes prefix "v"
fi
