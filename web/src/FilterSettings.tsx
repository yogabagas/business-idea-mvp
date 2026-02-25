import type { FilterConfig } from './types'

interface FilterSettingsProps {
  config: FilterConfig
  onChange: (c: FilterConfig) => void
}

export function FilterSettings({ config, onChange }: FilterSettingsProps) {
  const update = (patch: Partial<FilterConfig>) => onChange({ ...config, ...patch })

  return (
    <section
      style={{
        marginBottom: 24,
        padding: 20,
        background: 'var(--bg-elevated)',
        borderRadius: 'var(--radius)',
        border: '1px solid var(--border)',
      }}
    >
      <h2 style={{ fontSize: '1rem', fontWeight: 600, margin: '0 0 16px 0' }}>Spam filters</h2>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <label style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <input
            type="checkbox"
            checked={config.linkDetection}
            onChange={(e) => update({ linkDetection: e.target.checked })}
          />
          <span>Flag messages containing links</span>
        </label>
        <label style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <input
            type="checkbox"
            checked={config.hideFiltered}
            onChange={(e) => update({ hideFiltered: e.target.checked })}
          />
          <span>Hide spam (otherwise show as flagged)</span>
        </label>
        <div>
          <label style={{ display: 'block', marginBottom: 6, color: 'var(--text-muted)' }}>
            Repetition threshold (same message N times = spam, 0 = off)
          </label>
          <input
            type="number"
            min={0}
            value={config.repetitionThreshold}
            onChange={(e) => update({ repetitionThreshold: Math.max(0, parseInt(e.target.value, 10) || 0) })}
            style={{
              width: 80,
              padding: '8px 12px',
              borderRadius: 'var(--radius)',
              border: '1px solid var(--border)',
              background: 'var(--bg)',
              color: 'var(--text)',
            }}
          />
        </div>
        <div>
          <label style={{ display: 'block', marginBottom: 6, color: 'var(--text-muted)' }}>
            Blocklist (one word/phrase per line, case-insensitive)
          </label>
          <textarea
            value={config.blocklist.join('\n')}
            onChange={(e) =>
              update({
                blocklist: e.target.value
                  .split('\n')
                  .map((s) => s.trim())
                  .filter(Boolean),
              })
            }
            rows={4}
            placeholder="spam&#10;promo&#10;subscribe"
            style={{
              width: '100%',
              padding: '12px',
              borderRadius: 'var(--radius)',
              border: '1px solid var(--border)',
              background: 'var(--bg)',
              color: 'var(--text)',
              fontFamily: 'var(--font-mono)',
              fontSize: 14,
              resize: 'vertical',
            }}
          />
        </div>
        <div>
          <label style={{ display: 'block', marginBottom: 6, color: 'var(--text-muted)' }}>
            Custom regex (optional)
          </label>
          <input
            type="text"
            value={config.customRegex}
            onChange={(e) => update({ customRegex: e.target.value })}
            placeholder="e.g. (?:free|win|click)"
            style={{
              width: '100%',
              padding: '10px 12px',
              borderRadius: 'var(--radius)',
              border: '1px solid var(--border)',
              background: 'var(--bg)',
              color: 'var(--text)',
              fontFamily: 'var(--font-mono)',
              fontSize: 14,
            }}
          />
        </div>
      </div>
    </section>
  )
}
