export type GraphEventType = 'progress' | 'node' | 'edge' | 'complete' | 'error'

export interface ProgressPayload {
  stage: string
  message: string
  percent: number
}

export interface GraphNodePayload {
  id: string
  label: string
  file: string
  line: number
  changeType: string // 'added' | 'removed' | 'signature_changed' | 'body_changed' | 'renamed' | 'critical' | 'affected'
  depth: number
  score: number
  signature: string
  callerCount: number
  calleeCount: number
}

export interface GraphEdgePayload {
  source: string
  target: string
  frequency: number
  isHot: boolean
}

export interface CompletePayload {
  totalChanged: number
  totalAffected: number
  maxDepth: number
  topImpactFile: string
  durationMs: number
}

export interface ErrorPayload {
  message: string
}

export interface GraphEvent {
  type: GraphEventType
  payload: ProgressPayload | GraphNodePayload | GraphEdgePayload | CompletePayload | ErrorPayload
}

export interface AnalysisRequest {
  repoPath: string
  baseBranch: string
  headBranch: string
}

// D3 simulation node — extends GraphNodePayload with simulation coordinates
export interface SimNode extends GraphNodePayload {
  x?: number
  y?: number
  vx?: number
  vy?: number
  fx?: number | null
  fy?: number | null
}

// D3 simulation link
export interface SimLink extends GraphEdgePayload {
  sourceNode?: SimNode
  targetNode?: SimNode
}

export type AnalysisStatus = 'idle' | 'running' | 'complete' | 'error'

export interface AnalysisState {
  status: AnalysisStatus
  progress: ProgressPayload | null
  nodes: Map<string, GraphNodePayload>
  edges: GraphEdgePayload[]
  summary: CompletePayload | null
  error: string | null
}
