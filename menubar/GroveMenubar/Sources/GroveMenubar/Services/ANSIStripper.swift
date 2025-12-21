import Foundation

/// Utility for stripping ANSI escape codes from log output
struct ANSIStripper {
    /// Regex pattern for ANSI escape sequences
    /// ICU regex uses \x{1B} for the ESC character (0x1B)
    private static let ansiPattern = "\\x{1B}\\[[0-9;]*[a-zA-Z]|\\x{1B}\\([AB]"
    private static let ansiRegex = try? NSRegularExpression(pattern: ansiPattern, options: [])

    /// Strips ANSI escape codes from a string (e.g., color codes like ESC[1m ESC[35m)
    static func strip(_ string: String) -> String {
        guard let regex = ansiRegex else { return string }
        let range = NSRange(string.startIndex..., in: string)
        return regex.stringByReplacingMatches(in: string, options: [], range: range, withTemplate: "")
    }
}
