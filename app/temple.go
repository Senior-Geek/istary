package app

import (
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// The Temple of Cant (T-1), moved intact from the terminal (cmd/istary
// handleTempleNameInput, handleTempleInput, templePerformHeal,
// handleTempleReturn and the castle menu's C): the terminal reaches it
// through HandleKey. The flow is traced from the SHOPS segment p-code
// (procs 28, 27, 23, 25); the treatment is engine.TempleHeal and the
// donation engine.TempleDonation. Only the treatment draws random numbers.
//
// One change of place, not of outcome: the terminal's treatment saved the
// game itself; the controller does no I/O, and the terminal's main loop
// saves after every key as before, so the same state reaches the disk.

// castleTempleKey is the castle menu's C: the Temple, for a party.
func (c *Controller) castleTempleKey() {
	town := c.Game.Town
	town.Message = ""
	if town.Party.Size() == 0 {
		town.Message = "YOU HAVE NO PARTY!"
	} else {
		town.Location = engine.Temple
		town.TempleStep = engine.TempleWho
		town.TempleChar = nil
		town.TempleCost = 0
		town.TempleMessages = nil
		town.InputMode = engine.InputTempleHelp
		town.InputBuf = ""
	}
}

// templeKey handles a key at the Temple with no prompt open: RETURN steps
// back, a letter or digit goes to the step on screen.
func (c *Controller) templeKey(k Key) {
	town := c.Game.Town
	switch k.Kind {
	case KeyEnter, KeyEscape:
		templeReturn(town)
		return
	case KeyRune:
	default:
		return
	}
	town.Message = ""
	templeInput(town, upper(k.Rune))
}

// templeNameKey handles text input at "WHO ARE YOU HELPING ? >".
// From SHOPS segment p-code proc 28 (IC 76-436):
//
//	Read name, search roster, check status, display result.
func (c *Controller) templeNameKey(k Key) {
	town := c.Game.Town

	switch k.Kind {
	case KeyEscape:
		// ESC exits temple
		town.InputMode = engine.InputNone
		town.InputBuf = ""
		GoToCastle(town)
	case KeyBackspace:
		if len(town.InputBuf) > 0 {
			town.InputBuf = town.InputBuf[:len(town.InputBuf)-1]
		}
	case KeyEnter:
		name := strings.TrimSpace(town.InputBuf)
		if name == "" {
			// Empty name = exit temple (from p-code: empty string → EXIT)
			town.InputMode = engine.InputNone
			town.InputBuf = ""
			GoToCastle(town)
			return
		}

		// Search roster for character by name (case-insensitive)
		// From p-code proc 28 (IC 205-299): loop through roster comparing names
		var found *engine.Character
		for _, ch := range town.Roster.Characters {
			if ch == nil {
				continue
			}
			if strings.EqualFold(ch.Name, name) {
				found = ch
				break
			}
		}

		if found == nil {
			// Character not found — silent return to name prompt
			// (p-code just loops back to "WHO ARE YOU HELPING")
			town.InputBuf = ""
			town.Message = ""
			return
		}

		// Check if character is lost — from p-code proc 28 (IC 374-396)
		if found.Status == engine.Lost {
			town.Message = found.Name + " IS LOST"
			town.InputBuf = ""
			return
		}

		// Check if character is OK — from p-code proc 28 (IC 397-416)
		if found.Status == engine.OK {
			town.Message = found.Name + " IS OK"
			town.InputBuf = ""
			return
		}

		// Character needs healing — check if they need temple services
		// Statuses that temple can handle: Paralyzed, Stoned, Dead, Ashed
		cost := engine.TempleDonation(found.Status, found.Level)
		if cost == 0 {
			// Asleep/Afraid — not handled by temple (will be OK after resting)
			town.Message = found.Name + " IS OK"
			town.InputBuf = ""
			return
		}

		// Set up donation display
		town.TempleChar = found
		town.TempleCost = cost
		town.TempleStep = engine.TempleTithe
		town.InputMode = engine.InputNone
		town.InputBuf = ""
		town.Message = ""
	case KeyRune:
		if len(town.InputBuf) < 15 {
			town.InputBuf += string(k.Rune)
		}
	}
}

// templeInput handles key presses at the temple based on current TempleStep.
// From SHOPS segment p-code procs 27/23/25.
func templeInput(town *engine.TownState, ch rune) {
	switch town.TempleStep {
	case engine.TempleWho:
		// Text input mode — handled by templeNameKey via InputTempleHelp
		// If we get here, the user pressed a key while in TempleWho without InputMode set
		town.InputMode = engine.InputTempleHelp
		town.InputBuf = string(ch)

	case engine.TempleTithe:
		// "WHO WILL TITHE" — select party member (1-6)
		// From p-code proc 27 (IC 561-597): WIZARDRY.proc15 reads party member selection
		if ch >= '1' && ch <= '6' {
			idx := int(ch-'0') - 1
			if idx < len(town.Party.Members) && town.Party.Members[idx] != nil {
				payer := town.Party.Members[idx]
				// Check if payer can afford the donation
				// From p-code proc 27 (IC 599-647): WIZARDRY.proc11 subtracts gold,
				// if result > 0 (not enough): "CHEAP APOSTATES! OUT!"
				if payer.Gold < town.TempleCost {
					town.TempleStep = engine.TempleResult
					town.TempleMessages = []string{"CHEAP APOSTATES! OUT!"}
					town.Message = ""
					return
				}

				// Deduct gold from payer
				payer.Gold -= town.TempleCost

				// Advance to ritual display
				// From p-code proc 23 (IC 796-1052): MURMUR-CHANT-PRAY-INVOKE
				town.TempleStep = engine.TempleRitual
				town.Message = ""
			}
		}

	case engine.TempleRitual:
		// During ritual display, any keypress performs the heal and shows result
		templePerformHeal(town)
	}
}

// templePerformHeal executes the healing attempt and transitions to result.
func templePerformHeal(town *engine.TownState) {
	msgs, _ := engine.TempleHeal(town.TempleChar)
	town.TempleMessages = msgs
	town.TempleStep = engine.TempleResult
	town.Message = ""
}

// templeReturn handles RETURN/ESC at the temple.
func templeReturn(town *engine.TownState) {
	switch town.TempleStep {
	case engine.TempleWho:
		// RETURN at name prompt without InputMode — exit temple
		GoToCastle(town)
	case engine.TempleTithe:
		// RETURN at tithe selection — back to name prompt
		town.TempleStep = engine.TempleWho
		town.TempleChar = nil
		town.TempleCost = 0
		town.InputMode = engine.InputTempleHelp
		town.InputBuf = ""
		town.Message = ""
	case engine.TempleRitual:
		// RETURN during ritual — perform heal and show result
		templePerformHeal(town)
	case engine.TempleResult:
		// RETURN at result — back to name prompt
		town.TempleStep = engine.TempleWho
		town.TempleChar = nil
		town.TempleCost = 0
		town.TempleMessages = nil
		town.InputMode = engine.InputTempleHelp
		town.InputBuf = ""
		town.Message = ""
	}
}
