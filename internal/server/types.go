package server

import "encoding/json"

// AnalysisRequest is the POST /api/analyze body.
type AnalysisRequest struct {
	RepoPath   string `json:"repoPath"`
	BaseBranch string `json:"baseBranch"`
	HeadBranch string `json:"headBranch"`
}

// GraphEventType classifies a streaming WebSocket event.
type GraphEventType string

const (
	EventProgress GraphEventType = "progress"
	EventNode     GraphEventType = "node"
	EventEdge     GraphEventType = "edge"
	EventComplete GraphEventType = "complete"
	EventError    GraphEventType = "error"
)

// GraphEvent is a single streaming message sent over WebSocket.
type GraphEvent struct {
	Type    GraphEventType  `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// ProgressPayload describes the current analysis stage.
type ProgressPayload struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
	Percent int    `json:"percent"`
}

// GraphNodePayload describes a node in the blast graph.
type GraphNodePayload struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	File       string  `json:"file"`
	Line       int     `json:"line"`
	ChangeType string  `json:"changeType"` // "changed" | "affected" | "critical"
	Depth      int     `json:"depth"`
	Score      float64 `json:"score"`
	Signature  string  `json:"signature"`
	CallerCount int    `json:"callerCount"`
	CalleeCount int    `json:"calleeCount"`
}

// GraphEdgePayload describes a directed call edge.
type GraphEdgePayload struct {
	Source    string  `json:"source"`
	Target    string  `json:"target"`
	Frequency int     `json:"frequency"`
	IsHot     bool    `json:"isHot"`
}

// CompletePayload summarizes the completed analysis.
type CompletePayload struct {
	TotalChanged  int     `json:"totalChanged"`
	TotalAffected int     `json:"totalAffected"`
	MaxDepth      int     `json:"maxDepth"`
	TopImpactFile string  `json:"topImpactFile"`
	Duration      float64 `json:"durationMs"`
}

// ErrorPayload carries an error message.
type ErrorPayload struct {
	Message string `json:"message"`
}

// AnalysisResponse is the initial POST response.
type AnalysisResponse struct {
	SessionID string `json:"sessionId"`
}
