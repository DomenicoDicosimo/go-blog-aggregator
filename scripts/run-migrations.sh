#!/bin/sh

# Wait for the database to be ready
echo "Waiting for database to be ready..."
while ! pg_isready -h db -U postgres > /dev/null 2>&1; do
  sleep 1
done

echo "Running database migrations..."
goose -dir ./sql/schema postgres "$DB" up

echo "Migrations completed."