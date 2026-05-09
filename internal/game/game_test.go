package game

import (
	"testing"
)

func TestSnakeMovesForwardOneTilePerTick(t *testing.T) {
	g := NewGame()
	s := g.Join("a")
	g.snakes[s.ID].Body = []Position{{10, 10}}
	g.snakes[s.ID].Dir = DirRight
	g.snakes[s.ID].NextDir = DirRight
	g.pellet = Position{0, 0}

	g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 1 {
		t.Fatalf("expected length 1, got %d", len(got.Body))
	}
	if got.Body[0] != (Position{11, 10}) {
		t.Fatalf("expected head at (11,10), got %v", got.Body[0])
	}
}

func TestSnakeWrapsAroundEdges(t *testing.T) {
	g := NewGame()
	s := g.Join("a")
	g.snakes[s.ID].Body = []Position{{29, 10}}
	g.snakes[s.ID].Dir = DirRight
	g.snakes[s.ID].NextDir = DirRight
	g.pellet = Position{0, 29}

	g.Step()

	got := g.snakes[s.ID]
	if got.Body[0] != (Position{0, 10}) {
		t.Fatalf("expected wrap to (0,10), got %v", got.Body[0])
	}
}

func TestPelletGrowsSnakeAndRespawns(t *testing.T) {
	g := NewGame()
	s := g.Join("a")
	g.snakes[s.ID].Body = []Position{{10, 10}}
	g.snakes[s.ID].Dir = DirRight
	g.snakes[s.ID].NextDir = DirRight
	g.pellet = Position{11, 10}

	ev := g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 2 {
		t.Fatalf("expected length 2 after eating, got %d", len(got.Body))
	}
	if g.pellet == (Position{11, 10}) {
		t.Fatalf("pellet should have respawned away from eaten tile")
	}
	if ev.Pellet == nil {
		t.Fatalf("event should report pellet respawn")
	}
}

func TestSelfCollisionRespawns(t *testing.T) {
	g := NewGame()
	s := g.Join("a")
	g.snakes[s.ID].Body = []Position{
		{5, 5}, {5, 4}, {5, 3}, {4, 3}, {4, 4}, {4, 5}, {4, 6},
	}
	g.snakes[s.ID].Dir = DirDown
	g.snakes[s.ID].NextDir = DirLeft
	g.snakes[s.ID].PeakLength = 7
	g.pellet = Position{0, 0}

	ev := g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 1 {
		t.Fatalf("expected respawn (length 1), got %d", len(got.Body))
	}
	foundDead := false
	for _, m := range ev.Moves {
		if m.ID == s.ID && m.Dead {
			foundDead = true
			break
		}
	}
	if !foundDead {
		t.Fatalf("event should report death+respawn for the snake")
	}
}

func TestFollowingOwnTailDoesNotKill(t *testing.T) {
	g := NewGame()
	s := g.Join("a")
	g.snakes[s.ID].Body = []Position{{5, 5}, {6, 5}, {6, 6}, {5, 6}}
	g.snakes[s.ID].Dir = DirLeft
	g.snakes[s.ID].NextDir = DirDown
	g.snakes[s.ID].PeakLength = 4
	g.pellet = Position{0, 0}

	g.Step()

	got := g.snakes[s.ID]
	if len(got.Body) != 4 {
		t.Fatalf("snake should survive following its own tail; got length %d", len(got.Body))
	}
}

func TestHeadOnHeadKillsBoth(t *testing.T) {
	g := NewGame()
	a := g.Join("a")
	g.snakes[a.ID].Body = []Position{{5, 5}, {4, 5}}
	g.snakes[a.ID].Dir = DirRight
	g.snakes[a.ID].NextDir = DirRight
	g.snakes[a.ID].PeakLength = 2

	b := g.Join("b")
	g.snakes[b.ID].Body = []Position{{7, 5}, {8, 5}}
	g.snakes[b.ID].Dir = DirLeft
	g.snakes[b.ID].NextDir = DirLeft
	g.snakes[b.ID].PeakLength = 2

	g.pellet = Position{0, 0}

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
	a := g.Join("a")
	g.snakes[a.ID].Body = []Position{{5, 5}, {4, 5}, {3, 5}}
	g.snakes[a.ID].Dir = DirRight
	g.snakes[a.ID].NextDir = DirRight
	g.snakes[a.ID].PeakLength = 3

	b := g.Join("b")
	g.snakes[b.ID].Body = []Position{{4, 4}, {4, 3}}
	g.snakes[b.ID].Dir = DirDown
	g.snakes[b.ID].NextDir = DirDown
	g.snakes[b.ID].PeakLength = 2

	g.pellet = Position{0, 0}

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
	s := g.Join("a")
	g.snakes[s.ID].Body = []Position{{5, 5}, {4, 5}}
	g.snakes[s.ID].Dir = DirRight
	g.snakes[s.ID].NextDir = DirRight
	g.snakes[s.ID].PeakLength = 2

	g.SetDirection(s.ID, DirLeft)
	if g.snakes[s.ID].NextDir == DirLeft {
		t.Fatalf("180-degree reverse should be rejected by SetDirection")
	}

	g.pellet = Position{0, 0}
	g.Step()

	got := g.snakes[s.ID]
	if got.Body[0] != (Position{6, 5}) {
		t.Fatalf("snake should have continued right to (6,5), got %v", got.Body[0])
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
