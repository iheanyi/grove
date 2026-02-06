import AppKit
import Carbon.HIToolbox

/// Service for managing a global keyboard shortcut to toggle the Grove menubar panel.
///
/// Uses NSEvent global monitor to detect Ctrl+G (configurable).
/// When triggered, activates the app and clicks the status bar button to toggle the panel.
class HotkeyService {
    static let shared = HotkeyService()

    private var globalMonitor: Any?
    private var localMonitor: Any?

    // UserDefaults keys
    private static let enabledKey = "globalHotkeyEnabled"

    var isEnabled: Bool {
        get { UserDefaults.standard.object(forKey: Self.enabledKey) as? Bool ?? true }
        set {
            UserDefaults.standard.set(newValue, forKey: Self.enabledKey)
            if newValue {
                start()
            } else {
                stop()
            }
        }
    }

    private init() {}

    /// Begin listening for the global hotkey.
    func start() {
        guard isEnabled else { return }
        guard globalMonitor == nil else { return }

        // Global monitor catches key events when the app is NOT focused
        globalMonitor = NSEvent.addGlobalMonitorForEvents(matching: .keyDown) { [weak self] event in
            self?.handleKeyEvent(event)
        }

        // Local monitor catches key events when the app IS focused
        localMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            if self?.handleKeyEvent(event) == true {
                return nil // Consume the event
            }
            return event
        }
    }

    /// Stop listening for the global hotkey.
    func stop() {
        if let monitor = globalMonitor {
            NSEvent.removeMonitor(monitor)
            globalMonitor = nil
        }
        if let monitor = localMonitor {
            NSEvent.removeMonitor(monitor)
            localMonitor = nil
        }
    }

    /// Returns true if the event matched the hotkey.
    @discardableResult
    private func handleKeyEvent(_ event: NSEvent) -> Bool {
        // Default hotkey: Ctrl+G
        guard event.modifierFlags.contains(.control),
              !event.modifierFlags.contains(.command),
              !event.modifierFlags.contains(.option),
              event.keyCode == UInt16(kVK_ANSI_G) else {
            return false
        }

        DispatchQueue.main.async {
            self.toggleMenubarPanel()
        }
        return true
    }

    /// Toggle the Grove menubar panel open/closed.
    ///
    /// For MenuBarExtra with `.window` style, we activate the app and
    /// programmatically click the status item button.
    private func toggleMenubarPanel() {
        // Activate the app (brings it to front)
        NSApp.activate(ignoringOtherApps: true)

        // Find the Grove status item button and click it
        // MenuBarExtra creates an NSStatusItem automatically.
        // We can find it by looking through the status bar's status items.
        if let button = findGroveStatusButton() {
            button.performClick(nil)
        }
    }

    /// Find the Grove menubar status item button.
    private func findGroveStatusButton() -> NSStatusBarButton? {
        // Iterate through all windows to find the status bar button
        for window in NSApp.windows {
            // Status item windows have a special class
            let windowClassName = String(describing: type(of: window))
            if windowClassName.contains("NSStatusBarWindow") {
                // The button is a subview of the window's content view
                if let button = window.contentView?.subviews.compactMap({ $0 as? NSStatusBarButton }).first {
                    return button
                }
            }
        }
        return nil
    }

    /// Human-readable description of the current hotkey.
    var hotkeyDescription: String {
        return "\u{2303}G" // ⌃G
    }
}
