# Include environment variables from .env file
include .env

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## dev/build: build the development environment
.PHONY: dev/build
dev/build:
	docker-compose build

## dev/up: start the development environment
.PHONY: dev/up
dev/up: db/migrations/up
	docker-compose up

## dev/down: stop the development environment
.PHONY: dev/down
dev/down:
	docker-compose down

## dev/rebuild: rebuild and restart the development environment
.PHONY: dev/rebuild
dev/rebuild: dev/down dev/build dev/up

## db/console: access the database console
.PHONY: db/console
db/console:
	docker-compose run --rm psql

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format all .go files, and tidy module dependencies
.PHONY: tidy
tidy:
	@echo 'Formatting .go files...'
	go fmt ./...
	@echo 'Tidying module dependencies...'
	go mod tidy
	@echo 'Verifying module dependencies...'
	go mod verify

## audit: run quality control checks
.PHONY: audit
audit:
	@echo 'Checking module dependencies'
	go mod tidy
	go mod verify
	@echo 'Vetting code...'
	go vet ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## ops/logs: view Docker logs
.PHONY: ops/logs
ops/logs:
	docker-compose logs

## ops/clean: stop and remove containers, delete volumes
.PHONY: ops/clean
ops/clean: confirm
	docker-compose down -v

## test: run tests in Docker environment
.PHONY: test
test:
	docker-compose run --rm app go test ./...

# ==================================================================================== #
# CI/CD
# ==================================================================================== #

## ci/test: run tests for CI
.PHONY: ci/test
ci/test:
	go test ./... -cover

## ci/lint: run linters for CI
.PHONY: ci/lint
ci/lint:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec ./...
	test -z "$$(go fmt ./...)"
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...