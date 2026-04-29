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
	// Match plain function calls: foo(
	reCallExpr = regexp.MustCompile(`\b(\w+)\s*\(`)
	// Match selector (method) calls: receiver.method( — captures receiver and method separately.
	// This allows the call graph to store qualified call sites like "Service.validate" and
	// later match them against extracted function names from the same file/module.
	reSelectorCall = regexp.MustCompile(`\b(\w+)\.(\w+)\s*\(`)
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

	// Collect all function calls per function (attribute calls to nearest enclosing function).
	var calls []CallSite
	for _, funcDef := range funcs {
		startLine := funcDef.Line
		endLine := findFuncEnd(lines, startLine)
		callerQName := funcDef.QualifiedName()

		for i := startLine; i < endLine && i < len(lines); i++ {
			line := lines[i]

			// Emit selector calls (obj.method) with a qualified callee name so the
			// call graph can distinguish service1.save() from service2.save().
			selectorSeen := map[string]bool{}
			for _, m := range reSelectorCall.FindAllStringSubmatch(line, -1) {
				receiver, method := m[1], m[2]
				if isJSKeyword(receiver) || isJSKeyword(method) {
					continue
				}
				qualified := receiver + "." + method
				selectorSeen[method] = true
				calls = append(calls, CallSite{
					CallerFunc: callerQName,
					CalleeFunc: qualified,
					File:       filePath,
					Line:       i + 1,
				})
			}

			// Emit plain calls (foo) only when not already covered by a selector match,
			// to avoid double-counting method portions of selector calls.
			for _, m := range reCallExpr.FindAllStringSubmatch(line, -1) {
				callee := m[1]
				if isJSKeyword(callee) || callee == funcDef.Name || selectorSeen[callee] {
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
