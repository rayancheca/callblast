import { GraphNodePayload } from '../types'

interface FileImpact {
  file: string
  count: number
  criticalCount: number
  changedCount: number
  maxScore: number
}

interface ImpactListProps {
  nodes: Map<string, GraphNodePayload>
  selectedFile: string | null
  onSelectFile: (file: string | null) => void
}

function computeFileImpacts(nodes: Map<string, GraphNodePayload>): FileImpact[] {
  const byFile = new Map<string, FileImpact>()
  const isOrigin = (ct: string) => ['added', 'removed', 'signature_changed', 'body_changed', 'renamed'].includes(ct)

  for (const node of nodes.values()) {
    const f = node.file || 'unknown'
    const existing = byFile.get(f) ?? {
      file: f,
      count: 0,
      criticalCount: 0,
      changedCount: 0,
      maxScore: 0,
    }
    byFile.set(f, {
      file: f,
      count: existing.count + 1,
      criticalCount: existing.criticalCount + (node.changeType === 'critical' ? 1 : 0),
      changedCount: existing.changedCount + (isOrigin(node.changeType) ? 1 : 0),
      maxScore: Math.max(existing.maxScore, node.score),
    })
  }

  return Array.from(byFile.values()).sort((a, b) => b.maxScore - a.maxScore || b.count - a.count)
}

export default function ImpactList({ nodes, selectedFile, onSelectFile }: ImpactListProps) {
  const impacts = computeFileImpacts(nodes)

  if (impacts.length === 0) return null

  return (
    <aside className="impact-list" aria-label="File impact list">
      <div className="impact-header">
        <div className="impact-title">Affected files</div>
        <span className="impact-count-badge mono">{impacts.length}</span>
      </div>

      <div className="impact-body">
        {impacts.map(fi => {
          const isSelected = selectedFile === fi.file
          const barWidth = Math.round(fi.maxScore * 100)
          const severity = fi.criticalCount > 0 ? 'critical' : fi.changedCount > 0 ? 'changed' : 'affected'

          return (
            <button
              key={fi.file}
              className={`impact-item ${isSelected ? 'impact-item--selected' : ''}`}
              onClick={() => onSelectFile(isSelected ? null : fi.file)}
              aria-pressed={isSelected}
            >
              <div className="impact-item-header">
                <span className={`impact-sev-dot impact-sev-dot--${severity}`} />
                <span className="impact-file-name mono">{fi.file}</span>
                <span className="impact-fn-count mono">{fi.count}</span>
              </div>
              <div className="impact-bar-row">
                <div className="impact-bar">
                  <div
                    className="impact-bar-fill"
                    style={{
                      width: `${barWidth}%`,
                      background: severity === 'critical' ? '#dc2626' : severity === 'changed' ? '#f59e0b' : '#3b82f6',
                    }}
                  />
                </div>
              </div>
              {(fi.changedCount > 0 || fi.criticalCount > 0) && (
                <div className="impact-item-tags">
                  {fi.changedCount > 0 && (
                    <span className="impact-tag impact-tag--amber">{fi.changedCount} changed</span>
                  )}
                  {fi.criticalCount > 0 && (
                    <span className="impact-tag impact-tag--red">{fi.criticalCount} critical</span>
                  )}
                </div>
              )}
            </button>
          )
        })}
      </div>

      <style>{`
        .impact-list {
          width: var(--sidebar-w);
          flex-shrink: 0;
          background: var(--surface);
          border-right: 1px solid var(--border);
          display: flex;
          flex-direction: column;
          overflow: hidden;
          animation: slideInLeft var(--dur-normal) var(--ease-out);
        }
        @keyframes slideInLeft {
          from { transform: translateX(calc(-1 * var(--sidebar-w))); opacity: 0; }
          to { transform: translateX(0); opacity: 1; }
        }
        .impact-header {
          padding: var(--space-3) var(--space-4);
          border-bottom: 1px solid var(--border);
          background: var(--surface-2);
          display: flex;
          align-items: center;
          justify-content: space-between;
          flex-shrink: 0;
        }
        .impact-title {
          font-size: var(--text-xs);
          text-transform: uppercase;
          letter-spacing: 0.06em;
          color: var(--text-secondary);
          font-weight: 500;
        }
        .impact-count-badge {
          font-size: var(--text-xs);
          color: var(--text-dim);
          padding: 2px 6px;
          background: var(--surface-3);
          border: 1px solid var(--border);
          border-radius: 10px;
        }
        .impact-body {
          flex: 1;
          overflow-y: auto;
        }
        .impact-item {
          display: flex;
          flex-direction: column;
          gap: var(--space-1);
          padding: var(--space-3) var(--space-4);
          border-bottom: 1px solid var(--border);
          background: transparent;
          text-align: left;
          width: 100%;
          transition: background var(--dur-fast);
          cursor: pointer;
        }
        .impact-item:hover {
          background: var(--surface-2);
        }
        .impact-item--selected {
          background: var(--surface-2);
          border-left: 2px solid var(--amber);
          padding-left: calc(var(--space-4) - 2px);
        }
        .impact-item-header {
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }
        .impact-sev-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          flex-shrink: 0;
        }
        .impact-sev-dot--critical { background: #dc2626; }
        .impact-sev-dot--changed  { background: #f59e0b; }
        .impact-sev-dot--affected { background: #3b82f6; }
        .impact-file-name {
          font-size: var(--text-xs);
          color: var(--text-secondary);
          flex: 1;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
        .impact-fn-count {
          font-size: var(--text-xs);
          color: var(--text-dim);
          flex-shrink: 0;
        }
        .impact-bar-row {
          padding-left: var(--space-3);
        }
        .impact-bar {
          height: 2px;
          background: var(--border);
          border-radius: 2px;
          overflow: hidden;
        }
        .impact-bar-fill {
          height: 100%;
          border-radius: 2px;
          transition: width var(--dur-slow) var(--ease-out);
        }
        .impact-item-tags {
          display: flex;
          gap: var(--space-1);
          padding-left: var(--space-3);
          flex-wrap: wrap;
        }
        .impact-tag {
          font-size: 10px;
          font-family: var(--font-mono);
          padding: 1px 5px;
          border-radius: var(--radius-sm);
          border: 1px solid;
        }
        .impact-tag--amber {
          color: var(--amber);
          border-color: var(--amber-dim);
          background: var(--amber-glow);
        }
        .impact-tag--red {
          color: #fca5a5;
          border-color: var(--red-dim);
          background: var(--red-glow);
        }
      `}</style>
    </aside>
  )
}
