package app

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// After entering, both maps put the party at L1:R19:C0 facing north, on the
// stairs up; the full map is the fixed 20x20 frame.
func TestMapsAtTheEntrance(t *testing.T) {
	c := explorers(t)
	must(t, c.EnterMaze())
	c.Game.Explore()
	lv, ok := c.LocalMap()
	if !ok || lv.Party != (MapParty{Row: LocalRadius, Col: LocalRadius, Facing: "N", Place: "L1:R19:C0"}) || lv.Level != 1 || lv.Size != 2*LocalRadius+1 {
		t.Fatalf("local %+v", lv)
	}
	if len(lv.Features) != 1 || lv.Features[0] != (MapFeature{Row: LocalRadius, Col: LocalRadius, Kind: engine.FeatureStairsUp}) {
		t.Fatalf("local features %+v", lv.Features)
	}
	// R19 is the bottom row: the level's north edge (R0) is the row below
	// the party, and its west edge (C0) the party's own column.
	if lv.NorthEdge != LocalRadius+1 || lv.WestEdge != LocalRadius {
		t.Fatalf("edges %d %d", lv.NorthEdge, lv.WestEdge)
	}
	m, err := c.Map(0)
	if err != nil || m.Level != 1 || m.Size != 20 || !reflect.DeepEqual(m.Levels, []int{1}) {
		t.Fatalf("map %+v %v", m, err)
	}
	if m.Party == nil || *m.Party != (MapParty{Row: 19, Col: 0, Facing: "N", Place: "L1:R19:C0"}) {
		t.Fatalf("party %+v", m.Party)
	}
	if !reflect.DeepEqual(m.Visited, []MapSquare{{19, 0}}) || !reflect.DeepEqual(lv.Visited, []MapSquare{{LocalRadius, LocalRadius}}) {
		t.Fatalf("visited %+v %+v", m.Visited, lv.Visited)
	}
	if _, err := c.Map(3); err != ErrNoMap {
		t.Fatalf("an unexplored level: %v", err)
	}
	c.Game.Phase = engine.PhaseTown
	if _, ok := c.LocalMap(); ok {
		t.Fatal("a local map in the castle")
	}
	if m, err := c.Map(0); err != nil || m.Party != nil || m.Level != 1 {
		t.Fatalf("map from the castle %+v %v", m, err)
	}
}

// handMap is a game whose level 1 exploration is set by hand.
func handMap(t *testing.T, set func(m *engine.LevelMap)) *Controller {
	t.Helper()
	c := explorers(t)
	must(t, c.EnterMaze())
	c.Game.Phase = engine.PhaseMaze
	m := &engine.LevelMap{}
	set(m)
	c.Game.Exploration = &engine.Exploration{Dungeon: "1", Levels: map[int]*engine.LevelMap{1: m}}
	return c
}

// A boundary seen alike from both sides, or from one only, is one wall; one
// seen differently from its two sides is each side's own wall, inside its
// square, so the map never shows a way through that is not there.
func TestMapWallsBySide(t *testing.T) {
	c := handMap(t, func(m *engine.LevelMap) {
		m.Walls[5][5][engine.East] = engine.SeenWall // alike from both sides
		m.Walls[5][6][engine.West] = engine.SeenWall
		m.Walls[7][5][engine.South] = engine.SeenDoor  // from one side only
		m.Walls[9][5][engine.East] = engine.SeenOpen   // one way: open from the west
		m.Walls[9][6][engine.West] = engine.SeenWall   // a wall from the east
		m.Walls[11][5][engine.North] = engine.SeenDoor // a door one way, a wall back
		m.Walls[10][5][engine.South] = engine.SeenWall
	})
	m, err := c.Map(1)
	if err != nil {
		t.Fatal(err)
	}
	want := []MapWall{
		{Row: 5, Col: 5, Side: "E", Kind: "wall"},
		{Row: 7, Col: 5, Side: "S", Kind: "door"},
		{Row: 9, Col: 6, Side: "W", Kind: "wall", OneWay: true},
		{Row: 10, Col: 5, Side: "S", Kind: "wall", OneWay: true},
		{Row: 11, Col: 5, Side: "N", Kind: "door", OneWay: true},
	}
	if !sameWalls(m.Walls, want) {
		t.Fatalf("walls %+v", m.Walls)
	}
}

func sameWalls(got, want []MapWall) bool {
	g := map[MapWall]int{}
	for _, w := range got {
		g[w]++
	}
	for _, w := range want {
		g[w]--
	}
	for _, n := range g {
		if n != 0 {
			return false
		}
	}
	return len(got) == len(want)
}

// The full map marks seen ways across the level's edge at the square they
// leave from, in its fixed frame; the minimap carries on across the edge
// as the maze does.
func TestMapAcrossTheEdge(t *testing.T) {
	c := handMap(t, func(m *engine.LevelMap) {
		m.Walls[0][10][engine.North] = engine.SeenOpen // L1:R0:C10 north, into R19:C10
		m.Walls[19][10][engine.South] = engine.SeenOpen
		m.Walls[19][10][engine.East] = engine.SeenWall
		m.Walls[4][19][engine.East] = engine.SeenDoor // R4:C19 east, a door across
		m.Walls[4][0][engine.West] = engine.SeenWall  // and a wall from the other side
	})
	m, _ := c.Map(1)
	wantWraps := []MapWall{
		{Row: 0, Col: 10, Side: "N", Kind: "open"},
		{Row: 19, Col: 10, Side: "S", Kind: "open"},
		{Row: 4, Col: 19, Side: "E", Kind: "door", OneWay: true},
	}
	if !sameWalls(m.Wraps, wantWraps) {
		t.Fatalf("wraps %+v", m.Wraps)
	}
	wantWalls := []MapWall{
		{Row: 19, Col: 10, Side: "E", Kind: "wall"},
		{Row: 4, Col: 19, Side: "E", Kind: "door", OneWay: true},
		{Row: 4, Col: 0, Side: "W", Kind: "wall", OneWay: true},
	}
	if !sameWalls(m.Walls, wantWalls) {
		t.Fatalf("walls %+v", m.Walls)
	}
	// The minimap from L1:R0:C10: R19 is the row above the party, and its
	// east wall is drawn there; the level's north edge is the party's row.
	lvl, x, y := engine.Place{Level: 1, Row: 0, Col: 10}.Internal()
	c.Game.MazeLevel, c.Game.PlayerX, c.Game.PlayerY = lvl, x, y
	lv, _ := c.LocalMap()
	if lv.NorthEdge != LocalRadius || lv.WestEdge != -1 {
		t.Fatalf("edges %d %d", lv.NorthEdge, lv.WestEdge)
	}
	if !sameWalls(lv.Walls, []MapWall{{Row: LocalRadius - 1, Col: LocalRadius, Side: "E", Kind: "wall"}}) {
		t.Fatalf("local walls %+v", lv.Walls)
	}
}

// walk plays seeded random steps and turns through the controller on each
// level, recording after each action as the GUI does, and calls look at
// every stop.
func walk(t *testing.T, seedN int64, steps int, look func(c *Controller)) *Controller {
	t.Helper()
	seed(seedN)
	rng := rand.New(rand.NewSource(seedN))
	c := explorers(t)
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	g := c.Game
	for level := 0; level < len(g.Scenario.Mazes.Levels); level++ {
		g.Phase, g.MazeLevel, g.PlayerX, g.PlayerY = engine.PhaseMaze, level, rng.Intn(20), rng.Intn(20)
		g.MazeMessage, g.MazeMessage2, g.MazeMessages, g.WaitButton = "", "", nil, false
		g.InitFightMap()
		for i := 0; i < steps; i++ {
			switch g.Phase {
			case engine.PhaseCombat:
				g.Combat, g.Phase = nil, engine.PhaseMaze
			case engine.PhaseTown, engine.PhaseCamp:
				g.Phase = engine.PhaseMaze
			}
			for _, m := range g.Town.Party.Members {
				m.HP, m.Status = m.MaxHP, engine.OK
			}
			g.LightLevel = map[bool]int{true: 30}[i%300 > 150]
			switch c.mazePrompt() {
			case "messages":
				_ = c.ContinueMessage()
			case "stairs", "search":
				_ = c.Answer(false)
			case "buttons":
				_ = c.mazeKeyAt(Key{Kind: KeyEnter})
			case "inspect":
				_ = c.LeaveInspect()
			case "":
				switch r := rng.Intn(10); {
				case r < 6:
					_ = c.Forward()
				case r < 7:
					_ = c.Kick()
				case r < 8:
					_ = c.TurnLeft()
				default:
					_ = c.TurnRight()
				}
			default:
				g.MazeMessage, g.MazeMessage2, g.MazeDelayInput, g.ShowMap = "", "", false, false
			}
			if g.MazeLevel < 0 || g.MazeLevel >= len(g.Scenario.Mazes.Levels) {
				g.MazeLevel = level
			}
			g.Explore()
			if look != nil {
				look(c)
			}
		}
	}
	return c
}

// mazeKeyAt sends a key in the maze, for the walk's prompts.
func (c *Controller) mazeKeyAt(k Key) error {
	c.mazeKey(k)
	return nil
}

// The minimap and the full map agree on every wall they both show, after
// seeded walks on every level, at every stop; looking at either changes
// nothing in the game.
func TestLocalAndFullMapsAgree(t *testing.T) {
	compared := 0
	walk(t, 11, 400, func(c *Controller) {
		here, ok := c.Game.Place()
		if !ok || c.Game.Phase != engine.PhaseMaze {
			return
		}
		before := mazeSnap(c.Game)
		ex := c.Game.Exploration.Clone()
		state := rand.Int63()
		seed(int64(len(before)))
		lv, _ := c.LocalMap()
		full, err := c.Map(0)
		if err != nil {
			t.Fatal(err)
		}
		after := rand.Int63()
		seed(int64(len(before)))
		if rand.Int63() != after || mazeSnap(c.Game) != before || !reflect.DeepEqual(ex, c.Game.Exploration) {
			t.Fatal("looking at the maps changed the game or drew a random number")
		}
		seed(state)
		abs := func(i, j int) engine.Place {
			return engine.Place{Level: here.Level,
				Row: ((here.Row+i-LocalRadius)%20 + 20) % 20, Col: ((here.Col+j-LocalRadius)%20 + 20) % 20}
		}
		inWindow := func(p engine.Place) bool {
			dr := ((p.Row-here.Row)%20 + 20 + LocalRadius) % 20
			dc := ((p.Col-here.Col)%20 + 20 + LocalRadius) % 20
			return dr < lv.Size && dc < lv.Size
		}
		// The visited squares are the record's, on both maps.
		rec := c.Game.Exploration.Level(here.Level)
		fullVisited := map[engine.Place]bool{}
		for _, q := range full.Visited {
			fullVisited[engine.Place{Level: here.Level, Row: q.Row, Col: q.Col}] = true
		}
		localVisited := map[engine.Place]bool{}
		for _, q := range lv.Visited {
			localVisited[abs(q.Row, q.Col)] = true
		}
		for r := 0; r < 20; r++ {
			for col := 0; col < 20; col++ {
				p := engine.Place{Level: here.Level, Row: r, Col: col}
				if fullVisited[p] != rec.Visited[r][col] || inWindow(p) && localVisited[p] != rec.Visited[r][col] {
					t.Fatalf("at %v the maps' visited squares differ from the record at %v", here, p)
				}
			}
		}
		if len(localVisited) != len(lv.Visited) || !localVisited[here] {
			t.Fatalf("at %v the minimap's visited squares %+v", here, lv.Visited)
		}
		dirOf := map[string]engine.Direction{"N": engine.North, "E": engine.East, "S": engine.South, "W": engine.West}
		key := func(p engine.Place, w MapWall) string {
			return fmt.Sprintf("%d:%d:%s:%s:%v", p.Row, p.Col, w.Side, w.Kind, w.OneWay)
		}
		fullSet := map[string]bool{}
		for _, w := range full.Walls {
			fullSet[key(engine.Place{Level: here.Level, Row: w.Row, Col: w.Col}, w)] = true
		}
		localSet := map[string]bool{}
		for _, w := range lv.Walls {
			p := abs(w.Row, w.Col)
			q := p.Step(dirOf[w.Side])
			if crosses(p, dirOf[w.Side]) || !inWindow(q) {
				continue
			}
			localSet[key(p, w)] = true
			if !fullSet[key(p, w)] {
				t.Fatalf("at %v the minimap shows %+v (%v) that the full map does not", here, w, p)
			}
			compared++
		}
		for _, w := range full.Walls {
			p := engine.Place{Level: here.Level, Row: w.Row, Col: w.Col}
			if crosses(p, dirOf[w.Side]) || !inWindow(p) || !inWindow(p.Step(dirOf[w.Side])) {
				continue
			}
			if !localSet[key(p, w)] {
				t.Fatalf("at %v the full map shows %+v that the minimap does not", here, w)
			}
		}
	})
	if compared < 5000 {
		t.Fatalf("only %d walls compared", compared)
	}
}

// A teleporter shows nothing on the way: after stepping onto one, only the
// destination is visited and only what is seen there is new.
func TestTeleportRevealsOnlyTheDestination(t *testing.T) {
	c := explorers(t)
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	g := c.Game
	for l, lv := range g.Scenario.Mazes.Levels {
		for y := 0; y < 20; y++ {
			for x := 0; x < 20; x++ {
				cell := lv.Cells[y][x]
				if cell.Type != data.SqTransfer || cell.DestLevel != l+1 || cell.Encounter {
					continue
				}
				for d := engine.North; d <= engine.West; d++ {
					nx, ny := engine.Shift(x, y, d, 0, 1)
					from := lv.Cells[ny][nx]
					if from.Type != data.SqNormal || [...]data.WallType{from.N, from.E, from.S, from.W}[d.Reverse()] != data.WallOpen {
						continue
					}
					// Stand beside it, facing it, and step on.
					g.Exploration = nil
					g.Phase, g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = engine.PhaseMaze, l, nx, ny, d.Reverse()
					g.MazeMessage, g.MazeMessage2 = "", ""
					g.InitFightMap()
					g.FightMap = [20][20]bool{}
					g.Explore()
					before := g.Exploration.Clone()
					src, _ := engine.PlaceOf(len(g.Scenario.Mazes.Levels), l, x, y)
					seed(5)
					must(t, c.Forward())
					if g.Phase != engine.PhaseMaze {
						t.Fatalf("phase %v", g.Phase)
					}
					g.Explore()
					dest, _ := g.Place()
					want, _ := engine.PlaceOf(len(g.Scenario.Mazes.Levels), cell.DestLevel-1, cell.DestX, cell.DestY)
					if dest != want {
						t.Fatalf("teleported to %v, want %v", dest, want)
					}
					m := g.Exploration.Level(dest.Level)
					if m.Visited[src.Row][src.Col] {
						t.Fatal("the teleporter's square was visited")
					}
					// Every side new since the step is one seen from the destination.
					sight, _ := g.Sight()
					seen := map[engine.Place]bool{}
					for _, sq := range sight {
						p, _ := engine.PlaceOf(len(g.Scenario.Mazes.Levels), g.MazeLevel, sq.X, sq.Y)
						seen[p] = true
						for _, r := range []int{-1, 1} {
							sx, sy := engine.Shift(sq.X, sq.Y, g.Facing, r, 0)
							q, _ := engine.PlaceOf(len(g.Scenario.Mazes.Levels), g.MazeLevel, sx, sy)
							seen[q] = true
						}
					}
					old := before.Level(dest.Level)
					for r := 0; r < 20; r++ {
						for col := 0; col < 20; col++ {
							p := engine.Place{Level: dest.Level, Row: r, Col: col}
							if (old == nil || m.Walls[r][col] != old.Walls[r][col]) && !seen[p] && m.Walls[r][col] != [4]string{} {
								t.Fatalf("%v was revealed by teleporting from %v to %v", p, src, dest)
							}
						}
					}
					return
				}
			}
		}
	}
	t.Fatal("no teleporter with an open approach")
}
