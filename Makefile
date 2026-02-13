BINARY_NAME=blogengine
MAIN_PACKAGE=./cmd/blogengine

# export GOEXPERIMENT := jsonv2

# load .env if exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## install-tools: install required binaries for development
.PHONY: install-tools
install-tools:
	go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/a-h/templ/cmd/templ@latest

.PHONY: audit
audit:
	@echo "1. Tidying dependencies..."
	@go mod tidy
	@echo "2. Formatting..."
	@go fmt ./...
	@echo "3. Vetting code..."
	@go vet ./...
	@echo "4. Static Check..."
	@if command -v staticcheck >/dev/null; then staticcheck ./...; else echo "staticcheck not found, skipping"; fi
	@echo "5. Running tests with race detector..."
	@go test -race -tags=integration ./...

.PHONY: fmt
fmt:
	@go fmt ./...

# testing

.PHONY: test
test:
	@go test -v -tags=integration ./...

.PHONY: test/coverage
test/coverage:
	@go test -coverprofile=coverage.out -tags=integration ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# building

.PHONY: build
build: generate
	@echo 'Building $(BINARY_NAME)...'
	@go build -ldflags="-s -w" -o=./bin/$(BINARY_NAME) $(MAIN_PACKAGE)

# amd64 & arm64
.PHONY: build/all
build/all: generate
	@echo 'Building for Linux/AMD64...'
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o=./bin/linux_amd64/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo 'Building for Linux/ARM64...'
	@GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o=./bin/linux_arm64/$(BINARY_NAME) $(MAIN_PACKAGE)

# if optional args are needed -> make run ARGS="-port=4000"
.PHONY: run
run: generate
	@go run $(MAIN_PACKAGE) $(ARGS)

.PHONY: clean
clean:
	@rm -rf bin
	@rm -f coverage.out coverage.html

.PHONY: generate
generate:
	@echo 'Generating templates...'
	@templ generate
	@make tailwind/build

TAILWIND_MAJOR_VERSION=4
TAILWIND_BIN=./bin/tailwindcss

# Determine OS and Arch for downloading the correct binary
OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)

# Map x86_64 to x64 for Tailwind naming convention
ifeq ($(ARCH),x86_64)
	ARCH := x64
endif
# Map arm64 (Mac/Pi) to arm64
ifeq ($(ARCH),aarch64)
	ARCH := arm64
endif

# Install the binary if it doesn't exist
.PHONY: tailwind/install
tailwind/install:
	@if [ ! -f $(TAILWIND_BIN) ]; then \
		echo "Downloading Tailwind CSS v$(TAILWIND_MAJOR_VERSION)..."; \
		mkdir -p bin; \
		LATEST=$$(curl -s https://api.github.com/repos/tailwindlabs/tailwindcss/releases | grep -o '"tag_name": "v$(TAILWIND_MAJOR_VERSION)\.[^"]*"' | head -1 | cut -d'"' -f4); \
		echo "Detected version: $$LATEST"; \
		curl -sL "https://github.com/tailwindlabs/tailwindcss/releases/download/$$LATEST/tailwindcss-$(OS)-$(ARCH)" -o $(TAILWIND_BIN); \
		echo "Tailwind installed."; \
	else \
		echo "Tailwind binary already exists."; \
	fi
	@chmod +x $(TAILWIND_BIN)

# Build CSS: depend on 'tailwind/install' to ensure binary exists
.PHONY: tailwind/build
tailwind/build: tailwind/install
	@echo "Compiling Tailwind..."
	@$(TAILWIND_BIN) -i ./static/tailwind.css -o ./static/style.css --content "internal/components/**/*.templ" --minify

# Watch Mode
.PHONY: tailwind/watch
tailwind/watch: tailwind/install
	@$(TAILWIND_BIN) -i ./static/tailwind.css -o ./static/style.css --watch

DB_URL ?= sqlite3://./blog.db
MIGRATIONS_DIR ?= ./migrations

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for $(name)...'
	@migrate create -seq -ext=.sql -dir=$(MIGRATIONS_DIR) $(name)

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up:
	@echo "running UP migrations..."
	@migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) up

## db/migrations/down: apply all down database migrations
.PHONY: db/migrations/down
db/migrations/down:
	@echo "running DOWN migrations..."
	@migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) down

.PHONY: db/migrations/version
db/migrations/version:
	@migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) version

.PHONY: db/migrations/force
db/migrations/force:
	@test -n "$(version)" || (echo "Error: version is required. Usage: make db/migrations/force version=1" && exit 1)
	@migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) force $(version)

.PHONY: db/migrations/test
db/migrations/test:
	@echo "Testing migrations..."
	@rm -f test.db
	@migrate -path $(MIGRATIONS_DIR) -database sqlite3://test.db up
	@echo "UP migrations passed"
	@migrate -path $(MIGRATIONS_DIR) -database sqlite3://test.db down
	@echo "DOWN migrations passed"
	@migrate -path $(MIGRATIONS_DIR) -database sqlite3://test.db up
	@echo "Re-applying UP migrations passed"
	@rm -f test.db
	@echo "All migration tests passed!"

.PHONY: killstale
killstale:
	@lsof -ti:3000 | xargs kill -9
