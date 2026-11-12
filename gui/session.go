package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
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
	resumed  string // the notice of a resumed expedition, headed EXPEDITION RESUMED while it is shown

	// practice is the practice encounter's or expedition's controller: a
	// separate, never-saved game with copies of the party. nil when none.
	practice      *app.Controller
	practiceChest bool
	expedition    bool // practice is an expedition (dungeon.go), not one encounter

	// real is true while the saved party is out on a real expedition
	// (dungeon.go): from entering the maze until it is back in the castle.
	// Its actions run on ctrl, the saved game.
	real bool

	// unsaved is the error of the last save that failed, until a save
	// succeeds; the view keeps showing it so a failure is not missed.
	unsaved string
	// uncertain is set while the last save replaced the file but could not
	// sync the folder: saved, but perhaps not durably.
	uncertain string

	// pristine is Wizardry I as loaded, to find and restore changed special
	// squares of an expedition (app.Controller.Checkpoint and Resume).
	pristine *data.Scenario

	// receipt is the transaction the last shop action completed (shop.go),
	// shown at Boltac's until the next shop action or the next visit.
	receipt *ShopReceipt
	// innLast is what the last Inn action completed (inn.go), and goldLast
	// the last pooling or sharing at the Tavern (gold.go), shown there
	// until the next one or the next visit.
	innLast  *InnReceipt
	goldLast *GoldReceipt
	// templeLast is the last treatment or pooling at the Temple (temple.go).
	templeLast *TempleLast

	// settings is set while the player is in Settings (navigation.go),
	// which the window opens at the Edge of Town; it is not part of the game.
	settings bool

	// quit closes the window (Wails's Quit), for LeaveGame (edge.go); nil
	// until the window starts, and in tests.
	quit func()
	// closing is set while the window asks what to do with an unsaved
	// game before closing (close.go), so a second close does not ask twice.
	closing bool

	// prefs are the window's own settings (settings.go); prefsNote says
	// when they could not be read or saved.
	prefs     Settings
	prefsNote string

	// problem is why the saved expedition cannot be resumed, while it
	// cannot; the expedition stays saved and untouched (expedition.go).
	problem string

	// lock is the save folder's single-writer lock (lock.go); lockNote says
	// when the folder could not be locked.
	lock     *saveLock
	lockNote string

	// root is the folder the window opened: locked, and holding the
	// window's settings. In a library window (games.go) it holds the game
	// slots and saveDir is the open game's slot folder; otherwise saveDir
	// is root itself, one game.
	root    string
	library bool
	// slot is the open game's slot (games.go), 0 when none is open or the
	// window is not a library; replacing is the slot whose replacement
	// awaits confirmation.
	slot      int
	replacing int
}

// NewSession loads Wizardry I and the roster from saveDir. saveDir must not
// be the terminal game's directory.
func NewSession(saveDir string) *Session {
	s := unopenedSession(saveDir)
	s.start()
	return s
}

// errNotStarted is the state of a session that has not opened its folder.
var errNotStarted = errors.New("Istary is starting.")

// unopenedSession returns a session that has not touched saveDir yet; start
// opens it. The app opens it (start) only once Wails starts the window: Wails also
// runs the program to generate its bindings (wails build, wails dev), and
// that run must not lock, resume or save anything.
func unopenedSession(saveDir string) *Session {
	return &Session{saveDir: saveDir, root: saveDir, startErr: errNotStarted}
}

// unopenedLibrary returns a window on the game slots in root (games.go),
// not yet opened: it opens on game selection, with no game open.
func unopenedLibrary(root string) *Session {
	return &Session{saveDir: root, root: root, library: true, startErr: errNotStarted}
}

// NewLibrary opens the game slots in root, on game selection.
func NewLibrary(root string) *Session {
	s := unopenedLibrary(root)
	s.start()
	return s
}

// start opens the save folder: lock, load and resume.
func (s *Session) start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startErr = s.open()
}

func (s *Session) open() error {
	terminalDir, err := terminalSaveDir()
	if err != nil {
		return err
	}
	if filepath.Clean(s.root) == filepath.Clean(terminalDir) {
		return fmt.Errorf("refusing to use the terminal game's save directory %s", terminalDir)
	}
	if s.lock == nil {
		lock, err := lockSaveFolder(s.root)
		switch {
		case errors.Is(err, errLockUnsupported):
			s.lockNote = "This save folder cannot be locked, so make sure no other Istary window uses it."
		case err != nil:
			return err
		}
		s.lock = lock
	}
	s.loadSettings()
	if s.library {
		s.forgetGame() // game selection first
		return nil
	}
	return s.loadGame(s.saveDir)
}

// closed reports whether there is no game to act on: the window has not
// opened its folder, or no game is open (games.go).
func (s *Session) closed() bool { return s.startErr != nil || s.ctrl == nil }

// closedErr is why there is no game to act on.
func (s *Session) closedErr() error {
	if s.startErr != nil {
		return s.startErr
	}
	return errNoGameOpen
}

// shutdown releases the save folder (on quitting). The session cannot be used
// afterwards.
func (s *Session) shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lock.release()
	s.lock = nil
	if s.startErr == nil {
		s.startErr = errors.New("This Istary window has closed its save folder.")
	}
}

// TryAgain retries a start that failed (another window holding the save
// folder, for instance).
func (s *Session) TryAgain() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		s.startErr = s.open()
	}
	return s.view(nil)
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
	s.settings = false
	s.resume()
	return nil
}

// do runs one action under the lock, saves, and returns the new view. An
// action error is shown to the player; a save error is reported too.
func (s *Session) do(action func() error) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed() {
		return s.view(nil)
	}
	s.notice = ""
	if s.real {
		return s.view(errExpeditionOut)
	}
	err := action()
	err = errors.Join(err, s.save())
	return s.view(err)
}

// errExpeditionOut refuses town actions while the party is in the maze:
// as in the game, the roster, the Tavern and the Training Grounds are only
// reachable from the castle.
var errExpeditionOut = errors.New("The party is out on an expedition. Return to the castle first.")

// saveGame writes c's game; tests replace it to see how a failure is
// reported.
var saveGame = (*app.Controller).Save

// save writes the saved game and remembers a failure until a save succeeds.
// The action itself has already happened in memory and is never repeated:
// the next successful save writes it with whatever follows. A save that
// replaced the file but could not sync the folder is kept apart: it is
// saved, perhaps not durably.
func (s *Session) save() error {
	err := saveGame(s.ctrl)
	var nd *engine.NotDurableError
	switch {
	case err == nil:
		s.unsaved, s.uncertain = "", ""
		return nil
	case errors.As(err, &nd):
		s.unsaved = ""
		s.uncertain = "Saved, but the save folder could not be synced (" + nd.Err.Error() +
			"); the last save may not survive a crash or power loss."
		return nil
	}
	s.unsaved = err.Error()
	return fmt.Errorf("Could not save: %w", err)
}

// SaveNow tries again to save the game as this window has it, after a
// save failed (View.Unsaved). Nothing is done again: what failed to be
// saved has already happened in the window, once, and this writes it, so
// nothing is bought, sold or moved twice. With nothing unsaved it writes
// nothing. On a real expedition the party's place is recorded first, as
// after any action there.
func (s *Session) SaveNow() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed() {
		return s.view(nil)
	}
	s.notice = ""
	if s.unsaved == "" {
		s.notice = "Everything is saved."
		return s.view(nil)
	}
	var err error
	if s.real {
		err = s.checkpoint()
	}
	err = errors.Join(err, s.save())
	if err == nil {
		s.notice = "Saved."
		durable := s.uncertain == ""
		if s.receipt != nil {
			s.receipt.Saved, s.receipt.Durable = true, durable
		}
		if s.innLast != nil {
			s.innLast.Saved, s.innLast.Durable = true, durable
		}
		if s.goldLast != nil {
			s.goldLast.Saved, s.goldLast.Durable = true, durable
		}
		if s.templeLast != nil {
			s.templeLast.Saved, s.templeLast.Durable = true, durable
		}
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
	if s.closed() {
		return s.view(nil)
	}
	s.notice = ""
	if s.real {
		return s.view(errExpeditionOut)
	}
	if s.ctrl.Game.Phase == engine.PhaseCreation {
		return s.view(errors.New("Finish or discard the character being created first."))
	}
	if err := s.inSettingsNow(); err != nil {
		return s.view(err)
	}
	err := s.reload()
	if err == nil {
		s.stayInSettings()
		s.notice = fmt.Sprintf("Roster reloaded from %s.", s.saveDir)
	}
	return s.view(err)
}

// StartCreation enters a name at the Training Grounds.
func (s *Session) StartCreation(name string) View {
	return s.do(func() error {
		c := s.ctrl
		if err := s.at(engine.Training); err != nil {
			return err
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

// DeleteCharacter deletes name from the roster through the Training Grounds
// flow: look up the name, answer its password, confirm the delete. The
// password check is the game's own rule for opening a character.
func (s *Session) DeleteCharacter(name, password string) View {
	return s.do(func() error {
		c := s.ctrl
		if err := s.at(engine.Training); err != nil {
			return err
		}
		result, err := c.TrainingEnterName(name)
		if err != nil {
			return err
		}
		if result != app.NameNeedsPassword {
			// Close whatever prompt the lookup opened, via the game's own steps.
			switch result {
			case app.NameOfferCreate:
				c.ConfirmCreate(false)
			case app.NameRoster:
				c.CloseRoster()
			}
			return fmt.Errorf("No character named %s is on the roster.", strings.ToUpper(strings.TrimSpace(name)))
		}
		if err := c.SubmitCharacterPassword(password); err != nil {
			return err
		}
		deleted := c.EditCharacter().Name
		if err := c.BeginDeleteCharacter(); err != nil {
			return err
		}
		if err := c.ConfirmDeleteCharacter(true); err != nil {
			return err
		}
		s.notice = fmt.Sprintf("%s was deleted.", deleted)
		return nil
	})
}

// ResetRoster backs up the current roster file into <saveDir>/backups, then
// starts an empty roster. Nothing is cleared if the backup fails.
func (s *Session) ResetRoster() View {
	return s.do(func() error {
		c := s.ctrl
		if c.Game.Phase != engine.PhaseTown {
			return errors.New("Finish or discard the character being created first.")
		}
		if err := s.inSettingsNow(); err != nil {
			return err
		}
		if c.Game.Expedition != nil || c.Game.OtherExpedition != nil {
			return errExpeditionSaved
		}
		// Make sure the file on disk matches memory before copying it.
		if err := c.Save(); err != nil {
			return fmt.Errorf("Could not save before backup: %w", err)
		}
		backup, err := s.backupPath("")
		if err != nil {
			return err
		}
		if err := engine.BackupRoster(c.Game, backup); err != nil {
			return fmt.Errorf("Backup failed, roster not reset: %w", err)
		}
		if err := c.ClearRoster(); err != nil {
			return err
		}
		s.notice = fmt.Sprintf("Roster backed up to %s and reset.", backup)
		return nil
	})
}

// backupPath returns a new, unused file name under <saveDir>/backups. tag,
// if set, is appended to say why the backup was made.
func (s *Session) backupPath(tag string) (string, error) {
	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("Could not create backup folder: %w", err)
	}
	stamp := time.Now().Format("20060102-150405")
	if tag != "" {
		stamp += "-" + tag
	}
	for n := 0; ; n++ {
		name := fmt.Sprintf("roster1-%s.json", stamp)
		if n > 0 {
			name = fmt.Sprintf("roster1-%s-%d.json", stamp, n)
		}
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		}
	}
}

func (s *Session) backupDir() string {
	return filepath.Join(s.saveDir, "backups")
}

// RenameCharacter renames the roster character at index. The name must pass
// the creation prompt's rules (app.ValidateName) and the game's rename rules
// (app.Controller.RenameCharacter).
func (s *Session) RenameCharacter(index int, newName string) View {
	return s.do(func() error {
		if err := app.ValidateName(newName); err != nil {
			return err
		}
		roster := s.ctrl.Game.Town.Roster.Characters
		if index < 0 || index >= len(roster) || roster[index] == nil {
			return app.ErrNoSuchCharacter
		}
		if err := s.at(engine.Training); err != nil {
			return err
		}
		old := roster[index].Name
		if err := s.ctrl.RenameCharacter(index, newName); err != nil {
			return err
		}
		s.notice = fmt.Sprintf("%s is now %s.", old, roster[index].Name)
		return nil
	})
}
