package main

// Characterization tests (spike P1a) for the terminal's side of combat:
// who is offered an order, how a round's messages are stepped through, and
// what is left over for the next round. They pin today's behaviour, quirks
// included; see engine/combat_characterization_test.go and
// docs/design/m2-exploration/COMBAT-DESIGN-BASELINE.md §3.

import (
	"testing"
	"time"

	"github.com/Senior-Geek/istary/engine"
)

// findNextActor offers an order to anyone engine.IsAlive accepts: OK,
// ASLEEP and AFRAID. PARALYZE, STONED and the dead are skipped. (Whether
// Classic offered orders to asleep or afraid members is not verified.)
func TestCharWhoIsOfferedAnOrder(t *testing.T) {
	g := newGame(t, 1)
	sts := []engine.Status{engine.Asleep, engine.Paralyzed, engine.Afraid, engine.Dead, engine.Stoned, engine.OK}
	for i, st := range sts {
		c := character(string(rune('A'+i)), engine.Human, engine.Fighter, engine.Neutral)
		c.Status = st
		g.Town.Party.Members = append(g.Town.Party.Members, c)
	}
	var offered []int
	for i := findNextActor(g, -1); i < len(sts); i = findNextActor(g, i) {
		offered = append(offered, i+1)
	}
	if len(offered) != 3 || offered[0] != 1 || offered[1] != 3 || offered[2] != 6 {
		t.Fatalf("offered slots %v, want [1 3 6] (asleep, afraid, OK)", offered)
	}
}

// Round completion in the terminal: each timer tick shows the next message
// block; after the last block the display snapshot is cleared and the
// terminal calls engine BeginNextRound, which compacts groups, advances the
// round, re-sorts the party and clears last round's orders (P1c, C2 and C3;
// before P1c the old orders stayed in place).
func TestCharRoundCompletion(t *testing.T) {
	seed(21)
	g, d := partyInMaze(t)
	g.Combat = engine.NewCombat(g)
	g.Phase = engine.PhaseCombat
	c := g.Combat
	c.Surprised = 0
	combatTick(g)
	if c.Phase != engine.CombatChoose {
		t.Fatalf("phase %v, want Choose", c.Phase)
	}
	for c.Phase == engine.CombatChoose {
		actor := c.CurrentActor
		d.keys("F")
		if c.SelectingGroup {
			d.keys("1")
		}
		if c.Phase == engine.CombatChoose && c.CurrentActor == actor {
			d.keys("P")
		}
	}
	given := append([]engine.PartyAction(nil), c.Actions...)
	d.enter()
	if c.Phase != engine.CombatExecute || c.DisplayPartySnap == nil {
		t.Fatalf("after Return: phase %v, snapshot %v", c.Phase, c.DisplayPartySnap != nil)
	}
	blocks := 0
	for _, m := range c.Messages {
		if m == engine.ActionSeparator {
			blocks++
		}
	}
	round := c.Round
	ticks := 0
	for g.Combat != nil && c.Phase == engine.CombatExecute && ticks < 100 {
		combatTimerTick(g, time.Hour)
		ticks++
	}
	if g.Combat == nil || c.Phase != engine.CombatChoose {
		t.Skipf("seed 21 ended the combat in round 1 (phase %v); the round-trip is not exercised", c.Phase)
	}
	// One tick per block after the first, plus one that ends the round.
	if ticks != blocks {
		t.Errorf("%d ticks for %d blocks", ticks, blocks)
	}
	if c.Round != round+1 || c.DisplayPartySnap != nil || c.DisplayAliveCounts != nil || len(c.Messages) != 0 {
		t.Errorf("round end: round %d→%d, snapshot cleared %v, messages %d", round, c.Round, c.DisplayPartySnap == nil, len(c.Messages))
	}
	for i, a := range c.Actions {
		if a != (engine.PartyAction{}) {
			t.Errorf("slot %d keeps last round's order %+v (given %+v)", i+1, a, given[i])
		}
	}
}
