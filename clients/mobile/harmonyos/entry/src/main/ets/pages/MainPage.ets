import router from '@ohos.router'
import { AppStorage } from '../model/AppStorage'

/**
 * Main page — bottom tab bar with 4 tabs.
 * Mirrors Android MainActivity and iOS MainTabView.
 *
 * Tabs: Home | Wallet | Verify | Settings
 */
@Entry
@Component
struct MainPage {

  @State private selectedTab: number = 0

  build() {
    Tabs({ barPosition: BarPosition.End, index: this.selectedTab }) {

      // ── Home ───────────────────────────────────────────────────────────
      TabContent() {
        this.HomeContent()
      }
      .tabBar(this.TabBarItem(0, $r('sys.symbol.house_fill'), 'خانه'))

      // ── Wallet ─────────────────────────────────────────────────────────
      TabContent() {
        Column()
          .width('100%').height('100%').backgroundColor('#0f1420')
        // WalletPage is loaded as a sub-component to keep routing simple
        WalletInline()
      }
      .tabBar(this.TabBarItem(1, $r('sys.symbol.wallet_pass_fill'), 'کیف پول'))

      // ── Verify ─────────────────────────────────────────────────────────
      TabContent() {
        Column()
          .width('100%').height('100%').backgroundColor('#0f1420')
        VerifyInline()
      }
      .tabBar(this.TabBarItem(2, $r('sys.symbol.qrcode_viewfinder'), 'تأیید'))

      // ── Settings ───────────────────────────────────────────────────────
      TabContent() {
        Column()
          .width('100%').height('100%').backgroundColor('#0f1420')
        SettingsInline()
      }
      .tabBar(this.TabBarItem(3, $r('sys.symbol.gearshape_fill'), 'تنظیمات'))
    }
    .width('100%').height('100%')
    .barHeight(56)
    .backgroundColor('#111827')
    .onChange((index: number) => { this.selectedTab = index })
  }

  @Builder HomeContent() {
    Column({ space: 20 }) {
      // DID card
      Column({ space: 8 }) {
        Text('هویت دیجیتال شما')
          .fontSize(12).fontColor('#9ca3af')
        Text(AppStorage.getDID())
          .fontSize(11).fontColor(Color.White)
          .fontFamily('monospace').maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .width('92%')
      .padding(16)
      .backgroundColor('#1f2937')
      .borderRadius(14)

      // Quick actions grid
      Grid() {
        GridItem() { this.QuickCard($r('sys.symbol.plus_circle_fill'), 'ثبت‌نام', '#1a56db') }
        GridItem() { this.QuickCard($r('sys.symbol.wallet_pass_fill'), 'کیف پول', '#059669') }
        GridItem() { this.QuickCard($r('sys.symbol.qrcode_viewfinder'), 'ارائه مدرک', '#d97706') }
        GridItem() { this.QuickCard($r('sys.symbol.lock_shield_fill'), 'حریم خصوصی', '#7c3aed') }
      }
      .columnsTemplate('1fr 1fr')
      .rowsGap(12).columnsGap(12)
      .width('92%')
    }
    .width('100%').height('100%')
    .backgroundColor('#0f1420')
    .padding({ top: 24 })
    .alignItems(HorizontalAlign.Center)
  }

  @Builder QuickCard(icon: Resource, label: string, color: string) {
    Column({ space: 10 }) {
      SymbolGlyph(icon).fontSize(32).fontColor([color])
      Text(label).fontSize(14).fontWeight(FontWeight.Bold).fontColor(Color.White)
    }
    .width('100%').padding(20)
    .backgroundColor('#1f2937').borderRadius(16)
    .alignItems(HorizontalAlign.Center)
  }

  @Builder TabBarItem(index: number, icon: Resource, label: string) {
    Column({ space: 4 }) {
      SymbolGlyph(icon)
        .fontSize(22)
        .fontColor([this.selectedTab === index ? '#1a56db' : '#6b7280'])
      Text(label)
        .fontSize(11)
        .fontColor(this.selectedTab === index ? '#1a56db' : '#6b7280')
    }
    .height('100%')
    .justifyContent(FlexAlign.Center)
  }
}

// ── Inline sub-components ──────────────────────────────────────────────────
// These are embedded in tabs to avoid full page navigation for tab switching.

@Component
struct WalletInline {
  build() {
    Column() {
      Text('کیف پول').fontSize(24).fontWeight(FontWeight.Bold).fontColor(Color.White)
        .padding({ top: 24, bottom: 8, left: 20 }).width('100%')
      // Full wallet list rendered in WalletPage when navigated directly
      Text('برای مشاهده مدارک روی «کیف پول» در نوار پایین ضربه بزنید.')
        .fontSize(14).fontColor('#6b7280').textAlign(TextAlign.Center)
        .padding(24)
    }
    .width('100%').height('100%').backgroundColor('#0f1420')
  }
}

@Component
struct VerifyInline {
  build() {
    Column() {
      Text('تأیید هویت').fontSize(24).fontWeight(FontWeight.Bold).fontColor(Color.White)
        .padding({ top: 24, bottom: 8, left: 20 }).width('100%')
      Text('برای اسکن QR روی «تأیید» در نوار پایین ضربه بزنید.')
        .fontSize(14).fontColor('#6b7280').textAlign(TextAlign.Center).padding(24)
    }
    .width('100%').height('100%').backgroundColor('#0f1420')
  }
}

@Component
struct SettingsInline {
  build() {
    Column() {
      Text('تنظیمات').fontSize(24).fontWeight(FontWeight.Bold).fontColor(Color.White)
        .padding({ top: 24, bottom: 8, left: 20 }).width('100%')
    }
    .width('100%').height('100%').backgroundColor('#0f1420')
  }
}
