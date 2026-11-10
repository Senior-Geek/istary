package engine

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/data"
)

// mazeAt is a game with the party in the maze of scenario 1 at an internal
// position and facing.
func mazeAt(t *testing.T, level, x, y int, f Direction) *GameState {
	t.Helper()
	g := newTestGame(t, 1)
	g.Phase = PhaseMaze
	g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = level, x, y, f
	return g
}

func placeOf(g *GameState, x, y int) Place {
	p, _ := PlaceOf(len(g.Scenario.Mazes.Levels), g.MazeLevel, x, y)
	return p
}

// side is what was recorded on side d of the square at internal x, y.
func side(g *GameState, x, y int, d Direction) string {
	p := placeOf(g, x, y)
	m := g.Exploration.Level(p.Level)
	if m == nil {
		return ""
	}
	return m.Walls[p.Row][p.Col][d]
}

func cellSide(c *data.MazeCell, d Direction) data.WallType {
	return [...]data.WallType{c.N, c.E, c.S, c.W}[d]
}

// Entering the maze: the party stands at L1:R19:C0 facing north, on the
// stairs up; at Camp only the square itself is known, and once in the maze
// what is seen ahead.
func TestExploreEntrance(t *testing.T) {
	g := newTestGame(t, 1)
	g.Phase = PhaseCamp
	g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = 0, 0, 0, North
	if here, ok := g.Place(); !ok || here != Entrance {
		t.Fatalf("at %v", here)
	}
	if !g.Explore() {
		t.Fatal("nothing recorded")
	}
	m := g.Exploration.Level(1)
	if !m.Visited[19][0] || m.Features[19][0] != FeatureStairsUp {
		t.Fatalf("entrance: %v %q", m.Visited[19][0], m.Features[19][0])
	}
	if m.Walls[19][0] != [4]string{} {
		t.Fatalf("at Camp a wall was seen: %v", m.Walls[19][0])
	}
	if g.Explore() {
		t.Error("recorded twice")
	}
	g.Phase = PhaseMaze
	g.Explore()
	sight, _ := g.Sight()
	if len(sight) == 0 || side(g, 0, 0, North) != sight[0].Front || side(g, 0, 0, West) != sight[0].Left || side(g, 0, 0, East) != sight[0].Right {
		t.Fatalf("entrance walls %v, sight %+v", m.Walls[19][0], sight)
	}
	if m.Walls[19][0][South] != "" {
		t.Error("the wall behind was recorded")
	}
	if g.Exploration.Dungeon != "1" {
		t.Errorf("dungeon %q", g.Exploration.Dungeon)
	}
}

// Seeded walks on every level, with and without light: what is recorded is
// exactly what Sight showed (each wall from the side seen), every recorded
// wall is true to the maze (a hidden door is a wall or, seen with light, a
// door), the squares visited are those stood on, and features are known
// only where the party stood.
func TestExploreRecordsOnlyWhatIsSeen(t *testing.T) {
	g := newTestGame(t, 1)
	levels := len(g.Scenario.Mazes.Levels)
	type key struct {
		p Place
		d Direction
	}
	seen := map[key]string{}
	stood := map[Place]bool{}
	hiddenLit := map[key]bool{}
	rng := rand.New(rand.NewSource(7))
	g.Phase = PhaseMaze
	for level := 0; level < levels; level++ {
		g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = level, rng.Intn(20), rng.Intn(20), Direction(rng.Intn(4))
		for step := 0; step < 1500; step++ {
			g.LightLevel = 0
			if step%400 > 250 {
				g.LightLevel = 1
			}
			g.QuickPlot = step%700 > 600
			switch r := rng.Intn(10); {
			case r < 6:
				if !g.MoveForward() && rng.Intn(3) == 0 {
					// Kick through a door now and then, as the party may.
					if g.KickDoor() {
						g.PlayerX, g.PlayerY = Shift(g.PlayerX, g.PlayerY, g.Facing, 0, 1)
					}
				}
			case r < 8:
				g.TurnLeft()
			default:
				g.TurnRight()
			}
			g.Explore()
			here, _ := g.Place()
			stood[here] = true
			sight, _ := g.Sight()
			f := g.Facing
			for _, sq := range sight {
				p := placeOf(g, sq.X, sq.Y)
				seen[key{p, f.TurnLeft()}] = sq.Left
				seen[key{p, f.TurnRight()}] = sq.Right
				seen[key{p, f}] = sq.Front
				if sq.LeftAhead != "" {
					x, y := Shift(sq.X, sq.Y, f, -1, 0)
					seen[key{placeOf(g, x, y), f}] = sq.LeftAhead
				}
				if sq.RightAhead != "" {
					x, y := Shift(sq.X, sq.Y, f, 1, 0)
					seen[key{placeOf(g, x, y), f}] = sq.RightAhead
				}
				if g.LightLevel > 0 {
					hiddenLit[key{p, f}] = true
				}
			}
		}
	}
	recorded := 0
	for l, m := range g.Exploration.Levels {
		for r := 0; r < MazeSize; r++ {
			for c := 0; c < MazeSize; c++ {
				p := Place{l, r, c}
				if m.Visited[r][c] != stood[p] {
					t.Fatalf("%v visited %v, stood %v", p, m.Visited[r][c], stood[p])
				}
				lvl, x, y := p.Internal()
				cell := &g.Scenario.Mazes.Levels[lvl].Cells[y][x]
				if f := m.Features[r][c]; f != "" && (!stood[p] || f != featureOf(cell, l)) {
					t.Fatalf("%v feature %q", p, f)
				}
				for d := North; d <= West; d++ {
					got, want := m.Walls[r][c][d], seen[key{p, d}]
					if got != want {
						t.Fatalf("%v %v: recorded %q, last seen %q", p, d, got, want)
					}
					if got == "" {
						continue
					}
					recorded++
					truth := map[data.WallType][]string{
						data.WallOpen: {SeenOpen}, data.WallWall: {SeenWall},
						data.WallDoor: {SeenDoor}, data.WallHidden: {SeenWall, SeenDoor},
					}[cellSide(cell, d)]
					if !strings.Contains(strings.Join(truth, " "), got) {
						t.Fatalf("%v %v: recorded %q, maze has %q", p, d, got, cellSide(cell, d))
					}
				}
			}
		}
	}
	if recorded < 1000 || len(g.Exploration.Levels) != levels {
		t.Fatalf("only %d sides on %d levels", recorded, len(g.Exploration.Levels))
	}
}

// A door ahead is seen before its square is entered; nothing beyond it is.
func TestExploreDoorAheadAndBlocking(t *testing.T) {
	g := newTestGame(t, 1)
	cells := g.Scenario.Mazes.Levels[0].Cells
	found := false
	for y := 0; y < MazeSize && !found; y++ {
		for x := 0; x < MazeSize && !found; x++ {
			for d := North; d <= West && !found; d++ {
				h := mazeAt(t, 0, x, y, d)
				sight, _ := h.Sight()
				if len(sight) < 2 || sight[len(sight)-1].Front != SeenDoor {
					continue
				}
				found = true
				last := sight[len(sight)-1]
				h.Explore()
				if side(h, last.X, last.Y, d) != SeenDoor {
					t.Fatalf("door ahead not recorded")
				}
				lp := placeOf(h, last.X, last.Y)
				if h.Exploration.Level(1).Visited[lp.Row][lp.Col] {
					t.Fatal("the door's square counted as visited")
				}
				bx, by := Shift(last.X, last.Y, d, 0, 1)
				bp := placeOf(h, bx, by)
				// Beyond the door: nothing, unless it is also beside the corridor.
				for _, sq := range sight {
					for _, r := range []int{-1, 1} {
						if sx, sy := Shift(sq.X, sq.Y, d, r, 0); sx == bx && sy == by {
							t.Skip("beyond square is beside the corridor")
						}
					}
				}
				if w := h.Exploration.Level(1).Walls[bp.Row][bp.Col]; w != [4]string{} {
					t.Fatalf("beyond the door at %v: %v", bp, w)
				}
			}
		}
	}
	if !found {
		t.Fatal("no door ahead on level 1")
	}
	_ = cells
	// A wall in front: one square seen, the square behind the wall unknown.
	for y := 0; y < MazeSize; y++ {
		for x := 0; x < MazeSize; x++ {
			c := &cells[y][x]
			if c.N != data.WallWall || c.Type == data.SqDark {
				continue
			}
			h := mazeAt(t, 0, x, y, North)
			h.Explore()
			bx, by := Shift(x, y, North, 0, 1)
			bp := placeOf(h, bx, by)
			if w := h.Exploration.Level(1).Walls[bp.Row][bp.Col]; w != [4]string{} {
				t.Fatalf("behind the wall north of %v: %v", placeOf(h, x, y), w)
			}
			return
		}
	}
	t.Fatal("no wall found")
}

// Turning shows the walls the party now faces.
func TestExploreTurning(t *testing.T) {
	g := mazeAt(t, 0, 0, 0, North)
	g.Explore()
	if side(g, 0, 0, South) != "" {
		t.Fatal("south seen while facing north")
	}
	g.TurnRight()
	g.TurnRight()
	if !g.Explore() {
		t.Fatal("turning about showed nothing new")
	}
	if side(g, 0, 0, South) != SeenWall {
		t.Errorf("south of the entrance: %q", side(g, 0, 0, South))
	}
}

// A hidden door is a wall without light and a door with it, never a door
// merely because the maze has one.
func TestExploreHiddenDoors(t *testing.T) {
	g := newTestGame(t, 1)
	cells := g.Scenario.Mazes.Levels[0].Cells
	for y := 0; y < MazeSize; y++ {
		for x := 0; x < MazeSize; x++ {
			for d := North; d <= West; d++ {
				if cellSide(&cells[y][x], d) != data.WallHidden || cells[y][x].Type == data.SqDark {
					continue
				}
				h := mazeAt(t, 0, x, y, d)
				h.Explore()
				if got := side(h, x, y, d); got != SeenWall {
					t.Fatalf("hidden door in the dark recorded as %q", got)
				}
				h.LightLevel = 10
				h.Explore()
				if got := side(h, x, y, d); got != SeenDoor {
					t.Fatalf("hidden door with light recorded as %q", got)
				}
				return
			}
		}
	}
	t.Fatal("no hidden door on level 1")
}

// one finds, on any level, a lit square whose side d leads (through an
// open side, or across the edge when edge) into a lit neighbour, whose own
// side back is other ("" for any). It returns the level and position.
func one(t *testing.T, g *GameState, edge bool, d Direction, mine data.WallType, other data.WallType) (int, int, int, bool) {
	t.Helper()
	for l, lv := range g.Scenario.Mazes.Levels {
		for y := 0; y < MazeSize; y++ {
			for x := 0; x < MazeSize; x++ {
				nx, ny := Shift(x, y, d, 0, 1)
				crossing := (nx-x)*(nx-x)+(ny-y)*(ny-y) > 1
				a, b := &lv.Cells[y][x], &lv.Cells[ny][nx]
				if crossing != edge || a.Type == data.SqDark || b.Type == data.SqDark {
					continue
				}
				if cellSide(a, d) == mine && (other == "" || cellSide(b, d.Reverse()) == other) {
					return l, x, y, true
				}
			}
		}
	}
	return 0, 0, 0, false
}

// One-way walls are kept by side: the party steps through an opening that
// is a wall from the other side, and the two sides are recorded apart.
// (The one on L1 at R19:C10-11 is in darkness, where nothing is seen.)
func TestExploreOneWay(t *testing.T) {
	g := newTestGame(t, 1)
	n := 0
	for d := North; d <= West; d++ {
		l, x, y, ok := one(t, g, false, d, data.WallOpen, data.WallWall)
		if !ok {
			continue
		}
		n++
		h := mazeAt(t, l, x, y, d)
		h.Explore()
		if !h.MoveForward() {
			t.Fatalf("cannot step %v from %v", d, placeOf(h, x, y))
		}
		nx, ny := h.PlayerX, h.PlayerY
		h.TurnRight()
		h.TurnRight()
		h.Explore()
		if h.MoveForward() {
			t.Fatal("stepped back through a one-way wall")
		}
		if side(h, x, y, d) != SeenOpen || side(h, nx, ny, d.Reverse()) != SeenWall {
			t.Fatalf("%v %v %q, back %q", placeOf(h, x, y), d, side(h, x, y, d), side(h, nx, ny, d.Reverse()))
		}
	}
	if n == 0 {
		t.Fatal("no lit one-way passage in the maze")
	}
}

// The maze wraps at all four edges, and so does what is seen.
func TestExploreWraps(t *testing.T) {
	g := newTestGame(t, 1)
	for d := North; d <= West; d++ {
		l, x, y, ok := one(t, g, true, d, data.WallOpen, "")
		if !ok {
			t.Fatalf("no lit opening across the %v edge", d)
		}
		h := mazeAt(t, l, x, y, d)
		h.Explore()
		from := placeOf(h, x, y)
		want := from.Step(d)
		m := h.Exploration.Level(l + 1)
		if m.Walls[want.Row][want.Col] == [4]string{} {
			t.Errorf("%v: nothing seen across the edge at %v", from, want)
		}
		if !h.MoveForward() {
			t.Fatalf("%v: cannot cross", from)
		}
		if here, _ := h.Place(); here != want {
			t.Errorf("%v %v crossed to %v, want %v", from, d, here, want)
		}
		h.Explore()
		if !m.Visited[want.Row][want.Col] {
			t.Error("not visited across the edge")
		}
	}
}

// Stairs and the elevator are known by standing on them, not by seeing them.
func TestExploreFeatures(t *testing.T) {
	for _, c := range []struct {
		x, y int
		want string
	}{{0, 0, FeatureStairsUp}, {0, 10, FeatureStairsDown}, {10, 8, FeatureElevator}} {
		g := newTestGame(t, 1)
		cells := g.Scenario.Mazes.Levels[0].Cells
		// From a neighbour looking in through an opening: seen, not known.
		for d := North; d <= West; d++ {
			nx, ny := Shift(c.x, c.y, d, 0, 1)
			if cellSide(&cells[ny][nx], d.Reverse()) != data.WallOpen || cells[ny][nx].Type == data.SqDark {
				continue
			}
			h := mazeAt(t, 0, nx, ny, d.Reverse())
			h.Explore()
			p := placeOf(h, c.x, c.y)
			if f := h.Exploration.Level(1).Features[p.Row][p.Col]; f != "" {
				t.Errorf("%v: %q known from a neighbour", p, f)
			}
		}
		h := mazeAt(t, 0, c.x, c.y, North)
		h.Explore()
		p := placeOf(h, c.x, c.y)
		if f := h.Exploration.Level(1).Features[p.Row][p.Col]; f != c.want {
			t.Errorf("%v: %q, want %q", p, f, c.want)
		}
	}
}

// The exploration survives save and load, the castle and going back in;
// the castle records nothing.
func TestExplorationPersists(t *testing.T) {
	freshHome(t)
	g := mazeAt(t, 0, 0, 0, North)
	g.Explore()
	g.TurnRight()
	g.Explore()
	g.MazeLevel, g.PlayerX, g.PlayerY = 1, 12, 7
	g.Explore()
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(mustRosterPath(t))
	if !bytes.Contains(raw, []byte(`"exploration"`)) {
		t.Fatal("no section saved")
	}
	h := newTestGame(t, 1)
	if err := h.Load(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h.Exploration, g.Exploration) {
		t.Fatal("exploration changed by save and load")
	}
	// The castle: nothing recorded, nothing lost; back in, more is added.
	before := h.Exploration.Clone()
	h.Phase = PhaseTown
	if h.Explore() || !reflect.DeepEqual(h.Exploration, before) {
		t.Fatal("recorded in the castle")
	}
	h.Phase, h.MazeLevel, h.PlayerX, h.PlayerY, h.Facing = PhaseMaze, 0, 0, 0, East
	h.Explore()
	if !reflect.DeepEqual(h.Exploration.Level(2), before.Level(2)) || h.Exploration.Level(1).Walls[19][0][East] == "" {
		t.Fatal("re-entry lost or failed to add exploration")
	}
	// A clone shares nothing.
	c := h.Exploration.Clone()
	c.Level(1).Visited[0][0] = true
	if h.Exploration.Level(1).Visited[0][0] {
		t.Fatal("clone shares a level")
	}
}

func mustRosterPath(t *testing.T) string {
	t.Helper()
	p, err := RosterPath("1")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// An old save has no section: nothing is made up, and the first record is
// only what is seen from where the party stands. Sections this version
// cannot use are kept and written back, and nothing is recorded over them.
func TestExplorationOldAndOtherSaves(t *testing.T) {
	freshHome(t)
	path := mustRosterPath(t)
	if err := os.WriteFile(path, []byte(`{"roster":[],"party":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	g := newTestGame(t, 1)
	if err := g.Load(); err != nil || g.Exploration != nil || g.OtherExploration != nil {
		t.Fatalf("old save: %v %v %v", err, g.Exploration, g.OtherExploration)
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(path); bytes.Contains(raw, []byte("exploration")) {
		t.Fatal("an empty section was written")
	}
	g.Phase, g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = PhaseMaze, 4, 8, 8, North
	g.Explore()
	if levels := g.Exploration.LevelsSeen(); len(levels) != 1 || levels[0] != 5 {
		t.Fatalf("levels %v", levels)
	}
	n := 0
	for r := range g.Exploration.Level(5).Visited {
		for c := range g.Exploration.Level(5).Visited[r] {
			if g.Exploration.Level(5).Visited[r][c] {
				n++
			}
		}
	}
	if n != 1 {
		t.Fatalf("%d squares visited", n)
	}

	for _, section := range []string{
		`{"version":2,"dungeon":"1","levels":[]}`,
		`{"version":1,"dungeon":"2","levels":[]}`,
		`{"version":1,"dungeon":"1","levels":[{"level":1,"visited":["x"],"walls":[]}]}`,
	} {
		file := `{"roster":[],"party":[],"exploration":` + section + `}`
		if err := os.WriteFile(path, []byte(file), 0644); err != nil {
			t.Fatal(err)
		}
		h := newTestGame(t, 1)
		if err := h.Load(); err != nil || h.Exploration != nil || h.OtherExploration == nil {
			t.Fatalf("%s: %v %v", section, err, h.Exploration)
		}
		h.Phase = PhaseMaze
		if h.Explore() || h.Exploration != nil {
			t.Fatal("recorded over a section kept as it was")
		}
		if err := h.Save(); err != nil {
			t.Fatal(err)
		}
		var st struct {
			Exploration json.RawMessage `json:"exploration"`
		}
		raw, _ := os.ReadFile(path)
		if err := json.Unmarshal(raw, &st); err != nil {
			t.Fatal(err)
		}
		var a, b bytes.Buffer
		json.Compact(&a, st.Exploration)
		json.Compact(&b, []byte(section))
		if a.String() != b.String() {
			t.Fatalf("written back as %s", a.String())
		}
	}
}

// Off the maze (the elevator's missing floors, #31) nothing is recorded.
func TestExploreOffTheMaze(t *testing.T) {
	g := mazeAt(t, 10, 10, 0, North)
	if g.Explore() || g.Exploration != nil {
		t.Fatal("recorded on a level that does not exist")
	}
}
