package transport

import "chaos-snake/internal/game"

type wireField struct {
	ID     game.FieldID  `json:"id"`
	Pellet game.Position `json:"pellet"`
}

type wireFieldSize struct {
	W      int `json:"w"`
	H      int `json:"h"`
	TickHz int `json:"tick_hz"`
}

type wireSnake struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Color string      `json:"color"`
	Body  []game.Tile `json:"body"`
	Dir   string      `json:"dir"`
}

type wireSnapshot struct {
	Type      string        `json:"type"`
	Tick      uint64        `json:"tick"`
	FieldSize wireFieldSize `json:"field_size"`
	Fields    []wireField   `json:"fields"`
	Snakes    []wireSnake   `json:"snakes"`
	You       string        `json:"you"`
}

type wireMove struct {
	ID     string    `json:"id"`
	Head   game.Tile `json:"head"`
	Grew   bool      `json:"grew,omitempty"`
	Dead   bool      `json:"dead,omitempty"`
	Length int       `json:"length"`
}

type wirePellet struct {
	Field game.FieldID  `json:"f"`
	Pos   game.Position `json:"p"`
}

type wireDelta struct {
	Type        string         `json:"type"`
	Tick        uint64         `json:"tick"`
	Moves       []wireMove     `json:"moves,omitempty"`
	Joins       []wireSnake    `json:"joins,omitempty"`
	Leaves      []string       `json:"leaves,omitempty"`
	FieldJoins  []wireField    `json:"field_joins,omitempty"`
	FieldLeaves []game.FieldID `json:"field_leaves,omitempty"`
	Pellets     []wirePellet   `json:"pellets,omitempty"`
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

func toWireField(f *game.Field) wireField {
	return wireField{ID: f.ID, Pellet: f.Pellet}
}

func makeSnapshot(s game.Snapshot, you string) wireSnapshot {
	snakes := make([]wireSnake, 0, len(s.Snakes))
	for _, sn := range s.Snakes {
		snakes = append(snakes, toWireSnake(sn))
	}
	fields := make([]wireField, 0, len(s.Fields))
	for _, f := range s.Fields {
		fields = append(fields, toWireField(f))
	}
	return wireSnapshot{
		Type:      "snapshot",
		Tick:      s.Tick,
		FieldSize: wireFieldSize{W: s.FieldW, H: s.FieldH, TickHz: game.TickHz},
		Fields:    fields,
		Snakes:    snakes,
		You:       you,
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
	fieldJoins := make([]wireField, 0, len(ev.FieldJoins))
	for _, f := range ev.FieldJoins {
		fieldJoins = append(fieldJoins, toWireField(f))
	}
	pellets := make([]wirePellet, 0, len(ev.Pellets))
	for _, p := range ev.Pellets {
		pellets = append(pellets, wirePellet{Field: p.Field, Pos: p.Pos})
	}
	return wireDelta{
		Type:        "delta",
		Tick:        ev.Tick,
		Moves:       moves,
		Joins:       joins,
		Leaves:      ev.Leaves,
		FieldJoins:  fieldJoins,
		FieldLeaves: ev.FieldLeaves,
		Pellets:     pellets,
	}
}

func emptyDelta(d wireDelta) bool {
	return len(d.Moves) == 0 &&
		len(d.Joins) == 0 &&
		len(d.Leaves) == 0 &&
		len(d.FieldJoins) == 0 &&
		len(d.FieldLeaves) == 0 &&
		len(d.Pellets) == 0
}
