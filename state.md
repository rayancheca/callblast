## Status
IN PROGRESS

## Project
callblast — Finds every function your PR will actually break before your teammates do.

## Session count
2

## Completed steps
1. Read all rule files and full project specification
2. Updated CLAUDE.md with complete technical spec including visual direction
3. Go module + gorilla/websocket dependency
4. Full directory structure (cmd/, internal/*, web/src/*)
5. internal/ast/types.go — FunctionDef, CallSite, SemanticChange types
6. internal/ast/go_extract.go — Go AST extraction via go/ast
7. internal/ast/ts_extract.go — TypeScript function extraction via regex
8. internal/gitutil/git.go — git diff, show, ls-files with ref validation
9. internal/diff/semantic.go — ADDED/REMOVED/SIG/BODY/RENAMED classification
10. internal/graph/callgraph.go — directed call graph, short-name resolution, reverse index
11. internal/graph/bfs.go — BFS reachability, depth scoring, critical-path detection
12. internal/analysis/analyzer.go — full pipeline with context/cancellation
13. internal/server/server.go — HTTP + WebSocket, concurrency semaphore, race-safe buffer
14. cmd/callblast/main.go — CLI entry point
15. web/src/styles/ — CSS custom properties, dark precision analytics palette
16. web/src/types/index.ts — TypeScript types for graph events
17. web/src/hooks/useAnalysis.ts — WebSocket + state management hook
18. web/src/components/Header.tsx — stats bar, reset button
19. web/src/components/AnalysisForm.tsx — repo/branch inputs, progress bar
20. web/src/components/BlastGraph.tsx — D3 force-directed, amber/red/blue nodes, pulse animation
21. web/src/components/NodeDetail.tsx — click-through panel, score bar, caller/callee lists
22. web/src/components/ImpactList.tsx — file impact ranking sidebar
23. web/src/App.tsx — layout orchestrator
24. 11 tests across 5 packages — all pass with -race
25. README.md — architecture, technical deep-dive, install instructions
26. Security review fixes: ref injection, data race, goroutine leak, semaphore limit
27. Pushed to https://github.com/rayancheca/callblast
28. Updated project_history.md
--- SESSION 2 ---
29. Makefile — build, test, dev, run, clean targets
30. TypeScript SelectorExpr — reSelectorCall regex emits qualified obj.method callee names; deduplicates plain form; new test
31. Demo mode — GET /api/demo returns cwd + HEAD~1/HEAD; frontend "Try demo" button fills + handles errors
32. GitHub PR integration — internal/github package + POST /api/github-pr endpoint + "Import from GitHub PR" form section

## In progress
Session 2 complete — ready for session 3

## Next steps (session 3)
1. Screenshot the running UI for README (need a live analysis to capture graph, detail panel, impact list)
2. Add Playwright E2E tests for: form submission, demo button, PR import, graph rendering
3. Consider adding TypeScript class method extraction (currently only free functions and arrow functions)
4. Consider a --demo CLI flag that auto-opens the browser and runs the demo on launch
5. Consider adding a GitHub Actions CI workflow

## Blockers
None

## Notes
Visual direction: dark precision analytics — graph is the hero, amber=#f59e0b changed nodes, red=#dc2626 critical-path pulse
Architecture: go/ast → semantic diff → call graph → BFS → WebSocket stream → D3 force simulation
Port: 7332 (default)
Frontend: web/dist/ (built); dev server: cd web && npm run dev
GITHUB_TOKEN env var required for private repos; optional for public (rate limiting)

## Git log
db96d30 feat: Makefile + TypeScript SelectorExpr call site resolution
aeb1f92 feat: demo mode — /api/demo endpoint and Try demo button
7c655ec feat: GitHub PR integration — resolve branch names from PR URL
