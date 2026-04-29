package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rayancheca/callblast/internal/ast"
	"github.com/rayancheca/callblast/internal/diff"
	"github.com/rayancheca/callblast/internal/gitutil"
	"github.com/rayancheca/callblast/internal/graph"
	"github.com/rayancheca/callblast/internal/server"
)

// RunAnalysis executes the full pipeline and streams graph events to the channel.
func RunAnalysis(req server.AnalysisRequest, events chan<- server.GraphEvent) {
	defer close(events)
	start := time.Now()

	emit := func(evtType server.GraphEventType, payload interface{}) {
		raw, err := json.Marshal(payload)
		if err != nil {
			return
		}
		events <- server.GraphEvent{Type: evtType, Payload: raw}
	}

	emitProgress := func(stage, msg string, pct int) {
		emit(server.EventProgress, server.ProgressPayload{Stage: stage, Message: msg, Percent: pct})
	}

	emitErr := func(msg string) {
		emit(server.EventError, server.ErrorPayload{Message: msg})
	}

	// Resolve repo path
	repoPath := req.RepoPath
	if repoPath == "" {
		repoPath = "."
	}
	repoPath, err := filepath.Abs(repoPath)
	if err != nil {
		emitErr("invalid repo path: " + err.Error())
		return
	}
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		emitErr("not a git repository: " + repoPath)
		return
	}

	emitProgress("git", "Identifying changed files…", 5)

	changedFiles, err := gitutil.ChangedFiles(repoPath, req.BaseBranch, req.HeadBranch)
	if err != nil {
		emitErr("git diff failed: " + err.Error())
		return
	}
	if len(changedFiles) == 0 {
		emitErr("no changed files found between " + req.BaseBranch + " and " + req.HeadBranch)
		return
	}

	emitProgress("parse", fmt.Sprintf("Parsing %d changed files…", len(changedFiles)), 15)

	// Extract functions from changed files (base vs head)
	var allChanges []ast.SemanticChange
	for _, relPath := range changedFiles {
		ext := strings.ToLower(filepath.Ext(relPath))
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			continue
		}

		absPath := filepath.Join(repoPath, relPath)

		baseSrc, _ := gitutil.FileAtRef(repoPath, req.BaseBranch, relPath)
		headSrc, _ := os.ReadFile(absPath)

		baseFuncs := extractFunctions(relPath, absPath, baseSrc, ext)
		headFuncs := extractFunctions(relPath, absPath, headSrc, ext)

		changes := diff.DiffFunctions(baseFuncs, headFuncs)
		allChanges = append(allChanges, changes...)
	}

	if len(allChanges) == 0 {
		emitErr("no semantic function changes detected in changed files")
		return
	}

	emitProgress("callgraph", fmt.Sprintf("Building call graph (%d changes detected)…", len(allChanges)), 35)

	// Build call graph from ALL source files
	allFiles, err := gitutil.AllSourceFiles(repoPath)
	if err != nil {
		emitErr("failed to enumerate source files: " + err.Error())
		return
	}

	cg, err := graph.BuildFromFiles(repoPath, allFiles)
	if err != nil {
		emitErr("call graph construction failed: " + err.Error())
		return
	}

	emitProgress("bfs", "Computing blast radius…", 60)

	// BFS reachability
	result := graph.ComputeBlastRadius(cg, allChanges)

	emitProgress("stream", "Streaming results…", 75)

	// Stream nodes
	for _, node := range result.Nodes {
		changeTypeStr := ""
		if node.IsOrigin {
			changeTypeStr = string(node.ChangeType)
		} else if node.IsCritical {
			changeTypeStr = "critical"
		} else {
			changeTypeStr = "affected"
		}

		emit(server.EventNode, server.GraphNodePayload{
			ID:          node.QName,
			Label:       node.Name,
			File:        node.File,
			Line:        node.Line,
			ChangeType:  changeTypeStr,
			Depth:       node.Depth,
			Score:       node.Score,
			Signature:   node.Signature,
			CallerCount: node.CallerCount,
			CalleeCount: node.CalleeCount,
		})
	}

	// Stream edges
	for _, edge := range result.Edges {
		emit(server.EventEdge, server.GraphEdgePayload{
			Source:    edge.Source,
			Target:    edge.Target,
			Frequency: edge.Frequency,
			IsHot:     edge.IsHot,
		})
	}

	elapsed := time.Since(start).Seconds() * 1000
	emit(server.EventComplete, server.CompletePayload{
		TotalChanged:  result.Stats.TotalChanged,
		TotalAffected: result.Stats.TotalAffected,
		MaxDepth:      result.Stats.MaxDepth,
		TopImpactFile: result.Stats.TopFile,
		Duration:      elapsed,
	})
}

func extractFunctions(relPath, absPath string, src []byte, ext string) []ast.FunctionDef {
	if src == nil {
		return nil
	}
	var funcs []ast.FunctionDef
	var err error
	switch ext {
	case ".go":
		funcs, _, err = ast.ExtractGoFile(absPath, src)
	case ".ts", ".tsx", ".js", ".jsx":
		funcs, _, err = ast.ExtractTSFile(absPath, src)
	}
	if err != nil {
		return nil
	}
	return funcs
}
