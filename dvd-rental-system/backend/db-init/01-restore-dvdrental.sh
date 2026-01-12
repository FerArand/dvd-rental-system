#!/bin/bash
set -e
# This script runs automatically on first-time DB init in the postgres container.
# It restores the sample dvdrental database from a tar dump if present.
if [ -f /docker-entrypoint-initdb.d/dvdrental.tar ]; then
  echo "Restoring dvdrental.tar ..."
  pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" /docker-entrypoint-initdb.d/dvdrental.tar
else
  echo "WARNING: /docker-entrypoint-initdb.d/dvdrental.tar not found. Place your dvdrental.tar here."
fi
