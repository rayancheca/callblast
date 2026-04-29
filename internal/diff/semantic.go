package diff

import (
	"strings"

	"github.com/rayancheca/callblast/internal/ast"
)

// DiffFunctions compares two snapshots of function definitions (base vs head)
// and returns a list of semantic changes.
func DiffFunctions(base, head []ast.FunctionDef) []ast.SemanticChange {
	baseMap := indexByQName(base)
	headMap := indexByQName(head)

	var changes []ast.SemanticChange

	// Find removed and modified functions
	for qname, baseFn := range baseMap {
		headFn, exists := headMap[qname]
		if !exists {
			// Check if it was renamed (same signature, different name)
			if renamed := findRename(baseFn, headMap, baseMap); renamed != nil {
				changes = append(changes, ast.SemanticChange{
					Func:    *renamed,
					Type:    ast.ChangeRenamed,
					OldFunc: &baseFn,
				})
			} else {
				changes = append(changes, ast.SemanticChange{
					Func: baseFn,
					Type: ast.ChangeRemoved,
				})
			}
			continue
		}

		// Both exist — check for changes
		if baseFn.Signature != headFn.Signature {
			changes = append(changes, ast.SemanticChange{
				Func:    headFn,
				Type:    ast.ChangeSignature,
				OldFunc: &baseFn,
			})
		} else if baseFn.BodyHash != headFn.BodyHash {
			changes = append(changes, ast.SemanticChange{
				Func:    headFn,
				Type:    ast.ChangeBody,
				OldFunc: &baseFn,
			})
		}
	}

	// Find added functions
	for qname, headFn := range headMap {
		if _, exists := baseMap[qname]; !exists {
			// Only mark as added if not already captured as a rename target
			if !isRenameTarget(headFn, base, headMap) {
				changes = append(changes, ast.SemanticChange{
					Func: headFn,
					Type: ast.ChangeAdded,
				})
			}
		}
	}

	return changes
}

func indexByQName(funcs []ast.FunctionDef) map[string]ast.FunctionDef {
	m := make(map[string]ast.FunctionDef, len(funcs))
	for _, f := range funcs {
		m[f.QualifiedName()] = f
	}
	return m
}

// findRename checks if a removed function was actually renamed in headMap.
// Heuristic: same body hash and same parameter structure.
func findRename(removed ast.FunctionDef, headMap, baseMap map[string]ast.FunctionDef) *ast.FunctionDef {
	for qname, headFn := range headMap {
		if _, existsInBase := baseMap[qname]; existsInBase {
			continue // not new in head
		}
		if removed.BodyHash != "" && removed.BodyHash == headFn.BodyHash {
			return &headFn
		}
		if similarSignature(removed.Signature, headFn.Signature) {
			return &headFn
		}
	}
	return nil
}

// isRenameTarget returns true if headFn is the target of a rename from base.
func isRenameTarget(headFn ast.FunctionDef, base []ast.FunctionDef, headMap map[string]ast.FunctionDef) bool {
	for _, baseFn := range base {
		if _, existsInHead := headMap[baseFn.QualifiedName()]; existsInHead {
			continue
		}
		if baseFn.BodyHash != "" && baseFn.BodyHash == headFn.BodyHash {
			return true
		}
		if similarSignature(baseFn.Signature, headFn.Signature) {
			return true
		}
	}
	return false
}

// similarSignature returns true if two signatures have the same parameter types
// (ignoring the function name), indicating a likely rename.
func similarSignature(a, b string) bool {
	aParams := extractParamTypes(a)
	bParams := extractParamTypes(b)
	if aParams == "" || bParams == "" {
		return false
	}
	return aParams == bParams
}

func extractParamTypes(sig string) string {
	start := strings.Index(sig, "(")
	end := strings.LastIndex(sig, ")")
	if start < 0 || end <= start {
		return ""
	}
	return sig[start : end+1]
}
