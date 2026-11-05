package engine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/data"
)

// mazeGame returns a scenario-1 game positioned on B1F. Tests may edit the
// cells freely because loadScenario returns a fresh copy.
func mazeGame(t *testing.T) *GameState {
	t.Helper()
	g := New(loadScenario(t, 1))
	g.Phase = PhaseMaze
	return g
}

func TestDirections(t *testing.T) {
	d := North
	for _, want := range []Direction{East, South, West, North} {
		d = d.TurnRight()
		if d != want {
			t.Fatalf("TurnRight = %v, want %v", d, want)
		}
	}
	for _, want := range []Direction{West, South, East, North} {
		d = d.TurnLeft()
		if d != want {
			t.Fatalf("TurnLeft = %v, want %v", d, want)
		}
	}
	if North.Reverse() != South || East.Reverse() != West {
		t.Error("Reverse wrong")
	}
	// Pascal coordinates: North is +Y, East is +X.
	if North.DY() != 1 || South.DY() != -1 || East.DX() != 1 || West.DX() != -1 {
		t.Error("DX/DY do not match Pascal coordinates")
	}
}

func TestMoveForwardAndKickByWallType(t *testing.T) {
	tests := []struct {
		wall       data.WallType
		walk, kick bool
	}{
		{data.WallOpen, true, true},
		{data.WallDoor, false, true},
		{data.WallHidden, false, true},
		{data.WallWall, false, false},
	}
	for _, tt := range tests {
		g := mazeGame(t)
		g.PlayerX, g.PlayerY, g.Facing = 5, 5, East
		g.CurrentCell().E = tt.wall
		if got := g.KickDoor(); got != tt.kick {
			t.Errorf("%s: KickDoor = %v, want %v", tt.wall, got, tt.kick)
		}
		moved := g.MoveForward()
		if moved != tt.walk {
			t.Errorf("%s: MoveForward = %v, want %v", tt.wall, moved, tt.walk)
		}
		wantX := 5
		if tt.walk {
			wantX = 6
		}
		if g.PlayerX != wantX || g.PlayerY != 5 {
			t.Errorf("%s: position = (%d,%d), want (%d,5)", tt.wall, g.PlayerX, g.PlayerY, wantX)
		}
	}
}

func TestMoveForwardWrapsAround(t *testing.T) {
	g := mazeGame(t)
	cases := []struct {
		x, y         int
		face         Direction
		wantX, wantY int
	}{
		{19, 3, East, 0, 3},
		{0, 3, West, 19, 3},
		{4, 19, North, 4, 0},
		{4, 0, South, 4, 19},
	}
	for _, c := range cases {
		g.PlayerX, g.PlayerY, g.Facing = c.x, c.y, c.face
		cell := g.CurrentCell()
		cell.N, cell.S, cell.E, cell.W = data.WallOpen, data.WallOpen, data.WallOpen, data.WallOpen
		if !g.MoveForward() || g.PlayerX != c.wantX || g.PlayerY != c.wantY {
			t.Errorf("from (%d,%d) facing %v: got (%d,%d), want (%d,%d)",
				c.x, c.y, c.face, g.PlayerX, g.PlayerY, c.wantX, c.wantY)
		}
	}
}

// Walls of the real B1F starting square, so a data or loader change that
// moves the entrance is caught.
func TestB1FStartSquare(t *testing.T) {
	g := mazeGame(t)
	c := g.CurrentCell()
	got := fmt.Sprintf("N=%s E=%s S=%s W=%s type=%s", c.N, c.E, c.S, c.W, c.Type)
	checkGolden(t, "b1f_start_square.golden", []byte(got+"\n"))
}

// Seeded fight-zone map for every level of every scenario.
func TestInitFightMapSeeded(t *testing.T) {
	var out strings.Builder
	for num := 1; num <= 3; num++ {
		g := New(loadScenario(t, num))
		for lvl := range g.Scenario.Mazes.Levels {
			seed(int64(1000*num + lvl))
			g.MazeLevel = lvl
			g.InitFightMap()
			fmt.Fprintf(&out, "wiz%d level %d\n", num, lvl+1)
			for y := 19; y >= 0; y-- {
				for x := 0; x < 20; x++ {
					if g.FightMap[x][y] {
						out.WriteByte('#')
					} else {
						out.WriteByte('.')
					}
				}
				out.WriteByte('\n')
			}
		}
	}
	checkGolden(t, "fightmap_seeded.golden", []byte(out.String()))
}

func TestClearFightRoomFloodFills(t *testing.T) {
	g := mazeGame(t)
	cells := g.Scenario.Mazes.Levels[0].Cells
	// Make a 2x1 room at (2,2)-(3,2) joined by an open wall.
	for _, p := range [][2]int{{2, 2}, {3, 2}} {
		c := &cells[p[1]][p[0]]
		c.N, c.S, c.E, c.W = data.WallWall, data.WallWall, data.WallWall, data.WallWall
		c.Encounter = true
	}
	cells[2][2].E = data.WallOpen
	cells[2][3].W = data.WallOpen
	g.FightMap = [20][20]bool{}
	g.fillRoom(cells, 2, 2)
	if !g.FightMap[2][2] || !g.FightMap[3][2] {
		t.Fatal("fillRoom should mark both connected squares")
	}
	g.ClearFightRoom(3, 2)
	if g.FightMap[2][2] || g.FightMap[3][2] {
		t.Error("ClearFightRoom should clear the whole connected room")
	}
}
