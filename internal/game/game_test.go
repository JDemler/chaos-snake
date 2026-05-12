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

// setPellet pins a field to a single pellet at p so per-position assertions
// in legacy single-pellet tests stay deterministic across reconciliation.
func setPellet(g *Game, fid FieldID, p Position) {
	g.pelletsPerField = 1
	g.fields[fid].Pellets = []Position{p}
}

// setPellets pins a field to the given pellet list and configures the global
// target to match so reconciliation does not add or remove anything.
func setPellets(g *Game, fid FieldID, pellets ...Position) {
	g.pelletsPerField = len(pellets)
	g.fields[fid].Pellets = append([]Position(nil), pellets...)
}

func TestSnakeMovesForwardOneTilePerTick(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	setPellet(g, fid, Position{0, 0})
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
	setPellet(g, fid, Position{0, 29})
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
	setPellet(g, f1, Position{0, 0})
	f2 := g.spawnFieldLocked()
	setPellet(g, f2.ID, Position{0, 0})

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
	setPellet(g, fid, Position{11, 10})
	s := g.Join("a")
	setSnake(g, s.ID, []Tile{{fid, Position{10, 10}}}, DirRight)

	ev := g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 2 {
		t.Fatalf("expected length 2 after eating, got %d", len(got.Body))
	}
	if pellets := g.fields[fid].Pellets; len(pellets) != 1 || pellets[0] == (Position{11, 10}) {
		t.Fatalf("pellet should respawn (single, away from eaten tile); got %v", pellets)
	}
	if len(ev.Pellets) != 1 || ev.Pellets[0].Field != fid || len(ev.Pellets[0].Pellets) != 1 {
		t.Fatalf("expected one pellet change with one pellet for %v, got %v", fid, ev.Pellets)
	}
}

func TestSelfCollisionRespawns(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	setPellet(g, fid, Position{0, 0})
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
	setPellet(g, fid, Position{0, 0})
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
	setPellet(g, fid, Position{0, 0})

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
	setPellet(g, fid, Position{0, 0})

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
	setPellet(g, fid, Position{0, 0})
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
	setPellet(g, f1, Position{0, 0})
	f2 := g.spawnFieldLocked()
	setPellet(g, f2.ID, Position{0, 0})

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
	setPellet(g, fid1, Position{0, 0})

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
	setPellet(g, f1, Position{0, 0})
	f2 := g.spawnFieldLocked()
	setPellet(g, f2.ID, Position{0, 0})

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

func TestNewGameSpawnsDefaultPelletsPerField(t *testing.T) {
	g := NewGame()
	if g.PelletsPerField() != DefaultPelletsPerField {
		t.Fatalf("expected default pellets/field = %d, got %d",
			DefaultPelletsPerField, g.PelletsPerField())
	}
	fid := firstFieldID(g)
	if got := len(g.fields[fid].Pellets); got != DefaultPelletsPerField {
		t.Fatalf("expected %d pellets on first field, got %d",
			DefaultPelletsPerField, got)
	}
}

func TestEatingOnePelletKeepsFieldAtTarget(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	// Pin three pellets, one of which the snake will eat.
	setPellets(g, fid,
		Position{11, 10}, // about to be eaten
		Position{0, 0},
		Position{29, 29},
	)
	s := g.Join("a")
	setSnake(g, s.ID, []Tile{{fid, Position{10, 10}}}, DirRight)

	g.Step()

	if got := len(g.fields[fid].Pellets); got != 3 {
		t.Fatalf("field should still carry 3 pellets after one is eaten, got %d", got)
	}
	for _, p := range g.fields[fid].Pellets {
		if p == (Position{11, 10}) {
			t.Fatalf("eaten pellet at (11,10) should not still be present")
		}
	}
}

func TestSetPelletsPerFieldReconcilesUp(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	setPellet(g, fid, Position{0, 0})
	if got := len(g.fields[fid].Pellets); got != 1 {
		t.Fatalf("setup: expected 1 pellet, got %d", got)
	}

	g.SetPelletsPerField(5)
	g.Step()

	if got := len(g.fields[fid].Pellets); got != 5 {
		t.Fatalf("after raising target to 5, expected 5 pellets, got %d", got)
	}
}

func TestSetPelletsPerFieldReconcilesDown(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	setPellets(g, fid,
		Position{0, 0}, Position{1, 1}, Position{2, 2},
		Position{3, 3}, Position{4, 4},
	)

	g.SetPelletsPerField(2)
	g.Step()

	if got := len(g.fields[fid].Pellets); got != 2 {
		t.Fatalf("after lowering target to 2, expected 2 pellets, got %d", got)
	}
}

func TestSetPelletsPerFieldReconcilesAcrossFields(t *testing.T) {
	g := NewGame()
	g.spawnFieldLocked()
	g.spawnFieldLocked()
	g.SetPelletsPerField(4)
	g.Step()

	for fid, f := range g.fields {
		if len(f.Pellets) != 4 {
			t.Fatalf("field %v should carry 4 pellets, got %d", fid, len(f.Pellets))
		}
	}
}

func TestNewFieldIsFilledToCurrentTarget(t *testing.T) {
	g := NewGame()
	g.SetPelletsPerField(2)
	g.Step() // reconcile existing field down to 2
	f := g.spawnFieldLocked()
	if got := len(f.Pellets); got != 2 {
		t.Fatalf("new field should be born with 2 pellets, got %d", got)
	}
}

func TestSetPelletsPerFieldClampsNegativeToZero(t *testing.T) {
	g := NewGame()
	g.SetPelletsPerField(-3)
	if got := g.PelletsPerField(); got != 0 {
		t.Fatalf("negative target should clamp to 0, got %d", got)
	}
	g.Step()
	for _, f := range g.fields {
		if len(f.Pellets) != 0 {
			t.Fatalf("field should have 0 pellets after target=0 reconcile, got %d", len(f.Pellets))
		}
	}
}

func TestLeaderboardRecordsPeakLengthAcrossDeaths(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	setPellet(g, fid, Position{0, 0})

	s := g.Join("rex")
	body := []Tile{
		{fid, Position{5, 5}}, {fid, Position{5, 4}}, {fid, Position{5, 3}},
		{fid, Position{4, 3}}, {fid, Position{4, 4}}, {fid, Position{4, 5}},
		{fid, Position{4, 6}},
	}
	setSnake(g, s.ID, body, DirDown)
	g.snakes[s.ID].NextDir = DirLeft

	g.Step()

	if got := len(g.snakes[s.ID].Body); got != 1 {
		t.Fatalf("snake should be respawned at length 1, got %d", got)
	}

	lb := g.Leaderboard()
	var peak int
	for _, e := range lb {
		if e.Name == "rex" {
			peak = e.Peak
		}
	}
	if peak < len(body) {
		t.Fatalf("rex peak should survive death; expected >=%d, got %d", len(body), peak)
	}
}

func TestLeaderboardOrdersByPeakThenName(t *testing.T) {
	g := NewGame()
	g.peakByName["alice"] = 7
	g.peakByName["bob"] = 9
	g.peakByName["carol"] = 9
	g.peakByName["dave"] = 3

	lb := g.Leaderboard()
	if len(lb) != 4 {
		t.Fatalf("want 4 entries, got %d", len(lb))
	}
	want := []string{"bob", "carol", "alice", "dave"}
	for i, n := range want {
		if lb[i].Name != n {
			t.Fatalf("position %d: want %s, got %s", i, n, lb[i].Name)
		}
	}
}

func TestLeaderboardIncludesNewlyJoinedPlayer(t *testing.T) {
	g := NewGame()
	g.Join("freshmeat")
	lb := g.Leaderboard()
	found := false
	for _, e := range lb {
		if e.Name == "freshmeat" {
			found = true
			if e.Peak != 1 {
				t.Fatalf("new join peak should be 1, got %d", e.Peak)
			}
		}
	}
	if !found {
		t.Fatalf("new player missing from leaderboard")
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
