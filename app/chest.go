package app

// The chest after combat (M5e): O)PEN, C)ALFO, I)NSPECT, D)ISARM, L)EAVE,
// and stepping through each result. The mechanics, chances and messages are
// the engine's (engine/combat.go, Pascal REWARDS), unchanged; the
// controller checks who may act and runs the transitions the terminal ran.
// The "WHO (#) WILL ...?" prompts and the PAUSE2 waits stay in the front
// end. No timers, no random numbers of its own.

import (
	"errors"

	"github.com/Senior-Geek/istary/engine"
)

// Errors returned by the chest operations.
var (
	ErrNotAtChest      = errors.New("no chest is waiting for an action")
	ErrNoChestResult   = errors.New("no chest result is being shown")
	ErrChestResultOpen = errors.New("the chest result has lines not yet shown")
)

// chestActor checks the phase and that ch is in the party and can act (the
// terminal's "1-6, alive" check), and returns ch's slot.
func (c *Controller) chestActor(ch *engine.Character) (*engine.CombatState, int, error) {
	cs, err := c.combatIn(engine.CombatChest, ErrNotAtChest)
	if err != nil {
		return nil, -1, err
	}
	slot := c.partySlot(ch)
	if slot < 0 {
		return nil, -1, ErrNoSuchPartyMember
	}
	if !CanGiveOrder(ch) {
		return nil, -1, ErrCannotAct
	}
	return cs, slot, nil
}

// OpenChest has ch open the chest. A trap may go off (the opener may dodge
// it); the chest is then open and its result is shown.
func (c *Controller) OpenChest(ch *engine.Character) error {
	cs, slot, err := c.chestActor(ch)
	if err != nil {
		return err
	}
	cs.OpenChest(c.Game, slot)
	return nil
}

// InspectChest has ch inspect the trap: a report that may be right or wrong,
// "YOU ALREADY LOOKED!" for a second look, or a fumble that sets it off.
func (c *Controller) InspectChest(ch *engine.Character) error {
	cs, slot, err := c.chestActor(ch)
	if err != nil {
		return err
	}
	cs.InspectChest(c.Game, slot)
	return nil
}

// CastCalfo has ch cast CALFO on the chest. Anyone may try, as in the
// terminal: one who cannot cast it gets "CAN'T CAST CALFO!".
func (c *Controller) CastCalfo(ch *engine.Character) error {
	cs, slot, err := c.chestActor(ch)
	if err != nil {
		return err
	}
	cs.CalfoChest(c.Game, slot)
	return nil
}

// DisarmChest has ch try to disarm the trap: disarmed (the chest opens
// safely), failed (back to the chest), or set off.
func (c *Controller) DisarmChest(ch *engine.Character) error {
	cs, slot, err := c.chestActor(ch)
	if err != nil {
		return err
	}
	cs.DisarmChest(c.Game, slot)
	return nil
}

// LeaveChest leaves the chest unopened: experience, but no treasure.
func (c *Controller) LeaveChest() error {
	cs, err := c.combatIn(engine.CombatChest, ErrNotAtChest)
	if err != nil {
		return err
	}
	cs.LeaveChest()
	return nil
}

// chestNext is the start of the result block after the one on screen, or
// -1 when the block on screen is the last.
func chestNext(cs *engine.CombatState) int {
	i := cs.MessageIndex
	for i < len(cs.Messages) {
		if cs.Messages[i] == engine.ActionSeparator {
			i++
			break
		}
		i++
	}
	if i < len(cs.Messages) {
		return i
	}
	return -1
}

// AdvanceChest shows the next block of a chest result. It returns done when
// the block on screen is the last; then EndChestResult continues.
func (c *Controller) AdvanceChest() (done bool, err error) {
	cs, err := c.combatIn(engine.CombatChestResult, ErrNoChestResult)
	if err != nil {
		return false, err
	}
	if len(cs.Messages) == 0 {
		return false, nil // nothing to show (the terminal waits here too)
	}
	if n := chestNext(cs); n >= 0 {
		cs.MessageIndex = n
		return false, nil
	}
	return true, nil
}

// EndChestResult ends a chest result whose lines have all been shown. If
// the chest was opened or left, the treasure (or, if left, the experience
// only) is shared out and combat moves to victory (complete is true);
// otherwise (an inspection, CALFO, or a failed disarm) the chest menu
// returns.
func (c *Controller) EndChestResult() (complete bool, err error) {
	cs, err := c.combatIn(engine.CombatChestResult, ErrNoChestResult)
	if err != nil {
		return false, err
	}
	if len(cs.Messages) == 0 || chestNext(cs) >= 0 {
		return false, ErrChestResultOpen
	}
	if cs.ChestOpened || cs.ChestLeft {
		cs.FinalizeChest(c.Game)
		return true, nil
	}
	cs.Phase = engine.CombatChest
	cs.ChestSubPhase = engine.ChestMenu
	cs.Messages = nil
	cs.MessageIndex = 0
	return false, nil
}
