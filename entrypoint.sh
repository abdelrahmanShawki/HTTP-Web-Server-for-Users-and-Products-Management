#!/bin/sh

# Wait for postgres to be ready
echo "Waiting for postgres..."
# while ! nc -z db 5432; do
while ! PGPASSWORD="${POSTGRES_PASSWORD}" pg_isready -h db -p 5432 -U "${POSTGRES_USER}" -d "${POSTGRES_DB}"; do
    echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done
echo "PostgreSQL started"

# Run migrations
echo "Running migrations..."
migrate -path=./migrations -database="${DSN_BUY_DB}" up

# Start the application
echo "Starting application..."
exec ./api