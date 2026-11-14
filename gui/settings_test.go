package main

import (
	"bytes"
	"maps"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// without is folder (navigation_test.go) less one file.
func without(t *testing.T, dir, name string) map[string]string {
	t.Helper()
	files := folder(t, dir)
	delete(files, filepath.Join(dir, name))
	return files
}

// Showing and hiding the party is kept in gui-settings.json, written whole,
// and survives a restart; it never touches the roster file, the backups or
// the random numbers.
func TestPartyVisibilityIsKept(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	if v := s.State(); !v.Settings.PartyVisible || v.Settings.Note != "" {
		t.Fatalf("defaults: %+v", v.Settings)
	}
	if _, err := os.Stat(filepath.Join(dir, settingsFile)); !os.IsNotExist(err) {
		t.Fatalf("settings written before any change: %v", err)
	}
	for _, visible := range []bool{false, true, false} {
		game := without(t, dir, settingsFile)
		rand.Seed(17)
		v := s.SetPartyVisible(visible)
		next := rand.Int63()
		rand.Seed(17)
		if next != rand.Int63() {
			t.Fatal("a random number was drawn")
		}
		if v.Error != "" || v.Settings.PartyVisible != visible || v.Settings.Note != "" {
			t.Fatalf("set %v: %q %+v", visible, v.Error, v.Settings)
		}
		if !maps.Equal(game, without(t, dir, settingsFile)) {
			t.Fatalf("set %v: changed another file in the save folder", visible)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(dir, settingsFile))
	if string(raw) != "{\n  \"version\": 1,\n  \"partyHidden\": true\n}\n" {
		t.Fatalf("settings file: %q", raw)
	}
	if roster, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); bytes.Contains(roster, []byte("partyHidden")) {
		t.Fatal("the setting is in the roster file")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("temporary file left: %s", e.Name())
		}
	}
	if v := reopened(t, s, dir).State(); v.Settings.PartyVisible || v.Settings.Note != "" {
		t.Fatalf("after a restart: %+v", v.Settings)
	}
}

// The party can be shown or hidden anywhere, out on an expedition and in
// practice too, without touching the game or its save.
func TestPartyVisibilityAnywhere(t *testing.T) {
	rand.Seed(5)
	s, dir := partySession(t, false)
	try := func(when string) {
		t.Helper()
		before := s.State()
		game := without(t, dir, settingsFile)
		for _, visible := range []bool{false, true} {
			v := s.SetPartyVisible(visible)
			if v.Error != "" || v.Settings.PartyVisible != visible {
				t.Fatalf("%s: %q %+v", when, v.Error, v.Settings)
			}
			v.Settings = before.Settings
			if !sameView(v, before) {
				t.Fatalf("%s: the view changed", when)
			}
		}
		if !maps.Equal(game, without(t, dir, settingsFile)) {
			t.Fatalf("%s: changed the save folder", when)
		}
	}
	try("at the Edge of Town")
	s.PracticeExpedition()
	try("practice expedition")
	s.AbandonPractice()
	s.PracticeEncounter(false)
	try("practice encounter")
	s.AbandonPractice()
	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	try("real expedition")
}

// A damaged settings file is left as it is and the defaults are used, with
// a note; the next change replaces it. The game loads as usual.
func TestDamagedSettingsFallBack(t *testing.T) {
	for name, content := range map[string]string{
		"not json":       "{oops",
		"wrong type":     `{"version": 1, "partyHidden": "yes"}`,
		"newer format":   `{"version": 2, "partyHidden": true}`,
		"no version":     `{"partyHidden": true}`,
		"empty":          "",
		"a json array":   "[]",
		"a json number":  "7",
		"trailing bytes": `{"version": 1, "partyHidden": true} x`,
	} {
		s, dir := newTestSession(t)
		rosterWith(t, s, hero("WALKER", engine.Good, ""))
		path := filepath.Join(dir, settingsFile)
		os.WriteFile(path, []byte(content), 0o644)
		s.shutdown()

		s = NewSession(dir)
		v := s.State()
		if v.Fatal != "" || len(v.Roster) != 1 {
			t.Fatalf("%s: the game did not load: %q", name, v.Fatal)
		}
		if !v.Settings.PartyVisible || !strings.Contains(v.Settings.Note, "could not be read") {
			t.Fatalf("%s: %+v", name, v.Settings)
		}
		if raw, _ := os.ReadFile(path); string(raw) != content {
			t.Fatalf("%s: the damaged file was changed", name)
		}
		walkTo(t, s, "tavern") // navigating and acting leave it alone too
		join(t, s, 0, "")
		if raw, _ := os.ReadFile(path); string(raw) != content {
			t.Fatalf("%s: the damaged file was changed by play", name)
		}
		if v := s.SetPartyVisible(false); v.Settings.PartyVisible || v.Settings.Note != "" {
			t.Fatalf("%s: after a change: %+v", name, v.Settings)
		}
		if v := reopened(t, s, dir).State(); v.Settings.PartyVisible || v.Settings.Note != "" {
			t.Fatalf("%s: replaced file after a restart: %+v", name, v.Settings)
		}
		s.shutdown()
	}
}

// If the settings cannot be written, the choice holds in this window and
// the view says it was not kept; the game is untouched.
func TestUnwritableSettings(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	game := folder(t, dir)
	os.Chmod(dir, 0o555)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	v := s.SetPartyVisible(false)
	os.Chmod(dir, 0o755)
	if v.Settings.PartyVisible || !strings.Contains(v.Settings.Note, "could not be saved") || v.Error != "" {
		t.Fatalf("unwritable: %q %+v", v.Error, v.Settings)
	}
	if !maps.Equal(game, folder(t, dir)) {
		t.Fatal("the save folder changed")
	}
	if v := s.SetPartyVisible(false); v.Settings.Note != "" {
		t.Fatalf("writable again: %+v", v.Settings)
	}
}

// Resetting, restoring and backing up the roster never touch the settings,
// and the settings are never offered as a backup.
func TestRosterManagementLeavesSettingsAlone(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "keeper")
	s.SetPartyVisible(false)
	path := filepath.Join(dir, settingsFile)
	before := folder(t, dir)[path]

	walkTo(t, s, "settings")
	s.ResetRoster()
	s.RestoreBackup(s.ListBackups()[0].File)
	s.ReloadRoster()
	if folder(t, dir)[path] != before {
		t.Fatal("roster management changed the settings file")
	}
	for _, b := range s.ListBackups() {
		raw, _ := os.ReadFile(filepath.Join(dir, "backups", b.File))
		if strings.Contains(b.File, "settings") || bytes.Contains(raw, []byte("partyHidden")) {
			t.Fatalf("settings in backup %s", b.File)
		}
	}
	if v := s.State(); v.Settings.PartyVisible {
		t.Fatal("a reload forgot the setting")
	}
}

// The illustrated places' appearance (ART-2C) is the window's, like the
// party's visibility: the defaults (slate panels, a parchment story box,
// the whole picture) until it is chosen; kept in gui-settings.json, which
// a settings file of before keeps reading; never in the game or its save;
// and drawing no random number. A value it does not know is refused and
// changes nothing.
func TestAppearanceIsKept(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	look := func(v View) [3]string { return [3]string{v.Settings.Panels, v.Settings.Story, v.Settings.Artwork} }
	if v := s.State(); look(v) != [3]string{"slate", "parchment", "full"} {
		t.Fatalf("defaults: %+v", v.Settings)
	}
	before := s.State()
	game := without(t, dir, settingsFile)
	for _, c := range [][3]string{{"dark", "slate", "subdued"}, {"light", "dark", "hidden"}, {"slate", "parchment", "full"}, {"light", "parchment", "subdued"}} {
		rand.Seed(17)
		v := s.SetAppearance(c[0], c[1], c[2])
		next := rand.Int63()
		rand.Seed(17)
		if next != rand.Int63() {
			t.Fatal("a random number was drawn")
		}
		if v.Error != "" || look(v) != c || v.Settings.Note != "" {
			t.Fatalf("set %v: %q %+v", c, v.Error, v.Settings)
		}
		v.Settings = before.Settings
		if !sameView(v, before) {
			t.Fatalf("set %v: the view changed", c)
		}
		if !maps.Equal(game, without(t, dir, settingsFile)) {
			t.Fatalf("set %v: changed another file in the save folder", c)
		}
	}
	for _, bad := range [][3]string{{"neon", "parchment", "full"}, {"slate", "light", "full"}, {"slate", "parchment", "dim"}, {"", "", ""}} {
		raw, _ := os.ReadFile(filepath.Join(dir, settingsFile))
		v := s.SetAppearance(bad[0], bad[1], bad[2])
		after, _ := os.ReadFile(filepath.Join(dir, settingsFile))
		if v.Error == "" || look(v) != [3]string{"light", "parchment", "subdued"} || !bytes.Equal(raw, after) {
			t.Fatalf("set %v: %q %+v", bad, v.Error, v.Settings)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(dir, settingsFile))
	if string(raw) != "{\n  \"version\": 1,\n  \"partyHidden\": false,\n  \"panels\": \"light\",\n  \"story\": \"parchment\",\n  \"artwork\": \"subdued\"\n}\n" {
		t.Fatalf("settings file: %q", raw)
	}
	if roster, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); bytes.Contains(roster, []byte("parchment")) || bytes.Contains(roster, []byte("artwork")) {
		t.Fatal("the appearance is in the roster file")
	}
	if v := reopened(t, s, dir).State(); look(v) != [3]string{"light", "parchment", "subdued"} || v.Settings.Note != "" {
		t.Fatalf("after a restart: %+v", v.Settings)
	}
}

// A settings file of before ART-2C has no appearance: the defaults, with
// no note; one with an appearance this Istary does not know reads it as the
// default, and keeps the rest.
func TestAppearanceFromOtherFiles(t *testing.T) {
	for name, c := range map[string]struct {
		file string
		want [3]string
		hide bool
	}{
		"before ART-2C": {"{\n  \"version\": 1,\n  \"partyHidden\": true\n}\n", [3]string{"slate", "parchment", "full"}, true},
		"unknown names": {`{"version": 1, "panels": "neon", "story": "dark", "artwork": "sepia"}`, [3]string{"slate", "dark", "full"}, false},
	} {
		s, dir := newTestSession(t)
		os.WriteFile(filepath.Join(dir, settingsFile), []byte(c.file), 0o644)
		v := reopened(t, s, dir).State()
		if got := [3]string{v.Settings.Panels, v.Settings.Story, v.Settings.Artwork}; got != c.want || v.Settings.PartyVisible == c.hide || v.Settings.Note != "" {
			t.Fatalf("%s: %+v", name, v.Settings)
		}
	}
}
