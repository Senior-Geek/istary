package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRenameCharacter(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "strider")
	createMage(t, s, "boromir")

	v := s.RenameCharacter(0, "aragorn")
	if v.Error != "" || v.Roster[0].Name != "ARAGORN" || v.Notice != "STRIDER is now ARAGORN." {
		t.Fatalf("rename: error=%q notice=%q roster=%+v", v.Error, v.Notice, v.Roster)
	}
	if names := savedNames(t, dir); names[0] != "ARAGORN" {
		t.Errorf("saved names = %v", names)
	}
	if v := reopened(t, s, dir).State(); v.Roster[0].Name != "ARAGORN" {
		t.Errorf("after reload: %+v", v.Roster)
	}

	for _, tc := range []struct{ name, want string }{
		{"boromir", "NAME ALREADY IN USE"},
		{"aragorn", "NAME ALREADY IN USE"},
		{"   ", "Enter a character name."},
		{"Elessar Telcontar", "Names may use up to 15 letters, digits, spaces, hyphens and apostrophes."},
		{"King!", "Names may use up to 15 letters, digits, spaces, hyphens and apostrophes."},
	} {
		if v := s.RenameCharacter(0, tc.name); v.Error != tc.want || v.Roster[0].Name != "ARAGORN" {
			t.Errorf("rename to %q: error=%q name=%q", tc.name, v.Error, v.Roster[0].Name)
		}
	}
	if v := s.RenameCharacter(9, "X"); v.Error != "That character is no longer on the roster." {
		t.Errorf("bad index: %q", v.Error)
	}
	// Attributes, race, class and alignment are untouched by a rename.
	before := s.State().Roster[0]
	s.RenameCharacter(0, "elessar")
	after := s.State().Roster[0]
	after.Name = before.Name
	if !reflect.DeepEqual(before, after) {
		t.Errorf("rename changed more than the name:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestListBackups(t *testing.T) {
	s, dir := newTestSession(t)
	if got := s.ListBackups(); len(got) != 0 {
		t.Fatalf("no backups yet, got %+v", got)
	}
	createMage(t, s, "one")
	createMage(t, s, "two")
	walkTo(t, s, "settings")
	s.ResetRoster()
	createMage(t, s, "three")
	walkTo(t, s, "settings")
	s.ResetRoster()
	os.WriteFile(filepath.Join(dir, "backups", "broken.json"), []byte("{oops"), 0644)
	os.WriteFile(filepath.Join(dir, "backups", "notes.txt"), []byte("ignored"), 0644)

	list := s.ListBackups()
	if len(list) != 3 {
		t.Fatalf("backups = %+v", list)
	}
	byFile := map[string]BackupInfo{}
	for _, b := range list {
		byFile[b.File] = b
		if b.Date == "" {
			t.Errorf("%s has no date", b.File)
		}
	}
	if b := byFile["broken.json"]; b.Readable || b.Problem != "not a roster file" {
		t.Errorf("broken backup: %+v", b)
	}
	counts := map[int][]string{}
	for _, b := range list {
		if b.Readable {
			counts[b.Characters] = b.Names
		}
	}
	if strings.Join(counts[2], ",") != "ONE,TWO" || strings.Join(counts[1], ",") != "THREE" {
		t.Errorf("counts/names = %v", counts)
	}
	// Newest first.
	for i := 1; i < len(list); i++ {
		if list[i-1].modTime < list[i].modTime {
			t.Errorf("not newest first: %v before %v", list[i-1].File, list[i].File)
		}
	}
}

func TestRestoreBackup(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "old one")
	createMage(t, s, "old two")
	walkTo(t, s, "settings")
	s.ResetRoster()
	var oldBackup string
	for _, b := range s.ListBackups() {
		if b.Characters == 2 {
			oldBackup = b.File
		}
	}
	createMage(t, s, "current")
	current, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))

	walkTo(t, s, "settings")
	v := s.RestoreBackup(oldBackup)
	if v.Error != "" || len(v.Roster) != 2 || v.Roster[0].Name != "OLD ONE" || v.Location != "settings" {
		t.Fatalf("restore: error=%q roster=%+v at %q", v.Error, v.Roster, v.Location)
	}
	if !strings.Contains(v.Notice, "before-restore") {
		t.Errorf("notice should name the safety backup: %q", v.Notice)
	}
	// The roster that was replaced is kept, byte for byte.
	var safety string
	for _, b := range s.ListBackups() {
		if strings.Contains(b.File, "before-restore") {
			safety = b.File
		}
	}
	got, err := os.ReadFile(filepath.Join(dir, "backups", safety))
	if err != nil || string(got) != string(current) {
		t.Errorf("safety backup %q does not match the replaced roster (err %v)", safety, err)
	}
	// Persisted: a fresh session sees the restored roster.
	if names := savedNames(t, dir); strings.Join(names, ",") != "OLD ONE,OLD TWO" {
		t.Errorf("saved after restore = %v", names)
	}
	if v := reopened(t, s, dir).State(); len(v.Roster) != 2 {
		t.Errorf("reload after restore = %+v", v.Roster)
	}
	// And the replaced roster can be restored back.
	if v := s.RestoreBackup(safety); v.Error != "" || len(v.Roster) != 1 || v.Roster[0].Name != "CURRENT" {
		t.Errorf("restore the safety backup: error=%q roster=%+v", v.Error, v.Roster)
	}
}

func TestRestoreBackupRefusals(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "keeper")
	walkTo(t, s, "settings")
	s.ResetRoster()
	createMage(t, s, "present")
	os.WriteFile(filepath.Join(dir, "backups", "broken.json"), []byte("{oops"), 0644)
	backupsBefore := len(s.ListBackups())
	walkTo(t, s, "settings")

	for file, want := range map[string]string{
		"":                "Choose a backup from the list.",
		"../roster1.json": "Choose a backup from the list.",
		"/etc/passwd":     "Choose a backup from the list.",
		"broken.json":     "Cannot restore broken.json: not a roster file.",
		"missing.json":    "Cannot restore missing.json: cannot read file.",
	} {
		walkTo(t, s, "settings")
		v := s.RestoreBackup(file)
		if v.Error != want || len(v.Roster) != 1 || v.Roster[0].Name != "PRESENT" {
			t.Errorf("restore %q: error=%q roster=%+v", file, v.Error, v.Roster)
		}
	}
	if n := len(s.ListBackups()); n != backupsBefore {
		t.Errorf("refused restores should not create backups: %d -> %d", backupsBefore, n)
	}

	walkTo(t, s, "training")
	mustOK(t, s.StartCreation("busy"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	var good string
	for _, b := range s.ListBackups() {
		if b.Readable {
			good = b.File
		}
	}
	if v := s.RestoreBackup(good); v.Error != "Finish or discard the character being created first." {
		t.Errorf("restore during creation: %q", v.Error)
	}
}

func TestBackupRestoreLeavesTerminalSavesAlone(t *testing.T) {
	s, _ := newTestSession(t)
	createMage(t, s, "pip")
	walkTo(t, s, "settings")
	s.ResetRoster()
	walkTo(t, s, "settings")
	s.RestoreBackup(s.ListBackups()[0].File)
	walkTo(t, s, "training")
	s.RenameCharacter(0, "peregrin")
	terminal := filepath.Join(os.Getenv("HOME"), ".config", "wizardry")
	if _, err := os.Stat(terminal); !os.IsNotExist(err) {
		t.Errorf("GUI must not create or write %s (stat err %v)", terminal, err)
	}
}
