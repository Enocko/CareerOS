#!/bin/sh
set -eu

if [ -n "${DATABASE_URL:-}" ]; then
  echo "Running database migrations..."
  migrate -path /app/migrations -database "$DATABASE_URL" up
fi

exec /app/service
