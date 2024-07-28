FROM golang:1.22-alpine

WORKDIR /app

# Install Goose and PostgreSQL client
RUN apk add --no-cache postgresql-client && \
    go install github.com/pressly/goose/v3/cmd/goose@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN go build -o main ./cmd/api

# Run the migration script
RUN chmod +x /app/scripts/run-migrations.sh

CMD ["/bin/sh", "-c", "./scripts/run-migrations.sh && ./main"]
