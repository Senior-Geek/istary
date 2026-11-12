package main

// Equipping in town (M8). The character sheet equips and takes off a
// roster character's items one at a time with EquipAction, which runs the
// controller's TownEquip and TownUnequip: the equip walk's rules, one item
// at a time (app/equip.go). Each item on the sheet carries the controller's
// answer before it is asked (ItemView), so the sheet decides nothing.
//
// It is open anywhere in town, as the sheet is, and not during an
// expedition or practice or while a character is being created. A change
// is saved at once, in one atomic write, as Boltac's transactions are; a
// refusal writes nothing. A failed save keeps the change in the window and
// says it is not saved (Unsaved; the notice). Nothing draws a random
// number.

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/Senior-Geek/istary/app"
)

// EquipRequest equips (Equip) or takes off item N (1-based) of the roster
// character at Index.
type EquipRequest struct {
	Index int  `json:"index"`
	N     int  `json:"n"`
	Equip bool `json:"equip"`
}

// EquipAction equips or takes off one of a roster character's items.
func (s *Session) EquipAction(r EquipRequest) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed() {
		return s.view(nil)
	}
	s.notice = ""
	if err := s.equipRefusal(); err != nil {
		return s.view(err)
	}
	c := s.ctrl
	before, err := c.Equipment(r.Index)
	if err != nil {
		return s.view(err)
	}
	roster := rosterBytes(c.Game)
	var cursed bool
	if r.Equip {
		cursed, err = c.TownEquip(r.Index, r.N)
	} else {
		err = c.TownUnequip(r.Index, r.N)
	}
	if err != nil || bytes.Equal(roster, rosterBytes(c.Game)) {
		return s.view(err)
	}
	saveErr := s.save()
	ch := c.Game.Town.Roster.Characters[r.Index]
	name := before[r.N-1].Name
	did := fmt.Sprintf("%s took off the %s. AC %d.", ch.Name, name, ch.AC)
	if r.Equip {
		did = fmt.Sprintf("%s equipped the %s. AC %d.", ch.Name, name, ch.AC)
		if cursed {
			did += " ** CURSED **: it is stuck; only Boltac can remove it."
		}
	}
	s.notice = did + savedWords(saveErr == nil, s.uncertain == "")
	return s.view(saveErr)
}

// savedWords says whether a change was saved: " Saved.", saved perhaps not
// durably, or NOT saved (it exists only in this window until a save
// succeeds).
func savedWords(saved, durable bool) string {
	switch {
	case !saved:
		return " It is NOT saved: it exists only in this window until a save succeeds."
	case !durable:
		return " Saved, but perhaps not durably."
	}
	return " Saved."
}

// equipRefusal is why nobody can equip in town now, or nil: during an
// expedition or practice, or while a character is being created; and at
// Boltac's while he waits for an answer (UX-2: the Trading Post equips
// beside his counter, and his question is about an item as it is worn).
func (s *Session) equipRefusal() error {
	if err := s.navigationRefusal(); err != nil {
		return err
	}
	if v, err := s.ctrl.ShopView(); err == nil && asking(v) {
		return errAnswerBoltac
	}
	return nil
}

// equipNote is equipRefusal in words for the sheet, or "".
func (s *Session) equipNote() string {
	if err := s.equipRefusal(); err != nil {
		return errorText(err)
	}
	return ""
}

// withEquipment fills cv's items with the controller's answers for
// equipping the roster character at index, when equipping is open.
func (s *Session) withEquipment(cv *CharacterView, index int) {
	if cv.EquipNote = s.equipNote(); cv.EquipNote != "" {
		return
	}
	eq, err := s.ctrl.Equipment(index)
	if err != nil {
		var r *app.Refusal
		if errors.As(err, &r) {
			cv.EquipNote = r.Message
		} else {
			cv.EquipNote = errorText(err)
		}
		return
	}
	for i, e := range eq {
		if i < len(cv.Items) {
			cv.Items[i].N, cv.Items[i].Kind = e.N, e.Kind
			cv.Items[i].EquipRefusal, cv.Items[i].UnequipRefusal = e.EquipRefusal, e.UnequipRefusal
		}
	}
}
