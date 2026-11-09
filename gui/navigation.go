package main

// Town navigation (Castle-0). The player walks between the castle's
// locations as in the original: the Castle is the hub; the Tavern, the
// Inn, Boltac's and the Temple are inside it; the Training Grounds, the
// maze and Settings (Istary's Utilities) are at the Edge of Town; leaving
// the Training Grounds, the Tavern or Settings returns to the Castle
// (Player's Guide pp.22, 36, 38).
//
// The moves are the controller's (app.Controller.GoTo, the terminal's menu
// keys); Settings is the window's own and has no place in the game, so it
// is kept here while the game stands at the Edge of Town. Navigate never
// saves (the town location is not part of the save) and draws no random
// number. It is refused while an expedition or practice is under way and
// while a character is being created; an open town prompt is closed first,
// as the game's own cancel steps would. Boltac's and the Inn are left by
// RETURN at every step (app.Controller.LeaveShop, LeaveInn), as their keys
// leave them.

import (
	"errors"
	"slices"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// places names the town locations for the front end.
var places = []struct {
	loc  engine.TownLocation
	name string
	full string // for refusals
}{
	{engine.Castle, "castle", "The Castle"},
	{engine.Tavern, "tavern", "Gilgamesh's Tavern"},
	{engine.Inn, "inn", "The Adventurer's Inn"},
	{engine.Trading, "trading", "Boltac's Trading Post"},
	{engine.Temple, "temple", "The Temple of Cant"},
	{engine.EdgeOfTown, "edge", "The Edge of Town"},
	{engine.Training, "training", "The Training Grounds"},
}

// settingsPlace is Settings, at the Edge of Town; "maze" is where the party
// is during an expedition or practice.
const settingsPlace = "settings"

func placeOf(loc engine.TownLocation) (name, full string) {
	for _, p := range places {
		if p.loc == loc {
			return p.name, p.full
		}
	}
	return "", ""
}

// ExitView is one way out of the current location: open, or why not.
type ExitView struct {
	Dest   string `json:"dest"`
	Open   bool   `json:"open"`
	Reason string `json:"reason"`
}

var (
	errNotAnExit   = errors.New("You cannot go there from here.")
	errCreating    = errors.New("Finish or discard the character being created first.")
	errPracticeOut = errors.New("A practice is under way. Finish or abandon it first.")
)

// notInWindow is why a location the game would open is not open here yet.
func notInWindow(full string) string {
	return full + " is not open in Istary's window yet."
}

// Navigate goes to dest, one of the current location's exits (View.Exits).
func (s *Session) Navigate(dest string) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	return s.view(s.navigate(dest))
}

func (s *Session) navigate(dest string) error {
	if err := s.navigationRefusal(); err != nil {
		return err
	}
	i := slices.IndexFunc(s.exits(), func(e ExitView) bool { return e.Dest == dest })
	if i < 0 {
		return errNotAnExit
	}
	if e := s.exits()[i]; !e.Open {
		return errors.New(e.Reason)
	}
	c := s.ctrl
	if c.Game.Town.Location == engine.Trading {
		return c.LeaveShop() // its one exit, the Castle, from any step
	}
	if c.Game.Town.Location == engine.Inn {
		return c.LeaveInn() // likewise
	}
	if err := s.settlePrompt(); err != nil {
		return err
	}
	switch {
	case s.inSettings():
		s.settings = false
		return c.LeaveEdgeOfTown()
	case dest == settingsPlace:
		s.settings = true
		return nil
	case dest == "trading":
		s.receipt = nil // a new visit
	case dest == "inn":
		s.innLast = nil
	case dest == "tavern":
		s.goldLast = nil
	}
	for _, p := range places {
		if p.name == dest {
			return c.GoTo(p.loc)
		}
	}
	return errNotAnExit
}

// ReturnView is the way back to the Castle from a town place (the frame's
// "← Castle", and Escape): Note says what returning abandons, if anything.
type ReturnView struct {
	Open bool   `json:"open"`
	Note string `json:"note"`
}

// returnView is nil at the Castle and out of town.
func (s *Session) returnView() *ReturnView {
	g := s.ctrl.Game
	switch {
	case s.active() != nil:
		return nil
	case g.Phase == engine.PhaseCreation:
		return &ReturnView{Open: true, Note: "Returning discards the character being created."}
	case g.Phase != engine.PhaseTown || (g.Town.Location == engine.Castle && !s.inSettings()):
		return nil
	case g.Town.InputMode == engine.InputTavernPassword:
		return &ReturnView{Open: true, Note: "Returning cancels the join waiting for its password."}
	case g.Town.Location == engine.Trading && g.Town.ShopStep == engine.ShopBuyConfirm:
		return &ReturnView{Open: true, Note: "Returning answers no to Boltac's question: the item is not bought."}
	case g.Town.Location == engine.Trading && g.Town.ShopStep == engine.ShopSellConfirm:
		return &ReturnView{Open: true, Note: "Returning answers no to Boltac's question: the item is not sold."}
	}
	return &ReturnView{Open: true}
}

// ReturnToCastle goes back to the Castle from anywhere in town, as L)EAVE
// does from each place: what is unfinished there is abandoned first, by
// the game's own steps (a character being created is discarded, an open
// prompt is cancelled), and nothing already done is undone. At the Castle
// it does nothing. It is refused out of town (an expedition, practice), and
// like any move it writes nothing and draws no random number.
func (s *Session) ReturnToCastle() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	return s.view(s.returnToCastle())
}

func (s *Session) returnToCastle() error {
	switch {
	case s.real:
		return errExpeditionOut
	case s.practice != nil:
		return errPracticeOut
	}
	c := s.ctrl
	if c.Game.Phase == engine.PhaseCreation {
		if err := c.AbandonCreation(); err != nil {
			return err
		}
		s.notice = "The character being created was discarded."
	}
	if c.Game.Phase != engine.PhaseTown {
		return app.ErrWrongState
	}
	if c.Game.Town.Location == engine.Trading {
		// RETURN at every step, as the keys do: a question waiting is
		// answered no, the customer leaves, then Boltac's.
		return c.LeaveShop()
	}
	if c.Game.Town.Location == engine.Inn {
		return c.LeaveInn() // the guest leaves, then the Inn
	}
	if err := s.settlePrompt(); err != nil {
		return err
	}
	if s.inSettings() {
		s.settings = false
		return c.LeaveEdgeOfTown()
	}
	if c.Game.Town.Location == engine.Castle {
		return nil
	}
	return c.GoTo(engine.Castle)
}

// navigationRefusal is why the player cannot walk around town now, or nil.
func (s *Session) navigationRefusal() error {
	switch {
	case s.real:
		return errExpeditionOut
	case s.practice != nil:
		return errPracticeOut
	case s.ctrl.Game.Phase == engine.PhaseCreation:
		return errCreating
	case s.ctrl.Game.Phase != engine.PhaseTown:
		return app.ErrWrongState
	}
	return nil
}

// inSettings reports whether the player is in Settings: it was opened at
// the Edge of Town, and the game still stands there.
func (s *Session) inSettings() bool {
	g := s.ctrl.Game
	return s.settings && g.Phase == engine.PhaseTown && g.Town.Location == engine.EdgeOfTown
}

// location is where the player is: a town location, "settings", or "maze"
// during an expedition or practice. A character is created at the Training
// Grounds.
func (s *Session) location() string {
	g := s.ctrl.Game
	switch {
	case s.active() != nil:
		return "maze"
	case g.Phase == engine.PhaseCreation:
		return "training"
	case s.inSettings():
		return settingsPlace
	}
	name, _ := placeOf(g.Town.Location)
	return name
}

// exits lists the ways out of the current location. In the maze there are
// none (the party comes back by the game's own ways); while a character is
// being created they are shown closed.
func (s *Session) exits() []ExitView {
	if s.active() != nil {
		return nil
	}
	if s.inSettings() {
		return []ExitView{{Dest: "castle", Open: true}}
	}
	var out []ExitView
	for _, e := range s.ctrl.Exits() {
		name, full := placeOf(e.To)
		v := ExitView{Dest: name, Open: e.Refusal == "", Reason: e.Refusal}
		switch e.To {
		case engine.Temple:
			if v.Open {
				v.Open, v.Reason = false, notInWindow(full)
			} else {
				v.Reason += " " + notInWindow(full) // and the game's own answer first
			}
		}
		out = append(out, v)
	}
	if s.ctrl.Game.Town.Location == engine.EdgeOfTown {
		out = append(out, ExitView{Dest: settingsPlace, Open: true})
	}
	if err := s.navigationRefusal(); err != nil {
		for i := range out {
			out[i].Open, out[i].Reason = false, err.Error()
		}
	}
	return out
}

// PartyBarSlot is one of the six party positions for the party panel shown
// on every screen. Empty positions have no name.
type PartyBarSlot struct {
	Slot     int    `json:"slot"`  // 1-based; 1..FrontRow are the front row
	Index    int    `json:"index"` // roster position in the saved game, or -1
	Name     string `json:"name"`
	Class    string `json:"class"` // alignment and class, e.g. "G-FIG"
	Level    int    `json:"level"`
	AC       int    `json:"ac"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	Status   string `json:"status"`
	Poisoned bool   `json:"poisoned"`
	Front    bool   `json:"front"`
}

// partyBar is the party of the game being played: the practice copies
// during practice, otherwise the saved game's party.
func (s *Session) partyBar() []PartyBarSlot {
	g := s.ctrl.Game
	if s.practice != nil {
		g = s.practice.Game
	}
	members := g.Town.Party.Members
	out := make([]PartyBarSlot, app.MaxPartySize)
	for i := range out {
		out[i] = PartyBarSlot{Slot: i + 1, Index: -1, Front: i < app.FrontRowSize}
		if i >= len(members) || members[i] == nil {
			continue
		}
		m := members[i]
		out[i].Name, out[i].Class, out[i].Level, out[i].AC = m.Name, engine.AlignClass(m.Alignment, m.Class), m.Level, m.AC
		out[i].HP, out[i].MaxHP, out[i].Status, out[i].Poisoned = m.HP, m.MaxHP, m.Status.String(), m.PoisonAmt > 0
		if g == s.ctrl.Game {
			out[i].Index = rosterIndex(g, m)
		}
	}
	return out
}

// rosterIndex is ch's position on g's roster, or -1.
func rosterIndex(g *engine.GameState, ch *engine.Character) int {
	return slices.Index(g.Town.Roster.Characters, ch)
}
