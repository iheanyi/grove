.PHONY: build run clean restart kill dev debug

APP_NAME = GroveMenubar
BUILD_DIR = .build
APP_BUNDLE = $(BUILD_DIR)/$(APP_NAME).app
BINARY = $(BUILD_DIR)/arm64-apple-macosx/debug/$(APP_NAME)

build:
	swift build
	@mkdir -p "$(APP_BUNDLE)/Contents/MacOS"
	@mkdir -p "$(APP_BUNDLE)/Contents/Resources"
	@cp "$(BINARY)" "$(APP_BUNDLE)/Contents/MacOS/$(APP_NAME)"
	@if [ ! -f "$(APP_BUNDLE)/Contents/Info.plist" ]; then \
		echo '<?xml version="1.0" encoding="UTF-8"?>' > "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<plist version="1.0"><dict>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<key>CFBundleExecutable</key><string>$(APP_NAME)</string>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<key>CFBundleIdentifier</key><string>com.iheanyi.grovemenubar</string>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<key>CFBundleName</key><string>$(APP_NAME)</string>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<key>CFBundlePackageType</key><string>APPL</string>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<key>LSUIElement</key><true/>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '</dict></plist>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
	fi
	@echo "Built $(APP_BUNDLE)"

# Build and run (full build with app bundle)
run: build
	@pkill -x "$(APP_NAME)" 2>/dev/null || true
	@sleep 0.5
	open "$(APP_BUNDLE)"

# Quick restart - just rebuild and run binary directly (faster for development)
restart:
	@pkill -x "$(APP_NAME)" 2>/dev/null || true
	@swift build
	@sleep 0.3
	@$(BINARY) &
	@echo "Restarted $(APP_NAME)"

# Kill the running app
kill:
	@pkill -x "$(APP_NAME)" 2>/dev/null && echo "Killed $(APP_NAME)" || echo "$(APP_NAME) not running"

# Development mode - build, kill old instance, and run with logs visible in terminal
dev:
	@pkill -x "$(APP_NAME)" 2>/dev/null || true
	@swift build
	@echo "Starting $(APP_NAME) with logs..."
	@$(BINARY)

# Debug mode - same as dev but with GROVE_DEBUG enabled
debug:
	@pkill -x "$(APP_NAME)" 2>/dev/null || true
	@swift build
	@echo "Starting $(APP_NAME) in debug mode..."
	@GROVE_DEBUG=1 $(BINARY)

clean:
	rm -rf $(BUILD_DIR)
