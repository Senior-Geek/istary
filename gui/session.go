package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
	"github.com/Senior-Geek/istary/scenarios/wiz1"
)

// Session is the object bound to the Svelte front end. Every method locks
// the session, calls app.Controller operations, saves, and returns a fresh
// View. Wails runs bound calls on separate goroutines, and the game uses the
// global math/rand source, so the lock keeps all game logic sequential.
type Session struct {
	mu       sync.Mutex
	scenario *data.Scenario
	ctrl     *app.Controller
	saveDir  string
	startErr error  // fatal startup problem shown instead of the UI
	notice   string // one-shot confirmation shown after an action
}

// NewSession loads Wizardry I and the roster from saveDir. saveDir must not
// be the terminal game's directory.
func NewSession(saveDir string) *Session {
	s := &Session{saveDir: saveDir}
	if err := s.open(); err != nil {
		s.startErr = err
	}
	return s
}

func (s *Session) open() error {
	terminalDir, err := terminalSaveDir()
	if err != nil {
		return err
	}
	if filepath.Clean(s.saveDir) == filepath.Clean(terminalDir) {
		return fmt.Errorf("refusing to use the terminal game's save directory %s", terminalDir)
	}
	engine.SetSaveDir(s.saveDir)

	scenario, err := wiz1.Load()
	if err != nil {
		return fmt.Errorf("load Wizardry I data: %w", err)
	}
	s.scenario = scenario
	return s.reload()
}

// terminalSaveDir is the directory the terminal game saves to.
func terminalSaveDir() (string, error) {
	engine.SetSaveDir("")
	return engine.SaveDir()
}

// reload replaces the session with a fresh game read from disk.
func (s *Session) reload() error {
	game := engine.New(s.scenario)
	game.Title = nil
	game.Phase = engine.PhaseTown
	if err := game.Load(); err != nil {
		return fmt.Errorf("load roster: %w", err)
	}
	s.ctrl = app.New(game)
	return nil
}

// do runs one action under the lock, saves, and returns the new view. An
// action error is shown to the player; a save error is reported too.
func (s *Session) do(action func() error) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	err := action()
	if saveErr := s.ctrl.Save(); saveErr != nil {
		err = errors.Join(err, fmt.Errorf("Could not save: %w", saveErr))
	}
	return s.view(err)
}

// State returns the current view without changing anything.
func (s *Session) State() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.view(nil)
}

// ReloadRoster discards in-memory state and reads the roster from disk.
func (s *Session) ReloadRoster() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	if s.ctrl.Game.Phase == engine.PhaseCreation {
		return s.view(errors.New("Finish or discard the character being created first."))
	}
	err := s.reload()
	if err == nil {
		s.notice = fmt.Sprintf("Roster reloaded from %s.", s.saveDir)
	}
	return s.view(err)
}

// StartCreation enters a name at the Training Grounds.
func (s *Session) StartCreation(name string) View {
	return s.do(func() error {
		c := s.ctrl
		if c.Game.Town.Location != engine.Training {
			if err := c.EnterTrainingGrounds(); err != nil {
				return err
			}
		}
		result, err := c.TrainingEnterName(name)
		switch result {
		case app.NameNeedsPassword:
			// Editing existing characters is not part of this prototype.
			existing := c.Game.Town.EditChar
			c.CancelPasswordCheck()
			return fmt.Errorf("%s is already on the roster. Choose another name.", existing.Name)
		case app.NameRoster:
			c.CloseRoster()
			return app.ErrEmptyName
		case app.NameLeft:
			return app.ErrEmptyName
		case app.NameNotFound:
			return errors.New("Characters cannot be created in this scenario.")
		}
		return err
	})
}

// AnswerCreate answers "THAT CHARACTER DOES NOT EXIST. DO YOU WANT TO CREATE IT?".
func (s *Session) AnswerCreate(yes bool) View {
	return s.do(func() error { return s.ctrl.ConfirmCreate(yes) })
}

func (s *Session) SetPassword(password, confirm string) View {
	return s.do(func() error { return s.ctrl.SetCreationPassword(password, confirm) })
}

func (s *Session) SetName(name string) View {
	return s.do(func() error { return s.ctrl.SetCreationName(name) })
}

func (s *Session) ChooseRace(race int) View {
	return s.do(func() error { return s.ctrl.ChooseRace(engine.Race(race)) })
}

func (s *Session) ChooseAlignment(align int) View {
	return s.do(func() error { return s.ctrl.ChooseAlignment(engine.Alignment(align)) })
}

// AddPoint spends a bonus point on stat (0=STR ... 5=LUCK).
func (s *Session) AddPoint(stat int) View {
	return s.do(func() error {
		if err := s.ctrl.SelectStat(stat); err != nil {
			return err
		}
		return s.ctrl.AddStatPoint()
	})
}

// RemovePoint takes a bonus point back from stat.
func (s *Session) RemovePoint(stat int) View {
	return s.do(func() error {
		if err := s.ctrl.SelectStat(stat); err != nil {
			return err
		}
		return s.ctrl.RemoveStatPoint()
	})
}

func (s *Session) FinishStats() View {
	return s.do(func() error { return s.ctrl.FinishStats() })
}

func (s *Session) ChooseClass(class int) View {
	return s.do(func() error { return s.ctrl.ChooseClass(engine.Class(class)) })
}

func (s *Session) Back() View {
	return s.do(func() error { return s.ctrl.CreationBack() })
}

func (s *Session) CancelCreation() View {
	return s.do(func() error { return s.ctrl.CancelCreation() })
}

// Confirm keeps (and saves) or discards the finished character.
func (s *Session) Confirm(keep bool) View {
	return s.do(func() error {
		ch, err := s.ctrl.ConfirmCreation(keep)
		if err == nil && ch != nil {
			s.notice = fmt.Sprintf("%s joined the roster and was saved.", ch.Name)
		}
		return err
	})
}
