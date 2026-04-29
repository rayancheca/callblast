import { useState } from 'react'
import { useAnalysis } from './hooks/useAnalysis'
import { GraphNodePayload } from './types'
import Header from './components/Header'
import AnalysisForm from './components/AnalysisForm'
import BlastGraph from './components/BlastGraph'
import NodeDetail from './components/NodeDetail'
import ImpactList from './components/ImpactList'

export default function App() {
  const { state, run, reset } = useAnalysis()
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [selectedFile, setSelectedFile] = useState<string | null>(null)

  const isRunning = state.status === 'running'
  const hasGraph = state.nodes.size > 0
  const selectedNode: GraphNodePayload | null = selectedNodeId ? (state.nodes.get(selectedNodeId) ?? null) : null

  function handleSelectNode(id: string | null) {
    setSelectedNodeId(id)
    if (id) setSelectedFile(null)
  }

  function handleSelectFile(file: string | null) {
    setSelectedFile(file)
    if (file) setSelectedNodeId(null)
  }

  function handleReset() {
    reset()
    setSelectedNodeId(null)
    setSelectedFile(null)
  }

  return (
    <div className="app">
      <Header
        summary={state.summary}
        onReset={handleReset}
        isRunning={isRunning}
      />

      <div className="app-body">
        {!hasGraph && state.status !== 'running' && (
          <AnalysisForm
            onSubmit={run}
            isRunning={isRunning}
            progress={state.progress}
            error={state.error}
          />
        )}

        {(hasGraph || isRunning) && (
          <>
            {hasGraph && (
              <ImpactList
                nodes={state.nodes}
                selectedFile={selectedFile}
                onSelectFile={handleSelectFile}
              />
            )}

            <BlastGraph
              nodes={state.nodes}
              edges={state.edges}
              selectedId={selectedNodeId}
              onSelectNode={handleSelectNode}
              isRunning={isRunning}
            />

            {selectedNode && (
              <NodeDetail
                node={selectedNode}
                nodes={state.nodes}
                edges={state.edges}
                onClose={() => setSelectedNodeId(null)}
              />
            )}
          </>
        )}

        {isRunning && !hasGraph && (
          <div className="app-loading">
            <div className="loading-spinner" aria-label="Analyzing repository…" />
            <p className="loading-stage">{state.progress?.stage ?? 'initializing'}</p>
            <p className="loading-msg">{state.progress?.message ?? 'Starting analysis…'}</p>
            <div className="loading-bar-wrap">
              <div className="loading-bar">
                <div
                  className="loading-bar-fill"
                  style={{ width: `${state.progress?.percent ?? 0}%` }}
                />
              </div>
              <span className="loading-pct mono">{state.progress?.percent ?? 0}%</span>
            </div>
          </div>
        )}

        {state.status === 'error' && !hasGraph && (
          <div className="app-error">
            <div className="error-icon" aria-hidden="true">
              <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
                <circle cx="16" cy="16" r="14" stroke="#dc2626" strokeWidth="1.5" />
                <path d="M16 9v9M16 21v2" stroke="#dc2626" strokeWidth="2" strokeLinecap="round" />
              </svg>
            </div>
            <p className="error-msg">{state.error}</p>
            <button className="error-retry" onClick={handleReset}>
              Try again
            </button>
          </div>
        )}
      </div>

      <style>{`
        .app {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg);
        }
        .app-body {
          flex: 1;
          display: flex;
          overflow: hidden;
          position: relative;
        }
        .app-loading {
          flex: 1;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: var(--space-4);
        }
        .loading-spinner {
          width: 36px;
          height: 36px;
          border: 2px solid var(--border-2);
          border-top-color: var(--amber);
          border-radius: 50%;
          animation: spin 0.9s linear infinite;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        .loading-stage {
          font-size: var(--text-xs);
          color: var(--text-dim);
          text-transform: uppercase;
          letter-spacing: 0.06em;
          font-family: var(--font-mono);
        }
        .loading-msg {
          font-size: var(--text-base);
          color: var(--text-secondary);
        }
        .loading-bar-wrap {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          width: 240px;
        }
        .loading-bar {
          flex: 1;
          height: 3px;
          background: var(--border);
          border-radius: 2px;
          overflow: hidden;
        }
        .loading-bar-fill {
          height: 100%;
          background: var(--amber);
          border-radius: 2px;
          transition: width 300ms var(--ease-out);
        }
        .loading-pct {
          font-size: var(--text-xs);
          color: var(--text-dim);
          width: 28px;
          text-align: right;
        }
        .app-error {
          flex: 1;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: var(--space-4);
        }
        .error-msg {
          font-size: var(--text-base);
          color: var(--text-secondary);
          max-width: 400px;
          text-align: center;
          line-height: 1.6;
        }
        .error-retry {
          padding: 8px var(--space-5);
          background: transparent;
          border: 1px solid var(--border-2);
          border-radius: var(--radius-md);
          color: var(--text-secondary);
          font-size: var(--text-sm);
          transition: color var(--dur-fast), border-color var(--dur-fast);
        }
        .error-retry:hover {
          color: var(--text-primary);
          border-color: var(--amber);
        }
      `}</style>
    </div>
  )
}
