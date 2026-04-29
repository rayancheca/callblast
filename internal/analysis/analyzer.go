package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
// It closes the events channel when complete.
func RunAnalysis(ctx context.Context, req server.AnalysisRequest, events chan<- server.GraphEvent) {
	defer close(events)
	start := time.Now()

	emit := func(evtType server.GraphEventType, payload interface{}) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return
		}
		select {
		case events <- server.GraphEvent{Type: evtType, Payload: raw}:
		case <-ctx.Done():
		}
	}

	emitProgress := func(stage, msg string, pct int) {
		emit(server.EventProgress, server.ProgressPayload{Stage: stage, Message: msg, Percent: pct})
	}

	emitErr := func(msg string) {
		emit(server.EventError, server.ErrorPayload{Message: msg})
	}

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

	if err := ctx.Err(); err != nil {
		emitErr("analysis cancelled")
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

	if err := ctx.Err(); err != nil {
		return
	}

	// Extract functions from changed files (base vs head)
	var allChanges []ast.SemanticChange
	for _, relPath := range changedFiles {
		ext := strings.ToLower(filepath.Ext(relPath))
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			continue
		}

		absPath := filepath.Join(repoPath, relPath)

		baseSrc, baseErr := gitutil.FileAtRef(repoPath, req.BaseBranch, relPath)
		if baseErr != nil {
			log.Printf("warn: could not read %s at %s: %v", relPath, req.BaseBranch, baseErr)
		}

		headSrc, headErr := os.ReadFile(absPath)
		if headErr != nil && !errors.Is(headErr, os.ErrNotExist) {
			log.Printf("warn: could not read %s: %v", absPath, headErr)
		}

		baseFuncs := extractFunctions(absPath, baseSrc, ext)
		headFuncs := extractFunctions(absPath, headSrc, ext)

		changes := diff.DiffFunctions(baseFuncs, headFuncs)
		allChanges = append(allChanges, changes...)
	}

	if len(allChanges) == 0 {
		emitErr("no semantic function changes detected in changed files")
		return
	}

	emitProgress("callgraph", fmt.Sprintf("Building call graph (%d changes detected)…", len(allChanges)), 35)

	if err := ctx.Err(); err != nil {
		return
	}

	allFiles, err := gitutil.AllSourceFiles(repoPath)
	if err != nil {
		emitErr("failed to enumerate source files: " + err.Error())
		return
	}

	cg, buildErrs := graph.BuildFromFiles(repoPath, allFiles)
	if len(buildErrs) > 0 {
		log.Printf("warn: %d files failed to parse during call graph construction", len(buildErrs))
		for _, be := range buildErrs {
			log.Printf("  %v", be)
		}
	}
	if cg == nil {
		emitErr("call graph construction failed")
		return
	}

	emitProgress("bfs", "Computing blast radius…", 60)

	result := graph.ComputeBlastRadius(cg, allChanges)

	emitProgress("stream", "Streaming results…", 75)

	for _, node := range result.Nodes {
		if err := ctx.Err(); err != nil {
			return
		}
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

func extractFunctions(absPath string, src []byte, ext string) []ast.FunctionDef {
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
		log.Printf("warn: failed to extract functions from %s: %v", absPath, err)
		return nil
	}
	return funcs
}
