// swift-tools-version: 5.9
// INDIS iOS App — Swift Package Manager manifest
// Minimum deployment target: iOS 14.0

import PackageDescription

let package = Package(
    name: "IndisApp",
    platforms: [
        .iOS(.v14),
    ],
    products: [
        .library(name: "IndisApp", targets: ["IndisApp"]),
    ],
    dependencies: [],
    targets: [
        .target(
            name: "IndisApp",
            path: "Sources/IndisApp",
            resources: [
                .process("Resources"),
            ]
        ),
        .testTarget(
            name: "IndisAppTests",
            dependencies: ["IndisApp"],
            path: "Tests/IndisAppTests"
        ),
    ]
)
