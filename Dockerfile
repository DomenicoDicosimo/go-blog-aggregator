FROM golang:1.22-alpine

WORKDIR /app

# Install PostgreSQL client and build dependencies
RUN apk add --no-cache postgresql-client gcc musl-dev

# Install Goose
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Debug: List contents of /app
RUN ls -R /app

# Debug: Print current working directory
RUN pwd

# Build the application
RUN go build -o bin/api ./cmd/api

# Expose the application port
EXPOSE 4000

# Command to run the application
CMD ["./bin/api"]
