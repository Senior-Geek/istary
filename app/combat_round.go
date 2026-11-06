package app

// Combat round lifecycle and playback (M5b). The controller starts combat,
// executes a confirmed round exactly once, releases the round's events one
// block at a time, and ends the round when everything has been shown. The
// front end keeps the keys, the display and the pacing: it calls Advance
// from its own timer. The controller has no timers and draws no random
// numbers of its own; ExecuteRound draws exactly as before.
//
// Information boundary: RevealedEvents and the steps returned by Commit and
// Advance never include an event that has not been released. The engine
// state behind Controller.Game is still public, so a front end that reads
// it directly can see the whole round; see docs/ARCHITECTURE.md.

import (
	"errors"

	"github.com/Senior-Geek/istary/engine"
)

// Errors returned by the round lifecycle.
var (
	ErrNotStarting        = errors.New("combat is not starting")
	ErrNotConfirming      = errors.New("orders are not awaiting confirmation")
	ErrNotPlayingBack     = errors.New("no combat round is being played back")
	ErrPlaybackUnfinished = errors.New("the round has events not yet shown")
	ErrAwaitingChoice     = errors.New("playback is waiting for a choice")
)

// PlaybackStep is one released block of the round. Done is set, with no
// Event, when every block has been released; the round can then be ended.
type PlaybackStep struct {
	Event engine.CombatEvent
	Index int // the event's position in the round, from 0
	Done  bool
}

// RoundOutcome is how a round ended.
type RoundOutcome int

const (
	RoundContinues RoundOutcome = iota // the next round's order entry has begun
	RoundEscaped                       // the party fled; combat is over, back in the maze
	RoundVictory                       // every monster is dead: the chest, or the rewards
	RoundDefeated                      // no party member can fight on
)

// combatIn returns the combat if it is in the given phase.
func (c *Controller) combatIn(phase engine.CombatPhase, err error) (*engine.CombatState, error) {
	cs := c.Game.Combat
	if c.Game.Phase != engine.PhaseCombat || cs == nil || cs.Phase != phase {
		return nil, err
	}
	return cs, nil
}

// firstActor is the first party member offered an order (as the terminal's
// findNextActor from the start), or the party size if none is.
func (c *Controller) firstActor() int {
	party := c.Game.Town.Party.Members
	for i, m := range party {
		if CanGiveOrder(m) {
			return i
		}
	}
	return len(party)
}

// StartCombat leaves the encounter's opening pause. If the monsters
// surprised the party, their free round is executed at once and playback
// begins (playback is true); otherwise order entry begins.
func (c *Controller) StartCombat() (playback bool, err error) {
	cs, err := c.combatIn(engine.CombatInit, ErrNotStarting)
	if err != nil {
		return false, err
	}
	if cs.Surprised == 2 {
		cs.ExecuteRound(c.Game)
		return true, nil
	}
	cs.Phase = engine.CombatChoose
	cs.CurrentActor = c.firstActor()
	return false, nil
}

// ReturnToOrders is B)ack at the confirmation prompt: every order is
// cleared and order entry starts again from the first slot (Classic B)ack
// clears all orders).
func (c *Controller) ReturnToOrders() error {
	cs, err := c.combatIn(engine.CombatConfirm, ErrNotConfirming)
	if err != nil {
		return err
	}
	cs.Phase = engine.CombatChoose
	cs.CurrentActor = 0
	for i := range cs.Actions {
		cs.Actions[i] = engine.PartyAction{}
	}
	return nil
}

// Commit executes the confirmed round, once, and releases its first block.
// A second Commit fails: the round has left the confirmation phase.
func (c *Controller) Commit() (PlaybackStep, error) {
	cs, err := c.combatIn(engine.CombatConfirm, ErrNotConfirming)
	if err != nil {
		return PlaybackStep{}, err
	}
	cs.ExecuteRound(c.Game)
	return c.currentStep(cs), nil
}

// cursor is the index of the released event the display is on.
func cursor(cs *engine.CombatState, evs []engine.CombatEvent) int {
	for i, e := range evs {
		if e.Start == cs.MessageIndex {
			return i
		}
	}
	return -1
}

// currentStep describes the block the playback cursor is on.
func (c *Controller) currentStep(cs *engine.CombatState) PlaybackStep {
	evs := cs.Events()
	k := cursor(cs, evs)
	if k < 0 {
		return PlaybackStep{Done: true}
	}
	return PlaybackStep{Event: evs[k], Index: k}
}

// Advance releases the next block of the round, in order, silent blocks
// included. When every block has been released it returns a step with Done
// set and changes nothing; EndRound then ends the round.
func (c *Controller) Advance() (PlaybackStep, error) {
	cs, err := c.combatIn(engine.CombatExecute, ErrNotPlayingBack)
	if err != nil {
		return PlaybackStep{}, err
	}
	if cs.HamanSelecting {
		return PlaybackStep{}, ErrAwaitingChoice
	}
	evs := cs.Events()
	k := cursor(cs, evs)
	if k+1 >= len(evs) {
		return PlaybackStep{Index: k, Done: true}, nil
	}
	cs.MessageIndex = evs[k+1].Start
	return PlaybackStep{Event: evs[k+1], Index: k + 1}, nil
}

// RevealedEvents returns the round's events released so far, and none
// beyond. Empty when no round is being played back.
func (c *Controller) RevealedEvents() []engine.CombatEvent {
	cs, err := c.combatIn(engine.CombatExecute, ErrNotPlayingBack)
	if err != nil {
		return nil
	}
	evs := cs.Events()
	k := cursor(cs, evs)
	if k < 0 {
		return nil
	}
	return evs[:k+1]
}

// ChooseBoon answers HAMAN or MAHAMAN's "WHICH BOON WILL YOU INVOKE ?"
// (Wiz 2/3), letting playback continue.
func (c *Controller) ChooseBoon(choice int) error {
	cs, err := c.combatIn(engine.CombatExecute, ErrNotPlayingBack)
	if err != nil {
		return err
	}
	if !cs.HamanSelecting {
		return ErrWrongState
	}
	if choice < 0 || choice > 2 {
		return ErrInvalidChoice
	}
	cs.ExecuteHamanChoice(c.Game, choice)
	return nil
}

// EndRound ends a round whose blocks have all been released. The frozen
// round display is dropped and the outcome decided, as Classic does after
// the last message: escape, victory, defeat, or the next round, whose
// formation change and cleared orders happen in engine BeginNextRound
// before any new order is entered.
func (c *Controller) EndRound() (RoundOutcome, error) {
	cs, err := c.combatIn(engine.CombatExecute, ErrNotPlayingBack)
	if err != nil {
		return 0, err
	}
	if cs.HamanSelecting {
		return 0, ErrAwaitingChoice
	}
	if evs := cs.Events(); cursor(cs, evs)+1 < len(evs) {
		return 0, ErrPlaybackUnfinished
	}
	game := c.Game
	cs.DisplayAliveCounts = nil
	cs.DisplayPartySnap = nil

	if cs.Fled {
		game.Combat = nil
		game.Phase = engine.PhaseMaze
		game.MazeMessage = "THE PARTY FLED!"
		return RoundEscaped, nil
	}
	if cs.AllMonstersDead() {
		if cs.HasChest {
			// Pascal CHSTGOLD: BCHEST=true → show chest interaction (traps)
			cs.Phase = engine.CombatChest
			cs.ChestSubPhase = engine.ChestMenu
			cs.Messages = nil
			cs.MessageIndex = 0
		} else {
			// No chest: rewards directly (GETREWRD/GIVEGOLD always run)
			cs.ChestOpened = true
			cs.FinalizeChest(game)
		}
		return RoundVictory, nil
	}
	for _, m := range game.Town.Party.Members {
		if m != nil && m.IsAlive() {
			cs.BeginNextRound(game)
			cs.CurrentActor = c.firstActor()
			return RoundContinues, nil
		}
	}
	cs.Phase = engine.CombatDefeat
	return RoundDefeated, nil
}
