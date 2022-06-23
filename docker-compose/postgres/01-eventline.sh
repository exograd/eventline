#!/bin/sh

set -eu

# Note that the test suite will reset the eventline_test database by deleting
# the public schema and recreating it itself.

psql -v ON_ERROR_STOP=1 -U $POSTGRES_USER <<-EOF
CREATE USER eventline;
ALTER USER eventline PASSWORD 'eventline';

CREATE DATABASE eventline;
GRANT ALL PRIVILEGES ON DATABASE eventline TO eventline;

CREATE DATABASE eventline_test;
GRANT ALL PRIVILEGES ON DATABASE eventline_test TO eventline;
EOF

psql -v ON_ERROR_STOP=1 -U $POSTGRES_USER -d eventline <<-EOF
ALTER SCHEMA public OWNER TO eventline;
GRANT ALL ON SCHEMA public TO eventline;
EOF

psql -v ON_ERROR_STOP=1 -U $POSTGRES_USER -d eventline_test <<-EOF
ALTER SCHEMA public OWNER TO eventline;
GRANT ALL ON SCHEMA public TO eventline;
EOF
