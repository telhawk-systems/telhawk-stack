#!/bin/sh
# Initialize multiple databases in single PostgreSQL instance
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE telhawk_auth;
    CREATE DATABASE telhawk_respond;
    GRANT ALL PRIVILEGES ON DATABASE telhawk_auth TO telhawk;
    GRANT ALL PRIVILEGES ON DATABASE telhawk_respond TO telhawk;
EOSQL

echo "Created databases: telhawk_auth, telhawk_respond"
