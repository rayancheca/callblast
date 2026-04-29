package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxConcurrentSessions = 10
	sessionTTL            = 5 * time.Minute
	analysisTimeout       = 2 * time.Minute
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // developer tool — runs locally only
	},
}

// session stores buffered events and a cancel func for the running analysis.
type session struct {
	mu     sync.Mutex
	buffer []GraphEvent
	done   bool
	cancel context.CancelFunc
}

// Server manages analysis sessions and WebSocket connections.
type Server struct {
	mu        sync.Mutex
	sessions  map[string]*session
	semaphore chan struct{}
	analyzer  func(ctx context.Context, req AnalysisRequest, events chan<- GraphEvent)
	port      int
	repoPath  string // working directory, used by /api/demo
}

// AnalyzerFunc is the signature of the analysis function accepted by the server.
type AnalyzerFunc func(ctx context.Context, req AnalysisRequest, events chan<- GraphEvent)

// New creates a new Server with the given analyzer function.
func New(port int, analyzer AnalyzerFunc) *Server {
	cwd, _ := os.Getwd()
	return &Server{
		sessions:  make(map[string]*session),
		semaphore: make(chan struct{}, maxConcurrentSessions),
		analyzer:  analyzer,
		port:      port,
		repoPath:  cwd,
	}
}

// Run starts the HTTP server and blocks.
func (s *Server) Run(staticDir string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/analyze", s.handleAnalyze)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/demo", s.handleDemo)

	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("callblast listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, withCORS(mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleDemo returns a pre-filled AnalysisRequest pointing at the server's own
// repository (cwd), comparing HEAD~1 → HEAD. Handy for a one-click "try it out".
func (s *Server) handleDemo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, AnalysisRequest{
		RepoPath:   s.repoPath,
		BaseBranch: "HEAD~1",
		HeadBranch: "HEAD",
	})
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorPayload{Message: "invalid request body"})
		return
	}

	if req.BaseBranch == "" || req.HeadBranch == "" {
		writeJSON(w, http.StatusBadRequest, ErrorPayload{Message: "baseBranch and headBranch are required"})
		return
	}

	// Enforce concurrency limit — reject if at capacity
	select {
	case s.semaphore <- struct{}{}:
	default:
		writeJSON(w, http.StatusTooManyRequests, ErrorPayload{Message: "too many concurrent analyses; try again shortly"})
		return
	}

	sessionID := newSessionID()
	ctx, cancel := context.WithTimeout(context.Background(), analysisTimeout)

	sess := &session{
		buffer: make([]GraphEvent, 0, 64),
		cancel: cancel,
	}

	s.mu.Lock()
	s.sessions[sessionID] = sess
	s.mu.Unlock()

	events := make(chan GraphEvent, 256)

	go func() {
		defer func() {
			cancel()
			<-s.semaphore
		}()

		s.analyzer(ctx, req, events)

		for evt := range events {
			sess.mu.Lock()
			sess.buffer = append(sess.buffer, evt)
			if evt.Type == EventComplete || evt.Type == EventError {
				sess.done = true
			}
			sess.mu.Unlock()
		}

		sess.mu.Lock()
		sess.done = true
		sess.mu.Unlock()

		// Evict session after TTL
		time.AfterFunc(sessionTTL, func() {
			s.mu.Lock()
			if s2, ok := s.sessions[sessionID]; ok && s2 == sess {
				delete(s.sessions, sessionID)
			}
			s.mu.Unlock()
		})
	}()

	writeJSON(w, http.StatusOK, AnalysisResponse{SessionID: sessionID})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "session parameter required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	sent := 0
	for {
		// Copy new events under the lock to avoid race on the slice backing array
		sess.mu.Lock()
		available := sess.buffer[sent:]
		batch := make([]GraphEvent, len(available))
		copy(batch, available)
		done := sess.done && len(available) == 0
		sess.mu.Unlock()

		for _, evt := range batch {
			if err := conn.WriteJSON(evt); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
			sent++
		}

		if done {
			break
		}

		time.Sleep(25 * time.Millisecond)
	}
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

func newSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
