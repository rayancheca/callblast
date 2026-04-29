package graph

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rayancheca/callblast/internal/ast"
	goast "github.com/rayancheca/callblast/internal/ast"
)

// CallGraph is a directed graph of function calls: caller → []callees.
type CallGraph struct {
	// Forward: caller qname → set of callee names
	forward map[string]map[string]int // caller → callee → call count
	// Reverse: callee name → set of caller qnames
	reverse map[string]map[string]int
	// Function metadata keyed by qname
	funcs map[string]goast.FunctionDef
	// Short name → list of qualified names (for resolving unqualified calls)
	shortIndex map[string][]string
}

// NewCallGraph creates an empty call graph.
func NewCallGraph() *CallGraph {
	return &CallGraph{
		forward:    make(map[string]map[string]int),
		reverse:    make(map[string]map[string]int),
		funcs:      make(map[string]goast.FunctionDef),
		shortIndex: make(map[string][]string),
	}
}

// AddFunction registers a function definition.
func (cg *CallGraph) AddFunction(fd goast.FunctionDef) {
	qname := fd.QualifiedName()
	cg.funcs[qname] = fd
	cg.shortIndex[fd.Name] = append(cg.shortIndex[fd.Name], qname)
	if _, ok := cg.forward[qname]; !ok {
		cg.forward[qname] = make(map[string]int)
	}
}

// AddCall registers a call edge. callee is a short or partial name.
func (cg *CallGraph) AddCall(site goast.CallSite) {
	caller := site.CallerFunc
	callee := site.CalleeFunc

	// Resolve callee to a qualified name if possible
	resolved := cg.resolve(callee, caller)
	if resolved == "" {
		return
	}

	if _, ok := cg.forward[caller]; !ok {
		cg.forward[caller] = make(map[string]int)
	}
	cg.forward[caller][resolved]++

	if _, ok := cg.reverse[resolved]; !ok {
		cg.reverse[resolved] = make(map[string]int)
	}
	cg.reverse[resolved][caller]++
}

// resolve maps a short callee name to a qualified name.
func (cg *CallGraph) resolve(callee, caller string) string {
	if qnames, ok := cg.shortIndex[callee]; ok {
		if len(qnames) == 1 {
			return qnames[0]
		}
		// Prefer same-file resolution
		callerFile := fileFromQName(caller)
		for _, qn := range qnames {
			if strings.HasPrefix(qn, callerFile+"::") {
				return qn
			}
		}
		return qnames[0]
	}
	return ""
}

// Callers returns all qualified caller names for a given callee qname.
func (cg *CallGraph) Callers(calleeQName string) []string {
	callers := cg.reverse[calleeQName]
	result := make([]string, 0, len(callers))
	for c := range callers {
		result = append(result, c)
	}
	return result
}

// Callees returns all qualified callee names for a given caller qname.
func (cg *CallGraph) Callees(callerQName string) []string {
	callees := cg.forward[callerQName]
	result := make([]string, 0, len(callees))
	for c := range callees {
		result = append(result, c)
	}
	return result
}

// CallFrequency returns how many times caller calls callee.
func (cg *CallGraph) CallFrequency(caller, callee string) int {
	return cg.forward[caller][callee]
}

// FuncDef returns the FunctionDef for a qname.
func (cg *CallGraph) FuncDef(qname string) (goast.FunctionDef, bool) {
	fd, ok := cg.funcs[qname]
	return fd, ok
}

// AllFunctions returns all registered function qnames.
func (cg *CallGraph) AllFunctions() []string {
	result := make([]string, 0, len(cg.funcs))
	for k := range cg.funcs {
		result = append(result, k)
	}
	return result
}

// BuildFromFiles builds a call graph from a set of source files.
func BuildFromFiles(repoPath string, filePaths []string) (*CallGraph, error) {
	cg := NewCallGraph()
	var allCalls []ast.CallSite

	for _, relPath := range filePaths {
		absPath := filepath.Join(repoPath, relPath)
		src, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		var funcs []ast.FunctionDef
		var calls []ast.CallSite

		ext := strings.ToLower(filepath.Ext(relPath))
		switch ext {
		case ".go":
			funcs, calls, err = ast.ExtractGoFile(absPath, src)
		case ".ts", ".tsx", ".js", ".jsx":
			funcs, calls, err = ast.ExtractTSFile(absPath, src)
		}
		if err != nil {
			continue
		}

		for _, fd := range funcs {
			cg.AddFunction(fd)
		}
		allCalls = append(allCalls, calls...)
	}

	// Add calls after all functions are registered (for resolution)
	for _, call := range allCalls {
		cg.AddCall(call)
	}

	return cg, nil
}

func fileFromQName(qname string) string {
	idx := strings.Index(qname, "::")
	if idx < 0 {
		return qname
	}
	return qname[:idx]
}
