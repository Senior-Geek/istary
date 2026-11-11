package engine

// The rescue area (R-2): the original's EXPLROOM on synthetic levels whose
// every wall is known. Numbers in the comments are the R-2 brief's test
// scenarios (section 12).

import (
	"testing"

	"github.com/Senior-Geek/istary/data"
)

// walled returns a scenario of n levels in which every square is walled in
// on all four sides.
func walled(n int) *data.Scenario {
	sc := &data.Scenario{}
	for l := 0; l < n; l++ {
		lv := data.MazeLevel{Level: l + 1}
		for y := 0; y < MazeSize; y++ {
			row := make([]data.MazeCell, MazeSize)
			for x := range row {
				row[x] = data.MazeCell{N: data.WallWall, S: data.WallWall, E: data.WallWall, W: data.WallWall}
			}
			lv.Cells = append(lv.Cells, row)
		}
		sc.Mazes.Levels = append(sc.Mazes.Levels, lv)
	}
	return sc
}

// setWall sets the wall of (x, y) on level l facing d to w; with back, the
// neighbour's wall facing back too (an ordinary two-sided wall).
func setWall(sc *data.Scenario, l, x, y int, d Direction, w data.WallType, back bool) {
	cells := sc.Mazes.Levels[l].Cells
	set := func(c *data.MazeCell, d Direction) {
		switch d {
		case North:
			c.N = w
		case East:
			c.E = w
		case South:
			c.S = w
		case West:
			c.W = w
		}
	}
	set(&cells[y][x], d)
	if back {
		nx, ny := (x+d.DX()+MazeSize)%MazeSize, (y+d.DY()+MazeSize)%MazeSize
		set(&cells[ny][nx], (d+2)%4)
	}
}

func areaOf(t *testing.T, sc *data.Scenario, l, x, y int) Area {
	t.Helper()
	a, ok := RescueArea(sc, l, x, y)
	if !ok {
		t.Fatalf("no area at L%d (%d,%d)", l, x, y)
	}
	return a
}

func count(a Area) int {
	n := 0
	for x := range a {
		for y := range a[x] {
			if a[x][y] {
				n++
			}
		}
	}
	return n
}

// (1, 2, 3, 4) The party's own square; through an open wall; not through a
// door or a hidden door, from either side.
func TestRescueAreaWallsAndDoors(t *testing.T) {
	sc := walled(1)
	if a := areaOf(t, sc, 0, 5, 5); !a[5][5] || count(a) != 1 {
		t.Fatalf("walled in: %d squares", count(a))
	}
	setWall(sc, 0, 5, 5, East, data.WallOpen, true)
	setWall(sc, 0, 6, 5, East, data.WallDoor, true)
	setWall(sc, 0, 5, 5, North, data.WallHidden, true)
	a := areaOf(t, sc, 0, 5, 5)
	if !a[6][5] || a[7][5] || a[5][6] || count(a) != 2 {
		t.Fatalf("area: (6,5) %v door (7,5) %v hidden (5,6) %v, %d squares", a[6][5], a[7][5], a[5][6], count(a))
	}
	if b := areaOf(t, sc, 0, 7, 5); b[6][5] || count(b) != 1 {
		t.Fatalf("through the door from the far side: %d squares", count(b))
	}
}

// (5, 6) A one-way passage is followed only the way it can be walked: each
// square's own wall decides.
func TestRescueAreaOneWay(t *testing.T) {
	sc := walled(1)
	setWall(sc, 0, 5, 5, East, data.WallOpen, false) // (5,5) -> (6,5) only; (6,5)'s west stays a wall
	setWall(sc, 0, 6, 5, East, data.WallOpen, true)  // and on to (7,5) both ways
	if a := areaOf(t, sc, 0, 5, 5); !a[6][5] || !a[7][5] || count(a) != 3 {
		t.Fatalf("the open way: (6,5) %v (7,5) %v, %d squares", a[6][5], a[7][5], count(a))
	}
	if a := areaOf(t, sc, 0, 7, 5); a[5][5] || !a[6][5] || count(a) != 2 {
		t.Fatalf("against it: (5,5) %v, %d squares", a[5][5], count(a))
	}
}

// (7) The area wraps at the edges, east to west and north to south, as the
// original's X and Y are taken modulo 20.
func TestRescueAreaWraps(t *testing.T) {
	sc := walled(1)
	setWall(sc, 0, 19, 5, East, data.WallOpen, true)
	setWall(sc, 0, 0, 19, North, data.WallOpen, true)
	if a := areaOf(t, sc, 0, 19, 5); !a[0][5] || count(a) != 2 {
		t.Fatalf("east edge: (0,5) %v, %d squares", a[0][5], count(a))
	}
	if a := areaOf(t, sc, 0, 0, 0); !a[0][19] || count(a) != 2 {
		t.Fatalf("south edge: (0,19) %v, %d squares", a[0][19], count(a))
	}
}

// (8, 10) The area is one level's: an open way never leads to another
// level, and stairs, chutes, pits and teleporters are only squares: their
// destinations are not followed, and they do not stop the search.
func TestRescueAreaStaysOnTheLevel(t *testing.T) {
	sc := walled(2)
	cells := sc.Mazes.Levels[0].Cells
	cells[5][5] = data.MazeCell{N: data.WallWall, S: data.WallWall, E: data.WallOpen, W: data.WallWall,
		Type: data.SqTransfer, DestLevel: 2, DestX: 15, DestY: 15}
	cells[5][6].W = data.WallOpen
	cells[5][6].Type, cells[5][6].DestLevel, cells[5][6].DestX, cells[5][6].DestY = data.SqChute, 2, 10, 10
	setWall(sc, 0, 6, 5, East, data.WallOpen, true)
	cells[5][7].Type, cells[5][7].DestLevel = data.SqStairs, 2
	setWall(sc, 0, 7, 5, East, data.WallOpen, true)
	cells[5][8].Type = data.SqPit
	a := areaOf(t, sc, 0, 5, 5)
	if count(a) != 4 || !a[8][5] || a[15][15] || a[10][10] {
		t.Fatalf("area: %d squares, past the pit %v, teleporter's destination %v, chute's %v", count(a), a[8][5], a[15][15], a[10][10])
	}
	if b := areaOf(t, sc, 1, 5, 5); count(b) != 1 {
		t.Fatalf("level 2: %d squares", count(b))
	}
	if _, ok := RescueArea(sc, 2, 5, 5); ok {
		t.Fatal("an area on a level past the last")
	}
	if _, ok := RescueArea(sc, 0, 20, 5); ok {
		t.Fatal("an area off the level")
	}
}

// The real data: on every level of every square of Wizardry I the area
// holds the party's square, and an open wall always leads into it.
func TestRescueAreaOnTheRealMazes(t *testing.T) {
	sc := loadScenario(t, 1)
	for l := range sc.Mazes.Levels {
		for _, xy := range [][2]int{{0, 0}, {9, 9}, {19, 19}} {
			a := areaOf(t, sc, l, xy[0], xy[1])
			for x := 0; x < MazeSize; x++ {
				for y := 0; y < MazeSize; y++ {
					if !a[x][y] {
						continue
					}
					c := sc.Mazes.Levels[l].Cells[y][x]
					for _, step := range []struct {
						w      data.WallType
						dx, dy int
					}{{c.N, 0, 1}, {c.E, 1, 0}, {c.S, 0, -1}, {c.W, -1, 0}} {
						if step.w == data.WallOpen && !a[(x+step.dx+MazeSize)%MazeSize][(y+step.dy+MazeSize)%MazeSize] {
							t.Fatalf("L%d from (%v): open wall of (%d,%d) leads out of the area", l+1, xy, x, y)
						}
					}
				}
			}
		}
	}
}
