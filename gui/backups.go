package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// BackupInfo describes one roster backup in <saveDir>/backups.
type BackupInfo struct {
	File       string   `json:"file"` // file name only
	Date       string   `json:"date"` // when the backup was written
	Characters int      `json:"characters"`
	Names      []string `json:"names"`
	Readable   bool     `json:"readable"`
	Problem    string   `json:"problem"` // why it cannot be restored
	modTime    int64
}

// ListBackups returns the roster backups, newest first, with their dates and
// the characters they contain.
func (s *Session) ListBackups() []BackupInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return nil
	}
	entries, err := os.ReadDir(s.backupDir())
	if err != nil {
		return nil // no backups yet
	}
	var out []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info := BackupInfo{File: e.Name()}
		if fi, err := e.Info(); err == nil {
			info.modTime = fi.ModTime().UnixNano()
			info.Date = fi.ModTime().Format("2006-01-02 15:04:05")
		}
		if state, err := readBackup(filepath.Join(s.backupDir(), e.Name())); err != nil {
			info.Problem = err.Error()
		} else {
			info.Readable = true
			for _, ch := range state.Roster {
				if ch != nil {
					info.Names = append(info.Names, ch.Name)
				}
			}
			info.Characters = len(info.Names)
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].modTime != out[j].modTime {
			return out[i].modTime > out[j].modTime
		}
		return out[i].File > out[j].File
	})
	return out
}

// readBackup parses a backup with the engine's save format.
func readBackup(path string) (*engine.SaveState, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file")
	}
	var state engine.SaveState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("not a roster file")
	}
	return &state, nil
}

// RestoreBackup replaces the roster with the named backup. The current roster
// is backed up first; nothing changes if that backup fails or the chosen file
// is not a readable roster.
func (s *Session) RestoreBackup(file string) View {
	return s.do(func() error {
		c := s.ctrl
		if c.Game.Phase != engine.PhaseTown {
			return errors.New("Finish or discard the character being created first.")
		}
		if file == "" || file != filepath.Base(file) || !strings.HasSuffix(file, ".json") {
			return errors.New("Choose a backup from the list.")
		}
		src := filepath.Join(s.backupDir(), file)
		if _, err := readBackup(src); err != nil {
			return fmt.Errorf("Cannot restore %s: %v.", file, err)
		}
		if err := c.Save(); err != nil {
			return fmt.Errorf("Could not save before backup: %w", err)
		}
		before, err := s.backupPath("before-restore")
		if err != nil {
			return err
		}
		if err := engine.BackupRoster(c.Game, before); err != nil {
			return fmt.Errorf("Backup of the current roster failed, nothing restored: %w", err)
		}
		if err := engine.RestoreRoster(c.Game, src); err != nil {
			return fmt.Errorf("Restore failed (current roster saved as %s): %w", filepath.Base(before), err)
		}
		// Start from a clean session on the restored file.
		if err := s.reload(); err != nil {
			return err
		}
		s.notice = fmt.Sprintf("Restored %s. The previous roster was backed up as %s.", file, filepath.Base(before))
		if s.real {
			s.notice += " The expedition saved in it resumes."
		}
		return nil
	})
}
