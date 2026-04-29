package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func mockAnalyzer(_ context.Context, req AnalysisRequest, events chan<- GraphEvent) {
	defer close(events)

	raw, _ := json.Marshal(ProgressPayload{Stage: "test", Message: "running", Percent: 50})
	events <- GraphEvent{Type: EventProgress, Payload: raw}

	nodeRaw, _ := json.Marshal(GraphNodePayload{
		ID: "test::Foo", Label: "Foo", ChangeType: "body_changed",
		Depth: 0, Score: 1.0,
	})
	events <- GraphEvent{Type: EventNode, Payload: nodeRaw}

	completeRaw, _ := json.Marshal(CompletePayload{
		TotalChanged: 1, TotalAffected: 0, MaxDepth: 0, Duration: 5,
	})
	events <- GraphEvent{Type: EventComplete, Payload: completeRaw}
}

func TestServer_HealthEndpoint(t *testing.T) {
	srv := New(0, mockAnalyzer)
	ts := httptest.NewServer(http.HandlerFunc(srv.handleHealth))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestServer_AnalyzeAndWebSocket(t *testing.T) {
	srv := New(0, mockAnalyzer)
	ts := httptest.NewServer(withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/analyze":
			srv.handleAnalyze(w, r)
		case "/ws":
			srv.handleWebSocket(w, r)
		}
	})))
	defer ts.Close()

	// Submit analysis
	body := `{"repoPath":".","baseBranch":"main","headBranch":"feature"}`
	resp, err := http.Post(ts.URL+"/api/analyze", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result AnalysisResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.SessionID == "" {
		t.Fatal("expected non-empty session ID")
	}

	// Give the goroutine time to buffer events
	time.Sleep(100 * time.Millisecond)

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?session=" + result.SessionID
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal("WebSocket dial:", err)
	}
	defer conn.Close()

	var events []GraphEvent
	for {
		var evt GraphEvent
		if err := conn.ReadJSON(&evt); err != nil {
			break
		}
		events = append(events, evt)
		if evt.Type == EventComplete || evt.Type == EventError {
			break
		}
	}

	if len(events) < 3 {
		t.Errorf("expected at least 3 events (progress, node, complete), got %d", len(events))
	}

	types := make([]string, len(events))
	for i, e := range events {
		types[i] = string(e.Type)
	}
	t.Logf("Events received: %v", types)

	lastType := events[len(events)-1].Type
	if lastType != EventComplete {
		t.Errorf("expected last event to be complete, got %s", lastType)
	}
}
