/**
 * INDIS Gateway API client.
 *
 * All requests go through the Vite dev-proxy at /v1 (see vite.config.ts).
 * In production the service worker rewrites origin to the gateway URL.
 */

const BASE = '/v1'

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  token?: string,
): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers['Authorization'] = `Bearer ${token}`

  const resp = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body != null ? JSON.stringify(body) : undefined,
  })

  if (!resp.ok) {
    const text = await resp.text()
    throw new Error(`${method} ${path} → ${resp.status}: ${text}`)
  }
  return resp.json() as Promise<T>
}

export const api = {
  get: <T>(path: string, token?: string) => request<T>('GET', path, undefined, token),
  post: <T>(path: string, body: unknown, token?: string) => request<T>('POST', path, body, token),
}
