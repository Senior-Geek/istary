package app

import (
	"fmt"
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// Gilgamesh's Tavern (Pascal CASTLE ADDPARTY/REMOVE): form the party.
// Inspecting a member opens the character sheet, which is still
// terminal-only.

const maxPartySize = 6

// PartyAlignment returns the party's alignment per Pascal GETALIGN: the last
// non-neutral member's alignment wins; empty or all-neutral is Neutral.
func PartyAlignment(town *engine.TownState) engine.Alignment {
	align := engine.Neutral
	for _, m := range town.Party.Members {
		if m != nil && m.Alignment != engine.Neutral {
			align = m.Alignment
		}
	}
	return align
}

// EnterTavern moves from the castle menu to the Tavern. In the terminal this
// is the castle's G key.
func (c *Controller) EnterTavern() error {
	if !c.inTown(engine.Castle) {
		return ErrWrongState
	}
	c.Game.Town.Location = engine.Tavern
	return nil
}

// LeaveTavern returns to the castle.
func (c *Controller) LeaveTavern() error {
	if !c.inTown(engine.Tavern) {
		return ErrWrongState
	}
	GoToCastle(c.Game.Town)
	return nil
}

// BeginAddMember opens the "WHO WILL JOIN ? >" prompt.
func (c *Controller) BeginAddMember() error {
	if !c.inTown(engine.Tavern) {
		return ErrWrongState
	}
	town := c.Game.Town
	town.Message = ""
	if town.Party.Size() >= maxPartySize {
		town.Message = "** PARTY IS FULL **"
		return ErrPartyFull
	}
	town.InputMode = engine.InputAddMember
	town.InputBuf = ""
	return nil
}

// AddMember asks name to join the party. When the character may join it
// returns needsPassword=true and the Tavern waits for SubmitTavernPassword.
// Otherwise it returns a *Refusal with the in-game message (WHO?, OUT,
// BAD ALIGNMENT, ...).
func (c *Controller) AddMember(name string) (needsPassword bool, err error) {
	town := c.Game.Town
	if c.inTown(engine.Tavern) {
		if err := c.BeginAddMember(); err != nil {
			return false, err
		}
	} else if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputAddMember {
		return false, ErrWrongState
	}
	town.InputBuf = NormalizeName(name)
	if strings.TrimSpace(town.InputBuf) == "" {
		c.submitAddMember()
		return false, ErrEmptyName
	}
	c.submitAddMember()
	if town.InputMode == engine.InputTavernPassword {
		return true, nil
	}
	return false, &Refusal{Message: town.Message}
}

// submitAddMember handles RETURN at "WHO WILL JOIN" (Pascal ADDPARTY,
// CASTLE.TEXT lines 148-191): find the name, skipping LOST characters, then
// check already-in-party, OUT, Wiz 3 legacy, and alignment before asking
// for the password.
func (c *Controller) submitAddMember() {
	town := c.Game.Town
	name := strings.TrimSpace(town.InputBuf)
	if name == "" {
		town.InputMode = engine.InputNone
		return
	}
	found := false
	for _, ch := range town.Roster.Characters {
		if ch == nil || ch.Status == engine.Lost {
			continue
		}
		if !strings.EqualFold(ch.Name, name) {
			continue
		}
		inParty := false
		for _, m := range town.Party.Members {
			if m == ch {
				inParty = true
				break
			}
		}
		switch {
		case inParty:
			town.Message = fmt.Sprintf("%s IS ALREADY IN THE PARTY!", ch.Name)
		case ch.InMaze || ch.MazeLevel != 0:
			// Pascal line 170-172: INMAZE or LOSTXYL.LOCATION[3] <> 0
			town.Message = "** OUT **"
		case c.Game.Scenario.ScenarioNum == 3 && !ch.IsLegacy:
			town.Message = "** ONLY A MEMORY **"
		default:
			// Pascal line 174-177: alignment check
			partyAlign := PartyAlignment(town)
			if partyAlign != engine.Neutral && ch.Alignment != engine.Neutral && partyAlign != ch.Alignment {
				town.Message = "** BAD ALIGNMENT **"
			} else {
				// From p-code CASTLE proc 30 (IC 1416-1468): "ENTER PASSWORD  >"
				town.EditChar = ch
				town.InputMode = engine.InputTavernPassword
				town.InputBuf = ""
				return
			}
		}
		found = true
		break
	}
	if !found {
		town.Message = "** WHO? **"
	}
	town.InputMode = engine.InputNone
	town.InputBuf = ""
}

// SubmitTavernPassword answers the joining character's password prompt.
func (c *Controller) SubmitTavernPassword(password string) error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputTavernPassword {
		return ErrWrongState
	}
	town.InputBuf = NormalizePassword(password)
	if !c.submitTavernPassword() {
		return &Refusal{Message: town.Message}
	}
	return nil
}

// submitTavernPassword checks the password (CASTLE proc 30) and reports
// whether the character joined.
func (c *Controller) submitTavernPassword() bool {
	town := c.Game.Town
	joined := false
	if town.EditChar != nil && town.InputBuf == town.EditChar.Password {
		town.EditChar.InMaze = true // Pascal line 179: INMAZE := TRUE
		town.Party.Members = append(town.Party.Members, town.EditChar)
		town.Message = fmt.Sprintf("%s JOINS THE PARTY!", town.EditChar.Name)
		joined = true
	} else {
		town.Message = "** THATS NOT IT **"
	}
	town.EditChar = nil
	town.InputMode = engine.InputNone
	town.InputBuf = ""
	return joined
}

// CancelTavernPassword abandons the password prompt.
func (c *Controller) CancelTavernPassword() error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputTavernPassword {
		return ErrWrongState
	}
	town.EditChar = nil
	town.InputMode = engine.InputNone
	town.InputBuf = ""
	return nil
}

// BeginRemoveMember opens "WHO WILL LEAVE ([RETURN] EXITS) >".
func (c *Controller) BeginRemoveMember() error {
	if !c.inTown(engine.Tavern) {
		return ErrWrongState
	}
	town := c.Game.Town
	town.Message = ""
	if town.Party.Size() == 0 {
		return ErrPartyEmpty
	}
	town.InputMode = engine.InputRemoveMember
	return nil
}

// RemoveMember removes the party member in slot (0-based). The character
// stays in the roster.
func (c *Controller) RemoveMember(slot int) error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || !(c.inTown(engine.Tavern) || town.InputMode == engine.InputRemoveMember) {
		return ErrWrongState
	}
	if slot < 0 || slot >= maxPartySize || slot >= len(town.Party.Members) || town.Party.Members[slot] == nil {
		if town.Party.Size() == 0 {
			return ErrPartyEmpty
		}
		return ErrNoSuchPartyMember
	}
	if town.InputMode == engine.InputNone {
		if err := c.BeginRemoveMember(); err != nil {
			return err
		}
	}
	c.removeSlot(slot)
	return nil
}

func (c *Controller) removeSlot(idx int) bool {
	town := c.Game.Town
	if idx >= len(town.Party.Members) || town.Party.Members[idx] == nil {
		return false
	}
	removed := town.Party.Members[idx]
	removed.InMaze = false // Pascal line 204: INMAZE := FALSE
	town.Party.Members = append(town.Party.Members[:idx], town.Party.Members[idx+1:]...)
	town.Message = fmt.Sprintf("%s LEAVES THE PARTY.", removed.Name)
	town.InputMode = engine.InputNone
	return true
}

// InspectMember opens the character sheet for the party member in slot
// (0-based). The sheet itself is still handled by the terminal front end.
func (c *Controller) InspectMember(slot int) error {
	if !c.inTown(engine.Tavern) {
		return ErrWrongState
	}
	town := c.Game.Town
	town.Message = ""
	if slot < 0 || slot >= len(town.Party.Members) || town.Party.Members[slot] == nil {
		return ErrNoSuchPartyMember
	}
	town.EditChar = town.Party.Members[slot]
	town.InputMode = engine.InputInspect
	return nil
}

// tavernMenuKey handles keys at the Tavern with no prompt open.
func (c *Controller) tavernMenuKey(k Key) {
	town := c.Game.Town
	switch k.Kind {
	case KeyEnter, KeyEscape:
		GoToCastle(town)
		return
	case KeyRune:
	default:
		return
	}
	town.Message = ""
	switch ch := upper(k.Rune); {
	case ch == 'A':
		c.BeginAddMember()
	case ch == 'R': // "WHO WILL LEAVE" (CASTLE byte 1619)
		if town.Party.Size() > 0 {
			c.BeginRemoveMember()
		}
	case ch >= '1' && ch <= '6': // #) SEE A MEMBER
		c.InspectMember(int(ch - '1'))
	}
}

// tavernPromptKey handles the join, password and leave prompts.
func (c *Controller) tavernPromptKey(k Key) {
	town := c.Game.Town
	switch town.InputMode {
	case engine.InputAddMember:
		switch k.Kind {
		case KeyEscape:
			town.InputMode = engine.InputNone
			town.InputBuf = ""
			town.Message = ""
		case KeyBackspace:
			town.InputBuf = backspace(town.InputBuf)
		case KeyEnter:
			c.submitAddMember()
		case KeyRune:
			town.InputBuf = appendNameChar(town.InputBuf, k.Rune)
		}

	case engine.InputTavernPassword:
		switch k.Kind {
		case KeyBackspace:
			town.InputBuf = backspace(town.InputBuf)
		case KeyEnter:
			c.submitTavernPassword()
		case KeyEscape:
			c.CancelTavernPassword()
		case KeyRune:
			town.InputBuf = appendPasswordChar(town.InputBuf, k.Rune)
		}

	case engine.InputRemoveMember:
		if k.Kind == KeyRune && k.Rune >= '1' && k.Rune <= '6' {
			c.removeSlot(int(k.Rune - '1'))
		}
		if k.Kind == KeyEnter || k.Kind == KeyEscape {
			town.InputMode = engine.InputNone
			town.Message = ""
		}
	}
}
