.PHONY: all build build-cli build-menubar run run-menubar clean clean-cli clean-menubar kill

# Default target: build both apps
all: build

# Build both CLI and menubar app
build: build-cli build-menubar

# Build Go CLI
build-cli:
	@echo "Building Go CLI..."
	cd cli && go build -o grove ./cmd/grove
	@echo "CLI built: cli/grove"

# Build Swift menubar app
build-menubar:
	@echo "Building Swift menubar app..."
	cd menubar/GroveMenubar && swift build
	@echo "Menubar app built"

# Run the menubar app (builds first if needed)
run: run-menubar

run-menubar: build-menubar kill
	@echo "Starting GroveMenubar..."
	menubar/GroveMenubar/.build/arm64-apple-macosx/debug/GroveMenubar &

# Kill any running instance of the menubar app
kill:
	@pkill -x GroveMenubar 2>/dev/null || true

# Clean all build artifacts
clean: clean-cli clean-menubar

clean-cli:
	@echo "Cleaning CLI build..."
	rm -f cli/grove

clean-menubar:
	@echo "Cleaning menubar build..."
	rm -rf menubar/GroveMenubar/.build

# Install CLI to /usr/local/bin
install-cli: build-cli
	@echo "Installing CLI to /usr/local/bin..."
	cp cli/grove /usr/local/bin/grove
	@echo "Installed: /usr/local/bin/grove"
