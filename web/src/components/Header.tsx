import { CompletePayload } from '../types'

interface HeaderProps {
  summary: CompletePayload | null
  onReset: () => void
  isRunning: boolean
}

export default function Header({ summary, onReset, isRunning }: HeaderProps) {
  return (
    <header className="header">
      <div className="header-brand">
        <div className="header-logo">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
            <circle cx="10" cy="10" r="3" fill="#f59e0b" />
            <circle cx="10" cy="10" r="6" stroke="#f59e0b" strokeWidth="1" opacity="0.4" />
            <circle cx="10" cy="10" r="9" stroke="#f59e0b" strokeWidth="0.5" opacity="0.2" />
            <line x1="10" y1="1" x2="10" y2="7" stroke="#f59e0b" strokeWidth="1.5" strokeLinecap="round" />
            <line x1="10" y1="13" x2="10" y2="19" stroke="#dc2626" strokeWidth="1.5" strokeLinecap="round" />
            <line x1="1" y1="10" x2="7" y2="10" stroke="#f59e0b" strokeWidth="1.5" strokeLinecap="round" />
            <line x1="13" y1="10" x2="19" y2="10" stroke="#dc2626" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
        </div>
        <span className="header-name">CallBlast</span>
        <span className="header-tag">blast-radius analyzer</span>
      </div>

      {summary && (
        <div className="header-stats">
          <div className="header-stat">
            <span className="header-stat-value amber">{summary.totalChanged}</span>
            <span className="header-stat-label">changed</span>
          </div>
          <div className="header-stat-sep" />
          <div className="header-stat">
            <span className="header-stat-value">{summary.totalAffected}</span>
            <span className="header-stat-label">affected</span>
          </div>
          <div className="header-stat-sep" />
          <div className="header-stat">
            <span className="header-stat-value">{summary.maxDepth}</span>
            <span className="header-stat-label">max depth</span>
          </div>
          <div className="header-stat-sep" />
          <div className="header-stat">
            <span className="header-stat-value mono">{summary.durationMs.toFixed(0)}ms</span>
            <span className="header-stat-label">duration</span>
          </div>
        </div>
      )}

      {(summary || isRunning) && (
        <button className="header-reset" onClick={onReset} aria-label="Reset analysis">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
            <path d="M2 7a5 5 0 1 0 1-3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <path d="M2 4V7h3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          New analysis
        </button>
      )}

      <style>{`
        .header {
          height: var(--header-h);
          background: var(--surface);
          border-bottom: 1px solid var(--border);
          display: flex;
          align-items: center;
          padding: 0 var(--space-5);
          gap: var(--space-6);
          flex-shrink: 0;
        }
        .header-brand {
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }
        .header-logo {
          display: flex;
          align-items: center;
        }
        .header-name {
          font-weight: 700;
          font-size: var(--text-md);
          letter-spacing: -0.02em;
          color: var(--text-primary);
        }
        .header-tag {
          font-size: var(--text-xs);
          color: var(--text-secondary);
          font-family: var(--font-mono);
          padding: 2px 6px;
          background: var(--surface-2);
          border: 1px solid var(--border);
          border-radius: var(--radius-sm);
        }
        .header-stats {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          margin-left: auto;
        }
        .header-stat {
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 1px;
        }
        .header-stat-value {
          font-family: var(--font-mono);
          font-size: var(--text-md);
          font-weight: 500;
          color: var(--text-primary);
          line-height: 1;
        }
        .header-stat-value.amber {
          color: var(--amber);
        }
        .header-stat-label {
          font-size: var(--text-xs);
          color: var(--text-secondary);
          letter-spacing: 0.04em;
          text-transform: uppercase;
        }
        .header-stat-sep {
          width: 1px;
          height: 28px;
          background: var(--border);
        }
        .header-reset {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          padding: 6px var(--space-3);
          font-size: var(--text-sm);
          color: var(--text-secondary);
          background: transparent;
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
          transition: color var(--dur-fast), border-color var(--dur-fast), background var(--dur-fast);
        }
        .header-reset:hover {
          color: var(--text-primary);
          border-color: var(--border-2);
          background: var(--surface-2);
        }
      `}</style>
    </header>
  )
}
