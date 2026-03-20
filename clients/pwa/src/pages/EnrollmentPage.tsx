import { useState, useRef, FormEvent } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../hooks/useAuth'

type Step = 'info' | 'document' | 'biometric' | 'submitted'

/**
 * Multi-step enrollment flow.
 *
 * Step 1: Personal information
 * Step 2: Document capture (camera or file upload)
 * Step 3: Biometric consent  (camera capture placeholder — WebRTC getUserMedia)
 * Step 4: Submission confirmation
 *
 * PRD FR-002 (standard enrollment), FR-003 (enhanced enrollment).
 */
export default function EnrollmentPage() {
  const { token } = useAuth()
  const [step, setStep] = useState<Step>('info')
  const [name, setName] = useState('')
  const [nationalId, setNationalId] = useState('')
  const [docFile, setDocFile] = useState<File | null>(null)
  const [cameraActive, setCameraActive] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const videoRef = useRef<HTMLVideoElement>(null)

  async function startCamera() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'user' } })
      if (videoRef.current) {
        videoRef.current.srcObject = stream
        setCameraActive(true)
      }
    } catch {
      setError('دسترسی به دوربین رد شد.')
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await api.post(
        '/enrollment',
        {
          full_name: name,
          national_id: nationalId,
          pathway: 'standard',
        },
        token ?? undefined,
      )
      setStep('submitted')
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  if (step === 'submitted') {
    return (
      <div style={{ padding: 20 }}>
        <div className="card text-center" style={{ marginTop: 40 }}>
          <p style={{ fontSize: 48, marginBottom: 12 }}>✅</p>
          <h2>درخواست ثبت شد</h2>
          <p className="text-muted" style={{ marginTop: 8 }}>
            درخواست ثبت‌نام شما ارسال شد. پس از بررسی، اعتبارنامه به کیف‌پول شما اضافه می‌شود.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div style={{ padding: 20 }}>
      <h2 style={{ marginBottom: 8 }}>ثبت‌نام</h2>
      <StepIndicator current={step} />

      {step === 'info' && (
        <form className="flex-col gap-4 mt-4" onSubmit={e => { e.preventDefault(); setStep('document') }}>
          <Field label="نام و نام خانوادگی" value={name} onChange={setName} placeholder="نام کامل" />
          <Field label="کد ملی" value={nationalId} onChange={setNationalId} placeholder="۱۰ رقم" dir="ltr" />
          <button type="submit" className="btn-primary" disabled={!name || !nationalId}>
            مرحله بعد
          </button>
        </form>
      )}

      {step === 'document' && (
        <div className="flex-col gap-4 mt-4">
          <p className="text-muted">تصویر کارت ملی یا سند هویتی خود را آپلود کنید.</p>
          <input
            type="file"
            accept="image/*"
            onChange={e => setDocFile(e.target.files?.[0] ?? null)}
          />
          {docFile && <p style={{ fontSize: 12, color: 'var(--color-success)' }}>✓ {docFile.name}</p>}
          <button
            className="btn-primary"
            disabled={!docFile}
            onClick={() => setStep('biometric')}
          >
            مرحله بعد
          </button>
          <button className="btn-ghost" onClick={() => setStep('info')}>
            بازگشت
          </button>
        </div>
      )}

      {step === 'biometric' && (
        <div className="flex-col gap-4 mt-4">
          <p className="text-muted">
            برای تأیید هویت بیومتریک، لطفاً مستقیم به دوربین نگاه کنید.
          </p>

          {!cameraActive ? (
            <button className="btn-primary" onClick={startCamera}>
              فعال‌سازی دوربین
            </button>
          ) : (
            <video
              ref={videoRef}
              autoPlay
              playsInline
              muted
              style={{ width: '100%', borderRadius: 8, background: '#000' }}
            />
          )}

          {error && <p style={{ color: 'var(--color-error)', fontSize: 13 }}>{error}</p>}

          <button
            className="btn-primary"
            disabled={loading}
            onClick={handleSubmit as unknown as React.MouseEventHandler}
          >
            {loading ? 'در حال ارسال…' : 'ارسال درخواست'}
          </button>
          <button className="btn-ghost" onClick={() => setStep('document')}>
            بازگشت
          </button>
        </div>
      )}
    </div>
  )
}

function Field({
  label,
  value,
  onChange,
  placeholder,
  dir = 'rtl',
}: {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  dir?: 'rtl' | 'ltr'
}) {
  return (
    <div>
      <label style={{ display: 'block', marginBottom: 6, fontSize: 14 }}>{label}</label>
      <input
        type="text"
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        dir={dir}
        required
        style={{
          width: '100%',
          padding: '10px 14px',
          border: '1px solid var(--color-border)',
          borderRadius: 8,
          fontSize: 14,
        }}
      />
    </div>
  )
}

function StepIndicator({ current }: { current: Step }) {
  const steps: { id: Step; label: string }[] = [
    { id: 'info', label: 'اطلاعات' },
    { id: 'document', label: 'مدارک' },
    { id: 'biometric', label: 'بیومتریک' },
  ]
  const idx = steps.findIndex(s => s.id === current)

  return (
    <div style={{ display: 'flex', gap: 8, marginBottom: 20, marginTop: 8 }}>
      {steps.map((s, i) => (
        <div key={s.id} style={{ flex: 1, textAlign: 'center' }}>
          <div
            style={{
              height: 4,
              borderRadius: 2,
              background: i <= idx ? 'var(--color-primary)' : 'var(--color-border)',
              marginBottom: 4,
            }}
          />
          <span style={{ fontSize: 10, color: i <= idx ? 'var(--color-primary)' : 'var(--color-text-muted)' }}>
            {s.label}
          </span>
        </div>
      ))}
    </div>
  )
}
