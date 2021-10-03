#!/bin/bash

if [ $# -eq 0 ]; then
    # generate config
    echo -e "Title = \"$TITLE\"" >> /etc/godns.conf
    echo -e "Version = \"$VERSION\"" >> /etc/godns.conf
    echo -e "Author = \"$AUTHOR\"" >> /etc/godns.conf
    echo -e "Debug = $DEBUG" >> /etc/godns.conf
    echo -e "[server]" >> /etc/godns.conf
    echo -e "host = \"0.0.0.0\"" >> /etc/godns.conf
    echo -e "port = $PORT" >> /etc/godns.conf
    echo -e "[resolv]" >> /etc/godns.conf
    echo -e "server-list-file = \"$SERVER_LIST_FILE\"" >> /etc/godns.conf
    echo -e "resolv-file = \"$RESOLV_FILE\"" >> /etc/godns.conf
    echo -e "timeout = $TIMEOUT" >> /etc/godns.conf
    echo -e "interval = $INTERVAL" >> /etc/godns.conf
    echo -e "setedns0 = $SETEDNS0" >> /etc/godns.conf
    echo -e "[log]" >> /etc/godns.conf
    echo -e "stdout = $LOG_STDOUT" >> /etc/godns.conf
    echo -e "file = \"$LOG_FILE\"" >> /etc/godns.conf
    echo -e "level = \"$LOG_LEVEL\"" >> /etc/godns.conf
    echo -e "[cache]" >> /etc/godns.conf
    echo -e "backend = \"$CACHE_BACKEND\"" >> /etc/godns.conf
    echo -e "expire = $CACHE_EXPIRE" >> /etc/godns.conf
    echo -e "maxcount = $CACHE_MAXCOUNT" >> /etc/godns.conf
    echo -e "refresh = $CACHE_REFRESH" >> /etc/godns.conf
    echo -e "no-negative = $CACHE_NO_NEGATIVE" >> /etc/godns.conf
    echo -e "[hosts]" >> /etc/godns.conf
    echo -e "enable = $HOSTS_ENABLE" >> /etc/godns.conf
    echo -e "host-file = \"$HOSTS_HOST_FILE\"" >> /etc/godns.conf
    echo -e "redis-enable = $HOSTS_REDIS_ENABLE" >> /etc/godns.conf
    echo -e "redis-key = \"$HOSTS_REDIS_KEY\"" >> /etc/godns.conf
    echo -e "ttl = $HOSTS_TTL" >> /etc/godns.conf
    echo -e "refresh-interval = $HOSTS_REFRESH_INTERVAL" >> /etc/godns.conf

    /app/godns -c /etc/godns.conf
else
    exec "$@"
fi