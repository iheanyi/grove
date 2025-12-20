import Foundation
import UserNotifications

/// Service for managing macOS notifications
class NotificationService: NSObject, UNUserNotificationCenterDelegate {
    static let shared = NotificationService()

    private let center = UNUserNotificationCenter.current()
    private let preferences = PreferencesManager.shared

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
                return "Server Stopped"
            }
        }

        var preferencesKey: String {
            "notification_\(rawValue)_enabled"
        }
    }

    // MARK: - Initialization

    private override init() {
        super.init()
        center.delegate = self
        requestAuthorization()
    }

    // MARK: - Authorization

    private func requestAuthorization() {
        center.requestAuthorization(options: [.alert, .sound, .badge, .criticalAlert]) { granted, error in
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

    // MARK: - Send Notifications

    func sendNotification(type: NotificationType, serverName: String, message: String) {
        // Check if notifications are enabled for this type
        guard isEnabled(for: type) else { return }

        let content = UNMutableNotificationContent()
        content.title = type.title
        content.body = "\(serverName): \(message)"
        content.sound = .default

        // Set critical alert for crashed servers
        if type == .serverCrashed {
            content.interruptionLevel = .critical
            content.sound = UNNotificationSound.defaultCritical
        } else if type == .serverHealthy {
            content.interruptionLevel = .passive
        } else {
            content.interruptionLevel = .active
        }

        // Category for actions (optional - can add later)
        content.categoryIdentifier = type.identifier

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
            message: "Server stopped due to idle timeout"
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
        // Handle notification tap (optional - can add actions later)
        completionHandler()
    }
}
