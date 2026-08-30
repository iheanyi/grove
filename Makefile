.PHONY: all help build build-cli build-web build-go install install-cli install-local clean clean-cli clean-web clean-menubar test test-coverage lint fmt run run-menubar restart dev deps dev-start dev-ls dev-doctor dev-dashboard dev-web release release-patch release-dry release-local release-menubar install-menubar kill

# Web dashboard directory
WEB_DIR=internal/dashboard/web

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
GOMOD=$(GO) mod
GOFMT=$(GO) fmt

# Default target: build CLI and menubar app
all: build build-menubar

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build-web: ## Build the web dashboard
	cd $(WEB_DIR) && npm install && npm run build

build: build-web ## Build the grove binary with the web dashboard
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/grove

build-cli: build ## Build the Go CLI

build-go: ## Build only the Go binary with the tracked dashboard stub
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/grove

# Build Swift menubar app
build-menubar:
	@echo "Building Swift menubar app..."
	cd menubar/GroveMenubar && swift build
	@echo "Menubar app built"

install: build ## Install grove to $$GOPATH/bin
	cp $(BINARY_NAME) $(GOPATH)/bin/

install-cli: build ## Install CLI to /usr/local/bin
	@echo "Installing CLI to /usr/local/bin..."
	cp $(BINARY_NAME) /usr/local/bin/grove
	@echo "Installed: /usr/local/bin/grove"

install-local: build ## Install grove to /usr/local/bin
	sudo cp $(BINARY_NAME) /usr/local/bin/

clean: clean-cli clean-web clean-menubar ## Remove build artifacts

clean-cli:
	@echo "Cleaning CLI build..."
	rm -f $(BINARY_NAME)
	rm -rf dist/

clean-web: ## Remove generated web assets and restore the tracked stub
	rm -rf $(WEB_DIR)/node_modules $(WEB_DIR)/.svelte-kit $(WEB_DIR)/build
	mkdir -p $(WEB_DIR)/build
	git restore -- $(WEB_DIR)/build/index.html

clean-menubar:
	@echo "Cleaning menubar build..."
	rm -rf menubar/GroveMenubar/.build

test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	$(GOFMT) ./...

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

dev-dashboard: build-go ## Run dashboard with web dev server
	./$(BINARY_NAME) dashboard --dev

dev-web: ## Run web dev server for frontend development
	cd $(WEB_DIR) && npm run dev

# Run the menubar app (builds first if needed)
run: run-menubar

run-menubar: build-menubar kill
	@echo "Starting GroveMenubar..."
	menubar/GroveMenubar/.build/arm64-apple-macosx/debug/GroveMenubar &

# Quick restart - kill, rebuild, and restart menubar app
restart: kill build-menubar
	@sleep 0.3
	@menubar/GroveMenubar/.build/arm64-apple-macosx/debug/GroveMenubar &
	@echo "Restarted GroveMenubar"

# Development mode - run menubar with logs visible in terminal (blocks)
dev: kill build-menubar
	@echo "Starting GroveMenubar with logs..."
	menubar/GroveMenubar/.build/arm64-apple-macosx/debug/GroveMenubar

# Kill any running instance of the menubar app
kill:
	@pkill -x GroveMenubar 2>/dev/null || true

# Install menubar app to /Applications
install-menubar:
	@echo "Building menubar for release..."
	cd menubar/GroveMenubar && swift build -c release
	@echo "Creating Grove.app bundle..."
	@mkdir -p menubar/GroveMenubar/dist/Grove.app/Contents/MacOS
	@mkdir -p menubar/GroveMenubar/dist/Grove.app/Contents/Resources
	@cp menubar/GroveMenubar/.build/release/GroveMenubar menubar/GroveMenubar/dist/Grove.app/Contents/MacOS/Grove 2>/dev/null || \
		cp menubar/GroveMenubar/.build/arm64-apple-macosx/release/GroveMenubar menubar/GroveMenubar/dist/Grove.app/Contents/MacOS/Grove
	@echo '<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>CFBundleExecutable</key><string>Grove</string><key>CFBundleIdentifier</key><string>com.iheanyi.grove</string><key>CFBundleName</key><string>Grove</string><key>CFBundlePackageType</key><string>APPL</string><key>CFBundleShortVersionString</key><string>1.0</string><key>LSMinimumSystemVersion</key><string>14.0</string><key>LSUIElement</key><true/></dict></plist>' > menubar/GroveMenubar/dist/Grove.app/Contents/Info.plist
	@echo "Installing Grove.app to /Applications..."
	@rm -rf /Applications/Grove.app
	@cp -R menubar/GroveMenubar/dist/Grove.app /Applications/
	@echo "Installed: /Applications/Grove.app"
	@echo ""
	@echo "Note: First launch, right-click the app and select 'Open' to bypass Gatekeeper."

# Release menubar app via GitHub Actions
release-menubar:
	gh workflow run release-menubar.yml --repo iheanyi/grove
	@echo "Menubar release triggered! Watch with: gh run watch"

# Release targets (requires goreleaser)
release-dry: ## Dry run release locally
	goreleaser release --snapshot --clean

release-local: ## Create a release locally
	goreleaser release --clean

# GitHub release targets
release: ## Tag and push a release, usage: make release V=0.10.3
	@if [ -z "$(V)" ]; then echo "Usage: make release V=0.10.3"; exit 1; fi
	@case "$(V)" in v*) echo "Use V without a leading v, e.g. V=0.10.3"; exit 1;; esac
	@if ! printf '%s\n' "$(V)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$'; then echo "V must look like semver, e.g. 0.10.3"; exit 1; fi
	@if [ "$$(git branch --show-current)" != "main" ]; then echo "Release must be cut from main"; exit 1; fi
	@if ! git diff --quiet || ! git diff --cached --quiet; then echo "Working tree must be clean"; exit 1; fi
	git fetch origin
	@if [ "$$(git rev-parse HEAD)" != "$$(git rev-parse origin/main)" ]; then echo "main is behind/ahead of origin/main"; exit 1; fi
	@if git rev-parse -q --verify "refs/tags/v$(V)" >/dev/null; then echo "Tag v$(V) already exists locally"; exit 1; fi
	@if git ls-remote --exit-code --tags origin "refs/tags/v$(V)" >/dev/null 2>&1; then echo "Tag v$(V) already exists on origin"; exit 1; fi
	git tag -a "v$(V)" -m "Release v$(V)"
	git push origin "v$(V)"
	@echo "Release v$(V) pushed. GitHub Actions will build the GitHub Release and update Homebrew."

release-patch: ## Bump the latest v* patch tag and release it
	@latest=$$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | awk 'NR==1 { print; exit }'); \
	if [ -z "$$latest" ]; then echo "No v* semver tags found"; exit 1; fi; \
	base=$${latest#v}; base=$${base%%-*}; \
	major=$$(printf '%s\n' "$$base" | cut -d. -f1); \
	minor=$$(printf '%s\n' "$$base" | cut -d. -f2); \
	patch=$$(printf '%s\n' "$$base" | cut -d. -f3); \
	$(MAKE) release V=$$major.$$minor.$$((patch + 1))
