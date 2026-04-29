# Agent Instructions — Read Before Every Action

Read in this order, every session:

1. `~/daily-builder/prompts/rules/session_protocol.md`
2. `~/daily-builder/prompts/rules/quality_bar.md`
3. `~/daily-builder/prompts/rules/code_rules.md`
4. `state.md` in this directory
5. This file

---

# Project: CallBlast — PR Blast-Radius Analyzer

**Tagline:** Finds every function your PR will actually break before your teammates do.

**Domain:** Developer Tooling, CLIs and Platform Engineering

**Tech stack:** Go (go/ast, WebSocket), TypeScript, React, D3.js

---

## Architecture

```
[CLI / HTTP API]
      │
      ▼
[Git Integration] → git show, git diff → [Changed Files]
      │
      ▼
[AST Extractor] → go/ast (Go files), regex-heuristic (TS files)
      │           → FunctionDef{name, file, line, signature, body}
      ▼
[Semantic Diff] → compare function versions: RENAME / SIGNATURE_CHANGE / BODY_CHANGE
      │
      ▼
[Call Graph] → directed graph: callerFn → []calleeFns (per repo)
      │
      ▼
[BFS Reachability] → from changed functions, trace transitive callers
      │               → score by depth, frequency, critical-path
      ▼
[WebSocket Server] → stream AnalysisEvent{type, nodes, edges, progress}
      │
      ▼
[React + D3 Frontend] → force-directed graph, amber changed, red critical
```

## Data Flow

1. User submits: `{repoPath, baseBranch, headBranch}`
2. Server: run `git diff baseBranch...headBranch --name-only` → changed files
3. For each changed file: extract function defs from base and head versions
4. Semantic diff: classify each changed function (rename / sig / body)
5. Parse ALL source files in repo → build call graph
6. BFS from changed functions → compute blast radius
7. Stream graph events via WebSocket: `{type:"node"|"edge"|"complete"}`
8. Frontend renders force-directed D3 graph in real time

## Visual Direction

**Derived from:** Platform engineers living in GitHub/terminal/VSCode, viewing precise static analysis data (call graphs, impact scores), tool must communicate depth, correctness, analytical power.

**Visual mood:** Dark precision analytics — the call graph is the hero, color encodes impact severity, monospace-heavy data labels, Bloomberg Terminal density without the clutter. Feels like a polished version of what `go tool pprof` would look like if it had a web UI.

- `background`: `#0f0f0f` — near-black, graph floats above it
- `surface`: `#161616` — panel backgrounds
- `surface-elevated`: `#1e1e1e` — card/detail panels
- `border`: `#2d2d2d` — subtle separators
- `text-primary`: `#e6e6e6` — primary labels
- `text-secondary`: `#737373` — secondary metadata
- `accent` (amber): `#f59e0b` — changed functions, the blast origin
- `accent-secondary` (crimson): `#dc2626` — critical-path high-impact nodes
- `safe` (green): `#16a34a` — unaffected neighbors shown for context
- `ui-font`: `Inter, system-ui` — functional, clean, developer-native
- `mono-font`: `JetBrains Mono, Fira Code, monospace` — signatures, paths, counts

## Complete File Structure

```
callblast/
├── cmd/callblast/main.go        # CLI entry: parse flags, start server
├── internal/
│   ├── ast/
│   │   ├── types.go             # FunctionDef, CallSite, ChangeType
│   │   ├── go_extract.go        # Extract functions + callsites from Go via go/ast
│   │   └── ts_extract.go        # Extract functions + callsites from TS via regex
│   ├── diff/
│   │   └── semantic.go          # Semantic diff: compare FunctionDef pairs
│   ├── gitutil/
│   │   └── git.go               # git diff, git show, git ls-files
│   ├── graph/
│   │   ├── callgraph.go         # Build directed call graph from all source
│   │   └── bfs.go               # BFS reachability, scoring
│   ├── analysis/
│   │   └── analyzer.go          # Orchestrate full pipeline
│   └── server/
│       ├── server.go            # HTTP + WebSocket server
│       └── types.go             # API types: AnalysisRequest, GraphEvent
├── web/
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── types/index.ts       # GraphNode, GraphEdge, AnalysisEvent
│       ├── styles/
│       │   ├── tokens.css       # CSS custom properties
│       │   └── global.css       # Reset + global styles
│       ├── components/
│       │   ├── Header.tsx
│       │   ├── AnalysisForm.tsx # Repo + branch inputs
│       │   ├── BlastGraph.tsx   # D3 force-directed graph
│       │   ├── NodeDetail.tsx   # Click-through detail panel
│       │   └── ImpactList.tsx   # Ranked list of affected files
│       └── hooks/
│           ├── useWebSocket.ts  # WebSocket connection + events
│           └── useD3.ts         # D3 ref helper hook
├── go.mod
├── go.sum
├── .gitignore
├── CLAUDE.md
├── state.md
└── README.md
```

## Implementation Steps (strict order)

### Step 1 — Go module + HTTP server skeleton
- Create go.mod with module `github.com/rayancheca/callblast`
- Add `github.com/gorilla/websocket` dependency
- `cmd/callblast/main.go`: parse `--port` flag, start HTTP server, serve static files from `web/dist`
- `internal/server/types.go`: define `AnalysisRequest`, `GraphEvent`, `GraphNode`, `GraphEdge`
- `internal/server/server.go`: POST `/api/analyze` → start analysis goroutine, WebSocket `/ws` → stream events
- Verify: `go build ./...` succeeds, server starts

### Step 2 — AST types and Go extractor
- `internal/ast/types.go`: `FunctionDef{Name, File, Line, Signature, BodyHash}`, `CallSite{CallerFunc, CalleeFunc, File, Line}`, `ChangeType` enum
- `internal/ast/go_extract.go`: `ExtractGoFile(path string, src []byte) ([]FunctionDef, []CallSite, error)`
  - Use `go/parser.ParseFile`, walk AST
  - Visit `*ast.FuncDecl` → FunctionDef
  - Visit `*ast.CallExpr` within each func body → CallSite
- Verify: write a test that parses a small Go file and prints extracted functions/calls

### Step 3 — Git integration
- `internal/gitutil/git.go`:
  - `ChangedFiles(repoPath, base, head string) ([]string, error)` — run `git diff --name-only base...head`
  - `FileAtRef(repoPath, ref, path string) ([]byte, error)` — run `git show ref:path`
  - `AllSourceFiles(repoPath string) ([]string, error)` — run `git ls-files *.go *.ts *.tsx`
- Verify: test against this repo itself

### Step 4 — Semantic diff engine
- `internal/diff/semantic.go`:
  - `DiffFunctions(base, head []FunctionDef) []SemanticChange`
  - `SemanticChange{Func FunctionDef, Type ChangeType, OldFunc *FunctionDef}`
  - `ChangeType`: ADDED, REMOVED, SIGNATURE_CHANGED, BODY_CHANGED, RENAMED
  - Match by name first, then by signature similarity for renames
- Verify: diff two versions of a test function and classify correctly

### Step 5 — Call graph construction
- `internal/graph/callgraph.go`:
  - `CallGraph` struct: adjacency list `map[string][]string` (caller → callees)
  - `BuildFromRepo(repoPath string) (*CallGraph, error)` — parse all Go/TS files, collect all CallSites
  - `Callers(funcName string) []string` — reverse lookup (reverse graph)
  - `Callees(funcName string) []string` — forward lookup
- Verify: print call graph of this project

### Step 6 — BFS reachability engine
- `internal/graph/bfs.go`:
  - `BlastRadius(cg *CallGraph, changedFuncs []string) *BlastResult`
  - `BlastResult{Nodes []BlastNode, Edges []BlastEdge, MaxDepth int}`
  - `BlastNode{Func, File, Depth, Score, IsCritical bool}`
  - BFS from each changed function using reverse graph (find who calls them)
  - Score: `1.0 / (depth + 1)` × call_frequency_weight
  - Critical path: nodes reachable from multiple changed functions
- Verify: run on synthetic call graph, verify transitive closure

### Step 7 — Analysis orchestrator
- `internal/analysis/analyzer.go`:
  - `RunAnalysis(req AnalysisRequest, events chan<- GraphEvent) error`
  - Steps: git diff → extract functions → semantic diff → build call graph → BFS
  - Stream `GraphEvent{Type:"progress"|"node"|"edge"|"complete", Payload json.RawMessage}` to channel
- Verify: run on a real Go repo, check events are emitted correctly

### Step 8 — WebSocket server integration
- `internal/server/server.go`:
  - POST `/api/analyze`: validate request, start goroutine, return `{sessionId}`
  - GET `/ws?session=ID`: upgrade to WebSocket, relay events from analysis channel
  - Handle errors: stream `GraphEvent{Type:"error", Payload: errorMsg}`
- Verify: curl the API, verify events come through WebSocket

### Step 9 — Frontend setup (Vite + React + TypeScript)
- `web/package.json`: vite, react, react-dom, d3, @types/*
- `web/vite.config.ts`: proxy `/api` and `/ws` to Go server
- `web/src/types/index.ts`: `GraphNode`, `GraphEdge`, `AnalysisEvent`, `AnalysisRequest`
- `web/src/styles/tokens.css`: all CSS custom properties
- `web/src/styles/global.css`: reset + base styles
- `web/src/main.tsx`: mount React app
- `web/src/App.tsx`: layout — Header, AnalysisForm, BlastGraph, NodeDetail
- Verify: `pnpm dev` starts, blank dark page renders

### Step 10 — AnalysisForm component
- `web/src/components/AnalysisForm.tsx`:
  - Inputs: repo path (text), base branch (text), head branch (text)
  - Submit → POST `/api/analyze` → connect WebSocket → stream results
  - Show progress spinner during analysis
  - Error states with clear messages
- Verify: form submits, WebSocket connects, events received in console

### Step 11 — D3 blast graph component
- `web/src/components/BlastGraph.tsx`:
  - SVG with force-directed layout: `d3.forceSimulation`
  - Changed function nodes: amber glow, larger radius
  - Critical-path nodes: red pulse animation
  - Unaffected neighbor nodes: dim green
  - Edges: thin dark lines, width encodes call frequency
  - Zoom + pan via `d3.zoom`
  - Click node → emit selection event
- Verify: graph renders, nodes appear as events stream in, zoom works

### Step 12 — NodeDetail panel
- `web/src/components/NodeDetail.tsx`:
  - Slide-in panel on right when node selected
  - Show: function name, file, line, change type, depth score
  - Show: list of direct callers, list of direct callees
  - Show: downstream call tree (collapsible)
  - Impact badge: "Affected by 3 changes"
- Verify: click node, panel slides in with correct data

### Step 13 — ImpactList component
- `web/src/components/ImpactList.tsx`:
  - Left sidebar: ranked list of affected files
  - Each file: name, affected function count, highest severity
  - Click file → highlight all its nodes in graph
- Verify: list renders, click highlights work

### Step 14 — Polish + README
- Refine animations: node entrance stagger, edge draw-on, critical-path pulse
- Refine typography: function names in mono, depth scores prominent
- README.md: complete with architecture diagram, screenshots description, install steps
- Verify: end-to-end run on this repo or another Go project, graph looks great

## Session rules
- Read state.md FIRST, every session
- After every working step: `git add -A && git commit -m "feat: ..."`
- Update state.md after each step
- Never leave broken code committed
- Push before stopping
