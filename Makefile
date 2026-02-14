# BB-Stream Makefile
# Build and development commands

.PHONY: all build build-cli build-sidecar build-all test clean dev \
        build-mac build-mac-intel build-linux build-windows \
        build-universal desktop-dev desktop-build desktop-release \
        install lint release

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=bb-stream
CLI_OUTPUT=./$(BINARY_NAME)

# Directories
CMD_DIR=./cmd/bb-stream
DESKTOP_DIR=./desktop
BINARIES_DIR=$(DESKTOP_DIR)/src-tauri/binaries

# Build flags
LDFLAGS=-ldflags="-s -w"
CGO_OFF=CGO_ENABLED=0

# Default target
all: test build

# Build CLI for current platform
build: build-cli

build-cli:
	$(GOBUILD) $(LDFLAGS) -o $(CLI_OUTPUT) $(CMD_DIR)

# Build sidecar for all platforms
build-sidecar:
	./scripts/build-sidecar.sh

build-all: build-cli build-sidecar

# Platform-specific builds
build-mac: build-mac-arm build-mac-intel

build-mac-arm:
	GOOS=darwin GOARCH=arm64 $(CGO_OFF) $(GOBUILD) $(LDFLAGS) \
		-o $(BINARIES_DIR)/$(BINARY_NAME)-aarch64-apple-darwin $(CMD_DIR)

build-mac-intel:
	GOOS=darwin GOARCH=amd64 $(CGO_OFF) $(GOBUILD) $(LDFLAGS) \
		-o $(BINARIES_DIR)/$(BINARY_NAME)-x86_64-apple-darwin $(CMD_DIR)

build-linux:
	GOOS=linux GOARCH=amd64 $(CGO_OFF) $(GOBUILD) $(LDFLAGS) \
		-o $(BINARIES_DIR)/$(BINARY_NAME)-x86_64-unknown-linux-gnu $(CMD_DIR)

build-windows:
	GOOS=windows GOARCH=amd64 $(CGO_OFF) $(GOBUILD) $(LDFLAGS) \
		-o $(BINARIES_DIR)/$(BINARY_NAME)-x86_64-pc-windows-msvc.exe $(CMD_DIR)

# Run tests
test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -cover ./...

test-coverage-html:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(CLI_OUTPUT)
	rm -f $(BINARIES_DIR)/$(BINARY_NAME)-*
	rm -f coverage.out coverage.html

# Development mode
dev:
	$(GOBUILD) -o $(CLI_OUTPUT) $(CMD_DIR)
	./$(BINARY_NAME) serve

# Desktop app commands
desktop-dev:
	cd $(DESKTOP_DIR) && npm run tauri dev

desktop-build: build-sidecar
	cd $(DESKTOP_DIR) && npm run tauri build

desktop-build-current:
	./scripts/build-sidecar.sh --current
	cd $(DESKTOP_DIR) && npm run tauri build

desktop-check:
	cd $(DESKTOP_DIR) && npm run check

# Release builds
build-universal:
	./scripts/build-sidecar.sh --universal

release: test build-universal
	./scripts/release.sh

# Install dependencies
install:
	$(GOMOD) download
	cd $(DESKTOP_DIR) && npm install

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GOCMD) fmt ./...

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Help
help:
	@echo "BB-Stream Makefile Commands:"
	@echo ""
	@echo "  make              - Run tests and build CLI"
	@echo "  make build        - Build CLI for current platform"
	@echo "  make build-sidecar- Build sidecar for all platforms"
	@echo "  make build-all    - Build CLI and all sidecars"
	@echo ""
	@echo "Platform-specific builds:"
	@echo "  make build-mac    - Build for macOS (ARM + Intel)"
	@echo "  make build-mac-arm- Build for macOS ARM (Apple Silicon)"
	@echo "  make build-mac-intel - Build for macOS Intel"
	@echo "  make build-linux  - Build for Linux x64"
	@echo "  make build-windows- Build for Windows x64"
	@echo "  make build-universal - Build with macOS universal binary"
	@echo ""
	@echo "Testing:"
	@echo "  make test         - Run all tests"
	@echo "  make test-coverage- Run tests with coverage"
	@echo "  make test-coverage-html - Generate HTML coverage report"
	@echo ""
	@echo "Desktop app:"
	@echo "  make desktop-dev  - Run desktop app in dev mode"
	@echo "  make desktop-build- Build desktop app for all platforms"
	@echo "  make desktop-build-current - Build for current platform only"
	@echo "  make desktop-check- Run Svelte type checking"
	@echo ""
	@echo "Release:"
	@echo "  make release      - Full release build (tests + all platforms)"
	@echo ""
	@echo "Development:"
	@echo "  make dev          - Build and run API server"
	@echo "  make install      - Install Go and npm dependencies"
	@echo "  make lint         - Run linter"
	@echo "  make fmt          - Format code"
	@echo "  make tidy         - Tidy Go modules"
	@echo "  make clean        - Remove build artifacts"
