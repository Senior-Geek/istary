package app

// CampView (M6b-3A) is Camp and the inspect screen as a front end shows
// them: the open screen, the game's message and the choices it offers, and
// the operations open now (camp_ops.go). Characters themselves are read
// from the roster as elsewhere.
//
// It shows nothing the terminal's Camp does not: no position, floor or
// facing, except DUMAPIC's result, and only while its screen is open (the
// spell's purpose). Unidentified items have their unknown names.

import (
	"github.com/Senior-Geek/istary/engine"
)

// CampView is Camp's state for a front end.
type CampView struct {
	// Screen: "menu", "reorder", "inspect", "equip", "drop", "trade",
	// "tradeGold", "tradeItem", "spellBooks", "spellList", "castSpell",
	// "spellTarget", "useItem", "dumapic" or "malor" (terminal only).
	Screen    string            `json:"screen"`
	Message   string            `json:"message"`
	Member    int               `json:"member"`              // slot (0-based) of the inspected character, -1 for none
	Placed    []int             `json:"placed,omitempty"`    // reorder: slots (before reordering) placed so far
	Category  string            `json:"category,omitempty"`  // equip: the category offered
	Choices   []EquipChoiceView `json:"choices,omitempty"`   // equip: its usable items
	TradeWith int               `json:"tradeWith"`           // slot receiving a trade, -1 for none
	SpellBook string            `json:"spellBook,omitempty"` // "mage" or "priest" on the spell list
	Spell     string            `json:"spell,omitempty"`     // the spell waiting for its target
	Dumapic   *DumapicView      `json:"dumapic,omitempty"`   // only on the DUMAPIC screen
	Actions   []string          `json:"actions"`
}

// EquipChoiceView is one item offered by the equip screen.
type EquipChoiceView struct {
	Number int    `json:"number"` // for EquipChoose (1-based)
	Item   int    `json:"item"`   // the item's position in the pack (1-based)
	Name   string `json:"name"`   // unknown name until identified
}

// DumapicView is DUMAPIC's result, as its screen tells it.
type DumapicView struct {
	Facing string `json:"facing"` // NORTH, EAST, SOUTH, WEST
	East   int    `json:"east"`   // squares east of the stairs to the castle
	North  int    `json:"north"`  // squares north of them
	Level  int    `json:"level"`  // levels below the castle (1 = B1F)
}

// Camp actions, named after their operations.
const (
	ActCampInspect  = "inspect"       // CampInspect(slot)
	ActReorder      = "reorder"       // BeginReorder
	ActReorderPick  = "reorder-pick"  // ReorderPick(slot)
	ActEquipParty   = "equip-party"   // EquipParty
	ActDisband      = "disband"       // Disband
	ActLeaveInspect = "leave"         // LeaveInspection
	ActEquip        = "equip"         // BeginEquip
	ActEquipChoose  = "equip-choose"  // EquipChoose(n)
	ActEquipSkip    = "equip-skip"    // EquipSkip
	ActDrop         = "drop"          // BeginDrop
	ActDropItem     = "drop-item"     // DropItem(n)
	ActEndDrop      = "end-drop"      // EndDrop
	ActTrade        = "trade"         // BeginTrade
	ActTradeWith    = "trade-with"    // TradeWith(slot)
	ActTradeGold    = "trade-gold"    // TradeGold(amount)
	ActTradeItem    = "trade-item"    // TradeItem(n)
	ActEndTrade     = "end-trade"     // EndTrade
	ActSpellBooks   = "spell-books"   // OpenSpellBooks
	ActSpellBook    = "spell-book"    // ShowSpellBook(priest)
	ActCloseBook    = "close-book"    // CloseSpellBook
	ActCast         = "cast"          // CastSpell(name)
	ActCastOn       = "cast-on"       // CastOn(slot)
	ActCancelCast   = "cancel-cast"   // CancelCast
	ActIdentify     = "identify"      // Identify
	ActUse          = "use"           // UseItem(n)
	ActLeaveDumapic = "leave-dumapic" // LeaveDumapic
)

var campScreens = map[engine.InputMode]string{
	engine.InputReorder: "reorder", engine.InputInspect: "inspect", engine.InputEquip: "equip",
	engine.InputDrop: "drop", engine.InputTrade: "trade", engine.InputTradeGold: "tradeGold",
	engine.InputTradeTarget: "tradeItem", engine.InputSpellBooks: "spellBooks", engine.InputSpellList: "spellList",
	engine.InputCastSpell: "castSpell", engine.InputSpellTarget: "spellTarget", engine.InputUseItem: "useItem",
	engine.InputDumapic: "dumapic", engine.InputMalor: "malor",
}

// CampView returns Camp's state, or ErrWrongState outside Camp. It changes
// nothing and draws no random number.
func (c *Controller) CampView() (CampView, error) {
	g := c.Game
	if g.Phase != engine.PhaseCamp {
		return CampView{}, ErrWrongState
	}
	town := g.Town
	v := CampView{Screen: "menu", Message: town.Message, Member: -1, TradeWith: -1}
	if c.campAtMenu() {
		v.Actions = []string{ActCampInspect}
		if town.Party.Size() > 1 {
			v.Actions = append(v.Actions, ActReorder)
		}
		v.Actions = append(v.Actions, ActEquipParty, ActDisband, ActLeaveCamp)
		return v, nil
	}
	v.Screen = campScreens[town.InputMode]
	v.Member = PartySlot(town, town.EditChar)
	v.TradeWith = PartySlot(town, town.TradeTarget)
	items := g.Scenario.Items
	switch town.InputMode {
	case engine.InputReorder:
		for _, m := range town.ReorderResult {
			v.Placed = append(v.Placed, PartySlot(town, m))
		}
		v.Actions = []string{ActReorderPick}
	case engine.InputInspect:
		v.Actions = []string{ActEquip, ActDrop, ActTrade, ActSpellBooks, ActCast, ActUse, ActIdentify, ActLeaveInspect}
	case engine.InputEquip:
		if town.EquipCategory < len(engine.EquipCategories) {
			v.Category = engine.EquipCategoryNames[engine.EquipCategories[town.EquipCategory]]
		}
		for i, pos := range town.EquipChoices {
			poss := town.EditChar.Items[pos]
			name := ""
			if poss.ItemIndex >= 0 && poss.ItemIndex < len(items) {
				name = items[poss.ItemIndex].NameUnknown
				if poss.Identified {
					name = items[poss.ItemIndex].Name
				}
			}
			v.Choices = append(v.Choices, EquipChoiceView{Number: i + 1, Item: pos + 1, Name: name})
		}
		v.Actions = []string{ActEquipChoose, ActEquipSkip}
	case engine.InputDrop:
		v.Actions = []string{ActDropItem, ActEndDrop}
	case engine.InputTrade:
		v.Actions = []string{ActTradeWith, ActEndTrade}
	case engine.InputTradeGold:
		v.Actions = []string{ActTradeGold, ActEndTrade}
	case engine.InputTradeTarget:
		v.Actions = []string{ActTradeItem, ActEndTrade}
	case engine.InputSpellBooks:
		v.Actions = []string{ActSpellBook, ActCloseBook}
	case engine.InputSpellList:
		v.SpellBook = "mage"
		if town.InspectSpells == engine.PriestSpell {
			v.SpellBook = "priest"
		}
		v.Actions = []string{ActCloseBook}
	case engine.InputSpellTarget:
		if town.PendingSpell != nil {
			v.Spell = town.PendingSpell.Name
		}
		v.Actions = []string{ActCastOn, ActCancelCast}
	case engine.InputDumapic:
		v.Dumapic = &DumapicView{
			Facing: [4]string{"NORTH", "EAST", "SOUTH", "WEST"}[g.Facing&3],
			East:   g.PlayerX, North: g.PlayerY, Level: g.MazeLevel + 1,
		}
		v.Actions = []string{ActLeaveDumapic}
	}
	return v, nil
}
