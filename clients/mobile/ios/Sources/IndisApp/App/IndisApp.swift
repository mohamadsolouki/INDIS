import SwiftUI

/// INDIS Citizen iOS Application entry point.
///
/// Checks for an existing DID/JWT in the Keychain on launch.
/// If present the user lands on MainTabView; otherwise Onboarding is shown.
@main
struct IndisApp: App {

    @StateObject private var appState = AppState()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(appState)
                .environment(\.layoutDirection, .rightToLeft)  // Persian RTL
        }
    }
}
