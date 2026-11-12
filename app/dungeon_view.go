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
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

type DungeonView struct {
	Phase    string              `json:"phase"` // "camp", "maze", "combat", "town" or "other"
	Party    []DungeonMemberView `json:"party"`
	Corridor []CorridorView      `json:"corridor"`          // nearest first; empty in the dark
	Dark     bool                `json:"dark"`              // the view ends in a dark square
	Viewport []string            `json:"viewport"`          // text in the view: "OUCH!", "IT'S VERY" / "DARK HERE"
	Lines    []string            `json:"lines"`             // the message area
	More     bool                `json:"more"`              // a long message has more pages ([RET] FOR MORE)
	Prompt   string              `json:"prompt"`            // what the maze waits for: "", "messages", "stairs", "search", "buttons", "inspect", "map", "delay"
	Buttons  []string            `json:"buttons"`           // the buttons to press ("A", "B", ...)
	Found    []string            `json:"found"`             // Inspect: names of the characters found in the party's area
	Inspect  *InspectView        `json:"inspect,omitempty"` // Inspect: who was found, and the room for them
	Spells   []string            `json:"spells"`            // LIGHT and PROTECT, as on the maze screen
	Actions  []string            `json:"actions"`           // operations open now (the Act names)
	Default  string              `json:"default,omitempty"` // the stairs' suggested answer, ActYes or ActNo (stairs.go); never taken by itself
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

// InspectView is the Inspect screen (R-2): each character found in the
// party's area, and how many more the party has room for. It says nothing
// of where they lie: the maze view never gives positions.
//
// The rest is the rescue's progress (R-3), all read from the game as it
// stands, so it is the same after a restart: who else is known to lie in
// the maze, and who in the party cannot act. Whether a member was picked
// up or started out with the party is recorded nowhere, so the view does
// not say; it says only how each member is.
type InspectView struct {
	Found     []FoundView     `json:"found"`
	Room      int             `json:"room"`      // more characters the party can take (6 - its size)
	Size      int             `json:"size"`      // the party's members, the incapacitated included
	Picked    bool            `json:"picked"`    // the message is a pickup's ("NAME JOINS THE PARTY.")
	Elsewhere int             `json:"elsewhere"` // known to lie in the maze, but not found from here
	Carried   []CharacterNote `json:"carried"`   // party members who cannot act, as they are
}

// FoundView is one character found by Inspect.
type FoundView struct {
	N      int    `json:"n"` // from 1: PickUp(N) takes this character
	Name   string `json:"name"`
	Class  string `json:"class"` // alignment and class ("G-FIG")
	Level  int    `json:"level"`
	Status string `json:"status"`
	// Condition is the status as Inspect names it (ConditionLabel).
	Condition string `json:"condition"`
	CanPick   bool   `json:"canPick"`
	Refusal   string `json:"refusal,omitempty"` // why not, when CanPick is false
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
		if v.Prompt == "stairs" {
			v.Default = c.stairsDefault()
		}
	case "buttons":
		v.Actions = []string{ActButton}
		for i := 0; i < g.ButtonCount; i++ {
			v.Buttons = append(v.Buttons, string(rune('A'+i)))
		}
	case "inspect":
		v.Inspect = c.inspectView()
		for i, ch := range g.MazeInspectFound {
			v.Found = append(v.Found, ch.Name)
			f := FoundView{N: i + 1, Name: ch.Name, Class: engine.AlignClass(ch.Alignment, ch.Class),
				Level: ch.Level, Status: ch.Status.String(), Condition: ConditionLabel(ch.Status), CanPick: v.Inspect.Room > 0}
			if !f.CanPick {
				f.Refusal = partyFullRefusal(ch).Message
			}
			v.Inspect.Found = append(v.Inspect.Found, f)
		}
		if len(v.Found) > 0 && v.Inspect.Room > 0 {
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

// corridor is the corridor ahead as the Classic view draws it: the
// engine's Sight, which the exploration record reads too.
func (c *Controller) corridor() ([]CorridorView, bool) {
	sight, dark := c.Game.Sight()
	var out []CorridorView
	for _, sq := range sight {
		out = append(out, CorridorView{Left: sq.Left, Right: sq.Right, Front: sq.Front,
			LeftAhead: sq.LeftAhead, RightAhead: sq.RightAhead})
	}
	return out, dark
}

// inspectView is the Inspect screen's progress, before its found list.
func (c *Controller) inspectView() *InspectView {
	g := c.Game
	v := &InspectView{Found: []FoundView{}, Room: c.RescueRoom(), Size: g.Town.Party.Size(),
		Picked: strings.HasSuffix(g.MazeMessage, joinsTheParty), Carried: []CharacterNote{}}
	here := map[*engine.Character]bool{}
	for _, ch := range g.MazeInspectFound {
		here[ch] = true
	}
	for _, s := range g.StrandedSites() {
		for _, ch := range s.Characters {
			if !here[ch] {
				v.Elsewhere++
			}
		}
	}
	for _, m := range g.Town.Party.Members {
		if m != nil && !m.IsAlive() {
			v.Carried = append(v.Carried, noteOf(m))
		}
	}
	return v
}
