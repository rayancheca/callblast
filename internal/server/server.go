package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // dev mode; restrict in production
	},
}

// session stores the event channel for a running analysis.
type session struct {
	events chan GraphEvent
	done   bool
	mu     sync.Mutex
	buffer []GraphEvent
}

// Server manages analysis sessions and WebSocket connections.
type Server struct {
	mu       sync.Mutex
	sessions map[string]*session
	analyzer func(req AnalysisRequest, events chan<- GraphEvent)
	port     int
}

// New creates a new Server with the given analyzer function.
func New(port int, analyzer func(req AnalysisRequest, events chan<- GraphEvent)) *Server {
	return &Server{
		sessions: make(map[string]*session),
		analyzer: analyzer,
		port:     port,
	}
}

// Run starts the HTTP server and blocks.
func (s *Server) Run(staticDir string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/analyze", s.handleAnalyze)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/health", s.handleHealth)

	// Serve static frontend if directory exists
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

	sessionID := newSessionID()
	events := make(chan GraphEvent, 256)

	sess := &session{events: events, buffer: make([]GraphEvent, 0, 64)}
	s.mu.Lock()
	s.sessions[sessionID] = sess
	s.mu.Unlock()

	// Start analysis in background, buffer events for WebSocket to pick up
	go func() {
		go s.analyzer(req, events)
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

		// Clean up session after 5 minutes
		time.AfterFunc(5*time.Minute, func() {
			s.mu.Lock()
			delete(s.sessions, sessionID)
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

	// Stream all buffered and future events
	sent := 0
	for {
		sess.mu.Lock()
		newEvents := sess.buffer[sent:]
		done := sess.done && sent >= len(sess.buffer)
		sess.mu.Unlock()

		for _, evt := range newEvents {
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
