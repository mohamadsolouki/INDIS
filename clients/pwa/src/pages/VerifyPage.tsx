import { useState } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../hooks/useAuth'
import { useCredentials } from '../hooks/useCredentials'

/**
 * ZK-proof-based verification page.
 *
 * The citizen selects which credential attribute to prove (e.g., "age ≥ 18"),
 * the service generates a ZK proof locally (TODO: WASM bridge to zkproof Rust
 * crate), and the result is a QR code that the verifier terminal can scan.
 *
 * PRD FR-007 (ZK proof generation), FR-008 (selective disclosure).
 */
export default function VerifyPage() {
  const { token } = useAuth()
  const { credentials } = useCredentials()
  const [selectedCredId, setSelectedCredId] = useState('')
  const [predicate, setPredicate] = useState('age_gte_18')
  const [qrData, setQrData] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function generateProof() {
    if (!selectedCredId) return
    setLoading(true)
    setError('')
    try {
      const resp = await api.post<{ proof_b64: string; nonce: string }>(
        '/credentials/prove',
        { credential_id: selectedCredId, predicate },
        token ?? undefined,
      )
      // Encode proof + nonce as QR payload for verifier terminal to scan.
      const payload = JSON.stringify({ proof: resp.proof_b64, nonce: resp.nonce, predicate })
      setQrData(payload)
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ padding: 20 }}>
      <h2 style={{ marginBottom: 20 }}>اثبات هویت</h2>

      <div className="card" style={{ marginBottom: 16 }}>
        <p style={{ fontSize: 13, color: 'var(--color-text-muted)', marginBottom: 12 }}>
          با استفاده از اثبات‌های ZK، فقط یک ادعای بولی (درست/نادرست) به تأییدکننده
          نشان داده می‌شود — هیچ اطلاعات شخصی افشا نمی‌شود.
        </p>

        <label style={{ display: 'block', marginBottom: 6, fontSize: 14 }}>
          اعتبارنامه
        </label>
        <select
          value={selectedCredId}
          onChange={e => setSelectedCredId(e.target.value)}
          style={{
            width: '100%',
            padding: '10px 14px',
            border: '1px solid var(--color-border)',
            borderRadius: 8,
            fontSize: 14,
            marginBottom: 12,
          }}
        >
          <option value="">انتخاب کنید…</option>
          {credentials.map(c => (
            <option key={c.id} value={c.id}>{c.type}</option>
          ))}
        </select>

        <label style={{ display: 'block', marginBottom: 6, fontSize: 14 }}>
          ادعا
        </label>
        <select
          value={predicate}
          onChange={e => setPredicate(e.target.value)}
          style={{
            width: '100%',
            padding: '10px 14px',
            border: '1px solid var(--color-border)',
            borderRadius: 8,
            fontSize: 14,
            marginBottom: 16,
          }}
        >
          <option value="age_gte_18">سن ≥ ۱۸ سال</option>
          <option value="citizen">تابعیت ایران</option>
          <option value="voter_eligible">واجد شرایط رأی‌گیری</option>
          <option value="credential_valid">اعتبارنامه معتبر است</option>
        </select>

        <button
          className="btn-primary"
          onClick={generateProof}
          disabled={!selectedCredId || loading}
        >
          {loading ? 'در حال تولید اثبات…' : 'تولید اثبات ZK'}
        </button>
      </div>

      {error && (
        <div className="card" style={{ color: 'var(--color-error)', marginBottom: 16 }}>
          {error}
        </div>
      )}

      {qrData && (
        <div className="card text-center">
          <p style={{ fontWeight: 600, marginBottom: 12 }}>کد QR برای تأییدکننده</p>
          {/* QR rendering via CSS trick — replace with qrcode.react in production */}
          <div
            style={{
              width: 200,
              height: 200,
              margin: '0 auto',
              background: '#f4f4f4',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: 8,
              fontSize: 11,
              wordBreak: 'break-all',
              padding: 8,
              color: '#666',
            }}
          >
            [QR — install qrcode.react for real rendering]
          </div>
          <p className="text-muted" style={{ marginTop: 8, fontSize: 12 }}>
            این کد را به تأییدکننده نشان دهید.
          </p>
        </div>
      )}
    </div>
  )
}
