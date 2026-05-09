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
	FieldW = 30
	FieldH = 30
	TickHz = 10
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

type Snake struct {
	ID         string
	Name       string
	Color      string
	Body       []Position
	Dir        Direction
	NextDir    Direction
	PeakLength int
}

func (s *Snake) clone() *Snake {
	body := make([]Position, len(s.Body))
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

type SnakeMove struct {
	ID     string
	Head   Position
	Grew   bool
	Dead   bool
	Length int
}

type TickEvent struct {
	Tick   uint64
	Moves  []SnakeMove
	Joins  []*Snake
	Leaves []string
	Pellet *Position
}

type Snapshot struct {
	Tick   uint64
	FieldW int
	FieldH int
	Snakes []*Snake
	Pellet Position
}

type Game struct {
	mu            sync.Mutex
	tick          uint64
	snakes        map[string]*Snake
	pellet        Position
	pendingJoins  []*Snake
	pendingLeaves []string
	rng           *rand.Rand
}

func NewGame() *Game {
	g := &Game{
		snakes: map[string]*Snake{},
		rng: rand.New(rand.NewPCG(
			uint64(time.Now().UnixNano()),
			0x9e3779b97f4a7c15,
		)),
	}
	g.pellet = g.randomFreeTileLocked()
	return g
}

func (g *Game) Join(name string) *Snake {
	g.mu.Lock()
	defer g.mu.Unlock()
	s := &Snake{
		ID:         randomID(),
		Name:       sanitizeName(name),
		Color:      pickColor(g.rng),
		Body:       []Position{g.randomFreeTileLocked()},
		PeakLength: 1,
	}
	s.Dir = randomDir(g.rng)
	s.NextDir = s.Dir
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
	return Snapshot{
		Tick:   g.tick,
		FieldW: FieldW,
		FieldH: FieldH,
		Snakes: snakes,
		Pellet: g.pellet,
	}
}

// Step advances the world by one tick and returns a description of what
// changed, suitable for broadcasting as a delta.
func (g *Game) Step() TickEvent {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tick++
	ev := TickEvent{
		Tick:   g.tick,
		Joins:  g.pendingJoins,
		Leaves: g.pendingLeaves,
	}
	g.pendingJoins = nil
	g.pendingLeaves = nil

	if len(g.snakes) == 0 {
		return ev
	}

	type plan struct {
		s       *Snake
		newHead Position
	}
	plans := make([]plan, 0, len(g.snakes))
	for _, s := range g.snakes {
		dir := s.NextDir
		if dir == DirNone || (len(s.Body) > 1 && dir == opposite(s.Dir)) {
			dir = s.Dir
		}
		s.Dir = dir
		nh := wrap(step(s.Body[0], dir))
		plans = append(plans, plan{s, nh})
	}

	headCount := map[Position]int{}
	for _, p := range plans {
		headCount[p.newHead]++
	}

	eaten := map[*Snake]bool{}
	for _, p := range plans {
		if p.newHead == g.pellet {
			eaten[p.s] = true
		}
	}

	occupied := map[Position]bool{}
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

	pelletEaten := false
	for _, p := range plans {
		if deaths[p.s] {
			continue
		}
		newBody := make([]Position, 0, len(p.s.Body)+1)
		newBody = append(newBody, p.newHead)
		if eaten[p.s] {
			newBody = append(newBody, p.s.Body...)
			pelletEaten = true
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

	if pelletEaten {
		g.pellet = g.randomFreeTileLocked()
	}

	for s := range deaths {
		spawn := g.randomFreeTileLocked()
		s.Body = []Position{spawn}
		s.Dir = randomDir(g.rng)
		s.NextDir = s.Dir
		ev.Moves = append(ev.Moves, SnakeMove{
			ID:     s.ID,
			Head:   spawn,
			Dead:   true,
			Length: 1,
		})
	}

	if pelletEaten {
		p := g.pellet
		ev.Pellet = &p
	}
	return ev
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

func wrap(p Position) Position {
	if p.X < 0 {
		p.X = FieldW - 1
	} else if p.X >= FieldW {
		p.X = 0
	}
	if p.Y < 0 {
		p.Y = FieldH - 1
	} else if p.Y >= FieldH {
		p.Y = 0
	}
	return p
}

func (g *Game) randomFreeTileLocked() Position {
	occupied := map[Position]bool{g.pellet: true}
	for _, s := range g.snakes {
		for _, p := range s.Body {
			occupied[p] = true
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
