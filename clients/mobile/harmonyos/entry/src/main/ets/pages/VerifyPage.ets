import router from '@ohos.router'
import { AppStorage } from '../model/AppStorage'
import { GatewayClient } from '../network/GatewayClient'

interface VerifyProofResponse {
  valid: boolean
  message?: string
}

/**
 * Verify page — ZK credential presentation and scanning.
 *
 * PRD FR-006: proof generation works fully offline with cached credentials.
 * PRD FR-013: verifiers see ONLY a boolean result — no raw citizen data.
 *
 * Present mode: generates a mock ZK proof and displays it as a QR matrix.
 * Scan mode: uses the camera scanner to decode a citizen's QR and verify.
 */
@Entry
@Component
struct VerifyPage {

  @State private mode: number = 0   // 0=present 1=scan
  @State private predicate: string = 'age_over_18'
  @State private qrText: string = ''
  @State private generating: boolean = false
  @State private verifyResult: boolean | null = null
  @State private verifying: boolean = false

  private readonly predicates = [
    { id: 'age_over_18',       label: 'سن بیش از ۱۸' },
    { id: 'citizenship_ir',    label: 'تابعیت ایران' },
    { id: 'voter_eligible',    label: 'واجد شرایط رأی' },
    { id: 'credential_valid',  label: 'مدرک معتبر' },
  ]

  build() {
    Column() {
      // Header
      Row() {
        Button() {
          SymbolGlyph($r('sys.symbol.chevron_right')).fontSize(20).fontColor(Color.White)
        }.backgroundColor(Color.Transparent).onClick(() => router.back())
        Text('تأیید هویت').fontSize(20).fontWeight(FontWeight.Bold).fontColor(Color.White)
          .margin({ right: 32 })
        Blank()
      }.width('100%').padding({ top: 20, left: 8, right: 20 })

      // Mode toggle
      Row({ space: 0 }) {
        Button('ارائه مدرک').layoutWeight(1).height(40)
          .backgroundColor(this.mode === 0 ? '#1a56db' : Color.Transparent)
          .fontColor(this.mode === 0 ? Color.White : '#9ca3af').fontSize(14)
          .border({ width: 1, color: '#374151' }).borderRadius({ topLeft: 10, bottomLeft: 10 })
          .onClick(() => { this.mode = 0 })
        Button('اسکن تأیید').layoutWeight(1).height(40)
          .backgroundColor(this.mode === 1 ? '#1a56db' : Color.Transparent)
          .fontColor(this.mode === 1 ? Color.White : '#9ca3af').fontSize(14)
          .border({ width: 1, color: '#374151' }).borderRadius({ topRight: 10, bottomRight: 10 })
          .onClick(() => { this.mode = 1 })
      }
      .width('88%').margin({ top: 16, bottom: 20 })
      .alignSelf(ItemAlign.Center)

      if (this.mode === 0) {
        this.PresentMode()
      } else {
        this.ScanMode()
      }
    }
    .width('100%').height('100%').backgroundColor('#0f1420')
  }

  @Builder PresentMode() {
    Scroll() {
      Column({ space: 16 }) {
        Text('انتخاب ادعا').fontSize(14).fontColor('#9ca3af')
        ForEach(this.predicates, (p: { id: string; label: string }) => {
          Row() {
            Text(p.label).fontSize(14).fontColor(Color.White).layoutWeight(1)
            if (this.predicate === p.id) {
              SymbolGlyph($r('sys.symbol.checkmark_circle_fill')).fontSize(20).fontColor(['#1a56db'])
            }
          }
          .width('92%').padding(14)
          .backgroundColor(this.predicate === p.id ? '#1a56db18' : '#1f2937')
          .borderRadius(10)
          .border({ width: this.predicate === p.id ? 1 : 0, color: '#1a56db' })
          .onClick(() => { this.predicate = p.id; this.qrText = '' })
        })

        if (this.qrText !== '') {
          // QR representation (text-based for ArkUI — real QR via barcode library in production)
          Column() {
            Text('✅ ZK Proof تولید شد').fontSize(14).fontColor('#10b981')
            Text(this.qrText.substring(0, 60) + '…')
              .fontSize(10).fontColor('#6b7280').fontFamily('monospace')
              .maxLines(3).textOverflow({ overflow: TextOverflow.Ellipsis })
              .padding({ top: 8 })
          }
          .width('92%').padding(16).backgroundColor('#1f2937').borderRadius(14)
        }

        Button(this.generating ? 'در حال تولید…' : (this.qrText ? 'تولید مجدد' : 'تولید ZK Proof'))
          .width('88%').height(52)
          .backgroundColor(this.generating ? '#374151' : '#1a56db')
          .fontColor(Color.White).fontSize(16).borderRadius(14)
          .enabled(!this.generating)
          .onClick(() => { this.generateProof() })
          .margin({ bottom: 32 })
      }
      .width('100%').alignItems(HorizontalAlign.Center).padding({ top: 8 })
    }
    .layoutWeight(1)
  }

  @Builder ScanMode() {
    Column({ space: 20 }) {
      if (this.verifying) {
        LoadingProgress().color('#1a56db').width(56).height(56).margin({ top: 40 })
        Text('در حال تأیید…').fontSize(16).fontColor(Color.White)
      } else if (this.verifyResult !== null) {
        // Full-screen boolean result (PRD FR-013)
        Column({ space: 16 }) {
          SymbolGlyph(this.verifyResult ?
            $r('sys.symbol.checkmark_seal_fill') :
            $r('sys.symbol.xmark_seal_fill'))
            .fontSize(100)
            .fontColor([this.verifyResult ? '#10b981' : '#ef4444'])
          Text(this.verifyResult ? 'تأیید شد ✓' : 'رد شد ✗')
            .fontSize(32).fontWeight(FontWeight.Bold)
            .fontColor(this.verifyResult ? '#10b981' : '#ef4444')
          Button('اسکن مجدد').width('60%').height(48)
            .backgroundColor('#1f2937').fontColor(Color.White)
            .borderRadius(12).onClick(() => { this.verifyResult = null })
        }
        .width('100%').alignItems(HorizontalAlign.Center).layoutWeight(1)
        .backgroundColor(this.verifyResult ? '#10b98112' : '#ef444412')
      } else {
        SymbolGlyph($r('sys.symbol.qrcode_viewfinder')).fontSize(80).fontColor(['#374151'])
          .margin({ top: 40 })
        Text('QR تأیید‌کننده را اسکن کنید').fontSize(14).fontColor('#9ca3af')
        // HarmonyOS camera scanner would be invoked here via @ohos.multimedia.camera
        Button('شروع اسکن').width('88%').height(52)
          .backgroundColor('#1a56db').fontColor(Color.White).fontSize(16).borderRadius(14)
          .onClick(() => { this.simulateScan() })
        Blank()
      }
    }
    .width('100%').alignItems(HorizontalAlign.Center).layoutWeight(1)
  }

  private generateProof(): void {
    this.generating = true
    const payload = {
      proof:        btoa(String.fromCharCode(...new Array(128).fill(0xAB))),
      nonce:        Math.random().toString(36).slice(2),
      predicate:    this.predicate,
      proof_system: 'groth16',
    }
    // Simulated async proof generation
    setTimeout(() => {
      this.qrText = JSON.stringify(payload)
      this.generating = false
    }, 600)
  }

  private simulateScan(): void {
    // In production: use @ohos.multimedia.camera + barcode decoding
    this.verifying = true
    const mockPayload = JSON.stringify({
      verifier_id:       'dev-verifier',
      proof_b64:         btoa(String.fromCharCode(...new Array(128).fill(0xAB))),
      nonce:             'mock-nonce',
      predicate:         'age_over_18',
      credential_type:   'national_id',
      proof_system:      'groth16',
      public_inputs_b64: '',
    })
    const client = new GatewayClient()
    client.post<VerifyProofResponse>('/v1/verifier/verify', JSON.parse(mockPayload))
      .then((resp: VerifyProofResponse) => { this.verifyResult = resp.valid })
      .catch(() => { this.verifyResult = false })
      .finally(() => { this.verifying = false })
  }
}
