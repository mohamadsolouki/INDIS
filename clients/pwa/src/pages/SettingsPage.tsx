import { useAuth } from '../hooks/useAuth'
import { wallet } from '../lib/wallet'
import { useState } from 'react'

export default function SettingsPage() {
  const { did, logout } = useAuth()
  const [clearing, setClearing] = useState(false)
  const [cleared, setCleared] = useState(false)

  async function clearWallet() {
    if (!confirm('آیا مطمئن هستید؟ تمام اعتبارنامه‌های محلی حذف می‌شوند.')) return
    setClearing(true)
    const creds = await wallet.list()
    for (const c of creds) await wallet.delete(c.id)
    setCleared(true)
    setClearing(false)
  }

  return (
    <div style={{ padding: 20 }}>
      <h2 style={{ marginBottom: 20 }}>تنظیمات</h2>

      <div className="card" style={{ marginBottom: 12 }}>
        <h3 style={{ fontSize: 15, marginBottom: 8 }}>شناسه دیجیتال</h3>
        <p dir="ltr" style={{ fontSize: 12, wordBreak: 'break-all', color: 'var(--color-text-muted)' }}>
          {did ?? '—'}
        </p>
      </div>

      <div className="card" style={{ marginBottom: 12 }}>
        <h3 style={{ fontSize: 15, marginBottom: 8 }}>حریم خصوصی</h3>
        <p className="text-muted" style={{ fontSize: 13, marginBottom: 12 }}>
          اطلاعات شما به‌صورت رمزگذاری‌شده در دستگاه ذخیره می‌شود.
          هیچ‌گاه اطلاعات هویتی خام با تأییدکننده به اشتراک گذاشته نمی‌شود.
        </p>
        <button
          className="btn-ghost"
          style={{ color: 'var(--color-error)', borderColor: 'var(--color-error)' }}
          onClick={clearWallet}
          disabled={clearing}
        >
          {clearing ? 'در حال پاک‌سازی…' : 'پاک‌سازی اعتبارنامه‌های محلی'}
        </button>
        {cleared && (
          <p style={{ color: 'var(--color-success)', fontSize: 12, marginTop: 8 }}>
            ✓ اعتبارنامه‌های محلی پاک شدند.
          </p>
        )}
      </div>

      <div className="card">
        <h3 style={{ fontSize: 15, marginBottom: 8 }}>خروج</h3>
        <button className="btn-primary" style={{ background: 'var(--color-error)' }} onClick={logout}>
          خروج از حساب
        </button>
      </div>
    </div>
  )
}
