import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'

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

      const resp = await fetch(`${GATEWAY}/v1/verifier/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
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
      style={{
        minHeight: '100dvh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#111',
        color: '#fff',
        padding: 24,
      }}
    >
      <h1 style={{ marginBottom: 8 }}>پایانه تأیید INDIS</h1>
      <p style={{ color: '#aaa', marginBottom: 24, fontSize: 14 }}>
        کد QR شهروند را اسکن کنید
      </p>

      <div
        id="qr-reader"
        style={{ width: 300, height: 300, borderRadius: 12, overflow: 'hidden' }}
      />

      {!scanning && !error && (
        <p style={{ color: '#aaa', marginTop: 16 }}>در حال راه‌اندازی دوربین…</p>
      )}
      {error && (
        <p style={{ color: '#ff6b6b', marginTop: 16 }}>{error}</p>
      )}
    </div>
  )
}
