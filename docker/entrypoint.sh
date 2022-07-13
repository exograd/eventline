#!/bin/sh

set -eu
set -o pipefail

echo "waiting for postgresql"
until psql --quiet $EVENTLINE_PG_URI -c "\q"; do
    sleep 1
done
echo "postgresql ready"

exec "$@"
