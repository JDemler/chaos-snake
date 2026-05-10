package game

import (
	"strings"
	"testing"
)

func countBots(g *Game) int {
	n := 0
	for _, s := range g.snakes {
		if s.IsBot {
			n++
		}
	}
	return n
}

func TestAddBotJoinsAsBotSnake(t *testing.T) {
	g := NewGame()
	s := g.AddBot()
	if !s.IsBot {
		t.Fatalf("AddBot should return a bot snake")
	}
	if !strings.HasPrefix(s.Name, "bot-") {
		t.Fatalf("expected bot name prefix 'bot-', got %q", s.Name)
	}
	if g.snakes[s.ID] == nil || !g.snakes[s.ID].IsBot {
		t.Fatalf("bot should be registered in the game")
	}
}

func TestRemoveBotRemovesOnlyThatBot(t *testing.T) {
	g := NewGame()
	a := g.AddBot()
	b := g.AddBot()
	if !g.RemoveBot(a.ID) {
		t.Fatalf("RemoveBot should report success for a bot id")
	}
	if _, ok := g.snakes[a.ID]; ok {
		t.Fatalf("bot %s should be removed", a.ID)
	}
	if _, ok := g.snakes[b.ID]; !ok {
		t.Fatalf("other bot %s should still be present", b.ID)
	}
}

func TestRemoveBotRefusesHumanID(t *testing.T) {
	g := NewGame()
	h := g.Join("alice")
	if g.RemoveBot(h.ID) {
		t.Fatalf("RemoveBot must refuse a human id")
	}
	if _, ok := g.snakes[h.ID]; !ok {
		t.Fatalf("human %s must remain in the game", h.ID)
	}
}

func TestRemoveAllBotsKeepsHumans(t *testing.T) {
	g := NewGame()
	g.Join("alice")
	g.AddBot()
	g.AddBot()
	g.AddBot()
	if n := g.RemoveAllBots(); n != 3 {
		t.Fatalf("expected 3 bots removed, got %d", n)
	}
	if countBots(g) != 0 {
		t.Fatalf("expected no bots after RemoveAllBots, got %d", countBots(g))
	}
	if len(g.snakes) != 1 {
		t.Fatalf("expected 1 human snake remaining, got %d", len(g.snakes))
	}
}

func TestBotMovesTowardPellet(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	setPellet(g, fid, Position{15, 10})
	b := g.AddBot()
	// Place bot to the left of the pellet, currently moving up — best move is right.
	setSnake(g, b.ID, []Tile{{fid, Position{10, 10}}}, DirUp)
	g.snakes[b.ID].IsBot = true

	g.Step()

	got := g.snakes[b.ID]
	if got.Body[0] != (Tile{fid, Position{11, 10}}) {
		t.Fatalf("bot should step right toward pellet to (11,10), got %v", got.Body[0])
	}
}

func TestBotTargetsNearestOfMultiplePellets(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	// Far pellet at (0,10); near pellet at (15,10). The bot is at (10,10)
	// moving up. The nearest pellet is up-right of the bot, but since
	// (15,10) is closer (5 vs 10), the right step minimizes distance.
	setPellets(g, fid, Position{0, 10}, Position{15, 10})
	b := g.AddBot()
	setSnake(g, b.ID, []Tile{{fid, Position{10, 10}}}, DirUp)
	g.snakes[b.ID].IsBot = true

	g.Step()

	got := g.snakes[b.ID]
	if got.Body[0] != (Tile{fid, Position{11, 10}}) {
		t.Fatalf("bot should step right toward the nearer pellet (15,10), got %v", got.Body[0])
	}
}

func TestBotEvadesAdjacentBody(t *testing.T) {
	g := NewGame()
	fid := firstFieldID(g)
	// Pellet far left so the bot would prefer to go left without evasion.
	setPellet(g, fid, Position{0, 10})

	// Human snake forms a wall blocking left and up; pellet pull to (0,10)
	// would prefer left or up. Force evasion to pick down or right.
	h := g.Join("wall")
	setSnake(g, h.ID, []Tile{
		{fid, Position{9, 10}},  // blocks left
		{fid, Position{10, 9}},  // blocks up
	}, DirRight)

	b := g.AddBot()
	setSnake(g, b.ID, []Tile{{fid, Position{10, 10}}}, DirRight)
	g.snakes[b.ID].IsBot = true

	g.Step()

	got := g.snakes[b.ID]
	head := got.Body[0]
	// Must NOT have stepped into a blocked tile.
	if head == (Tile{fid, Position{9, 10}}) || head == (Tile{fid, Position{10, 9}}) {
		t.Fatalf("bot stepped into a body cell at %v", head)
	}
	// Of the safe options (right -> (11,10) dist 11, down -> (10,11) dist 11),
	// either is acceptable — both have equal Manhattan distance to (0,10).
	if head != (Tile{fid, Position{11, 10}}) && head != (Tile{fid, Position{10, 11}}) {
		t.Fatalf("bot should pick a safe tile, got %v", head)
	}
}

func TestTargetCountSpawnsBots(t *testing.T) {
	g := NewGame()
	g.SetTargetCount(3)
	g.Step()
	if len(g.snakes) != 3 {
		t.Fatalf("expected 3 snakes after target=3 step, got %d", len(g.snakes))
	}
	if countBots(g) != 3 {
		t.Fatalf("expected 3 bots, got %d", countBots(g))
	}
}

func TestTargetCountRemovesBotsWhenHumansArrive(t *testing.T) {
	g := NewGame()
	g.SetTargetCount(2)
	g.Step()
	if len(g.snakes) != 2 {
		t.Fatalf("setup: expected 2 snakes, got %d", len(g.snakes))
	}
	g.Join("alice")
	g.Join("bob")
	g.Join("carol") // total now 5, target is 2.
	g.Step()
	if len(g.snakes) != 3 {
		t.Fatalf("expected 3 snakes (3 humans, 0 bots) after target enforcement, got %d", len(g.snakes))
	}
	if countBots(g) != 0 {
		t.Fatalf("expected all bots removed, got %d bots", countBots(g))
	}
}

func TestTargetCountNeverRemovesHumans(t *testing.T) {
	g := NewGame()
	g.Join("alice")
	g.Join("bob")
	g.Join("carol")
	g.SetTargetCount(1)
	g.Step()
	if countBots(g) != 0 {
		t.Fatalf("no bots should be present, got %d", countBots(g))
	}
	if len(g.snakes) != 3 {
		t.Fatalf("humans must not be auto-removed; expected 3 humans, got %d", len(g.snakes))
	}
}

func TestClearTargetCountStopsMaintenance(t *testing.T) {
	g := NewGame()
	g.SetTargetCount(2)
	g.Step()
	g.ClearTargetCount()
	g.RemoveAllBots()
	g.Step()
	if len(g.snakes) != 0 {
		t.Fatalf("after ClearTargetCount no respawn should occur, got %d snakes", len(g.snakes))
	}
}
