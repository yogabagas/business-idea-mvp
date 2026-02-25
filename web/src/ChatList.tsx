import type { ChatMessage } from './types'

interface ChatListProps {
  messages: ChatMessage[]
  onRefresh: () => void
}

function formatTime(iso: string) {
  try {
    const d = new Date(iso)
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return ''
  }
}

export function ChatList({ messages, onRefresh }: ChatListProps) {
  return (
    <section
      style={{
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius)',
        background: 'var(--bg-elevated)',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          padding: '12px 16px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <h2 style={{ fontSize: '1rem', fontWeight: 600, margin: 0 }}>Live chat</h2>
        <button
          type="button"
          onClick={onRefresh}
          style={{
            background: 'transparent',
            color: 'var(--text-muted)',
            border: '1px solid var(--border)',
            padding: '6px 12px',
            borderRadius: 'var(--radius)',
            fontSize: 13,
          }}
        >
          Refresh
        </button>
      </div>
      <div
        style={{
          maxHeight: 480,
          overflowY: 'auto',
          padding: 12,
        }}
      >
        {messages.length === 0 && (
          <p style={{ color: 'var(--text-muted)', textAlign: 'center', padding: 24 }}>
            No messages yet. Messages appear as they’re sent in the live chat.
          </p>
        )}
        {messages.map((m) => (
          <div
            key={m.id}
            style={{
              padding: '10px 12px',
              marginBottom: 6,
              borderRadius: 'var(--radius)',
              background: m.isSpam ? 'var(--spam-bg)' : 'transparent',
              borderLeft: m.isSpam ? '3px solid var(--spam-border)' : '3px solid transparent',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
              {m.profileImageUrl && (
                <img
                  src={m.profileImageUrl}
                  alt=""
                  width={32}
                  height={32}
                  style={{ borderRadius: '50%', flexShrink: 0 }}
                />
              )}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4, flexWrap: 'wrap' }}>
                  <span style={{ fontWeight: 600, fontSize: 14 }}>{m.authorName || 'Unknown'}</span>
                  <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>{formatTime(m.publishedAt)}</span>
                  {m.isSpam && (
                    <span
                      style={{
                        fontSize: 11,
                        background: 'var(--accent)',
                        color: '#fff',
                        padding: '2px 8px',
                        borderRadius: 4,
                      }}
                    >
                      Spam {m.reasons?.length ? `(${m.reasons.join(', ')})` : ''}
                    </span>
                  )}
                </div>
                <p style={{ margin: 0, wordBreak: 'break-word', whiteSpace: 'pre-wrap' }}>{m.text}</p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}
