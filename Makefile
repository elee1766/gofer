# Makefile for gofer

.PHONY: test test-unit test-integration test-coverage test-race lint build clean help

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint
BINARY_NAME=gofer
MAIN_PATH=./cmd/gofer

# Test parameters
TEST_TIMEOUT=30s
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

## Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PATH)

## Run all tests
test: test-unit test-integration

## Run unit tests
test-unit:
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./src/...

## Run integration tests  
test-integration:
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -tags=integration ./tests/integration/...

## Run tests with coverage
test-coverage:
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

## Run tests with race detection
test-race:
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -race ./...

## Run linter
lint:
	$(GOLINT) run

## Run linter with fixes
lint-fix:
	$(GOLINT) run --fix

## Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

## Tidy go.mod
tidy:
	$(GOMOD) tidy

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)

## Install pre-commit hooks
install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

## Run all checks (lint, test, build)
check: lint test build

## Build release binaries for all platforms
release-build:
	./scripts/release/build-release.sh

## Generate changelog
changelog:
	./scripts/release/generate-changelog.sh

## Create a new release (interactive)
release:
	./scripts/release/release.sh

## Install locally
install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

## Uninstall
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

## Show help
help:
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)