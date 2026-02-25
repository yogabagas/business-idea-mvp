import { useEffect, useState, useCallback } from 'react'
import { getMe, getLoginUrl, logout as apiLogout, getLiveChatId, getMessages } from './api'
import type { FilterConfig, ChatMessage } from './types'
import { defaultFilterConfig } from './types'
import { Header } from './Header'
import { VideoInput } from './VideoInput'
import { FilterSettings } from './FilterSettings'
import { ChatList } from './ChatList'

export default function App() {
  const [loggedIn, setLoggedIn] = useState(false)
  const [loading, setLoading] = useState(true)
  const [videoId, setVideoId] = useState('')
  const [liveChatId, setLiveChatId] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [nextPageToken, setNextPageToken] = useState('')
  const [pollingIntervalMs, setPollingIntervalMs] = useState(2000)
  const [filterConfig, setFilterConfig] = useState<FilterConfig>(defaultFilterConfig)
  const [error, setError] = useState('')

  useEffect(() => {
    getMe()
      .then((res) => setLoggedIn(res.loggedIn))
      .catch(() => setLoggedIn(false))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const err = params.get('error')
    if (err) {
      setError(err === 'exchange_failed' ? 'Login failed. Please try again.' : 'Auth error.')
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [])

  const handleLogin = () => {
    window.location.href = getLoginUrl()
  }

  const handleLogout = async () => {
    await apiLogout()
    setLoggedIn(false)
    setLiveChatId('')
    setMessages([])
    setError('')
  }

  const handleLoadChat = useCallback(async (input: string) => {
    setError('')
    try {
      const { liveChatId: id } = await getLiveChatId(input)
      setVideoId(input.replace(/^.*(?:v=|\/watch\/|youtu\.be\/)([a-zA-Z0-9_-]{11}).*$/, '$1').slice(0, 20))
      setLiveChatId(id)
      setMessages([])
      setNextPageToken('')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load live chat')
    }
  }, [])

  const fetchMessages = useCallback(async () => {
    if (!liveChatId) return
    try {
      const res = await getMessages(liveChatId, nextPageToken, filterConfig)
      setMessages((prev) => {
        const byId = new Map(prev.map((m) => [m.id, m]))
        for (const m of res.messages) byId.set(m.id, m)
        return [...byId.values()].sort(
          (a, b) => new Date(a.publishedAt).getTime() - new Date(b.publishedAt).getTime()
        )
      })
      setNextPageToken(res.nextPageToken || '')
      if (res.pollingIntervalMillis) setPollingIntervalMs(res.pollingIntervalMillis)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch messages')
    }
  }, [liveChatId, nextPageToken, filterConfig])

  useEffect(() => {
    if (!liveChatId) return
    fetchMessages()
  }, [liveChatId])

  useEffect(() => {
    if (!liveChatId || !nextPageToken) return
    const t = setInterval(fetchMessages, Math.max(pollingIntervalMs, 1000))
    return () => clearInterval(t)
  }, [liveChatId, nextPageToken, pollingIntervalMs, fetchMessages])

  if (loading) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <span style={{ color: 'var(--text-muted)' }}>Loading…</span>
      </div>
    )
  }

  if (!loggedIn) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: 24 }}>
        <h1 style={{ fontSize: '1.75rem', fontWeight: 600, marginBottom: 8 }}>YouTube Live Spam Filter</h1>
        <p style={{ color: 'var(--text-muted)', marginBottom: 24, textAlign: 'center' }}>
          Sign in with Google to filter spam in any YouTube Live chat.
        </p>
        {error && <p style={{ color: 'var(--accent)', marginBottom: 16 }}>{error}</p>}
        <button
          type="button"
          onClick={handleLogin}
          style={{
            background: 'var(--accent)',
            color: '#fff',
            border: 'none',
            padding: '12px 24px',
            borderRadius: 'var(--radius)',
            fontSize: 16,
            fontWeight: 600,
          }}
        >
          Sign in with Google
        </button>
      </div>
    )
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Header onLogout={handleLogout} />
      <main style={{ flex: 1, maxWidth: 900, width: '100%', margin: '0 auto', padding: 24 }}>
        <VideoInput onLoad={handleLoadChat} />
        {error && <p style={{ color: 'var(--accent)', marginTop: 8 }}>{error}</p>}
        <FilterSettings config={filterConfig} onChange={setFilterConfig} />
        {liveChatId && (
          <ChatList
            messages={messages}
            onRefresh={fetchMessages}
          />
        )}
      </main>
    </div>
  )
}
