# ROSA HCP CLI Makefile

# Build variables
BINARY_NAME := rosa
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Go build flags
LDFLAGS := -ldflags "\
	-s -w \
	-X github.com/openshift/rosa-hcp/internal/version.Version=$(VERSION) \
	-X github.com/openshift/rosa-hcp/internal/version.BuildTime=$(BUILD_TIME) \
	-X github.com/openshift/rosa-hcp/internal/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/openshift/rosa-hcp/internal/version.GoVersion=$(GO_VERSION)"

# Directories
BIN_DIR := bin
DIST_DIR := dist
COVERAGE_DIR := coverage

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet

# Test flags
TEST_FLAGS := -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic

.DEFAULT_GOAL := build

## help: Display this help message
.PHONY: help
help:
	@echo "ROSA HCP CLI - Makefile Commands"
	@echo ""
	@echo "Usage: make <command>"
	@echo ""
	@echo "Commands:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## build: Build the binary for current platform
.PHONY: build
build: clean
	@echo "Building $(BINARY_NAME) $(VERSION) for current platform..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/rosa

## build-all: Build binaries for all supported platforms
.PHONY: build-all
build-all: clean
	@echo "Building $(BINARY_NAME) $(VERSION) for all platforms..."
	@mkdir -p $(DIST_DIR)
	
	@echo "Building for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/rosa
	
	@echo "Building for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/rosa
	
	@echo "Building for Darwin AMD64..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/rosa
	
	@echo "Building for Darwin ARM64..."
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/rosa
	
	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/rosa
	
	@echo "Build complete! Binaries are in $(DIST_DIR)/"

## install: Install the binary to $GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installation complete!"

## run: Run the application
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BIN_DIR)/$(BINARY_NAME)

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	@echo "Clean complete!"

## test: Run all tests
.PHONY: test
test:
	@echo "Running tests..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(TEST_FLAGS) ./...
	@echo "Tests complete!"

## test-unit: Run unit tests only
.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(TEST_FLAGS) ./pkg/... ./internal/...
	@echo "Unit tests complete!"

## test-integration: Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(TEST_FLAGS) ./tests/integration/...
	@echo "Integration tests complete!"

## test-e2e: Run end-to-end tests
.PHONY: test-e2e
test-e2e:
	@echo "Running e2e tests..."
	$(GOTEST) -v ./tests/e2e/...
	@echo "E2E tests complete!"

## coverage: Generate test coverage report
.PHONY: coverage
coverage: test
	@echo "Generating coverage report..."
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

## benchmark: Run benchmarks
.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## lint: Run linters
.PHONY: lint
lint:
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: make install-tools"; exit 1; }
	golangci-lint run --fix ./...
	@echo "Linting complete!"

## fmt: Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@echo "Formatting complete!"

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete!"

## mod-tidy: Tidy go modules
.PHONY: mod-tidy
mod-tidy:
	@echo "Tidying go modules..."
	$(GOMOD) tidy
	@echo "Module tidy complete!"

## mod-verify: Verify go modules
.PHONY: mod-verify
mod-verify:
	@echo "Verifying go modules..."
	$(GOMOD) verify
	@echo "Module verification complete!"

## mod-download: Download go modules
.PHONY: mod-download
mod-download:
	@echo "Downloading go modules..."
	$(GOMOD) download
	@echo "Module download complete!"

## generate: Generate code
.PHONY: generate
generate:
	@echo "Generating code..."
	@command -v mockgen >/dev/null 2>&1 || { echo "mockgen not installed. Run: make install-tools"; exit 1; }
	go generate ./...
	@echo "Code generation complete!"

## install-tools: Install development tools
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install go.uber.org/mock/mockgen@latest
	go install github.com/goreleaser/goreleaser@latest
	@echo "Tools installation complete!"

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t rosa-hcp:$(VERSION) -t rosa-hcp:latest .
	@echo "Docker build complete!"

## docker-run: Run Docker container
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it rosa-hcp:$(VERSION)

## release: Create a new release
.PHONY: release
release:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Run: make install-tools"; exit 1; }
	@echo "Creating release $(VERSION)..."
	goreleaser release --clean
	@echo "Release complete!"

## release-snapshot: Create a snapshot release (doesn't publish)
.PHONY: release-snapshot
release-snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Run: make install-tools"; exit 1; }
	@echo "Creating snapshot release..."
	goreleaser release --snapshot --clean
	@echo "Snapshot release complete!"

## check: Run all checks (fmt, vet, lint, test)
.PHONY: check
check: fmt vet lint test
	@echo "All checks passed!"

## ci: Run CI pipeline locally
.PHONY: ci
ci: clean mod-verify check build-all
	@echo "CI pipeline complete!"

## version: Display version information
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"

.PHONY: all
all: clean build test
