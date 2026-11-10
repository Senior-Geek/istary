package app

// Map views (M-1): the explored maze as the GUI's minimap and automap draw
// it, prepared from the engine's exploration record (engine/explore.go) in
// L:R:C. Front ends draw what these say and decide nothing: which walls to
// draw, where a one-way wall lies, which connections cross the level's
// edge, where the party stands and which way it faces.
//
// The maps are an enhancement (decision D9): DungeonView, the Classic maze
// screen, still shows no position; these views are separate.
//
// Each wall is recorded as seen from one side (a boundary can be a door on
// one side and a wall on the other). A boundary seen from one side only, or
// alike from both, is one wall on the boundary; a boundary seen differently
// from its two sides is drawn as each side's own wall, set just inside its
// square (OneWay), so the map never shows a way through that is not there.

import (
	"errors"

	"github.com/Senior-Geek/istary/engine"
)

// MapWall is a wall or door to draw: the square (row and column, in the
// map's own frame), the side of it, and whether it is one side's wall only
// (OneWay: drawn just inside the square). In Wraps, Kind may be "open":
// a connection across the level's edge.
type MapWall struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Side   string `json:"side"` // N, E, S or W
	Kind   string `json:"kind"` // wall or door (open in Wraps)
	OneWay bool   `json:"oneWay,omitempty"`
}

// MapFeature is a feature of a square seen: stairs up or down, the elevator.
type MapFeature struct {
	Row  int    `json:"row"`
	Col  int    `json:"col"`
	Kind string `json:"kind"`
}

// MapParty is the party on a map: its square in the map's frame, its
// facing, and where that is in L:R:C.
type MapParty struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Facing string `json:"facing"` // N, E, S or W
	Place  string `json:"place"`  // L:R:C
}

// MapView is one whole level as explored (the automap): a fixed 20x20
// frame, R0 at the top and C0 at the left, never moved or repeated.
type MapView struct {
	Level    int          `json:"level"`  // L
	Levels   []int        `json:"levels"` // the levels with anything seen
	Size     int          `json:"size"`
	Walls    []MapWall    `json:"walls"`
	Wraps    []MapWall    `json:"wraps"` // seen ways across the level's edge, at the square they leave from
	Features []MapFeature `json:"features"`
	Party    *MapParty    `json:"party"` // nil when the party is not on this level
}

// LocalMapView is the squares around the party (the minimap), north up: a
// window Size squares wide and high with the party at its centre. It
// continues across the level's edge as the maze does; NorthEdge and
// WestEdge, when not -1, are the window row and column whose top or left
// side is the level's edge (where R0 or C0 begins).
type LocalMapView struct {
	Level     int          `json:"level"`
	Size      int          `json:"size"`
	Walls     []MapWall    `json:"walls"`
	Features  []MapFeature `json:"features"`
	Party     MapParty     `json:"party"`
	NorthEdge int          `json:"northEdge"`
	WestEdge  int          `json:"westEdge"`
}

// LocalRadius is how many squares the minimap shows on each side of the
// party.
const LocalRadius = 3

// ErrNoMap is returned when there is no explored level to show.
var ErrNoMap = errors.New("nothing of the maze has been explored")

var sideNames = [4]string{"N", "E", "S", "W"}

// boundary decides how the boundary on side d of square p is drawn, given
// what was seen from each side: the walls to draw, each on the square it
// belongs to.
func boundary(m *engine.LevelMap, p engine.Place, d engine.Direction) []MapWall {
	q := p.Step(d)
	mine, theirs := m.Walls[p.Row][p.Col][d], m.Walls[q.Row][q.Col][d.Reverse()]
	solid := func(k string) bool { return k == engine.SeenWall || k == engine.SeenDoor }
	switch {
	case mine == "" && theirs == "":
		return nil
	case mine == theirs, theirs == "":
		if solid(mine) {
			return []MapWall{{Row: p.Row, Col: p.Col, Side: sideNames[d], Kind: mine}}
		}
		return nil
	case mine == "":
		if solid(theirs) {
			return []MapWall{{Row: q.Row, Col: q.Col, Side: sideNames[d.Reverse()], Kind: theirs}}
		}
		return nil
	}
	var out []MapWall
	if solid(mine) {
		out = append(out, MapWall{Row: p.Row, Col: p.Col, Side: sideNames[d], Kind: mine, OneWay: true})
	}
	if solid(theirs) {
		out = append(out, MapWall{Row: q.Row, Col: q.Col, Side: sideNames[d.Reverse()], Kind: theirs, OneWay: true})
	}
	return out
}

// crosses reports whether side d of square p is on the level's edge.
func crosses(p engine.Place, d engine.Direction) bool {
	switch d {
	case engine.North:
		return p.Row == 0
	case engine.South:
		return p.Row == engine.MazeSize-1
	case engine.West:
		return p.Col == 0
	}
	return p.Col == engine.MazeSize-1
}

// Map returns explored level l (L, from 1) as the automap draws it; 0 is
// the party's level, or else the first level explored. It reads the game
// and changes nothing.
func (c *Controller) Map(l int) (MapView, error) {
	g := c.Game
	e := g.Exploration
	here, onMaze := partyPlace(g)
	if l == 0 {
		switch {
		case onMaze && e.Level(here.Level) != nil:
			l = here.Level
		case len(e.LevelsSeen()) > 0:
			l = e.LevelsSeen()[0]
		}
	}
	m := e.Level(l)
	if m == nil {
		return MapView{}, ErrNoMap
	}
	v := MapView{Level: l, Levels: e.LevelsSeen(), Size: engine.MazeSize, Walls: []MapWall{}, Wraps: []MapWall{}, Features: []MapFeature{}}
	for r := 0; r < engine.MazeSize; r++ {
		for col := 0; col < engine.MazeSize; col++ {
			p := engine.Place{Level: l, Row: r, Col: col}
			if k := m.Features[r][col]; k != "" {
				v.Features = append(v.Features, MapFeature{Row: r, Col: col, Kind: k})
			}
			for d := engine.North; d <= engine.West; d++ {
				if crosses(p, d) {
					// Across the edge the two sides are drawn apart, each at
					// its own square; a way through is marked there.
					k := m.Walls[r][col][d]
					if k == "" {
						continue
					}
					q := p.Step(d)
					other := m.Walls[q.Row][q.Col][d.Reverse()]
					oneWay := other != "" && other != k
					if k != engine.SeenOpen {
						v.Walls = append(v.Walls, MapWall{Row: r, Col: col, Side: sideNames[d], Kind: k, OneWay: oneWay})
					}
					if k != engine.SeenWall {
						v.Wraps = append(v.Wraps, MapWall{Row: r, Col: col, Side: sideNames[d], Kind: k, OneWay: oneWay})
					}
					continue
				}
				// Each inner boundary once: from the square on its north or west.
				if d == engine.South || d == engine.East {
					v.Walls = append(v.Walls, boundary(m, p, d)...)
				}
			}
		}
	}
	if onMaze && here.Level == l {
		v.Party = &MapParty{Row: here.Row, Col: here.Col, Facing: g.Facing.Letter(), Place: here.String()}
	}
	return v, nil
}

// LocalMap returns the squares around the party as the minimap draws them,
// or false off the maze. It reads the game and changes nothing.
func (c *Controller) LocalMap() (LocalMapView, bool) {
	g := c.Game
	here, ok := partyPlace(g)
	if !ok {
		return LocalMapView{}, false
	}
	size := 2*LocalRadius + 1
	v := LocalMapView{Level: here.Level, Size: size, Walls: []MapWall{}, Features: []MapFeature{},
		Party:     MapParty{Row: LocalRadius, Col: LocalRadius, Facing: g.Facing.Letter(), Place: here.String()},
		NorthEdge: -1, WestEdge: -1}
	// at is the square at window row i and column j; window places by square.
	at := func(i, j int) engine.Place {
		p := here
		for ; i > 0; i-- {
			p = p.Step(engine.South)
		}
		for ; i < 0; i++ {
			p = p.Step(engine.North)
		}
		for ; j > 0; j-- {
			p = p.Step(engine.East)
		}
		for ; j < 0; j++ {
			p = p.Step(engine.West)
		}
		return p
	}
	where := map[engine.Place][2]int{}
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			p := at(i-LocalRadius, j-LocalRadius)
			where[p] = [2]int{i, j}
			if p.Row == 0 && i > 0 {
				v.NorthEdge = i
			}
			if p.Col == 0 && j > 0 {
				v.WestEdge = j
			}
		}
	}
	m := g.Exploration.Level(here.Level)
	if m == nil {
		return v, true
	}
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			p := at(i-LocalRadius, j-LocalRadius)
			if k := m.Features[p.Row][p.Col]; k != "" {
				v.Features = append(v.Features, MapFeature{Row: i, Col: j, Kind: k})
			}
			for d := engine.North; d <= engine.West; d++ {
				// Each boundary once: from the square on its north or west,
				// or from this one when that square is outside the window.
				if d == engine.North && i > 0 || d == engine.West && j > 0 {
					continue
				}
				for _, w := range boundary(m, p, d) {
					// Place each wall in the window by the square it belongs to.
					// One belonging to a square outside the window is drawn on
					// this square's side if it lies on the boundary, and not at
					// all if it lies inside that square (one way).
					mine := w.Row == p.Row && w.Col == p.Col
					switch {
					case mine:
						w.Row, w.Col = i, j
					case boundaryInWindow(i, j, d, size):
						pos := where[engine.Place{Level: here.Level, Row: w.Row, Col: w.Col}]
						w.Row, w.Col = pos[0], pos[1]
					case w.OneWay:
						continue
					default:
						w.Row, w.Col, w.Side = i, j, sideNames[d]
					}
					v.Walls = append(v.Walls, w)
				}
			}
		}
	}
	return v, true
}

// partyPlace is where the party stands in the maze (in it, at Camp or in
// an encounter), or false when it is not in the maze.
func partyPlace(g *engine.GameState) (engine.Place, bool) {
	switch g.Phase {
	case engine.PhaseMaze, engine.PhaseCamp, engine.PhaseCombat:
		return g.Place()
	}
	return engine.Place{}, false
}

// boundaryInWindow reports whether the square across side d of window
// square (i, j) is inside the window too.
func boundaryInWindow(i, j int, d engine.Direction, size int) bool {
	switch d {
	case engine.North:
		return i > 0
	case engine.South:
		return i < size-1
	case engine.West:
		return j > 0
	}
	return j < size-1
}

// Explore records what the party sees and stands on now into the
// exploration (engine.GameState.Explore), and reports whether anything was
// new. Front ends that show the maps call it after every action in the
// maze; the terminal does not.
func (c *Controller) Explore() bool { return c.Game.Explore() }
