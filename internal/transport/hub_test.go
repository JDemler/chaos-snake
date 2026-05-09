package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"chaos-snake/internal/game"
)

// readMessage reads one JSON message from the websocket and decodes it into
// a generic map. Fails the test if no message arrives in time.
func readMessage(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("decode %q: %v", data, err)
	}
	return m
}

// readUntilType reads messages until one with matching type is found, or fails.
func readUntilType(t *testing.T, conn *websocket.Conn, wantType string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		if m["type"] == wantType {
			return m
		}
	}
	t.Fatalf("did not receive message of type %q", wantType)
	return nil
}

func TestJoinAndInputFlow(t *testing.T) {
	g := game.NewGame()
	hub := NewHub(g)

	srv := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// First message should be a snapshot with you=""
	first := readUntilType(t, conn, "snapshot")
	if first["you"] != "" {
		t.Fatalf("initial snapshot should have empty you, got %v", first["you"])
	}

	// Join
	if err := conn.WriteJSON(map[string]string{"type": "join", "name": "alice"}); err != nil {
		t.Fatalf("write join: %v", err)
	}

	// Should get a snapshot back with our snake ID
	joined := readUntilType(t, conn, "snapshot")
	you, _ := joined["you"].(string)
	if you == "" {
		t.Fatalf("expected non-empty you after join, got %v", joined)
	}
	snakes, _ := joined["snakes"].([]any)
	if len(snakes) != 1 {
		t.Fatalf("expected 1 snake in snapshot, got %d", len(snakes))
	}

	// Send a direction input
	if err := conn.WriteJSON(map[string]string{"type": "input", "dir": "up"}); err != nil {
		t.Fatalf("write input: %v", err)
	}

	// Should get a delta showing our snake moved
	delta := readUntilType(t, conn, "delta")
	moves, _ := delta["moves"].([]any)
	if len(moves) == 0 {
		t.Fatalf("expected moves in delta, got %v", delta)
	}
	found := false
	for _, m := range moves {
		mm, _ := m.(map[string]any)
		if mm["id"] == you {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected delta move for our snake %s; got moves=%v", you, moves)
	}
}
