package main

// The window's own settings (Castle-0 C0-3): preferences of this window,
// not of the game. They live in gui-settings.json in the folder the window
// opened (the library, games.go), the same for every game and never in one: no save, checkpoint, backup, reset or
// restore reads or writes them, and the roster file never holds them.
// Writes are atomic, like the save (engine.WriteFileAtomic). A missing
// file means the defaults; a damaged one is left as it is, the defaults
// are used and the view says so, and the next change replaces it.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/Senior-Geek/istary/engine"
)

const settingsFile = "gui-settings.json"

// settingsFormat is the settings file's format version.
const settingsFormat = 1

// Settings are the window's preferences. The zero value is the defaults.
type Settings struct {
	Version     int  `json:"version"`
	PartyHidden bool `json:"partyHidden"` // the party panel is hidden
	// MinimapHidden: the dungeon's minimap is hidden (M-1).
	MinimapHidden bool `json:"minimapHidden,omitempty"`
	// The illustrated places' appearance (ART-2C): their panels', their
	// story box's and their picture's. Empty is the default; a value this
	// Istary does not know is read as the default too.
	Panels  string `json:"panels,omitempty"`
	Story   string `json:"story,omitempty"`
	Artwork string `json:"artwork,omitempty"`
}

// The appearances an illustrated place can have (ART-2C), the default
// first: the panels', the story box's, and how its picture is shown.
var (
	panelLooks = []string{"slate", "dark", "light"}
	storyLooks = []string{"parchment", "dark", "slate"}
	artworks   = []string{"full", "subdued", "hidden"}
)

// chosen is choice if it is one of options, and otherwise the default,
// options[0].
func chosen(choice string, options []string) string {
	if slices.Contains(options, choice) {
		return choice
	}
	return options[0]
}

// SettingsView is the settings as the page sees them, and Note, when they
// could not be read or saved.
type SettingsView struct {
	PartyVisible   bool `json:"partyVisible"`
	MinimapVisible bool `json:"minimapVisible"`
	// The illustrated places' appearance (ART-2C), never empty: the
	// panels (slate, dark or light), the story box (parchment, dark or
	// slate) and the picture (full, subdued or hidden).
	Panels  string `json:"panels"`
	Story   string `json:"story"`
	Artwork string `json:"artwork"`
	Note    string `json:"note"`
}

func (s *Session) settingsPath() string {
	return filepath.Join(s.root, settingsFile)
}

// loadSettings reads the settings file: the defaults if there is none, and
// the defaults with a note if it cannot be read.
func (s *Session) loadSettings() {
	s.prefs, s.prefsNote = Settings{}, ""
	raw, err := os.ReadFile(s.settingsPath())
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	var st Settings
	if err == nil {
		err = json.Unmarshal(raw, &st)
	}
	if err == nil && st.Version != settingsFormat {
		err = fmt.Errorf("format %d, this Istary reads %d", st.Version, settingsFormat)
	}
	if err != nil {
		s.prefsNote = fmt.Sprintf("The window's settings in %s could not be read (%v); the defaults are used until a setting is changed.", settingsFile, err)
		return
	}
	s.prefs = st
}

// SetPartyVisible shows or hides the party panel and saves the choice. It is
// not a game action: it is accepted anywhere, even out on an expedition or
// in combat, and touches nothing of the game. If the file cannot be written,
// the choice holds in this window and the view says why it was not kept.
func (s *Session) SetPartyVisible(visible bool) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.prefs.PartyHidden = !visible
	s.prefsNote = ""
	if err := s.saveSettings(); err != nil {
		s.prefsNote = fmt.Sprintf("The window's settings could not be saved (%v); this choice holds until Istary closes.", err)
	}
	return s.view(nil)
}

// SetMinimapVisible shows or hides the dungeon's minimap and saves the
// choice, as SetPartyVisible does: accepted anywhere, touching nothing of
// the game.
func (s *Session) SetMinimapVisible(visible bool) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.prefs.MinimapHidden = !visible
	s.prefsNote = ""
	if err := s.saveSettings(); err != nil {
		s.prefsNote = fmt.Sprintf("The window's settings could not be saved (%v); this choice holds until Istary closes.", err)
	}
	return s.view(nil)
}

// SetAppearance sets the illustrated places' appearance (ART-2C): their
// panels (slate, dark or light), their story box (parchment, dark or
// slate) and their picture (full, subdued or hidden), and saves it, as
// SetPartyVisible does: accepted anywhere, touching nothing of the game.
// A value it does not know is refused and nothing changes.
func (s *Session) SetAppearance(panels, story, artwork string) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	for _, c := range []struct {
		what, choice string
		options      []string
	}{{"panels", panels, panelLooks}, {"story box", story, storyLooks}, {"picture", artwork, artworks}} {
		if !slices.Contains(c.options, c.choice) {
			return s.view(fmt.Errorf("there is no %q appearance for the %s", c.choice, c.what))
		}
	}
	s.prefs.Panels, s.prefs.Story, s.prefs.Artwork = panels, story, artwork
	s.prefsNote = ""
	if err := s.saveSettings(); err != nil {
		s.prefsNote = fmt.Sprintf("The window's settings could not be saved (%v); this choice holds until Istary closes.", err)
	}
	return s.view(nil)
}

func (s *Session) saveSettings() error {
	st := s.prefs
	st.Version = settingsFormat
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	err = engine.WriteFileAtomic(s.settingsPath(), append(raw, '\n'), 0o644)
	var nd *engine.NotDurableError
	if errors.As(err, &nd) {
		return nil // in place; at worst a crash forgets a preference
	}
	return err
}

func (s *Session) settingsView() SettingsView {
	return SettingsView{
		PartyVisible: !s.prefs.PartyHidden, MinimapVisible: !s.prefs.MinimapHidden,
		Panels: chosen(s.prefs.Panels, panelLooks), Story: chosen(s.prefs.Story, storyLooks), Artwork: chosen(s.prefs.Artwork, artworks),
		Note: s.prefsNote,
	}
}
