package engine

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failStep makes writeFileAtomic fail at the named step until the test ends.
func failStep(t *testing.T, step string) error {
	t.Helper()
	injected := fmt.Errorf("injected failure at %s", step)
	saveStepHook = func(s string) error {
		if s == step {
			return injected
		}
		return nil
	}
	t.Cleanup(func() { saveStepHook = nil })
	return injected
}

// dirNames lists the names in dir, sorted.
func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// savedGame returns a game with a roster and party, saved once, and the path
// and bytes of its roster file.
func savedGame(t *testing.T) (*GameState, string, []byte) {
	t.Helper()
	home := freshHome(t)
	g := newTestGame(t, 1)
	c := sampleCharacter()
	g.Town.Roster.Add(c)
	g.Town.Roster.Add(NewCharacter("BENCHWARMER", Human, Fighter, Neutral))
	g.Town.Party.Members = []*Character{c}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".config", "wizardry", "roster1.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return g, path, before
}

// The atomic write produces exactly the bytes os.WriteFile did, for the same
// data (the save golden, TestSaveFileFormat, checks the data itself).
func TestAtomicWriteSameBytesAsWriteFile(t *testing.T) {
	_, path, atomic := savedGame(t)
	plain := filepath.Join(t.TempDir(), "plain.json")
	if err := os.WriteFile(plain, atomic, 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(path, atomic, 0644); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(path)
	p, _ := os.ReadFile(plain)
	if !bytes.Equal(a, p) {
		t.Fatal("atomic write and os.WriteFile differ")
	}
}

// A failure at any step before the rename returns the error, leaves the
// previous save byte-identical and removes the temporary file.
func TestFailedSaveKeepsThePreviousSave(t *testing.T) {
	for _, step := range []string{"write", "sync", "close", "rename"} {
		t.Run(step, func(t *testing.T) {
			g, path, before := savedGame(t)
			names := dirNames(t, filepath.Dir(path))
			g.Town.Roster.Characters[0].Gold = 123456 // a change that must not reach disk
			injected := failStep(t, step)
			err := g.Save()
			var nd *NotDurableError
			if !errors.Is(err, injected) || errors.As(err, &nd) {
				t.Fatalf("Save error = %v, want the injected failure, not replaced", err)
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(after, before) {
				t.Fatal("the previous save changed")
			}
			if got := dirNames(t, filepath.Dir(path)); strings.Join(got, ",") != strings.Join(names, ",") {
				t.Fatalf("directory after failure = %v, want %v (temporary file left?)", got, names)
			}
			saveStepHook = nil
			g2 := newTestGame(t, 1)
			if err := g2.Load(); err != nil || g2.Town.Roster.Characters[0].Gold != 777 {
				t.Fatalf("previous save does not load: %v", err)
			}
		})
	}
}

// Failing to sync the directory comes after the rename: the new save is in
// place and the error says so.
func TestDirectorySyncFailureIsReportedAfterTheSave(t *testing.T) {
	g, path, _ := savedGame(t)
	g.Town.Roster.Characters[0].Gold = 4242
	injected := failStep(t, "syncdir")
	err := g.Save()
	var nd *NotDurableError
	if !errors.Is(err, injected) || !errors.As(err, &nd) || !strings.Contains(err.Error(), "saved, but") {
		t.Fatalf("Save error = %v", err)
	}
	saveStepHook = nil
	g2 := newTestGame(t, 1)
	if err := g2.Load(); err != nil || g2.Town.Roster.Characters[0].Gold != 4242 {
		t.Fatalf("the new save should be in place: %v", err)
	}
	if names := dirNames(t, filepath.Dir(path)); len(names) != 1 {
		t.Fatalf("directory = %v", names)
	}
}

// Without a previous save, a failed first save leaves no file at all.
func TestFailedFirstSaveLeavesNothing(t *testing.T) {
	home := freshHome(t)
	g := newTestGame(t, 1)
	g.Town.Roster.Add(sampleCharacter())
	injected := failStep(t, "rename")
	if err := g.Save(); !errors.Is(err, injected) {
		t.Fatalf("Save error = %v", err)
	}
	if names := dirNames(t, filepath.Join(home, ".config", "wizardry")); len(names) != 0 {
		t.Fatalf("directory = %v, want empty", names)
	}
}

// Repeated saves each replace the file completely and leave nothing behind.
func TestRepeatedSaves(t *testing.T) {
	g, path, _ := savedGame(t)
	for i := 0; i < 50; i++ {
		g.Town.Roster.Characters[0].Gold = i
		if i%2 == 0 {
			g.Town.Roster.Add(NewCharacter(fmt.Sprintf("RECRUIT %d", i), Human, Fighter, Neutral))
		}
		if err := g.Save(); err != nil {
			t.Fatal(err)
		}
		g2 := newTestGame(t, 1)
		if err := g2.Load(); err != nil {
			t.Fatalf("save %d does not load: %v", i, err)
		}
		if g2.Town.Roster.Characters[0].Gold != i || len(g2.Town.Roster.Characters) != len(g.Town.Roster.Characters) {
			t.Fatalf("save %d: reloaded roster differs", i)
		}
	}
	if names := dirNames(t, filepath.Dir(path)); len(names) != 1 || names[0] != "roster1.json" {
		t.Fatalf("directory = %v", names)
	}
}

// Rosters saved before this change load, and saving them again writes the
// bytes the old writer (os.WriteFile at a1eef2a) wrote for the same roster:
// save_roster1.resaved.golden.json was produced that way. (It differs from
// the original golden only where Load works out each AC from what is
// worn, finding #35: the golden's AC 4 and AC 0 were hand-written, and
// every AC is 10 for what they wear. Until COMBAT-1 it also differed in
// known spells, which Load widened, #32: ALDRIC now keeps DIOS and BADIOS.) The
// legacy upstream roster loads, saves and reloads.
func TestExistingRostersStillLoadAndResave(t *testing.T) {
	home := freshHome(t)
	dir := filepath.Join(home, ".config", "wizardry")
	golden, err := os.ReadFile(filepath.Join("testdata", "save_roster1.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "roster1.json")
	if err := os.WriteFile(path, golden, 0644); err != nil {
		t.Fatal(err)
	}
	g := newTestGame(t, 1)
	if err := g.Load(); err != nil || len(g.Town.Roster.Characters) != 2 || g.Town.Party.Size() != 1 {
		t.Fatalf("golden save: err=%v roster=%d party=%d", err, len(g.Town.Roster.Characters), g.Town.Party.Size())
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "save_roster1.resaved.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(path); !bytes.Equal(again, want) {
		t.Fatal("re-saving the golden save wrote different bytes from the old writer")
	}

	home = freshHome(t)
	installLegacySave(t, home, "1")
	g = newTestGame(t, 1)
	if err := g.Load(); err != nil || len(g.Town.Roster.Characters) == 0 {
		t.Fatalf("legacy save: %v", err)
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	g2 := newTestGame(t, 1)
	if err := g2.Load(); err != nil || len(g2.Town.Roster.Characters) != len(g.Town.Roster.Characters) {
		t.Fatalf("migrated legacy save does not reload: %v", err)
	}
}

// The replacement keeps the old file's permissions; a new file gets 0644
// less the umask, as os.WriteFile gave it.
func TestSaveKeepsPermissions(t *testing.T) {
	g, path, _ := savedGame(t)
	plain := filepath.Join(t.TempDir(), "plain")
	os.WriteFile(plain, nil, 0644)
	want, _ := os.Stat(plain)
	if fi, _ := os.Stat(path); fi.Mode().Perm() != want.Mode().Perm() {
		t.Fatalf("new save mode %v, os.WriteFile gives %v", fi.Mode().Perm(), want.Mode().Perm())
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0600 {
		t.Fatalf("mode after save %v, want 0600", fi.Mode().Perm())
	}
}

// A roster file that is a symbolic link stays a link; the file it points to
// is replaced, as os.WriteFile wrote through the link.
func TestSaveThroughSymlink(t *testing.T) {
	g, path, before := savedGame(t)
	elsewhere := filepath.Join(t.TempDir(), "real-roster.json")
	if err := os.WriteFile(elsewhere, before, 0644); err != nil {
		t.Fatal(err)
	}
	os.Remove(path)
	if err := os.Symlink(elsewhere, path); err != nil {
		t.Fatal(err)
	}
	g.Town.Roster.Characters[0].Gold = 99
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(path); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("the roster link was replaced by a file")
	}
	if got, _ := os.ReadFile(elsewhere); bytes.Equal(got, before) {
		t.Fatal("the linked file was not updated")
	}
	if names := dirNames(t, filepath.Dir(elsewhere)); len(names) != 1 {
		t.Fatalf("link target directory = %v", names)
	}
}

// A real file-system failure, not an injected one: with the save folder
// read-only the temporary file cannot be created, Save reports it and the
// previous save is untouched.
func TestSaveIntoReadOnlyFolder(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	g, path, before := savedGame(t)
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	g.Town.Roster.Characters[0].Gold = 1
	if err := g.Save(); err == nil {
		t.Fatal("Save into a read-only folder succeeded")
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, before) {
		t.Fatal("the previous save changed")
	}
	if names := dirNames(t, dir); len(names) != 1 {
		t.Fatalf("directory = %v", names)
	}
}

// Restoring a backup replaces the roster the same way: a failed restore
// leaves the current roster as it was; a failed backup leaves no file.
func TestRestoreAndBackupAreAtomic(t *testing.T) {
	g, path, before := savedGame(t)
	backup := filepath.Join(t.TempDir(), "backup.json")
	if err := BackupRoster(g, backup); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(backup); !bytes.Equal(got, before) {
		t.Fatal("backup differs from the roster")
	}
	g.Town.Roster.Characters[0].Gold = 5
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	current, _ := os.ReadFile(path)

	injected := failStep(t, "rename")
	if err := RestoreRoster(g, backup); !errors.Is(err, injected) {
		t.Fatalf("restore error = %v", err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, current) {
		t.Fatal("a failed restore changed the roster")
	}
	other := filepath.Join(t.TempDir(), "other.json")
	if err := BackupRoster(g, other); !errors.Is(err, injected) {
		t.Fatalf("backup error = %v", err)
	}
	if names := dirNames(t, filepath.Dir(other)); len(names) != 0 {
		t.Fatalf("failed backup left %v", names)
	}
	if names := dirNames(t, filepath.Dir(path)); len(names) != 1 {
		t.Fatalf("roster folder = %v", names)
	}

	saveStepHook = nil
	if err := RestoreRoster(g, backup); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, before) || g.Town.Roster.Characters[0].Gold != 777 {
		t.Fatal("restore did not bring the backup back")
	}
}
