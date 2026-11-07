package engine

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// writeFileAtomic replaces path with data so that a reader (or the next
// start) sees either the old contents or the new ones, never a mixture or a
// truncated file:
//
//  1. write data to a new temporary file in the same directory;
//  2. sync it (on macOS Go's File.Sync asks the drive to flush its cache,
//     F_FULLFSYNC) and close it;
//  3. give it the old file's permissions, or perm for a new file;
//  4. rename it over path (atomic within one file system);
//  5. sync the directory so the rename itself is recorded.
//
// If any step before the rename fails, the temporary file is removed and the
// old file is untouched. If path is a symbolic link, the file it points to is
// replaced, as os.WriteFile would write through the link.
//
// Durability: after a successful return the new contents survive a crash or
// power loss as far as the platform's sync calls guarantee. A failure of the
// directory sync (step 5) is a *NotDurableError: the new file is already in
// place. Any other error: the old file is untouched.
func writeFileAtomic(path string, data []byte, perm fs.FileMode) error {
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	dir := filepath.Dir(path)

	tmp, err := createTemp(dir, filepath.Base(path), perm)
	if err != nil {
		return err
	}
	name := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			tmp.Close()
			os.Remove(name)
		}
	}()

	if err := saveStep("write"); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := saveStep("sync"); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := saveStep("close"); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// The umask may have narrowed the mode at creation; match the old file.
	if err := os.Chmod(name, perm); err != nil {
		return err
	}
	if err := saveStep("rename"); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	renamed = true

	if err := saveStep("syncdir"); err != nil {
		return &NotDurableError{Err: err}
	}
	if err := syncDir(dir); err != nil {
		return &NotDurableError{Err: err}
	}
	return nil
}

// NotDurableError is a save that replaced the old file but could not sync
// the folder afterwards: the new contents are in place and are read back,
// but may not survive a crash or power loss. Any other save error means the
// file was not replaced: the previous save is intact.
type NotDurableError struct{ Err error }

func (e *NotDurableError) Error() string {
	return "saved, but could not sync the directory: " + e.Err.Error()
}

func (e *NotDurableError) Unwrap() error { return e.Err }

// createTemp creates a new file next to base in dir, named
// ".<base>.tmp-<random>", with mode perm (less the umask, as for any new
// file). Its name never ends in .json, so nothing reads it as a save.
func createTemp(dir, base string, perm fs.FileMode) (*os.File, error) {
	var b [8]byte
	for i := 0; i < 10; i++ {
		if _, err := rand.Read(b[:]); err != nil {
			return nil, err
		}
		name := filepath.Join(dir, "."+base+".tmp-"+hex.EncodeToString(b[:]))
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, fmt.Errorf("could not create a temporary file in %s", dir)
}

// syncDir syncs a directory so a rename in it is durable.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// saveStep is a failure-injection point for tests: it is nil in the game,
// and a test sets it to fail one named step of writeFileAtomic.
var saveStepHook func(step string) error

func saveStep(step string) error {
	if saveStepHook == nil {
		return nil
	}
	return saveStepHook(step)
}
