#!/bin/sh
set -e

# first arg is `something.conf`
if [ "${1#-}" != "$1" ] || [ "${1%.conf}" != "$1" ]; then
	set -- imgry-server "$@"
fi

# allow the container to be started with `--user`
if [ "$1" = 'imgry-server' -a "$(id -u)" = '0' ]; then
	chown -R imgry .
	exec gosu imgry "$0" "$@"
fi

exec "$@"