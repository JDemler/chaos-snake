package transport

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"chaos-snake/internal/game"
)

const (
	tickInterval     = time.Second / time.Duration(game.TickHz)
	snapshotEvery    = 50
	writeWait        = 5 * time.Second
	pongWait         = 60 * time.Second
	pingPeriod       = 30 * time.Second
	maxMessageSize   = 1024
	clientSendBuffer = 32
)

type Hub struct {
	game     *game.Game
	upgrader websocket.Upgrader

	mu      sync.Mutex
	clients map[*Client]bool
}

func NewHub(g *game.Game) *Hub {
	return &Hub{
		game:    g,
		clients: map[*Client]bool{},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (h *Hub) Run(ctx context.Context) {
	t := time.NewTicker(tickInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ev := h.game.Step()
			h.broadcastDelta(ev)
			if ev.Tick%snapshotEvery == 0 {
				h.broadcastSnapshot()
			}
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	c := &Client{
		conn: conn,
		hub:  h,
		send: make(chan []byte, clientSendBuffer),
		done: make(chan struct{}),
	}
	h.addClient(c)
	c.queue(h.snapshotBytes(""))

	go c.writePump()
	c.readPump()
}

func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = true
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	exists := h.clients[c]
	if exists {
		delete(h.clients, c)
	}
	h.mu.Unlock()
	if !exists {
		return
	}
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	if c.snakeID != "" {
		h.game.Leave(c.snakeID)
		c.snakeID = ""
	}
}

func (h *Hub) broadcastDelta(ev game.TickEvent) {
	d := makeDelta(ev)
	if emptyDelta(d) {
		return
	}
	msg, err := json.Marshal(d)
	if err != nil {
		log.Printf("marshal delta: %v", err)
		return
	}
	h.mu.Lock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
		}
	}
	h.mu.Unlock()
}

func (h *Hub) broadcastSnapshot() {
	snap := h.game.Snapshot()
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		b, err := json.Marshal(makeSnapshot(snap, c.snakeID))
		if err != nil {
			continue
		}
		select {
		case c.send <- b:
		default:
		}
	}
}

func (h *Hub) snapshotBytes(forID string) []byte {
	snap := h.game.Snapshot()
	b, _ := json.Marshal(makeSnapshot(snap, forID))
	return b
}

type Client struct {
	conn    *websocket.Conn
	hub     *Hub
	send    chan []byte
	done    chan struct{}
	snakeID string
}

func (c *Client) queue(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.removeClient(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg clientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "join":
			if c.snakeID != "" {
				continue
			}
			s := c.hub.game.Join(msg.Name)
			c.snakeID = s.ID
			c.queue(c.hub.snapshotBytes(c.snakeID))
		case "input":
			if c.snakeID == "" {
				continue
			}
			c.hub.game.SetDirection(c.snakeID, game.ParseDirection(msg.Dir))
		}
	}
}

func (c *Client) writePump() {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case <-c.done:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case msg := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-pingTicker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
