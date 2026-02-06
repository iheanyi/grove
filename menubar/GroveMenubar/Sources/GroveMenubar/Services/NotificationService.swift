import Foundation
import AppKit
import UserNotifications

/// Service for managing macOS notifications with actionable buttons and sound effects.
class NotificationService: NSObject, UNUserNotificationCenterDelegate {
    static let shared = NotificationService()

    private var center: UNUserNotificationCenter?
    private let preferences = PreferencesManager.shared
    private var isAvailable = false

    // MARK: - Notification Action Identifiers

    private enum ActionID {
        static let restartServer = "RESTART_SERVER"
        static let openLogs = "OPEN_LOGS"
        static let keepRunning = "KEEP_RUNNING"
    }

    private enum CategoryID {
        static let serverCrash = "SERVER_CRASH"
        static let serverIdle = "SERVER_IDLE"
        static let serverHealthy = "SERVER_HEALTHY"
    }

    // Notification types
    enum NotificationType: String, CaseIterable {
        case serverCrashed = "server_crashed"
        case serverHealthy = "server_healthy"
        case serverIdleTimeout = "server_idle_timeout"

        var identifier: String { rawValue }

        var title: String {
            switch self {
            case .serverCrashed:
                return "Server Crashed"
            case .serverHealthy:
                return "Server Healthy"
            case .serverIdleTimeout:
                return "Server Auto-Stopped (Idle)"
            }
        }

        var categoryIdentifier: String {
            switch self {
            case .serverCrashed: return CategoryID.serverCrash
            case .serverHealthy: return CategoryID.serverHealthy
            case .serverIdleTimeout: return CategoryID.serverIdle
            }
        }

        var preferencesKey: String {
            "notification_\(rawValue)_enabled"
        }
    }

    // MARK: - Initialization

    private override init() {
        super.init()
        setupNotificationCenter()
    }

    private func setupNotificationCenter() {
        // Only initialize notifications if we have a proper bundle
        guard Bundle.main.bundleIdentifier != nil else {
            print("NotificationService: Running without bundle, notifications disabled")
            return
        }

        center = UNUserNotificationCenter.current()
        center?.delegate = self
        isAvailable = true
        registerCategories()
        requestAuthorization()
    }

    // MARK: - Notification Categories with Actions

    private func registerCategories() {
        // Crash category: Restart + View Logs
        let restartAction = UNNotificationAction(
            identifier: ActionID.restartServer,
            title: "Restart",
            options: [.foreground]
        )
        let openLogsAction = UNNotificationAction(
            identifier: ActionID.openLogs,
            title: "View Logs",
            options: [.foreground]
        )
        let crashCategory = UNNotificationCategory(
            identifier: CategoryID.serverCrash,
            actions: [restartAction, openLogsAction],
            intentIdentifiers: [],
            options: []
        )

        // Idle timeout category: Keep Running
        let keepRunningAction = UNNotificationAction(
            identifier: ActionID.keepRunning,
            title: "Start Again",
            options: [.foreground]
        )
        let idleCategory = UNNotificationCategory(
            identifier: CategoryID.serverIdle,
            actions: [keepRunningAction, openLogsAction],
            intentIdentifiers: [],
            options: []
        )

        // Healthy category: View Logs (informational)
        let healthyCategory = UNNotificationCategory(
            identifier: CategoryID.serverHealthy,
            actions: [openLogsAction],
            intentIdentifiers: [],
            options: []
        )

        center?.setNotificationCategories([crashCategory, idleCategory, healthyCategory])
    }

    // MARK: - Authorization

    private func requestAuthorization() {
        center?.requestAuthorization(options: [.alert, .sound, .badge]) { granted, error in
            if let error = error {
                print("Notification authorization error: \(error.localizedDescription)")
            }
        }
    }

    // MARK: - Preferences

    func isEnabled(for type: NotificationType) -> Bool {
        switch type {
        case .serverCrashed:
            return preferences.notifyOnServerCrash
        case .serverHealthy:
            return preferences.notifyOnServerStart
        case .serverIdleTimeout:
            return preferences.notifyOnServerStop
        }
    }

    // MARK: - Sound Effects

    /// Play a system sound effect if sounds are enabled in preferences.
    static func playSound(_ name: String) {
        guard PreferencesManager.shared.enableSounds else { return }
        NSSound(named: NSSound.Name(name))?.play()
    }

    /// Play a sound for a specific notification type.
    private func playSoundEffect(for type: NotificationType) {
        switch type {
        case .serverCrashed:
            Self.playSound("Basso")
        case .serverHealthy:
            Self.playSound("Glass")
        case .serverIdleTimeout:
            Self.playSound("Submarine")
        }
    }

    // MARK: - Send Notifications

    func sendNotification(type: NotificationType, serverName: String, message: String) {
        // Play sound effect regardless of notification availability
        playSoundEffect(for: type)

        // Check if notifications are available and enabled
        guard isAvailable, let center = center, isEnabled(for: type) else { return }

        let content = UNMutableNotificationContent()
        content.title = type.title
        content.body = "\(serverName): \(message)"
        content.sound = .default

        // Set interruption level for different notification types
        if type == .serverCrashed {
            content.interruptionLevel = .timeSensitive
        } else if type == .serverHealthy {
            content.interruptionLevel = .passive
        } else {
            content.interruptionLevel = .active
        }

        // Set category for actionable buttons
        content.categoryIdentifier = type.categoryIdentifier

        // Include server name in userInfo for action handling
        content.userInfo = ["serverName": serverName]

        // Create trigger (deliver immediately)
        let request = UNNotificationRequest(
            identifier: UUID().uuidString,
            content: content,
            trigger: nil
        )

        // Schedule notification
        center.add(request) { error in
            if let error = error {
                print("Failed to send notification: \(error.localizedDescription)")
            }
        }
    }

    // MARK: - Convenience Methods

    func notifyServerCrashed(serverName: String) {
        sendNotification(
            type: .serverCrashed,
            serverName: serverName,
            message: "Server has crashed and needs attention"
        )
    }

    func notifyServerHealthy(serverName: String) {
        sendNotification(
            type: .serverHealthy,
            serverName: serverName,
            message: "Server started successfully"
        )
    }

    func notifyServerIdleTimeout(serverName: String) {
        sendNotification(
            type: .serverIdleTimeout,
            serverName: serverName,
            message: "was automatically stopped after being idle"
        )
    }

    // MARK: - UNUserNotificationCenterDelegate

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        // Show notification even when app is in foreground
        completionHandler([.banner, .sound, .badge])
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        let actionIdentifier = response.actionIdentifier
        let userInfo = response.notification.request.content.userInfo
        let serverName = userInfo["serverName"] as? String

        switch actionIdentifier {
        case ActionID.restartServer:
            handleRestartAction(serverName: serverName)

        case ActionID.keepRunning:
            handleKeepRunningAction(serverName: serverName)

        case ActionID.openLogs:
            handleOpenLogsAction(serverName: serverName)

        case UNNotificationDefaultActionIdentifier:
            // User tapped the notification itself - activate the app
            NSApp.activate(ignoringOtherApps: true)

        default:
            break
        }

        completionHandler()
    }

    // MARK: - Action Handlers

    private func handleRestartAction(serverName: String?) {
        guard let serverName = serverName else { return }

        DispatchQueue.main.async {
            NSApp.activate(ignoringOtherApps: true)
            if let manager = ServerManagerAccessor.shared,
               let server = manager.servers.first(where: { $0.name == serverName }) {
                manager.startServer(server)
            }
        }
    }

    private func handleKeepRunningAction(serverName: String?) {
        guard let serverName = serverName else { return }

        DispatchQueue.main.async {
            NSApp.activate(ignoringOtherApps: true)
            if let manager = ServerManagerAccessor.shared,
               let server = manager.servers.first(where: { $0.name == serverName }) {
                manager.startServer(server)
            }
        }
    }

    private func handleOpenLogsAction(serverName: String?) {
        guard let serverName = serverName else { return }

        DispatchQueue.main.async {
            NSApp.activate(ignoringOtherApps: true)

            // Start streaming logs for the server
            if let manager = ServerManagerAccessor.shared,
               let server = manager.servers.first(where: { $0.name == serverName }) {
                manager.startStreamingLogs(for: server)
            }

            // Post notification to open the log viewer window
            NotificationCenter.default.post(
                name: NSNotification.Name("OpenLogViewer"),
                object: nil,
                userInfo: ["serverName": serverName]
            )
        }
    }
}
