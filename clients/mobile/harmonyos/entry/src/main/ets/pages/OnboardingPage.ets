import router from '@ohos.router'
import { AppStorage } from '../model/AppStorage'
import { GatewayClient } from '../network/GatewayClient'

interface RegisterResponse {
  did: string
  token: string
}

interface OnboardSlide {
  icon: Resource
  title: string
  body: string
}

/**
 * Onboarding screen — three introduction slides + national ID registration.
 * On successful registration navigates to MainPage.
 * PRD FR-001: DID is generated on-device; national ID is sent only to register.
 */
@Entry
@Component
struct OnboardingPage {

  @State private currentSlide: number = 0
  @State private showRegister: boolean = false
  @State private nationalId: string = ''
  @State private loading: boolean = false
  @State private errorMsg: string = ''

  private slides: OnboardSlide[] = [
    { icon: $r('sys.symbol.shield_lefthalf_filled'), title: 'هویت دیجیتال ایرانی',   body: 'کارت هویت ملی دیجیتال امن با رمزنگاری پیشرفته' },
    { icon: $r('sys.symbol.lock_icloud'),            title: 'کلید در دست شما',       body: 'کلید خصوصی روی دستگاه شما می‌ماند — هیچ‌وقت منتقل نمی‌شود' },
    { icon: $r('sys.symbol.checkmark_seal_fill'),    title: 'اثبات بدون افشا',       body: 'اثبات هویت با ZK-Proof — بدون نمایش اطلاعات خصوصی' },
  ]

  build() {
    Stack() {
      // Background
      Column()
        .width('100%').height('100%')
        .backgroundColor('#111827')

      // Slide content + controls
      Column({ space: 0 }) {
        // Slide body
        Column({ space: 24 }) {
          Blank()
          SymbolGlyph(this.slides[this.currentSlide].icon)
            .fontSize(80)
            .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          Text(this.slides[this.currentSlide].title)
            .fontSize(22).fontWeight(FontWeight.Bold)
            .fontColor(Color.White).textAlign(TextAlign.Center)
          Text(this.slides[this.currentSlide].body)
            .fontSize(15).fontColor('#B0B8C8').textAlign(TextAlign.Center)
            .padding({ left: 32, right: 32 })
          Blank()
        }
        .width('100%').layoutWeight(1)

        // Dots
        Row({ space: 8 }) {
          ForEach(this.slides, (_: OnboardSlide, i: number) => {
            Circle({ width: 8, height: 8 })
              .fill(i === this.currentSlide ? '#1a56db' : '#ffffff50')
          })
        }.margin({ bottom: 20 })

        // CTA Button
        Button(this.currentSlide < this.slides.length - 1 ? 'ادامه' : 'شروع ثبت‌نام')
          .width('88%').height(52)
          .backgroundColor('#1a56db').fontColor(Color.White)
          .fontSize(16).fontWeight(FontWeight.Bold)
          .borderRadius(14)
          .onClick(() => {
            if (this.currentSlide < this.slides.length - 1) {
              this.currentSlide++
            } else {
              this.showRegister = true
            }
          })
          .margin({ bottom: 16 })

        if (this.currentSlide === this.slides.length - 1) {
          // Dev bypass
          Button('ورود توسعه‌دهنده')
            .width('88%').height(36)
            .backgroundColor(Color.Transparent).fontColor('#ffffff30')
            .fontSize(12).borderRadius(8)
            .border({ width: 1, color: '#333' })
            .onClick(() => {
              AppStorage.login('did:indis:devdevdevdev', 'dev-token')
              router.replaceUrl({ url: 'pages/MainPage' })
            })
            .margin({ bottom: 32 })
        }
      }
      .width('100%').height('100%')

      // Registration sheet overlay
      if (this.showRegister) {
        this.RegistrationSheet()
      }
    }
    .width('100%').height('100%')
  }

  @Builder RegistrationSheet() {
    Column({ space: 20 }) {
      // Header bar
      Row() {
        Button('انصراف')
          .fontColor(Color.White).backgroundColor(Color.Transparent).fontSize(14)
          .onClick(() => { this.showRegister = false; this.errorMsg = '' })
        Blank()
      }.width('100%').padding({ top: 12, left: 16, right: 16 })

      SymbolGlyph($r('sys.symbol.person_crop_circle_badge_plus'))
        .fontSize(56).fontColor(['#1a56db'])

      Text('ثبت کد ملی').fontSize(22).fontWeight(FontWeight.Bold).fontColor(Color.White)
      Text('کد ملی ۱۰ رقمی خود را وارد کنید')
        .fontSize(14).fontColor('#aaa')

      TextInput({ placeholder: 'کد ملی', text: this.nationalId })
        .width('88%').height(52)
        .backgroundColor('#1f2937').fontColor(Color.White)
        .caretColor('#1a56db').placeholderColor('#aaa')
        .borderRadius(12).fontSize(16)
        .type(InputType.Number)
        .onChange((v: string) => { this.nationalId = v })

      if (this.errorMsg !== '') {
        Text(this.errorMsg).fontSize(13).fontColor('#ef4444').textAlign(TextAlign.Center)
          .padding({ left: 24, right: 24 })
      }

      Button(this.loading ? 'در حال ثبت‌نام…' : 'ثبت‌نام')
        .width('88%').height(52)
        .backgroundColor(this.nationalId.length >= 10 && !this.loading ? '#1a56db' : '#6b7280')
        .fontColor(Color.White).fontSize(16).fontWeight(FontWeight.Bold).borderRadius(14)
        .enabled(this.nationalId.length >= 10 && !this.loading)
        .onClick(() => { this.register() })

      Blank()
    }
    .width('100%').height('100%')
    .backgroundColor('#111827')
  }

  private register(): void {
    this.loading = true
    this.errorMsg = ''
    const client = new GatewayClient()
    // On HarmonyOS derive a mock DID (Secure Element integration is platform-specific)
    const did = 'did:indis:' + Math.random().toString(16).slice(2, 22)
    client.post<RegisterResponse>('/v1/identity/register', {
      national_id: this.nationalId,
      did: did,
    }).then((resp: RegisterResponse) => {
      AppStorage.login(resp.did, resp.token)
      router.replaceUrl({ url: 'pages/MainPage' })
    }).catch((e: Error) => {
      this.errorMsg = e.message ?? 'خطا در ثبت‌نام'
      this.loading = false
    })
  }
}
