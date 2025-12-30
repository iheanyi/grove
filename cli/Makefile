.PHONY: build install clean test lint run help

# Build variables
BINARY_NAME=grove
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/iheanyi/grove/internal/cli.Version=$(VERSION) -X github.com/iheanyi/grove/internal/cli.Commit=$(COMMIT) -X github.com/iheanyi/grove/internal/cli.Date=$(DATE)"

# Go commands
GO=go
GOBUILD=$(GO) build
GOTEST=$(GO) test
GOGET=$(GO) get
GOMOD=$(GO) mod
GOFMT=$(GO) fmt

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the grove binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/grove

install: build ## Install grove to $GOPATH/bin
	cp $(BINARY_NAME) $(GOPATH)/bin/

install-local: build ## Install grove to /usr/local/bin
	sudo cp $(BINARY_NAME) /usr/local/bin/

clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
	rm -rf dist/

test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	$(GOFMT) ./...

run: build ## Build and run
	./$(BINARY_NAME)

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

# Development helpers
dev-start: build ## Start a test server
	./$(BINARY_NAME) start --foreground python3 -m http.server

dev-ls: build ## List servers
	./$(BINARY_NAME) ls

dev-doctor: build ## Run doctor
	./$(BINARY_NAME) doctor

# Release targets (requires goreleaser)
release-dry: ## Dry run release locally
	goreleaser release --snapshot --clean

release-local: ## Create a release locally (not recommended)
	goreleaser release --clean

# GitHub release targets (recommended)
release: ## Release via GitHub Actions (auto-increment patch)
	gh workflow run release.yml --repo iheanyi/grove
	@echo "Release triggered! Watch with: gh run watch"

release-version: ## Release specific version (usage: make release-version V=0.3.0)
	@if [ -z "$(V)" ]; then echo "Usage: make release-version V=0.3.0"; exit 1; fi
	gh workflow run release.yml --repo iheanyi/grove -f version=$(V)
	@echo "Release $(V) triggered! Watch with: gh run watch"

release-prerelease: ## Release pre-release (usage: make release-prerelease V=0.3.0-beta.1)
	@if [ -z "$(V)" ]; then echo "Usage: make release-prerelease V=0.3.0-beta.1"; exit 1; fi
	gh workflow run release.yml --repo iheanyi/grove -f version=$(V) -f prerelease=true
	@echo "Pre-release $(V) triggered! Watch with: gh run watch"
