package app

import "github.com/Senior-Geek/istary/engine"

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
