# CallBlast

**Finds every function your PR will actually break before your teammates do.**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.4-3178C6?style=flat&logo=typescript)](https://typescriptlang.org)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat)](LICENSE)

Code reviewers see line diffs. CallBlast sees impact. A one-line signature change in a shared utility can silently break dozens of callers across the codebase — with no warning in the PR diff. CallBlast parses both sides of your PR with Go's native AST, constructs a full call graph over the entire repository, and traces the transitive blast radius of every semantic change before the branch merges.

## What it does

You give it a repo path, a base branch, and a PR branch. It gives you an interactive force-directed graph of every function that will be affected — scored by impact depth, highlighted by severity, and explorable by click.

```
callblast — Analyze PR blast radius
  Repo:   /path/to/your/repo
  Base:   main
  Head:   feat/refactor-auth
  
  Found 3 changed functions
  Blast radius: 17 affected callers across 6 files
  Critical path: 4 nodes reachable from 2+ changes
  Duration: 142ms
```

## Architecture

```
[Web UI — React + D3]
       ↕ WebSocket
[HTTP Server — Go]
       │
       ├── [Git Integration]   git diff, git show, git ls-files
       │
       ├── [AST Extractor]     go/ast → FunctionDef{name, sig, bodyHash}
       │                       regex  → TypeScript function extraction
       │
       ├── [Semantic Diff]     compare function versions across base/head
       │                       classify: ADDED | REMOVED | SIG_CHANGED | BODY_CHANGED | RENAMED
       │
       ├── [Call Graph]        directed adjacency map of all function calls
       │                       resolves short names to qualified file::funcName
       │
       └── [BFS Reachability]  transitive closure from changed functions
                               scores nodes by depth + critical-path membership
```

## Technical deep-dive

### Why go/ast instead of tree-sitter

The standard Go `go/ast` package gives us richer semantic information than tree-sitter for Go specifically — method receivers, type resolution, and clean position data. It requires no CGO, no compiled grammar binaries, and handles all of Go's syntax edge cases. For TypeScript we use a regex-heuristic approach: less precise, but sufficient for identifying function boundaries and extracting call sites for the call graph.

### The blast radius algorithm

Semantic diff compares function definitions by qualified name (`file::Type.method` or `file::function`). If a function exists in both base and head, we compare its signature and body hash (SHA-1 of the body text). Renames are detected by matching body hashes across files — if a function disappeared and a new function appeared with the same body hash, it was renamed.

The call graph is a directed adjacency map: `caller → []callees`, with a parallel reverse map `callee → []callers`. Short names like `validateInput` are resolved to qualified names using a short-name index built during graph construction, preferring same-file resolution on ambiguity.

BFS runs backwards through the reverse graph from each changed function. A node's impact score is `1 / (depth + 1)^0.7` — decaying with distance but sub-linearly, so third-level callers still show meaningful impact. Nodes reachable from two or more changed functions are marked "critical path" and rendered in red.

### Why stream via WebSocket

Call graph construction over a large repo (10,000+ files) takes time. Streaming events as they're discovered means the D3 graph starts rendering immediately with changed functions, then populates outward as BFS explores. The server buffers events in memory so late WebSocket connections can replay the full session.

## Install

**Prerequisites:** Go 1.21+, Node.js 18+

```bash
git clone https://github.com/rayancheca/callblast
cd callblast

# Build Go backend
go build -o callblast ./cmd/callblast

# Build React frontend
cd web && npm install && npm run build && cd ..
```

## Run

```bash
./callblast --port 7332
# Open http://localhost:7332
```

Or run frontend separately in development:

```bash
# Terminal 1: backend
./callblast --port 7332 --static=""

# Terminal 2: frontend with hot reload
cd web && npm run dev
# Open http://localhost:5173
```

## Usage

1. Open `http://localhost:7332`
2. Enter your repo path (absolute or `.` for current directory)
3. Set the base branch (e.g. `main`) and your PR branch (e.g. `feat/my-change`)
4. Click **Run Analysis**

The graph streams in live as analysis runs:
- **Amber nodes** — functions that changed in the PR (origin of blast)
- **Red nodes** — critical-path functions reachable from two or more changed functions
- **Blue nodes** — transitively affected callers
- Click any node for full detail: signature, location, callers, callees, impact score
- Left sidebar ranks affected files by maximum impact score
- Drag nodes to reorganize; scroll to zoom; click background to deselect

## Supported languages

| Language | Function extraction | Call site extraction |
|----------|--------------------|--------------------|
| Go       | Full (go/ast)      | Full                |
| TypeScript / JavaScript | Heuristic | Heuristic |

## Run tests

```bash
go test ./...
```

## License

MIT
