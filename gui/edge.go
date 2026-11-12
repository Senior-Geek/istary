package main

// The Edge of Town (Castle-0 C0-4): where expeditions set out and the
// player leaves the game. The original offers M)AZE, T)RAINING GROUNDS,
// C)ASTLE, U)TILITIES and L)EAVE GAME here (Player's Guide p.36).

import (
	"errors"

	"github.com/Senior-Geek/istary/engine"
)

// DeparturesView says whether each departure from the Edge of Town is
// open now: "" if it is, otherwise why not, in the words its action would
// refuse with.
type DeparturesView struct {
	Real     string `json:"real"`     // RealExpedition
	Practice string `json:"practice"` // PracticeExpedition and PracticeEncounter
}

func (s *Session) departures() DeparturesView {
	var d DeparturesView
	if err := s.realRefusal(); err != nil {
		d.Real = err.Error()
	}
	if err := s.practiceRefusal(); err != nil {
		d.Practice = err.Error()
	}
	return d
}

// setQuit gives the session the way to close the window (Wails's Quit),
// once the window has started.
func (s *Session) setQuit(quit func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quit = quit
}

var errUnsavedLeave = errors.New("The last action is not saved yet. Istary tries again with every action; leave once a save succeeds, or the game since the last save is lost.")

// LeaveGame leaves Istary from the Edge of Town (L)EAVE GAME). Every action
// is already saved when it happens, so leaving only closes the window,
// which lets go of the save folder (OnShutdown). It is refused anywhere
// else, and while the last save has failed.
func (s *Session) LeaveGame() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed() {
		return s.view(nil)
	}
	s.notice = ""
	if err := s.navigationRefusal(); err != nil {
		return s.view(err)
	}
	if err := s.at(engine.EdgeOfTown); err != nil {
		return s.view(err)
	}
	if s.unsaved != "" {
		return s.view(errUnsavedLeave)
	}
	if s.quit == nil {
		return s.view(errors.New("This window cannot be closed from here; close it with the window's own button."))
	}
	// Quitting runs OnShutdown, which takes the session's lock: not under it.
	go s.quit()
	s.notice = "Leaving Istary."
	return s.view(nil)
}
