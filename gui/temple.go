package main

// The Temple of Cant (T-3). The Temple screen calls one method,
// TempleAction, with one of the requests below; it runs the controller's
// Temple operation (app/temple_ops.go), which presses the terminal's keys.
// View.Temple is the controller's TempleView: the patients, their
// conditions, donations, refusals and funding come from it, so the screen
// decides nothing; it never shows a chance of success.
//
// A treatment is one action: the donation, the treatment and its roll
// happen together in Go and are saved together, in one atomic write of the
// roster, so no treatment is ever half saved and closing the window cannot
// undo or roll it again. Pooling the party's gold to a payer is one action
// and one save too. The screen's confirmation is its own and calls nothing,
// so cancelling changes nothing and draws nothing; a refusal and looking
// write nothing.
//
// A save that fails does not undo the treatment: it has happened in this
// window and stays (session.go save); the View says it is not saved
// (Unsaved, the receipt's Saved), Try saving again (SaveNow) writes it as
// it is, charging and rolling nothing again, and the window's close asks
// first (close.go). The Temple is refused during practice and
// expeditions.

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// Temple actions (TempleRequest.Action).
const (
	TempleActTreat = "treat" // Patient (roster index), Payer (party slot, 0-based)
	TempleActPool  = "pool"  // Payer: give that member every other member's gold (Pool Gold)
)

// TempleRequest is a Temple action chosen in the front end.
type TempleRequest struct {
	Action  string `json:"action"`
	Patient int    `json:"patient"`
	Payer   int    `json:"payer"`
}

// TemplePanel is the Temple for the front end: the controller's TempleView
// and the last treatment or pooling made there.
type TemplePanel struct {
	View app.TempleView `json:"view"`
	// Last is what the last treatment or pooling completed, until the next
	// one or the next visit; a refusal keeps it.
	Last *TempleLast `json:"last"`
}

// TempleLast is a completed treatment or pooling, and whether it is saved.
type TempleLast struct {
	Treat   *app.TempleReceipt `json:"treat"` // a treatment, or nil
	Gold    *app.GoldMove      `json:"gold"`  // a pooling, or nil
	Saved   bool               `json:"saved"`
	Durable bool               `json:"durable"`
}

var errNoTempleAction = errors.New("The Temple has no such service.")

// TempleAction performs one of the Temple's actions.
func (s *Session) TempleAction(r TempleRequest) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	if err := s.templeRefusal(); err != nil {
		return s.view(err)
	}
	c := s.ctrl
	roster := rosterBytes(c.Game)
	var last *TempleLast
	var err error
	switch r.Action {
	case TempleActTreat:
		var t *app.TempleReceipt
		if t, err = c.Treat(r.Patient, r.Payer, false); err == nil {
			last = &TempleLast{Treat: t}
		}
	case TempleActPool:
		var m *app.GoldMove
		if m, err = c.PoolGold(r.Payer); err == nil {
			last = &TempleLast{Gold: m}
		}
	default:
		err = errNoTempleAction
	}
	var saveErr error
	if !bytes.Equal(roster, rosterBytes(c.Game)) {
		saveErr = s.save()
	}
	if last != nil {
		last.Saved = saveErr == nil && s.unsaved == ""
		last.Durable = last.Saved && s.uncertain == ""
		s.templeLast = last
		s.notice = templeNotice(last)
	}
	return s.view(errors.Join(err, saveErr))
}

// templeRefusal is why no Temple action can be taken now, or nil.
func (s *Session) templeRefusal() error {
	if err := s.navigationRefusal(); err != nil {
		return err
	}
	if s.ctrl.Game.Town.Location != engine.Temple {
		return notAt(engine.Temple)
	}
	return nil
}

// templeNotice says what was done, and whether it is saved.
func templeNotice(l *TempleLast) string {
	var did string
	switch t := l.Treat; {
	case t != nil && t.Healed:
		did = fmt.Sprintf("%s is well again. %s tithed %d GP.", t.Patient, t.Payer, t.Donation)
	case t != nil && t.StatusAfter == "ASHED":
		did = fmt.Sprintf("The priests could not raise %s, who is now ashes. %s tithed %d GP.", t.Patient, t.Payer, t.Donation)
	case t != nil:
		did = fmt.Sprintf("%s could not be restored and is lost forever. %s tithed %d GP.", t.Patient, t.Payer, t.Donation)
	case l.Gold != nil:
		did = poolWords(l.Gold)
	}
	return did + savedWords(l.Saved, l.Durable)
}

// templePanel is the Temple for the View, or nil away from it.
func (s *Session) templePanel() *TemplePanel {
	if s.active() != nil {
		return nil
	}
	v, err := s.ctrl.TempleView()
	if err != nil {
		return nil
	}
	return &TemplePanel{View: v, Last: s.templeLast}
}
