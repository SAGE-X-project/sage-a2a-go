# Makefile for sage-a2a-go
# Go project build automation

# ==================================================================================== #
# VARIABLES
# ==================================================================================== #

# Project metadata
PROJECT_NAME := sage-a2a-go
MODULE := github.com/sage-x-project/sage-a2a-go
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go commands
GO := go
GOFMT := gofmt
GOVET := $(GO) vet
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOMOD := $(GO) mod
GOINSTALL := $(GO) install

# Directories
PKG_DIR := ./pkg/...
CMD_DIR := ./cmd/...
TEST_DIR := ./test/...
EXAMPLES_DIR := ./cmd/examples

# Build output
BUILD_DIR := ./build
COVERAGE_DIR := ./coverage

# Test flags
TEST_FLAGS := -race -timeout=5m
COVERAGE_FLAGS := -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic
BENCH_FLAGS := -bench=. -benchmem

# Tools
GOLANGCI_LINT := golangci-lint
GOLANGCI_LINT_VERSION := v1.61.0

# Colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

.PHONY: help
help: ## Display this help message
	@echo "$(CYAN)$(PROJECT_NAME) - Build System$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""

# ==================================================================================== #
##@ Development
# ==================================================================================== #

.PHONY: build
build: ## Build the library
	@echo "$(GREEN)Building library...$(NC)"
	@$(GOBUILD) -v ./...
	@echo "$(GREEN)✓ Build complete$(NC)"

.PHONY: build-examples
build-examples: ## Build example programs
	@echo "$(GREEN)Building examples...$(NC)"
	@mkdir -p $(BUILD_DIR)/examples
	@for dir in $(EXAMPLES_DIR)/*; do \
		if [ -d "$$dir" ]; then \
			name=$$(basename $$dir); \
			echo "  Building $$name..."; \
			$(GOBUILD) -o $(BUILD_DIR)/examples/$$name $$dir/main.go; \
		fi \
	done
	@echo "$(GREEN)✓ Examples built in $(BUILD_DIR)/examples/$(NC)"

.PHONY: install
install: ## Install the library locally
	@echo "$(GREEN)Installing library...$(NC)"
	@$(GO) install ./...
	@echo "$(GREEN)✓ Installation complete$(NC)"

# ==================================================================================== #
##@ Testing
# ==================================================================================== #

.PHONY: test
test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	@$(GOTEST) $(TEST_FLAGS) $(PKG_DIR)
	@echo "$(GREEN)✓ Tests passed$(NC)"

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@echo "$(GREEN)Running tests (verbose)...$(NC)"
	@$(GOTEST) $(TEST_FLAGS) -v $(PKG_DIR)

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) $(TEST_FLAGS) $(COVERAGE_FLAGS) $(PKG_DIR)
	@$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)✓ Coverage report generated: $(COVERAGE_DIR)/coverage.html$(NC)"
	@$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print "Total coverage: " $$3}'

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	@$(GOTEST) $(TEST_FLAGS) -v $(TEST_DIR)
	@echo "$(GREEN)✓ Integration tests passed$(NC)"

.PHONY: test-all
test-all: test test-integration ## Run all tests (unit + integration)

.PHONY: bench
bench: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(NC)"
	@$(GOTEST) $(BENCH_FLAGS) $(PKG_DIR)

.PHONY: bench-compare
bench-compare: ## Run benchmarks and save results
	@echo "$(GREEN)Running benchmarks (saving results)...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) $(BENCH_FLAGS) $(PKG_DIR) | tee $(COVERAGE_DIR)/bench.txt

# ==================================================================================== #
##@ Code Quality
# ==================================================================================== #

.PHONY: fmt
fmt: ## Format code with gofmt
	@echo "$(GREEN)Formatting code...$(NC)"
	@$(GOFMT) -s -w .
	@echo "$(GREEN)✓ Code formatted$(NC)"

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@echo "$(GREEN)Checking code formatting...$(NC)"
	@files=$$($(GOFMT) -l . 2>&1); \
	if [ -n "$$files" ]; then \
		echo "$(RED)✗ The following files need formatting:$(NC)"; \
		echo "$$files"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ Code is properly formatted$(NC)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	@$(GOVET) ./...
	@echo "$(GREEN)✓ Vet passed$(NC)"

.PHONY: lint
lint: install-golangci-lint ## Run golangci-lint
	@echo "$(GREEN)Running linter...$(NC)"
	@$(GOLANGCI_LINT) run --timeout=5m
	@echo "$(GREEN)✓ Lint passed$(NC)"

.PHONY: lint-fix
lint-fix: install-golangci-lint ## Run golangci-lint with auto-fix
	@echo "$(GREEN)Running linter with auto-fix...$(NC)"
	@$(GOLANGCI_LINT) run --fix --timeout=5m
	@echo "$(GREEN)✓ Lint fixes applied$(NC)"

# ==================================================================================== #
##@ Dependencies
# ==================================================================================== #

.PHONY: deps
deps: ## Download dependencies
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	@$(GO) mod download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	@echo "$(GREEN)Tidying dependencies...$(NC)"
	@$(GOMOD) tidy
	@echo "$(GREEN)✓ Dependencies tidied$(NC)"

.PHONY: verify
verify: ## Verify dependencies
	@echo "$(GREEN)Verifying dependencies...$(NC)"
	@$(GOMOD) verify
	@echo "$(GREEN)✓ Dependencies verified$(NC)"

.PHONY: deps-update
deps-update: ## Update all dependencies
	@echo "$(GREEN)Updating dependencies...$(NC)"
	@$(GO) get -u ./...
	@$(GOMOD) tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

# ==================================================================================== #
##@ Tools
# ==================================================================================== #

.PHONY: install-tools
install-tools: install-golangci-lint ## Install development tools
	@echo "$(GREEN)✓ All tools installed$(NC)"

.PHONY: install-golangci-lint
install-golangci-lint: ## Install golangci-lint
	@if ! command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		echo "$(GREEN)Installing golangci-lint $(GOLANGCI_LINT_VERSION)...$(NC)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	else \
		echo "$(GREEN)golangci-lint already installed$(NC)"; \
	fi

# ==================================================================================== #
##@ Cleanup
# ==================================================================================== #

.PHONY: clean
clean: ## Clean build artifacts and test cache
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -f coverage.out coverage_signer.out
	@$(GO) clean -cache -testcache -modcache
	@echo "$(GREEN)✓ Cleanup complete$(NC)"

.PHONY: clean-build
clean-build: ## Clean only build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@echo "$(GREEN)✓ Build artifacts cleaned$(NC)"

.PHONY: clean-coverage
clean-coverage: ## Clean coverage reports
	@echo "$(GREEN)Cleaning coverage reports...$(NC)"
	@rm -rf $(COVERAGE_DIR)
	@rm -f coverage.out coverage_signer.out
	@echo "$(GREEN)✓ Coverage reports cleaned$(NC)"

# ==================================================================================== #
##@ CI/CD
# ==================================================================================== #

.PHONY: ci
ci: fmt-check vet lint test ## Run CI checks (format, vet, lint, test)

.PHONY: ci-full
ci-full: fmt-check vet lint test-all test-coverage ## Run full CI suite

.PHONY: pre-commit
pre-commit: fmt vet lint test ## Run pre-commit checks

# ==================================================================================== #
##@ Information
# ==================================================================================== #

.PHONY: info
info: ## Display project information
	@echo "$(CYAN)Project Information:$(NC)"
	@echo "  Name:        $(PROJECT_NAME)"
	@echo "  Module:      $(MODULE)"
	@echo "  Version:     $(VERSION)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  Go Version:  $$($(GO) version)"
	@echo ""

.PHONY: version
version: ## Display version information
	@echo "$(VERSION)"

# ==================================================================================== #
##@ Shortcuts
# ==================================================================================== #

.PHONY: all
all: clean build test ## Clean, build, and test

.PHONY: dev
dev: fmt vet test ## Quick development cycle (fmt, vet, test)

.PHONY: check
check: fmt-check vet lint test ## Check code quality

# Default target
.DEFAULT_GOAL := help
