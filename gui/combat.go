package main

// GUI combat (M7a). Combat is reached through a practice encounter (a real
// encounter from the engine at the maze entrance) or a practice expedition
// (dungeon.go, M6a), both played by copies of the current party in a
// separate game. The practice game is never saved and its results are
// discarded, so the roster is unchanged whatever happens.
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

// errPracticeRunning is returned when a practice is already under way.
var errPracticeRunning = errors.New("An encounter or expedition is already in progress.")

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

// practiceGame returns a new, never-saved game in town whose party is a
// copy of the current party.
func (s *Session) practiceGame() (*engine.GameState, error) {
	g := s.ctrl.Game
	if g.Phase != engine.PhaseTown || len(g.Town.Party.Members) == 0 {
		return nil, errors.New("Form a party in the Tavern first.")
	}
	pg := engine.New(s.scenario)
	pg.Title = nil
	pg.Phase = engine.PhaseTown
	for _, m := range g.Town.Party.Members {
		cp, err := cloneCharacter(m)
		if err != nil {
			return nil, fmt.Errorf("copy %s: %w", m.Name, err)
		}
		pg.Town.Party.Members = append(pg.Town.Party.Members, cp)
	}
	return pg, nil
}

// PracticeEncounter starts a practice encounter for the current party at the
// maze entrance. withChest marks the encounter as guarding a chest (the
// entrance's encounters never do), to try the chest; it is a practice
// setting, not a rule.
func (s *Session) PracticeEncounter(withChest bool) View {
	return s.combatDo(func() error {
		if s.practice != nil {
			return errPracticeRunning
		}
		pg, err := s.practiceGame()
		if err != nil {
			return err
		}
		pg.MazeLevel, pg.PlayerX, pg.PlayerY, pg.Facing = 0, 0, 0, engine.North
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
		s.practiceChest, s.expedition = withChest, false
		return nil
	})
}

// combatDo runs an action on the practice game under the lock. The
// practice game is never saved. A practice encounter is over when its
// combat is; an expedition goes on in the maze and at Camp and is over when
// the party is back in town (by the stairs, or perished). Then the practice
// game is discarded and a notice says how it ended.
func (s *Session) combatDo(action func() error) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice = ""
	err := action()
	if p := s.practice; p != nil {
		g := p.Game
		inCombat := g.Phase == engine.PhaseCombat && g.Combat != nil
		exploring := s.expedition && (g.Phase == engine.PhaseMaze || g.Phase == engine.PhaseCamp)
		if !inCombat && !exploring {
			what := "Practice encounter"
			if s.expedition {
				what = "Practice expedition"
			}
			switch {
			case g.Phase == engine.PhaseTown && (!s.expedition || !anyAlive(g)):
				s.notice = "The party perished. " + what + " over; the roster is unchanged."
			case g.Phase == engine.PhaseTown:
				s.notice = "The party is back in the castle. " + what + " over; the roster is unchanged."
			case g.MazeMessage != "":
				s.notice = g.MazeMessage + " " + what + " over; the roster is unchanged."
			default:
				s.notice = what + " over; the roster is unchanged."
			}
			s.practice, s.expedition = nil, false
		}
	}
	return s.view(err)
}

// anyAlive reports whether any party member is alive.
func anyAlive(g *engine.GameState) bool {
	for _, m := range g.Town.Party.Members {
		if m != nil && m.IsAlive() {
			return true
		}
	}
	return false
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

// AbandonPractice ends the practice encounter or expedition at once.
func (s *Session) AbandonPractice() View {
	return s.combatDo(func() error {
		if s.practice == nil {
			return errNoPractice
		}
		if s.expedition {
			s.notice = "Practice expedition abandoned; the roster is unchanged."
		} else {
			s.notice = "Practice encounter abandoned; the roster is unchanged."
		}
		s.practice, s.expedition = nil, false
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
