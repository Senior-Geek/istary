package main

import (
	"errors"
	"fmt"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// The game has one current town location, and the player walks between
// locations with Navigate (navigation.go); nothing else moves the game in
// town. An action that belongs to a location is accepted only there: the
// party is formed at the Tavern, characters are created, renamed and
// deleted at the Training Grounds, expeditions and practice set out from
// the Edge of Town, and the roster is reloaded, reset and restored in
// Settings.

// at accepts an action that belongs to loc (the Tavern, the Training
// Grounds or the Edge of Town) only while the player is there, and closes
// any town prompt open there with the game's own cancel steps.
func (s *Session) at(loc engine.TownLocation) error {
	g := s.ctrl.Game
	if g.Phase != engine.PhaseTown {
		return app.ErrWrongState // e.g. a character is being created
	}
	if g.Town.Location != loc || s.inSettings() {
		return notAt(loc)
	}
	return s.settlePrompt()
}

// inSettingsNow accepts a Settings action only while the player is in
// Settings.
func (s *Session) inSettingsNow() error {
	if !s.inSettings() {
		return errNotInSettings
	}
	return nil
}

// notAt is the refusal of an action that belongs to loc, elsewhere.
func notAt(loc engine.TownLocation) error {
	switch loc {
	case engine.Tavern:
		return errors.New("The party is formed at Gilgamesh's Tavern. Go there first.")
	case engine.Training:
		return errors.New("Characters are created, renamed and deleted at the Training Grounds. Go there first.")
	case engine.EdgeOfTown:
		return errors.New("Expeditions and practice set out from the Edge of Town. Go there first.")
	case engine.Trading:
		return errors.New("Items are bought, sold, uncursed and identified at Boltac's Trading Post, in the Castle. Go there first.")
	}
	return fmt.Errorf("That is done at %v. Go there first.", loc)
}

var errNotInSettings = errors.New("The roster is reloaded, reset and restored in Settings, at the Edge of Town. Go there first.")

// stayInSettings puts the player back in Settings after the roster was
// read again (a reload or a restore starts a new game at the castle), by
// the castle's own way to the Edge of Town. A resumed expedition is in the
// maze instead.
func (s *Session) stayInSettings() {
	if s.active() == nil && s.ctrl.EnterEdgeOfTown() == nil {
		s.settings = true
	}
}

// settlePrompt closes any open town prompt using the game's own cancel
// steps, leaving no prompt open.
func (s *Session) settlePrompt() error {
	c := s.ctrl
	switch mode := c.Game.Town.InputMode; mode {
	case engine.InputNone:
		return nil
	case engine.InputTrainingName:
		return c.CloseNamePrompt()
	case engine.InputConfirmCreate:
		c.ConfirmCreate(false)
		return c.CloseNamePrompt()
	case engine.InputRoster:
		c.CloseRoster()
		return c.CloseNamePrompt()
	case engine.InputPassword:
		c.CancelPasswordCheck()
		return c.CloseNamePrompt()
	case engine.InputConfirmDelete:
		c.ConfirmDeleteCharacter(false)
		c.LeaveCharacterEdit()
		return c.CloseNamePrompt()
	case engine.InputCharEdit:
		c.LeaveCharacterEdit()
		return c.CloseNamePrompt()
	case engine.InputTavernPassword:
		return c.CancelTavernPassword()
	default:
		// The GUI never leaves other prompts open.
		return fmt.Errorf("unexpected open prompt %d", mode)
	}
}
