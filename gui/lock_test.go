//go:build unix

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// Two windows on one save folder: the second refuses to start, says who
// holds the folder, and changes nothing, whatever it is asked; once the
// first closes, Try again opens it.
func TestSecondWindowIsRefused(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "radagast")
	path := filepath.Join(dir, "roster1.json")
	before, _ := os.ReadFile(path)

	second := NewSession(dir)
	v := second.State()
	if !strings.Contains(v.Fatal, "Another Istary window is using this save folder") ||
		!strings.Contains(v.Fatal, "process ") || !strings.Contains(v.Fatal, "Try again") {
		t.Fatalf("second window: %q", v.Fatal)
	}
	for _, op := range []func() View{
		func() View { return second.RenameCharacter(0, "SARUMAN") },
		second.ResetRoster,
		second.RealExpedition,
		second.ReloadRoster,
		second.TryAgain,
	} {
		if v := op(); v.Fatal == "" {
			t.Fatal("the second window acted on a held folder")
		}
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, before) {
		t.Fatal("the second window changed the save")
	}

	s.Close() // as on quitting
	if v := s.RenameCharacter(0, "SARUMAN"); v.Fatal == "" {
		t.Fatal("a closed window still acts")
	}
	if v := second.TryAgain(); v.Fatal != "" || len(v.Roster) != 1 {
		t.Fatalf("try again after the first closed: %q", v.Fatal)
	}
	if v := second.RenameCharacter(0, "SARUMAN"); v.Error != "" {
		t.Fatal(v.Error)
	}
	second.Close()
}

// TestLockHolderProcess is not a test: TestLockSurvivesACrash runs it in a
// child process, which takes the lock and kills itself without letting go.
func TestLockHolderProcess(t *testing.T) {
	dir := os.Getenv("ISTARY_LOCK_CHILD")
	if dir == "" {
		t.Skip("helper process")
	}
	if _, err := lockSaveFolder(dir); err != nil {
		os.Exit(2)
	}
	os.Stdout.WriteString("locked\n")
	syscall.Kill(os.Getpid(), syscall.SIGKILL)
}

// A window that crashes (killed while holding the folder) leaves no lock
// behind: the next start opens the folder, though the lock file and its
// stale text remain.
func TestLockSurvivesACrash(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "saves")
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockHolderProcess$")
	cmd.Env = append(os.Environ(), "ISTARY_LOCK_CHILD="+dir)
	out, err := cmd.Output()
	if !strings.Contains(string(out), "locked") {
		t.Fatalf("child: %q %v", out, err)
	}
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); !ok || !ws.Signaled() || ws.Signal() != syscall.SIGKILL {
		t.Fatalf("the child was not killed: %v", err)
	}
	stale, err := os.ReadFile(filepath.Join(dir, lockFileName))
	if err != nil || !strings.Contains(string(stale), "process ") {
		t.Fatalf("lock file after the crash: %q %v", stale, err)
	}
	s := NewSession(dir)
	if v := s.State(); v.Fatal != "" {
		t.Fatalf("start after a crash: %q", v.Fatal)
	}
	s.Close()
}

// Closing releases the folder at once, and closing twice is harmless.
func TestCloseReleasesTheFolder(t *testing.T) {
	s, dir := newTestSession(t)
	s.Close()
	s.Close()
	r := NewSession(dir)
	if v := r.State(); v.Fatal != "" {
		t.Fatalf("start after close: %q", v.Fatal)
	}
	r.Close()
}
