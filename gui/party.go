package main

import (
	"errors"
	"fmt"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// JoinParty asks the roster character at index to join the party at
// Gilgamesh's Tavern. If the Tavern accepts, it waits for the character's
// password (View.PendingJoin); otherwise it returns the Tavern's refusal.
func (s *Session) JoinParty(index int) View {
	return s.do(func() error {
		c := s.ctrl
		roster := c.Game.Town.Roster.Characters
		if index < 0 || index >= len(roster) || roster[index] == nil {
			return app.ErrNoSuchCharacter
		}
		ch := roster[index]
		if err := s.at(engine.Tavern); err != nil {
			return err
		}
		needs, err := c.AddMember(ch.Name)
		if err != nil {
			return err
		}
		if needs && c.Game.Town.EditChar != ch {
			c.CancelTavernPassword()
			return errors.New("Another character has the same name; rename one first.")
		}
		return nil
	})
}

// SubmitJoinPassword answers the pending join's password prompt. The
// original asks for the password on every join.
func (s *Session) SubmitJoinPassword(password string) View {
	return s.do(func() error {
		c := s.ctrl
		town := c.Game.Town
		if town.InputMode != engine.InputTavernPassword || town.EditChar == nil {
			return app.ErrWrongState
		}
		name := town.EditChar.Name
		if err := c.SubmitTavernPassword(password); err != nil {
			return err
		}
		s.notice = fmt.Sprintf("%s joined the party.", name)
		return nil
	})
}

// CancelJoin abandons the pending join.
func (s *Session) CancelJoin() View {
	return s.do(func() error { return s.ctrl.CancelTavernPassword() })
}

// LeaveParty removes the party member in slot (0-based) at the Tavern. The
// character stays on the roster.
func (s *Session) LeaveParty(slot int) View {
	return s.do(func() error {
		c := s.ctrl
		members := c.Game.Town.Party.Members
		if slot < 0 || slot >= len(members) || members[slot] == nil {
			return app.ErrNoSuchPartyMember
		}
		name := members[slot].Name
		if err := s.at(engine.Tavern); err != nil {
			return err
		}
		if err := c.RemoveMember(slot); err != nil {
			return err
		}
		s.notice = fmt.Sprintf("%s left the party (still on the roster).", name)
		return nil
	})
}
