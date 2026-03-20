import { useEffect, useState } from 'react'

interface PortalUser {
  id: string
  username: string
  ministry: string
  role: string
  created_at: string
}

const ROLES = ['viewer', 'operator', 'senior', 'admin']

export default function UsersPage() {
  const [users, setUsers] = useState<PortalUser[]>([])
  const [loading, setLoading] = useState(true)
  const token = localStorage.getItem('gov_token')

  useEffect(() => {
    fetch('/v1/portal/users', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(data => setUsers((data as { users: PortalUser[] }).users ?? []))
      .finally(() => setLoading(false))
  }, [token])

  async function changeRole(id: string, newRole: string) {
    await fetch(`/v1/portal/users/${id}/role`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ role: newRole }),
    })
    setUsers(prev => prev.map(u => u.id === id ? { ...u, role: newRole } : u))
  }

  return (
    <div>
      <h1 style={{ fontSize: 24, marginBottom: 24 }}>کاربران پرتال</h1>

      {loading ? (
        <p style={{ color: '#666' }}>در حال بارگذاری…</p>
      ) : (
        <div style={{ background: '#fff', borderRadius: 12, overflow: 'hidden', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
            <thead>
              <tr style={{ background: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                {['نام کاربری', 'وزارتخانه', 'نقش', 'تاریخ ایجاد'].map(h => (
                  <th key={h} style={{ padding: '12px 16px', textAlign: 'right', fontWeight: 600, color: '#555' }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {users.map(user => (
                <tr key={user.id} style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <td style={{ padding: '12px 16px' }}>{user.username}</td>
                  <td style={{ padding: '12px 16px' }}>{user.ministry}</td>
                  <td style={{ padding: '12px 16px' }}>
                    <select
                      value={user.role}
                      onChange={e => changeRole(user.id, e.target.value)}
                      style={{ fontSize: 13, padding: '4px 8px', borderRadius: 6, border: '1px solid #d1d5db' }}
                    >
                      {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
                    </select>
                  </td>
                  <td style={{ padding: '12px 16px', fontSize: 12, color: '#666' }}>
                    {new Date(user.created_at).toLocaleDateString('fa-IR')}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
