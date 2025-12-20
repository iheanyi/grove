// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "GroveMenubar",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .executable(name: "GroveMenubar", targets: ["GroveMenubar"])
    ],
    targets: [
        .executableTarget(
            name: "GroveMenubar",
            path: "Sources/GroveMenubar"
        ),
        .testTarget(
            name: "GroveMenubarTests",
            path: "Tests/GroveMenubarTests"
        )
    ]
)
