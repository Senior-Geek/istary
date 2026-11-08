package main

import (
	"fmt"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// The game has one current town location, but the GUI shows the roster,
// Training Grounds and party side by side. Every GUI operation that needs a
// particular location calls goTo first; nothing else changes the location.

// goTo moves the game to loc (Castle, Training Grounds or Tavern) through
// the game's own transitions: close any open prompt, leave the current
// location for the castle, then enter loc.
func (s *Session) goTo(loc engine.TownLocation) error {
	c := s.ctrl
	if c.Game.Phase != engine.PhaseTown {
		return app.ErrWrongState // e.g. a character is being created
	}
	s.settings = false
	if err := s.settlePrompt(); err != nil {
		return err
	}
	town := c.Game.Town
	if town.Location == loc {
		return nil
	}
	switch town.Location {
	case engine.Castle:
	case engine.Training:
		if err := c.LeaveTrainingGrounds(); err != nil {
			return err
		}
	case engine.Tavern:
		if err := c.LeaveTavern(); err != nil {
			return err
		}
	default:
		app.GoToCastle(town) // locations the GUI never opens
	}
	switch loc {
	case engine.Castle:
		return nil
	case engine.Training:
		return c.EnterTrainingGrounds()
	case engine.Tavern:
		return c.EnterTavern()
	}
	return fmt.Errorf("the GUI cannot go to %v", loc)
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
