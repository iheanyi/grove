# Performance Guidelines for Grove Menubar App

## Main Thread Blocking - The #1 Enemy

MenuBarExtra apps are especially sensitive to main thread blocking because:
1. The popup view body is evaluated on every open
2. After sleep, macOS pages out memory for inactive apps
3. When the menubar opens, paged-out code must load back into memory
4. This happens ON THE MAIN THREAD during SwiftUI view evaluation

### The Scanner Incident (Dec 2024)

**Symptom:** App would beachball (spinning cursor) on:
- Fresh app start
- Opening menubar after wake from sleep

**Root Cause:** Using `Scanner` class to parse uptime strings like "2h34m12s"

```swift
// BAD - Scanner is a heavy Foundation class
var formattedUptime: String? {
    let scanner = Scanner(string: uptime)  // Heavy allocation
    scanner.charactersToBeSkipped = CharacterSet.letters
    while !scanner.isAtEnd { ... }
}
```

**Fix:** Simple character iteration with no Foundation dependencies

```swift
// GOOD - No heavy Foundation classes
var formattedUptime: String? {
    var currentNumber = ""
    for char in uptime {
        if char.isNumber {
            currentNumber.append(char)
        } else if let value = Int(currentNumber) {
            // parse h/m/s
        }
    }
}
```

### Guidelines

#### Never Use These in Computed Properties Called During View Rendering:
- `Scanner`
- `NSRegularExpression` (create as `static let` if needed)
- `DateFormatter` (create as `static let` if needed)
- `NumberFormatter` (create as `static let` if needed)
- Any heavy Foundation class that requires initialization

#### Safe Patterns:
1. **Pre-compile regexes as static lets:**
   ```swift
   private static let myRegex = try? NSRegularExpression(pattern: "...")
   ```

2. **Do parsing on background threads:**
   ```swift
   DispatchQueue.global().async {
       let result = heavyParsing()
       DispatchQueue.main.async {
           self.value = result
       }
   }
   ```

3. **Cache computed values instead of recalculating:**
   ```swift
   // Store formatted string when data is fetched, not in view
   ```

4. **Use simple Swift for string parsing:**
   - Character iteration
   - `String.split()`
   - `String.components(separatedBy:)`

#### Testing for Main Thread Issues:
1. Put computer to sleep for 30+ seconds
2. Wake and immediately click menubar icon
3. If it beachballs, profile with Instruments > Time Profiler
4. Look for main thread blocking in view body evaluation

## Current Architecture Safety

- **ServerManager.refresh()** - JSON parsing on background thread ✅
- **GitHubService** - All work on background threads ✅
- **LogHighlighter** - Static regex compilation, only used in log viewer ✅
- **Server.formattedUptime** - Simple character iteration ✅
