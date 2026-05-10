// Package admin exposes the unauthenticated /admin HTTP surface for managing
// bots. It serves a small static page and a handful of JSON endpoints.
package admin

import (
	"encoding/json"
	_ "embed"
	"net/http"

	"chaos-snake/internal/game"
)

//go:embed admin.html
var adminHTML []byte

// NewHandler returns an http.Handler that serves the admin page at "/admin"
// and JSON endpoints at "/admin/api/...". The handler must be registered with
// a mux at the prefix "/admin/" (and "/admin" exact, if served separately).
func NewHandler(g *game.Game) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin", servePage)
	mux.HandleFunc("/admin/", servePage)
	mux.HandleFunc("/admin/api/state", func(w http.ResponseWriter, r *http.Request) { handleState(w, r, g) })
	mux.HandleFunc("/admin/api/bots/add", func(w http.ResponseWriter, r *http.Request) { handleAdd(w, r, g) })
	mux.HandleFunc("/admin/api/bots/remove", func(w http.ResponseWriter, r *http.Request) { handleRemove(w, r, g) })
	mux.HandleFunc("/admin/api/bots/remove_all", func(w http.ResponseWriter, r *http.Request) { handleRemoveAll(w, r, g) })
	mux.HandleFunc("/admin/api/target", func(w http.ResponseWriter, r *http.Request) { handleTarget(w, r, g) })
	mux.HandleFunc("/admin/api/pellets", func(w http.ResponseWriter, r *http.Request) { handlePellets(w, r, g) })
	return mux
}

func servePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin" && r.URL.Path != "/admin/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(adminHTML)
}

type stateSnake struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	IsBot  bool   `json:"is_bot"`
	Length int    `json:"length"`
}

type stateResponse struct {
	Humans          int          `json:"humans"`
	Bots            int          `json:"bots"`
	Target          int          `json:"target"` // -1 means unset
	PelletsPerField int          `json:"pellets_per_field"`
	Snakes          []stateSnake `json:"snakes"`
}

func handleState(w http.ResponseWriter, r *http.Request, g *game.Game) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeState(w, g)
}

func handleAdd(w http.ResponseWriter, r *http.Request, g *game.Game) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	g.AddBot()
	writeState(w, g)
}

func handleRemove(w http.ResponseWriter, r *http.Request, g *game.Game) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	g.RemoveBot(body.ID)
	writeState(w, g)
}

func handleRemoveAll(w http.ResponseWriter, r *http.Request, g *game.Game) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	g.RemoveAllBots()
	writeState(w, g)
}

func handlePellets(w http.ResponseWriter, r *http.Request, g *game.Game) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Value *int `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Value == nil {
		http.Error(w, "missing value", http.StatusBadRequest)
		return
	}
	g.SetPelletsPerField(*body.Value)
	writeState(w, g)
}

func handleTarget(w http.ResponseWriter, r *http.Request, g *game.Game) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Target *int `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	if body.Target == nil || *body.Target < 0 {
		g.ClearTargetCount()
	} else {
		g.SetTargetCount(*body.Target)
	}
	writeState(w, g)
}

func writeState(w http.ResponseWriter, g *game.Game) {
	snap := g.Snapshot()
	resp := stateResponse{
		Target:          g.TargetCount(),
		PelletsPerField: g.PelletsPerField(),
		Snakes:          make([]stateSnake, 0, len(snap.Snakes)),
	}
	for _, s := range snap.Snakes {
		if s.IsBot {
			resp.Bots++
		} else {
			resp.Humans++
		}
		resp.Snakes = append(resp.Snakes, stateSnake{
			ID:     s.ID,
			Name:   s.Name,
			IsBot:  s.IsBot,
			Length: len(s.Body),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
