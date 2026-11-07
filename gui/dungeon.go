package main

// The dungeon. Two kinds of expedition walk the maze through the
// controller's dungeon operations: the same step as the terminal, square
// effects, healing and poison, and the encounter check included. An
// encounter opens the combat screen on the same expedition; after a
// victory the walk goes on.
//
// A practice expedition (M6a) takes copies of the party in a separate,
// never-saved game. When the party is back in town, or the expedition is
// abandoned, the practice game is discarded and the roster is unchanged.
//
// A real expedition (M6b-1) takes the party itself, in the saved game, and
// saves after every action: injuries, spells spent, experience, gold and
// items are kept. It ends only in the castle: by the stairs, or with the
// engine's defeat, which returns the dead party there. Until M6b-2 adds
// checkpoints, the save holds no position: if the app closes during an
// expedition, the party is in the castle when it starts again, with
// everything saved up to the last action (the terminal behaves the same).

import (
	"errors"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
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

// RealExpedition takes the party into the maze, to Camp at the entrance,
// and saves.
func (s *Session) RealExpedition() View {
	return s.combatDo(func() error {
		if s.real || s.practice != nil {
			return errPracticeRunning
		}
		c := s.ctrl
		if c.Game.Phase != engine.PhaseTown || len(c.Game.Town.Party.Members) == 0 {
			return errors.New("Form a party in the Tavern first.")
		}
		if err := s.goTo(engine.Castle); err != nil {
			return err
		}
		if err := c.EnterMaze(); err != nil {
			return err
		}
		s.real = true
		return nil
	})
}

// DungeonAction performs one of the dungeon view's actions (app.Act...).
// button is the button to press (1 is A) for "button".
func (s *Session) DungeonAction(action string, button int) View {
	return s.onPractice(func(c *app.Controller) error {
		if !s.expedition && !s.real {
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
