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
            path: "Sources/GroveMenubar",
            resources: [
                .process("Resources")
            ]
        ),
        // WidgetKit extension source. This compiles as a library target for
        // validation. To run as a real widget, create an Xcode widget extension
        // target that embeds these sources (see Sources/GroveWidget/README.md).
        .executableTarget(
            name: "GroveWidget",
            path: "Sources/GroveWidget"
        ),
        .testTarget(
            name: "GroveMenubarTests",
            dependencies: ["GroveMenubar"],
            path: "Tests/GroveMenubarTests"
        )
    ]
)
