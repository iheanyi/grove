// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "WTMenubar",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .executable(name: "WTMenubar", targets: ["WTMenubar"])
    ],
    targets: [
        .executableTarget(
            name: "WTMenubar",
            path: "Sources/WTMenubar"
        )
    ]
)
