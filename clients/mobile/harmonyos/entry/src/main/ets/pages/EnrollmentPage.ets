import router from '@ohos.router'
import { AppStorage } from '../model/AppStorage'
import { GatewayClient } from '../network/GatewayClient'

interface StartEnrollmentResponse {
  enrollment_id: string
  status: string
}

interface EnrollmentStatusResponse {
  enrollment_id: string
  status: string
  message?: string
}

/**
 * Enrollment page — 4-step flow (pathway → document → biometric → wait).
 * Mirrors Android EnrollmentActivity and iOS EnrollmentView.
 * PRD FR-003: three enrollment pathways.
 */
@Entry
@Component
struct EnrollmentPage {

  @State private step: number = 0    // 0=pathway 1=document 2=biometric 3=waiting
  @State private pathway: string = 'standard'
  @State private enrollmentId: string = ''
  @State private enrollStatus: string = ''
  @State private frontDone: boolean = false
  @State private backDone: boolean = false
  @State private faceDone: boolean = false
  @State private fpDone: boolean = false
  @State private loading: boolean = false
  @State private errorMsg: string = ''

  private readonly pathways = [
    { id: 'standard', title: 'استاندارد',      desc: 'اسکن مدارک + بیومتریک',    icon: $r('sys.symbol.doc_text_fill') },
    { id: 'enhanced', title: 'پیشرفته',         desc: 'تطبیق با ثبت احوال',       icon: $r('sys.symbol.building_columns_fill') },
    { id: 'social',   title: 'تأیید اجتماعی',  desc: '۳+ گواه + بیومتریک',      icon: $r('sys.symbol.person_3_fill') },
  ]

  build() {
    Column() {
      // Header
      Row() {
        Button() {
          SymbolGlyph($r('sys.symbol.chevron_right')).fontSize(20).fontColor(Color.White)
        }
        .backgroundColor(Color.Transparent)
        .onClick(() => router.back())
        Text('ثبت‌نام').fontSize(20).fontWeight(FontWeight.Bold).fontColor(Color.White)
          .margin({ right: 32 })
        Blank()
      }.width('100%').padding({ top: 20, left: 8, right: 20 })

      // Progress bar
      this.ProgressBar()

      // Step content
      if (this.step === 0) { this.PathwayStep() }
      else if (this.step === 1) { this.DocumentStep() }
      else if (this.step === 2) { this.BiometricStep() }
      else { this.WaitingStep() }
    }
    .width('100%').height('100%').backgroundColor('#0f1420')
  }

  @Builder ProgressBar() {
    Row({ space: 0 }) {
      ForEach([0, 1, 2, 3], (i: number) => {
        Row() {
          Column() {
            Circle({ width: 24, height: 24 })
              .fill(i <= this.step ? '#1a56db' : '#374151')
              .overlay(
                Text(String(i + 1)).fontSize(11).fontWeight(FontWeight.Bold).fontColor(Color.White)
              )
            Text(['مسیر', 'مدارک', 'بیومتریک', 'تأیید'][i])
              .fontSize(10)
              .fontColor(i <= this.step ? Color.White : '#6b7280')
              .margin({ top: 4 })
          }.alignItems(HorizontalAlign.Center)
          if (i < 3) {
            Divider().vertical(false).strokeWidth(2)
              .color(i < this.step ? '#1a56db' : '#374151')
              .layoutWeight(1)
          }
        }.layoutWeight(i < 3 ? 1 : 0)
      })
    }
    .width('92%').alignSelf(ItemAlign.Center).margin({ top: 16, bottom: 24 })
  }

  @Builder PathwayStep() {
    Column({ space: 16 }) {
      Text('مسیر ثبت‌نام را انتخاب کنید')
        .fontSize(16).fontColor('#d1d5db')
      ForEach(this.pathways, (p: { id: string; title: string; desc: string; icon: Resource }) => {
        Row({ space: 14 }) {
          SymbolGlyph(p.icon).fontSize(26)
            .fontColor([this.pathway === p.id ? '#1a56db' : '#9ca3af'])
          Column({ space: 4 }) {
            Text(p.title).fontSize(15).fontWeight(FontWeight.Bold).fontColor(Color.White)
            Text(p.desc).fontSize(12).fontColor('#9ca3af')
          }.alignItems(HorizontalAlign.Start).layoutWeight(1)
          if (this.pathway === p.id) {
            SymbolGlyph($r('sys.symbol.checkmark_circle_fill')).fontSize(20).fontColor(['#1a56db'])
          }
        }
        .width('92%').padding(16)
        .backgroundColor(this.pathway === p.id ? '#1a56db18' : '#1f2937')
        .borderRadius(12)
        .border({ width: this.pathway === p.id ? 1.5 : 0, color: '#1a56db' })
        .onClick(() => { this.pathway = p.id })
      })
      Blank()
      Button('ادامه').width('88%').height(52).backgroundColor('#1a56db')
        .fontColor(Color.White).fontSize(16).borderRadius(14)
        .onClick(() => { this.step = 1 })
        .margin({ bottom: 32 })
    }
    .width('100%').alignItems(HorizontalAlign.Center).layoutWeight(1)
  }

  @Builder DocumentStep() {
    Column({ space: 20 }) {
      Text('عکس کارت ملی').fontSize(18).fontWeight(FontWeight.Bold).fontColor(Color.White)
      Text('روی کارت ملی را اسکن کنید').fontSize(14).fontColor('#9ca3af')
      Row({ space: 16 }) {
        this.DocumentTile('روی کارت', this.frontDone, () => { this.frontDone = true })
        this.DocumentTile('پشت کارت', this.backDone,  () => { this.backDone = true })
      }.width('92%')
      Blank()
      Button('ادامه').width('88%').height(52)
        .backgroundColor(this.frontDone && this.backDone ? '#1a56db' : '#374151')
        .fontColor(Color.White).fontSize(16).borderRadius(14)
        .enabled(this.frontDone && this.backDone)
        .onClick(() => { this.step = 2 })
        .margin({ bottom: 32 })
    }
    .width('100%').alignItems(HorizontalAlign.Center).layoutWeight(1)
  }

  @Builder DocumentTile(label: string, done: boolean, onTap: () => void) {
    Column({ space: 10 }) {
      Stack() {
        RoundedRectangle({ width: '100%', height: 120, radius: 12 })
          .fill(done ? '#10b98118' : '#1f2937')
        SymbolGlyph(done ? $r('sys.symbol.checkmark_circle_fill') : $r('sys.symbol.creditcard_fill'))
          .fontSize(36).fontColor([done ? '#10b981' : '#9ca3af'])
      }
      Text(done ? 'ثبت شد ✓' : label).fontSize(13)
        .fontColor(done ? '#10b981' : '#d1d5db')
    }
    .layoutWeight(1).onClick(onTap)
  }

  @Builder BiometricStep() {
    Column({ space: 16 }) {
      Text('بیومتریک').fontSize(18).fontWeight(FontWeight.Bold).fontColor(Color.White)
      Text('چهره و اثر انگشت خود را ثبت کنید').fontSize(14).fontColor('#9ca3af')
      this.BiometricTile('اسکن چهره', $r('sys.symbol.face_smiling'), this.faceDone, '#1a56db',
        () => { this.faceDone = true })
      this.BiometricTile('اثر انگشت', $r('sys.symbol.hand_point_up_left_fill'), this.fpDone, '#7c3aed',
        () => { this.fpDone = true })
      if (this.errorMsg !== '') {
        Text(this.errorMsg).fontSize(12).fontColor('#ef4444').padding({ left: 24, right: 24 })
      }
      Blank()
      Button(this.loading ? 'در حال ارسال…' : 'تأیید و ادامه')
        .width('88%').height(52)
        .backgroundColor(this.faceDone && this.fpDone && !this.loading ? '#1a56db' : '#374151')
        .fontColor(Color.White).fontSize(16).borderRadius(14)
        .enabled(this.faceDone && this.fpDone && !this.loading)
        .onClick(() => { this.submitEnrollment() })
        .margin({ bottom: 32 })
    }
    .width('100%').alignItems(HorizontalAlign.Center).layoutWeight(1)
    .padding({ left: 16, right: 16 })
  }

  @Builder BiometricTile(label: string, icon: Resource, done: boolean, color: string, onTap: () => void) {
    Row({ space: 14 }) {
      Stack() {
        Circle({ width: 52, height: 52 })
          .fill(done ? '#10b98118' : color + '20')
        SymbolGlyph(done ? $r('sys.symbol.checkmark_circle_fill') : icon)
          .fontSize(26).fontColor([done ? '#10b981' : color])
      }
      Column({ space: 4 }) {
        Text(label).fontSize(15).fontWeight(FontWeight.Bold).fontColor(Color.White)
        Text(done ? 'ثبت شد' : 'برای شروع لمس کنید').fontSize(12)
          .fontColor(done ? '#10b981' : '#9ca3af')
      }.alignItems(HorizontalAlign.Start).layoutWeight(1)
    }
    .width('100%').padding(16).backgroundColor('#1f2937').borderRadius(14).onClick(onTap)
  }

  @Builder WaitingStep() {
    Column({ space: 20 }) {
      Blank()
      if (this.enrollStatus === 'approved') {
        SymbolGlyph($r('sys.symbol.checkmark_seal_fill')).fontSize(80).fontColor(['#10b981'])
        Text('ثبت‌نام تأیید شد').fontSize(22).fontWeight(FontWeight.Bold).fontColor(Color.White)
      } else if (this.enrollStatus === 'rejected') {
        SymbolGlyph($r('sys.symbol.xmark_seal_fill')).fontSize(80).fontColor(['#ef4444'])
        Text('ثبت‌نام رد شد').fontSize(22).fontWeight(FontWeight.Bold).fontColor(Color.White)
      } else {
        LoadingProgress().color('#1a56db').width(64).height(64)
        Text('در حال بررسی…').fontSize(18).fontColor(Color.White)
        Text(`شناسه: ${this.enrollmentId}`).fontSize(12).fontColor('#6b7280')
      }
      Blank()
    }
    .width('100%').alignItems(HorizontalAlign.Center).layoutWeight(1)
  }

  private submitEnrollment(): void {
    this.loading = true
    this.errorMsg = ''
    const client = new GatewayClient()
    const did = AppStorage.getDID()
    client.post<StartEnrollmentResponse>('/v1/enrollment/start', {
      national_id: did,
      did: did,
      pathway: this.pathway,
    }).then((resp: StartEnrollmentResponse) => {
      this.enrollmentId = resp.enrollment_id
      return client.post<EnrollmentStatusResponse>(
        `/v1/enrollment/${resp.enrollment_id}/biometric`, {
          face_image_b64:   btoa(String.fromCharCode(...new Array(512).fill(0xFF))),
          fingerprint_b64:  btoa(String.fromCharCode(...new Array(256).fill(0xAA))),
        }
      )
    }).then((status: EnrollmentStatusResponse) => {
      this.enrollStatus = status.status
      this.step = 3
      this.loading = false
    }).catch((e: Error) => {
      this.errorMsg = e.message ?? 'خطا در ارسال'
      this.loading = false
    })
  }
}
