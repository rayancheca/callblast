package graph

import (
	"testing"

	"github.com/rayancheca/callblast/internal/ast"
)

func buildTestGraph() *CallGraph {
	cg := NewCallGraph()

	fns := []ast.FunctionDef{
		{Name: "handler", File: "/web.go"},
		{Name: "processRequest", File: "/logic.go"},
		{Name: "validateInput", File: "/util.go"},
		{Name: "parseInput", File: "/util.go"},
		{Name: "logError", File: "/log.go"},
	}
	for _, f := range fns {
		cg.AddFunction(f)
	}

	// handler → processRequest → validateInput
	// handler → processRequest → parseInput
	// processRequest → logError
	calls := []ast.CallSite{
		{CallerFunc: "/web.go::handler", CalleeFunc: "processRequest", File: "/web.go"},
		{CallerFunc: "/logic.go::processRequest", CalleeFunc: "validateInput", File: "/logic.go"},
		{CallerFunc: "/logic.go::processRequest", CalleeFunc: "parseInput", File: "/logic.go"},
		{CallerFunc: "/logic.go::processRequest", CalleeFunc: "logError", File: "/logic.go"},
	}
	for _, c := range calls {
		cg.AddCall(c)
	}
	return cg
}

func TestBFS_OriginOnly(t *testing.T) {
	cg := buildTestGraph()
	changes := []ast.SemanticChange{
		{
			Func: ast.FunctionDef{Name: "validateInput", File: "/util.go"},
			Type: ast.ChangeBody,
		},
	}
	result := ComputeBlastRadius(cg, changes)

	if len(result.Nodes) == 0 {
		t.Fatal("expected non-empty blast radius")
	}

	foundOrigin := false
	for _, n := range result.Nodes {
		if n.IsOrigin {
			foundOrigin = true
		}
	}
	if !foundOrigin {
		t.Error("expected origin node in result")
	}
}

func TestBFS_TransitiveClosure(t *testing.T) {
	cg := buildTestGraph()

	// validateInput is called by processRequest which is called by handler
	// Changing validateInput should affect processRequest and handler
	changes := []ast.SemanticChange{
		{
			Func: ast.FunctionDef{Name: "validateInput", File: "/util.go"},
			Type: ast.ChangeSignature,
		},
	}
	result := ComputeBlastRadius(cg, changes)

	nodeNames := make(map[string]bool)
	for _, n := range result.Nodes {
		nodeNames[n.Name] = true
	}

	if !nodeNames["processRequest"] {
		t.Error("expected processRequest to be in blast radius (it calls validateInput)")
	}
	if !nodeNames["handler"] {
		t.Error("expected handler to be in blast radius (it calls processRequest)")
	}
}

func TestBFS_CriticalPath(t *testing.T) {
	cg := buildTestGraph()

	// Change both validateInput and parseInput — processRequest calls both
	// so processRequest should be critical (reachable from 2 origins)
	changes := []ast.SemanticChange{
		{Func: ast.FunctionDef{Name: "validateInput", File: "/util.go"}, Type: ast.ChangeBody},
		{Func: ast.FunctionDef{Name: "parseInput", File: "/util.go"}, Type: ast.ChangeBody},
	}
	result := ComputeBlastRadius(cg, changes)

	for _, n := range result.Nodes {
		if n.Name == "processRequest" && !n.IsCritical {
			t.Error("processRequest should be critical (reachable from both changed functions)")
		}
	}
}

func TestBFS_EmptyChanges(t *testing.T) {
	cg := buildTestGraph()
	result := ComputeBlastRadius(cg, nil)
	if len(result.Nodes) != 0 {
		t.Errorf("expected empty result for no changes, got %d nodes", len(result.Nodes))
	}
}
