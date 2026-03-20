import router from '@ohos.router'
import { AppStorage } from '../model/AppStorage'
import { GatewayClient } from '../network/GatewayClient'
import { PersianCalendar } from '../utils/PersianCalendar'

interface PrivacyEvent {
  id: string
  event_type: string
  verifier_id?: string
  timestamp: string
  predicate?: string
}

interface ConsentRule {
  id: string
  verifier_id: string
  attribute: string
  granted: boolean
  expires_at?: string
}

interface ExportResponse {
  download_url: string
  expires_at: string
}

/**
 * Privacy Center — 3-tab screen.
 * Tab 1: Disclosure history  (GET /v1/privacy/history)
 * Tab 2: Consent rules       (GET /v1/privacy/consent)
 * Tab 3: Data export         (POST /v1/privacy/export)
 * Mirrors iOS PrivacyCenterView and Android PrivacyCenterActivity.
 */
@Entry
@Component
struct PrivacyCenterPage {

  @State private tab: number = 0
  @State private history: PrivacyEvent[] = []
  @State private consent: ConsentRule[] = []
  @State private loading: boolean = true
  @State private exporting: boolean = false
  @State private exportURL: string = ''
  @State private exportFormat: string = 'json'
  @State private errorMsg: string = ''

  aboutToAppear(): void {
    this.loadTab()
  }

  private loadTab(): void {
    this.loading = true
    const client = new GatewayClient()
    const token = AppStorage.getToken()
    if (this.tab === 0) {
      client.get<{ events: PrivacyEvent[] }>('/v1/privacy/history')
        .then(r => { this.history = r.events })
        .catch(() => {})
        .finally(() => { this.loading = false })
    } else if (this.tab === 1) {
      client.get<{ rules: ConsentRule[] }>('/v1/privacy/consent')
        .then(r => { this.consent = r.rules })
        .catch(() => {})
        .finally(() => { this.loading = false })
    } else {
      this.loading = false
    }
  }

  build() {
    Column() {
      // Header
      Row() {
        Button() {
          SymbolGlyph($r('sys.symbol.chevron_right')).fontSize(20).fontColor(Color.White)
        }.backgroundColor(Color.Transparent).onClick(() => router.back())
        Text('مرکز حریم خصوصی').fontSize(18).fontWeight(FontWeight.Bold).fontColor(Color.White)
        Blank()
      }.width('100%').padding({ top: 20, left: 8, right: 20, bottom: 8 })

      // Tab bar
      Row({ space: 0 }) {
        ForEach(['تاریخچه', 'رضایت', 'خروجی'], (label: string, i: number) => {
          Button(label).layoutWeight(1).height(38)
            .backgroundColor(this.tab === i ? '#1a56db' : Color.Transparent)
            .fontColor(this.tab === i ? Color.White : '#9ca3af').fontSize(13)
            .border({ width: 1, color: '#374151' })
            .borderRadius(i === 0 ? { topLeft: 10, bottomLeft: 10 } :
              i === 2 ? { topRight: 10, bottomRight: 10 } : 0)
            .onClick(() => { this.tab = i; this.loadTab() })
        })
      }.width('92%').alignSelf(ItemAlign.Center).margin({ bottom: 16 })

      if (this.loading) {
        LoadingProgress().color('#1a56db').width(48).height(48).margin({ top: 40 })
      } else if (this.tab === 0) {
        this.HistoryTab()
      } else if (this.tab === 1) {
        this.ConsentTab()
      } else {
        this.ExportTab()
      }
    }
    .width('100%').height('100%').backgroundColor('#0f1420')
  }

  @Builder HistoryTab() {
    if (this.history.length === 0) {
      this.EmptyState('doc.text', 'هیچ رویدادی ثبت نشده')
    } else {
      List({ space: 8 }) {
        ForEach(this.history, (e: PrivacyEvent) => {
          ListItem() {
            Column({ space: 6 }) {
              Row() {
                Text(this.localizeEventType(e.event_type))
                  .fontSize(14).fontWeight(FontWeight.Bold).fontColor(Color.White).layoutWeight(1)
                Text(PersianCalendar.formatISO(e.timestamp))
                  .fontSize(11).fontColor('#6b7280')
              }.width('100%')
              if (e.verifier_id) {
                Text(`تأیید‌کننده: ${e.verifier_id}`).fontSize(12).fontColor('#9ca3af')
              }
              if (e.predicate) {
                Text(`ادعا: ${e.predicate}`).fontSize(12).fontColor('#1a56db')
              }
            }
            .width('100%').padding(14).backgroundColor('#1f2937').borderRadius(12)
          }
        })
      }
      .width('92%').alignSelf(ItemAlign.Center).layoutWeight(1)
    }
  }

  @Builder ConsentTab() {
    if (this.consent.length === 0) {
      this.EmptyState('hand.raised', 'هیچ قانون رضایتی ثبت نشده')
    } else {
      List({ space: 8 }) {
        ForEach(this.consent, (r: ConsentRule) => {
          ListItem() {
            Row({ space: 12 }) {
              SymbolGlyph(r.granted ? $r('sys.symbol.checkmark_circle_fill') : $r('sys.symbol.xmark_circle_fill'))
                .fontSize(22).fontColor([r.granted ? '#10b981' : '#ef4444'])
              Column({ space: 4 }) {
                Text(r.attribute).fontSize(14).fontWeight(FontWeight.Bold).fontColor(Color.White)
                Text(r.verifier_id).fontSize(12).fontColor('#9ca3af')
              }.alignItems(HorizontalAlign.Start)
            }
            .width('100%').padding(14).backgroundColor('#1f2937').borderRadius(12)
          }
        })
      }
      .width('92%').alignSelf(ItemAlign.Center).layoutWeight(1)
    }
  }

  @Builder ExportTab() {
    Column({ space: 20 }) {
      SymbolGlyph($r('sys.symbol.square_and_arrow_up')).fontSize(56).fontColor(['#1a56db'])
        .margin({ top: 32 })
      Text('صدور اطلاعات شخصی').fontSize(18).fontWeight(FontWeight.Bold).fontColor(Color.White)
      Text('کلیه اطلاعاتی که INDIS درباره شما دارد را دریافت کنید.')
        .fontSize(14).fontColor('#9ca3af').textAlign(TextAlign.Center)
        .padding({ left: 24, right: 24 })

      Row({ space: 0 }) {
        Button('JSON').layoutWeight(1).height(38)
          .backgroundColor(this.exportFormat === 'json' ? '#1a56db' : Color.Transparent)
          .fontColor(this.exportFormat === 'json' ? Color.White : '#9ca3af').fontSize(14)
          .border({ width: 1, color: '#374151' }).borderRadius({ topLeft: 10, bottomLeft: 10 })
          .onClick(() => { this.exportFormat = 'json' })
        Button('PDF').layoutWeight(1).height(38)
          .backgroundColor(this.exportFormat === 'pdf' ? '#1a56db' : Color.Transparent)
          .fontColor(this.exportFormat === 'pdf' ? Color.White : '#9ca3af').fontSize(14)
          .border({ width: 1, color: '#374151' }).borderRadius({ topRight: 10, bottomRight: 10 })
          .onClick(() => { this.exportFormat = 'pdf' })
      }.width('60%')

      if (this.exportURL !== '') {
        Text('لینک دانلود:').fontSize(12).fontColor('#9ca3af')
        Text(this.exportURL).fontSize(11).fontColor('#1a56db').fontFamily('monospace')
          .padding({ left: 16, right: 16 }).textAlign(TextAlign.Center)
      }
      if (this.errorMsg !== '') {
        Text(this.errorMsg).fontSize(12).fontColor('#ef4444')
      }
      Button(this.exporting ? 'در حال پردازش…' : 'درخواست خروجی')
        .width('88%').height(52).backgroundColor('#1a56db')
        .fontColor(Color.White).fontSize(16).borderRadius(14)
        .enabled(!this.exporting)
        .onClick(() => { this.requestExport() })
      Blank()
    }
    .width('100%').alignItems(HorizontalAlign.Center).layoutWeight(1)
  }

  @Builder EmptyState(icon: string, message: string) {
    Column({ space: 12 }) {
      SymbolGlyph($r('sys.symbol.' + icon)).fontSize(56).fontColor(['#374151'])
      Text(message).fontSize(14).fontColor('#6b7280')
    }.width('100%').alignItems(HorizontalAlign.Center).margin({ top: 60 })
  }

  private requestExport(): void {
    this.exporting = true
    this.errorMsg = ''
    const client = new GatewayClient()
    client.post<ExportResponse>('/v1/privacy/export', { format: this.exportFormat })
      .then(r => { this.exportURL = r.download_url })
      .catch(e => { this.errorMsg = String(e) })
      .finally(() => { this.exporting = false })
  }

  private localizeEventType(t: string): string {
    const map: Record<string, string> = {
      'disclosure': 'افشای اطلاعات',
      'consent':    'رضایت',
      'export':     'صدور خروجی',
    }
    return map[t] ?? t
  }
}
