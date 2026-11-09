package main

// Closing the window (its close button, ⌘Q or Quit in the menu) while the
// last save has failed. Leave the game refuses then (edge.go); closing the
// window asks instead, since a window must be able to close: keep playing,
// try saving again, or quit and lose what is unsaved. Nothing closes an
// unsaved game without the player choosing to.

import "fmt"

// The answers to the closing question, as its buttons.
const (
	closeKeepPlaying = "Keep playing"
	closeSaveAgain   = "Try saving again"
	closeDiscard     = "Quit without saving"
)

// closeTitle heads the closing question.
const closeTitle = "The last change is not saved"

// closeMessage is the closing question, for the last save's error.
func closeMessage(unsaved string) string {
	return fmt.Sprintf("Istary could not save (%s). What this window shows since the last save is lost if Istary quits now.\n\n"+
		"Try saving again writes it as it is: nothing is bought, sold or moved twice.", unsaved)
}

// beforeClose is the window's OnBeforeClose: whether to keep the window
// open (prevent). With everything saved it closes at once. Otherwise it
// asks, with ask, until the player keeps playing, quits without saving, or
// a save succeeds; a save writes the game as the window has it (SaveNow),
// repeating nothing. If the question cannot be asked the window stays
// open. A close while the question is open is refused without asking
// again, and a later close asks afresh.
func (s *Session) beforeClose(ask func(message string) (string, error)) (prevent bool) {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return true
	}
	if s.startErr != nil || s.unsaved == "" {
		s.mu.Unlock()
		return false
	}
	s.closing = true
	unsaved := s.unsaved
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.closing = false
		s.mu.Unlock()
	}()
	for {
		answer, err := ask(closeMessage(unsaved))
		if err != nil {
			return true
		}
		switch answer {
		case closeDiscard:
			return false
		case closeSaveAgain:
			v := s.SaveNow()
			if v.Unsaved == "" {
				return false
			}
			unsaved = v.Unsaved
		default:
			return true
		}
	}
}
