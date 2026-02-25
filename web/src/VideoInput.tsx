import { useState } from 'react'

interface VideoInputProps {
  onLoad: (videoIdOrUrl: string) => void
  disabled?: boolean
}

export function VideoInput({ onLoad, disabled }: VideoInputProps) {
  const [value, setValue] = useState('')

  const submit = () => {
    const v = value.trim()
    if (!v) return
    onLoad(v)
  }

  return (
    <section style={{ marginBottom: 24 }}>
      <label style={{ display: 'block', marginBottom: 8, fontWeight: 500, color: 'var(--text-muted)' }}>
        YouTube Live URL or Video ID
      </label>
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
        <input
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && submit()}
          placeholder="https://www.youtube.com/watch?v=... or VIDEO_ID"
          disabled={disabled}
          style={{
            flex: 1,
            minWidth: 200,
            padding: '12px 16px',
            borderRadius: 'var(--radius)',
            border: '1px solid var(--border)',
            background: 'var(--bg-elevated)',
            color: 'var(--text)',
            fontSize: 15,
          }}
        />
        <button
          type="button"
          onClick={submit}
          disabled={disabled || !value.trim()}
          style={{
            background: 'var(--accent)',
            color: '#fff',
            border: 'none',
            padding: '12px 20px',
            borderRadius: 'var(--radius)',
            fontSize: 15,
            fontWeight: 500,
          }}
        >
          Load chat
        </button>
      </div>
    </section>
  )
}
