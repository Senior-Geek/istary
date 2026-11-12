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
// items are kept. It ends by the stairs, or when no one can go on: then the
// party stays stranded in the maze (R-1) and the player is in the castle.
//
// Each save carries the expedition's checkpoint (M6b-2, app.Controller.
// Checkpoint): the last safe point, in the same atomic write as the roster.
// When the app starts with an expedition saved, it resumes there (resume);
// an encounter it was in is over, with nothing more awarded. A saved
// expedition that cannot be resumed is kept untouched and shown as a
// problem, with a way to return its party to the castle.

import (
	"errors"
	"fmt"
	"path/filepath"

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
		if err := s.at(engine.EdgeOfTown); err != nil {
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
		if err := s.realRefusal(); err != nil {
			return err
		}
		c := s.ctrl
		if err := s.at(engine.EdgeOfTown); err != nil {
			return err
		}
		if err := c.EnterMaze(); err != nil {
			return err
		}
		s.real = true
		return nil
	})
}

// realRefusal is why a real expedition cannot set out now, or nil. The
// Edge of Town shows it (View.Departures) on the button.
func (s *Session) realRefusal() error {
	g := s.ctrl.Game
	switch {
	case s.real || s.practice != nil:
		return errPracticeRunning
	case g.Expedition != nil || g.OtherExpedition != nil:
		return errExpeditionSaved
	case g.Phase != engine.PhaseTown || len(g.Town.Party.Members) == 0:
		return errNoParty
	case !engine.PartyCanContinue(g.Town.Party.Members):
		return errNoOneCanGoOn
	}
	return nil
}

// errNoOneCanGoOn refuses setting out when no one in the party can act
// (app.ErrNoOneCanGoOn): the incapacitated must be taken to the Temple, or
// led by someone who can.
var errNoOneCanGoOn = errors.New("No one in the party can go on. Take them to the Temple, or add someone who can lead them.")

// errNoParty refuses setting out with no party (the original's Edge of
// Town offers M)AZE only to a party).
var errNoParty = errors.New("Form a party in the Tavern first.")

// errExpeditionSaved refuses starting an expedition while another one is
// saved and not resumed: starting would replace it.
var errExpeditionSaved = errors.New("An expedition is already saved. Resume it or return its party to the castle first.")

// checkpoint updates the saved expedition to where the real expedition now
// stands, before it is saved: a checkpoint while out, none once the party
// is back in the castle. If this moment cannot be recorded, the previous
// checkpoint (the last safe point) is kept and the error says so.
func (s *Session) checkpoint() error {
	g := s.ctrl.Game
	if g.Phase == engine.PhaseTown {
		g.Expedition = nil
		return nil
	}
	exp, err := s.ctrl.Checkpoint(s.pristine)
	if err != nil {
		return fmt.Errorf("This moment could not be recorded (%v); the last safe point is kept.", err)
	}
	g.Expedition = exp
	return nil
}

// resume picks up a saved expedition after loading: the party is back at
// its last safe point (the dungeon screen), or, if it cannot be resumed,
// the problem is shown and the expedition stays saved as it is.
func (s *Session) resume() {
	g := s.ctrl.Game
	s.real, s.problem = false, ""
	if g.OtherExpedition != nil {
		s.problem = "This save holds an expedition from another version of Istary. It cannot be resumed here and is left as it is."
		return
	}
	e := g.Expedition
	if e == nil {
		return
	}
	interrupted := e.Interrupted
	if err := s.ctrl.Resume(s.pristine); err != nil {
		s.problem = err.Error()
		return
	}
	if g.Phase == engine.PhaseTown {
		// No one could go on: the party is stranded where it was (R-1, D-R).
		s.notice = "Istary closed after the party was defeated. " + strandedNotice(g)
		if err := s.checkpoint(); err == nil {
			s.save()
		}
		return
	}
	s.real = true
	s.ctrl.Explore() // an old save has no map: only what is seen from here
	what := ""
	switch interrupted {
	case app.InterruptedCombat:
		what = "Istary closed during an encounter: it is over, and its unclaimed rewards are lost. "
	case app.InterruptedChest:
		what = "Istary closed while the party dealt with a chest: the chest and its rewards are lost. "
	}
	s.notice = "Your party has returned to the dungeon. " + what + resumedStranded
	s.resumed = s.notice
	if err := s.checkpoint(); err == nil {
		s.save() // record that the encounter, if any, is over
	}
}

// The resumed expedition's notice (View.NoticeTitle): where the party
// stands is the dungeon's own; the stranded stay where they were recorded.
const (
	resumedTitle    = "EXPEDITION RESUMED"
	resumedStranded = "Stranded adventurers remain at their recorded locations."
)

// ReturnExpeditionToCastle ends a saved expedition that cannot be resumed:
// its party joins the castle party, as a restart left it before
// expeditions were saved. A newer version's expedition is never ended here.
func (s *Session) ReturnExpeditionToCastle() View {
	return s.do(func() error {
		g := s.ctrl.Game
		if s.problem == "" || g.Expedition == nil {
			return errors.New("There is no saved expedition to return.")
		}
		backup, err := s.backupPath("before-return")
		if err != nil {
			return err
		}
		if err := engine.BackupRoster(g, backup); err != nil {
			return fmt.Errorf("Backup failed, nothing changed: %w", err)
		}
		if err := s.ctrl.ReturnExpeditionToCastle(); err != nil {
			return err
		}
		s.problem = ""
		s.notice = "The expedition's party is back in the castle (the save before this was backed up as " + filepath.Base(backup) + ")."
		return nil
	})
}

// TryResumeAgain reads the save again to resume an expedition that could
// not be resumed (View.Recovery), from anywhere in town, as starting Istary
// again would: the expedition resumes if it now can, otherwise the problem
// is shown again; either way the player starts over at the Castle.
func (s *Session) TryResumeAgain() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	switch {
	case s.problem == "":
		return s.view(errors.New("There is no saved expedition waiting to be resumed."))
	case s.ctrl.Game.Phase == engine.PhaseCreation:
		return s.view(errCreating)
	}
	return s.view(s.reload())
}

// DungeonAction performs one of the dungeon view's actions (app.Act...).
// button is the button to press (1 is A) for "button", and the number of
// the found character (Inspect.Found[].N) for "pick-up".
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
			return c.PickUp(button)
		case app.ActLeave:
			return c.LeaveInspect()
		case app.ActLeaveCamp:
			return c.LeaveCamp()
		}
		return app.ErrInvalidChoice
	})
}
