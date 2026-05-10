package game

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	FieldW           = 30
	FieldH           = 30
	TickHz           = 10
	PlayersPerField  = 5
)

type Direction uint8

const (
	DirNone Direction = iota
	DirUp
	DirRight
	DirDown
	DirLeft
)

func (d Direction) String() string {
	switch d {
	case DirUp:
		return "up"
	case DirRight:
		return "right"
	case DirDown:
		return "down"
	case DirLeft:
		return "left"
	}
	return ""
}

func ParseDirection(s string) Direction {
	switch strings.ToLower(s) {
	case "up", "u":
		return DirUp
	case "right", "r":
		return DirRight
	case "down", "d":
		return DirDown
	case "left", "l":
		return DirLeft
	}
	return DirNone
}

func opposite(d Direction) Direction {
	switch d {
	case DirUp:
		return DirDown
	case DirDown:
		return DirUp
	case DirLeft:
		return DirRight
	case DirRight:
		return DirLeft
	}
	return DirNone
}

type Position struct {
	X, Y int
}

func (p Position) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%d,%d]", p.X, p.Y)), nil
}

func (p *Position) UnmarshalJSON(data []byte) error {
	var arr [2]int
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	p.X, p.Y = arr[0], arr[1]
	return nil
}

type FieldID string

// Tile identifies a position on a specific field.
type Tile struct {
	Field FieldID  `json:"f"`
	Pos   Position `json:"p"`
}

type Snake struct {
	ID         string
	Name       string
	Color      string
	Body       []Tile
	Dir        Direction
	NextDir    Direction
	PeakLength int
}

func (s *Snake) clone() *Snake {
	body := make([]Tile, len(s.Body))
	copy(body, s.Body)
	return &Snake{
		ID:         s.ID,
		Name:       s.Name,
		Color:      s.Color,
		Body:       body,
		Dir:        s.Dir,
		NextDir:    s.NextDir,
		PeakLength: s.PeakLength,
	}
}

type Field struct {
	ID     FieldID
	Pellet Position
}

func (f *Field) clone() *Field {
	return &Field{ID: f.ID, Pellet: f.Pellet}
}

type SnakeMove struct {
	ID     string
	Head   Tile
	Grew   bool
	Dead   bool
	Length int
}

type PelletChange struct {
	Field FieldID
	Pos   Position
}

type TickEvent struct {
	Tick        uint64
	Moves       []SnakeMove
	Joins       []*Snake
	Leaves      []string
	FieldJoins  []*Field
	FieldLeaves []FieldID
	Pellets     []PelletChange
}

type Snapshot struct {
	Tick    uint64
	FieldW  int
	FieldH  int
	Fields  []*Field
	Snakes  []*Snake
}

type Game struct {
	mu                 sync.Mutex
	tick               uint64
	nextFieldNum       int
	fields             map[FieldID]*Field
	snakes             map[string]*Snake
	pendingJoins       []*Snake
	pendingLeaves      []string
	pendingFieldJoins  []*Field
	pendingFieldLeaves []FieldID
	rng                *rand.Rand
}

func NewGame() *Game {
	g := &Game{
		fields: map[FieldID]*Field{},
		snakes: map[string]*Snake{},
		rng: rand.New(rand.NewPCG(
			uint64(time.Now().UnixNano()),
			0x9e3779b97f4a7c15,
		)),
	}
	g.spawnFieldLocked()
	return g
}

func (g *Game) spawnFieldLocked() *Field {
	g.nextFieldNum++
	f := &Field{
		ID:     FieldID(fmt.Sprintf("f%d", g.nextFieldNum)),
		Pellet: Position{-1, -1},
	}
	g.fields[f.ID] = f
	f.Pellet = g.randomFreeTileInFieldLocked(f.ID)
	g.pendingFieldJoins = append(g.pendingFieldJoins, f.clone())
	return f
}

func (g *Game) Join(name string) *Snake {
	g.mu.Lock()
	defer g.mu.Unlock()

	target := (len(g.snakes) + 1 + PlayersPerField - 1) / PlayersPerField
	if target < 1 {
		target = 1
	}
	for len(g.fields) < target {
		g.spawnFieldLocked()
	}

	fid := g.randomFieldIDLocked()
	spawn := g.randomFreeTileInFieldLocked(fid)
	s := &Snake{
		ID:    randomID(),
		Name:  sanitizeName(name),
		Color: pickColor(g.rng),
		Body:  []Tile{{Field: fid, Pos: spawn}},
	}
	s.Dir = randomDir(g.rng)
	s.NextDir = s.Dir
	s.PeakLength = 1
	g.snakes[s.ID] = s
	g.pendingJoins = append(g.pendingJoins, s.clone())
	return s.clone()
}

func (g *Game) Leave(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.snakes[id]; !ok {
		return
	}
	delete(g.snakes, id)
	g.pendingLeaves = append(g.pendingLeaves, id)
}

func (g *Game) SetDirection(id string, dir Direction) {
	g.mu.Lock()
	defer g.mu.Unlock()
	s, ok := g.snakes[id]
	if !ok || dir == DirNone {
		return
	}
	if len(s.Body) > 1 && dir == opposite(s.Dir) {
		return
	}
	s.NextDir = dir
}

func (g *Game) Snapshot() Snapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	snakes := make([]*Snake, 0, len(g.snakes))
	for _, s := range g.snakes {
		snakes = append(snakes, s.clone())
	}
	fields := make([]*Field, 0, len(g.fields))
	for _, f := range g.fields {
		fields = append(fields, f.clone())
	}
	return Snapshot{
		Tick:   g.tick,
		FieldW: FieldW,
		FieldH: FieldH,
		Fields: fields,
		Snakes: snakes,
	}
}

// Step advances the world by one tick.
func (g *Game) Step() TickEvent {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tick++

	ev := TickEvent{
		Tick:        g.tick,
		Joins:       g.pendingJoins,
		Leaves:      g.pendingLeaves,
		FieldJoins:  g.pendingFieldJoins,
		FieldLeaves: g.pendingFieldLeaves,
	}
	g.pendingJoins = nil
	g.pendingLeaves = nil
	g.pendingFieldJoins = nil
	g.pendingFieldLeaves = nil

	if len(g.snakes) == 0 {
		return ev
	}

	type plan struct {
		s       *Snake
		newHead Tile
	}
	plans := make([]plan, 0, len(g.snakes))
	for _, s := range g.snakes {
		dir := s.NextDir
		if dir == DirNone || (len(s.Body) > 1 && dir == opposite(s.Dir)) {
			dir = s.Dir
		}
		s.Dir = dir
		head := s.Body[0]
		nh := stepTile(head, dir, g)
		plans = append(plans, plan{s, nh})
	}

	headCount := map[Tile]int{}
	for _, p := range plans {
		headCount[p.newHead]++
	}

	eaten := map[*Snake]bool{}
	for _, p := range plans {
		f := g.fields[p.newHead.Field]
		if f != nil && p.newHead.Pos == f.Pellet {
			eaten[p.s] = true
		}
	}

	occupied := map[Tile]bool{}
	for _, s := range g.snakes {
		end := len(s.Body)
		if !eaten[s] {
			end-- // tail vacates this tick
		}
		for i := 0; i < end; i++ {
			occupied[s.Body[i]] = true
		}
	}

	deaths := map[*Snake]bool{}
	for _, p := range plans {
		if headCount[p.newHead] > 1 {
			deaths[p.s] = true
			continue
		}
		if occupied[p.newHead] {
			deaths[p.s] = true
		}
	}

	pelletEaten := map[FieldID]bool{}
	for _, p := range plans {
		if deaths[p.s] {
			continue
		}
		newBody := make([]Tile, 0, len(p.s.Body)+1)
		newBody = append(newBody, p.newHead)
		if eaten[p.s] {
			newBody = append(newBody, p.s.Body...)
			pelletEaten[p.newHead.Field] = true
		} else {
			newBody = append(newBody, p.s.Body[:len(p.s.Body)-1]...)
		}
		p.s.Body = newBody
		if len(p.s.Body) > p.s.PeakLength {
			p.s.PeakLength = len(p.s.Body)
		}
		ev.Moves = append(ev.Moves, SnakeMove{
			ID:     p.s.ID,
			Head:   p.newHead,
			Grew:   eaten[p.s],
			Length: len(p.s.Body),
		})
	}

	for fid := range pelletEaten {
		f := g.fields[fid]
		if f == nil {
			continue
		}
		f.Pellet = g.randomFreeTileInFieldLocked(fid)
		ev.Pellets = append(ev.Pellets, PelletChange{Field: fid, Pos: f.Pellet})
	}

	for s := range deaths {
		fid := g.randomFieldIDLocked()
		spawn := g.randomFreeTileInFieldLocked(fid)
		s.Body = []Tile{{Field: fid, Pos: spawn}}
		s.Dir = randomDir(g.rng)
		s.NextDir = s.Dir
		ev.Moves = append(ev.Moves, SnakeMove{
			ID:     s.ID,
			Head:   Tile{Field: fid, Pos: spawn},
			Dead:   true,
			Length: 1,
		})
	}

	g.destroyEmptyFieldsLocked(&ev)
	return ev
}

// destroyEmptyFieldsLocked removes any field with no snake body cells, except
// the final remaining field, and appends destroyed IDs to the event.
func (g *Game) destroyEmptyFieldsLocked(ev *TickEvent) {
	if len(g.fields) <= 1 {
		return
	}
	occupied := map[FieldID]bool{}
	for _, s := range g.snakes {
		for _, t := range s.Body {
			occupied[t.Field] = true
		}
	}
	for id := range g.fields {
		if !occupied[id] && len(g.fields) > 1 {
			delete(g.fields, id)
			ev.FieldLeaves = append(ev.FieldLeaves, id)
		}
	}
}

func stepTile(t Tile, d Direction, g *Game) Tile {
	np := step(t.Pos, d)
	if np.X >= 0 && np.X < FieldW && np.Y >= 0 && np.Y < FieldH {
		return Tile{Field: t.Field, Pos: np}
	}
	dest := g.pickTeleportDestinationLocked(t.Field)
	return Tile{Field: dest, Pos: oppositeEdgePosition(d, t.Pos)}
}

// pickTeleportDestinationLocked picks a random field other than `from`. If
// `from` is the only field, it returns `from` (single-field wrap).
func (g *Game) pickTeleportDestinationLocked(from FieldID) FieldID {
	if len(g.fields) <= 1 {
		return from
	}
	others := make([]FieldID, 0, len(g.fields)-1)
	for id := range g.fields {
		if id != from {
			others = append(others, id)
		}
	}
	return others[g.rng.IntN(len(others))]
}

// oppositeEdgePosition returns the entry position on the edge opposite the
// one the snake exited, preserving the perpendicular coordinate of the exit.
func oppositeEdgePosition(exitDir Direction, exitFromPos Position) Position {
	switch exitDir {
	case DirRight:
		return Position{X: 0, Y: clampY(exitFromPos.Y)}
	case DirLeft:
		return Position{X: FieldW - 1, Y: clampY(exitFromPos.Y)}
	case DirDown:
		return Position{X: clampX(exitFromPos.X), Y: 0}
	case DirUp:
		return Position{X: clampX(exitFromPos.X), Y: FieldH - 1}
	}
	return Position{X: 0, Y: 0}
}

func clampX(x int) int {
	if x < 0 {
		return 0
	}
	if x >= FieldW {
		return FieldW - 1
	}
	return x
}

func clampY(y int) int {
	if y < 0 {
		return 0
	}
	if y >= FieldH {
		return FieldH - 1
	}
	return y
}

func step(p Position, d Direction) Position {
	switch d {
	case DirUp:
		p.Y--
	case DirDown:
		p.Y++
	case DirLeft:
		p.X--
	case DirRight:
		p.X++
	}
	return p
}

func (g *Game) randomFieldIDLocked() FieldID {
	ids := make([]FieldID, 0, len(g.fields))
	for id := range g.fields {
		ids = append(ids, id)
	}
	return ids[g.rng.IntN(len(ids))]
}

func (g *Game) randomFreeTileInFieldLocked(fid FieldID) Position {
	occupied := map[Position]bool{}
	if f := g.fields[fid]; f != nil && f.Pellet.X >= 0 {
		occupied[f.Pellet] = true
	}
	for _, s := range g.snakes {
		for _, t := range s.Body {
			if t.Field == fid {
				occupied[t.Pos] = true
			}
		}
	}
	if len(occupied) >= FieldW*FieldH {
		return Position{g.rng.IntN(FieldW), g.rng.IntN(FieldH)}
	}
	for {
		p := Position{g.rng.IntN(FieldW), g.rng.IntN(FieldH)}
		if !occupied[p] {
			return p
		}
	}
}

func randomDir(r *rand.Rand) Direction {
	return Direction(r.IntN(4)) + DirUp
}

func randomID() string {
	b := make([]byte, 6)
	_, _ = cryptorand.Read(b)
	return hex.EncodeToString(b)
}

var palette = []string{
	"#e6194B", "#3cb44b", "#ffe119", "#4363d8",
	"#f58231", "#911eb4", "#42d4f4", "#f032e6",
	"#bfef45", "#fabed4", "#469990", "#dcbeff",
}

func pickColor(r *rand.Rand) string {
	return palette[r.IntN(len(palette))]
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	if s == "" {
		s = "anon"
	}
	if utf8.RuneCountInString(s) > 20 {
		runes := []rune(s)
		s = string(runes[:20])
	}
	return s
}
