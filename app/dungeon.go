package app

// Dungeon operations (M6a): entering the maze, a step, a turn, a kick, Camp,
// and the answers to the maze's prompts. Each operation is one key of the
// terminal's maze handler (mazeKey), so a GUI and the terminal take exactly
// the same transaction: the same rules, the same order, the same random
// numbers. A step may end in an encounter (Phase becomes PhaseCombat); that
// is the original game, not suppressed here.

import (
	"errors"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// ErrPromptPending is returned when the maze is waiting for an answer
// (a message to read, the stairs, a search, the buttons) and the operation
// is not that answer.
var ErrPromptPending = errors.New("answer the prompt first")

// stairsPrompt is the stairs question, set by the square's effect.
const stairsPrompt = "TAKE THEM (Y/N) ?"

// beep asks the front end for a beep ("OUCH!"), if it wants one.
func (c *Controller) beep() {
	if c.Beep != nil {
		c.Beep()
	}
}

// mazePrompt names what the maze is waiting for, in the order the maze
// handler checks: "" when it takes a command.
func (c *Controller) mazePrompt() string {
	g := c.Game
	switch {
	case g.ShowMap:
		return "map"
	case g.MazeDelayInput:
		return "delay"
	case g.MazeInspecting:
		return "inspect"
	case len(g.MazeMessages) > 0:
		return "messages"
	case g.MazeSearchYN:
		return "search"
	case g.WaitButton:
		return "buttons"
	case g.MazeMessage2 == stairsPrompt:
		return "stairs"
	}
	return ""
}

// command sends a maze command key when the maze takes commands.
func (c *Controller) command(r rune) error {
	if c.Game.Phase != engine.PhaseMaze {
		return ErrWrongState
	}
	if c.mazePrompt() != "" {
		return ErrPromptPending
	}
	c.mazeKey(Rune(r))
	return nil
}

// Forward steps one square ahead, or says "OUCH!" at a wall. A step runs
// the square's effect, healing and poison, and the encounter check.
func (c *Controller) Forward() error { return c.command('F') }

// TurnLeft and TurnRight turn the party in place.
func (c *Controller) TurnLeft() error  { return c.command('L') }
func (c *Controller) TurnRight() error { return c.command('R') }

// Kick moves through anything but a wall (doors, hidden doors), as a step.
func (c *Controller) Kick() error { return c.command('K') }

// Camp stops to camp.
func (c *Controller) Camp() error { return c.command('C') }

// Inspect searches the square for characters lost there.
func (c *Controller) Inspect() error { return c.command('I') }

// prompted sends a key answering the given prompts.
func (c *Controller) prompted(k Key, prompts ...string) error {
	if c.Game.Phase != engine.PhaseMaze {
		return ErrWrongState
	}
	p := c.mazePrompt()
	for _, want := range prompts {
		if p == want {
			c.mazeKey(k)
			return nil
		}
	}
	return ErrWrongState
}

// ContinueMessage shows the next page of a message, or dismisses it ([RET]).
func (c *Controller) ContinueMessage() error {
	return c.prompted(Key{Kind: KeyEnter}, "messages")
}

// Answer answers the stairs or search question.
func (c *Controller) Answer(yes bool) error {
	r := 'N'
	if yes {
		r = 'Y'
	}
	return c.prompted(Rune(r), "stairs", "search")
}

// PressButton presses button n (1 is A).
func (c *Controller) PressButton(n int) error {
	if c.mazePrompt() == "buttons" && (n < 1 || n > c.Game.ButtonCount) {
		return ErrInvalidChoice
	}
	return c.prompted(Rune('A'+rune(n-1)), "buttons")
}

// PickUp takes the characters Inspect found into the party.
func (c *Controller) PickUp() error {
	if c.mazePrompt() == "inspect" && len(c.Game.MazeInspectFound) == 0 {
		return ErrInvalidChoice
	}
	return c.prompted(Rune('P'), "inspect")
}

// LeaveInspect leaves the Inspect screen.
func (c *Controller) LeaveInspect() error { return c.prompted(Rune('L'), "inspect") }

// EnterMaze takes the party from the Edge of Town into the maze, to Camp at
// the entrance (EDGE OF TOWN M)AZE).
func (c *Controller) EnterMaze() error {
	g := c.Game
	if g.Phase != engine.PhaseTown {
		return ErrWrongState
	}
	if g.Town.Party.Size() == 0 {
		return ErrPartyEmpty
	}
	g.Phase = engine.PhaseCamp
	g.MazeLevel = 0
	g.PlayerX = 0
	g.PlayerY = 0
	g.Facing = engine.North
	g.MazeMessage = ""
	g.MazeMessage2 = ""
	g.InitFightMap()
	return nil
}

// campAtMenu reports whether Camp shows its menu rather than one of its
// screens (inspect, equip, spells, reorder ...).
func (c *Controller) campAtMenu() bool {
	switch c.Game.Town.InputMode {
	case engine.InputCastSpell, engine.InputSpellTarget, engine.InputMalor, engine.InputDumapic,
		engine.InputUseItem, engine.InputInspect, engine.InputEquip, engine.InputDrop,
		engine.InputTrade, engine.InputTradeGold, engine.InputTradeTarget,
		engine.InputSpellBooks, engine.InputSpellList, engine.InputReorder:
		return false
	}
	return true
}

// LeaveCamp leaves Camp for the maze (CAMP L)EAVE). At the stairs the
// question is asked at once, as in the original.
func (c *Controller) LeaveCamp() error {
	g := c.Game
	if g.Phase != engine.PhaseCamp || !c.campAtMenu() {
		return ErrWrongState
	}
	g.Town.Message = ""
	g.Phase = engine.PhaseMaze
	// Check if at stairs (0,0) — original shows "STAIRS GOING UP" immediately
	cell := g.CurrentCell()
	if cell != nil && cell.Type == data.SqStairs {
		if cell.DestLevel == 0 {
			g.MazeMessage = "STAIRS GOING UP."
		} else {
			g.MazeMessage = "STAIRS GOING DOWN."
		}
		g.MazeMessage2 = stairsPrompt
	}
	return nil
}

// campKey handles a key in Camp: L)EAVE at the menu (LeaveCamp), and
// everything else (campInput, camp.go).
func (c *Controller) campKey(k Key) bool {
	if k.Kind == KeyRune && upper(k.Rune) == 'L' && c.campAtMenu() {
		c.LeaveCamp()
		return true
	}
	c.campInput(k)
	return true
}

// edgeOfTownKey handles M)AZE at the Edge of Town.
func (c *Controller) edgeOfTownKey(k Key) bool {
	if k.Kind != KeyRune || upper(k.Rune) != 'M' {
		return false
	}
	c.Game.Town.Message = ""
	c.EnterMaze() // with no party, nothing happens, as in the original
	return true
}
