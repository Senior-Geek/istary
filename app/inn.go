package app

import (
	"fmt"

	"github.com/Senior-Geek/istary/engine"
)

// The Adventurer's Inn (I-1), moved intact from the terminal (cmd/istary
// handleInnInput, innTransitionToLevelUp, handleInnReturn, the heal timer's
// innHealTick, and the castle menu's A): the terminal reaches it through
// HandleKey, and its 0.7-second timer calls InnTick. The flow is traced
// from the CASTLE segment p-code and the Pascal CASTLE2.TEXT. Healing draws
// no random number; the level check does.

// castleInnKey is the castle menu's A: the Inn, for a party.
func (c *Controller) castleInnKey() {
	town := c.Game.Town
	town.Message = ""
	if town.Party.Size() == 0 {
		town.Message = "YOU HAVE NO PARTY!"
	} else {
		town.Location = engine.Inn
		town.InnStep = engine.InnWho
		town.InnChar = nil
		town.InnMessages = nil
	}
}

// innKey handles a key at the Inn: RETURN steps back, a letter or digit
// goes to the menu on screen.
func (c *Controller) innKey(k Key) {
	town := c.Game.Town
	switch k.Kind {
	case KeyEnter, KeyEscape:
		innReturn(town)
		return
	case KeyRune:
	default:
		return
	}
	town.Message = ""
	innInput(town, upper(k.Rune))
}

// innInput handles key presses at the inn based on current InnStep.
// From CASTLE segment p-code procs 26/24/23/15/14:
//
//	InnWho: "WHO WILL STAY" → select party member 1-6, RETURN exits to castle
//	InnSelectRoom: "WELCOME [name]. WE HAVE:" → A-E selects room, RETURN → InnWho
//	InnResult: rest results + XP message, RETURN → InnSelectRoom (same character)
func innInput(town *engine.TownState, ch rune) {
	switch town.InnStep {
	case engine.InnWho:
		if ch >= '1' && ch <= '6' {
			idx := int(ch-'0') - 1
			if idx < len(town.Party.Members) && town.Party.Members[idx] != nil {
				c := town.Party.Members[idx]
				if c.Status != engine.OK {
					town.Message = c.Name + " IS NOT OK"
				} else {
					town.InnChar = c
					town.InnStep = engine.InnSelectRoom
					town.Message = ""
				}
			}
		}

	case engine.InnSelectRoom:
		// Pascal TAKENAP (CASTLE2.TEXT lines 415-479):
		// Sets up healing parameters, then enters animated healing loop.
		var healAmt, healCost int
		switch ch {
		case 'A':
			healAmt = 0
			healCost = 0
		case 'B':
			healAmt = 1
			healCost = 10
		case 'C':
			healAmt = 3
			healCost = 50
		case 'D':
			healAmt = 7
			healCost = 200
		case 'E':
			healAmt = 10 // Pascal CASTLE2.TEXT line 500: TAKENAP(10, 500)
			healCost = 500
		default:
			return
		}
		c := town.InnChar

		// Restore spells on any room (Pascal calls SETSPELS in TAKENAP)
		engine.RestoreSpells(c)

		if healAmt == 0 {
			// Stables: no healing, just "IS NAPPING" then straight to level check
			town.InnMessages = nil
			town.InnStep = engine.InnHealing
			town.InnHealAmt = 0
			town.InnHealCost = 0
		} else {
			// Paid room: start animated healing loop
			town.InnHealAmt = healAmt
			town.InnHealCost = healCost
			town.InnStep = engine.InnHealing
		}
		town.Message = ""
	}
}

// InnTick is one tick of the Inn's healing animation — Pascal HEALHP loop
// (CASTLE2.TEXT lines 423-448). Each tick: heal HPADD HP, deduct gold.
// When done → level-up screen. Pascal delay: FOR PAUSEX := 1 TO 500 DO ;
// (~0.5-1s on Apple II). It reports whether anything changed; outside a
// stay it does nothing.
func (c *Controller) InnTick() bool {
	game := c.Game
	if game.Phase != engine.PhaseTown || game.Town.Location != engine.Inn ||
		game.Town.InnStep != engine.InnHealing || game.Town.InnChar == nil {
		return false
	}
	ch := game.Town.InnChar
	town := game.Town
	if town.InnHealAmt == 0 {
		// Stables: no healing animation, go straight to level-up
		innLevelCheck(game)
	} else if ch.HP < ch.MaxHP && ch.Gold >= town.InnHealCost {
		// One heal step per tick
		ch.HP += town.InnHealAmt
		if ch.HP > ch.MaxHP {
			ch.HP = ch.MaxHP
		}
		ch.Gold -= town.InnHealCost
	} else {
		// Healing done — transition to level-up
		innLevelCheck(game)
	}
	return true
}

// innLevelCheck runs CHNEWLEV (Pascal CASTLE2.TEXT lines 386-412)
// after healing completes. Shows level-up or XP needed.
func innLevelCheck(game *engine.GameState) {
	town := game.Town
	c := town.InnChar
	prevLevel := c.Level
	// Pascal MADELEV order: level++, MAXLEVAC, SETSPELS, TRYLEARN, GAINLOST, HP calc
	learnedSpells := engine.CheckLevelUp(c, game) // level++, MAXLEVAC, TRYLEARN

	if c.Level > prevLevel {
		town.InnMessages = []string{
			"YOU MADE A LEVEL!",
			"",
		}
		if learnedSpells {
			town.InnMessages = append(town.InnMessages, "YOU LEARNED NEW SPELLS!!!!")
		}
		// GAINLOST — age-based stat changes (must run BEFORE HP recalc)
		statMsgs := engine.InnStatChanges(c)
		town.InnMessages = append(town.InnMessages, statMsgs...)
		// HP recalculation — uses VIT which may have changed in GAINLOST
		engine.RecalcHP(c)
	} else {
		needed := engine.XPForNextLevel(c, game)
		if needed > 0 {
			diff := needed - c.XP
			if diff > 0 {
				town.InnMessages = []string{
					"",
					fmt.Sprintf("YOU NEED %d MORE", diff),
					"EXPERIENCE POINTS TO MAKE LEVEL",
				}
			}
		}
	}
	town.InnStep = engine.InnLevelUp
}

// innReturn handles RETURN/ESC at the inn.
// From p-code: Return at InnResult goes back to InnSelectRoom (same character),
// Return at InnSelectRoom goes back to InnWho, Return at InnWho exits to castle.
func innReturn(town *engine.TownState) {
	switch town.InnStep {
	case engine.InnWho:
		GoToCastle(town)
	case engine.InnSelectRoom:
		town.InnStep = engine.InnWho
		town.InnChar = nil
		town.Message = ""
	case engine.InnHealing:
		// Key press during healing skips the animation (Pascal KEYAVAIL check)
		// Transition to level-up immediately — handled by next tick
	case engine.InnLevelUp:
		// Return goes back to room selection for the SAME character
		town.InnStep = engine.InnSelectRoom
		town.InnMessages = nil
		town.Message = ""
	}
}
