package app

// The character inspect screen and its sub-screens (M6b-3A): equip, drop,
// trade gold and items, the spell books, and in Camp identify, cast and
// use. Moved intact from the terminal (cmd/istary handleTownPrompt and its
// helpers); only key decoding changed. The castle (Tavern, Training
// Grounds) and Camp share them. Camp's cast, use and MALOR screens are in
// camp.go.

import (
	"fmt"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// inspectModes are the inspect screen and its sub-screens.
var inspectModes = map[engine.InputMode]bool{
	engine.InputInspect: true, engine.InputEquip: true, engine.InputDrop: true,
	engine.InputTrade: true, engine.InputTradeGold: true, engine.InputTradeTarget: true,
	engine.InputSpellBooks: true, engine.InputSpellList: true,
}

// inspectKey handles one key on the inspect screen or a sub-screen.
func (c *Controller) inspectKey(k Key) {
	game := c.Game
	town := game.Town
	// Inspect / spell book modes
	if town.InputMode == engine.InputInspect {
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch >= 'a' && ch <= 'z' {
				ch -= 32
			}
			switch ch {
			case 'R':
				town.InputMode = engine.InputSpellBooks
				town.Message = ""
			case 'L':
				leaveInspect(town)
			case 'E':
				startEquipFlow(game, town)
			case 'D':
				town.InputMode = engine.InputDrop
				town.Message = ""
			case 'T':
				town.InputMode = engine.InputTrade
				town.Message = ""
			case 'S':
				// Cast spell — only available in camp (p-code CAMP IC 1939)
				if game.Phase == engine.PhaseCamp {
					town.InputMode = engine.InputCastSpell
					town.InputBuf = ""
					town.Message = ""
				}
			case 'U':
				// Use item — only available in camp (p-code CAMP IC 2399)
				if game.Phase == engine.PhaseCamp {
					town.InputMode = engine.InputUseItem
					town.Message = ""
				}
			case 'I':
				// Bishop identify — Pascal CAMP2.TEXT IDENTIFY proc (lines 6-28):
				// CAMPCHAR (the character doing camp actions) must be a Bishop.
				// Identifies unidentified items on the inspected character.
				if game.Phase == engine.PhaseCamp {
					c := town.EditChar
					if c == nil {
						break
					}
					// Check if the inspected character is a Bishop
					// (Pascal checks CAMPCHAR which is the active camp character)
					if c.Class != engine.Bishop {
						town.Message = "NOT BISHOP"
					} else {
						// Bishop identifies items on themselves
						// To identify OTHER characters' items, inspect them via the Bishop
						identified := 0
						for i := 0; i < c.ItemCount; i++ {
							if !c.Items[i].Identified {
								c.Items[i].Identified = true
								identified++
							}
						}
						if identified > 0 {
							town.Message = fmt.Sprintf("%d ITEM(S) IDENTIFIED", identified)
						} else {
							town.Message = "NOTHING TO IDENTIFY"
						}
					}
				}
			}
		} else if k.Kind == KeyEnter || k.Kind == KeyEscape {
			leaveInspect(town)
		}
		return
	}

	if town.InputMode == engine.InputEquip {
		if k.Kind == KeyEnter {
			// RETURN = skip this category (equip nothing for it)
			advanceEquipCategory(game, town)
		} else if k.Kind == KeyRune {
			ch := k.Rune
			if ch >= '1' && ch <= '9' {
				sel := int(ch - '0')
				handleEquipSelection(game, town, sel)
			}
		}
		return
	}

	if town.InputMode == engine.InputDrop {
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch == '0' {
				// 0=EXIT from p-code
				town.InputMode = engine.InputInspect
				town.Message = ""
			} else if ch >= '1' && ch <= '8' {
				handleDropItem(game, town, int(ch-'0'))
			}
		}
		if k.Kind == KeyEnter || k.Kind == KeyEscape {
			town.InputMode = engine.InputInspect
			town.Message = ""
		}
		return
	}

	if town.InputMode == engine.InputTrade {
		// "TRADE WITH" — select party member (1-6)
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch >= '1' && ch <= '6' {
				idx := int(ch-'0') - 1
				if idx < len(town.Party.Members) && town.Party.Members[idx] != nil &&
					town.Party.Members[idx] != town.EditChar {
					town.TradeTarget = town.Party.Members[idx]
					town.InputMode = engine.InputTradeGold
					town.InputBuf = ""
					town.Message = ""
				}
			}
		}
		if k.Kind == KeyEnter || k.Kind == KeyEscape {
			town.InputMode = engine.InputInspect
			town.Message = ""
		}
		return
	}

	if town.InputMode == engine.InputTradeGold {
		// "AMT OF GOLD ? >" — type gold amount, RETURN to confirm
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch >= '0' && ch <= '9' && len(town.InputBuf) < 12 {
				town.InputBuf += string(ch)
			}
		}
		if k.Kind == KeyBackspace {
			if len(town.InputBuf) > 0 {
				town.InputBuf = town.InputBuf[:len(town.InputBuf)-1]
			}
		}
		if k.Kind == KeyEnter {
			handleTradeGold(game, town)
		}
		if k.Kind == KeyEscape {
			town.InputMode = engine.InputInspect
			town.Message = ""
		}
		return
	}

	if town.InputMode == engine.InputTradeTarget {
		// "WHAT ITEM ([RET] EXITS) ? >" — select item to trade
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch >= '1' && ch <= '8' {
				town.TradeItemIdx = int(ch - '0')
				handleTradeItem(game, town)
			}
		}
		if k.Kind == KeyEnter || k.Kind == KeyEscape {
			town.InputMode = engine.InputInspect
			town.Message = ""
		}
		return
	}

	// Cast spell at camp — "WHAT SPELL ? >" (p-code CAMP IC 1939)
	if town.InputMode == engine.InputSpellBooks {
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch == 'm' || ch == 'M' {
				town.InspectSpells = engine.MageSpell
				town.InputMode = engine.InputSpellList
			} else if ch == 'p' || ch == 'P' {
				town.InspectSpells = engine.PriestSpell
				town.InputMode = engine.InputSpellList
			} else if ch == 'l' || ch == 'L' {
				town.InputMode = engine.InputInspect
			}
		} else if k.Kind == KeyEnter || k.Kind == KeyEscape {
			town.InputMode = engine.InputInspect
		}
		return
	}

	if town.InputMode == engine.InputSpellList {
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch == 'l' || ch == 'L' {
				town.InputMode = engine.InputSpellBooks
			}
		} else if k.Kind == KeyEnter || k.Kind == KeyEscape {
			town.InputMode = engine.InputSpellBooks
		}
		return
	}

}

// handleCharEdit handles the character edit menu at Training Grounds.
// From ROLLER segment p-code: I)nspect, D)elete, R)eroll, C)hange class,
// S)et password, or [RET] to leave.
func leaveInspect(town *engine.TownState) {
	if town.Location == engine.Training {
		town.InputMode = engine.InputCharEdit
	} else {
		town.InputMode = engine.InputNone
		town.EditChar = nil
	}
	town.Message = ""
}

// startEquipFlow begins the category-based equip process.
// From UTILITIE segment p-code: walks through WEAPON→ARMOR→SHIELD→HELMET→GAUNTLETS→MISC.
// For each category, shows matching items and lets user pick one.
func startEquipFlow(game *engine.GameState, town *engine.TownState) {
	// First, unequip everything (the original does this at UTILITIE byte 6626-6654)
	c := town.EditChar
	for i := 0; i < c.ItemCount; i++ {
		if c.Items[i].Equipped && !c.Items[i].Cursed {
			c.Items[i].Equipped = false
		}
	}
	c.RecalcAC(game.Scenario.Items) // AC always matches what is worn (#35)

	town.InputMode = engine.InputEquip
	town.EquipCategory = 0 // start with first category in EquipCategories
	town.Message = ""
	buildEquipChoices(game, town)
}

// buildEquipChoices scans the character's items for ones matching the current equip category.
// Populates town.EquipChoices with 0-based item positions.
func buildEquipChoices(game *engine.GameState, town *engine.TownState) {
	c := town.EditChar
	items := game.Scenario.Items
	catIdx := town.EquipCategory
	if catIdx >= len(engine.EquipCategories) {
		// Done with all categories — recalculate AC from equipped items
		c.RecalcAC(items)
		town.EquipChoices = nil

		// Party-wide equip mode: advance to next party member
		if town.EquipPartyMode {
			town.EquipPartyIdx++
			for town.EquipPartyIdx < len(town.Party.Members) {
				m := town.Party.Members[town.EquipPartyIdx]
				if m != nil && m.IsAlive() {
					break
				}
				town.EquipPartyIdx++
			}
			if town.EquipPartyIdx < len(town.Party.Members) {
				town.EditChar = town.Party.Members[town.EquipPartyIdx]
				startEquipFlow(game, town)
				return
			}
			// All party members equipped
			town.EquipPartyMode = false
			town.EditChar = nil
			town.InputMode = engine.InputNone
			return
		}

		town.InputMode = engine.InputInspect
		return
	}
	cat := engine.EquipCategories[catIdx]
	town.EquipChoices = nil

	// A cursed item holds its category: it stays equipped, and nothing
	// else of its kind can be (finding #34).
	if cursedIn(c, items, cat) {
		town.EquipCategory++
		buildEquipChoices(game, town)
		return
	}

	for i := 0; i < c.ItemCount; i++ {
		poss := c.Items[i]
		if poss.ItemIndex < 0 || poss.ItemIndex >= len(items) {
			continue
		}
		item := items[poss.ItemIndex]
		if item.TypeID != cat {
			continue
		}
		// Class usability check — from p-code byte 5779: IXP 16,1 on ClassUse field
		// Each bit in ClassUse corresponds to a class (bit 0=Fighter, bit 1=Mage, etc.)
		if item.ClassUse != 0 && (item.ClassUse&(1<<uint(c.Class))) == 0 {
			continue
		}
		town.EquipChoices = append(town.EquipChoices, i)
	}

	// If no items match this category, skip to next
	if len(town.EquipChoices) == 0 {
		town.EquipCategory++
		buildEquipChoices(game, town)
	}
}

// handleEquipSelection handles the user picking an item number in the equip category screen.
func handleEquipSelection(game *engine.GameState, town *engine.TownState, sel int) {
	if sel < 1 || sel > len(town.EquipChoices) {
		return
	}
	pos := town.EquipChoices[sel-1]
	c := town.EditChar

	// Equip the selected item; a cursed type, or another alignment's item,
	// sticks (finding #34).
	c.Items[pos].Equipped = true
	if engine.CursesOnEquip(game.Scenario.Items[c.Items[pos].ItemIndex], c) {
		c.Items[pos].Cursed = true
	}
	c.RecalcAC(game.Scenario.Items)
	advanceEquipCategory(game, town)
	if c.Items[pos].Cursed {
		town.Message = "** CURSED **" // after the next category's prompt clears it
	}
}

// cursedIn reports whether c wears a cursed item of category cat.
func cursedIn(c *engine.Character, items []data.Item, cat int) bool {
	for i := 0; i < c.ItemCount; i++ {
		p := c.Items[i]
		if p.Equipped && p.Cursed && p.ItemIndex >= 0 && p.ItemIndex < len(items) && items[p.ItemIndex].TypeID == cat {
			return true
		}
	}
	return false
}

// advanceEquipCategory moves to the next equipment category.
func advanceEquipCategory(game *engine.GameState, town *engine.TownState) {
	town.EquipCategory++
	town.Message = ""
	buildEquipChoices(game, town)
}


// handleDropItem removes an item from the character.
// From CAMP segment p-code proc 19: "DROP ITEM (0=EXIT) ? >"
// Warns "CURSED" if cursed+equipped (can't drop). Warns "EQUIPPED" if equipped.
// Then shifts items down, decrements ItemCount.
func handleDropItem(game *engine.GameState, town *engine.TownState, num int) {
	c := town.EditChar
	pos := num - 1

	if pos < 0 || pos >= c.ItemCount {
		return
	}

	// Cursed equipped items can't be dropped
	if c.Items[pos].Equipped && c.Items[pos].Cursed {
		town.Message = "CURSED"
		return
	}

	// Warn about equipped items (but still drop)
	if c.Items[pos].Equipped {
		town.Message = "EQUIPPED"
	}

	c.DropItem(pos)
	c.RecalcAC(game.Scenario.Items) // an equipped item may have gone (#35)
	if town.Message == "" {
		town.Message = "DROPPED"
	}
}

// handleTradeGold processes the gold amount entry.
// From CAMP segment p-code byte 3029: "AMT OF GOLD ? >"
// Validates amount, checks "NOT ENOUGH $", transfers gold, then moves to item trade.
func handleTradeGold(game *engine.GameState, town *engine.TownState) {
	c := town.EditChar
	target := town.TradeTarget

	if target == nil {
		town.InputMode = engine.InputInspect
		return
	}

	// Parse the gold amount from InputBuf
	amount := 0
	valid := true
	if town.InputBuf == "" {
		amount = 0
	} else {
		for _, ch := range town.InputBuf {
			if ch < '0' || ch > '9' {
				valid = false
				break
			}
			amount = amount*10 + int(ch-'0')
		}
	}

	if !valid || amount < 0 {
		town.Message = "BAD AMT"
		town.InputBuf = ""
		return
	}

	if amount > 0 {
		if c.Gold < amount {
			town.Message = "NOT ENOUGH $"
			town.InputBuf = ""
			return
		}
		c.Gold -= amount
		target.Gold += amount
	}

	// Move to item trading phase
	town.InputMode = engine.InputTradeTarget
	town.InputBuf = ""
	town.Message = ""
}

// handleTradeItem trades an item to another party member.
// From CAMP segment p-code proc 23, byte 3278: "WHAT ITEM ([RET] EXITS) ? >"
// Checks FULL (8 items), CURSED (equipped+cursed), EQUIPPED warning.
func handleTradeItem(game *engine.GameState, town *engine.TownState) {
	c := town.EditChar
	target := town.TradeTarget

	if target == nil {
		town.InputMode = engine.InputInspect
		return
	}

	// Check if target is full (8 items max)
	if target.ItemCount >= 8 {
		town.Message = "FULL"
		town.InputMode = engine.InputInspect
		return
	}

	num := town.TradeItemIdx
	pos := num - 1

	if pos < 0 || pos >= c.ItemCount {
		town.InputMode = engine.InputInspect
		return
	}

	// Equipped items can't be traded — show "EQUIPPED" and block
	if c.Items[pos].Equipped {
		if c.Items[pos].Cursed {
			town.Message = "** CURSED **"
		} else {
			town.Message = "** EQUIPPED **"
		}
		return
	}

	c.TradeItem(pos, target)
	town.Message = "TRADED"
}
