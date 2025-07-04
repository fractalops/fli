# Variables
BINARY_NAME=fli
MAIN_PATH=./cmd/fli
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -s -w"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_UNIX=$(BINARY_NAME)_unix

# Default target
.DEFAULT_GOAL := build

# Build the application
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

# Build for current platform
build-local:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

# Build for multiple platforms
build-all: clean
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out

# Run tests
test:
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with coverage and show in browser
test-coverage-html: test-coverage
	open coverage.html

# Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Run linting
lint:
	golangci-lint run

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Run security audit
audit:
	govulncheck ./...

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Generate documentation
docs:
	godoc -http=:6060

# Install the binary (user-local installation)
install:
	$(GOBUILD) $(LDFLAGS) -o $(HOME)/.local/bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Installed $(BINARY_NAME) to $(HOME)/.local/bin/"
	@echo "Make sure $(HOME)/.local/bin/ is in your PATH"

# Install the binary (system-wide installation - requires sudo)
install-system:
	$(GOBUILD) $(LDFLAGS) -o /usr/local/bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

# Create release directory with binaries
release: build-all
	mkdir -p release
	mv $(BINARY_NAME)-* release/
	cd release && sha256sum $(BINARY_NAME)-* > checksums.txt

# Test release process locally (dry run)
release-dry:
	goreleaser release --snapshot --clean --skip-publish

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  build-local    - Build for current platform"
	@echo "  build-all      - Build for all platforms"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  bench          - Run benchmarks"
	@echo "  lint           - Run linting"
	@echo "  deps           - Install dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  audit          - Run security audit"
	@echo "  fmt            - Format code"
	@echo "  docs           - Generate documentation"
	@echo "  install        - Install the binary (user-local)"
	@echo "  install-system - Install the binary (system-wide, requires sudo)"
	@echo "  release        - Create release binaries"
	@echo "  release-dry    - Test release process locally"
	@echo "  help           - Show this help"

.PHONY: build build-local build-all clean test test-coverage test-coverage-html bench lint deps deps-update audit fmt docs install install-system release docker-build docker-run help 