import { useEffect, useState, FormEvent } from 'react'
import { hasRole } from '../hooks/useGovAuth'
import type { GovRole } from '../hooks/useGovAuth'
import './Page.css'

interface PortalUser {
  id: string
  username: string
  ministry: string
  role: string
  created_at: string
}

interface Props {
  role: GovRole
  token: string
}

const ROLES = ['viewer', 'operator', 'senior', 'admin']

const MINISTRIES = [
  'وزارت کشور',
  'وزارت بهداشت',
  'وزارت آموزش و پرورش',
  'وزارت دفاع',
  'وزارت اطلاعات',
  'وزارت دادگستری',
  'وزارت ارتباطات',
  'سایر',
]

export default function UsersPage({ role, token }: Props) {
  const [users, setUsers] = useState<PortalUser[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)

  const canChangeRole = hasRole(role, 'senior')
  const canCreateUser = hasRole(role, 'senior')

  useEffect(() => {
    fetch('/v1/portal/users', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(data => setUsers((data as { users: PortalUser[] }).users ?? []))
      .finally(() => setLoading(false))
  }, [token])

  async function changeRole(id: string, newRole: string) {
    if (!canChangeRole) return
    await fetch(`/v1/portal/users/${id}/role`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ role: newRole }),
    })
    setUsers(prev => prev.map(u => u.id === id ? { ...u, role: newRole } : u))
  }

  function onUserCreated(user: PortalUser) {
    setUsers(prev => [user, ...prev])
    setShowModal(false)
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">کاربران پرتال</h1>
        {canCreateUser && (
          <button type="button" className="btn btn-primary" onClick={() => setShowModal(true)}>
            + کاربر جدید
          </button>
        )}
      </div>

      {!canCreateUser && (
        <p className="role-notice">برای ایجاد کاربر جدید به نقش «ارشد» یا بالاتر نیاز دارید.</p>
      )}

      {loading ? (
        <p className="page-loading">در حال بارگذاری…</p>
      ) : users.length === 0 ? (
        <p className="page-empty">هیچ کاربری یافت نشد.</p>
      ) : (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                {['نام کاربری', 'وزارتخانه', 'نقش', 'تاریخ ایجاد'].map(h => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {users.map(user => (
                <tr key={user.id}>
                  <td>{user.username}</td>
                  <td>{user.ministry}</td>
                  <td>
                    {canChangeRole ? (
                      <select
                        value={user.role}
                        onChange={e => changeRole(user.id, e.target.value)}
                        className="role-select"
                        title={`نقش ${user.username}`}
                        aria-label={`نقش ${user.username}`}
                      >
                        {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
                      </select>
                    ) : (
                      <span className="role-label">{user.role}</span>
                    )}
                  </td>
                  <td className="text-muted">
                    {new Date(user.created_at).toLocaleDateString('fa-IR')}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showModal && (
        <CreateUserModal
          token={token}
          onClose={() => setShowModal(false)}
          onCreated={onUserCreated}
          ministries={MINISTRIES}
          roles={ROLES}
        />
      )}
    </div>
  )
}

// ── Create User Modal ──────────────────────────────────────────────────────────

interface CreateUserModalProps {
  token: string
  onClose: () => void
  onCreated: (user: PortalUser) => void
  ministries: string[]
  roles: string[]
}

function CreateUserModal({ token, onClose, onCreated, ministries, roles }: CreateUserModalProps) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [ministry, setMinistry] = useState(ministries[0])
  const [role, setRole]         = useState('viewer')
  const [loading, setLoading]   = useState(false)
  const [error, setError]       = useState('')

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!username.trim() || !password.trim()) { setError('نام کاربری و رمز عبور الزامی است'); return }
    setLoading(true)
    setError('')
    try {
      const resp = await fetch('/v1/portal/users', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ username: username.trim(), password, ministry, role }),
      })
      if (!resp.ok) throw new Error(`خطای ${resp.status}: ${await resp.text()}`)
      onCreated(await resp.json() as PortalUser)
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-user-title"
      onClick={(e) => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="modal">
        <div className="modal-header">
          <h2 id="create-user-title" className="modal-title">ایجاد کاربر جدید</h2>
          <button type="button" onClick={onClose} className="modal-close" aria-label="بستن">✕</button>
        </div>

        <form onSubmit={(e) => void handleSubmit(e)} className="modal-form" noValidate>
          <label htmlFor="new-username" className="form-label">
            نام کاربری
            <input
              id="new-username"
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value)}
              dir="ltr"
              className="form-input"
              autoComplete="username"
              placeholder="نام کاربری"
            />
          </label>

          <label htmlFor="new-password" className="form-label">
            رمز عبور اولیه
            <input
              id="new-password"
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              className="form-input"
              autoComplete="new-password"
              placeholder="رمز عبور"
            />
          </label>

          <label htmlFor="new-ministry" className="form-label">
            وزارتخانه
            <select
              id="new-ministry"
              value={ministry}
              onChange={e => setMinistry(e.target.value)}
              className="form-input"
              title="وزارتخانه"
            >
              {ministries.map(m => <option key={m} value={m}>{m}</option>)}
            </select>
          </label>

          <label htmlFor="new-role" className="form-label">
            نقش
            <select
              id="new-role"
              value={role}
              onChange={e => setRole(e.target.value)}
              className="form-input"
              title="نقش کاربر"
            >
              {roles.map(r => <option key={r} value={r}>{r}</option>)}
            </select>
          </label>

          {error && <p className="form-error" role="alert">{error}</p>}

          <div className="modal-actions">
            <button type="button" onClick={onClose} className="btn btn-secondary">انصراف</button>
            <button type="submit" disabled={loading} className="btn btn-primary">
              {loading ? 'در حال ایجاد…' : 'ایجاد کاربر'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
