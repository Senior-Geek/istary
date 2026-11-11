package app

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// explorers returns a controller in town with a party of six sturdy
// fighters, ready to enter the maze.
func explorers(t *testing.T) *Controller {
	t.Helper()
	c := newController(t, 1)
	for i := 0; i < 6; i++ {
		ch := member(fmt.Sprintf("WALKER %d", i+1), engine.Neutral)
		ch.HP, ch.MaxHP, ch.Level = 300, 300, 3
		c.Game.Town.Roster.Add(ch)
		c.Game.Town.Party.Members = append(c.Game.Town.Party.Members, ch)
	}
	return c
}

// inMaze enters the maze, leaves Camp and declines the entrance stairs.
func inMaze(t *testing.T) *Controller {
	t.Helper()
	c := explorers(t)
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(false))
	return c
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// mazeSnap describes everything a maze key can change.
func mazeSnap(g *engine.GameState) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%v L%d (%d,%d) %v light%d prot%d %q %q %q %q %v/%d/%v search%v btn%v/%d insp%v %d map%v delay%v town%q %v\n",
		g.Phase, g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing, g.LightLevel, g.ProtectLevel,
		g.MazeMessage, g.MazeMessage2, g.ViewportMsg, g.ViewportMsg2, g.MazeMessages, g.MazeMsgScroll, g.MazeMsgWait,
		g.MazeSearchYN, g.WaitButton, g.ButtonCount, g.MazeInspecting, len(g.MazeInspectFound),
		g.ShowMap, g.MazeDelayInput, g.Town.Message, g.Town.Location)
	for _, m := range g.Town.Party.Members {
		fmt.Fprintf(&b, " %s %d %v %d %d", m.Name, m.HP, m.Status, m.PoisonAmt, m.ItemCount)
	}
	if g.Combat != nil {
		fmt.Fprintf(&b, " combat%d", g.Combat.EncounterType)
	}
	return b.String()
}

// at puts the party on a square of the current floor, facing dir.
func at(c *Controller, x, y int, dir engine.Direction) {
	c.Game.PlayerX, c.Game.PlayerY, c.Game.Facing = x, y, dir
}

// findSquare finds a plain square on floor 1 whose wall in some direction
// is the given type, and returns it with that direction.
func findSquare(t *testing.T, c *Controller, want data.WallType) (int, int, engine.Direction) {
	t.Helper()
	lv := c.Game.Scenario.Mazes.Levels[0]
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			cell := lv.Cells[y][x]
			if cell.Type != data.SqNormal || cell.Encounter {
				continue
			}
			for dir, w := range map[engine.Direction]data.WallType{engine.North: cell.N, engine.East: cell.E, engine.South: cell.S, engine.West: cell.W} {
				if w == want {
					return x, y, dir
				}
			}
		}
	}
	t.Fatalf("no %s on floor 1", want)
	return 0, 0, 0
}

func TestEnterMazeCampAndStairsHome(t *testing.T) {
	c := newController(t, 1)
	if err := c.EnterMaze(); !errors.Is(err, ErrPartyEmpty) || c.Game.Phase != engine.PhaseTown {
		t.Fatalf("entering with no party: %v, phase %v", err, c.Game.Phase)
	}

	c = explorers(t)
	c.Game.PlayerX, c.Game.Facing = 7, engine.West
	must(t, c.EnterMaze())
	g := c.Game
	if g.Phase != engine.PhaseCamp || g.MazeLevel != 0 || g.PlayerX != 0 || g.PlayerY != 0 || g.Facing != engine.North {
		t.Fatalf("enter: phase %v L%d (%d,%d) %v", g.Phase, g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing)
	}
	if err := c.Forward(); !errors.Is(err, ErrWrongState) {
		t.Fatalf("forward in camp: %v", err)
	}

	// Leaving Camp at the entrance asks about the stairs at once, and the
	// maze takes nothing else until it is answered.
	must(t, c.LeaveCamp())
	if g.Phase != engine.PhaseMaze || g.MazeMessage != "STAIRS GOING UP." || g.MazeMessage2 != stairsPrompt {
		t.Fatalf("leave camp: %v %q %q", g.Phase, g.MazeMessage, g.MazeMessage2)
	}
	before := mazeSnap(g)
	for name, op := range map[string]func() error{"forward": c.Forward, "left": c.TurnLeft, "kick": c.Kick, "camp": c.Camp, "continue": c.ContinueMessage} {
		if err := op(); err == nil {
			t.Errorf("%s at the stairs prompt succeeded", name)
		}
	}
	if after := mazeSnap(g); after != before {
		t.Fatalf("refused operations changed the game:\n%s\n%s", before, after)
	}

	// Y climbs the stairs back to the castle.
	for _, m := range g.Town.Party.Members {
		m.InMaze = true
	}
	must(t, c.Answer(true))
	if g.Phase != engine.PhaseTown || g.Town.Location != engine.Castle {
		t.Fatalf("stairs up: %v %v", g.Phase, g.Town.Location)
	}
	for _, m := range g.Town.Party.Members {
		if m.InMaze {
			t.Fatalf("%s still marked in the maze", m.Name)
		}
	}
}

func TestCampFromTheMaze(t *testing.T) {
	c := inMaze(t)
	must(t, c.Camp())
	if c.Game.Phase != engine.PhaseCamp {
		t.Fatalf("camp: %v", c.Game.Phase)
	}
	// Leaving is refused while one of Camp's screens is open; there L is
	// that screen's key (reorder ignores it) and Camp stays (M6b-3A).
	c.Game.Town.InputMode = engine.InputReorder
	if err := c.LeaveCamp(); !errors.Is(err, ErrWrongState) {
		t.Fatalf("leave camp while reordering: %v", err)
	}
	if !c.HandleKey(Rune('L')) || c.Game.Phase != engine.PhaseCamp || c.Game.Town.InputMode != engine.InputReorder {
		t.Fatal("L while reordering left Camp or the reorder")
	}
	c.Game.Town.InputMode = engine.InputNone
	if !c.HandleKey(Rune('l')) || c.Game.Phase != engine.PhaseMaze {
		t.Fatalf("l at the camp menu: %v", c.Game.Phase)
	}
	if c.HandleKey(Rune('R')) != true || c.Game.Phase != engine.PhaseMaze {
		t.Fatal("maze keys are the controller's")
	}
}

func TestTurning(t *testing.T) {
	c := inMaze(t)
	g := c.Game
	x, y := g.PlayerX, g.PlayerY
	want := []engine.Direction{engine.East, engine.South, engine.West, engine.North}
	for _, d := range want {
		must(t, c.TurnRight())
		if g.Facing != d || g.PlayerX != x || g.PlayerY != y {
			t.Fatalf("turn right: %v at (%d,%d)", g.Facing, g.PlayerX, g.PlayerY)
		}
	}
	for _, d := range []engine.Direction{engine.West, engine.South, engine.East, engine.North} {
		must(t, c.TurnLeft())
		if g.Facing != d {
			t.Fatalf("turn left: %v", g.Facing)
		}
	}
	// A turn redraws the maze, which burns light (Pascal DRAWMAZE).
	g.LightLevel = 5
	must(t, c.TurnLeft())
	if g.LightLevel != 4 {
		t.Fatalf("light after a turn: %d", g.LightLevel)
	}
}

func TestWallsBlockAndDoorsNeedAKick(t *testing.T) {
	c := inMaze(t)
	g := c.Game
	beeps := 0
	c.Beep = func() { beeps++ }

	x, y, dir := findSquare(t, c, data.WallWall)
	at(c, x, y, dir)
	must(t, c.Forward())
	if g.PlayerX != x || g.PlayerY != y || g.ViewportMsg != "OUCH!" || beeps != 1 {
		t.Fatalf("forward into a wall: (%d,%d) %q beeps %d", g.PlayerX, g.PlayerY, g.ViewportMsg, beeps)
	}
	must(t, c.Kick())
	if g.PlayerX != x || g.PlayerY != y || beeps != 2 {
		t.Fatalf("kick at a wall: (%d,%d) beeps %d", g.PlayerX, g.PlayerY, beeps)
	}

	x, y, dir = findSquare(t, c, data.WallDoor)
	at(c, x, y, dir)
	must(t, c.TurnLeft()) // clears OUCH!
	must(t, c.TurnRight())
	must(t, c.Forward())
	if g.PlayerX != x || g.PlayerY != y || g.ViewportMsg != "OUCH!" {
		t.Fatalf("forward into a door: (%d,%d) %q", g.PlayerX, g.PlayerY, g.ViewportMsg)
	}
	seed(1)
	g.Phase = engine.PhaseMaze
	must(t, c.Kick())
	nx, ny := (x+dir.DX()+20)%20, (y+dir.DY()+20)%20
	if g.PlayerX != nx || g.PlayerY != ny {
		t.Fatalf("kick through a door: (%d,%d), want (%d,%d)", g.PlayerX, g.PlayerY, nx, ny)
	}
}

// The operations are the terminal's keys: driving one game with operations
// and another with the same keys through HandleKey, from the same random
// sequence, gives the same game after every step, encounters included.
func TestOperationsAreTheMazeKeys(t *testing.T) {
	type pair struct {
		key rune
		op  func(c *Controller) error
	}
	commands := []pair{
		{'F', (*Controller).Forward}, {'F', (*Controller).Forward}, {'F', (*Controller).Forward},
		{'L', (*Controller).TurnLeft}, {'R', (*Controller).TurnRight}, {'K', (*Controller).Kick},
		{'C', (*Controller).Camp}, {'I', (*Controller).Inspect},
	}
	run := func(s int64, useOps bool) []string {
		seed(s)
		c := inMaze(t)
		pick := rand.New(rand.NewSource(s))
		if s%2 == 0 {
			c.Game.MazeLevel = pick.Intn(len(c.Game.Scenario.Mazes.Levels))
			at(c, pick.Intn(20), pick.Intn(20), engine.Direction(pick.Intn(4)))
			c.Game.InitFightMap()
		}
		var trace []string
		for step := 0; step < 300; step++ {
			g := c.Game
			var k Key
			var op func() error
			if g.Phase == engine.PhaseCombat {
				// Combat is not the maze's; the walk goes on without it.
				trace = append(trace, "encounter")
				g.Combat, g.Phase = nil, engine.PhaseMaze
				continue
			}
			switch c.mazePrompt() {
			case "":
				switch g.Phase {
				case engine.PhaseCamp:
					k, op = Rune('L'), c.LeaveCamp
				case engine.PhaseTown:
					return append(trace, "town")
				default:
					p := commands[pick.Intn(len(commands))]
					k, op = Rune(p.key), func() error { return p.op(c) }
				}
			case "messages":
				k, op = Key{Kind: KeyEnter}, c.ContinueMessage
			case "stairs", "search":
				yes := pick.Intn(2) == 0
				k, op = Rune(map[bool]rune{true: 'Y', false: 'N'}[yes]), func() error { return c.Answer(yes) }
			case "buttons":
				n := 1 + pick.Intn(g.ButtonCount)
				k, op = Rune('A'+rune(n-1)), func() error { return c.PressButton(n) }
			case "inspect":
				k, op = Rune('L'), c.LeaveInspect
			default:
				t.Fatalf("unexpected prompt %q", c.mazePrompt())
			}
			if useOps {
				if err := op(); err != nil {
					t.Fatalf("seed %d step %d: %v (%s)", s, step, err, mazeSnap(g))
				}
			} else if !c.HandleKey(k) {
				t.Fatalf("seed %d step %d: key %v not handled", s, step, k)
			}
			trace = append(trace, mazeSnap(g))
		}
		return append(trace, fmt.Sprint("rand ", rand.Int63()))
	}
	encounters := 0
	for s := int64(1); s <= 60; s++ {
		a, b := run(s, true), run(s, false)
		if strings.Join(a, "\n") != strings.Join(b, "\n") {
			for i := range a {
				if i >= len(b) || a[i] != b[i] {
					t.Fatalf("seed %d diverges at step %d:\nops:  %s\nkeys: %s", s, i, a[i], b[min(i, len(b)-1)])
				}
			}
			t.Fatalf("seed %d: traces differ in length", s)
		}
		encounters += strings.Count(strings.Join(a, "\n"), "encounter")
	}
	if encounters == 0 {
		t.Fatal("no walk met an encounter; the test is not exercising the encounter check")
	}
}

// A step that meets an encounter ends in combat, and the maze then takes
// no commands: nothing is suppressed, and nothing moves on by itself.
func TestAStepCanMeetAnEncounter(t *testing.T) {
	for s := int64(1); s < 500; s++ {
		seed(s)
		c := inMaze(t)
		at(c, 0, 0, engine.North)
		for step := 0; step < 200 && c.Game.Phase == engine.PhaseMaze; step++ {
			if c.mazePrompt() != "" {
				c.mazeKey(Rune('N'))
				c.mazeKey(Key{Kind: KeyEnter})
				continue
			}
			if c.Forward() != nil || c.Game.ViewportMsg == "OUCH!" {
				c.TurnRight()
			}
		}
		if c.Game.Phase != engine.PhaseCombat {
			continue
		}
		if c.Game.Combat == nil {
			t.Fatal("combat phase without an encounter")
		}
		snap := mazeSnap(c.Game)
		for name, op := range map[string]func() error{"forward": c.Forward, "turn": c.TurnLeft, "camp": c.Camp, "leave camp": c.LeaveCamp} {
			if err := op(); err == nil {
				t.Errorf("%s during an encounter succeeded", name)
			}
		}
		if mazeSnap(c.Game) != snap {
			t.Fatal("refused operations changed the game")
		}
		return
	}
	t.Fatal("no encounter met in 500 walks")
}

func TestPromptAnswersCheckTheirPrompt(t *testing.T) {
	c := inMaze(t)
	if err := c.Answer(true); !errors.Is(err, ErrWrongState) {
		t.Fatalf("answer with no question: %v", err)
	}
	if err := c.PressButton(1); !errors.Is(err, ErrWrongState) {
		t.Fatalf("button with no buttons: %v", err)
	}
	c.Game.WaitButton, c.Game.ButtonCount = true, 2
	if err := c.PressButton(3); !errors.Is(err, ErrInvalidChoice) {
		t.Fatalf("button C of A-B: %v", err)
	}
	if err := c.Forward(); !errors.Is(err, ErrPromptPending) {
		t.Fatalf("forward at the buttons: %v", err)
	}
	c.Game.WaitButton, c.Game.ButtonCount = false, 0
	must(t, c.Inspect())
	if err := c.PickUp(1); !errors.Is(err, ErrInvalidChoice) {
		t.Fatalf("pick up with nobody found: %v", err)
	}
	must(t, c.LeaveInspect())
	if c.Game.MazeInspecting {
		t.Fatal("still inspecting")
	}
}
