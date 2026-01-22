BINARY_NAME=blogengine
MAIN_PACKAGE=./cmd/blogengine

# export GOEXPERIMENT := jsonv2

# load .env if exists
# ifneq (,$(wildcard .env))
#     include .env
#     export
# endif

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
	@go run $(MAIN_PACKAGE) $(ARGS)

.PHONY: clean
clean:
	@rm -rf bin
	@rm -f coverage.out coverage.html

.PHONY: generate
generate:
	@echo 'Generating templates...'
	@templ generate