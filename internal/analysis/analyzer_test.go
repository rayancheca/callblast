package analysis

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rayancheca/callblast/internal/server"
)

func TestRunAnalysis_E2E(t *testing.T) {
	// Set up a temp git repo
	dir := t.TempDir()
	mustRun(t, dir, "git", "init")
	mustRun(t, dir, "git", "config", "user.email", "test@test.com")
	mustRun(t, dir, "git", "config", "user.name", "Test")

	// Write base version
	writeFile(t, filepath.Join(dir, "service.go"), `package main

func Validate(input string) bool {
	return len(input) > 0
}

func Process(input string) string {
	if !Validate(input) {
		return ""
	}
	return input + "_processed"
}
`)
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "base")

	// Write head version (change Validate signature)
	writeFile(t, filepath.Join(dir, "service.go"), `package main

func Validate(input string, minLen int) bool {
	return len(input) >= minLen
}

func Process(input string) string {
	if !Validate(input, 1) {
		return ""
	}
	return input + "_processed_v2"
}
`)
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "update")

	req := server.AnalysisRequest{
		RepoPath:   dir,
		BaseBranch: "HEAD~1",
		HeadBranch: "HEAD",
	}

	events := make(chan server.GraphEvent, 128)
	RunAnalysis(req, events)

	var nodeEvents []server.GraphNodePayload
	var edgeEvents []server.GraphEdgePayload
	var complete *server.CompletePayload
	var errMsg string

	for ev := range events {
		switch ev.Type {
		case server.EventNode:
			var n server.GraphNodePayload
			if err := json.Unmarshal(ev.Payload, &n); err == nil {
				nodeEvents = append(nodeEvents, n)
			}
		case server.EventEdge:
			var e server.GraphEdgePayload
			if err := json.Unmarshal(ev.Payload, &e); err == nil {
				edgeEvents = append(edgeEvents, e)
			}
		case server.EventComplete:
			var c server.CompletePayload
			if err := json.Unmarshal(ev.Payload, &c); err == nil {
				complete = &c
			}
		case server.EventError:
			var e server.ErrorPayload
			if err := json.Unmarshal(ev.Payload, &e); err == nil {
				errMsg = e.Message
			}
		}
	}

	if errMsg != "" {
		t.Fatalf("analysis error: %s", errMsg)
	}

	if complete == nil {
		t.Fatal("no complete event received")
	}

	t.Logf("Analysis complete: %d changed, %d affected, maxDepth=%d, duration=%.1fms",
		complete.TotalChanged, complete.TotalAffected, complete.MaxDepth, complete.Duration)

	if complete.TotalChanged == 0 {
		t.Error("expected at least one changed function")
	}

	// Validate changed in head: Process calls Validate with new signature
	// Validate should be detected as signature_changed
	foundValidate := false
	for _, n := range nodeEvents {
		if n.Label == "Validate" {
			foundValidate = true
			t.Logf("Validate node: changeType=%s depth=%d score=%.2f", n.ChangeType, n.Depth, n.Score)
		}
	}
	if !foundValidate {
		t.Errorf("expected Validate in node events, got: %v", nodeLabels(nodeEvents))
	}

	t.Logf("Nodes: %v", nodeLabels(nodeEvents))
	t.Logf("Edges: %d", len(edgeEvents))
}

func nodeLabels(nodes []server.GraphNodePayload) []string {
	labels := make([]string, len(nodes))
	for i, n := range nodes {
		labels[i] = n.Label
	}
	return labels
}

func mustRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("command %v failed: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
