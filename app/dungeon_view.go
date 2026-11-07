package app

// DungeonView is what the player sees in the maze and at Camp (M6a), for
// any front end: the corridor ahead as far as the Classic view draws it, the
// party, the messages, what the maze is waiting for, and the actions that
// are open. It is read-only and JSON-ready.
//
// It shows no more than the Classic maze screen: no position, facing or
// floor (only DUMAPIC tells those), no map, nothing beyond the light, and no
// encounter state. A hidden door looks like a wall unless the party has
// light; the Classic renderer also lets one flicker into view one draw in
// six without light, which needs a random number, so the view, which must
// never draw from the game's random numbers, leaves it out.

import (
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

type DungeonView struct {
	Phase    string              `json:"phase"` // "camp", "maze", "combat", "town" or "other"
	Party    []DungeonMemberView `json:"party"`
	Corridor []CorridorView      `json:"corridor"` // nearest first; empty in the dark
	Dark     bool                `json:"dark"`     // the view ends in a dark square
	Viewport []string            `json:"viewport"` // text in the view: "OUCH!", "IT'S VERY" / "DARK HERE"
	Lines    []string            `json:"lines"`    // the message area
	More     bool                `json:"more"`     // a long message has more pages ([RET] FOR MORE)
	Prompt   string              `json:"prompt"`   // what the maze waits for: "", "messages", "stairs", "search", "buttons", "inspect", "map", "delay"
	Buttons  []string            `json:"buttons"`  // the buttons to press ("A", "B", ...)
	Found    []string            `json:"found"`    // Inspect: characters found here
	Spells   []string            `json:"spells"`   // LIGHT and PROTECT, as on the maze screen
	Actions  []string            `json:"actions"`  // operations open now (the Act names)
}

// DungeonMemberView is one party member as the maze screen lists them.
type DungeonMemberView struct {
	Slot     int    `json:"slot"` // from 1; 1-3 are the front row
	Name     string `json:"name"`
	Class    string `json:"class"` // alignment and class ("G-FIG")
	AC       int    `json:"ac"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	Status   string `json:"status"`
	Poisoned bool   `json:"poisoned"`
}

// CorridorView is one square of the corridor ahead: its walls to the left,
// right and front ("open", "wall" or "door"). Where a side is open, the
// wall ahead of the square on that side is given, as the Classic view
// draws it.
type CorridorView struct {
	Left       string `json:"left"`
	Right      string `json:"right"`
	Front      string `json:"front"`
	LeftAhead  string `json:"leftAhead,omitempty"`
	RightAhead string `json:"rightAhead,omitempty"`
}

// Action names in DungeonView.Actions, each one operation.
const (
	ActForward   = "forward"    // Forward
	ActLeft      = "left"       // TurnLeft
	ActRight     = "right"      // TurnRight
	ActKick      = "kick"       // Kick
	ActCamp      = "camp"       // Camp
	ActInspect   = "inspect"    // Inspect
	ActContinue  = "continue"   // ContinueMessage
	ActYes       = "yes"        // Answer(true)
	ActNo        = "no"         // Answer(false)
	ActButton    = "button"     // PressButton
	ActPickUp    = "pick-up"    // PickUp
	ActLeave     = "leave"      // LeaveInspect
	ActLeaveCamp = "leave-camp" // LeaveCamp
)

// DungeonView returns the view of the maze or Camp. It reads the game and
// changes nothing.
func (c *Controller) DungeonView() DungeonView {
	g := c.Game
	v := DungeonView{Phase: phaseName(g.Phase), Party: c.dungeonParty()}
	switch g.Phase {
	case engine.PhaseCamp:
		if g.Town.Message != "" {
			v.Lines = []string{g.Town.Message}
		}
		if c.campAtMenu() {
			v.Actions = []string{ActLeaveCamp}
		}
		return v
	case engine.PhaseMaze:
	default:
		return v
	}

	v.Prompt = c.mazePrompt()
	v.Corridor, v.Dark = c.corridor()
	for _, s := range []string{g.ViewportMsg, g.ViewportMsg2} {
		if s != "" {
			v.Viewport = append(v.Viewport, s)
		}
	}
	// The message area, as the maze screen draws it.
	switch {
	case g.MazeDelayInput:
		v.Lines = []string{g.MazeMessage + g.MazeDelayBuf}
	case len(g.MazeMessages) > 0:
		for i := 0; i < 4 && g.MazeMsgScroll+i < len(g.MazeMessages); i++ {
			v.Lines = append(v.Lines, g.MazeMessages[g.MazeMsgScroll+i])
		}
		v.More = g.MazeMsgWait
	default:
		for _, s := range []string{g.MazeMessage, g.MazeMessage2} {
			if s != "" {
				v.Lines = append(v.Lines, s)
			}
		}
	}
	if g.LightLevel > 0 {
		v.Spells = append(v.Spells, "LIGHT")
	}
	if g.ProtectLevel > 0 {
		v.Spells = append(v.Spells, "PROTECT")
	}
	switch v.Prompt {
	case "":
		v.Actions = []string{ActForward, ActLeft, ActRight, ActKick, ActCamp, ActInspect}
	case "messages":
		v.Actions = []string{ActContinue}
	case "stairs", "search":
		v.Actions = []string{ActYes, ActNo}
	case "buttons":
		v.Actions = []string{ActButton}
		for i := 0; i < g.ButtonCount; i++ {
			v.Buttons = append(v.Buttons, string(rune('A'+i)))
		}
	case "inspect":
		for _, ch := range g.MazeInspectFound {
			v.Found = append(v.Found, ch.Name)
		}
		if len(v.Found) > 0 {
			v.Actions = append(v.Actions, ActPickUp)
		}
		v.Actions = append(v.Actions, ActLeave)
	}
	return v
}

func phaseName(p engine.Phase) string {
	switch p {
	case engine.PhaseCamp:
		return "camp"
	case engine.PhaseMaze:
		return "maze"
	case engine.PhaseCombat:
		return "combat"
	case engine.PhaseTown:
		return "town"
	}
	return "other"
}

func (c *Controller) dungeonParty() []DungeonMemberView {
	var out []DungeonMemberView
	for i, m := range c.Game.Town.Party.Members {
		if m == nil {
			continue
		}
		out = append(out, DungeonMemberView{Slot: i + 1, Name: m.Name, Class: engine.AlignClass(m.Alignment, m.Class),
			AC: m.AC, HP: m.HP, MaxHP: m.MaxHP, Status: m.Status.String(), Poisoned: m.PoisonAmt > 0})
	}
	return out
}

// corridor walks ahead as the Classic view draws (Pascal DRAWMAZE): five
// squares with light (three with QUICK plot), two without, stopping at a
// dark square or the first wall or door in front.
func (c *Controller) corridor() ([]CorridorView, bool) {
	g := c.Game
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
			return "open"
		case data.WallDoor:
			return "door"
		case data.WallHidden:
			if light {
				return "door"
			}
		}
		return "wall"
	}
	var out []CorridorView
	x, y := g.PlayerX, g.PlayerY
	for ; depth > 0; depth-- {
		if level.Cells[y][x].Type == data.SqDark {
			return out, true
		}
		sq := CorridorView{
			Left:  shown(wallAt(level, x, y, g.Facing, 0, g.Facing.TurnLeft())),
			Right: shown(wallAt(level, x, y, g.Facing, 0, g.Facing.TurnRight())),
			Front: shown(wallAt(level, x, y, g.Facing, 0, g.Facing)),
		}
		if sq.Left == "open" {
			sq.LeftAhead = shown(wallAt(level, x, y, g.Facing, -1, g.Facing))
		}
		if sq.Right == "open" {
			sq.RightAhead = shown(wallAt(level, x, y, g.Facing, 1, g.Facing))
		}
		out = append(out, sq)
		if sq.Front != "open" {
			break
		}
		x, y = shift(x, y, g.Facing, 0, 1)
	}
	return out, false
}

// shift moves right and forward relative to facing, wrapping at the edge
// (Pascal SHFTPOS; north is Y+1).
func shift(x, y int, facing engine.Direction, right, forward int) (int, int) {
	switch facing {
	case engine.North:
		x, y = x+right, y+forward
	case engine.East:
		x, y = x+forward, y-right
	case engine.South:
		x, y = x-right, y-forward
	case engine.West:
		x, y = x-forward, y+right
	}
	return (x%20 + 20) % 20, (y%20 + 20) % 20
}

// wallAt is the wall on side dir of the square right squares to the right.
func wallAt(level *data.MazeLevel, x, y int, facing engine.Direction, right int, dir engine.Direction) data.WallType {
	x, y = shift(x, y, facing, right, 0)
	cell := level.Cells[y][x]
	switch dir {
	case engine.North:
		return cell.N
	case engine.East:
		return cell.E
	case engine.South:
		return cell.S
	case engine.West:
		return cell.W
	}
	return data.WallWall
}
