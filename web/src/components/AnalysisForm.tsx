import { FormEvent, useState } from 'react'
import { AnalysisRequest, ProgressPayload } from '../types'

interface AnalysisFormProps {
  onSubmit: (req: AnalysisRequest) => void
  isRunning: boolean
  progress: ProgressPayload | null
  error: string | null
}

export default function AnalysisForm({ onSubmit, isRunning, progress, error }: AnalysisFormProps) {
  const [repoPath, setRepoPath] = useState('')
  const [baseBranch, setBaseBranch] = useState('main')
  const [headBranch, setHeadBranch] = useState('')
  const [demoLoading, setDemoLoading] = useState(false)
  const [demoError, setDemoError] = useState<string | null>(null)
  const [prURL, setPrURL] = useState('')
  const [prLoading, setPrLoading] = useState(false)
  const [prError, setPrError] = useState<string | null>(null)
  const [prResolved, setPrResolved] = useState<{ repo: string; prNumber: number } | null>(null)

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!headBranch.trim()) return
    onSubmit({
      repoPath: repoPath.trim() || '.',
      baseBranch: baseBranch.trim() || 'main',
      headBranch: headBranch.trim(),
    })
  }

  async function handleImportPR() {
    if (!prURL.trim()) return
    setPrLoading(true)
    setPrError(null)
    setPrResolved(null)
    try {
      const res = await fetch('/api/github-pr', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prUrl: prURL.trim(), repoPath: repoPath.trim() || undefined }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.message ?? `Server returned ${res.status}`)
      setRepoPath(data.repoPath)
      setBaseBranch(data.baseBranch)
      setHeadBranch(data.headBranch)
      setPrResolved({ repo: data.repo, prNumber: data.prNumber })
    } catch (err) {
      setPrError(err instanceof Error ? err.message : 'Failed to resolve PR')
    } finally {
      setPrLoading(false)
    }
  }

  async function handleDemo() {
    setDemoLoading(true)
    setDemoError(null)
    try {
      const res = await fetch('/api/demo')
      if (!res.ok) throw new Error(`Server returned ${res.status}`)
      const data: AnalysisRequest = await res.json()
      setRepoPath(data.repoPath)
      setBaseBranch(data.baseBranch)
      setHeadBranch(data.headBranch)
    } catch (err) {
      setDemoError(err instanceof Error ? err.message : 'Failed to load demo config')
    } finally {
      setDemoLoading(false)
    }
  }

  return (
    <div className="form-wrap">
      <div className="form-card">
        <div className="form-header">
          <div className="form-title-row">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <circle cx="5" cy="4" r="1.5" stroke="#f59e0b" strokeWidth="1.2" />
              <circle cx="11" cy="4" r="1.5" stroke="#f59e0b" strokeWidth="1.2" />
              <circle cx="8" cy="12" r="1.5" stroke="#dc2626" strokeWidth="1.2" />
              <line x1="5" y1="5.5" x2="7" y2="10.5" stroke="#737373" strokeWidth="0.8" />
              <line x1="11" y1="5.5" x2="9" y2="10.5" stroke="#737373" strokeWidth="0.8" />
            </svg>
            <h1 className="form-title">Analyze Blast Radius</h1>
            <button
              type="button"
              className="demo-btn"
              onClick={handleDemo}
              disabled={demoLoading || isRunning}
              aria-label="Load demo configuration"
            >
              {demoLoading ? (
                <span className="form-spinner demo-spinner" aria-hidden="true" />
              ) : (
                <svg width="11" height="11" viewBox="0 0 11 11" fill="none" aria-hidden="true">
                  <polygon points="2,1 10,5.5 2,10" fill="currentColor" />
                </svg>
              )}
              {demoLoading ? 'Loading…' : 'Try demo'}
            </button>
          </div>
          <p className="form-subtitle">
            Trace every function your PR will break — before your teammates find them.
          </p>
          {demoError && (
            <p className="demo-error" role="alert">{demoError}</p>
          )}
        </div>

        <form className="form-body" onSubmit={handleSubmit}>
          {/* GitHub PR import */}
          <div className="form-field">
            <label className="form-label" htmlFor="prURL">
              Import from GitHub PR
              <span className="form-label-optional"> — optional</span>
            </label>
            <div className="pr-row">
              <input
                id="prURL"
                className="form-input mono"
                type="url"
                value={prURL}
                onChange={e => { setPrURL(e.target.value); setPrResolved(null); setPrError(null) }}
                placeholder="https://github.com/owner/repo/pull/123"
                autoComplete="off"
                spellCheck={false}
              />
              <button
                type="button"
                className="pr-import-btn"
                onClick={handleImportPR}
                disabled={prLoading || !prURL.trim() || isRunning}
              >
                {prLoading ? <span className="form-spinner pr-spinner" aria-hidden="true" /> : 'Import'}
              </button>
            </div>
            {prError && <span className="pr-error">{prError}</span>}
            {prResolved && (
              <span className="pr-resolved">
                <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
                  <path d="M2 5l2.5 2.5L8 3" stroke="#16a34a" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
                {prResolved.repo} #{prResolved.prNumber} — branches filled below
              </span>
            )}
            <span className="form-hint">Fills base/head branches automatically. Requires <span className="mono">GITHUB_TOKEN</span> env var for private repos.</span>
          </div>

          <div className="form-divider" aria-hidden="true"><span>or enter manually</span></div>

          <div className="form-field">
            <label className="form-label" htmlFor="repoPath">
              Repository path
            </label>
            <input
              id="repoPath"
              className="form-input"
              type="text"
              value={repoPath}
              onChange={e => setRepoPath(e.target.value)}
              placeholder="/path/to/your/repo (or . for current)"
              autoComplete="off"
              spellCheck={false}
            />
            <span className="form-hint">Absolute path to the git repository root</span>
          </div>

          <div className="form-row">
            <div className="form-field">
              <label className="form-label" htmlFor="baseBranch">
                Base branch
              </label>
              <input
                id="baseBranch"
                className="form-input mono"
                type="text"
                value={baseBranch}
                onChange={e => setBaseBranch(e.target.value)}
                placeholder="main"
                autoComplete="off"
                spellCheck={false}
              />
            </div>
            <div className="form-field-arrow" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path d="M4 10h12M12 6l4 4-4 4" stroke="#4a4a4a" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </div>
            <div className="form-field">
              <label className="form-label" htmlFor="headBranch">
                PR branch <span className="form-required">*</span>
              </label>
              <input
                id="headBranch"
                className="form-input mono"
                type="text"
                value={headBranch}
                onChange={e => setHeadBranch(e.target.value)}
                placeholder="feature/my-changes"
                required
                autoComplete="off"
                spellCheck={false}
              />
            </div>
          </div>

          {error && (
            <div className="form-error" role="alert">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                <circle cx="7" cy="7" r="6" stroke="#dc2626" strokeWidth="1.2" />
                <path d="M7 4v4M7 9.5v.5" stroke="#dc2626" strokeWidth="1.2" strokeLinecap="round" />
              </svg>
              {error}
            </div>
          )}

          {isRunning && progress && (
            <div className="form-progress">
              <div className="form-progress-bar">
                <div
                  className="form-progress-fill"
                  style={{ width: `${progress.percent}%` }}
                />
              </div>
              <span className="form-progress-msg">{progress.message}</span>
            </div>
          )}

          <button
            type="submit"
            className="form-submit"
            disabled={isRunning || !headBranch.trim()}
            aria-busy={isRunning}
          >
            {isRunning ? (
              <>
                <span className="form-spinner" aria-hidden="true" />
                Analyzing…
              </>
            ) : (
              <>
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                  <path d="M2 7h10M8 3l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
                Run Analysis
              </>
            )}
          </button>
        </form>

        <div className="form-example">
          <span className="form-example-label">Example:</span>
          <span className="form-example-code mono">
            Repo: /home/user/myproject · Base: main · PR: feat/refactor-auth
          </span>
        </div>
      </div>

      <style>{`
        .form-wrap {
          display: flex;
          align-items: center;
          justify-content: center;
          flex: 1;
          padding: var(--space-8);
          overflow-y: auto;
        }
        .form-card {
          width: 100%;
          max-width: 560px;
          background: var(--surface);
          border: 1px solid var(--border);
          border-radius: var(--radius-lg);
          overflow: hidden;
        }
        .form-header {
          padding: var(--space-6);
          border-bottom: 1px solid var(--border);
          background: linear-gradient(to bottom, var(--surface-2), var(--surface));
        }
        .form-title-row {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          margin-bottom: var(--space-2);
        }
        .form-title {
          font-size: var(--text-lg);
          font-weight: 600;
          letter-spacing: -0.02em;
          color: var(--text-primary);
        }
        .form-subtitle {
          font-size: var(--text-sm);
          color: var(--text-secondary);
          line-height: 1.6;
        }
        .form-body {
          padding: var(--space-6);
          display: flex;
          flex-direction: column;
          gap: var(--space-5);
        }
        .form-field {
          display: flex;
          flex-direction: column;
          gap: var(--space-1);
          flex: 1;
        }
        .form-row {
          display: flex;
          align-items: flex-end;
          gap: var(--space-3);
        }
        .form-field-arrow {
          padding-bottom: 10px;
          flex-shrink: 0;
        }
        .form-label {
          font-size: var(--text-xs);
          font-weight: 500;
          letter-spacing: 0.06em;
          text-transform: uppercase;
          color: var(--text-secondary);
        }
        .form-required {
          color: var(--amber);
        }
        .form-input {
          height: 36px;
          padding: 0 var(--space-3);
          background: var(--surface-2);
          border: 1px solid var(--border);
          border-radius: var(--radius-md);
          color: var(--text-primary);
          font-size: var(--text-sm);
          transition: border-color var(--dur-fast);
          width: 100%;
        }
        .form-input::placeholder {
          color: var(--text-dim);
        }
        .form-input:hover {
          border-color: var(--border-2);
        }
        .form-input:focus {
          border-color: var(--amber);
          background: var(--surface-3);
          box-shadow: 0 0 0 3px var(--amber-glow);
          outline: none;
        }
        .form-hint {
          font-size: var(--text-xs);
          color: var(--text-dim);
        }
        .form-error {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          padding: var(--space-3);
          background: var(--red-glow);
          border: 1px solid var(--red-dim);
          border-radius: var(--radius-md);
          font-size: var(--text-sm);
          color: #fca5a5;
        }
        .form-progress {
          display: flex;
          flex-direction: column;
          gap: var(--space-2);
        }
        .form-progress-bar {
          height: 2px;
          background: var(--border);
          border-radius: 2px;
          overflow: hidden;
        }
        .form-progress-fill {
          height: 100%;
          background: var(--amber);
          border-radius: 2px;
          transition: width 300ms var(--ease-out);
        }
        .form-progress-msg {
          font-size: var(--text-xs);
          color: var(--text-secondary);
          font-family: var(--font-mono);
        }
        .form-submit {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: var(--space-2);
          height: 38px;
          padding: 0 var(--space-5);
          background: var(--amber);
          color: #1a0e00;
          font-size: var(--text-sm);
          font-weight: 600;
          border-radius: var(--radius-md);
          transition: opacity var(--dur-fast), transform var(--dur-fast);
          border: none;
        }
        .form-submit:hover:not(:disabled) {
          opacity: 0.9;
          transform: translateY(-1px);
        }
        .form-submit:active:not(:disabled) {
          transform: translateY(0);
        }
        .form-submit:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
        .form-spinner {
          display: inline-block;
          width: 12px;
          height: 12px;
          border: 2px solid rgba(26, 14, 0, 0.3);
          border-top-color: #1a0e00;
          border-radius: 50%;
          animation: spin 0.7s linear infinite;
        }
        @keyframes spin {
          to { transform: rotate(360deg); }
        }
        .form-example {
          padding: var(--space-3) var(--space-6);
          border-top: 1px solid var(--border);
          display: flex;
          align-items: center;
          gap: var(--space-2);
          background: var(--surface-2);
        }
        .form-example-label {
          font-size: var(--text-xs);
          color: var(--text-dim);
          white-space: nowrap;
        }
        .form-example-code {
          font-size: var(--text-xs);
          color: var(--text-secondary);
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
        .demo-btn {
          display: flex;
          align-items: center;
          gap: 5px;
          margin-left: auto;
          padding: 4px 10px;
          height: 26px;
          background: transparent;
          border: 1px solid var(--border-2);
          border-radius: var(--radius-sm);
          color: var(--text-secondary);
          font-size: 11px;
          font-weight: 500;
          cursor: pointer;
          transition: color var(--dur-fast), border-color var(--dur-fast);
          white-space: nowrap;
        }
        .demo-btn:hover:not(:disabled) {
          color: var(--amber);
          border-color: var(--amber);
        }
        .demo-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
        .demo-spinner {
          border-color: rgba(115, 115, 115, 0.3) !important;
          border-top-color: var(--text-secondary) !important;
        }
        .demo-error {
          margin-top: var(--space-2);
          font-size: var(--text-xs);
          color: #fca5a5;
        }
        .form-label-optional {
          font-weight: 400;
          text-transform: none;
          letter-spacing: 0;
          color: var(--text-dim);
        }
        .pr-row {
          display: flex;
          gap: var(--space-2);
        }
        .pr-row .form-input {
          flex: 1;
          min-width: 0;
        }
        .pr-import-btn {
          flex-shrink: 0;
          height: 36px;
          padding: 0 var(--space-4);
          background: var(--surface-3);
          border: 1px solid var(--border-2);
          border-radius: var(--radius-md);
          color: var(--text-primary);
          font-size: var(--text-sm);
          font-weight: 500;
          cursor: pointer;
          transition: border-color var(--dur-fast), background var(--dur-fast);
          display: flex;
          align-items: center;
          justify-content: center;
          min-width: 68px;
        }
        .pr-import-btn:hover:not(:disabled) {
          border-color: var(--amber);
          color: var(--amber);
        }
        .pr-import-btn:disabled {
          opacity: 0.45;
          cursor: not-allowed;
        }
        .pr-spinner {
          border-color: rgba(230, 230, 230, 0.25) !important;
          border-top-color: var(--text-secondary) !important;
        }
        .pr-error {
          font-size: var(--text-xs);
          color: #fca5a5;
          margin-top: 2px;
        }
        .pr-resolved {
          display: flex;
          align-items: center;
          gap: 4px;
          font-size: var(--text-xs);
          color: #4ade80;
          margin-top: 2px;
        }
        .form-divider {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          font-size: var(--text-xs);
          color: var(--text-dim);
        }
        .form-divider::before,
        .form-divider::after {
          content: '';
          flex: 1;
          height: 1px;
          background: var(--border);
        }
      `}</style>
    </div>
  )
}
