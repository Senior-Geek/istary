package engine

// Exploration (M-1): what the player has legitimately seen of the maze, for
// the GUI's minimap and automap. It is kept per scenario and per level, in
// L:R:C (place.go), and saved in the roster file as an optional
// "exploration" section, beside the expedition. It belongs to the player,
// not to a party: changing the party keeps it.
//
// It records only what the party saw (Sight) or stood on, each wall as seen
// from the side the party was on: the maze has one-way walls and doors (a
// door on one side, a wall on the other), so the two sides of a boundary are
// kept apart and never assumed equal. A later sighting of the same side
// replaces the earlier one (a hidden door seen as a wall in the dark is a
// door once seen with light). Nothing is ever read from the maze data that
// the party has not seen: a square's feature (stairs, the elevator) is
// recorded only when the party stands on it.
//
// The terminal never records: a terminal save has no section, and a section
// it loads is written back unchanged.

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Senior-Geek/istary/data"
)

// ExplorationVersion is the version of the exploration section written.
const ExplorationVersion = 1

// Features of a square, recorded when the party stands on it. Teleporters,
// chutes, special squares and rescue markers may join them later; the
// section stores any kind as a string.
const (
	FeatureStairsUp   = "stairs-up"   // stairs to the level above, or to the castle
	FeatureStairsDown = "stairs-down" // stairs to the level below
	FeatureElevator   = "elevator"    // the buttons that move the party between levels
)

// Exploration is what has been seen, by level (L, from 1).
type Exploration struct {
	Dungeon string // the scenario, as the roster file is named ("1")
	Levels  map[int]*LevelMap
}

// LevelMap is what has been seen of one level, indexed [R][C].
type LevelMap struct {
	Visited  [MazeSize][MazeSize]bool
	Walls    [MazeSize][MazeSize][4]string // by Direction; "" unknown, else SeenOpen, SeenWall or SeenDoor
	Features [MazeSize][MazeSize]string
}

// Level returns the map of level L, or nil if nothing of it has been seen.
func (e *Exploration) Level(l int) *LevelMap {
	if e == nil {
		return nil
	}
	return e.Levels[l]
}

// LevelsSeen lists the levels with anything seen, in order.
func (e *Exploration) LevelsSeen() []int {
	if e == nil {
		return nil
	}
	var out []int
	for l := range e.Levels {
		out = append(out, l)
	}
	slices.Sort(out)
	return out
}

// Clone returns a copy that shares nothing with e.
func (e *Exploration) Clone() *Exploration {
	if e == nil {
		return nil
	}
	c := &Exploration{Dungeon: e.Dungeon, Levels: map[int]*LevelMap{}}
	for l, m := range e.Levels {
		cp := *m
		c.Levels[l] = &cp
	}
	return c
}

// Explore records what the party sees and stands on now: the square it
// stands on (visited, with its feature) in the maze, at Camp or in an
// encounter, and in the maze what it sees ahead (Sight). It reports
// whether anything new was recorded. Off the maze, on a square that is
// not part of it (finding #31), or when the save holds an exploration
// section this version cannot read, it records nothing.
func (g *GameState) Explore() bool {
	if g.OtherExploration != nil {
		return false
	}
	switch g.Phase {
	case PhaseMaze, PhaseCamp, PhaseCombat:
	default:
		return false
	}
	here, ok := g.Place()
	if !ok {
		return false
	}
	if g.Exploration == nil {
		g.Exploration = &Exploration{Dungeon: scenarioKey(g.Scenario.Game), Levels: map[int]*LevelMap{}}
	}
	m := g.Exploration.Levels[here.Level]
	if m == nil {
		m = &LevelMap{}
		g.Exploration.Levels[here.Level] = m
	}
	changed := false
	set := func(p Place, side Direction, seen string) {
		if seen != "" && m.Walls[p.Row][p.Col][side] != seen {
			m.Walls[p.Row][p.Col][side] = seen
			changed = true
		}
	}
	if !m.Visited[here.Row][here.Col] {
		m.Visited[here.Row][here.Col] = true
		changed = true
	}
	if f := featureOf(g.CurrentCell(), here.Level); f != "" && m.Features[here.Row][here.Col] != f {
		m.Features[here.Row][here.Col] = f
		changed = true
	}
	if g.Phase != PhaseMaze {
		return changed
	}
	f := g.Facing
	sight, _ := g.Sight()
	levels := len(g.Scenario.Mazes.Levels)
	for _, sq := range sight {
		p, _ := PlaceOf(levels, g.MazeLevel, sq.X, sq.Y)
		set(p, f.TurnLeft(), sq.Left)
		set(p, f.TurnRight(), sq.Right)
		set(p, f, sq.Front)
		if sq.LeftAhead != "" {
			lx, ly := Shift(sq.X, sq.Y, f, -1, 0)
			lp, _ := PlaceOf(levels, g.MazeLevel, lx, ly)
			set(lp, f, sq.LeftAhead)
		}
		if sq.RightAhead != "" {
			rx, ry := Shift(sq.X, sq.Y, f, 1, 0)
			rp, _ := PlaceOf(levels, g.MazeLevel, rx, ry)
			set(rp, f, sq.RightAhead)
		}
	}
	return changed
}

// featureOf is the feature of a square on level l, as the party standing on
// it learns it: stairs up (to the level above or the castle) or down, as the
// stairs' own message says, and the elevator's buttons.
func featureOf(cell *data.MazeCell, l int) string {
	if cell == nil {
		return ""
	}
	switch cell.Type {
	case data.SqStairs:
		if cell.DestLevel < l {
			return FeatureStairsUp
		}
		return FeatureStairsDown
	case data.SqButtons:
		return FeatureElevator
	}
	return ""
}

// The saved section. Rows run from R0; each square of a row is one
// character of "visited" ('v' or '.') and four of "walls" (N, E, S, W:
// 'o' open, 'w' wall, 'd' door, '.' unknown).
type explorationSection struct {
	Version int            `json:"version"`
	Dungeon string         `json:"dungeon"`
	Levels  []levelSection `json:"levels"`
}

type levelSection struct {
	Level    int              `json:"level"`
	Visited  []string         `json:"visited"`
	Walls    []string         `json:"walls"`
	Features []featureSection `json:"features,omitempty"`
}

type featureSection struct {
	Row  int    `json:"row"`
	Col  int    `json:"col"`
	Kind string `json:"kind"`
}

var wallCodes = map[string]byte{"": '.', SeenOpen: 'o', SeenWall: 'w', SeenDoor: 'd'}

// explorationSection returns the section to save, or nil when there is
// none. A section this version could not read is written back as it was.
func (g *GameState) explorationSection() (json.RawMessage, error) {
	if g.OtherExploration != nil {
		return g.OtherExploration, nil
	}
	e := g.Exploration
	if e == nil || len(e.Levels) == 0 {
		return nil, nil
	}
	s := explorationSection{Version: ExplorationVersion, Dungeon: e.Dungeon}
	for _, l := range e.LevelsSeen() {
		m := e.Levels[l]
		ls := levelSection{Level: l}
		for r := 0; r < MazeSize; r++ {
			var v, w strings.Builder
			for c := 0; c < MazeSize; c++ {
				if m.Visited[r][c] {
					v.WriteByte('v')
				} else {
					v.WriteByte('.')
				}
				for _, side := range m.Walls[r][c] {
					w.WriteByte(wallCodes[side])
				}
				if k := m.Features[r][c]; k != "" {
					ls.Features = append(ls.Features, featureSection{Row: r, Col: c, Kind: k})
				}
			}
			ls.Visited = append(ls.Visited, v.String())
			ls.Walls = append(ls.Walls, w.String())
		}
		s.Levels = append(s.Levels, ls)
	}
	return json.Marshal(s)
}

// loadExploration sets the game's exploration from a saved section. A
// section of another version, for another scenario, or that cannot be
// read is kept as it is (OtherExploration) and written back unchanged;
// the map then records nothing, rather than lose it.
func (g *GameState) loadExploration(raw json.RawMessage) {
	g.Exploration, g.OtherExploration = nil, nil
	if len(raw) == 0 || string(raw) == "null" {
		return
	}
	e, err := parseExploration(raw, len(g.Scenario.Mazes.Levels), scenarioKey(g.Scenario.Game))
	if err != nil {
		g.OtherExploration = append(json.RawMessage(nil), raw...)
		return
	}
	g.Exploration = e
}

func parseExploration(raw json.RawMessage, levels int, dungeon string) (*Exploration, error) {
	var s explorationSection
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	if s.Version != ExplorationVersion {
		return nil, fmt.Errorf("version %d", s.Version)
	}
	if s.Dungeon != dungeon {
		return nil, fmt.Errorf("dungeon %q", s.Dungeon)
	}
	codes := map[byte]string{'.': "", 'o': SeenOpen, 'w': SeenWall, 'd': SeenDoor}
	e := &Exploration{Dungeon: s.Dungeon, Levels: map[int]*LevelMap{}}
	for _, ls := range s.Levels {
		if ls.Level < 1 || ls.Level > levels || e.Levels[ls.Level] != nil {
			return nil, fmt.Errorf("level %d", ls.Level)
		}
		if len(ls.Visited) != MazeSize || len(ls.Walls) != MazeSize {
			return nil, fmt.Errorf("level %d: %d rows", ls.Level, len(ls.Visited))
		}
		m := &LevelMap{}
		for r := 0; r < MazeSize; r++ {
			v, w := ls.Visited[r], ls.Walls[r]
			if len(v) != MazeSize || len(w) != 4*MazeSize {
				return nil, fmt.Errorf("level %d row %d", ls.Level, r)
			}
			for c := 0; c < MazeSize; c++ {
				switch v[c] {
				case 'v':
					m.Visited[r][c] = true
				case '.':
				default:
					return nil, fmt.Errorf("level %d row %d", ls.Level, r)
				}
				for side := 0; side < 4; side++ {
					k, ok := codes[w[4*c+side]]
					if !ok {
						return nil, fmt.Errorf("level %d row %d", ls.Level, r)
					}
					m.Walls[r][c][side] = k
				}
			}
		}
		for _, f := range ls.Features {
			if f.Row < 0 || f.Row >= MazeSize || f.Col < 0 || f.Col >= MazeSize || f.Kind == "" {
				return nil, fmt.Errorf("level %d feature", ls.Level)
			}
			m.Features[f.Row][f.Col] = f.Kind
		}
		e.Levels[ls.Level] = m
	}
	return e, nil
}
