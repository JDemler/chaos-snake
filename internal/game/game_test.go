package game

import (
	"testing"
)

// firstFieldID returns one of the fields' IDs (used in tests with one field).
func firstFieldID(g *Game) FieldID {
	for id := range g.fields {
		return id
	}
	return ""
}

func setSnake(g *Game, id string, body []Tile, dir Direction) {
	s := g.snakes[id]
	s.Body = body
	s.Dir = dir
	s.NextDir = dir
	s.PeakLength = len(body)
}

func TestSnakeMovesForwardOneTilePerTick(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	g.fields[fid].Pellet = Position{0, 0}
	s := g.Join("a")
	setSnake(g, s.ID, []Tile{{fid, Position{10, 10}}}, DirRight)

	g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 1 {
		t.Fatalf("expected length 1, got %d", len(got.Body))
	}
	if got.Body[0] != (Tile{fid, Position{11, 10}}) {
		t.Fatalf("expected head at %v(11,10), got %v", fid, got.Body[0])
	}
}

func TestSingleFieldEdgeWraps(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	g.fields[fid].Pellet = Position{0, 29}
	s := g.Join("a")
	setSnake(g, s.ID, []Tile{{fid, Position{29, 10}}}, DirRight)

	g.Step()

	got := g.snakes[s.ID]
	if got.Body[0] != (Tile{fid, Position{0, 10}}) {
		t.Fatalf("single-field wrap to (0,10) expected, got %v", got.Body[0])
	}
}

func TestMultiFieldTeleportPreservesPerpendicularCoord(t *testing.T) {
	g := NewGame()
	f1 := firstFieldID(g)
	g.fields[f1].Pellet = Position{0, 0}
	f2 := g.spawnFieldLocked()
	g.fields[f2.ID].Pellet = Position{0, 0}

	s := g.Join("a")
	// Force the snake onto f1 right edge, going right.
	setSnake(g, s.ID, []Tile{{f1, Position{29, 17}}}, DirRight)

	g.Step()

	got := g.snakes[s.ID]
	if got.Body[0].Field != f2.ID {
		t.Fatalf("expected teleport to %v, got %v", f2.ID, got.Body[0].Field)
	}
	if got.Body[0].Pos != (Position{0, 17}) {
		t.Fatalf("expected entry at (0,17) on opposite edge, got %v", got.Body[0].Pos)
	}
}

func TestPelletGrowsSnakeAndRespawns(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	g.fields[fid].Pellet = Position{11, 10}
	s := g.Join("a")
	setSnake(g, s.ID, []Tile{{fid, Position{10, 10}}}, DirRight)

	ev := g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 2 {
		t.Fatalf("expected length 2 after eating, got %d", len(got.Body))
	}
	if g.fields[fid].Pellet == (Position{11, 10}) {
		t.Fatalf("pellet should respawn away from eaten tile")
	}
	if len(ev.Pellets) != 1 || ev.Pellets[0].Field != fid {
		t.Fatalf("expected one pellet change for %v, got %v", fid, ev.Pellets)
	}
}

func TestSelfCollisionRespawns(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	g.fields[fid].Pellet = Position{0, 0}
	s := g.Join("a")
	body := []Tile{
		{fid, Position{5, 5}}, {fid, Position{5, 4}}, {fid, Position{5, 3}},
		{fid, Position{4, 3}}, {fid, Position{4, 4}}, {fid, Position{4, 5}},
		{fid, Position{4, 6}},
	}
	setSnake(g, s.ID, body, DirDown)
	g.snakes[s.ID].NextDir = DirLeft

	ev := g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 1 {
		t.Fatalf("expected respawn (length 1), got %d", len(got.Body))
	}
	dead := false
	for _, m := range ev.Moves {
		if m.ID == s.ID && m.Dead {
			dead = true
		}
	}
	if !dead {
		t.Fatalf("event should report death+respawn for the snake")
	}
}

func TestFollowingOwnTailDoesNotKill(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	g.fields[fid].Pellet = Position{0, 0}
	s := g.Join("a")
	body := []Tile{
		{fid, Position{5, 5}}, {fid, Position{6, 5}},
		{fid, Position{6, 6}}, {fid, Position{5, 6}},
	}
	setSnake(g, s.ID, body, DirLeft)
	g.snakes[s.ID].NextDir = DirDown

	g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 4 {
		t.Fatalf("snake should survive following its own tail; got length %d", len(got.Body))
	}
}

func TestHeadOnHeadKillsBoth(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	g.fields[fid].Pellet = Position{0, 0}

	a := g.Join("a")
	setSnake(g, a.ID, []Tile{{fid, Position{5, 5}}, {fid, Position{4, 5}}}, DirRight)
	b := g.Join("b")
	setSnake(g, b.ID, []Tile{{fid, Position{7, 5}}, {fid, Position{8, 5}}}, DirLeft)

	g.Step()

	if len(g.snakes[a.ID].Body) != 1 {
		t.Fatalf("a should respawn at length 1, got %d", len(g.snakes[a.ID].Body))
	}
	if len(g.snakes[b.ID].Body) != 1 {
		t.Fatalf("b should respawn at length 1, got %d", len(g.snakes[b.ID].Body))
	}
}

func TestRunningIntoOtherBodyKillsMover(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	g.fields[fid].Pellet = Position{0, 0}

	a := g.Join("a")
	setSnake(g, a.ID, []Tile{
		{fid, Position{5, 5}}, {fid, Position{4, 5}}, {fid, Position{3, 5}},
	}, DirRight)
	b := g.Join("b")
	setSnake(g, b.ID, []Tile{{fid, Position{4, 4}}, {fid, Position{4, 3}}}, DirDown)

	g.Step()

	if len(g.snakes[b.ID].Body) != 1 {
		t.Fatalf("b should respawn (hit a's body), got length %d", len(g.snakes[b.ID].Body))
	}
	if len(g.snakes[a.ID].Body) != 3 {
		t.Fatalf("a should survive at length 3, got %d", len(g.snakes[a.ID].Body))
	}
}

func TestReverseDirectionIsIgnoredForLongerSnake(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	g.fields[fid].Pellet = Position{0, 0}
	s := g.Join("a")
	setSnake(g, s.ID, []Tile{{fid, Position{5, 5}}, {fid, Position{4, 5}}}, DirRight)

	g.SetDirection(s.ID, DirLeft)
	if g.snakes[s.ID].NextDir == DirLeft {
		t.Fatalf("180-degree reverse should be rejected by SetDirection")
	}

	g.Step()

	got := g.snakes[s.ID]
	if got.Body[0] != (Tile{fid, Position{6, 5}}) {
		t.Fatalf("snake should have continued right to (6,5), got %v", got.Body[0])
	}
}

func TestFieldSpawnsOnSixthJoin(t *testing.T) {
	g := NewGame()
	for i := 0; i < 5; i++ {
		g.Join("p")
	}
	if len(g.fields) != 1 {
		t.Fatalf("expected 1 field with 5 players, got %d", len(g.fields))
	}
	g.Join("sixth")
	if len(g.fields) != 2 {
		t.Fatalf("expected 2 fields after 6th player, got %d", len(g.fields))
	}
}

func TestFieldSpawnsScaleAtEleventhJoin(t *testing.T) {
	g := NewGame()
	for i := 0; i < 10; i++ {
		g.Join("p")
	}
	if len(g.fields) != 2 {
		t.Fatalf("expected 2 fields with 10 players, got %d", len(g.fields))
	}
	g.Join("eleventh")
	if len(g.fields) != 3 {
		t.Fatalf("expected 3 fields after 11th player, got %d", len(g.fields))
	}
}

func TestExcessEmptyFieldDestroyed(t *testing.T) {
	g := NewGame()
	f1 := firstFieldID(g)
	g.fields[f1].Pellet = Position{0, 0}
	f2 := g.spawnFieldLocked()
	g.fields[f2.ID].Pellet = Position{0, 0}

	// 1 snake -> target = 1, current = 2, excess = 1.
	s := g.Join("a")
	setSnake(g, s.ID, []Tile{{f2.ID, Position{15, 15}}}, DirRight)

	g.Step()

	if _, ok := g.fields[f1]; ok {
		t.Fatalf("excess empty field %v should have been destroyed", f1)
	}
	if _, ok := g.fields[f2.ID]; !ok {
		t.Fatalf("non-empty field %v should still exist", f2.ID)
	}
}

func TestEmptyFieldKeptWhenAtTarget(t *testing.T) {
	g := NewGame()
	fid1 := firstFieldID(g)
	g.fields[fid1].Pellet = Position{0, 0}

	// 6 joins -> auto-scales to 2 fields (target=2)
	snakes := make([]*Snake, 0, 6)
	for i := 0; i < 6; i++ {
		snakes = append(snakes, g.Join("p"))
	}
	if len(g.fields) != 2 {
		t.Fatalf("setup: expected 2 fields after 6 joins, got %d", len(g.fields))
	}

	// Force all 6 onto fid1, leaving the other field empty.
	for i, s := range snakes {
		setSnake(g, s.ID, []Tile{{fid1, Position{i, 15}}}, DirRight)
	}

	g.Step()

	if len(g.fields) != 2 {
		t.Fatalf("empty field at-target should be kept; got %d fields", len(g.fields))
	}
}

func TestLastFieldIsNotDestroyedWhenEmpty(t *testing.T) {
	g := NewGame()
	// No snakes, one field, target=1, excess=0 — must remain.
	g.Step()
	if len(g.fields) != 1 {
		t.Fatalf("last field must be preserved; got %d fields", len(g.fields))
	}
}

func TestExcessEmptyFieldsDestroyedWithNoPlayers(t *testing.T) {
	g := NewGame() // f1
	g.spawnFieldLocked()
	g.spawnFieldLocked()
	// 3 fields, 0 players, target=1, excess=2 — destroy 2 empties.
	g.Step()
	if len(g.fields) != 1 {
		t.Fatalf("expected 1 field after destroying excess empties; got %d", len(g.fields))
	}
}

func TestSnakeStraddlesTwoFieldsAfterCrossing(t *testing.T) {
	g := NewGame()
	f1 := firstFieldID(g)
	g.fields[f1].Pellet = Position{0, 0}
	f2 := g.spawnFieldLocked()
	g.fields[f2.ID].Pellet = Position{0, 0}

	s := g.Join("a")
	// Length-3 snake near f1's right edge moving right.
	setSnake(g, s.ID, []Tile{
		{f1, Position{29, 10}},
		{f1, Position{28, 10}},
		{f1, Position{27, 10}},
	}, DirRight)

	g.Step()

	got := g.snakes[s.ID]
	if got.Body[0].Field != f2.ID {
		t.Fatalf("head should be on %v after crossing, got %v", f2.ID, got.Body[0].Field)
	}
	for i := 1; i < len(got.Body); i++ {
		if got.Body[i].Field != f1 {
			t.Fatalf("body cell %d should still be on f1, got %v", i, got.Body[i].Field)
		}
	}
}

func TestSanitizeNameStripsControlChars(t *testing.T) {
	got := sanitizeName("  he\x00\x07llo  ")
	if got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
	if sanitizeName("") != "anon" {
		t.Fatalf("empty name should become 'anon'")
	}
}
