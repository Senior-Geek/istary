package engine

// What the party sees from where it stands (M-1): the corridor ahead as far
// as the Classic view draws it (Pascal DRAWMAZE), and nothing more. The
// first-person view (app.DungeonView) and the exploration record
// (explore.go) both read it, so the map can never show more than the view.
//
// Five squares with light (three with QUICK plot), two without; the walk
// stops at a dark square (or sees nothing in one) and after the first wall
// or door in front. Each wall is seen from the side the party is on, as
// "open", "wall" or "door": a hidden door looks like a wall unless the
// party has light. (The Classic renderer also lets one flicker into view
// one draw in six without light; that needs a random number, which nothing
// that only looks may draw, so it is left out.)

import "github.com/Senior-Geek/istary/data"

// Wall kinds as seen.
const (
	SeenOpen = "open"
	SeenWall = "wall"
	SeenDoor = "door"
)

// SightSquare is one square of the corridor ahead, nearest first: its
// internal position and the walls seen on its left, right and front. Where
// a side is open, the wall in front of the square on that side is seen too
// (LeftAhead, RightAhead), as the Classic view draws it.
type SightSquare struct {
	X, Y       int
	Left       string
	Right      string
	Front      string
	LeftAhead  string
	RightAhead string
}

// Sight returns the corridor the party sees, nearest square first, and
// whether the view ends in a dark square. It reads the game and changes
// nothing; it is empty off the maze.
func (g *GameState) Sight() ([]SightSquare, bool) {
	if g.MazeLevel < 0 || g.MazeLevel >= len(g.Scenario.Mazes.Levels) {
		return nil, false
	}
	level := &g.Scenario.Mazes.Levels[g.MazeLevel]
	light := g.LightLevel > 0
	depth := 2
	if light {
		depth = 5
		if g.QuickPlot {
			depth = 3
		}
	}
	shown := func(w data.WallType) string {
		switch w {
		case data.WallOpen:
			return SeenOpen
		case data.WallDoor:
			return SeenDoor
		case data.WallHidden:
			if light {
				return SeenDoor
			}
		}
		return SeenWall
	}
	f := g.Facing
	var out []SightSquare
	x, y := g.PlayerX, g.PlayerY
	for ; depth > 0; depth-- {
		if level.Cells[y][x].Type == data.SqDark {
			return out, true
		}
		sq := SightSquare{
			X: x, Y: y,
			Left:  shown(wallAt(level, x, y, f, 0, f.TurnLeft())),
			Right: shown(wallAt(level, x, y, f, 0, f.TurnRight())),
			Front: shown(wallAt(level, x, y, f, 0, f)),
		}
		if sq.Left == SeenOpen {
			sq.LeftAhead = shown(wallAt(level, x, y, f, -1, f))
		}
		if sq.Right == SeenOpen {
			sq.RightAhead = shown(wallAt(level, x, y, f, 1, f))
		}
		out = append(out, sq)
		if sq.Front != SeenOpen {
			break
		}
		x, y = Shift(x, y, f, 0, 1)
	}
	return out, false
}

// Shift moves right and forward relative to facing, in internal
// coordinates, wrapping at the edges (Pascal SHFTPOS; north is Y+1).
func Shift(x, y int, facing Direction, right, forward int) (int, int) {
	switch facing {
	case North:
		x, y = x+right, y+forward
	case East:
		x, y = x+forward, y-right
	case South:
		x, y = x-right, y-forward
	case West:
		x, y = x-forward, y+right
	}
	return (x%MazeSize + MazeSize) % MazeSize, (y%MazeSize + MazeSize) % MazeSize
}

// wallAt is the wall on side dir of the square right squares to the right.
func wallAt(level *data.MazeLevel, x, y int, facing Direction, right int, dir Direction) data.WallType {
	x, y = Shift(x, y, facing, right, 0)
	cell := level.Cells[y][x]
	switch dir {
	case North:
		return cell.N
	case East:
		return cell.E
	case South:
		return cell.S
	case West:
		return cell.W
	}
	return data.WallWall
}
