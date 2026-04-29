package ast

import (
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"
)

var (
	// Match: function foo(...) { or async function foo(...) {
	reFuncDecl = regexp.MustCompile(`(?m)^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)`)
	// Match: const foo = (...) => { or const foo = async (...) => {
	reArrowFunc = regexp.MustCompile(`(?m)^(?:export\s+)?(?:const|let)\s+(\w+)\s*=\s*(?:async\s+)?\(([^)]*)\)\s*(?::\s*\S+\s*)?=>`)
	// Match: foo = (...) => { inside class
	reMethodFunc = regexp.MustCompile(`(?m)^\s+(?:async\s+)?(\w+)\s*\(([^)]*)\)\s*(?::\s*\S+\s*)?\{`)
	// Match function calls: foo( or foo.bar(
	reCallExpr = regexp.MustCompile(`\b(\w+)\s*\(`)
)

// ExtractTSFile extracts function definitions and call sites from TypeScript/JavaScript source.
func ExtractTSFile(filePath string, src []byte) ([]FunctionDef, []CallSite, error) {
	content := string(src)
	lines := strings.Split(content, "\n")

	var funcs []FunctionDef

	// Extract standard function declarations
	for _, m := range reFuncDecl.FindAllStringSubmatchIndex(content, -1) {
		name := content[m[2]:m[3]]
		params := content[m[4]:m[5]]
		lineNum := countLines(content, m[0])
		sig := fmt.Sprintf("function %s(%s)", name, params)
		bodyHash := hashSubstring(content, m[0])
		funcs = append(funcs, FunctionDef{
			Name:      name,
			File:      filePath,
			Line:      lineNum,
			Signature: sig,
			BodyHash:  bodyHash,
			Language:  "typescript",
		})
	}

	// Extract arrow function declarations
	for _, m := range reArrowFunc.FindAllStringSubmatchIndex(content, -1) {
		name := content[m[2]:m[3]]
		params := content[m[4]:m[5]]
		lineNum := countLines(content, m[0])
		sig := fmt.Sprintf("const %s = (%s) =>", name, params)
		bodyHash := hashSubstring(content, m[0])
		funcs = append(funcs, FunctionDef{
			Name:      name,
			File:      filePath,
			Line:      lineNum,
			Signature: sig,
			BodyHash:  bodyHash,
			Language:  "typescript",
		})
	}

	// Collect all function calls per function (simple: attribute all calls to nearest function)
	var calls []CallSite
	for _, funcDef := range funcs {
		startLine := funcDef.Line
		endLine := findFuncEnd(lines, startLine)
		callerQName := funcDef.QualifiedName()

		for i := startLine; i < endLine && i < len(lines); i++ {
			for _, m := range reCallExpr.FindAllStringSubmatch(lines[i], -1) {
				callee := m[1]
				if isJSKeyword(callee) || callee == funcDef.Name {
					continue
				}
				calls = append(calls, CallSite{
					CallerFunc: callerQName,
					CalleeFunc: callee,
					File:       filePath,
					Line:       i + 1,
				})
			}
		}
	}

	return funcs, calls, nil
}

func countLines(content string, offset int) int {
	return strings.Count(content[:offset], "\n") + 1
}

func hashSubstring(content string, offset int) string {
	end := offset + 200
	if end > len(content) {
		end = len(content)
	}
	h := sha1.Sum([]byte(content[offset:end]))
	return fmt.Sprintf("%x", h)
}

func findFuncEnd(lines []string, startLine int) int {
	depth := 0
	for i := startLine - 1; i < len(lines); i++ {
		for _, ch := range lines[i] {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return len(lines)
}

var jsKeywords = map[string]bool{
	"if": true, "else": true, "for": true, "while": true, "switch": true,
	"case": true, "return": true, "new": true, "typeof": true, "instanceof": true,
	"async": true, "await": true, "import": true, "export": true, "class": true,
	"const": true, "let": true, "var": true, "function": true, "catch": true,
	"throw": true, "try": true, "delete": true, "void": true, "super": true,
	"this": true, "null": true, "true": true, "false": true,
}

func isJSKeyword(name string) bool {
	return jsKeywords[name]
}
