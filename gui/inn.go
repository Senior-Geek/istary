package main

// The Adventurer's Inn (I-4). The Inn screen calls one method, InnAction,
// with one of the requests below; it runs the controller's Inn operation
// (app/inn_ops.go), which presses the terminal's keys and runs its heal
// ticks. View.Inn is the controller's InnView: every room, rate, quote,
// limit and refusal comes from it, so the screen decides nothing; InnQuote
// quotes a chosen number of weeks the same way, changing nothing.
//
// A stay is one action: the room, every week of it and the level check
// after it happen together in Go and are saved together, in one atomic
// write of the roster, so no stay is ever half saved and closing the
// window between its steps cannot undo or reroll its level check (there
// are no steps between). Pooling the party's gold to the guest is one
// action and one save too. Choosing or switching the guest, a refusal and
// looking write nothing, and so does a stay that changes nothing (a night
// in the stables short of a level); its receipt still shows.
//
// A save that fails does not undo the stay: it has happened in this window
// and stays (session.go save); the View says it is not saved (Unsaved, the
// receipt's Saved), Try saving again (SaveNow) writes it as it is, and the
// window's close asks first (close.go). The Inn is refused during
// practice and expeditions. Healing draws no random number; the level
// check does, once per stay, as the terminal's does.

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// Inn actions (InnRequest.Action).
const (
	InnActEnter      = "enter"       // Slot: the party member who stays (0-based), in place of any guest
	InnActLeaveGuest = "leave-guest" // back to "WHO WILL STAY"
	InnActStay       = "stay"        // Room (engine.InnRoom), Weeks (0: until healed or out of gold)
	InnActPool       = "pool"        // the party's gold to the guest (Pool Gold)
)

// InnRequest is an Inn action chosen in the front end.
type InnRequest struct {
	Action string `json:"action"`
	Slot   int    `json:"slot"`
	Room   int    `json:"room"`
	Weeks  int    `json:"weeks"`
}

// InnPanel is the Inn for the front end: the controller's InnView and the
// last stay or pooling made there.
type InnPanel struct {
	View app.InnView `json:"view"`
	// Last is what the last stay or pooling completed, until the next one,
	// a change of guest or the next visit; a refusal keeps it.
	Last *InnReceipt `json:"last"`
}

// InnReceipt is a completed stay or pooling, and whether it is saved.
type InnReceipt struct {
	Stay *app.InnStay  `json:"stay"` // a stay, or nil
	Gold *app.GoldMove `json:"gold"` // a pooling, or nil
	// Saved and Durable as for Boltac's receipts (ShopReceipt).
	Saved   bool `json:"saved"`
	Durable bool `json:"durable"`
}

var (
	errNoGuest = errors.New("Choose who will stay at the Inn first.")
	errNoRoom  = errors.New("The Inn has no such room.")
)

// InnAction performs one of the Inn's actions.
func (s *Session) InnAction(r InnRequest) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed() {
		return s.view(nil)
	}
	s.notice = ""
	if err := s.innRefusal(); err != nil {
		return s.view(err)
	}
	c := s.ctrl
	roster := rosterBytes(c.Game)
	last, err := innOp(c, r)
	var saveErr error
	if !bytes.Equal(roster, rosterBytes(c.Game)) {
		saveErr = s.save()
	}
	if err == nil && (r.Action == InnActEnter || r.Action == InnActLeaveGuest) {
		s.innLast = nil // another guest, or none: the last receipt was theirs
	}
	if last != nil {
		// A stay or a pooling that changed nothing on disk (a night in
		// the stables short of a level) still has its receipt: it is
		// saved exactly when nothing else waits to be. A refusal keeps
		// the last receipt (a death of old age stays in view).
		last.Saved = saveErr == nil && s.unsaved == ""
		last.Durable = last.Saved && s.uncertain == ""
		s.innLast = last
		s.notice = innNotice(last)
	}
	return s.view(errors.Join(err, saveErr))
}

// InnQuote quotes the guest's stay in room for weeks (0: until healed), as
// a stay would take it. It changes nothing and draws no random number; its
// Refusal says why such a stay cannot be chosen.
func (s *Session) InnQuote(room, weeks int) (app.InnQuote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed() {
		return app.InnQuote{}, s.closedErr()
	}
	if err := s.innRefusal(); err != nil {
		return app.InnQuote{}, err
	}
	q, err := s.ctrl.Quote(engine.InnRoom(room), weeks)
	if errors.Is(err, app.ErrInvalidChoice) {
		return q, errNoRoom
	}
	if errors.Is(err, app.ErrWrongState) {
		return q, errNoGuest
	}
	return q, err
}

// innRefusal is why no Inn action can be taken now, or nil.
func (s *Session) innRefusal() error {
	if err := s.navigationRefusal(); err != nil {
		return err // an expedition, practice, a character being created
	}
	if s.ctrl.Game.Town.Location != engine.Inn {
		return notAt(engine.Inn)
	}
	return nil
}

// innOp runs r on c, which is at the Inn.
func innOp(c *app.Controller, r InnRequest) (*InnReceipt, error) {
	v, err := c.InnView()
	if err != nil {
		return nil, err
	}
	switch r.Action {
	case InnActEnter:
		switch {
		case r.Slot == v.Guest:
			return nil, nil // already staying
		case !isMember(c.Game, r.Slot):
			return nil, app.ErrNoSuchPartyMember
		case v.Step != "who":
			if err := c.InnLeaveGuest(); err != nil {
				return nil, err
			}
		}
		return nil, c.InnEnter(r.Slot)
	case InnActLeaveGuest:
		if v.Guest < 0 {
			return nil, errNoGuest
		}
		return nil, c.InnLeaveGuest()
	case InnActStay:
		if v.Guest < 0 {
			return nil, errNoGuest
		}
		st, err := c.Stay(engine.InnRoom(r.Room), r.Weeks)
		if errors.Is(err, app.ErrInvalidChoice) {
			return nil, errNoRoom
		}
		if err != nil {
			return nil, err
		}
		return &InnReceipt{Stay: st}, nil
	case InnActPool:
		if v.Guest < 0 {
			return nil, errNoGuest
		}
		m, err := c.PoolGold(v.Guest)
		if err != nil {
			return nil, err
		}
		return &InnReceipt{Gold: m}, nil
	}
	return nil, app.ErrInvalidChoice
}

// innNotice says what was done, and whether it is saved.
func innNotice(r *InnReceipt) string {
	var did string
	switch {
	case r.Stay != nil:
		did = stayWords(r.Stay)
	case r.Gold != nil:
		did = poolWords(r.Gold)
	}
	return did + savedWords(r.Saved, r.Durable)
}

func stayWords(st *app.InnStay) string {
	var b strings.Builder
	if st.Weeks == 0 {
		fmt.Fprintf(&b, "%s spent a night in %s", st.Guest, strings.ToLower(st.RoomName))
	} else {
		fmt.Fprintf(&b, "%s stayed %d week%s in %s for %d GP: HP %d → %d", st.Guest, st.Weeks, map[bool]string{true: "", false: "s"}[st.Weeks == 1],
			strings.ToLower(st.RoomName), st.Charged, st.HPBefore, st.HPAfter)
	}
	switch {
	case st.Died:
		b.WriteString(", made a level and died of old age")
	case st.LevelAfter > st.LevelBefore:
		fmt.Fprintf(&b, ", and made level %d", st.LevelAfter)
	}
	b.WriteString(".")
	return b.String()
}

func poolWords(m *app.GoldMove) string {
	if m.To < 0 {
		return fmt.Sprintf("The party's %d GP were shared out.", m.Total)
	}
	return fmt.Sprintf("The party's gold was pooled to %s: %d GP.", m.Names[m.To], m.After[m.To])
}

// innPanel is the Inn for the View, or nil away from it.
func (s *Session) innPanel() *InnPanel {
	if s.active() != nil {
		return nil
	}
	v, err := s.ctrl.InnView()
	if err != nil {
		return nil
	}
	return &InnPanel{View: v, Last: s.innLast}
}
