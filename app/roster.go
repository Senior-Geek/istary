package app

import (
	"errors"
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// ErrNoSuchCharacter is returned for a roster index with no character.
var ErrNoSuchCharacter = errors.New("no character at that roster position")

// ErrInvalidName is returned by ValidateName.
var ErrInvalidName = errors.New("names may use up to 15 letters, digits, spaces, hyphens and apostrophes")

// ClearRoster empties the roster and party for a fresh start and closes any
// town prompt. It is only available in town (not during creation). It does
// not save or back up; callers back up first with engine.BackupRoster and
// then call Save.
func (c *Controller) ClearRoster() error {
	if c.Game.Phase != engine.PhaseTown {
		return ErrWrongState
	}
	town := c.Game.Town
	town.Roster.Characters = nil
	town.Party.Members = nil
	town.EditChar = nil
	town.InputMode = engine.InputNone
	town.InputBuf = ""
	town.Message = ""
	town.Message2 = ""
	town.PendingCreateName = ""
	return nil
}

// RenameCharacter renames the roster character at index (0-based), with the
// Utilities menu's rules: the name is trimmed and uppercased, must not be
// empty ("NAME CANNOT BE EMPTY") and must not match any roster name,
// including the character's own ("NAME ALREADY IN USE"). Not available
// during creation, nor for a character out on a saved expedition (the
// expedition refers to its members by name).
func (c *Controller) RenameCharacter(index int, newName string) error {
	if c.Game.Phase == engine.PhaseCreation {
		return ErrWrongState
	}
	roster := c.Game.Town.Roster.Characters
	if index < 0 || index >= len(roster) || roster[index] == nil {
		return ErrNoSuchCharacter
	}
	if c.Game.OnExpedition(roster[index].Name) {
		return ErrOnExpedition
	}
	newName = strings.ToUpper(strings.TrimSpace(newName))
	if newName == "" {
		return &Refusal{Message: "NAME CANNOT BE EMPTY"}
	}
	for _, ch := range roster {
		if ch != nil && ch.Name == newName {
			return &Refusal{Message: "NAME ALREADY IN USE"}
		}
	}
	roster[index].Name = newName
	return nil
}

// ValidateName reports whether name is acceptable at the character-name
// prompt as typed: at most 15 characters of A-Z, 0-9, space, - and '
// (lowercase letters are fine; they are uppercased). Front ends that take a
// whole name at once use it to reject what the prompt would have dropped.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrEmptyName
	}
	if NormalizeName(trimmed) != strings.ToUpper(trimmed) {
		return ErrInvalidName
	}
	return nil
}
