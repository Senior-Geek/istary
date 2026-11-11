package app

import (
	"encoding/json"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// The coordinate readout (UX-1): Position is the canonical L:R:C and the
// facing, the maps' own, in the maze and at Camp; none off the maze.

// mustPosition is Position, which must be known.
func mustPosition(t *testing.T, c *Controller) PositionView {
	t.Helper()
	p, ok := c.Position()
	if !ok {
		t.Fatal("no position")
	}
	return p
}

// agrees checks Position against the game and both maps. Position is
// read before the square is explored, as the window reads it.
func agrees(t *testing.T, c *Controller, want string, facing string) {
	t.Helper()
	p := mustPosition(t, c)
	here := herePlace(t, c)
	if p.Place != want || p.Facing != facing || p.Place != here.String() || p.Level != here.Level || p.Row != here.Row || p.Col != here.Col {
		t.Fatalf("position %+v, want %s %s (game %s)", p, want, facing, here)
	}
	c.Game.Explore() // as the window does after every action
	lv, _ := c.LocalMap()
	m, err := c.Map(0)
	must(t, err)
	if lv.Party.Place != p.Place || lv.Party.Facing != p.Facing || m.Party == nil || m.Party.Place != p.Place || m.Party.Facing != p.Facing {
		t.Fatalf("maps disagree: minimap %+v, map %+v, position %+v", lv.Party, m.Party, p)
	}
}

// The convention: the north-west square is R0:C0, rows grow southward and
// columns eastward; the facing is apart from the square.
func TestPositionConvention(t *testing.T) {
	c := inMaze(t)
	agrees(t, c, "L1:R19:C0", "N") // the entrance, facing north
	at(c, 0, 19, engine.East)
	agrees(t, c, "L1:R0:C0", "E")
	at(c, 0, 18, engine.East) // a step south
	agrees(t, c, "L1:R1:C0", "E")
	at(c, 1, 18, engine.East) // a step east
	agrees(t, c, "L1:R1:C1", "E")
	at(c, 5, 8, engine.South)
	agrees(t, c, "L1:R11:C5", "S")
	c.Game.MazeLevel = 2
	agrees(t, c, "L3:R11:C5", "S")
}

// After real moves and turns the readout is where the party went.
func TestPositionFollowsMovesAndTurns(t *testing.T) {
	c := inMaze(t)
	agrees(t, c, "L1:R19:C0", "N")
	turns := []struct {
		turn func() error
		want string
	}{{c.TurnRight, "E"}, {c.TurnRight, "S"}, {c.TurnLeft, "E"}, {c.TurnLeft, "N"}, {c.TurnLeft, "W"}, {c.TurnRight, "N"}}
	for _, tr := range turns {
		must(t, tr.turn())
		agrees(t, c, "L1:R19:C0", tr.want)
	}
	moved := 0
	for i := 0; i < 40 && c.Game.Phase == engine.PhaseMaze; i++ {
		before := mustPosition(t, c)
		from := herePlace(t, c)
		c.Game.MazeMessage, c.Game.MazeMessage2 = "", ""
		if i%5 == 4 {
			must(t, c.TurnRight())
			agrees(t, c, before.Place, c.Game.Facing.Letter())
			continue
		}
		if c.Forward() != nil || c.Game.Phase != engine.PhaseMaze || c.mazePrompt() != "" {
			break // an encounter, a message or a stairway: the walk ends here
		}
		after := mustPosition(t, c)
		if after.Place != before.Place {
			moved++
			if want := from.Step(c.Game.Facing).String(); after.Place != want {
				t.Fatalf("a step from %s facing %s went to %s, want %s", before.Place, before.Facing, after.Place, want)
			}
		}
		agrees(t, c, after.Place, after.Facing)
	}
	if moved == 0 {
		t.Fatal("the party never moved")
	}
}

// Camp shows the very square and facing the maze does, and so does the
// maze again after Camp.
func TestPositionInCampIsTheMaze(t *testing.T) {
	c := inMaze(t)
	at(c, 3, 15, engine.West)
	maze := mustPosition(t, c)
	must(t, c.Camp())
	if c.Game.Phase != engine.PhaseCamp {
		t.Fatalf("phase %v", c.Game.Phase)
	}
	if camp := mustPosition(t, c); camp != maze {
		t.Fatalf("camp %+v, maze %+v", camp, maze)
	}
	must(t, c.LeaveCamp())
	if again := mustPosition(t, c); again != maze {
		t.Fatalf("after Camp %+v, before %+v", again, maze)
	}
}

// No coordinates are made up: none in town, none off the maze's levels.
func TestNoPositionOffTheMaze(t *testing.T) {
	c := explorers(t)
	if p, ok := c.Position(); ok {
		t.Fatalf("in town: %+v", p)
	}
	c = inMaze(t)
	c.Game.MazeLevel = len(c.Game.Scenario.Mazes.Levels) // below the last (finding #31)
	if p, ok := c.Position(); ok {
		t.Fatalf("off the levels: %+v", p)
	}
}

// Stranded counts those on the party's square, as its marker does, and
// follows each pickup; a neighbour's, the LOST and the uncertain do not
// count. Reading it changes nothing, the exploration record included.
func TestPositionStrandedHere(t *testing.T) {
	c := rescueParty(t, 2)
	here := herePlace(t, c)
	strandedAt(c, here, "HERE", 2)
	strandedAt(c, here.Step(engine.North), "NEXT", 1)
	lost := strandedAt(c, here, "LOST", 1)[0]
	lost.Status = engine.Lost
	legacy := strandedAt(c, here, "OLD", 1)[0]
	legacy.Stranded = nil
	before, _ := json.Marshal(c.Game.Exploration)
	if p := mustPosition(t, c); p.Stranded != 2 {
		t.Fatalf("stranded here %d, want 2", p.Stranded)
	}
	if after, _ := json.Marshal(c.Game.Exploration); string(after) != string(before) {
		t.Fatal("reading the position changed the exploration record")
	}
	must(t, c.Inspect())
	n := 0
	for i, f := range c.DungeonView().Inspect.Found {
		if f.Name == "HERE1" {
			n = i + 1
		}
	}
	must(t, c.PickUp(n))
	if p := mustPosition(t, c); p.Stranded != 1 {
		t.Fatalf("after one pickup %d, want 1", p.Stranded)
	}
	for i, f := range c.DungeonView().Inspect.Found {
		if f.Name == "HERE2" {
			n = i + 1
		}
	}
	must(t, c.PickUp(n))
	if p := mustPosition(t, c); p.Stranded != 0 {
		t.Fatalf("after both %d, want 0", p.Stranded)
	}
}
