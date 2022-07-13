#!/bin/sh

if [ "$1" = "eventline" ]; then
    uri="$EVENTLINE_PG_URI"
    if [ -z "$uri" ]; then
        uri="postgres://eventline:eventline@localhost:5432/eventline"
    fi

    echo "waiting for postgresql ($uri)"
    until psql -c '\q' "$uri"; do
        sleep 1
    done
    echo "postgresql ready"
fi

exec "$@"
