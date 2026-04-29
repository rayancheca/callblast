import { GraphNodePayload, GraphEdgePayload } from '../types'

interface NodeDetailProps {
  node: GraphNodePayload | null
  nodes: Map<string, GraphNodePayload>
  edges: GraphEdgePayload[]
  onClose: () => void
}

const CHANGE_TYPE_LABEL: Record<string, { label: string; color: string }> = {
  added: { label: 'ADDED', color: '#16a34a' },
  removed: { label: 'REMOVED', color: '#dc2626' },
  signature_changed: { label: 'SIG CHANGED', color: '#f59e0b' },
  body_changed: { label: 'BODY CHANGED', color: '#f59e0b' },
  renamed: { label: 'RENAMED', color: '#fb923c' },
  critical: { label: 'CRITICAL PATH', color: '#dc2626' },
  affected: { label: 'AFFECTED', color: '#3b82f6' },
}

export default function NodeDetail({ node, nodes, edges, onClose }: NodeDetailProps) {
  if (!node) return null

  const callers = edges
    .filter(e => e.target === node.id)
    .map(e => ({ edge: e, node: nodes.get(e.source) }))
    .filter(x => x.node)

  const callees = edges
    .filter(e => e.source === node.id)
    .map(e => ({ edge: e, node: nodes.get(e.target) }))
    .filter(x => x.node)

  const typeInfo = CHANGE_TYPE_LABEL[node.changeType] ?? { label: node.changeType.toUpperCase(), color: '#737373' }
  const depthLabel = node.depth === 0 ? 'origin' : `depth ${node.depth}`
  const scorePercent = Math.round(node.score * 100)

  return (
    <aside className="detail-panel" aria-label="Node detail panel">
      <div className="detail-header">
        <div className="detail-title-row">
          <span className="detail-func-name mono">{node.label}</span>
          <button className="detail-close" onClick={onClose} aria-label="Close detail panel">
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <path d="M1 1l10 10M11 1L1 11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </button>
        </div>
        <div className="detail-badges">
          <span
            className="detail-badge"
            style={{ color: typeInfo.color, borderColor: typeInfo.color, background: `${typeInfo.color}18` }}
          >
            {typeInfo.label}
          </span>
          <span className="detail-badge-neutral mono">{depthLabel}</span>
          <span className="detail-badge-neutral mono">{scorePercent}% impact</span>
        </div>
      </div>

      <div className="detail-body">
        <div className="detail-section">
          <div className="detail-section-title">Location</div>
          <div className="detail-location mono">
            <span className="detail-file">{node.file}</span>
            {node.line > 0 && <span className="detail-line">:{node.line}</span>}
          </div>
        </div>

        {node.signature && (
          <div className="detail-section">
            <div className="detail-section-title">Signature</div>
            <pre className="detail-signature mono">{node.signature}</pre>
          </div>
        )}

        <div className="detail-section">
          <div className="detail-section-title">Impact score</div>
          <div className="detail-score-bar-wrap">
            <div className="detail-score-bar">
              <div
                className="detail-score-fill"
                style={{
                  width: `${scorePercent}%`,
                  background: node.changeType === 'critical' || node.depth === 0 ? '#dc2626' : '#f59e0b',
                }}
              />
            </div>
            <span className="detail-score-label mono">{scorePercent}%</span>
          </div>
        </div>

        {callers.length > 0 && (
          <div className="detail-section">
            <div className="detail-section-title">
              Called by <span className="detail-count">{callers.length}</span>
            </div>
            <div className="detail-func-list">
              {callers.map(({ edge, node: callerNode }) => (
                <div key={edge.source} className="detail-func-item">
                  <span className="detail-func-dot" style={{ background: '#3b82f6' }} />
                  <span className="detail-func-item-name mono">{callerNode!.label}</span>
                  {edge.frequency > 1 && (
                    <span className="detail-func-freq">×{edge.frequency}</span>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {callees.length > 0 && (
          <div className="detail-section">
            <div className="detail-section-title">
              Calls <span className="detail-count">{callees.length}</span>
            </div>
            <div className="detail-func-list">
              {callees.map(({ edge, node: calleeNode }) => (
                <div key={edge.target} className="detail-func-item">
                  <span className="detail-func-dot" style={{ background: '#f59e0b' }} />
                  <span className="detail-func-item-name mono">{calleeNode!.label}</span>
                  {edge.frequency > 1 && (
                    <span className="detail-func-freq">×{edge.frequency}</span>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="detail-section">
          <div className="detail-section-title">Connectivity</div>
          <div className="detail-meta-row">
            <div className="detail-meta-item">
              <span className="detail-meta-value mono">{node.callerCount}</span>
              <span className="detail-meta-label">callers</span>
            </div>
            <div className="detail-meta-item">
              <span className="detail-meta-value mono">{node.calleeCount}</span>
              <span className="detail-meta-label">callees</span>
            </div>
          </div>
        </div>
      </div>

      <style>{`
        .detail-panel {
          width: var(--detail-w);
          flex-shrink: 0;
          background: var(--surface);
          border-left: 1px solid var(--border);
          display: flex;
          flex-direction: column;
          overflow: hidden;
          animation: slideIn var(--dur-normal) var(--ease-out);
        }
        @keyframes slideIn {
          from { transform: translateX(var(--detail-w)); opacity: 0; }
          to { transform: translateX(0); opacity: 1; }
        }
        .detail-header {
          padding: var(--space-4) var(--space-4) var(--space-3);
          border-bottom: 1px solid var(--border);
          background: var(--surface-2);
          flex-shrink: 0;
        }
        .detail-title-row {
          display: flex;
          align-items: flex-start;
          justify-content: space-between;
          gap: var(--space-2);
          margin-bottom: var(--space-2);
        }
        .detail-func-name {
          font-size: var(--text-base);
          font-weight: 500;
          color: var(--text-primary);
          word-break: break-all;
          line-height: 1.4;
        }
        .detail-close {
          flex-shrink: 0;
          display: flex;
          align-items: center;
          justify-content: center;
          width: 24px;
          height: 24px;
          background: transparent;
          border: 1px solid var(--border);
          border-radius: var(--radius-sm);
          color: var(--text-secondary);
          transition: color var(--dur-fast), border-color var(--dur-fast);
        }
        .detail-close:hover {
          color: var(--text-primary);
          border-color: var(--border-2);
        }
        .detail-badges {
          display: flex;
          flex-wrap: wrap;
          gap: var(--space-1);
        }
        .detail-badge {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          font-weight: 500;
          padding: 2px 6px;
          border: 1px solid;
          border-radius: var(--radius-sm);
          letter-spacing: 0.02em;
        }
        .detail-badge-neutral {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          padding: 2px 6px;
          border: 1px solid var(--border);
          border-radius: var(--radius-sm);
          color: var(--text-secondary);
          background: var(--surface-2);
        }
        .detail-body {
          flex: 1;
          overflow-y: auto;
          padding: var(--space-3) 0;
        }
        .detail-section {
          padding: var(--space-3) var(--space-4);
          border-bottom: 1px solid var(--border);
        }
        .detail-section:last-child {
          border-bottom: none;
        }
        .detail-section-title {
          font-size: var(--text-xs);
          letter-spacing: 0.06em;
          text-transform: uppercase;
          color: var(--text-dim);
          font-weight: 500;
          margin-bottom: var(--space-2);
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }
        .detail-count {
          font-family: var(--font-mono);
          padding: 1px 5px;
          background: var(--surface-2);
          border: 1px solid var(--border);
          border-radius: 10px;
          font-size: var(--text-xs);
          color: var(--text-secondary);
          text-transform: none;
          letter-spacing: 0;
        }
        .detail-location {
          font-size: var(--text-sm);
          color: var(--text-secondary);
          word-break: break-all;
        }
        .detail-file { color: var(--text-secondary); }
        .detail-line { color: var(--text-dim); }
        .detail-signature {
          font-size: var(--text-xs);
          color: var(--text-secondary);
          white-space: pre-wrap;
          word-break: break-all;
          padding: var(--space-2) var(--space-3);
          background: var(--surface-2);
          border: 1px solid var(--border);
          border-radius: var(--radius-sm);
          line-height: 1.6;
        }
        .detail-score-bar-wrap {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }
        .detail-score-bar {
          flex: 1;
          height: 4px;
          background: var(--border);
          border-radius: 2px;
          overflow: hidden;
        }
        .detail-score-fill {
          height: 100%;
          border-radius: 2px;
          transition: width var(--dur-slow) var(--ease-out);
        }
        .detail-score-label {
          font-size: var(--text-xs);
          color: var(--text-secondary);
          width: 28px;
          text-align: right;
        }
        .detail-func-list {
          display: flex;
          flex-direction: column;
          gap: 2px;
        }
        .detail-func-item {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          padding: 4px 6px;
          border-radius: var(--radius-sm);
          transition: background var(--dur-fast);
        }
        .detail-func-item:hover {
          background: var(--surface-2);
        }
        .detail-func-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          flex-shrink: 0;
        }
        .detail-func-item-name {
          font-size: var(--text-sm);
          color: var(--text-secondary);
          flex: 1;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
        .detail-func-freq {
          font-size: var(--text-xs);
          color: var(--text-dim);
          font-family: var(--font-mono);
        }
        .detail-meta-row {
          display: flex;
          gap: var(--space-6);
        }
        .detail-meta-item {
          display: flex;
          flex-direction: column;
          gap: 2px;
        }
        .detail-meta-value {
          font-size: var(--text-md);
          font-weight: 500;
          color: var(--text-primary);
        }
        .detail-meta-label {
          font-size: var(--text-xs);
          color: var(--text-dim);
          text-transform: uppercase;
          letter-spacing: 0.04em;
        }
      `}</style>
    </aside>
  )
}
