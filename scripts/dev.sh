#!/bin/bash

# Run migrations
./scripts/run-migrations.sh

# If migrations succeed, start the application
if [ $? -eq 0 ]; then
    docker-compose up --build
else
    echo "Migrations failed. Please review and fix the issues before starting the application."
fi