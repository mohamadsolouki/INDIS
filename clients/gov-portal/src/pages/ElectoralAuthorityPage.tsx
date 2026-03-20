import { useState } from 'react'

function bytesInputToBase64(input: string): string {
  const s = input.trim().replace(/\s+/g, '')
  if (!s) throw new Error('bytes input is required')

  const isHex = /^(0x)?[0-9a-fA-F]+$/.test(s) && s.replace(/^0x/, '').length % 2 === 0
  if (isHex) {
    const hex = s.startsWith('0x') ? s.slice(2) : s
    const bytes = new Uint8Array(hex.length / 2)
    for (let i = 0; i < bytes.length; i++) bytes[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16)

    // Dev-only: convert bytes → binary string → base64
    let bin = ''
    for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i])
    return btoa(bin)
  }

  // Treat as base64 (accept base64url) and normalize to standard base64.
  const normalized = s.replace(/-/g, '+').replace(/_/g, '/')
  const padLen = normalized.length % 4
  const padded = padLen ? normalized + '='.repeat(4 - padLen) : normalized

  // Validate base64 content early.
  atob(padded)
  return padded
}

export default function ElectoralAuthorityPage() {
  const token = localStorage.getItem('gov_token')

  const [name, setName] = useState('')
  const [opensAt, setOpensAt] = useState('')
  const [closesAt, setClosesAt] = useState('')
  const [adminDid, setAdminDid] = useState('')
  const [registerResult, setRegisterResult] = useState<string>('')

  const [electionId, setElectionId] = useState('')
  const [nullifierHash, setNullifierHash] = useState('')
  const [encryptedVote, setEncryptedVote] = useState('')
  const [zkProof, setZkProof] = useState('')
  const [ballotResult, setBallotResult] = useState<string>('')

  const [statusElectionId, setStatusElectionId] = useState('')
  const [statusResult, setStatusResult] = useState<string>('')

  async function registerElection() {
    setRegisterResult('')
    const resp = await fetch('/v1/electoral/elections', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ name, opens_at: opensAt, closes_at: closesAt, admin_did: adminDid }),
    })
    const data = await resp.json()
    setRegisterResult(resp.ok ? JSON.stringify(data, null, 2) : `HTTP ${resp.status}: ${JSON.stringify(data)}`)
  }

  async function castBallot() {
    setBallotResult('')
    try {
      const body = {
        election_id: electionId,
        nullifier_hash: nullifierHash,
        encrypted_vote: bytesInputToBase64(encryptedVote),
        zk_proof: bytesInputToBase64(zkProof),
      }
      const resp = await fetch('/v1/electoral/ballot', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body),
      })
      const data = await resp.json()
      setBallotResult(resp.ok ? JSON.stringify(data, null, 2) : `HTTP ${resp.status}: ${JSON.stringify(data)}`)
    } catch (e) {
      setBallotResult(String(e))
    }
  }

  async function getElectionStatus() {
    setStatusResult('')
    const resp = await fetch(`/v1/electoral/elections/${statusElectionId}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    const data = await resp.json()
    setStatusResult(resp.ok ? JSON.stringify(data, null, 2) : `HTTP ${resp.status}: ${JSON.stringify(data)}`)
  }

  return (
    <div>
      <h1 style={{ fontSize: 24, marginBottom: 24 }}>ماژول انتخابات (FR-010)</h1>

      <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)', marginBottom: 16 }}>
        <h2 style={{ fontSize: 16, marginBottom: 12 }}>ثبت انتخابات (admin)</h2>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <label>
            نام
            <input value={name} onChange={e => setName(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8 }} />
          </label>
          <label>
            DID ادمین
            <input value={adminDid} onChange={e => setAdminDid(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
          <label>
            باز شدن (RFC3339)
            <input value={opensAt} onChange={e => setOpensAt(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
          <label>
            بسته شدن (RFC3339)
            <input value={closesAt} onChange={e => setClosesAt(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
        </div>

        <div style={{ marginTop: 12 }}>
          <button onClick={registerElection} style={{ background: '#1a56db', color: '#fff', border: 'none', borderRadius: 8, padding: '10px 14px', cursor: 'pointer' }}>
            ارسال
          </button>
        </div>
        {registerResult && <pre style={{ marginTop: 12, background: '#f8fafc', padding: 12, borderRadius: 8, overflow: 'auto' }}>{registerResult}</pre>}
      </div>

      <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)', marginBottom: 16 }}>
        <h2 style={{ fontSize: 16, marginBottom: 12 }}>ثبت/ارسال رأی (authenticated)</h2>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <label>
            election_id
            <input value={electionId} onChange={e => setElectionId(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
          <label>
            nullifier_hash
            <input value={nullifierHash} onChange={e => setNullifierHash(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
          <label style={{ gridColumn: '1 / -1' }}>
            encrypted_vote (hex یا base64)
            <textarea value={encryptedVote} onChange={e => setEncryptedVote(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, minHeight: 120, fontFamily: 'monospace' }} />
          </label>
          <label style={{ gridColumn: '1 / -1' }}>
            zk_proof (hex یا base64)
            <textarea value={zkProof} onChange={e => setZkProof(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, minHeight: 120, fontFamily: 'monospace' }} />
          </label>
        </div>

        <div style={{ marginTop: 12 }}>
          <button onClick={castBallot} style={{ background: '#0f9960', color: '#fff', border: 'none', borderRadius: 8, padding: '10px 14px', cursor: 'pointer' }}>
            ارسال رأی
          </button>
        </div>
        {ballotResult && <pre style={{ marginTop: 12, background: '#f8fafc', padding: 12, borderRadius: 8, overflow: 'auto' }}>{ballotResult}</pre>}
      </div>

      <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
        <h2 style={{ fontSize: 16, marginBottom: 12 }}>وضعیت انتخابات</h2>

        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
          <label style={{ flex: 1 }}>
            election_id
            <input value={statusElectionId} onChange={e => setStatusElectionId(e.target.value)} style={{ width: '100%', padding: 10, border: '1px solid #d1d5db', borderRadius: 8, fontFamily: 'monospace' }} />
          </label>
          <button onClick={getElectionStatus} style={{ background: '#1a56db', color: '#fff', border: 'none', borderRadius: 8, padding: '10px 14px', cursor: 'pointer', height: 42 }}>
            دریافت
          </button>
        </div>
        {statusResult && <pre style={{ marginTop: 12, background: '#f8fafc', padding: 12, borderRadius: 8, overflow: 'auto' }}>{statusResult}</pre>}
      </div>
    </div>
  )
}

