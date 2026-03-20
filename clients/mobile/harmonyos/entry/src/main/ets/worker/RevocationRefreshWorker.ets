import WorkSchedulerExtensionAbility from '@ohos.WorkSchedulerExtensionAbility'
import workScheduler from '@ohos.resourceschedule.workScheduler'
import hilog from '@ohos.hilog'
import { GatewayClient } from '../network/GatewayClient'
import { AppStorage } from '../model/AppStorage'

interface RevocationListResponse {
  revoked_credential_ids: string[]
}

/**
 * Background worker that refreshes the credential revocation list.
 *
 * Triggered by WorkScheduler every 6 hours (NETWORK_CONNECTED).
 * Fetches GET /v1/credential/revocations and stores the result in AppStorage.
 * Cached list is valid for 72h (PRD FR-006 offline operation).
 *
 * Equivalent to Android RevocationCacheWorker (WorkManager) and
 * iOS RevocationCacheService (BGAppRefreshTask).
 */
export class RevocationRefreshWorker extends WorkSchedulerExtensionAbility {

  private static readonly WORK_ID = 100

  onWorkStart(workInfo: workScheduler.WorkInfo): void {
    hilog.info(0x0000, 'INDIS', 'RevocationRefreshWorker onWorkStart')
    this.doRefresh()
  }

  onWorkStop(workInfo: workScheduler.WorkInfo): void {
    hilog.info(0x0000, 'INDIS', 'RevocationRefreshWorker onWorkStop')
  }

  private async doRefresh(): Promise<void> {
    try {
      const client = new GatewayClient()
      const resp = await client.get<RevocationListResponse>('/v1/credential/revocations')
      AppStorage.saveRevocationList(JSON.stringify(resp))
      hilog.info(0x0000, 'INDIS', 'Revocation list refreshed: %{public}d ids',
        resp.revoked_credential_ids.length)
    } catch (e) {
      hilog.warn(0x0000, 'INDIS', 'Revocation refresh failed (using cached): %{public}s', String(e))
    }
  }

  // ── Scheduling ────────────────────────────────────────────────────────────

  static schedule(): void {
    const workInfo: workScheduler.WorkInfo = {
      workId:          RevocationRefreshWorker.WORK_ID,
      bundleName:      'org.indis.harmonyos',
      abilityName:     'RevocationRefreshWorker',
      networkType:     workScheduler.NetworkType.NETWORK_TYPE_ANY,
      isRepeat:        true,
      repeatCycleTime: 6 * 60 * 60 * 1000,   // 6 hours in ms
      isPersisted:     true,
    }
    try {
      workScheduler.startWork(workInfo)
    } catch (e) {
      hilog.warn(0x0000, 'INDIS', 'WorkScheduler.startWork failed: %{public}s', String(e))
    }
  }
}
