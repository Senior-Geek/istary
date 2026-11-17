package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// The perspective renderer's tests (frontend/src/lib/mazeScene.test.js and
// mazeProjection.test.js) draw real views of known corridors (M-ART-1A.1).
// This builds each corridor as a small synthetic level, in memory only (the
// practice game's scenario is a copy whose one level is the synthetic one;
// no scenario data is changed), stands the party in it, and records the
// dungeon view the GUI receives. For each case it checks that the corridor
// is the engine's Sight, square for square, and that the view ends as the
// case says (how many squares are seen, and what ends them). Run with
// ISTARY_UPDATE_FIXTURES=1 to rewrite the file after a deliberate change.

// mazeSceneCase is one recorded view: the case, what Go was asked to show
// and what it must show, and the dungeon view itself.
type mazeSceneCase struct {
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	Facing    string           `json:"facing"`
	Light     bool             `json:"light"`
	QuickPlot bool             `json:"quickPlot,omitempty"`
	Depths    int              `json:"depths"` // squares seen
	Ends      string           `json:"ends"`   // "wall", "door", "dark" or "limit"
	Dungeon   *app.DungeonView `json:"dungeon"`
}

// synthMaze is a level of solid walls into which corridors are cut.
type synthMaze struct{ lv data.MazeLevel }

func newSynthMaze() *synthMaze {
	m := &synthMaze{lv: data.MazeLevel{Level: 1, Cells: make([][]data.MazeCell, engine.MazeSize)}}
	for y := range m.lv.Cells {
		m.lv.Cells[y] = make([]data.MazeCell, engine.MazeSize)
		for x := range m.lv.Cells[y] {
			m.lv.Cells[y][x] = data.MazeCell{N: data.WallWall, S: data.WallWall, E: data.WallWall, W: data.WallWall}
		}
	}
	return m
}

// side returns the wall on side dir of square (x, y).
func (m *synthMaze) side(x, y int, dir engine.Direction) *data.WallType {
	c := &m.lv.Cells[y][x]
	return [...]*data.WallType{&c.N, &c.E, &c.S, &c.W}[dir]
}

// set makes the boundary on side dir of (x, y) w, seen alike from both
// squares (across the level's edge too).
func (m *synthMaze) set(x, y int, dir engine.Direction, w data.WallType) *synthMaze {
	*m.side(x, y, dir) = w
	nx, ny := engine.Shift(x, y, dir, 0, 1)
	*m.side(nx, ny, dir.Reverse()) = w
	return m
}

// run cuts a corridor of n squares from (x, y) towards dir.
func (m *synthMaze) run(x, y int, dir engine.Direction, n int) *synthMaze {
	for i := 1; i < n; i++ {
		m.set(x, y, dir, data.WallOpen)
		x, y = engine.Shift(x, y, dir, 0, 1)
	}
	return m
}

// The cases stand the party at (10, 2) facing north unless they say
// otherwise: left is west, right is east, square d is (10, 2+d).
const sx, sy = 10, 2

// ahead is the square d ahead of the start, and beside it on the given
// side (-1 left, +1 right), facing north.
func ahead(d, side int) (int, int) { return sx + side, sy + d }

type sceneSetup struct {
	id, title  string
	maze       func() *synthMaze
	x, y       int
	facing     engine.Direction
	light      bool
	quick      bool
	depths     int
	ends       string
	beforeView func(c *app.Controller) // optional: run before the view is taken
}

func mazeSceneSetups() []sceneSetup {
	n := engine.North
	straight := func(len int) func() *synthMaze {
		return func() *synthMaze { return newSynthMaze().run(sx, sy, n, len) }
	}
	var cases []sceneSetup
	add := func(c sceneSetup) {
		if c.maze == nil {
			c.maze = straight(1)
		}
		if c.x == 0 && c.y == 0 {
			c.x, c.y = sx, sy
		}
		cases = append(cases, c)
	}

	add(sceneSetup{id: "G01", title: "Straight corridor", maze: straight(8), light: true, depths: 5, ends: "limit"})
	add(sceneSetup{id: "G02", title: "Dead end", maze: straight(3), light: true, depths: 3, ends: "wall"})
	add(sceneSetup{id: "G03", title: "Left opening", light: true, depths: 5, ends: "limit", maze: func() *synthMaze {
		m := straight(8)()
		x, y := ahead(1, 0)
		return m.set(x, y, engine.West, data.WallOpen) // the side square's own walls stay solid
	}})
	add(sceneSetup{id: "G04", title: "Right opening onto a passage that goes on", light: true, depths: 5, ends: "limit", maze: func() *synthMaze {
		m := straight(8)()
		x, y := ahead(2, 0)
		m.set(x, y, engine.East, data.WallOpen)
		rx, ry := ahead(2, 1)
		return m.run(rx, ry, n, 4) // ahead of the side square, open, and further: not seen
	}})
	add(sceneSetup{id: "G05", title: "Corner turning right", light: true, depths: 2, ends: "wall", maze: func() *synthMaze {
		m := straight(2)()
		x, y := ahead(1, 0)
		return m.set(x, y, engine.East, data.WallOpen)
	}})
	add(sceneSetup{id: "G06", title: "T-junction", light: true, depths: 2, ends: "wall", maze: func() *synthMaze {
		m := straight(2)()
		x, y := ahead(1, 0)
		lx, ly := ahead(1, -1)
		m.set(x, y, engine.West, data.WallOpen).set(x, y, engine.East, data.WallOpen)
		return m.run(lx-3, ly, engine.East, 7) // the cross passage, on both sides
	}})
	add(sceneSetup{id: "G07", title: "Four-way intersection", light: true, depths: 4, ends: "wall", maze: func() *synthMaze {
		m := straight(4)()
		lx, ly := ahead(1, -1)
		return m.run(lx-3, ly, engine.East, 7)
	}})
	add(sceneSetup{id: "G08", title: "Door ahead", light: true, depths: 3, ends: "door", maze: func() *synthMaze {
		m := straight(6)()
		x, y := ahead(2, 0)
		return m.set(x, y, n, data.WallDoor)
	}})
	add(sceneSetup{id: "G09", title: "Door on the left", light: true, depths: 4, ends: "wall", maze: func() *synthMaze {
		m := straight(4)()
		x, y := ahead(1, 0)
		return m.set(x, y, engine.West, data.WallDoor)
	}})
	add(sceneSetup{id: "G10", title: "Doors right, left and ahead", light: true, depths: 4, ends: "door", maze: func() *synthMaze {
		m := straight(6)()
		x0, y0 := ahead(0, 0)
		x1, y1 := ahead(1, 0)
		x2, y2 := ahead(2, 0)
		x3, y3 := ahead(3, 0)
		m.set(x0, y0, engine.East, data.WallDoor).set(x1, y1, engine.West, data.WallDoor)
		m.set(x2, y2, engine.West, data.WallOpen) // an opening with a door ahead of it
		lx, ly := ahead(2, -1)
		m.set(lx, ly, n, data.WallDoor)
		return m.set(x3, y3, n, data.WallDoor)
	}})
	// G11: one square seen facing each way. North a wall, east a door,
	// south and west open; the squares beyond are solid.
	g11 := func() *synthMaze {
		m := newSynthMaze()
		m.set(sx, sy, engine.East, data.WallDoor)
		m.set(sx, sy, engine.South, data.WallOpen)
		return m.set(sx, sy, engine.West, data.WallOpen)
	}
	for _, f := range []engine.Direction{engine.North, engine.East, engine.South, engine.West} {
		d, e := map[engine.Direction]int{engine.North: 1, engine.East: 1, engine.South: 2, engine.West: 2}[f], "wall"
		if f == engine.East {
			e = "door"
		}
		add(sceneSetup{id: "G11-" + f.String()[:1], title: "Facing " + f.String(), maze: g11, facing: f, depths: d, ends: e})
	}
	// G12: a corridor across the level's east edge, and the same corridor
	// in the middle of the level.
	add(sceneSetup{id: "G12-edge", title: "Corridor across the east edge", x: 17, y: 10, facing: engine.East, light: true, depths: 5, ends: "wall",
		maze: func() *synthMaze {
			m := newSynthMaze().run(17, 10, engine.East, 5)
			return m.set(1, 10, engine.North, data.WallOpen).set(19, 10, engine.South, data.WallDoor)
		}})
	add(sceneSetup{id: "G12-inside", title: "The same corridor inside the level", x: 5, y: 10, facing: engine.East, light: true, depths: 5, ends: "wall",
		maze: func() *synthMaze {
			m := newSynthMaze().run(5, 10, engine.East, 5)
			return m.set(9, 10, engine.North, data.WallOpen).set(7, 10, engine.South, data.WallDoor)
		}})

	add(sceneSetup{id: "V01", title: "No magical light", maze: straight(8), depths: 2, ends: "limit"})
	add(sceneSetup{id: "V02", title: "Magical light", maze: straight(8), light: true, depths: 5, ends: "limit"})
	add(sceneSetup{id: "V03", title: "QUICK plot with light", maze: straight(8), light: true, quick: true, depths: 3, ends: "limit"})
	// Behind a blocking wall or door the corridor goes on, with a door in
	// it: none of it is seen.
	behind := func(w data.WallType) func() *synthMaze {
		return func() *synthMaze {
			m := straight(8)()
			x, y := ahead(1, 0)
			m.set(x, y, n, w)
			x, y = ahead(3, 0)
			return m.set(x, y, engine.West, data.WallDoor)
		}
	}
	add(sceneSetup{id: "V04", title: "A wall ahead, a corridor behind it", maze: behind(data.WallWall), light: true, depths: 2, ends: "wall"})
	add(sceneSetup{id: "V05", title: "A door ahead, a corridor behind it", maze: behind(data.WallDoor), light: true, depths: 2, ends: "door"})
	add(sceneSetup{id: "V06", title: "Hidden door ahead, without light", maze: behind(data.WallHidden), depths: 2, ends: "wall"})
	add(sceneSetup{id: "V06-wall", title: "A plain wall ahead, without light (V06's twin)", maze: behind(data.WallWall), depths: 2, ends: "wall"})
	add(sceneSetup{id: "V07", title: "Hidden door ahead, with light", maze: behind(data.WallHidden), light: true, depths: 2, ends: "door"})
	add(sceneSetup{id: "V08", title: "A dark square two ahead, a door beyond it", light: true, depths: 2, ends: "dark", maze: func() *synthMaze {
		m := straight(8)()
		x, y := ahead(2, 0)
		m.lv.Cells[y][x].Type = data.SqDark
		x, y = ahead(3, 0)
		return m.set(x, y, n, data.WallDoor)
	}})
	add(sceneSetup{id: "V09", title: "Standing in a dark square", light: true, depths: 0, ends: "dark", maze: func() *synthMaze {
		m := straight(8)()
		m.lv.Cells[sy][sx].Type = data.SqDark
		return m
	}})
	add(sceneSetup{id: "V10", title: "The corridor goes on past the light, to a door", light: true, depths: 5, ends: "limit", maze: func() *synthMaze {
		m := straight(10)()
		x, y := ahead(6, 0)
		m.set(x, y, n, data.WallDoor)
		x, y = ahead(5, 0)
		return m.set(x, y, engine.East, data.WallDoor)
	}})
	// V11: the party has looked down the corridor with light, so the
	// exploration record holds five squares of it; now, without light, the
	// view is two squares, as ever.
	add(sceneSetup{id: "V11", title: "Explored with light, now seen without", maze: straight(8), depths: 2, ends: "limit",
		beforeView: func(c *app.Controller) {
			c.Game.LightLevel = 10
			c.Explore()
			c.Game.LightLevel = 0
		}})
	return cases
}

// recordMazeScenes builds every case in a practice expedition and returns
// the recording.
func recordMazeScenes(t *testing.T) []mazeSceneCase {
	t.Helper()
	rand.Seed(5)
	s, _ := partySession(t, false)
	mustOK(t, s.PracticeExpedition(), "dungeon")
	defer s.AbandonPractice()
	c := s.practice
	g := c.Game
	own := g.Scenario
	defer func() { g.Scenario = own }()
	var out []mazeSceneCase
	for _, k := range mazeSceneSetups() {
		m := k.maze()
		sc := *own // a copy: the scenario's own levels are not touched
		sc.Mazes.Levels = []data.MazeLevel{m.lv}
		g.Scenario = &sc
		g.Phase, g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = engine.PhaseMaze, 0, k.x, k.y, k.facing
		g.LightLevel, g.QuickPlot, g.ProtectLevel = 0, k.quick, 0
		if k.light {
			g.LightLevel = 10
		}
		g.MazeMessage, g.MazeMessage2, g.ViewportMsg, g.ViewportMsg2, g.MazeMessages = "", "", "", "", nil
		g.Exploration = nil
		if k.beforeView != nil {
			k.beforeView(c)
		}
		if k.id == "V11" {
			// The record knows the fifth square's walls (L1:R13:C10).
			if lm := g.Exploration.Level(1); lm == nil || lm.Walls[13][10][engine.East] != engine.SeenWall {
				t.Fatalf("V11: the exploration record does not hold the far squares")
			}
		}

		before := fmt.Sprintf("%v %d %d %d %v %d %v", g.Phase, g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing, g.LightLevel, g.QuickPlot)
		v := s.State()
		after := fmt.Sprintf("%v %d %d %d %v %d %v", g.Phase, g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing, g.LightLevel, g.QuickPlot)
		if before != after {
			t.Fatalf("%s: taking the view changed the game: %s -> %s", k.id, before, after)
		}
		d := mustOK(t, v, "dungeon").Dungeon
		if d.Phase != "maze" || d.Prompt != "" {
			t.Fatalf("%s: phase %q prompt %q", k.id, d.Phase, d.Prompt)
		}

		// The corridor is the engine's Sight, square for square.
		sight, dark := g.Sight()
		var want []app.CorridorView
		for _, q := range sight {
			want = append(want, app.CorridorView{Left: q.Left, Right: q.Right, Front: q.Front, LeftAhead: q.LeftAhead, RightAhead: q.RightAhead})
		}
		if !reflect.DeepEqual(d.Corridor, want) || d.Dark != dark {
			t.Fatalf("%s: corridor %+v dark %v, Sight %+v dark %v", k.id, d.Corridor, d.Dark, want, dark)
		}
		// And it ends as the case says.
		ends := "limit"
		switch {
		case dark:
			ends = "dark"
		case len(sight) > 0 && sight[len(sight)-1].Front != engine.SeenOpen:
			ends = sight[len(sight)-1].Front
		}
		if len(sight) != k.depths || ends != k.ends {
			t.Fatalf("%s: %d squares ending %q, want %d ending %q", k.id, len(sight), ends, k.depths, k.ends)
		}
		out = append(out, mazeSceneCase{ID: k.id, Title: k.title, Facing: k.facing.String(), Light: k.light, QuickPlot: k.quick,
			Depths: k.depths, Ends: k.ends, Dungeon: d})
	}
	return out
}

func TestMazeSceneFixture(t *testing.T) {
	cases := recordMazeScenes(t)
	byID := map[string]mazeSceneCase{}
	for _, c := range cases {
		byID[c.ID] = c
	}
	// A hidden door without light is a wall, exactly; the same corridor
	// across the level's edge is the corridor inside it.
	if !reflect.DeepEqual(byID["V06"].Dungeon.Corridor, byID["V06-wall"].Dungeon.Corridor) {
		t.Fatalf("hidden door unlit %+v, wall %+v", byID["V06"].Dungeon.Corridor, byID["V06-wall"].Dungeon.Corridor)
	}
	if !reflect.DeepEqual(byID["G12-edge"].Dungeon.Corridor, byID["G12-inside"].Dungeon.Corridor) {
		t.Fatalf("across the edge %+v, inside %+v", byID["G12-edge"].Dungeon.Corridor, byID["G12-inside"].Dungeon.Corridor)
	}

	got, _ := json.MarshalIndent(cases, "", " ")
	got = append(got, '\n')
	// Recording again gives the same bytes.
	again, _ := json.MarshalIndent(recordMazeScenes(t), "", " ")
	if !bytes.Equal(got, append(again, '\n')) {
		t.Fatal("recording the maze scenes twice gave different files")
	}
	path := filepath.Join("frontend", "src", "lib", "testdata", "maze-scenes.json")
	if os.Getenv("ISTARY_UPDATE_FIXTURES") != "" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with ISTARY_UPDATE_FIXTURES=1)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("maze-scenes.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
