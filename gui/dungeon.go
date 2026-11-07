package main

// The dungeon (M6a): a practice expedition. The current party's copies walk
// the real maze in a separate, never-saved game, through the controller's
// dungeon operations: the same step as the terminal, square effects,
// healing and poison, and the encounter check included. An encounter opens
// the combat screen on the same expedition; after a victory the walk goes
// on. When the party is back in town, or the expedition is abandoned, the
// practice game is discarded and the roster is unchanged.
//
// Persistent expeditions (saving, the party's real state, rescue) are M6b.

import (
	"github.com/Senior-Geek/istary/app"
)

// PracticeExpedition takes copies of the party into the maze, to Camp at
// the entrance.
func (s *Session) PracticeExpedition() View {
	return s.combatDo(func() error {
		if s.practice != nil {
			return errPracticeRunning
		}
		pg, err := s.practiceGame()
		if err != nil {
			return err
		}
		c := app.New(pg)
		if err := c.EnterMaze(); err != nil {
			return err
		}
		s.practice, s.practiceChest, s.expedition = c, false, true
		return nil
	})
}

// DungeonAction performs one of the dungeon view's actions (app.Act...).
// button is the button to press (1 is A) for "button".
func (s *Session) DungeonAction(action string, button int) View {
	return s.onPractice(func(c *app.Controller) error {
		if !s.expedition {
			return errNoPractice
		}
		switch action {
		case app.ActForward:
			return c.Forward()
		case app.ActLeft:
			return c.TurnLeft()
		case app.ActRight:
			return c.TurnRight()
		case app.ActKick:
			return c.Kick()
		case app.ActCamp:
			return c.Camp()
		case app.ActInspect:
			return c.Inspect()
		case app.ActContinue:
			return c.ContinueMessage()
		case app.ActYes:
			return c.Answer(true)
		case app.ActNo:
			return c.Answer(false)
		case app.ActButton:
			return c.PressButton(button)
		case app.ActPickUp:
			return c.PickUp()
		case app.ActLeave:
			return c.LeaveInspect()
		case app.ActLeaveCamp:
			return c.LeaveCamp()
		}
		return app.ErrInvalidChoice
	})
}
