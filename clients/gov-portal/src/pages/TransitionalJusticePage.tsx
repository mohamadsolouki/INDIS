import { useState } from 'react'

function bytesInputToBase64(input: string): string {
  const s = input.trim().replace(/\s+/g, '')
  if (!s) throw new Error('bytes input is required')

  const isHex = /^(0x)?[0-9a-fA-F]+$/.test(s) && s.replace(/^0x/, '').length % 2 === 0
  if (isHex) {
    const hex = s.startsWith('0x') ? s.slice(2) : s
    const bytes = new Uint8Array(hex.length / 2)
    for (let i = 0; i < bytes.length; i++) bytes[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16)

    let bin = ''
    for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i])
    return btoa(bin)
  }

  const normalized = s.replace(/-/g, '+').replace(/_/g, '/')
  const padLen = normalized.length % 4
  const padded = padLen ? normalized + '='.repeat(4 - padLen) : normalized

  atob(padded)
  return padded
}

export default function TransitionalJusticePage() {
  const token = localStorage.getItem('gov_token')

  // Submit testimony
  const [zkCitizenshipProof, setZkCitizenshipProof] = useState('')
  const [encryptedTestimony, setEncryptedTestimony] = useState('')
  const [category, setCategory] = useState('')
  const [locale, setLocale] = useState('fa')
  const [submitResult, setSubmitResult] = useState('')

  // Link testimony
  const [receiptToken, setReceiptToken] = useState('')
  const [linkedEncryptedTestimony, setLinkedEncryptedTestimony] = useState('')
  const [linkLocale, setLinkLocale] = useState('fa')
  const [linkResult, setLinkResult] = useState('')

  // Amnesty
  const [applicantDid, setApplicantDid] = useState('')
  const [encryptedDeclaration, setEncryptedDeclaration] = useState('')
  const [amnestyCategory, setAmnestyCategory] = useState('')
  const [amnestyResult, setAmnestyResult] = useState('')

  // Case status
  const [caseId, setCaseId] = useState('')
  const [caseStatusResult, setCaseStatusResult] = useState('')

  async function submitTestimony() {
    setSubmitResult('')
    try {
      const resp = await fetch('/v1/justice/testimony', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          zk_citizenship_proof: bytesInputToBase64(zkCitizenshipProof),
          encrypted_testimony: bytesInputToBase64(encryptedTestimony),
          category,
          locale,
        }),
      })
      const data = await resp.json()
      setSubmitResult(resp.ok ? JSON.stringify(data, null, 2) : `HTTP ${resp.status}: ${JSON.stringify(data)}`)
    } catch (e) {
      setSubmitResult(String(e))
    }
  }

  async function linkTestimony() {
    setLinkResult('')
    try {
      const resp = await fetch('/v1/justice/testimony/link', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          receipt_token: receiptToken,
          encrypted_testimony: bytesInputToBase64(linkedEncryptedTestimony),
          locale: linkLocale,
        }),
      })
      const data = await resp.json()
      setLinkResult(resp.ok ? JSON.stringify(data, null, 2) : `HTTP ${resp.status}: ${JSON.stringify(data)}`)
    } catch (e) {
      setLinkResult(String(e))
    }
  }

  async function initiateAmnesty() {
    setAmnestyResult('')
    try {
      const resp = await fetch('/v1/justice/amnesty', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          applicant_did: applicantDid,
          encrypted_declaration: bytesInputToBase64(encryptedDeclaration),
          category: amnestyCategory,
        }),
      })
      const data = await resp.json()
      setAmnestyResult(resp.ok ? JSON.stringify(data, null, 2) : `HTTP ${resp.status}: ${JSON.stringify(data)}`)
    } catch (e) {
      setAmnestyResult(String(e))
    }
  }

  async function getCaseStatus() {
    setCaseStatusResult('')
    const resp = await fetch(`/v1/justice/cases/${caseId}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    const data = await resp.json()
    setCaseStatusResult(resp.ok ? JSON.stringify(data, null, 2) : `HTTP ${resp.status}: ${JSON.stringify(data)}`)
  }

  return (
    <div>
      <h1 style={{ fontSize: 24, marginBottom: 24 }}>عدالت انتقالی (FR-011)</h1>

      <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)', marginBottom: 16 }}>
        <h2 style={{ fontSize: 16, marginBottom: 12 }}>ثبت شهادت (authenticated)</h2>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <label style={{ gridColumn: '1 / -1' }}>
            zk_citizenship_proof (hex یا base64)
            <textarea value={zkCitizenshipProof} onChange={e => setZkCitizenshipProof(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, minHeight: 120, fontFamily: 'monospace' }} />
          </label>
          <label style={{ gridColumn: '1 / -1' }}>
            encrypted_testimony (hex یا base64)
            <textarea value={encryptedTestimony} onChange={e => setEncryptedTestimony(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, minHeight: 120, fontFamily: 'monospace' }} />
          </label>
          <label>
            category
            <input value={category} onChange={e => setCategory(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8 }} />
          </label>
          <label>
            locale
            <input value={locale} onChange={e => setLocale(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8 }} />
          </label>
        </div>

        <div style={{ marginTop: 12 }}>
          <button onClick={submitTestimony} style={{ background: '#1a56db', color: '#fff', border: 'none', borderRadius: 8, padding: '10px 14px', cursor: 'pointer' }}>
            ارسال
          </button>
        </div>
        {submitResult && <pre style={{ marginTop: 12, background: '#f8fafc', padding: 12, borderRadius: 8, overflow: 'auto' }}>{submitResult}</pre>}
      </div>

      <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)', marginBottom: 16 }}>
        <h2 style={{ fontSize: 16, marginBottom: 12 }}>لینک شهادت (authenticated)</h2>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <label style={{ gridColumn: '1 / -1' }}>
            receipt_token
            <input value={receiptToken} onChange={e => setReceiptToken(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
          <label style={{ gridColumn: '1 / -1' }}>
            encrypted_testimony (hex یا base64)
            <textarea value={linkedEncryptedTestimony} onChange={e => setLinkedEncryptedTestimony(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, minHeight: 120, fontFamily: 'monospace' }} />
          </label>
          <label>
            locale
            <input value={linkLocale} onChange={e => setLinkLocale(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8 }} />
          </label>
          <div />
        </div>

        <div style={{ marginTop: 12 }}>
          <button onClick={linkTestimony} style={{ background: '#0f9960', color: '#fff', border: 'none', borderRadius: 8, padding: '10px 14px', cursor: 'pointer' }}>
            لینک
          </button>
        </div>
        {linkResult && <pre style={{ marginTop: 12, background: '#f8fafc', padding: 12, borderRadius: 8, overflow: 'auto' }}>{linkResult}</pre>}
      </div>

      <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)', marginBottom: 16 }}>
        <h2 style={{ fontSize: 16, marginBottom: 12 }}>شروع عفو (authenticated)</h2>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <label style={{ gridColumn: '1 / -1' }}>
            applicant_did
            <input value={applicantDid} onChange={e => setApplicantDid(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
          <label style={{ gridColumn: '1 / -1' }}>
            encrypted_declaration (hex یا base64)
            <textarea value={encryptedDeclaration} onChange={e => setEncryptedDeclaration(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, minHeight: 120, fontFamily: 'monospace' }} />
          </label>
          <label>
            category
            <input value={amnestyCategory} onChange={e => setAmnestyCategory(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8 }} />
          </label>
          <div />
        </div>

        <div style={{ marginTop: 12 }}>
          <button onClick={initiateAmnesty} style={{ background: '#1a56db', color: '#fff', border: 'none', borderRadius: 8, padding: '10px 14px', cursor: 'pointer' }}>
            شروع
          </button>
        </div>
        {amnestyResult && <pre style={{ marginTop: 12, background: '#f8fafc', padding: 12, borderRadius: 8, overflow: 'auto' }}>{amnestyResult}</pre>}
      </div>

      <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
        <h2 style={{ fontSize: 16, marginBottom: 12 }}>وضعیت پرونده</h2>

        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
          <label style={{ flex: 1 }}>
            case_id
            <input value={caseId} onChange={e => setCaseId(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
          <button onClick={getCaseStatus} style={{ background: '#1a56db', color: '#fff', border: 'none', borderRadius: 8, padding: '10px 14px', cursor: 'pointer', height: 42 }}>
            دریافت
          </button>
        </div>
        {caseStatusResult && <pre style={{ marginTop: 12, background: '#f8fafc', padding: 12, borderRadius: 8, overflow: 'auto' }}>{caseStatusResult}</pre>}
      </div>
    </div>
  )
}

