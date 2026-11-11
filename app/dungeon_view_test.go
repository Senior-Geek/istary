package app

import (
	"encoding/json"
	"math/rand"
	"slices"
	"sort"
	"testing"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// jsonKeys collects every object key in a JSON value.
func jsonKeys(v any, into map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			into[k] = true
			jsonKeys(e, into)
		}
	case []any:
		for _, e := range x {
			jsonKeys(e, into)
		}
	}
}

// actionOps maps each view action to its operation.
func actionOps(c *Controller) map[string]func() error {
	return map[string]func() error{
		ActForward: c.Forward, ActLeft: c.TurnLeft, ActRight: c.TurnRight, ActKick: c.Kick,
		ActCamp: c.Camp, ActInspect: c.Inspect, ActContinue: c.ContinueMessage,
		ActYes: func() error { return c.Answer(true) }, ActNo: func() error { return c.Answer(false) },
		ActButton: func() error { return c.PressButton(1) }, ActPickUp: func() error { return c.PickUp(1) },
		ActLeave: c.LeaveInspect, ActLeaveCamp: c.LeaveCamp,
	}
}

// Walking every floor: the view's JSON only ever has the fields of the
// schema (no position, facing, floor, map or encounter state); building it
// changes nothing, not even the random numbers; and its actions are exactly
// the operations that succeed.
func TestDungeonViewAlongRandomWalks(t *testing.T) {
	allowed := map[string]bool{}
	for _, k := range []string{"phase", "party", "corridor", "dark", "viewport", "lines", "more", "prompt", "buttons", "found", "spells", "actions",
		"slot", "name", "class", "ac", "hp", "maxHp", "status", "poisoned",
		"inspect", "room", "n", "level", "canPick", "refusal",
		"left", "right", "front", "leftAhead", "rightAhead"} {
		allowed[k] = true
	}
	seen := map[string]bool{}
	prompts := map[string]bool{}
	for s := int64(1); s <= 40; s++ {
		seed(s)
		c := inMaze(t)
		pick := rand.New(rand.NewSource(s))
		if s%2 == 0 {
			c.Game.MazeLevel = pick.Intn(len(c.Game.Scenario.Mazes.Levels))
			at(c, pick.Intn(20), pick.Intn(20), engine.Direction(pick.Intn(4)))
			c.Game.InitFightMap()
		}
		for step := 0; step < 250; step++ {
			g := c.Game
			if g.Phase == engine.PhaseCombat {
				g.Combat, g.Phase = nil, engine.PhaseMaze
			}
			if g.Phase == engine.PhaseTown {
				break
			}
			before := mazeSnap(g)
			state := rand.Int63()
			rand.Seed(state)
			v := c.DungeonView()
			if after := rand.Int63(); after != rand.New(rand.NewSource(state)).Int63() {
				t.Fatal("building the view drew a random number")
			}
			rand.Seed(state)
			if mazeSnap(g) != before {
				t.Fatal("building the view changed the game")
			}
			raw, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			var tree any
			json.Unmarshal(raw, &tree)
			jsonKeys(tree, seen)
			prompts[v.Prompt] = true

			// Unlisted operations are refused and change nothing.
			ops := actionOps(c)
			for name, op := range ops {
				if slices.Contains(v.Actions, name) {
					continue
				}
				if err := op(); err == nil {
					t.Fatalf("seed %d step %d: %s succeeded but is not in %v (prompt %q)", s, step, name, v.Actions, v.Prompt)
				}
				if mazeSnap(g) != before {
					t.Fatalf("refused %s changed the game", name)
				}
			}
			// A listed operation succeeds; walking forward most of the time.
			if len(v.Actions) == 0 {
				t.Fatalf("seed %d step %d: no actions (phase %s prompt %q)", s, step, v.Phase, v.Prompt)
			}
			act := v.Actions[pick.Intn(len(v.Actions))]
			if v.Phase == "maze" && v.Prompt == "" && pick.Intn(3) > 0 {
				act = ActForward
			}
			if act == ActCamp && pick.Intn(2) == 0 {
				act = ActRight
			}
			if err := ops[act](); err != nil {
				t.Fatalf("seed %d step %d: listed %s failed: %v", s, step, act, err)
			}
		}
	}
	var extra []string
	for k := range seen {
		if !allowed[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Fatalf("the view has fields outside the schema: %v", extra)
	}
	for _, p := range []string{"", "messages", "stairs", "inspect"} {
		if !prompts[p] {
			t.Errorf("no walk reached prompt %q; the test is not exercising it", p)
		}
	}
}

func TestDungeonViewInTheDarkAndWithLight(t *testing.T) {
	c := inMaze(t)
	g := c.Game
	lv := &g.Scenario.Mazes.Levels[0]

	// Standing on a dark square, nothing is seen.
	found := false
	for y := 0; y < 20 && !found; y++ {
		for x := 0; x < 20 && !found; x++ {
			if lv.Cells[y][x].Type == data.SqDark {
				at(c, x, y, engine.North)
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no dark square on floor 1")
	}
	v := c.DungeonView()
	if !v.Dark || len(v.Corridor) != 0 {
		t.Fatalf("on a dark square: dark %v corridor %v", v.Dark, v.Corridor)
	}

	// A hidden door looks like a wall, unless the party has light.
	x, y, dir := findSquare(t, c, data.WallHidden)
	at(c, x, y, dir)
	if v := c.DungeonView(); v.Corridor[0].Front != "wall" {
		t.Fatalf("hidden door without light: %q", v.Corridor[0].Front)
	}
	g.LightLevel = 10
	if v := c.DungeonView(); v.Corridor[0].Front != "door" || !slices.Contains(v.Spells, "LIGHT") {
		t.Fatalf("hidden door with light: %q %v", v.Corridor[0].Front, v.Spells)
	}

	// Without light the view reaches two squares, with light five.
	at(c, 0, 0, engine.North)
	for _, light := range []int{0, 10} {
		g.LightLevel = light
		if n := len(c.DungeonView().Corridor); n > map[int]int{0: 2, 10: 5}[light] {
			t.Fatalf("light %d: %d squares", light, n)
		}
	}
}

func TestDungeonViewAtCampAndTheStairs(t *testing.T) {
	c := explorers(t)
	if v := c.DungeonView(); v.Phase != "town" || v.Actions != nil {
		t.Fatalf("in town: %+v", v)
	}
	must(t, c.EnterMaze())
	v := c.DungeonView()
	if v.Phase != "camp" || !slices.Equal(v.Actions, []string{ActLeaveCamp}) || len(v.Party) != 6 || v.Corridor != nil {
		t.Fatalf("at camp: %+v", v)
	}
	must(t, c.LeaveCamp())
	v = c.DungeonView()
	if v.Prompt != "stairs" || !slices.Equal(v.Actions, []string{ActYes, ActNo}) ||
		!slices.Equal(v.Lines, []string{"STAIRS GOING UP.", stairsPrompt}) {
		t.Fatalf("at the stairs: %+v", v)
	}
	if v.Party[0].Class != "N-FIG" || v.Party[0].HP != 300 || v.Party[0].Status != "OK" {
		t.Fatalf("party: %+v", v.Party[0])
	}
}
