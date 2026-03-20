import router from '@ohos.router'
import { AppStorage } from '../model/AppStorage'
import promptAction from '@ohos.promptAction'

/**
 * Settings page — mirrors Android SettingsActivity and iOS SettingsView.
 *
 * Sections: Language, Display (Persian numerals), Network, Privacy, About, Logout.
 */
@Entry
@Component
struct SettingsPage {

  @State private locale: string = AppStorage.getLocale()
  @State private usePersian: boolean = AppStorage.usePersianNumerals()
  @State private gatewayURL: string = AppStorage.getGatewayURL()
  @State private editingURL: boolean = false
  @State private draftURL: string = ''

  private readonly locales: Array<{ code: string; label: string }> = [
    { code: 'fa',  label: 'فارسی' },
    { code: 'en',  label: 'English' },
    { code: 'ckb', label: 'کوردی سۆرانی' },
    { code: 'kmr', label: 'Kurdî Kurmancî' },
    { code: 'ar',  label: 'العربية' },
    { code: 'az',  label: 'Azərbaycanca' },
  ]

  build() {
    Column() {
      // Header
      Row() {
        Button() {
          SymbolGlyph($r('sys.symbol.chevron_right')).fontSize(20).fontColor(Color.White)
        }.backgroundColor(Color.Transparent).onClick(() => router.back())
        Text('تنظیمات').fontSize(20).fontWeight(FontWeight.Bold).fontColor(Color.White)
          .margin({ right: 32 })
        Blank()
      }.width('100%').padding({ top: 20, left: 8, right: 20, bottom: 4 })

      Scroll() {
        Column({ space: 12 }) {

          // Language
          this.Section('زبان') {
            Select(this.locales.map(l => ({ value: l.label })))
              .selected(this.locales.findIndex(l => l.code === this.locale))
              .onSelect((index: number) => {
                this.locale = this.locales[index].code
                AppStorage.setLocale(this.locale)
              })
              .width('100%').fontColor(Color.White)
              .backgroundColor('#1f2937')
          }

          // Display
          this.Section('نمایش') {
            Row() {
              SymbolGlyph($r('sys.symbol.textformat_123')).fontSize(20).fontColor(['#1a56db'])
              Text('اعداد فارسی').fontSize(14).fontColor(Color.White).layoutWeight(1).margin({ left: 10 })
              Toggle({ type: ToggleType.Switch, isOn: this.usePersian })
                .selectedColor('#1a56db')
                .onChange((v: boolean) => {
                  this.usePersian = v
                  AppStorage.setPersianNumerals(v)
                })
            }.width('100%')
          }

          // Network
          this.Section('شبکه') {
            Column({ space: 8 }) {
              Row() {
                SymbolGlyph($r('sys.symbol.network')).fontSize(20).fontColor(['#1a56db'])
                Text('آدرس Gateway').fontSize(14).fontColor(Color.White).layoutWeight(1).margin({ left: 10 })
              }
              if (this.editingURL) {
                TextInput({ text: this.draftURL, placeholder: 'http://...' })
                  .backgroundColor('#374151').fontColor(Color.White).borderRadius(8)
                  .caretColor('#1a56db').onChange((v: string) => { this.draftURL = v })
                Row({ space: 8 }) {
                  Button('ذخیره').height(36).backgroundColor('#1a56db').fontColor(Color.White)
                    .borderRadius(8).onClick(() => {
                      this.gatewayURL = this.draftURL
                      AppStorage.setGatewayURL(this.draftURL)
                      this.editingURL = false
                    })
                  Button('انصراف').height(36).backgroundColor('#374151').fontColor(Color.White)
                    .borderRadius(8).onClick(() => { this.editingURL = false })
                }
              } else {
                Text(this.gatewayURL).fontSize(12).fontColor('#9ca3af').fontFamily('monospace')
                  .onClick(() => { this.draftURL = this.gatewayURL; this.editingURL = true })
              }
            }.width('100%').alignItems(HorizontalAlign.Start)
          }

          // Privacy
          this.Section('حریم خصوصی') {
            Row() {
              SymbolGlyph($r('sys.symbol.lock_shield')).fontSize(20).fontColor(['#7c3aed'])
              Text('مرکز حریم خصوصی').fontSize(14).fontColor(Color.White).layoutWeight(1).margin({ left: 10 })
              SymbolGlyph($r('sys.symbol.chevron_left')).fontSize(16).fontColor(['#6b7280'])
            }.width('100%').onClick(() => router.pushUrl({ url: 'pages/PrivacyCenterPage' }))
          }

          // About
          this.Section('درباره') {
            Column({ space: 10 }) {
              Row() {
                SymbolGlyph($r('sys.symbol.info_circle')).fontSize(18).fontColor(['#6b7280'])
                Text('نسخه').fontSize(14).fontColor(Color.White).layoutWeight(1).margin({ left: 10 })
                Text('1.0.0').fontSize(13).fontColor('#9ca3af')
              }.width('100%')
              Divider().color('#374151')
              Column({ space: 4 }) {
                Row() {
                  SymbolGlyph($r('sys.symbol.key_fill')).fontSize(18).fontColor(['#6b7280'])
                  Text('شناسه دستگاه (DID)').fontSize(14).fontColor(Color.White).margin({ left: 10 })
                }
                Text(AppStorage.getDID()).fontSize(10).fontColor('#6b7280').fontFamily('monospace')
                  .maxLines(2).textOverflow({ overflow: TextOverflow.Ellipsis })
              }.width('100%').alignItems(HorizontalAlign.Start)
            }.width('100%')
          }

          // Logout
          Row() {
            SymbolGlyph($r('sys.symbol.rectangle_portrait_and_arrow_right'))
              .fontSize(20).fontColor(['#ef4444'])
            Text('خروج از حساب').fontSize(14).fontColor('#ef4444').margin({ left: 10 }).layoutWeight(1)
          }
          .width('88%').padding(16).backgroundColor('#1f2937').borderRadius(14)
          .onClick(() => {
            promptAction.showDialog({
              title: 'خروج از حساب',
              message: 'آیا مطمئن هستید؟',
              buttons: [
                { text: 'خروج', color: '#ef4444' },
                { text: 'انصراف', color: '#9ca3af' },
              ]
            }).then((result: promptAction.ShowDialogSuccessResponse) => {
              if (result.index === 0) {
                AppStorage.logout()
                router.replaceUrl({ url: 'pages/OnboardingPage' })
              }
            })
          })

          Blank().height(40)
        }
        .width('100%').alignItems(HorizontalAlign.Center).padding({ top: 16 })
      }
      .layoutWeight(1)
    }
    .width('100%').height('100%').backgroundColor('#0f1420')
  }

  @Builder Section(title: string, @BuilderParam content: () => void) {
    Column({ space: 10 }) {
      Text(title).fontSize(12).fontColor('#6b7280').alignSelf(ItemAlign.Start)
      Column() { content() }
        .width('100%').padding(16).backgroundColor('#1f2937').borderRadius(14)
    }.width('88%')
  }
}
