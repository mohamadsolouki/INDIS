import UIAbility from '@ohos.app.ability.UIAbility'
import hilog from '@ohos.hilog'
import window from '@ohos.window'
import WorkSchedulerExtensionAbility from '@ohos.WorkSchedulerExtensionAbility'
import { AppStorage } from '../model/AppStorage'
import { RevocationRefreshWorker } from '../worker/RevocationRefreshWorker'

/**
 * INDIS HarmonyOS Entry Ability.
 *
 * Initialises persistent AppStorage (DID, JWT, settings) from preferences,
 * schedules the revocation cache background worker, then launches the root page.
 *
 * PRD FR-001: device-bound key; PRD FR-006: 72h offline revocation cache.
 */
export default class EntryAbility extends UIAbility {

  onCreate(want: object, launchParam: object): void {
    hilog.info(0x0000, 'INDIS', 'EntryAbility onCreate')
    AppStorage.load(this.context)
    RevocationRefreshWorker.schedule()
  }

  onWindowStageCreate(windowStage: window.WindowStage): void {
    windowStage.loadContent('pages/Index', (err) => {
      if (err.code !== 0) {
        hilog.error(0x0000, 'INDIS', 'loadContent failed: %{public}s', JSON.stringify(err))
      }
    })
  }

  onForeground(): void {}
  onBackground(): void {}
  onDestroy(): void {}
}
