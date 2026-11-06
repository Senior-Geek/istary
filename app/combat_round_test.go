package app

import (
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// confirmed returns a controller at the confirmation prompt with the given
// orders (slot order; missing ones parry).
func confirmed(t *testing.T, groups int, orders ...engine.PartyAction) (*Controller, []*engine.Character) {
	t.Helper()
	c, p := combatController(t, groups)
	for _, g := range c.Game.Combat.Groups {
		for _, m := range g.Members {
			m.HP, m.MaxHP = 40, 40 // the round should not end the fight
		}
	}
	for i := range c.Game.Combat.Actions {
		c.Game.Combat.Actions[i] = engine.PartyAction{Action: engine.ActionParry}
		if i < len(orders) {
			c.Game.Combat.Actions[i] = orders[i]
		}
	}
	c.Game.Combat.Phase = engine.CombatConfirm
	return c, p
}

// playAll releases every block and returns the steps.
func playAll(t *testing.T, c *Controller, first PlaybackStep) []PlaybackStep {
	t.Helper()
	steps := []PlaybackStep{first}
	for i := 0; i < 200; i++ {
		s, err := c.Advance()
		if err != nil {
			t.Fatal(err)
		}
		if s.Done {
			return steps
		}
		steps = append(steps, s)
	}
	t.Fatal("playback never completed")
	return nil
}

// Commit executes the round once and releases its first block; Advance
// releases the rest one at a time, in order, silent blocks included; only
// released events are visible; then Done.
func TestCombatCommitAndAdvance(t *testing.T) {
	seed(3)
	// Slot 2 uses nothing usable: a silent block (a separator alone).
	c, _ := confirmed(t, 1, engine.PartyAction{Action: engine.ActionParry}, engine.PartyAction{Action: engine.ActionUse, UseItemIdx: -1})
	first, err := c.Commit()
	if err != nil || first.Index != 0 || first.Done {
		t.Fatalf("Commit: %+v %v", first, err)
	}
	cs := c.Game.Combat
	msgs := append([]string(nil), cs.Messages...)
	if _, err := c.Commit(); !errors.Is(err, ErrNotConfirming) {
		t.Fatalf("second Commit: %v", err)
	}
	if !reflect.DeepEqual(cs.Messages, msgs) {
		t.Fatal("the round was executed twice")
	}
	if got := c.RevealedEvents(); len(got) != 1 || !reflect.DeepEqual(got[0], first.Event) {
		t.Fatalf("after Commit only the first block is revealed: %d", len(got))
	}
	all := cs.Events()
	steps := []PlaybackStep{first}
	silent := false
	for i := 0; i < 100; i++ {
		s, err := c.Advance()
		if err != nil {
			t.Fatal(err)
		}
		if s.Done {
			break
		}
		steps = append(steps, s)
		silent = silent || s.Event.Silent
		if rev := c.RevealedEvents(); len(rev) != s.Index+1 || !reflect.DeepEqual(rev[s.Index], s.Event) {
			t.Fatalf("step %d: revealed %d events", s.Index, len(rev))
		}
		if cs.MessageIndex != s.Event.Start {
			t.Fatalf("display cursor %d, event starts at %d", cs.MessageIndex, s.Event.Start)
		}
	}
	if len(steps) != len(all) {
		t.Fatalf("%d steps for %d events", len(steps), len(all))
	}
	for i, s := range steps {
		if s.Index != i || !reflect.DeepEqual(s.Event, all[i]) {
			t.Fatalf("step %d out of order: %+v", i, s)
		}
	}
	if !silent {
		t.Fatal("the silent block was not released as its own step")
	}
	again, err := c.Advance()
	if err != nil || !again.Done || len(c.RevealedEvents()) != len(all) {
		t.Fatalf("Advance after the end: %+v %v", again, err)
	}
}

// Playback draws no random numbers.
func TestCombatPlaybackUsesNoRandomNumbers(t *testing.T) {
	seed(5)
	c, _ := confirmed(t, 1)
	first, err := c.Commit()
	if err != nil {
		t.Fatal(err)
	}
	seed(9)
	want := rand.Int63()
	seed(9)
	playAll(t, c, first)
	c.RevealedEvents()
	if got := rand.Int63(); got != want {
		t.Fatal("playback consumed random numbers")
	}
}

// The round ends only after every block has been shown. Continuing: the
// formation change and cleared orders happen at EndRound, before any new
// order; a member incapacitated in the round keeps its place until then.
func TestCombatEndRoundContinues(t *testing.T) {
	seed(3)
	c, p := confirmed(t, 1)
	first, _ := c.Commit()
	if _, err := c.EndRound(); !errors.Is(err, ErrPlaybackUnfinished) {
		t.Fatalf("EndRound with blocks unshown: %v", err)
	}
	playAll(t, c, first)
	a, b := p[0], p[1]          // p is the party slice itself: keep the characters
	a.Status = engine.Paralyzed // as if paralyzed during the round
	if c.Game.Town.Party.Members[0] != a {
		t.Fatal("the formation changed before the round ended")
	}
	out, err := c.EndRound()
	if err != nil || out != RoundContinues {
		t.Fatalf("EndRound: %v %v", out, err)
	}
	cs := c.Game.Combat
	if cs.Phase != engine.CombatChoose || cs.Round != 2 || cs.DisplayPartySnap != nil {
		t.Fatalf("next round: phase %v round %d", cs.Phase, cs.Round)
	}
	if c.Game.Town.Party.Members[5] != a || cs.CurrentActor != 0 || c.Game.Town.Party.Members[0] != b {
		t.Fatalf("formation change before order entry: first actor %d", cs.CurrentActor)
	}
	for i, a := range cs.Actions {
		if a != (engine.PartyAction{}) {
			t.Fatalf("slot %d keeps an order", i+1)
		}
	}
	if _, err := c.EndRound(); !errors.Is(err, ErrNotPlayingBack) {
		t.Fatalf("second EndRound: %v", err)
	}
}

// Victory, with or without a chest, and defeat.
func TestCombatEndRoundVictoryAndDefeat(t *testing.T) {
	kill := func(c *Controller) {
		for _, g := range c.Game.Combat.Groups {
			for _, m := range g.Members {
				m.Status = 5
			}
		}
	}
	for _, chest := range []bool{false, true} {
		seed(3)
		c, _ := confirmed(t, 1)
		first, _ := c.Commit()
		playAll(t, c, first)
		kill(c)
		c.Game.Combat.HasChest = chest
		out, err := c.EndRound()
		want := engine.CombatVictory
		if chest {
			want = engine.CombatChest
		}
		if err != nil || out != RoundVictory || c.Game.Combat.Phase != want {
			t.Fatalf("chest %v: %v %v phase %v", chest, out, err, c.Game.Combat.Phase)
		}
	}
	seed(3)
	c, p := confirmed(t, 1)
	first, _ := c.Commit()
	playAll(t, c, first)
	for _, m := range p {
		m.Status = engine.Dead
	}
	if out, err := c.EndRound(); err != nil || out != RoundDefeated || c.Game.Combat.Phase != engine.CombatDefeat {
		t.Fatalf("defeat: %v %v", out, err)
	}
}

// Escape (finding #17 corrected): the flight message is the last block
// played back, then EndRound leaves combat; no further round can run. A
// failed escape continues normally.
func TestCombatEscape(t *testing.T) {
	fled, failed := false, false
	for s := int64(1); s < 60 && !(fled && failed); s++ {
		seed(s)
		c, _ := confirmed(t, 1, engine.PartyAction{Action: engine.ActionRun})
		first, err := c.Commit()
		if err != nil {
			t.Fatal(err)
		}
		steps := playAll(t, c, first)
		last := strings.Join(steps[len(steps)-1].Event.Lines, " ")
		out, err := c.EndRound()
		if err != nil {
			t.Fatal(err)
		}
		if out == RoundEscaped {
			fled = true
			if last != "THE PARTY FLED!" || c.Game.Combat != nil || c.Game.Phase != engine.PhaseMaze || c.Game.MazeMessage != "THE PARTY FLED!" {
				t.Fatalf("seed %d: escape: last block %q, phase %v", s, last, c.Game.Phase)
			}
			if _, err := c.Commit(); !errors.Is(err, ErrNotConfirming) {
				t.Fatalf("a round after escaping: %v", err)
			}
		} else {
			failed = true
			if out != RoundContinues || c.Game.Combat.Phase != engine.CombatChoose {
				t.Fatalf("seed %d: failed escape: %v", s, out)
			}
		}
	}
	if !fled || !failed {
		t.Fatalf("seeds did not cover both outcomes (fled %v, failed %v)", fled, failed)
	}
}

// Starting combat: order entry from the first member who can act, or the
// monsters' surprise round straight into playback.
func TestCombatStart(t *testing.T) {
	c, p := combatController(t, 1)
	c.Game.Combat.Phase = engine.CombatInit
	p[0].Status = engine.Dead
	if pb, err := c.StartCombat(); err != nil || pb || c.Game.Combat.Phase != engine.CombatChoose || c.Game.Combat.CurrentActor != 1 {
		t.Fatalf("start: %v %v actor %d", pb, err, c.Game.Combat.CurrentActor)
	}
	if _, err := c.StartCombat(); !errors.Is(err, ErrNotStarting) {
		t.Fatalf("second start: %v", err)
	}
	seed(4)
	c, _ = combatController(t, 1)
	c.Game.Combat.Phase, c.Game.Combat.Surprised = engine.CombatInit, 2
	if pb, err := c.StartCombat(); err != nil || !pb || c.Game.Combat.Phase != engine.CombatExecute || len(c.RevealedEvents()) != 1 {
		t.Fatalf("surprise: %v %v", pb, err)
	}
}

// B)ack at the confirmation prompt clears every order.
func TestCombatReturnToOrders(t *testing.T) {
	c, _ := confirmed(t, 1)
	if err := c.ReturnToOrders(); err != nil {
		t.Fatal(err)
	}
	cs := c.Game.Combat
	if cs.Phase != engine.CombatChoose || cs.CurrentActor != 0 || cs.Actions[0] != (engine.PartyAction{}) {
		t.Fatalf("B)ack: phase %v", cs.Phase)
	}
	if err := c.ReturnToOrders(); !errors.Is(err, ErrNotConfirming) {
		t.Fatalf("B)ack outside confirmation: %v", err)
	}
}

// Lifecycle operations in the wrong phase are refused, and change nothing.
func TestCombatLifecycleInvalidPhases(t *testing.T) {
	c, _ := combatController(t, 1) // order entry
	if _, err := c.Commit(); !errors.Is(err, ErrNotConfirming) {
		t.Errorf("Commit during order entry: %v", err)
	}
	if _, err := c.Advance(); !errors.Is(err, ErrNotPlayingBack) {
		t.Errorf("Advance during order entry: %v", err)
	}
	if _, err := c.EndRound(); !errors.Is(err, ErrNotPlayingBack) {
		t.Errorf("EndRound during order entry: %v", err)
	}
	if err := c.ChooseBoon(0); !errors.Is(err, ErrNotPlayingBack) {
		t.Errorf("ChooseBoon during order entry: %v", err)
	}
	if c.RevealedEvents() != nil {
		t.Error("events revealed outside playback")
	}
	c.Game.Combat = nil
	if _, err := c.Commit(); !errors.Is(err, ErrNotConfirming) {
		t.Errorf("Commit with no combat: %v", err)
	}
}

// HAMAN's boon prompt pauses playback until a boon is chosen.
func TestCombatPlaybackWaitsForBoon(t *testing.T) {
	seed(3)
	c, _ := confirmed(t, 1)
	c.Commit()
	cs := c.Game.Combat
	cs.HamanSelecting = true
	idx := cs.MessageIndex
	if _, err := c.Advance(); !errors.Is(err, ErrAwaitingChoice) || cs.MessageIndex != idx {
		t.Fatalf("Advance while waiting: %v", err)
	}
	if _, err := c.EndRound(); !errors.Is(err, ErrAwaitingChoice) {
		t.Fatalf("EndRound while waiting: %v", err)
	}
	if err := c.ChooseBoon(3); !errors.Is(err, ErrInvalidChoice) {
		t.Fatalf("boon 4: %v", err)
	}
	cs.HamanSelecting = false
	if err := c.ChooseBoon(0); !errors.Is(err, ErrWrongState) {
		t.Fatalf("boon with no prompt: %v", err)
	}
}
