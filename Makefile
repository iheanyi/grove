.PHONY: all build build-cli build-menubar run run-menubar restart dev clean clean-cli clean-menubar kill release-menubar install-menubar

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
