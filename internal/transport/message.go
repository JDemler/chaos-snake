package transport

import "chaos-snake/internal/game"

type wireField struct {
	W      int `json:"w"`
	H      int `json:"h"`
	TickHz int `json:"tick_hz"`
}

type wireSnake struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Color string          `json:"color"`
	Body  []game.Position `json:"body"`
	Dir   string          `json:"dir"`
}

type wireSnapshot struct {
	Type   string        `json:"type"`
	Tick   uint64        `json:"tick"`
	Field  wireField     `json:"field"`
	Snakes []wireSnake   `json:"snakes"`
	Pellet game.Position `json:"pellet"`
	You    string        `json:"you"`
}

type wireMove struct {
	ID     string        `json:"id"`
	Head   game.Position `json:"head"`
	Grew   bool          `json:"grew,omitempty"`
	Dead   bool          `json:"dead,omitempty"`
	Length int           `json:"length"`
}

type wireDelta struct {
	Type   string         `json:"type"`
	Tick   uint64         `json:"tick"`
	Moves  []wireMove     `json:"moves,omitempty"`
	Joins  []wireSnake    `json:"joins,omitempty"`
	Leaves []string       `json:"leaves,omitempty"`
	Pellet *game.Position `json:"pellet,omitempty"`
}

type clientMessage struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	Dir  string `json:"dir,omitempty"`
}

func toWireSnake(s *game.Snake) wireSnake {
	return wireSnake{
		ID:    s.ID,
		Name:  s.Name,
		Color: s.Color,
		Body:  s.Body,
		Dir:   s.Dir.String(),
	}
}

func makeSnapshot(s game.Snapshot, you string) wireSnapshot {
	snakes := make([]wireSnake, 0, len(s.Snakes))
	for _, sn := range s.Snakes {
		snakes = append(snakes, toWireSnake(sn))
	}
	return wireSnapshot{
		Type:   "snapshot",
		Tick:   s.Tick,
		Field:  wireField{W: s.FieldW, H: s.FieldH, TickHz: game.TickHz},
		Snakes: snakes,
		Pellet: s.Pellet,
		You:    you,
	}
}

func makeDelta(ev game.TickEvent) wireDelta {
	moves := make([]wireMove, 0, len(ev.Moves))
	for _, m := range ev.Moves {
		moves = append(moves, wireMove{
			ID:     m.ID,
			Head:   m.Head,
			Grew:   m.Grew,
			Dead:   m.Dead,
			Length: m.Length,
		})
	}
	joins := make([]wireSnake, 0, len(ev.Joins))
	for _, sn := range ev.Joins {
		joins = append(joins, toWireSnake(sn))
	}
	return wireDelta{
		Type:   "delta",
		Tick:   ev.Tick,
		Moves:  moves,
		Joins:  joins,
		Leaves: ev.Leaves,
		Pellet: ev.Pellet,
	}
}

func emptyDelta(d wireDelta) bool {
	return len(d.Moves) == 0 && len(d.Joins) == 0 && len(d.Leaves) == 0 && d.Pellet == nil
}
