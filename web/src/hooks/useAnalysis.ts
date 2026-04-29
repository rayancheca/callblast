import { useCallback, useRef, useState } from 'react'
import {
  AnalysisRequest,
  AnalysisState,
  GraphEdgePayload,
  GraphEvent,
  GraphNodePayload,
} from '../types'

const API_BASE = '/api'
const WS_BASE = window.location.protocol === 'https:' ? 'wss://' : 'ws://'

export function useAnalysis() {
  const [state, setState] = useState<AnalysisState>({
    status: 'idle',
    progress: null,
    nodes: new Map(),
    edges: [],
    summary: null,
    error: null,
  })
  const wsRef = useRef<WebSocket | null>(null)

  const reset = useCallback(() => {
    wsRef.current?.close()
    setState({
      status: 'idle',
      progress: null,
      nodes: new Map(),
      edges: [],
      summary: null,
      error: null,
    })
  }, [])

  const run = useCallback(async (req: AnalysisRequest) => {
    wsRef.current?.close()

    setState({
      status: 'running',
      progress: { stage: 'init', message: 'Starting analysis…', percent: 0 },
      nodes: new Map(),
      edges: [],
      summary: null,
      error: null,
    })

    let sessionId: string
    try {
      const res = await fetch(`${API_BASE}/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({ message: res.statusText }))
        throw new Error(body.message ?? res.statusText)
      }
      const data = await res.json()
      sessionId = data.sessionId
    } catch (err) {
      setState(s => ({ ...s, status: 'error', error: String(err) }))
      return
    }

    const wsUrl = `${WS_BASE}${window.location.host}/ws?session=${sessionId}`
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onmessage = (ev: MessageEvent<string>) => {
      const event: GraphEvent = JSON.parse(ev.data)

      setState(s => {
        switch (event.type) {
          case 'progress': {
            const payload = event.payload as typeof s.progress
            return { ...s, progress: payload }
          }
          case 'node': {
            const node = event.payload as GraphNodePayload
            const nodes = new Map(s.nodes)
            nodes.set(node.id, node)
            return { ...s, nodes }
          }
          case 'edge': {
            const edge = event.payload as GraphEdgePayload
            return { ...s, edges: [...s.edges, edge] }
          }
          case 'complete': {
            return { ...s, status: 'complete', summary: event.payload as typeof s.summary }
          }
          case 'error': {
            const err = event.payload as { message: string }
            return { ...s, status: 'error', error: err.message }
          }
          default:
            return s
        }
      })
    }

    ws.onerror = () => {
      setState(s => ({ ...s, status: 'error', error: 'WebSocket connection failed' }))
    }

    ws.onclose = () => {
      setState(s => {
        if (s.status === 'running') {
          return { ...s, status: 'error', error: 'Connection closed unexpectedly' }
        }
        return s
      })
    }
  }, [])

  return { state, run, reset }
}
