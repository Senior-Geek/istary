package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// savedNames reads roster1.json and returns the character names in it.
func savedNames(t *testing.T, dir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Roster []struct{ Name string } `json:"roster"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range f.Roster {
		names = append(names, c.Name)
	}
	return names
}

func backupFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, _ := os.ReadDir(filepath.Join(dir, "backups"))
	var out []string
	for _, e := range entries {
		out = append(out, filepath.Join(dir, "backups", e.Name()))
	}
	return out
}

func TestCharacterDetailsInView(t *testing.T) {
	s, _ := newTestSession(t)
	v := createMage(t, s, "elrond")
	ch := v.Roster[0]
	if ch.Race != "ELF" || ch.Class != "MAGE" || ch.Alignment != "NEUTRAL" || ch.Level != 1 ||
		!ch.HasPassword || ch.MageSpells[0] != 2 || ch.MaxMage[0] != 2 || ch.NextXP <= 0 ||
		ch.InMaze || ch.MazeLevel != 0 || len(ch.Items) != 0 {
		t.Errorf("details = %+v", ch)
	}
}

func TestDeleteCharacter(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "boromir")
	createMage(t, s, "faramir")

	// Wrong password: refused with the game's message, nothing deleted.
	v := s.DeleteCharacter("boromir", "wrong")
	if v.Error != "** THATS NOT IT **" || len(v.Roster) != 2 {
		t.Fatalf("wrong password: error=%q roster=%d", v.Error, len(v.Roster))
	}
	// Unknown name: refused, no prompt left open.
	if v := s.DeleteCharacter("nobody", ""); !strings.Contains(v.Error, "No character named NOBODY") || v.Screen != "roster" {
		t.Fatalf("unknown: error=%q screen=%s", v.Error, v.Screen)
	}

	v = s.DeleteCharacter("boromir", "secret")
	if v.Error != "" || len(v.Roster) != 1 || v.Roster[0].Name != "FARAMIR" || v.Notice != "BOROMIR was deleted." {
		t.Fatalf("delete: error=%q notice=%q roster=%+v", v.Error, v.Notice, v.Roster)
	}
	if names := savedNames(t, dir); len(names) != 1 || names[0] != "FARAMIR" {
		t.Errorf("saved roster after delete = %v", names)
	}
	// Still gone after reload, and creation still works afterwards.
	if v := reopened(t, s, dir).State(); len(v.Roster) != 1 || v.Roster[0].Name != "FARAMIR" {
		t.Errorf("reloaded roster = %+v", v.Roster)
	}
	createMage(t, s, "boromir")
}

func TestResetRosterBacksUpThenEmpties(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "aragorn")
	createMage(t, s, "legolas")
	before, err := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}

	walkTo(t, s, "settings")
	v := s.ResetRoster()
	if v.Error != "" || len(v.Roster) != 0 || !strings.Contains(v.Notice, "backed up to") {
		t.Fatalf("reset: error=%q notice=%q roster=%d", v.Error, v.Notice, len(v.Roster))
	}
	backups := backupFiles(t, dir)
	if len(backups) != 1 {
		t.Fatalf("backups = %v", backups)
	}
	got, _ := os.ReadFile(backups[0])
	if string(got) != string(before) {
		t.Errorf("backup does not match the roster before reset")
	}
	if !strings.Contains(v.Notice, backups[0]) {
		t.Errorf("notice should name the backup file: %q", v.Notice)
	}
	if names := savedNames(t, dir); len(names) != 0 {
		t.Errorf("saved roster after reset = %v", names)
	}
	if v := reopened(t, s, dir).State(); len(v.Roster) != 0 {
		t.Errorf("reloaded roster after reset = %+v", v.Roster)
	}

	// The backup is a normal roster file: the terminal game's restore and a
	// fresh load both read it.
	restoreDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(restoreDir, "roster1.json"), got, 0644); err != nil {
		t.Fatal(err)
	}
	if v := NewSession(restoreDir).State(); len(v.Roster) != 2 {
		t.Errorf("loading the backup gave %d characters, want 2", len(v.Roster))
	}

	// A second reset in the same second gets its own file.
	createMage(t, s, "gimli")
	walkTo(t, s, "settings")
	s.ResetRoster()
	if n := len(backupFiles(t, dir)); n != 2 {
		t.Errorf("after second reset: %d backups, want 2", n)
	}
}

func TestResetRosterWithNoSaveYet(t *testing.T) {
	s, dir := newTestSession(t)
	walkTo(t, s, "settings")
	v := s.ResetRoster()
	if v.Error != "" || len(backupFiles(t, dir)) != 1 {
		t.Errorf("reset on a fresh save folder: error=%q backups=%v", v.Error, backupFiles(t, dir))
	}
}

func TestResetRosterRefusedDuringCreation(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "bilbo")
	mustOK(t, s.StartCreation("frodo"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	v := s.ResetRoster()
	if v.Error != "Finish or discard the character being created first." || len(v.Roster) != 1 {
		t.Errorf("reset during creation: error=%q roster=%d", v.Error, len(v.Roster))
	}
	if len(backupFiles(t, dir)) != 0 {
		t.Error("no backup should be made when reset is refused")
	}
}

func TestRosterManagementLeavesTerminalSavesAlone(t *testing.T) {
	s, _ := newTestSession(t)
	createMage(t, s, "sam")
	s.DeleteCharacter("sam", "secret")
	createMage(t, s, "rosie")
	walkTo(t, s, "settings")
	s.ResetRoster()
	terminal := filepath.Join(os.Getenv("HOME"), ".config", "wizardry")
	if _, err := os.Stat(terminal); !os.IsNotExist(err) {
		t.Errorf("GUI must not create or write %s (stat err %v)", terminal, err)
	}
}
