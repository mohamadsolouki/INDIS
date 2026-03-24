import { useEffect, useRef, useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import FeedbackState from '../components/FeedbackState'

const VERIFIER_ID = import.meta.env.VITE_VERIFIER_ID ?? 'dev-verifier'
const GATEWAY = import.meta.env.VITE_GATEWAY_URL ?? 'http://localhost:8080'

/**
 * QR scanner page for the verifier terminal.
 *
 * Uses html5-qrcode to activate the device camera, decodes the citizen's
 * ZK-proof QR, then POSTs to the INDIS gateway /v1/verifier/verify and
 * redirects to ResultPage with the boolean outcome.
 *
 * PRD FR-013: verifiers see ONLY a green/red result — never citizen data.
 */
export default function ScanPage() {
  const navigate = useNavigate()
  const scannerRef = useRef<unknown>(null)
  const [scanning, setScanning] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let stopped = false

    async function startScan() {
      // Dynamically import to avoid SSR issues.
      const { Html5Qrcode } = await import('html5-qrcode')
      const scanner = new Html5Qrcode('qr-reader')
      scannerRef.current = scanner
      setScanning(true)

      await scanner.start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: { width: 250, height: 250 } },
        async (decodedText) => {
          if (stopped) return
          stopped = true
          await scanner.stop()
          setScanning(false)
          await verifyProof(decodedText)
        },
        () => { /* ignore per-frame errors */ },
      )
    }

    startScan().catch(err => setError(String(err)))

    return () => {
      stopped = true
      if (scannerRef.current) {
        (scannerRef.current as { stop: () => Promise<void> }).stop().catch(() => {})
      }
    }
  }, [])

  async function verifyProof(qrPayload: string) {
    try {
      const parsed = JSON.parse(qrPayload) as {
        proof: string
        nonce: string
        predicate: string
      }

      const token = localStorage.getItem('verifier_token') ?? ''
      const resp = await fetch(`${GATEWAY}/v1/verifier/verify`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          verifier_id: VERIFIER_ID,
          proof_b64: parsed.proof,
          nonce: parsed.nonce,
          predicate: parsed.predicate,
          credential_type: 'national_id',
          proof_system: 'groth16',
          public_inputs_b64: '',
        }),
      })

      const data = await resp.json() as { valid: boolean }
      navigate('/result', { state: { valid: data.valid } })
    } catch (err) {
      navigate('/result', { state: { valid: false, error: String(err) } })
    }
  }

  return (
    <div
      className="verifier-screen"
      style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%', maxWidth: 340, marginBottom: 8 }}>
        <h1 style={{ margin: 0 }}>پایانه تأیید INDIS</h1>
        <div style={{ display: 'flex', gap: 8 }}>
          <Link
            to="/history"
            style={{ fontSize: 12, color: '#aaa', padding: '6px 10px', borderRadius: 6, border: '1px solid #333', background: 'transparent' }}
          >
            تاریخچه
          </Link>
          <button
            onClick={() => { localStorage.removeItem('verifier_id'); window.location.href = '/login' }}
            className="verifier-btn"
            style={{ fontSize: 12 }}
          >
            خروج
          </button>
        </div>
      </div>
      <p style={{ color: '#aaa', marginBottom: 24, fontSize: 14 }}>
        کد QR شهروند را اسکن کنید
      </p>

      <div
        id="qr-reader"
        style={{ width: 300, height: 300, borderRadius: 12, overflow: 'hidden' }}
      />

      {!scanning && !error && <FeedbackState kind="loading" title="راه‌اندازی دوربین" message="چند لحظه صبر کنید تا اسکنر فعال شود." />}
      {error && (
        <FeedbackState kind="error" title="اسکنر در دسترس نیست" message={error} />
      )}
    </div>
  )
}
