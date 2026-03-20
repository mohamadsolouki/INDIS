import { useCredentials } from '../hooks/useCredentials'
import { StoredCredential } from '../lib/wallet'

export default function WalletPage() {
  const { credentials, loading, error } = useCredentials()

  if (loading) return <Loading />

  return (
    <div style={{ padding: 20 }}>
      <h2 style={{ marginBottom: 20 }}>کیف اعتبارنامه</h2>

      {error && (
        <div className="card" style={{ color: 'var(--color-error)', marginBottom: 16 }}>
          خطا: {error}
        </div>
      )}

      {credentials.length === 0 ? (
        <div className="card text-center">
          <p style={{ fontSize: 40, marginBottom: 8 }}>🪪</p>
          <p className="text-muted">هنوز اعتبارنامه‌ای ندارید.</p>
          <p className="text-muted" style={{ fontSize: 12, marginTop: 4 }}>
            پس از تأیید ثبت‌نام، اعتبارنامه‌ها اینجا نمایش داده می‌شوند.
          </p>
        </div>
      ) : (
        <div className="flex-col gap-4">
          {credentials.map(c => (
            <CredentialCard key={c.id} cred={c} />
          ))}
        </div>
      )}
    </div>
  )
}

function CredentialCard({ cred }: { cred: StoredCredential }) {
  const issued = new Date(cred.issuedAt).toLocaleDateString('fa-IR')
  const expires = new Date(cred.expiresAt).toLocaleDateString('fa-IR')
  const isExpired = new Date(cred.expiresAt) < new Date()

  return (
    <div
      className="card"
      style={{
        borderLeft: `4px solid ${isExpired ? 'var(--color-error)' : 'var(--color-primary)'}`,
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <p style={{ fontWeight: 600, fontSize: 16 }}>{cred.type}</p>
          <p className="text-muted" style={{ fontSize: 12, marginTop: 2 }}>{cred.issuer}</p>
        </div>
        <span
          style={{
            fontSize: 11,
            padding: '3px 8px',
            borderRadius: 20,
            background: isExpired ? '#fde8e8' : 'var(--color-primary-light)',
            color: isExpired ? 'var(--color-error)' : 'var(--color-primary)',
          }}
        >
          {isExpired ? 'منقضی' : 'معتبر'}
        </span>
      </div>
      <div style={{ marginTop: 12, display: 'flex', gap: 20 }}>
        <div>
          <p style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>صادر شده</p>
          <p style={{ fontSize: 13 }}>{issued}</p>
        </div>
        <div>
          <p style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>انقضا</p>
          <p style={{ fontSize: 13 }}>{expires}</p>
        </div>
      </div>
    </div>
  )
}

function Loading() {
  return (
    <div style={{ padding: 20 }}>
      <h2 style={{ marginBottom: 20 }}>کیف اعتبارنامه</h2>
      {[1, 2, 3].map(i => (
        <div
          key={i}
          className="card"
          style={{
            height: 100,
            marginBottom: 12,
            background: 'linear-gradient(90deg, #f0f0f0 25%, #e8e8e8 50%, #f0f0f0 75%)',
            backgroundSize: '200% 100%',
            animation: 'shimmer 1.5s infinite',
          }}
        />
      ))}
    </div>
  )
}
