package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestMain isolates HOME so the terminal save directory used by the guard is
// a throwaway one.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "istary-gui-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

func newTestSession(t *testing.T) (*Session, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "saves")
	s := NewSession(dir)
	if v := s.State(); v.Fatal != "" {
		t.Fatalf("startup failed: %s", v.Fatal)
	}
	return s, dir
}

// reopened is the app started again on dir while s is still open, as after
// a restart: s lets go of the save folder, a new session opens it, and s
// takes it back. The new session no longer holds the folder; use it only
// to look at what was loaded.
func reopened(t *testing.T, s *Session, dir string) *Session {
	t.Helper()
	s.lock.release()
	r := NewSession(dir)
	r.lock.release()
	lock, err := lockSaveFolder(dir)
	if err != nil {
		t.Fatalf("take the folder back: %v", err)
	}
	s.lock = lock
	return r
}

func mustOK(t *testing.T, v View, wantScreen string) View {
	t.Helper()
	if v.Error != "" || v.Fatal != "" {
		t.Fatalf("unexpected error: %q fatal=%q", v.Error, v.Fatal)
	}
	if v.Screen != wantScreen {
		t.Fatalf("screen = %q, want %q", v.Screen, wantScreen)
	}
	return v
}

// spendAll spends every bonus point, filling stats in the given order while
// the engine allows another point there. Bounded by the number of points.
func spendAll(t *testing.T, s *Session, v View, order []int) View {
	t.Helper()
	for _, stat := range order {
		for n := 0; n < 40 && v.Creation.Points > 0 && v.Creation.CanAdd[stat]; n++ {
			v = mustOK(t, s.AddPoint(stat), "creation")
		}
	}
	if v.Creation.Points != 0 {
		t.Fatalf("could not spend all points: %+v", v.Creation)
	}
	return v
}

// createMage runs the whole creation workflow through the bound methods.
func createMage(t *testing.T, s *Session, name string) View {
	t.Helper()
	v := mustOK(t, s.StartCreation(name), "confirmCreate")
	if !strings.Contains(v.Message, "DOES NOT EXIST") {
		t.Fatalf("confirm message = %q", v.Message)
	}
	v = mustOK(t, s.AnswerCreate(true), "creation")
	if v.Creation.Step != "password" || v.Creation.Name != strings.ToUpper(name) {
		t.Fatalf("creation = %+v", v.Creation)
	}
	v = mustOK(t, s.SetPassword("secret", "secret"), "creation")
	v = mustOK(t, s.ChooseRace(1), "creation") // ELF
	v = mustOK(t, s.ChooseAlignment(1), "creation")
	if v.Creation.Step != "stats" || v.Creation.Race != "ELF" || v.Creation.Points < 7 {
		t.Fatalf("after alignment: %+v", v.Creation)
	}
	// Pour points into I.Q. first so the elf qualifies as a mage.
	v = spendAll(t, s, v, []int{1, 3, 0, 2, 4, 5})
	v = mustOK(t, s.FinishStats(), "creation")
	if v.Creation.Step != "class" || !v.Creation.ClassOK[1] {
		t.Fatalf("class step: %+v", v.Creation)
	}
	v = mustOK(t, s.ChooseClass(1), "creation") // MAGE
	if v.Creation.Step != "confirm" || v.Creation.Class != "MAGE" {
		t.Fatalf("confirm step: %+v", v.Creation)
	}
	return mustOK(t, s.Confirm(true), "roster")
}

func TestCreateSaveAndReload(t *testing.T) {
	s, dir := newTestSession(t)
	v := createMage(t, s, "radagast")
	if len(v.Roster) != 1 || v.Roster[0].Name != "RADAGAST" || v.Roster[0].Short != "N-MAG" {
		t.Fatalf("roster = %+v", v.Roster)
	}
	if !strings.Contains(v.Notice, "RADAGAST") {
		t.Errorf("notice = %q", v.Notice)
	}

	// The roster file is in the GUI save directory, in the normal format.
	raw, err := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Roster []struct {
			Name     string
			Password string `json:"password"`
			Class    int
		} `json:"roster"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil || len(saved.Roster) != 1 ||
		saved.Roster[0].Name != "RADAGAST" || saved.Roster[0].Password != "SECRET" || saved.Roster[0].Class != 1 {
		t.Fatalf("saved roster = %+v (err %v)", saved, err)
	}

	// A new session (as on app restart) reads it back.
	again := reopened(t, s, dir)
	v2 := again.State()
	if len(v2.Roster) != 1 || !sameExceptSpells(v2.Roster[0], v.Roster[0]) {
		t.Errorf("reloaded roster = %+v, want %+v", v2.Roster, v.Roster)
	}
	// Existing engine behavior: Load rebuilds known spells from spell slots,
	// so the new mage knows HALITO and KATINO before a reload and every
	// level-1 mage spell after it.
	if got := v.Roster[0].Spells; !reflect.DeepEqual(got, []string{"HALITO", "KATINO"}) {
		t.Errorf("spells before reload = %v", got)
	}
	if got := v2.Roster[0].Spells; !reflect.DeepEqual(got, []string{"HALITO", "MOGREF", "KATINO", "DUMAPIC"}) {
		t.Errorf("spells after reload = %v", got)
	}
	// ReloadRoster in the same session matches a fresh session.
	if v3 := s.ReloadRoster(); len(v3.Roster) != 1 || !reflect.DeepEqual(v3.Roster[0], v2.Roster[0]) {
		t.Errorf("ReloadRoster = %+v", v3.Roster)
	}
}

// sameExceptSpells compares two roster rows ignoring known spells (see the
// reload note in TestCreateSaveAndReload).
func sameExceptSpells(a, b CharacterView) bool {
	a.Spells, b.Spells = nil, nil
	return reflect.DeepEqual(a, b)
}

func TestTerminalSavesUntouched(t *testing.T) {
	s, _ := newTestSession(t)
	createMage(t, s, "gandalf")
	terminal := filepath.Join(os.Getenv("HOME"), ".config", "wizardry")
	if _, err := os.Stat(terminal); !os.IsNotExist(err) {
		t.Errorf("GUI must not create or write %s (stat err %v)", terminal, err)
	}
}

func TestRefusesTerminalSaveDirectory(t *testing.T) {
	terminal := filepath.Join(os.Getenv("HOME"), ".config", "wizardry")
	s := NewSession(terminal)
	v := s.State()
	if !strings.Contains(v.Fatal, "refusing") {
		t.Fatalf("Fatal = %q, want refusal", v.Fatal)
	}
	if v2 := s.StartCreation("X"); v2.Fatal == "" || len(v2.Roster) != 0 {
		t.Error("a refused session must not act")
	}
}

func TestValidationMessages(t *testing.T) {
	s, _ := newTestSession(t)
	if v := s.StartCreation("   "); v.Error != "Enter a character name." || v.Screen != "roster" {
		t.Errorf("blank name: %q screen=%s", v.Error, v.Screen)
	}
	createMage(t, s, "saruman")
	if v := s.StartCreation("SARUMAN"); !strings.Contains(v.Error, "already on the roster") || v.Screen != "roster" {
		t.Errorf("duplicate name: %q screen=%s", v.Error, v.Screen)
	}

	mustOK(t, s.StartCreation("pippin"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	if v := s.SetPassword("a", "b"); v.Error != "The passwords do not match. Enter the password again." {
		t.Errorf("mismatch: %q", v.Error)
	}
	s.SetPassword("", "")
	s.ChooseRace(4) // HOBBIT
	v := s.ChooseAlignment(0)
	if v := s.FinishStats(); v.Error != "Spend all bonus points before choosing a class." {
		t.Errorf("points left: %q", v.Error)
	}
	v = spendAll(t, s, v, []int{5, 0, 1, 2, 3, 4})
	v = s.FinishStats()
	if v.Error == "" && !v.Creation.ClassOK[6] {
		if v := s.ChooseClass(6); v.Error != "Your statistics and alignment do not qualify for that class." {
			t.Errorf("unavailable class: %q", v.Error)
		}
	}
	if v := s.ReloadRoster(); v.Error != "Finish or discard the character being created first." {
		t.Errorf("reload during creation: %q", v.Error)
	}
}

func TestDeclineAndBack(t *testing.T) {
	s, _ := newTestSession(t)
	mustOK(t, s.StartCreation("merry"), "confirmCreate")
	mustOK(t, s.AnswerCreate(false), "roster")

	mustOK(t, s.StartCreation("merry"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	s.SetPassword("", "")
	v := mustOK(t, s.Back(), "creation")
	if v.Creation.Step != "name" {
		t.Fatalf("back from race: step=%s", v.Creation.Step)
	}
	v = mustOK(t, s.SetName("meriadoc"), "creation")
	if v.Creation.Name != "MERIADOC" || v.Creation.Step != "race" {
		t.Fatalf("rename: %+v", v.Creation)
	}
	s.Back()
	mustOK(t, s.CancelCreation(), "roster")
	if len(s.State().Roster) != 0 {
		t.Error("cancelled creation must not add a character")
	}
}

func TestClassGuideInView(t *testing.T) {
	s, _ := newTestSession(t)
	mustOK(t, s.StartCreation("guide"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	mustOK(t, s.SetPassword("", ""), "creation")
	v := mustOK(t, s.ChooseRace(0), "creation") // HUMAN
	if v.Creation.Guide != nil {
		t.Fatal("guide should not appear before alignment is chosen")
	}
	v = mustOK(t, s.ChooseAlignment(1), "creation") // NEUTRAL
	g := v.Creation.Guide
	if len(g) != 8 {
		t.Fatalf("guide has %d rows, want 8", len(g))
	}
	for i, row := range g {
		if row.Class != v.Classes[i] {
			t.Errorf("row %d class %q, want %q", i, row.Class, v.Classes[i])
		}
		if (row.Status == "eligible") != v.Creation.ClassOK[i] {
			t.Errorf("%s: status %q but classOk %v", row.Class, row.Status, v.Creation.ClassOK[i])
		}
	}
	// Neutral characters cannot be Priest, Bishop, Lord or Ninja.
	for _, i := range []int{2, 4, 6, 7} {
		if g[i].Status != "alignment" || g[i].Note != "Not open to NEUTRAL characters" {
			t.Errorf("%s: %q %q", g[i].Class, g[i].Status, g[i].Note)
		}
	}
	if g[0].Alignments != "ANY" || g[6].Alignments != "GOOD" || g[2].Alignments != "GOOD, EVIL" {
		t.Errorf("alignment text: %q %q %q", g[0].Alignments, g[6].Alignments, g[2].Alignments)
	}
	// Human STR 8 vs Fighter 11: short 3, reachable with the rolled points.
	if g[0].Shortfall[0] != 3 || g[0].Status != "reachable" {
		t.Errorf("fighter: %+v", g[0])
	}

	// The guide follows every point added or removed.
	v = mustOK(t, s.AddPoint(0), "creation")
	if f := v.Creation.Guide[0]; f.Shortfall[0] != 2 || !strings.Contains(f.Note, "Needs 2 points more") {
		t.Errorf("after +STR: %+v", f)
	}
	v = mustOK(t, s.RemovePoint(0), "creation")
	if f := v.Creation.Guide[0]; f.Shortfall[0] != 3 {
		t.Errorf("after -STR: %+v", f)
	}

	// Spend everything on LUCK and I.Q.: Fighter now needs points that are
	// only available by moving them.
	v = spendAll(t, s, v, []int{5, 1, 2, 3, 4, 0})
	for i, row := range v.Creation.Guide {
		if row.Status == "reachable" {
			t.Errorf("%s reachable with 0 points left: %+v", v.Classes[i], row)
		}
	}
	if v.Creation.Guide[0].Shortfall[0] > 0 && v.Creation.Guide[0].Status != "move" {
		t.Errorf("fighter after spending elsewhere: %+v", v.Creation.Guide[0])
	}

	// Lord needs 37 points from Human base; no roll provides that.
	if l := v.Creation.Guide[6]; l.Status != "alignment" {
		t.Errorf("lord (neutral): %+v", l)
	}
}

func TestClassGuideUnreachableByPoints(t *testing.T) {
	s, _ := newTestSession(t)
	mustOK(t, s.StartCreation("lordling"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	mustOK(t, s.SetPassword("", ""), "creation")
	mustOK(t, s.ChooseRace(0), "creation")
	v := mustOK(t, s.ChooseAlignment(0), "creation") // GOOD
	l := v.Creation.Guide[6]
	need := 0
	for _, n := range l.Shortfall {
		need += n
	}
	if need != 37 {
		t.Fatalf("human lord shortfall total = %d, want 37", need)
	}
	wantStatus := "points"
	if v.Creation.Points >= 37 {
		wantStatus = "reachable"
	}
	if l.Status != wantStatus || (wantStatus == "points" && !strings.HasPrefix(l.Note, "Cannot be reached: needs 37 points")) {
		t.Errorf("lord with %d points: %+v", v.Creation.Points, l)
	}
}

// A save that fails (here: the save folder made read-only) is reported, the
// previous save stays byte-identical with no temporary file left, and the
// next save after the folder is writable again succeeds. The GUI saves
// through the same engine Save as the terminal.
func TestFailedSaveKeepsThePreviousRoster(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	s, dir := newTestSession(t)
	createMage(t, s, "radagast")
	path := filepath.Join(dir, "roster1.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	v := s.RenameCharacter(0, "SARUMAN")
	if !strings.Contains(v.Error, "Could not save") {
		t.Fatalf("error = %q, want a save failure", v.Error)
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, before) {
		t.Fatal("the previous save changed")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("temporary file left: %s", e.Name())
		}
	}

	os.Chmod(dir, 0o755)
	v = s.RenameCharacter(0, "GANDALF")
	if v.Error != "" {
		t.Fatalf("save after recovery: %q", v.Error)
	}
	if v := reopened(t, s, dir).State(); v.Roster[0].Name != "GANDALF" {
		t.Fatalf("reloaded name = %q", v.Roster[0].Name)
	}
}
