package main

// Camp (M6b-3B). The Camp screen calls one method, CampAction, with one of
// the actions app.CampView lists; it runs the controller's Camp operation
// (app/camp_ops.go), which presses exactly the terminal's keys. On a real
// expedition the action is saved with its checkpoint, like every dungeon
// action (combatDo); a practice expedition saves nothing.
//
// Two of Camp's actions are not offered here and are refused if asked for:
// D)ISBAND, which does not record where the party was (finding #26) and
// waits for rescue (M6b-4), and MALOR, whose floor defect can lose the
// party for good (finding #25; app.Controller.CastSpell refuses it). Spells
// without an effect in Camp, which the game would waste (finding #33), are
// not cast either. The View says why each is unavailable.

import (
	"errors"
	"slices"
	"strings"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// CampRequest is a Camp action chosen in the front end. Slot is a party
// slot (0-based) for inspect, reorder-pick, trade-with and cast-on; N is a
// 1-based choice (equip-choose) or item (drop-item, trade-item, use); Gold
// is the amount for trade-gold; Spell is the spell for cast; Priest picks
// the book for spell-book.
type CampRequest struct {
	Action string `json:"action"`
	Slot   int    `json:"slot"`
	N      int    `json:"n"`
	Gold   int    `json:"gold"`
	Spell  string `json:"spell"`
	Priest bool   `json:"priest"`
}

// CampPanel is Camp for the front end: the controller's CampView (without
// the actions the GUI does not offer), each party member's sheet, and why
// an action is unavailable.
type CampPanel struct {
	View app.CampView `json:"view"`
	// Party is each party member's sheet, in formation order.
	Party []CharacterView `json:"party"`
	// Unavailable lists the actions of this screen that cannot be taken,
	// with the reason.
	Unavailable []Unavailable `json:"unavailable,omitempty"`
}

// Unavailable is an action that cannot be taken, and why.
type Unavailable struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

var (
	errDisbandUnavailable = errors.New("Disband is not available yet: the game does not record where a party disbanded, so it could not be found again (finding #26). It will come with rescue.")
	errUnknownSpell       = errors.New("This character does not know that spell.")
)

// CampAction performs one of the Camp screen's actions.
func (s *Session) CampAction(r CampRequest) View {
	return s.onPractice(func(c *app.Controller) error {
		if !s.expedition && !s.real || c.Game.Phase != engine.PhaseCamp {
			return app.ErrWrongState
		}
		switch r.Action {
		case app.ActCampInspect:
			return c.CampInspect(r.Slot)
		case app.ActReorder:
			return c.BeginReorder()
		case app.ActReorderPick:
			return c.ReorderPick(r.Slot)
		case app.ActEquipParty:
			return c.EquipParty()
		case app.ActDisband:
			return errDisbandUnavailable
		case app.ActLeaveCamp:
			return c.LeaveCamp()
		case app.ActLeaveInspect:
			return c.LeaveInspection()
		case app.ActEquip:
			return c.BeginEquip()
		case app.ActEquipChoose:
			return c.EquipChoose(r.N)
		case app.ActEquipSkip:
			return c.EquipSkip()
		case app.ActDrop:
			return c.BeginDrop()
		case app.ActDropItem:
			return c.DropItem(r.N)
		case app.ActEndDrop:
			return c.EndDrop()
		case app.ActTrade:
			return c.BeginTrade()
		case app.ActTradeWith:
			return c.TradeWith(r.Slot)
		case app.ActTradeGold:
			return c.TradeGold(r.Gold)
		case app.ActTradeItem:
			return c.TradeItem(r.N)
		case app.ActEndTrade:
			return c.EndTrade()
		case app.ActSpellBooks:
			return c.OpenSpellBooks()
		case app.ActSpellBook:
			return c.ShowSpellBook(r.Priest)
		case app.ActCloseBook:
			return c.CloseSpellBook()
		case app.ActCast:
			return castInCamp(c, r.Spell)
		case app.ActCastOn:
			return c.CastOn(r.Slot)
		case app.ActCancelCast:
			return c.CancelCast()
		case app.ActIdentify:
			return c.Identify()
		case app.ActUse:
			return c.UseItem(r.N)
		case app.ActLeaveDumapic:
			return c.LeaveDumapic()
		}
		return app.ErrInvalidChoice
	})
}

// castInCamp casts a spell CampView lists as castable; the others are
// refused with its reason, MALOR included, before anything is pressed.
func castInCamp(c *app.Controller, name string) error {
	v, err := c.CampView()
	if err != nil {
		return err
	}
	if v.Screen != "inspect" {
		return app.ErrWrongState
	}
	name = strings.ToUpper(strings.TrimSpace(name))
	for _, sp := range v.Spells {
		switch {
		case sp.Name != name:
		case sp.Name == "MALOR":
			return app.ErrMalorUnavailable
		case !sp.Castable:
			return errors.New(sp.Reason)
		default:
			return c.CastSpell(sp.Name)
		}
	}
	return errUnknownSpell
}

// campPanel builds the Camp panel of the controller c, which is in Camp.
func campPanel(c *app.Controller) *CampPanel {
	v, err := c.CampView()
	if err != nil {
		return nil
	}
	p := &CampPanel{View: v}
	p.View.Actions = slices.DeleteFunc(slices.Clone(v.Actions), func(a string) bool { return a == app.ActDisband })
	g := c.Game
	for _, m := range g.Town.Party.Members {
		if m != nil {
			p.Party = append(p.Party, characterView(g, m))
		}
	}
	if v.Screen == "menu" {
		if !slices.Contains(v.Actions, app.ActReorder) {
			p.Unavailable = append(p.Unavailable, Unavailable{app.ActReorder, "A party of one has no order to change."})
		}
		p.Unavailable = append(p.Unavailable, Unavailable{app.ActDisband, errDisbandUnavailable.Error()})
	}
	return p
}
