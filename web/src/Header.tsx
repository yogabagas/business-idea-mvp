interface HeaderProps {
  onLogout: () => void
}

export function Header({ onLogout }: HeaderProps) {
  return (
    <header
      style={{
        borderBottom: '1px solid var(--border)',
        padding: '16px 24px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        background: 'var(--bg-elevated)',
      }}
    >
      <h1 style={{ fontSize: '1.25rem', fontWeight: 600, margin: 0 }}>
        YouTube Live Spam Filter
      </h1>
      <button
        type="button"
        onClick={onLogout}
        style={{
          background: 'transparent',
          color: 'var(--text-muted)',
          border: '1px solid var(--border)',
          padding: '8px 16px',
          borderRadius: 'var(--radius)',
          fontSize: 14,
        }}
      >
        Log out
      </button>
    </header>
  )
}
