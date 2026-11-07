package main

// GUI combat (M7a). The GUI has no dungeon yet (maze movement is not in the
// controller), so combat is reached through a practice encounter: a real
// encounter from the engine at the maze entrance, fought by copies of the
// current party in a separate game. The practice game is never saved and its
// results are discarded, so the roster is unchanged whatever happens.
//
// Every combat method calls app.Controller operations on the practice game
// and returns a View whose Combat field is app.CombatView, the only combat
// data the front end receives. Pacing (when to Advance) is the front end's.

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// errNoPractice is returned by combat methods when no encounter is running.
var errNoPractice = errors.New("No encounter is in progress.")

// OrderRequest is an order chosen in the front end. Slots are 1-based as
// shown; Group is a monster group number (1-based); Ally is a party slot;
// Item is an index into the character's items (-1 for none).
type OrderRequest struct {
	Slot   int    `json:"slot"`
	Action string `json:"action"` // fight, spell, parry, run, use, dispel
	Group  int    `json:"group"`
	Spell  string `json:"spell"`
	Ally   int    `json:"ally"`
	Item   int    `json:"item"`
}

var orderActions = map[string]engine.CombatAction{
	"fight": engine.ActionFight, "spell": engine.ActionSpell, "parry": engine.ActionParry,
	"run": engine.ActionRun, "use": engine.ActionUse, "dispel": engine.ActionDispel,
}

// cloneCharacter copies a character through its saved form, so the practice
// game shares nothing with the roster.
func cloneCharacter(ch *engine.Character) (*engine.Character, error) {
	raw, err := json.Marshal(ch)
	if err != nil {
		return nil, err
	}
	var out engine.Character
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PracticeEncounter starts a practice encounter for the current party at the
// maze entrance. withChest marks the encounter as guarding a chest (the
// entrance's encounters never do), to try the chest; it is a practice
// setting, not a rule.
func (s *Session) PracticeEncounter(withChest bool) View {
	return s.combatDo(func() error {
		if s.practice != nil {
			return errors.New("An encounter is already in progress.")
		}
		g := s.ctrl.Game
		if g.Phase != engine.PhaseTown || len(g.Town.Party.Members) == 0 {
			return errors.New("Form a party in the Tavern first.")
		}
		pg := engine.New(s.scenario)
		pg.Title = nil
		pg.MazeLevel, pg.PlayerX, pg.PlayerY, pg.Facing = 0, 0, 0, engine.North
		for _, m := range g.Town.Party.Members {
			cp, err := cloneCharacter(m)
			if err != nil {
				return fmt.Errorf("copy %s: %w", m.Name, err)
			}
			pg.Town.Party.Members = append(pg.Town.Party.Members, cp)
		}
		for try := 0; try < 50; try++ {
			pg.Combat = engine.NewCombat(pg)
			if len(pg.Combat.Groups) > 0 {
				break
			}
		}
		if len(pg.Combat.Groups) == 0 {
			return errors.New("No monsters appeared. Try again.")
		}
		pg.Combat.HasChest = pg.Combat.HasChest || withChest
		pg.Phase = engine.PhaseCombat
		s.practice = app.New(pg)
		s.practiceChest = withChest
		return nil
	})
}

// combatDo runs a combat action on the practice game under the lock. The
// practice game is never saved. When combat is over the practice game is
// discarded and a notice says how it ended.
func (s *Session) combatDo(action func() error) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	err := action()
	if p := s.practice; p != nil && (p.Game.Phase != engine.PhaseCombat || p.Game.Combat == nil) {
		switch {
		case p.Game.Phase == engine.PhaseTown:
			s.notice = "The party perished. Practice encounter over; the roster is unchanged."
		case p.Game.MazeMessage != "":
			s.notice = p.Game.MazeMessage + " Practice encounter over; the roster is unchanged."
		default:
			s.notice = "Practice encounter over; the roster is unchanged."
		}
		s.practice = nil
	}
	return s.view(err)
}

// onPractice runs op against the practice controller.
func (s *Session) onPractice(op func(c *app.Controller) error) View {
	return s.combatDo(func() error {
		if s.practice == nil {
			return errNoPractice
		}
		return op(s.practice)
	})
}

// member returns the practice party member in a 1-based slot.
func member(c *app.Controller, slot int) (*engine.Character, error) {
	party := c.Game.Town.Party.Members
	if slot < 1 || slot > len(party) || party[slot-1] == nil {
		return nil, app.ErrNoSuchPartyMember
	}
	return party[slot-1], nil
}

// AbandonPractice ends the practice encounter at once.
func (s *Session) AbandonPractice() View {
	return s.combatDo(func() error {
		if s.practice == nil {
			return errNoPractice
		}
		s.practice = nil
		s.notice = "Practice encounter abandoned; the roster is unchanged."
		return nil
	})
}

// CombatStart leaves the opening pause.
func (s *Session) CombatStart() View {
	return s.onPractice(func(c *app.Controller) error { _, err := c.StartCombat(); return err })
}

// CombatFriendly answers a friendly encounter: fight or leave in peace.
func (s *Session) CombatFriendly(fight bool) View {
	return s.onPractice(func(c *app.Controller) error {
		if fight {
			return c.FightFriendly()
		}
		return c.LeaveInPeace()
	})
}

// CombatSelect chooses the member to give an order to.
func (s *Session) CombatSelect(slot int) View {
	return s.onPractice(func(c *app.Controller) error {
		ch, err := member(c, slot)
		if err != nil {
			return err
		}
		return c.SelectCombatActor(ch)
	})
}

// CombatOrder gives an order; the controller validates it.
func (s *Session) CombatOrder(r OrderRequest) View {
	return s.onPractice(func(c *app.Controller) error {
		ch, err := member(c, r.Slot)
		if err != nil {
			return err
		}
		act, ok := orderActions[r.Action]
		if !ok {
			return app.ErrInvalidChoice
		}
		o := app.Order{Action: act, Group: r.Group - 1, Spell: r.Spell, Item: r.Item}
		if r.Ally > 0 {
			if o.Ally, err = member(c, r.Ally); err != nil {
				return err
			}
		}
		return c.SetOrder(ch, o)
	})
}

// CombatFinishOrders moves to confirmation.
func (s *Session) CombatFinishOrders() View {
	return s.onPractice(func(c *app.Controller) error { return c.FinishOrders() })
}

// CombatBack clears every order (B)ack at confirmation).
func (s *Session) CombatBack() View {
	return s.onPractice(func(c *app.Controller) error { return c.ReturnToOrders() })
}

// CombatCommit executes the round and shows its first block.
func (s *Session) CombatCommit() View {
	return s.onPractice(func(c *app.Controller) error { _, err := c.Commit(); return err })
}

// CombatAdvance shows the next block of the round.
func (s *Session) CombatAdvance() View {
	return s.onPractice(func(c *app.Controller) error { _, err := c.Advance(); return err })
}

// CombatChooseBoon answers HAMAN's boon prompt (1-3).
func (s *Session) CombatChooseBoon(n int) View {
	return s.onPractice(func(c *app.Controller) error { return c.ChooseBoon(n - 1) })
}

// CombatEndRound ends a round that has been shown.
func (s *Session) CombatEndRound() View {
	return s.onPractice(func(c *app.Controller) error { _, err := c.EndRound(); return err })
}

// CombatLeave leaves combat after victory or defeat.
func (s *Session) CombatLeave() View {
	return s.onPractice(func(c *app.Controller) error { return c.LeaveCombat() })
}

// ChestAction has the member in slot open, inspect, CALFO or disarm the
// chest, or leaves it ("leave", slot ignored).
func (s *Session) ChestAction(action string, slot int) View {
	return s.onPractice(func(c *app.Controller) error {
		if action == "leave" {
			return c.LeaveChest()
		}
		ch, err := member(c, slot)
		if err != nil {
			return err
		}
		switch action {
		case "open":
			return c.OpenChest(ch)
		case "inspect":
			return c.InspectChest(ch)
		case "calfo":
			return c.CastCalfo(ch)
		case "disarm":
			return c.DisarmChest(ch)
		}
		return app.ErrInvalidChoice
	})
}

// ChestContinue moves a chest result on: the next block, or, after the
// last, the rewards or the chest menu again.
func (s *Session) ChestContinue() View {
	return s.onPractice(func(c *app.Controller) error {
		done, err := c.AdvanceChest()
		if err != nil || !done {
			return err
		}
		_, err = c.EndChestResult()
		return err
	})
}
