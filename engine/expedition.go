package engine

import (
	"encoding/json"
	"fmt"

	"github.com/Senior-Geek/istary/data"
)

// An expedition in progress, as saved in the roster file's optional
// "expedition" section (M6b-2; docs/design/m6b/EXPEDITION-PERSISTENCE.md).
// The section is absent when no expedition is under way, so ordinary saves
// are byte-identical to the format before it.
//
// The engine only stores and returns the section: front ends that resume
// expeditions (the GUI, through app.Controller) build and apply it. A front
// end that does not (the terminal) keeps it as loaded and writes it back, so
// it never drops another front end's expedition.

// ExpeditionVersion is the expedition section version this build reads and
// writes. A section of another version is kept and written back unchanged.
const ExpeditionVersion = 1

// Expedition is the saved expedition. Characters are referred to by name:
// their one authoritative record is the roster entry. The members are not
// in the castle party list ("party" in the file).
type Expedition struct {
	Version int        `json:"version"`
	Party   []string   `json:"party"` // the party in the maze, formation order
	Where   Checkpoint `json:"where"` // the last safe point
	// Interrupted is "combat" or "chest" while an encounter is unresolved;
	// Where then holds the game as it stands without the encounter.
	Interrupted string `json:"interrupted,omitempty"`
	// Squares are the special squares that differ from the scenario as
	// loaded (used one-time messages, items and encounters).
	Squares []SquareState `json:"squares,omitempty"`
}

// Checkpoint is the settled maze state at a safe point: everything a step
// leaves behind, so that restoring it replays nothing and draws no random
// number. The map overlay and delay entry (terminal only) are not kept.
type Checkpoint struct {
	Phase   string    `json:"phase"` // "maze" or "camp"
	Level   int       `json:"level"` // 0-based (0 = B1F)
	X       int       `json:"x"`
	Y       int       `json:"y"`
	Facing  Direction `json:"facing"`
	Light   int       `json:"light"`
	Protect int       `json:"protect"`
	Fizzles int       `json:"fizzles"`

	QuickPlot  bool     `json:"quickPlot,omitempty"`
	ChestAlarm int      `json:"chestAlarm,omitempty"`
	FightMap   []string `json:"fightMap"` // 20 strings of '0'/'1', indexed [x][y]

	// What the maze screen shows, and any open prompt.
	Message       string   `json:"message,omitempty"`
	Message2      string   `json:"message2,omitempty"`
	Viewport      string   `json:"viewport,omitempty"`
	Viewport2     string   `json:"viewport2,omitempty"`
	Messages      []string `json:"messages,omitempty"`
	MsgScroll     int      `json:"msgScroll,omitempty"`
	MsgWait       bool     `json:"msgWait,omitempty"`
	SearchYN      bool     `json:"searchYN,omitempty"`
	SearchHere    bool     `json:"searchHere,omitempty"` // the search is for the party's square
	SearchMonster int      `json:"searchMonster,omitempty"`
	Back2Shop     bool     `json:"back2Shop,omitempty"`
	TryGet        bool     `json:"tryGet,omitempty"`
	TryGetItem    int      `json:"tryGetItem,omitempty"`
	WaitButton    bool     `json:"waitButton,omitempty"`
	ButtonCount   int      `json:"buttonCount,omitempty"`
	Inspecting    bool     `json:"inspecting,omitempty"`
	InspectFound  []string `json:"inspectFound,omitempty"` // names
	CampMessage   string   `json:"campMessage,omitempty"`
}

// SquareState is the changeable part of one maze square.
type SquareState struct {
	Level       int             `json:"level"`
	X           int             `json:"x"`
	Y           int             `json:"y"`
	Type        data.SquareType `json:"type"`
	Count       int             `json:"count"`
	SpclMonster int             `json:"spclMonster"`
}

// expeditionHeader reads the version and members of any section version.
type expeditionHeader struct {
	Version int      `json:"version"`
	Party   []string `json:"party"`
}

// loadExpedition sets the game's expedition from a saved section.
func (g *GameState) loadExpedition(raw json.RawMessage) error {
	g.Expedition, g.OtherExpedition = nil, nil
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var h expeditionHeader
	if err := json.Unmarshal(raw, &h); err != nil {
		return fmt.Errorf("expedition: %w", err)
	}
	if h.Version != ExpeditionVersion {
		g.OtherExpedition = append(json.RawMessage(nil), raw...)
		return nil
	}
	var e Expedition
	if err := json.Unmarshal(raw, &e); err != nil {
		return fmt.Errorf("expedition: %w", err)
	}
	g.Expedition = &e
	return nil
}

// expeditionSection returns the section to save, or nil.
func (g *GameState) expeditionSection() (json.RawMessage, error) {
	if g.Expedition != nil {
		return json.Marshal(g.Expedition)
	}
	return g.OtherExpedition, nil
}

// ExpeditionMembers returns the names of the characters out on the saved
// expedition (of any version), or nil.
func (g *GameState) ExpeditionMembers() []string {
	if g.Expedition != nil {
		return g.Expedition.Party
	}
	if g.OtherExpedition != nil {
		var h expeditionHeader
		if json.Unmarshal(g.OtherExpedition, &h) == nil {
			return h.Party
		}
	}
	return nil
}

// OnExpedition reports whether the character named name is out on the
// saved expedition.
func (g *GameState) OnExpedition(name string) bool {
	for _, n := range g.ExpeditionMembers() {
		if n == name {
			return true
		}
	}
	return false
}

// ChangedSquares lists the squares of scenario sc whose changeable state
// (type, count, special monster) differs from pristine, the same scenario as
// loaded.
func ChangedSquares(pristine, sc *data.Scenario) []SquareState {
	var out []SquareState
	for l := range sc.Mazes.Levels {
		if l >= len(pristine.Mazes.Levels) {
			break
		}
		cur, orig := &sc.Mazes.Levels[l], &pristine.Mazes.Levels[l]
		for y := range cur.Cells {
			for x := range cur.Cells[y] {
				c, o := &cur.Cells[y][x], &orig.Cells[y][x]
				if c.Type != o.Type || c.Count != o.Count || c.SpclMonster != o.SpclMonster {
					out = append(out, SquareState{Level: l, X: x, Y: y, Type: c.Type, Count: c.Count, SpclMonster: c.SpclMonster})
				}
			}
		}
	}
	return out
}

// RestoreSquares sets the changeable state of every square of sc to
// pristine's, then applies changes. It reports a change outside the maze.
func RestoreSquares(pristine, sc *data.Scenario, changes []SquareState) error {
	for _, s := range changes {
		if s.Level < 0 || s.Level >= len(sc.Mazes.Levels) || s.Y < 0 || s.Y >= len(sc.Mazes.Levels[s.Level].Cells) ||
			s.X < 0 || s.X >= len(sc.Mazes.Levels[s.Level].Cells[s.Y]) {
			return fmt.Errorf("square L%d (%d,%d) is outside the maze", s.Level+1, s.X, s.Y)
		}
	}
	for l := range sc.Mazes.Levels {
		cur, orig := &sc.Mazes.Levels[l], &pristine.Mazes.Levels[l]
		for y := range cur.Cells {
			for x := range cur.Cells[y] {
				c, o := &cur.Cells[y][x], &orig.Cells[y][x]
				c.Type, c.Count, c.SpclMonster = o.Type, o.Count, o.SpclMonster
			}
		}
	}
	for _, s := range changes {
		c := &sc.Mazes.Levels[s.Level].Cells[s.Y][s.X]
		c.Type, c.Count, c.SpclMonster = s.Type, s.Count, s.SpclMonster
	}
	return nil
}
