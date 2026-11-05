package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	again := NewSession(dir)
	v2 := again.State()
	if len(v2.Roster) != 1 || v2.Roster[0] != v.Roster[0] {
		t.Errorf("reloaded roster = %+v, want %+v", v2.Roster, v.Roster)
	}
	// ReloadRoster in the same session also matches.
	if v3 := s.ReloadRoster(); len(v3.Roster) != 1 || v3.Roster[0] != v.Roster[0] {
		t.Errorf("ReloadRoster = %+v", v3.Roster)
	}
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
