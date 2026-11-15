package main

// Castle-first startup (NAV-1A; docs/design/nav1/NAV-1A.md). A library
// window (games.go) always opens at the Castle. The game last chosen,
// remembered in the window's settings (settings.go, Settings.LastGame),
// is opened again if its slot still holds a game that opens; an
// expedition under way resumes in the maze, as on Continue. Otherwise no
// game is open, and the window stands at the Castle all the same.
//
// With no game open the player can only walk between the Castle, the
// Edge of Town and Settings, which are the window's here and not the
// game's: the Castle's other destinations and the Training Grounds are
// shown closed, and so is Enter the Maze. Select Game, at the Edge,
// shows game selection, and Leave the Game closes the window. Nothing
// here reaches the engine: every game action still finds no game open
// (Session.closed) and does nothing, and no save is read or written but
// the slots' summaries, as game selection always read them.

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Senior-Geek/istary/app"
)

var (
	// errChooseGame is a destination's or a departure's reason with no
	// game open.
	errChooseGame = errors.New("Select a game first: Select Game is at the Edge of Town.")
	// errChoosing refuses walking about while game selection is shown.
	errChoosing = errors.New("Choose a game, or return to the Castle.")
)

// lobbyExits are the ways out of where the player stands with no game
// open, in the order the game's own exits have.
var lobbyExits = map[string][]ExitView{
	"castle": {
		{Dest: "tavern", Reason: errChooseGame.Error()},
		{Dest: "inn", Reason: errChooseGame.Error()},
		{Dest: "trading", Reason: errChooseGame.Error()},
		{Dest: "temple", Reason: errChooseGame.Error()},
		{Dest: "edge", Open: true},
	},
	"edge": {
		{Dest: "training", Reason: errChooseGame.Error()},
		{Dest: "castle", Open: true},
		{Dest: settingsPlace, Open: true},
	},
	settingsPlace: {{Dest: "castle", Open: true}},
}

// inLobby reports whether the window is open with no game: a library
// window that has opened its folder, no game open.
func (s *Session) inLobby() bool {
	return s.startErr == nil && s.library && s.ctrl == nil
}

// openLastGame opens the game last chosen, on starting. A slot that no
// longer holds a game that opens is left as it is, and the window stays
// at the Castle with no game open, saying why.
func (s *Session) openLastGame() {
	slot := s.prefs.LastGame
	if slot == 0 {
		return
	}
	if slot < 1 || slot > gameSlots {
		s.notice = "No game is open: the last game chosen is not one of the six slots. Select Game, at the Edge of Town, chooses one."
		return
	}
	switch v := s.slotView(slot); v.State {
	case "empty":
		s.notice = fmt.Sprintf("No game is open: game %d, the last chosen, is no longer in its slot. Select Game, at the Edge of Town, chooses one.", slot)
	case "damaged":
		s.notice = fmt.Sprintf("No game is open: game %d, the last chosen, cannot be opened: %s. It is left as it is.", slot, v.Problem)
	default:
		if err := s.openGame(slot); err != nil {
			s.notice = "No game is open. " + err.Error()
		}
	}
}

// remember keeps slot as the game to open on starting. It is the
// window's setting, not the game's; if it cannot be written, the view
// says so and the game is open all the same.
func (s *Session) remember(slot int) {
	if s.prefs.LastGame == slot {
		return
	}
	s.prefs.LastGame = slot
	if err := s.saveSettings(); err != nil {
		s.prefsNote = fmt.Sprintf("The window's settings could not be saved (%v); Istary may not open this game next time.", err)
	}
}

// lobbyView fills v for the window with no game open: where the player
// stands, its ways out, an empty party, and every departure closed.
func (s *Session) lobbyView(v *View) {
	v.Location = s.lobby
	v.Exits = slices.Clone(lobbyExits[s.lobby])
	if s.lobby != "castle" {
		v.Return = &ReturnView{Open: true}
	}
	v.PartyBar = make([]PartyBarSlot, app.MaxPartySize)
	for i := range v.PartyBar {
		v.PartyBar[i] = PartyBarSlot{Slot: i + 1, Index: -1, Front: i < app.FrontRowSize}
	}
	v.PartyMax, v.FrontRow = app.MaxPartySize, app.FrontRowSize
	v.Departures = DeparturesView{Real: errChooseGame.Error(), Practice: errChooseGame.Error()}
	v.Story = storyFor(s.lobby, false)
}

// lobbyNavigate goes to dest with no game open.
func (s *Session) lobbyNavigate(dest string) error {
	if s.choosing {
		return errChoosing
	}
	i := slices.IndexFunc(lobbyExits[s.lobby], func(e ExitView) bool { return e.Dest == dest })
	if i < 0 {
		return errNotAnExit
	}
	if e := lobbyExits[s.lobby][i]; !e.Open {
		return errors.New(e.Reason)
	}
	s.lobby = dest
	return nil
}

// lobbyReturn is ReturnToCastle with no game open: from game selection
// (Escape there too) or the Edge of Town or Settings.
func (s *Session) lobbyReturn() {
	s.choosing, s.replacing, s.lobby = false, 0, "castle"
}
