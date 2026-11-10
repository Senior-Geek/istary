package engine

// Canonical dungeon coordinates, L:R:C (M-1). The engine keeps its own
// coordinates, those of the original: a 0-based level (0 is B1F), X
// increasing east and Y increasing north from the south-west corner, where
// the stairs to the castle are (Player's Guide p.44, DUMAPIC). Everything
// shown to the player, and every record kept for display (exploration,
// later rescue markers), uses L:R:C instead:
//
//	L  0 is the castle; 1 is the first maze level (internal level 0)
//	R  rows from 0 at the north edge, increasing southward (R = 19 - Y)
//	C  columns from 0 at the west edge, increasing eastward (C = X)
//
// This file is the only place that converts between the two. Facing is
// kept apart, as N, E, S or W.

import (
	"errors"
	"fmt"

	"github.com/Senior-Geek/istary/data"
)

// MazeSize is the width and height of every maze level, in squares.
const MazeSize = 20

// Place is a square in L:R:C. The castle is Place{} (L0), with no row or
// column.
type Place struct {
	Level int `json:"level"` // L: 0 the castle, 1 the first maze level
	Row   int `json:"row"`   // R: 0 at the north edge
	Col   int `json:"col"`   // C: 0 at the west edge
}

// CastlePlace is L0.
var CastlePlace = Place{}

// IsCastle reports whether p is the castle.
func (p Place) IsCastle() bool { return p.Level == 0 }

func (p Place) String() string {
	if p.IsCastle() {
		return "L0"
	}
	return fmt.Sprintf("L%d:R%d:C%d", p.Level, p.Row, p.Col)
}

// PlaceOf converts an internal position (0-based level, X east, Y north) to
// L:R:C. It reports false for a position that is not a square of a maze
// level (a level below the last, as the elevator's H and I buttons give,
// finding #31): such a position has no canonical form and is never clamped.
func PlaceOf(levels, level, x, y int) (Place, bool) {
	if level < 0 || level >= levels || x < 0 || x >= MazeSize || y < 0 || y >= MazeSize {
		return Place{}, false
	}
	return Place{Level: level + 1, Row: MazeSize - 1 - y, Col: x}, true
}

// Internal converts a maze square back to the engine's level, X and Y.
func (p Place) Internal() (level, x, y int) {
	return p.Level - 1, p.Col, MazeSize - 1 - p.Row
}

// Valid reports whether p is the castle or a square of one of levels maze
// levels.
func (p Place) Valid(levels int) bool {
	if p.IsCastle() {
		return p.Row == 0 && p.Col == 0
	}
	return p.Level >= 1 && p.Level <= levels && p.Row >= 0 && p.Row < MazeSize && p.Col >= 0 && p.Col < MazeSize
}

// Place is where the party stands in L:R:C, or false when it is not on a
// square of the maze.
func (g *GameState) Place() (Place, bool) {
	return PlaceOf(len(g.Scenario.Mazes.Levels), g.MazeLevel, g.PlayerX, g.PlayerY)
}

// Letter is the facing as N, E, S or W.
func (d Direction) Letter() string { return [...]string{"N", "E", "S", "W"}[d&3] }

// Step returns the square one step from p in direction d on the same level,
// wrapping at the edges as movement does.
func (p Place) Step(d Direction) Place {
	switch d & 3 {
	case North:
		p.Row--
	case South:
		p.Row++
	case East:
		p.Col++
	case West:
		p.Col--
	}
	p.Row = (p.Row%MazeSize + MazeSize) % MazeSize
	p.Col = (p.Col%MazeSize + MazeSize) % MazeSize
	return p
}

// Entrance is where a party entering the maze from the castle stands, and
// its facing: L1:R19:C0 facing north, at the foot of the stairs up to the
// castle (Player's Guide p.44, p.55).
var (
	Entrance       = Place{Level: 1, Row: MazeSize - 1, Col: 0}
	EntranceFacing = North
)

// MazeEntrance finds the entrance in the maze data: the one square of the
// first level whose stairs lead to the castle. It is how Entrance is known
// to be right.
func MazeEntrance(m *data.MazeData) (Place, error) {
	if m == nil || len(m.Levels) == 0 {
		return Place{}, errors.New("no maze levels")
	}
	var found []Place
	cells := m.Levels[0].Cells
	for y := range cells {
		for x := range cells[y] {
			if c := cells[y][x]; c.Type == data.SqStairs && c.DestLevel == 0 {
				p, _ := PlaceOf(len(m.Levels), 0, x, y)
				found = append(found, p)
			}
		}
	}
	if len(found) != 1 {
		return Place{}, fmt.Errorf("%d squares on the first level have stairs to the castle", len(found))
	}
	return found[0], nil
}
