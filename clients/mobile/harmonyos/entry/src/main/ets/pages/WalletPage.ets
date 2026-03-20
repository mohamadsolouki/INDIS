import { AppStorage } from '../model/AppStorage'
import { CredentialRecord, CredentialStore, isRevoked, isExpired } from '../model/CredentialModel'
import { GatewayClient } from '../network/GatewayClient'
import { PersianCalendar } from '../utils/PersianCalendar'
import router from '@ohos.router'

interface CredentialListResponse {
  credentials: CredentialRecord[]
}

/**
 * Wallet page — lists all locally cached W3C Verifiable Credentials.
 *
 * Network strategy: try gateway → upsert to local store → return local cache.
 * PRD FR-006: credentials available offline for 72 hours.
 */
@Entry
@Component
struct WalletPage {

  @State private credentials: CredentialRecord[] = []
  @State private loading: boolean = true

  aboutToAppear(): void {
    this.load()
  }

  private load(): void {
    this.loading = true
    const did = AppStorage.getDID()
    const token = AppStorage.getToken()
    if (did && token) {
      const client = new GatewayClient()
      client.get<CredentialListResponse>(`/v1/identity/${did}/credentials`)
        .then((resp: CredentialListResponse) => {
          resp.credentials.forEach((c: CredentialRecord) => CredentialStore.upsert(c))
        })
        .catch(() => {/* fall through to local cache */})
        .finally(() => {
          this.credentials = CredentialStore.loadAll()
          this.loading = false
        })
    } else {
      this.credentials = CredentialStore.loadAll()
      this.loading = false
    }
  }

  build() {
    Column() {
      // Header
      Row() {
        Text('کیف پول').fontSize(24).fontWeight(FontWeight.Bold).fontColor(Color.White)
        Blank()
        Button() {
          SymbolGlyph($r('sys.symbol.arrow_clockwise')).fontSize(20).fontColor(Color.White)
        }
        .backgroundColor(Color.Transparent).onClick(() => this.load())
        Button() {
          SymbolGlyph($r('sys.symbol.lock_shield')).fontSize(20).fontColor(Color.White)
        }
        .backgroundColor(Color.Transparent)
        .onClick(() => router.pushUrl({ url: 'pages/PrivacyCenterPage' }))
      }
      .width('100%').padding({ top: 24, left: 20, right: 16, bottom: 12 })

      if (this.loading) {
        LoadingProgress().color('#1a56db').width(48).height(48).margin({ top: 60 })
      } else if (this.credentials.length === 0) {
        Column({ space: 12 }) {
          SymbolGlyph($r('sys.symbol.wallet_pass'))
            .fontSize(60).fontColor(['#374151'])
          Text('هیچ مدرکی در کیف پول یافت نشد')
            .fontSize(14).fontColor('#6b7280')
        }.margin({ top: 60 }).width('100%').alignItems(HorizontalAlign.Center)
      } else {
        List({ space: 12 }) {
          ForEach(this.credentials, (cred: CredentialRecord) => {
            ListItem() {
              CredentialCard({ record: cred })
            }
          })
        }
        .width('100%').layoutWeight(1)
        .padding({ left: 16, right: 16 })
      }
    }
    .width('100%').height('100%').backgroundColor('#0f1420')
  }
}

@Component
struct CredentialCard {

  record: CredentialRecord = {
    id: '', credentialType: '', issuer: '', issuedAt: '',
    expiresAt: '', vcJson: '{}', status: 'active'
  }

  build() {
    Column({ space: 12 }) {
      // Header
      Row() {
        SymbolGlyph($r('sys.symbol.doc_badge_checkmark'))
          .fontSize(22)
          .fontColor([isRevoked(this.record) ? '#ef4444' : '#1a56db'])
        Text(this.localizedType(this.record.credentialType))
          .fontSize(16).fontWeight(FontWeight.Bold).fontColor(Color.White).margin({ left: 8 })
        Blank()
        this.StatusBadge()
      }.width('100%')

      Divider().color('#ffffff1a')

      // Details
      this.DetailRow('صادرکننده', this.record.issuer)
      this.DetailRow('صدور', PersianCalendar.formatISO(this.record.issuedAt))
      this.DetailRow('انقضا', PersianCalendar.formatISO(this.record.expiresAt))
    }
    .width('100%').padding(16)
    .backgroundColor('#1f2937').borderRadius(16)
    .border({
      width: isRevoked(this.record) ? 1 : 0,
      color: isRevoked(this.record) ? '#ef444466' : Color.Transparent
    })
  }

  @Builder StatusBadge() {
    const revoked = isRevoked(this.record)
    const expired = isExpired(this.record)
    Text(revoked ? 'ابطال‌شده' : expired ? 'منقضی' : 'معتبر')
      .fontSize(11).fontWeight(FontWeight.Bold)
      .fontColor(revoked ? '#ef4444' : expired ? '#f59e0b' : '#10b981')
      .backgroundColor(revoked ? '#ef444420' : expired ? '#f59e0b20' : '#10b98120')
      .padding({ left: 8, right: 8, top: 3, bottom: 3 })
      .borderRadius(6)
  }

  @Builder DetailRow(label: string, value: string) {
    Row() {
      Text(label).fontSize(12).fontColor('#9ca3af').width(64)
      Text(value).fontSize(12).fontColor('#d1d5db')
    }.width('100%')
  }

  private localizedType(t: string): string {
    const map: Record<string, string> = {
      'NationalIdCredential': 'کارت ملی دیجیتال',
      'PassportCredential':   'گذرنامه',
      'VoterCredential':      'مدرک رأی‌دهی',
      'DriverLicense':        'گواهینامه رانندگی',
    }
    return map[t] ?? t
  }
}
