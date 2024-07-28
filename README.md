# Go Blog Aggregator

This project is a blog aggregator built with Go. It uses Docker for development and PostgreSQL for the database.

## Prerequisites

- Docker
- Docker Compose

## Getting Started

## Environment Setup

Create a `.env` file in the root directory with the following variables:
PORT=8080
DB=postgres://postgres:postgres@db:5432/blogaggregator?sslmode=disable

### Development Environment

To start the development environment:

1. Build and start the containers:

./scripts/dev.sh

This script builds the Docker images and starts the containers defined in `docker-compose.yml`.

2. The application will be available at `http://localhost:8080`.

### Database Console

To access the PostgreSQL database console:

./scripts/db-console.sh

- Manually run migrations (if needed):

./scripts/run-migrations.sh

### Docker Commands

- Stop and remove containers:

docker-compose down


- Stop containers but keep volumes:

docker-compose stop


- View container logs:

docker-compose logs


- Rebuild Docker images (after changes):

docker-compose build

## Testing

Run tests (uses Testcontainers for a separate PostgreSQL instance):
go test ./...

## Troubleshooting
If you encounter permission issues with shell scripts:
chmod +x scripts/*.sh

If the app can't connect to the database, check container health:
docker-compose ps

To completely reset your database:
docker-compose down -v