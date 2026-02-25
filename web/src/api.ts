const API = '/api'

function credentials(): RequestCredentials {
  return 'include'
}

export async function getMe(): Promise<{ loggedIn: boolean }> {
  const r = await fetch(`${API}/auth/me`, { credentials: credentials() })
  return r.json()
}

export function getLoginUrl(): string {
  return `${API}/auth/login`
}

export async function logout(): Promise<void> {
  await fetch(`${API}/auth/logout`, { method: 'POST', credentials: credentials() })
}

export async function getLiveChatId(videoId: string): Promise<{ liveChatId: string; videoId: string }> {
  const params = new URLSearchParams({ videoId })
  const r = await fetch(`${API}/live-chat/id?${params}`, { credentials: credentials() })
  if (!r.ok) {
    const e = await r.json().catch(() => ({}))
    throw new Error((e as { error?: string }).error || r.statusText)
  }
  return r.json()
}

export async function getMessages(
  liveChatId: string,
  pageToken: string,
  filterConfig: import('./types').FilterConfig
): Promise<import('./types').MessagesResponse> {
  const r = await fetch(`${API}/live-chat/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: credentials(),
    body: JSON.stringify({
      liveChatId,
      pageToken,
      filterConfig,
    }),
  })
  if (!r.ok) {
    const e = await r.json().catch(() => ({}))
    throw new Error((e as { error?: string }).error || r.statusText)
  }
  return r.json()
}
