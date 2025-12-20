.PHONY: build run clean

APP_NAME = WTMenubar
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
		echo '<key>CFBundleIdentifier</key><string>com.iheanyi.wtmenubar</string>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<key>CFBundleName</key><string>$(APP_NAME)</string>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<key>CFBundlePackageType</key><string>APPL</string>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<key>LSUIElement</key><true/>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '</dict></plist>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
	fi
	@echo "Built $(APP_BUNDLE)"

run: build
	@pkill -f "$(APP_NAME)" 2>/dev/null || true
	@sleep 0.5
	open "$(APP_BUNDLE)"

clean:
	rm -rf $(BUILD_DIR)
