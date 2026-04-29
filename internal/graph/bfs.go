package graph

import (
	"math"
	"strings"

	"github.com/rayancheca/callblast/internal/ast"
)

// BlastNode is a function node in the blast radius result.
type BlastNode struct {
	QName       string
	Name        string
	File        string
	Line        int
	Depth       int
	Score       float64 // impact score [0,1]
	IsCritical  bool    // reachable from ≥2 changed functions
	IsOrigin    bool    // is a directly changed function
	ChangeType  ast.ChangeType
	CallerCount int
	CalleeCount int
	Signature   string
}

// BlastEdge is a call edge in the blast radius result.
type BlastEdge struct {
	Source    string // caller qname
	Target    string // callee qname
	Frequency int
	IsHot     bool // both nodes in blast radius
}

// BlastResult is the full output of the BFS reachability analysis.
type BlastResult struct {
	Nodes    []BlastNode
	Edges    []BlastEdge
	MaxDepth int
	Stats    BlastStats
}

// BlastStats summarizes the blast radius.
type BlastStats struct {
	TotalChanged  int
	TotalAffected int
	CriticalCount int
	MaxDepth      int
	TopFile       string
}

// ComputeBlastRadius runs BFS from all changed functions through the reverse call graph,
// collecting all transitively affected callers and scoring them.
func ComputeBlastRadius(cg *CallGraph, changes []ast.SemanticChange) *BlastResult {
	if len(changes) == 0 {
		return &BlastResult{}
	}

	// Map from qname → how many distinct changed functions reach it
	reachCount := make(map[string]int)
	// Map from qname → minimum depth from any origin
	depth := make(map[string]int)
	// Track which change type each origin has
	originChangeType := make(map[string]ast.ChangeType)

	for _, ch := range changes {
		qname := ch.Func.QualifiedName()
		originChangeType[qname] = ch.Type
		bfsFromOrigin(cg, qname, reachCount, depth)
	}

	// Build node set
	nodeSet := make(map[string]*BlastNode)

	// Add origin nodes
	for _, ch := range changes {
		qname := ch.Func.QualifiedName()
		fd := ch.Func
		node := &BlastNode{
			QName:       qname,
			Name:        fd.Name,
			File:        shortFile(fd.File),
			Line:        fd.Line,
			Depth:       0,
			Score:       1.0,
			IsOrigin:    true,
			ChangeType:  ch.Type,
			CallerCount: len(cg.Callers(qname)),
			CalleeCount: len(cg.Callees(qname)),
			Signature:   fd.Signature,
		}
		nodeSet[qname] = node
	}

	// Add affected nodes
	maxDepth := 0
	for qname, d := range depth {
		if _, isOrigin := originChangeType[qname]; isOrigin {
			continue
		}
		if d > maxDepth {
			maxDepth = d
		}
		score := math.Max(0.05, 1.0/math.Pow(float64(d+1), 0.7))
		isCritical := reachCount[qname] >= 2

		fd, ok := cg.FuncDef(qname)
		name := lastName(qname)
		file := shortFile(qname)
		sig := ""
		line := 0
		if ok {
			name = fd.Name
			file = shortFile(fd.File)
			sig = fd.Signature
			line = fd.Line
		}

		nodeSet[qname] = &BlastNode{
			QName:       qname,
			Name:        name,
			File:        file,
			Line:        line,
			Depth:       d,
			Score:       score,
			IsCritical:  isCritical,
			IsOrigin:    false,
			CallerCount: len(cg.Callers(qname)),
			CalleeCount: len(cg.Callees(qname)),
			Signature:   sig,
		}
	}

	// Build edges between nodes in the blast radius
	var edges []BlastEdge
	seen := make(map[string]bool)
	for qname := range nodeSet {
		for _, callee := range cg.Callees(qname) {
			if _, inSet := nodeSet[callee]; !inSet {
				continue
			}
			key := qname + "→" + callee
			if seen[key] {
				continue
			}
			seen[key] = true
			freq := cg.CallFrequency(qname, callee)
			edges = append(edges, BlastEdge{
				Source:    qname,
				Target:    callee,
				Frequency: freq,
				IsHot:     nodeSet[qname].IsCritical || nodeSet[callee].IsCritical,
			})
		}
	}

	// Flatten nodes
	nodes := make([]BlastNode, 0, len(nodeSet))
	criticalCount := 0
	fileCount := make(map[string]int)
	for _, n := range nodeSet {
		nodes = append(nodes, *n)
		if n.IsCritical {
			criticalCount++
		}
		fileCount[n.File]++
	}

	topFile := ""
	topCount := 0
	for f, c := range fileCount {
		if c > topCount {
			topCount = c
			topFile = f
		}
	}

	return &BlastResult{
		Nodes:    nodes,
		Edges:    edges,
		MaxDepth: maxDepth,
		Stats: BlastStats{
			TotalChanged:  len(changes),
			TotalAffected: len(nodeSet) - len(changes),
			CriticalCount: criticalCount,
			MaxDepth:      maxDepth,
			TopFile:       topFile,
		},
	}
}

// bfsFromOrigin runs BFS backwards through the call graph from origin,
// updating reachCount and depth maps.
func bfsFromOrigin(cg *CallGraph, origin string, reachCount, depth map[string]int) {
	queue := []string{origin}
	visited := map[string]bool{origin: true}
	currentDepth := 0

	for len(queue) > 0 {
		nextQueue := []string{}
		for _, node := range queue {
			if existingDepth, ok := depth[node]; !ok || currentDepth < existingDepth {
				depth[node] = currentDepth
			}
			reachCount[node]++

			for _, caller := range cg.Callers(node) {
				if !visited[caller] {
					visited[caller] = true
					nextQueue = append(nextQueue, caller)
				}
			}
		}
		queue = nextQueue
		currentDepth++

		// Safety limit: don't traverse more than 10 hops
		if currentDepth > 10 {
			break
		}
	}
}

func lastName(qname string) string {
	idx := strings.LastIndex(qname, "::")
	if idx < 0 {
		return qname
	}
	return qname[idx+2:]
}

func shortFile(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}
