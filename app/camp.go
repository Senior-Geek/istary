package app

// Camp (M6b-3A): its menu (reorder, equip the party, disband, inspect a
// member) and its screens (cast a spell, choose its target, MALOR, DUMAPIC,
// use an item). Moved intact from the terminal (cmd/istary
// handleCampInput and its helpers); only key decoding changed. Inspect and
// its sub-screens are in inspect.go; L)EAVE is LeaveCamp (dungeon.go).

import (
	"fmt"
	"math/rand"

	"github.com/Senior-Geek/istary/engine"
)

func (c *Controller) campInput(k Key) {
	game := c.Game
	// Camp-only modes: cast spell, spell target, use item, MALOR
	if game.Town.InputMode == engine.InputCastSpell {
		handleCampCastSpell(game, k)
		return
	}
	if game.Town.InputMode == engine.InputSpellTarget {
		handleCampSpellTarget(game, k)
		return
	}
	if game.Town.InputMode == engine.InputMalor {
		handleMalorInput(game, k)
		return
	}
	if game.Town.InputMode == engine.InputDumapic {
		// Pascal p-code IC 2304-2314: wait for 'L' key to leave
		if k.Kind == KeyRune && (k.Rune == 'L' || k.Rune == 'l') {
			game.Town.InputMode = engine.InputInspect
			game.Town.Message = ""
		}
		return
	}
	if game.Town.InputMode == engine.InputUseItem {
		handleCampUseItem(game, k)
		return
	}

	// Shared inspect/equip/trade modes — route through the shared prompt handler

	if game.Town.InputMode == engine.InputInspect ||
		game.Town.InputMode == engine.InputEquip ||
		game.Town.InputMode == engine.InputDrop ||
		game.Town.InputMode == engine.InputTrade ||
		game.Town.InputMode == engine.InputTradeGold ||
		game.Town.InputMode == engine.InputTradeTarget ||
		game.Town.InputMode == engine.InputSpellBooks ||
		game.Town.InputMode == engine.InputSpellList {
		c.inspectKey(k)
		return
	}

	// Reorder mode — ">>" walks each slot, user picks who goes there
	if game.Town.InputMode == engine.InputReorder {
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch >= '1' && ch <= '6' {
				handleReorderPick(game, int(ch-'0'))
			}
		}
		return
	}

	if k.Kind != KeyRune {
		return
	}
	ch := k.Rune
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}
	game.Town.Message = ""

	switch ch {
	// L (leave camp) is handled by app.Controller.LeaveCamp.
	case 'R': // Reorder — from UTILITIE p-code byte 7121
		if game.Town.Party.Size() > 1 {
			game.Town.InputMode = engine.InputReorder
			game.Town.ReorderPos = 0
			game.Town.ReorderResult = nil
			game.Town.Message = ""
		}
	case 'E': // Equip — equip ALL party members in sequence
		// Pascal CAMP2.TEXT line 501-504: XGOTO := XEQPDSP; LLBASE04 := -1
		// LLBASE04 = -1 means equip all party members, iterating through each
		game.Town.EquipPartyMode = true
		game.Town.EquipPartyIdx = 0
		// Find first alive party member
		for game.Town.EquipPartyIdx < len(game.Town.Party.Members) {
			m := game.Town.Party.Members[game.Town.EquipPartyIdx]
			if m != nil && m.IsAlive() {
				break
			}
			game.Town.EquipPartyIdx++
		}
		if game.Town.EquipPartyIdx < len(game.Town.Party.Members) {
			game.Town.EditChar = game.Town.Party.Members[game.Town.EquipPartyIdx]
			startEquipFlow(game, game.Town)
		}
	case 'D': // Disband — from p-code CAMP proc 2 (IC 5834)
		// Save party members back to roster, age +25 weeks, return to town
		for i, m := range game.Town.Party.Members {
			if m != nil {
				m.Age += 25 // p-code: age increases 25 weeks per disband
				game.Town.Party.Members[i] = nil
			}
		}
		game.Phase = engine.PhaseTown
		GoToCastle(game.Town)
	case '1', '2', '3', '4', '5', '6': // Inspect — enter full inspect screen
		idx := int(ch-'0') - 1
		if idx < len(game.Town.Party.Members) && game.Town.Party.Members[idx] != nil {
			game.Town.EditChar = game.Town.Party.Members[idx]
			game.Town.InputMode = engine.InputInspect
			game.Town.Message = ""
		}
	}
}

// handleCampCastSpell handles spell name input during camp inspect.
// From p-code CAMP segment: "WHAT SPELL ? >" then spell hash lookup,
// then "CAST ON WHO" for targeted spells (proc 29, IC 708-772).
func handleCampCastSpell(game *engine.GameState, k Key) {
	town := game.Town
	if k.Kind == KeyRune {
		ch := k.Rune
		if ch >= 'A' && ch <= 'Z' {
			ch += 32
		}
		if (ch >= 'a' && ch <= 'z') && len(town.InputBuf) < 12 {
			town.InputBuf += string(ch)
		}
	}
	if k.Kind == KeyBackspace {
		if len(town.InputBuf) > 0 {
			town.InputBuf = town.InputBuf[:len(town.InputBuf)-1]
		}
	}
	if k.Kind == KeyEnter {
		sp := engine.LookupSpell(town.InputBuf)
		if sp == nil {
			town.Message = "** WHAT? **"
			town.InputMode = engine.InputInspect
			town.InputBuf = ""
			return
		}
		c := town.EditChar
		if !c.CanCastSpell(sp) {
			town.Message = "** CANT CAST **"
			town.InputMode = engine.InputInspect
			town.InputBuf = ""
			return
		}
		// Pascal CAMP.TEXT: IF FIZZLES > 0 THEN EXITCAST('SPELL HAS NO EFFECT')
		if game.Fizzles > 0 {
			town.Message = "SPELL HAS NO EFFECT"
			town.InputMode = engine.InputInspect
			town.InputBuf = ""
			return
		}
		// MALOR gets its own UI — Pascal UTILITIE.TEXT lines 432-473
		if sp.Name == "MALOR" {
			c.MageSpells[sp.Level-1]--
			town.MalorDeltaEW = 0
			town.MalorDeltaNS = 0
			town.MalorDeltaUD = 0
			town.InputMode = engine.InputMalor
			town.InputBuf = ""
			return
		}
		// Spell is valid and castable — check if it needs a target
		if sp.Target == engine.TargetPartyMember {
			// "CAST ON WHO" — from p-code proc 29 (IC 708)
			town.PendingSpell = sp
			town.InputMode = engine.InputSpellTarget
			town.InputBuf = ""
			return
		}
		// Non-targeted spells: deduct slot and apply immediately
		if sp.Type == engine.MageSpell {
			c.MageSpells[sp.Level-1]--
		} else {
			c.PriestSpells[sp.Level-1]--
		}
		applyCampSpell(game, sp, c)
		// Don't reset input mode if the spell set its own (e.g. DUMAPIC)
		if town.InputMode == engine.InputCastSpell {
			town.InputMode = engine.InputInspect
			town.InputBuf = ""
		}
	}
	if k.Kind == KeyEscape {
		town.InputMode = engine.InputInspect
		town.InputBuf = ""
		town.Message = ""
	}
}

// handleCampSpellTarget handles "CAST ON WHO" target selection.
// From p-code CAMP proc 29 (IC 708): WIZARDRY.proc15 selects party member.
func handleCampSpellTarget(game *engine.GameState, k Key) {
	town := game.Town
	if k.Kind == KeyRune {
		ch := k.Rune
		if ch >= '1' && ch <= '6' {
			idx := int(ch - '1')
			if idx < len(town.Party.Members) && town.Party.Members[idx] != nil {
				target := town.Party.Members[idx]
				sp := town.PendingSpell
				caster := town.EditChar
				// Deduct spell slot from caster
				if sp.Type == engine.MageSpell {
					caster.MageSpells[sp.Level-1]--
				} else {
					caster.PriestSpells[sp.Level-1]--
				}
				// Apply to target
				applyCampSpell(game, sp, target)
				town.PendingSpell = nil
				town.InputMode = engine.InputInspect
			}
		}
	}
	if k.Kind == KeyEnter || k.Kind == KeyEscape {
		town.PendingSpell = nil
		town.InputMode = engine.InputInspect
		town.Message = ""
	}
}

// campSpells are the spells applyCampSpell gives an effect in Camp; every
// other spell is "NOT IN CAMP" and still spent (finding #33). CampView
// offers only these (TestCampSpellsAreApplyCampSpells keeps them in step).
var campSpells = map[string]bool{
	"DIOS": true, "DIALMA": true, "DIAL": true, "MADI": true, "LATUMOFIS": true,
	"MILWA": true, "LOMILWA": true, "DUMAPIC": true, "MALOR": true,
}

// applyCampSpell applies a camp-castable spell effect to a target character.
func applyCampSpell(game *engine.GameState, sp *engine.Spell, target *engine.Character) {
	town := game.Town
	switch sp.Name {
	case "DIOS":
		target.HP += engine.RollDicePublic(1, 8, 0)
		if target.HP > target.MaxHP {
			target.HP = target.MaxHP
		}
		town.Message = fmt.Sprintf("%s HEALED", target.Name)
	case "DIALMA":
		target.HP += engine.RollDicePublic(3, 8, 0)
		if target.HP > target.MaxHP {
			target.HP = target.MaxHP
		}
		town.Message = fmt.Sprintf("%s HEALED", target.Name)
	case "DIAL":
		target.HP += engine.RollDicePublic(2, 8, 0)
		if target.HP > target.MaxHP {
			target.HP = target.MaxHP
		}
		town.Message = fmt.Sprintf("%s HEALED", target.Name)
	case "MADI":
		target.HP = target.MaxHP
		town.Message = fmt.Sprintf("%s FULLY HEALED", target.Name)
	case "LATUMOFIS":
		target.PoisonAmt = 0
		town.Message = fmt.Sprintf("%s POISON CURED", target.Name)
	case "MILWA":
		// Pascal CAMP.TEXT line 433: LIGHT := 15 + (RANDOM MOD 15)
		game.LightLevel = 15 + rand.Intn(15)
		town.Message = "LIGHT!"
	case "LOMILWA":
		// Pascal CAMP.TEXT line 443: LIGHT := 32000 (effectively permanent)
		game.LightLevel = 32000
		town.Message = "LIGHT!"
	case "DUMAPIC":
		// Pascal UTILITIE.TEXT p-code IC 1872-2314: full-screen location display
		town.InputMode = engine.InputDumapic
		return
	default:
		town.Message = "NOT IN CAMP"
	}
}

// handleCampUseItem handles item use during camp inspect.
// Camp-only — not available from town inspect.
func handleCampUseItem(game *engine.GameState, k Key) {
	town := game.Town
	if k.Kind == KeyRune {
		ch := k.Rune
		if ch == '0' {
			town.InputMode = engine.InputInspect
			town.Message = ""
		} else if ch >= '1' && ch <= '8' {
			idx := int(ch-'0') - 1
			c := town.EditChar
			if idx < c.ItemCount {
				item := game.Scenario.Items[c.Items[idx].ItemIndex]
				if item.SpellPower == 0 {
					town.Message = "** POWERLESS **"
				} else {
					town.Message = "** POWERLESS **"
				}
			}
			town.InputMode = engine.InputInspect
		}
	}
	if k.Kind == KeyEnter || k.Kind == KeyEscape {
		town.InputMode = engine.InputInspect
		town.Message = ""
	}
}

// handleMalorInput handles the MALOR teleport displacement UI.
// From Pascal UTILITIE.TEXT lines 432-473: N/S/E/W/U/D adjust displacement,
// RETURN teleports, ESC cancels.
func handleMalorInput(game *engine.GameState, k Key) {
	town := game.Town

	switch k.Kind {
	case KeyEscape:
		// "CHICKEN OUT" — cancel without teleporting (slot already spent)
		town.InputMode = engine.InputInspect
		town.Message = ""
	case KeyEnter:
		// TELEPORT — Pascal UTILITIE.TEXT lines 403-429
		// Pascal wraps X/Y mod 20 (coordinates are always 0-19)
		newX := ((game.PlayerX+town.MalorDeltaEW)%20 + 20) % 20
		newY := ((game.PlayerY+town.MalorDeltaNS)%20 + 20) % 20
		newLevel := game.MazeLevel + town.MalorDeltaUD

		maxLevel := len(game.Scenario.Mazes.Levels)

		// Check ROCK death: beyond max dungeon level
		if newLevel > 0 && newLevel > maxLevel {
			// "YOU LANDED IN SOLID ROCK" — all party LOST
			town.Message = "TELEPORTED INTO ROCK!"
			town.Message2 = "YOU ARE LOST FOREVER!"
			for _, m := range town.Party.Members {
				if m != nil {
					m.InMaze = false
					m.Status = engine.Lost
				}
			}
			game.Phase = engine.PhaseTown
			town.Location = engine.Castle
			town.InputMode = engine.InputNone
			return
		}

		// Check MOAT: level 0 but not at (0,0)
		if newLevel == 0 && !(newX == 0 && newY == 0) {
			town.Message = "TELEPORTED INTO THE MOAT!"
			town.Message2 = "YOU ARE LOST FOREVER!"
			for _, m := range town.Party.Members {
				if m != nil {
					m.InMaze = false
					m.Status = engine.Lost
				}
			}
			game.Phase = engine.PhaseTown
			town.Location = engine.Castle
			town.InputMode = engine.InputNone
			return
		}

		// Check VOLCANO: level < 0
		if newLevel < 0 {
			town.Message = "TELEPORTED INTO VOLCANO!"
			town.Message2 = "YOU ARE LOST FOREVER!"
			for _, m := range town.Party.Members {
				if m != nil {
					m.InMaze = false
					m.Status = engine.Lost
				}
			}
			game.Phase = engine.PhaseTown
			town.Location = engine.Castle
			town.InputMode = engine.InputNone
			return
		}

		// Level 0 at (0,0) = back to castle
		if newLevel == 0 && newX == 0 && newY == 0 {
			town.Message = "TELEPORTED TO CASTLE"
			game.PartyHome()
			for _, m := range town.Party.Members {
				if m != nil {
					m.PoisonAmt = 0
				}
			}
			game.Phase = engine.PhaseTown
			town.Location = engine.Castle
			town.InputMode = engine.InputNone
			return
		}

		// Valid teleport within dungeon
		game.PlayerX = newX
		game.PlayerY = newY
		game.MazeLevel = newLevel
		town.Message = fmt.Sprintf("TELEPORTED TO L%d (%d,%d)", newLevel+1, newX, newY)
		town.InputMode = engine.InputInspect
		game.Phase = engine.PhaseCamp

	case KeyRune:
		ch := k.Rune
		if ch >= 'a' && ch <= 'z' {
			ch -= 32
		}
		switch ch {
		case 'N':
			town.MalorDeltaNS++
		case 'S':
			town.MalorDeltaNS--
		case 'E':
			town.MalorDeltaEW++
		case 'W':
			town.MalorDeltaEW--
		case 'D':
			town.MalorDeltaUD++
		case 'U':
			town.MalorDeltaUD--
		}
	}
}

// handleReorderPick handles picking a party member for the current reorder slot.
// From UTILITIE p-code byte 7190: user presses a number (1-N) referring to the
// ORIGINAL party position. If that character hasn't been placed yet, they fill
// the current slot. Last remaining character auto-fills.
func handleReorderPick(game *engine.GameState, num int) {
	town := game.Town
	party := town.Party

	idx := num - 1
	if idx < 0 || idx >= len(party.Members) || party.Members[idx] == nil {
		return
	}

	picked := party.Members[idx]

	// Check not already placed — p-code checks local109[idx] == 99 (unassigned)
	for _, c := range town.ReorderResult {
		if c == picked {
			return
		}
	}

	town.ReorderResult = append(town.ReorderResult, picked)
	town.ReorderPos++

	// Auto-fill last remaining character — p-code only loops to party_size - 2
	if town.ReorderPos >= party.Size()-1 {
		for _, m := range party.Members {
			if m == nil {
				continue
			}
			placed := false
			for _, r := range town.ReorderResult {
				if r == m {
					placed = true
					break
				}
			}
			if !placed {
				town.ReorderResult = append(town.ReorderResult, m)
				break
			}
		}
	}

	if len(town.ReorderResult) >= party.Size() {
		party.Members = town.ReorderResult
		town.ReorderResult = nil
		town.InputMode = engine.InputNone
		town.Message = ""
	}
}
