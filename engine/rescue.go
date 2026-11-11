package engine

// Rescue (R-2): finding stranded characters and taking them into the
// party. The rules are the original's I)NSPECT and P)ICK UP, from the
// reconstructed Pascal source (SPECIALS: INSPECT, EXPLROOM, LOOKLOST,
// PICKUP; docs/design/r2/RESCUE.md), with the R-2 decisions:
//
//   - A stranded character is found anywhere in the party's area: the
//     squares the party could walk to through OPEN walls on its level
//     (RescueArea). The Player's Guide (p.42): "close enough to them so
//     that you could walk to where they are without going through a door".
//   - Characters with no Stranded record (left out by an older version) are
//     found, as before, only on the exact square their legacy fields name,
//     when those fields name a square of the maze. Nothing is made up.
//   - A character is taken into the party one at a time, only while there
//     is room (MaxPartySize members, the incapacitated included), and only
//     while still found from where the party stands. Nothing changes when a
//     character cannot be taken.
//   - Taking a character changes where they are, nothing else: condition,
//     HP, items, gold, experience, age and spells stay as they are.

import (
	"errors"

	"github.com/Senior-Geek/istary/data"
)

// MaxPartySize is the most characters a party can hold, the incapacitated
// included (Player's Guide p.39).
const MaxPartySize = 6

// Area is a set of squares of one maze level, indexed [x][y] in the
// engine's coordinates (X east, Y north).
type Area [MazeSize][MazeSize]bool

// RescueArea is the area around (x, y) on level (0-based) of sc: the
// squares reachable from it through OPEN walls only, as the original's
// EXPLROOM fills it. Each square's own wall on the side of the move
// decides, so a one-way passage is followed only the way it can be walked;
// a door, a hidden door or a wall stops the search; it wraps at the edges
// of the level (as the original takes X and Y modulo 20) and never leaves
// the level. Stairs, chutes, pits, teleporters and other special squares
// neither stop nor carry it: only walls matter. It reports false, with an
// empty area, when (x, y) is not a square of a maze level.
//
// The search is only for finding characters. It reveals nothing: the
// player's map (Exploration) is not touched.
func RescueArea(sc *data.Scenario, level, x, y int) (Area, bool) {
	var area Area
	if level < 0 || level >= len(sc.Mazes.Levels) || x < 0 || x >= MazeSize || y < 0 || y >= MazeSize {
		return area, false
	}
	cells := sc.Mazes.Levels[level].Cells
	wall := func(x, y int, side func(*data.MazeCell) data.WallType) data.WallType {
		if y >= len(cells) || x >= len(cells[y]) {
			return data.WallWall
		}
		return side(&cells[y][x])
	}
	type square struct{ x, y int }
	area[x][y] = true
	queue := []square{{x, y}}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		for _, step := range [...]struct {
			dx, dy int
			side   func(*data.MazeCell) data.WallType
		}{
			{1, 0, func(c *data.MazeCell) data.WallType { return c.E }},
			{-1, 0, func(c *data.MazeCell) data.WallType { return c.W }},
			{0, -1, func(c *data.MazeCell) data.WallType { return c.S }},
			{0, 1, func(c *data.MazeCell) data.WallType { return c.N }},
		} {
			if wall(s.x, s.y, step.side) != data.WallOpen {
				continue
			}
			nx, ny := (s.x+step.dx+MazeSize)%MazeSize, (s.y+step.dy+MazeSize)%MazeSize
			if !area[nx][ny] {
				area[nx][ny] = true
				queue = append(queue, square{nx, ny})
			}
		}
	}
	return area, true
}

// legacySquare is the square an older version's legacy fields (LOSTXYL)
// name for a character with no Stranded record, in the engine's
// coordinates, or false when they name no square of the maze.
func (g *GameState) legacySquare(c *Character) (level, x, y int, ok bool) {
	_, ok = PlaceOf(len(g.Scenario.Mazes.Levels), c.MazeLevel, c.MazeX, c.MazeY)
	return c.MazeLevel, c.MazeX, c.MazeY, ok
}

// FindStranded lists, in roster order, the characters the party can find
// from where it stands (I)NSPECT): stranded characters in its area, and
// characters an older version left out on its very square. Characters in
// the party, on a saved expedition, LOST, or not out of the castle are
// never found. It changes nothing.
func (g *GameState) FindStranded() []*Character {
	area, ok := RescueArea(g.Scenario, g.MazeLevel, g.PlayerX, g.PlayerY)
	if !ok {
		return nil
	}
	away := map[string]bool{} // out on a saved expedition: not lost here
	for _, n := range g.ExpeditionMembers() {
		away[n] = true
	}
	var found []*Character
	for _, c := range g.Town.Roster.Characters {
		if c == nil || c.Status == Lost || !c.InMaze || g.inParty(c) || away[c.Name] {
			continue
		}
		if c.Stranded != nil {
			level, x, y := c.Stranded.Internal()
			if c.Stranded.Valid(len(g.Scenario.Mazes.Levels)) && level == g.MazeLevel && area[x][y] {
				found = append(found, c)
			}
		} else if level, x, y, ok := g.legacySquare(c); ok && level == g.MazeLevel && x == g.PlayerX && y == g.PlayerY {
			found = append(found, c)
		}
	}
	return found
}

func (g *GameState) inParty(c *Character) bool {
	for _, m := range g.Town.Party.Members {
		if m == c {
			return true
		}
	}
	return false
}

// Reasons RecoverStranded refuses; nothing is changed.
var (
	ErrRescuePartyFull = errors.New("the party has six members")
	ErrNotFoundHere    = errors.New("not found from where the party stands")
)

// RecoverStranded takes c, found from where the party stands
// (FindStranded), into the party (P)ICK UP). It checks everything first and
// changes nothing when c cannot be taken: c must be found here and the
// party must have room. Then c joins the end of the party (any empty
// entries are closed up first) and is no longer stranded: Stranded and the
// legacy fields are cleared, as the original's PICKUP clears LOSTXYL, and
// InMaze stays set while the party is out. Nothing else about c changes.
func (g *GameState) RecoverStranded(c *Character) error {
	if c == nil {
		return ErrNotFoundHere
	}
	here := false
	for _, f := range g.FindStranded() {
		if f == c {
			here = true
		}
	}
	if !here {
		return ErrNotFoundHere
	}
	if g.Town.Party.Size() >= MaxPartySize {
		return ErrRescuePartyFull
	}
	var members []*Character
	for _, m := range g.Town.Party.Members {
		if m != nil {
			members = append(members, m)
		}
	}
	g.Town.Party.Members = append(members, c)
	c.Stranded = nil
	c.MazeLevel, c.MazeX, c.MazeY = 0, 0, 0
	c.InMaze = true
	return nil
}

// PartyHome marks every member of the party as back in the castle, at the
// end of a legitimate return (the stairs up, a square that sends the party
// home, Camp MALOR to the castle): no longer in the maze and stranded
// nowhere. Conditions, HP and possessions are untouched; nobody is healed
// or raised, and the party stays together, the incapacitated included
// (decision Q8; the original's CHK4WIN drops all but the OK from the party).
func (g *GameState) PartyHome() {
	for _, m := range g.Town.Party.Members {
		if m != nil {
			m.InMaze, m.Stranded = false, nil
		}
	}
}
