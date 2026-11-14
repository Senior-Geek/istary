package main

// Game slots (UX-3A; docs/design/ux3/UX-3A.md). The window opens a library
// folder (the save folder of earlier versions) and shows six slots, each
// one complete, independent adventure: its own roster, party, expedition,
// automap and backups. A slot is a folder, games/slot-N, holding exactly
// what the save folder held before: the engine's roster1.json, written by
// engine.GameState.Save as ever, and backups/. Only game.json, the slot's
// own record, is new. The window's settings and its lock stay in the
// library folder: they belong to the window, not to any game.
//
// One game is open at a time; engine.SetSaveDir points the engine at its
// folder, and opening a game loads Wizardry I afresh, so nothing of one
// game (a used special square, a receipt, a practice) reaches another.
//
// The single save of earlier versions, roster1.json in the library folder,
// is offered for import into an empty slot. Importing copies it (and its
// backups) and loads the copy before it counts; the original is never
// written, moved or deleted. A slot records the original's checksum, so
// once a slot holds it the import is not offered again.
//
// Replacing a game moves its folder aside, to replaced/, after the player
// confirms; nothing of a game is ever deleted here.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Senior-Geek/istary/engine"
	"github.com/Senior-Geek/istary/scenarios/wiz1"
)

const (
	gameSlots   = 6
	gamesDir    = "games"
	replacedDir = "replaced"
	gameFile    = "game.json"
	saveFile    = "roster1.json" // engine.RosterPath for Wizardry I
	gameFormat  = 1              // game.json's format version
)

// gameRecord is a slot's game.json: when and how its game began.
type gameRecord struct {
	Version int    `json:"version"`
	Created string `json:"created"`          // RFC 3339
	Origin  string `json:"origin"`           // "new" or "imported"
	Import  string `json:"import,omitempty"` // the imported save's SHA-256
}

// SlotView is one slot on the game selection screen.
type SlotView struct {
	Slot  int    `json:"slot"`
	State string `json:"state"` // "empty", "game" or "damaged"
	// Saved is when the game was last saved, local time; Created when it
	// began. Imported: it began as the save of an earlier Istary.
	Saved    string `json:"saved"`
	Created  string `json:"created"`
	Imported bool   `json:"imported"`
	gameSummary
	Problem string `json:"problem"` // why a damaged slot cannot be continued
}

// gameSummary is what a save holds, in brief.
type gameSummary struct {
	Characters int      `json:"characters"`
	Party      []string `json:"party"`      // the castle party, in order
	Expedition []string `json:"expedition"` // the party out on a saved expedition
	Depth      int      `json:"depth"`      // its maze level, 1 = B1F; 0 if none
	Stranded   int      `json:"stranded"`   // characters left in the maze
}

// LegacyView is the save of an earlier Istary, offered for import.
type LegacyView struct {
	Saved string `json:"saved"`
	gameSummary
	Problem string `json:"problem"` // why it cannot be imported
}

// GamesView is the game selection screen.
type GamesView struct {
	Slots  []SlotView  `json:"slots"`
	Legacy *LegacyView `json:"legacy"` // nil: none, or a slot holds it
	// Replacing is the slot whose replacement awaits confirmation, or 0.
	Replacing int `json:"replacing"`
}

func (s *Session) slotDir(slot int) string {
	return filepath.Join(s.root, gamesDir, fmt.Sprintf("slot-%d", slot))
}

func (s *Session) legacyPath() string { return filepath.Join(s.root, saveFile) }

var (
	errNoSuchSlot  = fmt.Errorf("Choose a game slot from 1 to %d.", gameSlots)
	errSlotTaken   = errors.New("That slot holds a game. Continue it, or replace it.")
	errSlotEmpty   = errors.New("That slot is empty.")
	errNoGameOpen  = errors.New("Choose a game first.")
	errNotLibrary  = errors.New("This window opened one save folder directly; it has no other games.")
	errNoLegacy    = errors.New("There is no earlier save to import.")
	errNotReplace  = errors.New("Choose Replace for that slot first.")
	errLeaveAtEdge = errors.New("Select Game is at the Edge of Town.")
)

// readSave parses a save as the engine writes it.
func readSave(path string) (*engine.SaveState, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var st engine.SaveState
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, raw, errors.New("the save cannot be read")
	}
	return &st, raw, nil
}

func summarize(st *engine.SaveState) gameSummary {
	var sum gameSummary
	for _, c := range st.Roster {
		if c == nil {
			continue
		}
		sum.Characters++
		if c.Stranded != nil {
			sum.Stranded++
		}
	}
	sum.Party = st.PartyNames
	if len(st.Expedition) > 0 {
		var e struct {
			Party []string `json:"party"`
			Where struct {
				Level int `json:"level"`
			} `json:"where"`
		}
		if json.Unmarshal(st.Expedition, &e) == nil {
			sum.Expedition, sum.Depth = e.Party, e.Where.Level+1
		}
	}
	return sum
}

func stamp(t time.Time) string { return t.Local().Format("2006-01-02 15:04") }

// slotView reads one slot from disk.
func (s *Session) slotView(slot int) SlotView {
	v := SlotView{Slot: slot, State: "empty"}
	dir := s.slotDir(slot)
	rec, recErr := readRecord(dir)
	save := filepath.Join(dir, saveFile)
	fi, statErr := os.Stat(save)
	if errors.Is(recErr, fs.ErrNotExist) && errors.Is(statErr, fs.ErrNotExist) {
		if _, err := os.Stat(dir); err == nil {
			// A folder with no game in it (a new game interrupted, say):
			// a new game cannot be made over it, so it is replaced, which
			// keeps whatever is in it.
			v.State, v.Problem = "damaged", "its folder holds no saved game"
		}
		return v
	}
	v.State = "game"
	switch {
	case recErr == nil:
		if t, err := time.Parse(time.RFC3339, rec.Created); err == nil {
			v.Created = stamp(t)
		}
		v.Imported = rec.Origin == "imported"
		if rec.Version > gameFormat {
			v.State, v.Problem = "damaged", "it was made by a newer Istary"
			return v
		}
	case !errors.Is(recErr, fs.ErrNotExist):
		v.State, v.Problem = "damaged", "its record (game.json) cannot be read"
		return v
	}
	if statErr != nil {
		v.State, v.Problem = "damaged", "its save (roster1.json) is missing"
		return v
	}
	v.Saved = stamp(fi.ModTime())
	st, _, err := readSave(save)
	if err != nil {
		v.State, v.Problem = "damaged", err.Error()
		return v
	}
	v.gameSummary = summarize(st)
	return v
}

func readRecord(dir string) (*gameRecord, error) {
	raw, err := os.ReadFile(filepath.Join(dir, gameFile))
	if err != nil {
		return nil, err
	}
	var rec gameRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func writeRecord(dir string, rec gameRecord) error {
	rec.Version = gameFormat
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	err = engine.WriteFileAtomic(filepath.Join(dir, gameFile), append(raw, '\n'), 0o644)
	var nd *engine.NotDurableError
	if errors.As(err, &nd) {
		return nil // in place; the folder sync is retried with the first save
	}
	return err
}

// legacy reads the earlier save: nil if there is none or a slot holds it.
func (s *Session) legacy() *LegacyView {
	path := s.legacyPath()
	fi, err := os.Stat(path)
	if err != nil {
		return nil
	}
	v := &LegacyView{Saved: stamp(fi.ModTime())}
	st, raw, err := readSave(path)
	if err != nil {
		if raw == nil {
			v.Problem = "it cannot be read"
		} else {
			v.Problem = err.Error()
		}
		return v
	}
	if s.importedSlot(checksum(raw)) != 0 {
		return nil
	}
	v.gameSummary = summarize(st)
	return v
}

func checksum(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// importedSlot is the slot holding the import of the save with this
// checksum, or 0.
func (s *Session) importedSlot(sum string) int {
	for slot := 1; slot <= gameSlots; slot++ {
		if rec, err := readRecord(s.slotDir(slot)); err == nil && rec.Origin == "imported" && rec.Import == sum {
			return slot
		}
	}
	return 0
}

func (s *Session) gamesView() *GamesView {
	v := &GamesView{Legacy: s.legacy(), Replacing: s.replacing}
	for slot := 1; slot <= gameSlots; slot++ {
		v.Slots = append(v.Slots, s.slotView(slot))
	}
	return v
}

// chooser runs a game selection action: only in a library window, with no
// game open. It returns the new view.
func (s *Session) chooser(slot int, action func() error) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	switch {
	case !s.library:
		return s.view(errNotLibrary)
	case s.ctrl != nil:
		return s.view(errors.New("A game is open. Select Game, at the Edge of Town, chooses another."))
	case slot < 1 || slot > gameSlots:
		return s.view(errNoSuchSlot)
	}
	return s.view(action())
}

// ContinueGame opens the game in slot, as it was last saved: its party in
// the castle, or its expedition resumed where it was (dungeon.go resume).
func (s *Session) ContinueGame(slot int) View {
	return s.chooser(slot, func() error {
		s.replacing = 0
		switch v := s.slotView(slot); v.State {
		case "empty":
			return errSlotEmpty
		case "damaged":
			return fmt.Errorf("Game %d cannot be continued: %s. It is left as it is.", slot, v.Problem)
		}
		return s.openGame(slot)
	})
}

// NewGame begins a new adventure in an empty slot: an empty roster, no
// party, nothing of the maze seen, no one stranded, the party's place the
// Castle. An occupied slot is refused: ReplaceGame asks first.
func (s *Session) NewGame(slot int) View {
	return s.chooser(slot, func() error {
		s.replacing = 0
		if s.slotView(slot).State != "empty" {
			return errSlotTaken
		}
		return s.createGame(slot, gameRecord{Origin: "new"}, nil)
	})
}

// ReplaceGame asks to replace the game in slot with a new one; nothing
// changes until ConfirmReplaceGame.
func (s *Session) ReplaceGame(slot int) View {
	return s.chooser(slot, func() error {
		s.replacing = 0
		if s.slotView(slot).State == "empty" {
			return errSlotEmpty
		}
		s.replacing = slot
		return nil
	})
}

// CancelReplaceGame keeps the game ReplaceGame asked about.
func (s *Session) CancelReplaceGame() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replacing, s.notice = 0, ""
	return s.view(nil)
}

// ConfirmReplaceGame moves the game in slot aside, to the library's
// replaced/ folder, and begins a new one there. Only the slot ReplaceGame
// asked about is replaced.
func (s *Session) ConfirmReplaceGame(slot int) View {
	return s.chooser(slot, func() error {
		if s.replacing != slot {
			s.replacing = 0
			return errNotReplace
		}
		s.replacing = 0
		aside, err := s.moveAside(slot)
		if err != nil {
			return fmt.Errorf("Game %d could not be moved aside, so it was not replaced: %v", slot, err)
		}
		if err := s.createGame(slot, gameRecord{Origin: "new"}, nil); err != nil {
			return err
		}
		s.notice = fmt.Sprintf("A new game began in slot %d. The game it replaced is kept in %s.", slot, aside)
		return nil
	})
}

func (s *Session) moveAside(slot int) (string, error) {
	dir := filepath.Join(s.root, replacedDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	base := fmt.Sprintf("slot-%d-%s", slot, time.Now().Format("20060102-150405"))
	for n := 0; ; n++ {
		name := base
		if n > 0 {
			name = fmt.Sprintf("%s-%d", base, n)
		}
		dest := filepath.Join(dir, name)
		if _, err := os.Stat(dest); errors.Is(err, fs.ErrNotExist) {
			return dest, os.Rename(s.slotDir(slot), dest)
		}
	}
}

// ImportEarlierGame copies the save of an earlier Istary into an empty
// slot, with its backups, and opens it once the copy loads. The original
// is left exactly as it is.
func (s *Session) ImportEarlierGame(slot int) View {
	return s.chooser(slot, func() error {
		s.replacing = 0
		if s.slotView(slot).State != "empty" {
			return errSlotTaken
		}
		legacy := s.legacy()
		if legacy == nil {
			return errNoLegacy
		}
		if legacy.Problem != "" {
			return fmt.Errorf("The earlier save cannot be imported: %s. It is left as it is.", legacy.Problem)
		}
		raw, err := os.ReadFile(s.legacyPath())
		if err != nil {
			return fmt.Errorf("The earlier save cannot be read: %v", err)
		}
		rec := gameRecord{Origin: "imported", Import: checksum(raw)}
		if err := s.createGame(slot, rec, raw); err != nil {
			return err
		}
		imported := fmt.Sprintf("The earlier game was imported into slot %d; the original save is left as it was.", slot)
		if s.notice != "" && s.notice == s.resumed {
			// Its expedition resumed: still headed EXPEDITION RESUMED.
			s.resumed = imported + " " + s.notice
			s.notice = s.resumed
		} else {
			s.notice = strings.TrimSpace(imported + " " + s.notice)
		}
		return nil
	})
}

// createGame makes slot's folder with save (an empty game if nil), copies
// an import's backups, checks the copy and opens the game. If anything
// fails before the game opens, the new folder is removed: the slot is
// empty again, and nothing else was touched.
func (s *Session) createGame(slot int, rec gameRecord, save []byte) (err error) {
	dir := s.slotDir(slot)
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		return fmt.Errorf("Could not make the game's folder: %w", err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dir)
		}
	}()
	path := filepath.Join(dir, saveFile)
	if save == nil {
		// The engine writes the new game's first save, as any other.
		sc, err := wiz1.Load()
		if err != nil {
			return err
		}
		engine.SetSaveDir(dir)
		game := engine.New(sc)
		if err := game.Save(); err != nil {
			var nd *engine.NotDurableError
			if !errors.As(err, &nd) {
				return fmt.Errorf("Could not save the new game: %w", err)
			}
		}
	} else {
		if err := engine.WriteFileAtomic(path, save, 0o644); err != nil {
			var nd *engine.NotDurableError
			if !errors.As(err, &nd) {
				return fmt.Errorf("Could not copy the earlier save: %w", err)
			}
		}
		if err := s.copyBackups(filepath.Join(dir, "backups")); err != nil {
			return fmt.Errorf("Could not copy the earlier save's backups: %w", err)
		}
		if got, err := os.ReadFile(path); err != nil || string(got) != string(save) {
			return errors.New("The copy of the earlier save does not match it; nothing was imported.")
		}
	}
	rec.Created = time.Now().Format(time.RFC3339)
	if err := writeRecord(dir, rec); err != nil {
		return fmt.Errorf("Could not record the new game: %w", err)
	}
	if err := s.openGame(slot); err != nil {
		return err
	}
	return nil
}

// copyBackups copies the library's backups of the earlier save into dest.
func (s *Session) copyBackups(dest string) error {
	src := filepath.Join(s.root, "backups")
	entries, err := os.ReadDir(src)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dest, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dest, e.Name()), raw, 0o644); err != nil {
			return err
		}
		if fi, err := e.Info(); err == nil {
			os.Chtimes(filepath.Join(dest, e.Name()), fi.ModTime(), fi.ModTime())
		}
	}
	return nil
}

// openGame opens slot's game: Wizardry I loaded afresh, the engine
// pointed at the slot's folder, the window's game state cleared, and the
// save loaded and resumed. On failure no game is open.
func (s *Session) openGame(slot int) error {
	s.forgetGame()
	if err := s.loadGame(s.slotDir(slot)); err != nil {
		s.forgetGame()
		return fmt.Errorf("Game %d could not be opened: %v. It is left as it is.", slot, err)
	}
	s.slot = slot
	return nil
}

// loadGame opens the game saved in dir.
func (s *Session) loadGame(dir string) error {
	scenario, err := wiz1.Load()
	if err != nil {
		return fmt.Errorf("load Wizardry I data: %w", err)
	}
	pristine, err := wiz1.Load()
	if err != nil {
		return fmt.Errorf("load Wizardry I data: %w", err)
	}
	s.scenario, s.pristine, s.saveDir = scenario, pristine, dir
	engine.SetSaveDir(dir)
	return s.reload()
}

// forgetGame closes the open game in memory (it is already saved) and
// clears everything the window held for it.
func (s *Session) forgetGame() {
	s.ctrl, s.slot = nil, 0
	s.notice, s.resumed, s.problem = "", "", ""
	s.practice, s.practiceChest, s.expedition, s.real = nil, false, false, false
	s.unsaved, s.uncertain = "", ""
	s.receipt, s.innLast, s.goldLast, s.templeLast = nil, nil, nil, nil
	s.settings, s.replacing = false, 0
}

// ChangeGame (Select Game since NAV-1) closes the open game and returns to
// the game selection (GamesScreen), from the Edge of Town. Every action is
// saved as it happens, so nothing is lost;
// it is refused while the last save has failed, and anywhere a game is
// under way (an expedition, a practice, a character being created).
func (s *Session) ChangeGame() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed() {
		return s.view(nil)
	}
	s.notice = ""
	if !s.library {
		return s.view(errNotLibrary)
	}
	if err := s.navigationRefusal(); err != nil {
		return s.view(err)
	}
	if err := s.at(engine.EdgeOfTown); err != nil {
		return s.view(errLeaveAtEdge)
	}
	if s.unsaved != "" {
		return s.view(errUnsavedLeave)
	}
	slot := s.slot
	s.forgetGame()
	s.notice = fmt.Sprintf("Game %d is saved.", slot)
	return s.view(nil)
}
