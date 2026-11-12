package app

import (
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// stairsAsked checks the maze asks about the stairs (going up or down, as
// the square says, unless going is ""), with both answers open and want
// suggested. Leaving Camp on stairs between floors names them by whether
// they lead to the castle (LeaveCamp), not by where they lead, so after
// Camp there the direction is not checked.
func stairsAsked(t *testing.T, c *Controller, when, going, want string) {
	t.Helper()
	g := c.Game
	v := c.DungeonView()
	if v.Prompt != "stairs" || (going != "" && g.MazeMessage != "STAIRS GOING "+going+".") || len(v.Actions) != 2 || v.Actions[0] != ActYes || v.Actions[1] != ActNo {
		t.Fatalf("%s: L%d (%d,%d) prompt %q %q actions %v", when, g.MazeLevel+1, g.PlayerX, g.PlayerY, v.Prompt, g.MazeMessage, v.Actions)
	}
	if v.Default != want {
		t.Fatalf("%s: suggested %q, want %q", when, v.Default, want)
	}
}

// standingAt checks the party stands in the maze on a square, asked nothing.
func standingAt(t *testing.T, c *Controller, when string, level, x, y int) {
	t.Helper()
	g := c.Game
	v := c.DungeonView()
	if g.Phase != engine.PhaseMaze || g.MazeLevel != level || g.PlayerX != x || g.PlayerY != y || v.Prompt != "" || v.Default != "" {
		t.Fatalf("%s: %v L%d (%d,%d) prompt %q suggested %q, want L%d (%d,%d)", when, g.Phase, g.MazeLevel+1, g.PlayerX, g.PlayerY, v.Prompt, v.Default, level+1, x, y)
	}
}

// campAndLeave camps where the party stands and leaves Camp again, which
// asks about the stairs at once on a staircase.
func campAndLeave(t *testing.T, c *Controller) {
	t.Helper()
	must(t, c.Camp())
	must(t, c.LeaveCamp())
}

// noEncounter fails if a step met an encounter: the walks here are seeded
// so that none does, and an encounter would end them.
func noEncounter(t *testing.T, c *Controller, when string) {
	t.Helper()
	if c.Game.Phase != engine.PhaseMaze {
		t.Fatalf("%s: %v (choose another seed)", when, c.Game.Phase)
	}
}

// offAndBack walks off the square the party stands on, through the first
// opening from it, turns round and walks back onto it.
func offAndBack(t *testing.T, c *Controller) {
	t.Helper()
	g := c.Game
	x, y := g.PlayerX, g.PlayerY
	for turn := 0; turn < 4; turn++ {
		must(t, c.Forward())
		noEncounter(t, c, "stepping off")
		if g.PlayerX != x || g.PlayerY != y {
			if c.mazePrompt() != "" {
				t.Fatalf("stepping off to (%d,%d) asked %q", g.PlayerX, g.PlayerY, c.mazePrompt())
			}
			must(t, c.TurnRight())
			must(t, c.TurnRight())
			must(t, c.Forward())
			noEncounter(t, c, "stepping back")
			if g.PlayerX != x || g.PlayerY != y {
				t.Fatalf("stepped back to (%d,%d), not (%d,%d)", g.PlayerX, g.PlayerY, x, y)
			}
			return
		}
		must(t, c.TurnRight())
	}
	t.Fatalf("no way off (%d,%d)", x, y)
}

// Entering from the castle, the entrance stairs suggest no: the party has
// just come down them. Turning in place asks nothing and Camp keeps the
// suggestion; either answer can be chosen.
func TestStairsSuggestNoOnEntering(t *testing.T) {
	seed(1)
	c := explorers(t)
	must(t, c.EnterMaze())
	if v := c.DungeonView(); v.Default != "" {
		t.Fatalf("Camp suggests %q", v.Default)
	}
	must(t, c.LeaveCamp())
	stairsAsked(t, c, "entering", "UP", ActNo)

	must(t, c.Answer(false))
	standingAt(t, c, "declined", 0, 0, 0)
	for i := 0; i < 4; i++ {
		must(t, c.TurnLeft())
		standingAt(t, c, "turned left", 0, 0, 0)
	}
	must(t, c.TurnRight())
	standingAt(t, c, "turned right", 0, 0, 0)
	campAndLeave(t, c)
	stairsAsked(t, c, "after Camp", "UP", ActNo)
	must(t, c.Answer(false))
	campAndLeave(t, c)
	stairsAsked(t, c, "after Camp again", "UP", ActNo)

	// The suggestion is not the only answer: yes goes back to the castle.
	must(t, c.Answer(true))
	if g := c.Game; g.Phase != engine.PhaseTown || g.Town.Location != engine.Castle {
		t.Fatalf("yes: %v %v", g.Phase, g.Town.Location)
	}
}

// Coming back to the entrance from inside the maze, the stairs suggest yes;
// turning and Camp keep it, no stays open, and the next entry suggests no
// again.
func TestStairsSuggestYesOnReturning(t *testing.T) {
	seed(1)
	c := inMaze(t)
	offAndBack(t, c)
	stairsAsked(t, c, "returning", "UP", ActYes)

	must(t, c.Answer(false))
	standingAt(t, c, "declined", 0, 0, 0)
	must(t, c.TurnLeft())
	must(t, c.TurnRight())
	campAndLeave(t, c)
	stairsAsked(t, c, "after Camp", "UP", ActYes)

	must(t, c.Answer(true))
	if g := c.Game; g.Phase != engine.PhaseTown || g.Town.Location != engine.Castle {
		t.Fatalf("yes: %v %v", g.Phase, g.Town.Location)
	}
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	stairsAsked(t, c, "entering again", "UP", ActNo)
}

// The same rule on the stairs between floors 1 and 2, both ways: walked
// onto, the stairs suggest taking them; arrived by them, the stairs back
// suggest staying. Walking off and back again suggests taking them.
func TestStairsBetweenFloorsFollowTheDirectionOfTravel(t *testing.T) {
	seed(1)
	c := inMaze(t)

	// Down: floor 1's stairs at (0,10) lead to floor 2's at (12,7).
	stepOnto(t, c, 0, 0, 10) // the stranded tests' walk
	noEncounter(t, c, "onto the stairs down")
	stairsAsked(t, c, "walked onto the stairs down", "DOWN", ActYes)
	must(t, c.Answer(true))
	standingAt(t, c, "down on floor 2", 1, 12, 7) // the stairs taken ask nothing on arrival
	campAndLeave(t, c)
	stairsAsked(t, c, "arrived by the stairs up", "", ActNo)
	must(t, c.Answer(false))
	standingAt(t, c, "staying on floor 2", 1, 12, 7)

	// Up: walked onto, floor 2's stairs suggest going up.
	offAndBack(t, c)
	stairsAsked(t, c, "walked onto the stairs up", "UP", ActYes)
	if err := c.TurnLeft(); err == nil {
		t.Fatal("turned with the question open")
	}
	stairsAsked(t, c, "turning refused", "UP", ActYes)
	must(t, c.Answer(true))
	standingAt(t, c, "up on floor 1", 0, 0, 10)
	campAndLeave(t, c)
	stairsAsked(t, c, "arrived by the stairs down", "DOWN", ActNo)

	// Against the suggestion: yes takes the stairs down again.
	must(t, c.Answer(true))
	standingAt(t, c, "down again", 1, 12, 7)
}

// Moved onto a staircase in any other way (here as a teleporter or MALOR
// would: the party set down on it), the stairs suggest yes, as before.
func TestStairsSuggestYesWhenArrivalIsUnknown(t *testing.T) {
	seed(1)
	c := inMaze(t)
	stepOnto(t, c, 0, 0, 10) // the stranded tests' walk
	noEncounter(t, c, "onto the stairs down")
	must(t, c.Answer(true))
	standingAt(t, c, "down on floor 2", 1, 12, 7)

	c.Game.PlayerX, c.Game.PlayerY = 16, 15 // floor 2's stairs down
	campAndLeave(t, c)
	stairsAsked(t, c, "set down on other stairs", "DOWN", ActYes)

	// A new controller on the same game knows nothing of how the party came
	// (a restart: the arrival is not saved).
	must(t, c.Answer(false))
	c.Game.PlayerX, c.Game.PlayerY = 12, 7
	r := New(c.Game)
	campAndLeave(t, r)
	stairsAsked(t, r, "restarted", "", ActYes)
}
