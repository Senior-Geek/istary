package app

// Camp and inspect operations (M6b-3A): one call for each thing a player can
// do on Camp's menu and screens and on the inspect screen. Each checks that
// its screen is open, then sends the terminal's keys through HandleKey, so a
// front end that calls them takes exactly the terminal's path (same rules,
// same random numbers). The game's refusals ("** CANT CAST **", "NOT ENOUGH
// $", ...) are shown in Town.Message, as on the terminal; the operations
// return an error only when the action is not offered at all.
//
// MALOR is not offered here: Camp MALOR has a floor defect that can lose a
// party for good (finding #25), so CastSpell refuses it until a correction is
// approved. The terminal still casts it as before.

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// ErrMalorUnavailable refuses MALOR through CastSpell (finding #25).
var ErrMalorUnavailable = errors.New("MALOR is not available here until its floor defect (finding #25) is decided")

// press sends keys through HandleKey.
func (c *Controller) press(keys ...Key) {
	for _, k := range keys {
		c.HandleKey(k)
	}
}

func (c *Controller) mode() engine.InputMode { return c.Game.Town.InputMode }

// inCampMode reports that the party is in Camp with the given screen open.
func (c *Controller) inCampMode(m engine.InputMode) bool {
	return c.Game.Phase == engine.PhaseCamp && c.mode() == m
}

// atCampMenu reports that Camp shows its menu.
func (c *Controller) atCampMenu() bool {
	return c.Game.Phase == engine.PhaseCamp && c.campAtMenu()
}

// inspecting reports that the inspect screen (or one of its sub-screens m)
// is open, in Camp or in town.
func (c *Controller) inspecting(m engine.InputMode) bool {
	p := c.Game.Phase
	return (p == engine.PhaseCamp || p == engine.PhaseTown) && c.mode() == m && c.Game.Town.EditChar != nil
}

// digit is the key for a 1-based choice 1..9.
func digit(n int) Key { return Rune(rune('0' + n)) }

// --- Camp's menu ---

// CampInspect opens the inspect screen for the party member in slot
// (0-based), as the menu's 1-6.
func (c *Controller) CampInspect(slot int) error {
	if !c.atCampMenu() {
		return ErrWrongState
	}
	m := c.Game.Town.Party.Members
	if slot < 0 || slot >= len(m) || slot >= MaxPartySize || m[slot] == nil {
		return ErrNoSuchPartyMember
	}
	c.press(digit(slot + 1))
	return nil
}

// BeginReorder starts R)EORDER: the party is placed again slot by slot. A
// party of one cannot be reordered (ErrInvalidChoice).
func (c *Controller) BeginReorder() error {
	if !c.atCampMenu() {
		return ErrWrongState
	}
	c.press(Rune('R'))
	if c.mode() != engine.InputReorder {
		return ErrInvalidChoice
	}
	return nil
}

// ReorderPick puts the member who was in slot (0-based, the order before
// reordering) in the next place. The last one is placed by itself; then
// the new order applies. A member already placed is refused.
func (c *Controller) ReorderPick(slot int) error {
	if !c.inCampMode(engine.InputReorder) {
		return ErrWrongState
	}
	town := c.Game.Town
	before := len(town.ReorderResult)
	if slot < 0 || slot >= len(town.Party.Members) || slot >= MaxPartySize || town.Party.Members[slot] == nil {
		return ErrNoSuchPartyMember
	}
	c.press(digit(slot + 1))
	if c.mode() == engine.InputReorder && len(town.ReorderResult) == before {
		return ErrInvalidChoice
	}
	return nil
}

// EquipParty starts E)QUIP for the whole party: each living member in turn
// goes through the equip screen.
func (c *Controller) EquipParty() error {
	if !c.atCampMenu() {
		return ErrWrongState
	}
	c.press(Rune('E'))
	return nil
}

// Disband is D)ISBAND (engine behaviour, finding #26): the members leave
// the party, still marked as in the maze, and age 25 weeks; the game goes
// to the castle. Where they were is not recorded.
func (c *Controller) Disband() error {
	if !c.atCampMenu() {
		return ErrWrongState
	}
	c.press(Rune('D'))
	return nil
}

// --- The inspect screen (Camp or town) ---

// LeaveInspection closes the inspect screen (L).
func (c *Controller) LeaveInspection() error {
	if !c.inspecting(engine.InputInspect) {
		return ErrWrongState
	}
	c.press(Rune('L'))
	return nil
}

// BeginEquip starts E)QUIP for the inspected character: everything not
// cursed is unequipped, then each category offers its usable items.
func (c *Controller) BeginEquip() error {
	if !c.inspecting(engine.InputInspect) {
		return ErrWrongState
	}
	c.press(Rune('E'))
	return nil
}

// EquipChoose equips choice n (1-based) of the current category.
func (c *Controller) EquipChoose(n int) error {
	if !c.inspecting(engine.InputEquip) {
		return ErrWrongState
	}
	if n < 1 || n > len(c.Game.Town.EquipChoices) || n > 9 {
		return ErrInvalidChoice
	}
	c.press(digit(n))
	return nil
}

// EquipSkip equips nothing for the current category ([RET]).
func (c *Controller) EquipSkip() error {
	if !c.inspecting(engine.InputEquip) {
		return ErrWrongState
	}
	c.press(Key{Kind: KeyEnter})
	return nil
}

// BeginDrop opens D)ROP.
func (c *Controller) BeginDrop() error {
	if !c.inspecting(engine.InputInspect) {
		return ErrWrongState
	}
	c.press(Rune('D'))
	return nil
}

// DropItem drops item n (1-based). A cursed equipped item is not dropped
// ("CURSED"); an equipped one is ("EQUIPPED").
func (c *Controller) DropItem(n int) error {
	if !c.inspecting(engine.InputDrop) {
		return ErrWrongState
	}
	if n < 1 || n > c.Game.Town.EditChar.ItemCount {
		return ErrInvalidChoice
	}
	c.press(digit(n))
	return nil
}

// EndDrop closes D)ROP (0).
func (c *Controller) EndDrop() error {
	if !c.inspecting(engine.InputDrop) {
		return ErrWrongState
	}
	c.press(Rune('0'))
	return nil
}

// BeginTrade opens T)RADE: choose whom to trade with.
func (c *Controller) BeginTrade() error {
	if !c.inspecting(engine.InputInspect) {
		return ErrWrongState
	}
	c.press(Rune('T'))
	return nil
}

// TradeWith chooses the party member in slot (0-based) to trade with;
// then the gold amount is asked.
func (c *Controller) TradeWith(slot int) error {
	if !c.inspecting(engine.InputTrade) {
		return ErrWrongState
	}
	m := c.Game.Town.Party.Members
	if slot < 0 || slot >= len(m) || slot >= MaxPartySize || m[slot] == nil || m[slot] == c.Game.Town.EditChar {
		return ErrNoSuchPartyMember
	}
	c.press(digit(slot + 1))
	return nil
}

// TradeGold gives amount gold (0 for none) and moves on to the items. Too
// much is refused by the game ("NOT ENOUGH $") and the amount is asked
// again.
func (c *Controller) TradeGold(amount int) error {
	if !c.inspecting(engine.InputTradeGold) {
		return ErrWrongState
	}
	digits := ""
	if amount > 0 {
		digits = strconv.Itoa(amount)
	}
	if amount < 0 || len(digits) > 12 {
		return ErrInvalidChoice
	}
	for len(c.Game.Town.InputBuf) > 0 {
		c.press(Key{Kind: KeyBackspace})
	}
	for _, r := range digits {
		c.press(Rune(r))
	}
	c.press(Key{Kind: KeyEnter})
	return nil
}

// TradeItem gives item n (1-based). Equipped items are refused, full
// packs too ("FULL", which ends the trade).
func (c *Controller) TradeItem(n int) error {
	if !c.inspecting(engine.InputTradeTarget) {
		return ErrWrongState
	}
	if n < 1 || n > c.Game.Town.EditChar.ItemCount {
		return ErrInvalidChoice
	}
	c.press(digit(n))
	return nil
}

// EndTrade closes the trade ([RET]), from choosing whom or the items.
func (c *Controller) EndTrade() error {
	if !c.inspecting(engine.InputTrade) && !c.inspecting(engine.InputTradeTarget) && !c.inspecting(engine.InputTradeGold) {
		return ErrWrongState
	}
	if c.mode() == engine.InputTradeGold {
		c.press(Key{Kind: KeyEscape})
		return nil
	}
	c.press(Key{Kind: KeyEnter})
	return nil
}

// OpenSpellBooks opens R)EAD: choose the mage or priest book.
func (c *Controller) OpenSpellBooks() error {
	if !c.inspecting(engine.InputInspect) {
		return ErrWrongState
	}
	c.press(Rune('R'))
	return nil
}

// ShowSpellBook shows the mage (priest false) or priest book.
func (c *Controller) ShowSpellBook(priest bool) error {
	if !c.inspecting(engine.InputSpellBooks) {
		return ErrWrongState
	}
	if priest {
		c.press(Rune('P'))
	} else {
		c.press(Rune('M'))
	}
	return nil
}

// CloseSpellBook goes back one step: from a book to the choice, from the
// choice to the inspect screen.
func (c *Controller) CloseSpellBook() error {
	if !c.inspecting(engine.InputSpellList) && !c.inspecting(engine.InputSpellBooks) {
		return ErrWrongState
	}
	c.press(Rune('L'))
	return nil
}

// --- Camp only: spells, identify, use ---

// CastSpell casts the named spell from the inspect screen in Camp (S, the
// name, [RET]). Unknown names, spells the character cannot cast, spells
// with no effect in Camp and anti-magic are the game's refusals, shown in
// Town.Message; a spell cast on one member asks for the target (CastOn).
// DUMAPIC opens its location screen. MALOR is refused here (finding #25).
func (c *Controller) CastSpell(name string) error {
	if c.Game.Phase != engine.PhaseCamp || !c.inspecting(engine.InputInspect) {
		return ErrWrongState
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || len(name) > 12 || strings.Trim(name, "abcdefghijklmnopqrstuvwxyz") != "" {
		return ErrInvalidChoice
	}
	if name == "malor" {
		return ErrMalorUnavailable
	}
	c.press(Rune('S'))
	for _, r := range name {
		c.press(Rune(r))
	}
	c.press(Key{Kind: KeyEnter})
	return nil
}

// CastOn casts the pending spell on the party member in slot (0-based).
func (c *Controller) CastOn(slot int) error {
	if !c.inCampMode(engine.InputSpellTarget) {
		return ErrWrongState
	}
	m := c.Game.Town.Party.Members
	if slot < 0 || slot >= len(m) || slot >= MaxPartySize || m[slot] == nil {
		return ErrNoSuchPartyMember
	}
	c.press(digit(slot + 1))
	return nil
}

// CancelCast drops the pending spell before choosing its target; the spell
// is not spent.
func (c *Controller) CancelCast() error {
	if !c.inCampMode(engine.InputSpellTarget) {
		return ErrWrongState
	}
	c.press(Key{Kind: KeyEscape})
	return nil
}

// LeaveDumapic closes DUMAPIC's location screen.
func (c *Controller) LeaveDumapic() error {
	if !c.inCampMode(engine.InputDumapic) {
		return ErrWrongState
	}
	c.press(Rune('L'))
	return nil
}

// Identify is I)DENTIFY in Camp: a bishop identifies their own items.
func (c *Controller) Identify() error {
	if c.Game.Phase != engine.PhaseCamp || !c.inspecting(engine.InputInspect) {
		return ErrWrongState
	}
	c.press(Rune('I'))
	return nil
}

// UseItem is U)SE in Camp with item n (1-based). The engine's Use is
// unfinished: every item answers "** POWERLESS **" and nothing happens
// (finding #30).
func (c *Controller) UseItem(n int) error {
	if c.Game.Phase != engine.PhaseCamp || !c.inspecting(engine.InputInspect) {
		return ErrWrongState
	}
	if n < 1 || n > c.Game.Town.EditChar.ItemCount {
		return ErrInvalidChoice
	}
	c.press(Rune('U'), digit(n))
	return nil
}
