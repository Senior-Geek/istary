package app

// Rescue (R-2): the I)NSPECT screen's P)ICK UP, for every front end. The
// terminal picks with the original's prompt, "GET WHO (0=EXIT) >", one
// character a key (SPECIALS PICKUP); a GUI calls PickUp with the number of
// the character in the found list. Both go through pickUp, so the rules are
// the same: engine.RecoverStranded decides, and checks everything before it
// changes anything.

import (
	"fmt"

	"github.com/Senior-Geek/istary/engine"
)

// Conditions (R-3), as the maps and Inspect name them: the original's
// status words, written out in full where it shortened them. Every front
// end shows the word, so a condition never depends on a colour alone.
var conditionLabels = [...]string{
	engine.OK:        "OK",
	engine.Asleep:    "ASLEEP",
	engine.Afraid:    "AFRAID",
	engine.Paralyzed: "PARALYZED",
	engine.Stoned:    "STONED",
	engine.Dead:      "DEAD",
	engine.Ashed:     "ASHES",
	engine.Lost:      "LOST",
}

// ConditionLabel names condition s for the maps and Inspect.
func ConditionLabel(s engine.Status) string {
	if int(s) >= 0 && int(s) < len(conditionLabels) {
		return conditionLabels[s]
	}
	return s.String()
}

// noteOf is ch as the maps and Inspect note a character.
func noteOf(ch *engine.Character) CharacterNote {
	return CharacterNote{Name: ch.Name, Class: engine.AlignClass(ch.Alignment, ch.Class), Level: ch.Level, Condition: ConditionLabel(ch.Status)}
}

// joinsTheParty ends the message after a pickup ("NAME JOINS THE PARTY.").
const joinsTheParty = " JOINS THE PARTY."

// PartyFullMessage is the refusal when the party has six members
// (the original's "YOU HAVE 6").
const PartyFullMessage = "YOU HAVE 6"

// partyFullRefusal is the refusal to pick up ch into a full party.
func partyFullRefusal(ch *engine.Character) *Refusal {
	return &Refusal{Message: fmt.Sprintf("YOUR PARTY HAS %d; %s STAYS HERE.", MaxPartySize, ch.Name)}
}

// PickUp takes the n-th character (from 1) of those Inspect found into the
// party, if there is room; the Inspect screen stays open for the next. A
// full party is refused, and so is a number not on the list; either way
// nothing changes.
func (c *Controller) PickUp(n int) error {
	if c.Game.Phase != engine.PhaseMaze || c.mazePrompt() != "inspect" {
		return ErrWrongState
	}
	if len(c.Game.MazeInspectFound) == 0 {
		return ErrInvalidChoice
	}
	_, err := c.pickUp(n)
	return err
}

// LeaveInspect leaves the Inspect screen. Whoever was not picked up stays
// where they are.
func (c *Controller) LeaveInspect() error {
	if c.Game.Phase != engine.PhaseMaze || c.mazePrompt() != "inspect" {
		return ErrWrongState
	}
	c.leaveInspect()
	return nil
}

// RescueRoom is how many more characters the party has room for.
func (c *Controller) RescueRoom() int {
	room := MaxPartySize - c.Game.Town.Party.Size()
	if room < 0 {
		return 0
	}
	return room
}

func (c *Controller) leaveInspect() {
	g := c.Game
	g.MazeInspecting, g.MazeInspectFound = false, nil
	g.MazePicking, g.MazePickBuf = false, ""
	g.MazeMessage = ""
}

// pickUp is P)ICK UP of the n-th found character: the one place the rule
// is applied. On success the character leaves the found list and the
// message says so.
func (c *Controller) pickUp(n int) (*engine.Character, error) {
	g := c.Game
	found := g.MazeInspectFound
	if n < 1 || n > len(found) {
		return nil, ErrInvalidChoice
	}
	ch := found[n-1]
	if g.Town.Party.Size() >= MaxPartySize {
		return nil, partyFullRefusal(ch)
	}
	if err := g.RecoverStranded(ch); err != nil {
		if err == engine.ErrRescuePartyFull {
			return nil, partyFullRefusal(ch)
		}
		return nil, &Refusal{Message: ch.Name + " IS NOT HERE."}
	}
	g.MazeInspectFound = append(append([]*engine.Character(nil), found[:n-1]...), found[n:]...)
	g.MazeMessage = ch.Name + joinsTheParty
	return ch, nil
}

// mazeInspectKey is a key on the maze's Inspect screen (SPECIALS INSPECT): P)ICK UP
// or L)EAVE, and at P)ICK UP's prompt the number of the character to take,
// 0 (or [ESC]) to stop. With more than nine found, the number is typed and
// [RET] takes it.
func (c *Controller) mazeInspectKey(k Key) {
	g := c.Game
	if !g.MazePicking {
		if k.Kind != KeyRune {
			return
		}
		switch upper(k.Rune) {
		case 'L':
			c.leaveInspect()
		case 'P':
			if len(g.MazeInspectFound) == 0 {
				return
			}
			if g.Town.Party.Size() >= MaxPartySize {
				g.MazeMessage = PartyFullMessage
				return
			}
			g.MazePicking, g.MazePickBuf = true, ""
			g.MazeMessage = ""
		}
		return
	}

	stop := func() { g.MazePicking, g.MazePickBuf = false, "" }
	take := func(n int) {
		g.MazePickBuf = ""
		if n == 0 {
			stop()
			return
		}
		if _, err := c.pickUp(n); err != nil {
			if r, ok := err.(*Refusal); ok {
				g.MazeMessage = r.Message
			}
			// A number not on the list is ignored, as the original's
			// prompt waits for a valid one.
		}
		if len(g.MazeInspectFound) == 0 || g.Town.Party.Size() >= MaxPartySize {
			stop()
		}
	}
	switch {
	case k.Kind == KeyEscape:
		stop()
	case k.Kind == KeyBackspace:
		if b := g.MazePickBuf; b != "" {
			g.MazePickBuf = b[:len(b)-1]
		}
	case k.Kind == KeyEnter:
		if g.MazePickBuf != "" {
			n := 0
			fmt.Sscan(g.MazePickBuf, &n)
			take(n)
		}
	case k.Kind == KeyRune && k.Rune >= '0' && k.Rune <= '9':
		d := int(k.Rune - '0')
		switch {
		case len(g.MazeInspectFound) <= 9:
			take(d) // one key, as the original
		case g.MazePickBuf == "" && d == 0:
			stop()
		case len(g.MazePickBuf) < 2:
			g.MazePickBuf += string(k.Rune)
		}
	}
}
