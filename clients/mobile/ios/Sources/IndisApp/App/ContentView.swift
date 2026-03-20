import SwiftUI

/// Root view: routes to Onboarding or MainTabView based on auth state.
struct ContentView: View {
    @EnvironmentObject private var appState: AppState

    var body: some View {
        Group {
            if appState.isAuthenticated {
                MainTabView()
            } else {
                OnboardingView()
            }
        }
        .animation(.easeInOut, value: appState.isAuthenticated)
    }
}
