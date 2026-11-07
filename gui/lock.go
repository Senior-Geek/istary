//go:build unix

package main

// The save folder's single-writer lock (M6b-2, decision L). One Istary
// window saves to a folder at a time: a second one opening the same folder
// refuses to start and changes nothing.
//
// The lock is an advisory flock(2) on "istary.lock" in the save folder,
// held on an open file for as long as the session runs. The kernel releases
// it when the file is closed or the process ends in any way, a crash or a
// forced quit included, so there is never a stale lock to clear: after a
// crash the next start simply takes it. The file's text (the holder's
// process id and start time) only explains a refusal; it is never trusted
// to decide anything, and the file itself is left in place.
//
// Where the file system has no flock (some network volumes), the session
// runs unlocked and says so, rather than strand the player.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// lockFileName is the lock file in the save folder.
const lockFileName = "istary.lock"

// saveLock is a held lock; nil when not held.
type saveLock struct{ f *os.File }

// lockedError is returned when another session holds the folder.
type lockedError struct{ holder string }

func (e *lockedError) Error() string {
	msg := "Another Istary window is using this save folder"
	if e.holder != "" {
		msg += " (" + e.holder + ")"
	}
	return msg + ". Close it, then choose Try again."
}

// errLockUnsupported is returned when the folder's file system cannot lock.
var errLockUnsupported = errors.New("this save folder's file system does not support locking")

// lockSaveFolder takes the folder's lock, or reports who holds it.
func lockSaveFolder(dir string) (*saveLock, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, lockFileName), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		holder, _ := os.ReadFile(f.Name())
		f.Close()
		switch {
		case errors.Is(err, syscall.EWOULDBLOCK):
			return nil, &lockedError{holder: strings.TrimSpace(string(holder))}
		case errors.Is(err, syscall.ENOTSUP), errors.Is(err, syscall.EOPNOTSUPP), errors.Is(err, syscall.ENOLCK):
			return nil, errLockUnsupported
		}
		return nil, fmt.Errorf("lock %s: %w", f.Name(), err)
	}
	// Say who holds it, for a refusal elsewhere. Best effort.
	f.Truncate(0)
	f.WriteAt([]byte(fmt.Sprintf("process %d, since %s", os.Getpid(), time.Now().Format("15:04 on 2 Jan"))), 0)
	return &saveLock{f: f}, nil
}

// release lets go of the lock. Safe on nil and twice.
func (l *saveLock) release() {
	if l == nil || l.f == nil {
		return
	}
	syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	l.f.Close()
	l.f = nil
}
