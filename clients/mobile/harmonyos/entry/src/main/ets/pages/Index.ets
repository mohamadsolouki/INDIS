import router from '@ohos.router'
import { AppStorage } from '../model/AppStorage'

/**
 * Root index page — redirects to OnboardingPage or MainPage based on auth state.
 * This is a pass-through; no UI is rendered here.
 */
@Entry
@Component
struct Index {
  aboutToAppear(): void {
    if (AppStorage.isAuthenticated()) {
      router.replaceUrl({ url: 'pages/MainPage' })
    } else {
      router.replaceUrl({ url: 'pages/OnboardingPage' })
    }
  }

  build() {
    Column() {
      // Blank while routing
    }
    .width('100%')
    .height('100%')
    .backgroundColor('#111827')
  }
}
