import Foundation

/// Provides global access to the ServerManager instance.
///
/// Since ServerManager is created as a @StateObject in the App struct,
/// services that need to call back into it (notification actions, URL scheme,
/// App Intents) use this accessor. The App sets the instance at startup.
enum ServerManagerAccessor {
    private static weak var _instance: ServerManager?

    /// The shared ServerManager instance. Set by GroveMenubarApp at startup.
    static var shared: ServerManager? {
        get { _instance }
        set { _instance = newValue }
    }
}
