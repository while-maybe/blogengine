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
	@go test -race ./...

.PHONY: fmt
fmt:
	@go fmt ./...

# testing

.PHONY: test
test:
	@go test ./... -v

.PHONY: test/coverage
test/coverage:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# building

.PHONY: build
build:
	@echo 'Generating templates...'
	@templ generate
	@echo 'Building $(BINARY_NAME)...'
	@go build -ldflags="-s -w" -o=./bin/$(BINARY_NAME) $(MAIN_PACKAGE)

# amd64 & arm64
.PHONY: build/all
build/all:
	@echo 'Generating templates...'
	@templ generate
	@echo 'Building for Linux/AMD64...'
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o=./bin/linux_amd64/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo 'Building for Linux/ARM64...'
	@GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o=./bin/linux_arm64/$(BINARY_NAME) $(MAIN_PACKAGE)

# if optional args are needed -> make run ARGS="-port=4000"
.PHONY: run
run:
	@templ generate
	@make tailwind/build
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
		chmod +x $(TAILWIND_BIN); \
		echo "Tailwind installed."; \
	else \
		echo "Tailwind binary already exists."; \
	fi

# Build CSS: depend on 'tailwind/install' to ensure binary exists
.PHONY: tailwind/build
tailwind/build: tailwind/install
	@echo "Compiling Tailwind..."
	@$(TAILWIND_BIN) -i ./static/tailwind.css -o ./static/style.css --minify

# Watch Mode
.PHONY: tailwind/watch
tailwind/watch: tailwind/install
	@$(TAILWIND_BIN) -i ./static/tailwind.css -o ./static/style.css --watch
