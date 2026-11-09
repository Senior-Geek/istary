package main

// Sharing and pooling the party's gold in town (I-3, app/gold.go). The
// Tavern's D)IVVY GOLD shares it evenly (Player's Guide p.24); P)OOL GOLD
// gives one member all of it (p.28; at the Inn it is InnAction "pool").
// Each is one action and one save, like any change in town; a refusal
// writes nothing. The rules (who takes part, where a remainder goes) are
// the controller's.

import (
	"bytes"
	"errors"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// Gold actions (GoldRequest.Action).
const (
	GoldActDivvy = "divvy" // share the party's gold evenly (the Tavern)
	GoldActPool  = "pool"  // Slot: give that member everyone's gold (the Tavern, Boltac's, the Inn)
)

// GoldRequest is a gold action chosen in the front end.
type GoldRequest struct {
	Action string `json:"action"`
	Slot   int    `json:"slot"`
}

// GoldReceipt is a completed pooling or sharing, and whether it is saved.
type GoldReceipt struct {
	Move    *app.GoldMove `json:"move"`
	Saved   bool          `json:"saved"`
	Durable bool          `json:"durable"`
}

// GoldAction pools or shares the party's gold.
func (s *Session) GoldAction(r GoldRequest) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice, s.goldLast = "", nil
	if err := s.navigationRefusal(); err != nil {
		return s.view(err)
	}
	c := s.ctrl
	roster := rosterBytes(c.Game)
	var m *app.GoldMove
	var err error
	switch r.Action {
	case GoldActDivvy:
		m, err = c.DivvyGold()
		if errors.Is(err, app.ErrWrongState) {
			err = notAt(engine.Tavern)
		}
	case GoldActPool:
		m, err = c.PoolGold(r.Slot)
		if errors.Is(err, app.ErrWrongState) {
			err = errors.New("Gold is pooled at Gilgamesh's Tavern, Boltac's Trading Post or the Adventurer's Inn, with a party.")
		}
	default:
		err = app.ErrInvalidChoice
	}
	if err != nil || bytes.Equal(roster, rosterBytes(c.Game)) {
		return s.view(err) // a refusal, or nothing moved: nothing to save
	}
	saveErr := s.save()
	s.goldLast = &GoldReceipt{Move: m, Saved: saveErr == nil, Durable: saveErr == nil && s.uncertain == ""}
	s.notice = poolWords(m) + savedWords(s.goldLast.Saved, s.goldLast.Durable)
	return s.view(saveErr)
}

// goldPanel is the last pooling or sharing, at the Tavern only.
func (s *Session) goldPanel() *GoldReceipt {
	if s.active() != nil || s.ctrl.Game.Phase != engine.PhaseTown || s.ctrl.Game.Town.Location != engine.Tavern {
		return nil
	}
	return s.goldLast
}
