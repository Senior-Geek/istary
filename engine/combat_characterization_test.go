package engine

// Characterization tests (spike P1a), updated by P1c. They pin what the
// engine does, so that changes are deliberate. See docs/design/m2-exploration/
// COMBAT-DESIGN-BASELINE.md §3 (observations H1–H8, corrections C1–C3) and
// WIZARDRY-COMPATIBILITY.md finding #15.
//
// Historical (Player's Guide, 1987 printing): the dead move to the back "at
// the end of the round" (p.48), incapacitated members likewise (p.55).
// Before P1c the engine re-sorted at the start of ExecuteRound, after the
// orders were entered, and orders and ally targets stayed with their slots
// (P1a pinned that at commit a27adb4). P1c corrections: the re-sort happens
// in BeginNextRound (C2), orders and ally targets follow their characters
// (C1), and the transition clears last round's orders (C3). The tests below
// marked P1c replace the P1a tests that pinned the old behaviour.

import (
	"strings"
	"testing"
)

// combatFixture builds a party with the given statuses and one sleeping
// monster that wakes and does nothing this round, so the messages contain
// only party actions. Orders are left empty (ActionNone).
func combatFixture(t *testing.T, names []string, st []Status) (*GameState, *CombatState) {
	t.Helper()
	var ms []*Character
	for i, n := range names {
		c := NewCharacter(n, Human, Fighter, Neutral)
		c.HP, c.MaxHP, c.Level, c.AC = 10, 10, 1, 10
		c.Status = st[i]
		if st[i] >= Dead {
			c.HP = 0
		}
		ms = append(ms, c)
	}
	g := &GameState{Scenario: loadScenario(t, 1), Town: &TownState{Party: &Party{Members: ms}}}
	mon := &CombatMonster{MonsterID: 0, HP: 50, MaxHP: 50, Status: 1}
	cs := &CombatState{Phase: CombatConfirm, Round: 2, Actions: make([]PartyAction, len(ms)),
		Groups: []*MonsterGroup{{MonsterID: 0, Members: []*CombatMonster{mon}, AliveCnt: 1}}}
	g.Combat = cs
	return g, cs
}

func names(ms []*Character) string {
	var out []string
	for _, m := range ms {
		out = append(out, m.Name)
	}
	return strings.Join(out, " ")
}

func lines(cs *CombatState) string {
	var out []string
	for _, m := range cs.Messages {
		if m != ActionSeparator {
			out = append(out, m)
		}
	}
	return strings.Join(out, " | ")
}

// The sort moves every worse status back, not only death, and is an
// exchange sort: it is not stable.
func TestCharResortStatusOrder(t *testing.T) {
	cases := []struct {
		st   []Status
		want string
	}{
		{[]Status{OK, Paralyzed, OK, OK, OK, OK}, "A C D E F B"},
		{[]Status{OK, Dead, Paralyzed, OK, OK, OK}, "A D E F C B"},
		{[]Status{Asleep, OK, OK, OK, OK, OK}, "B C D E F A"},
		{[]Status{Dead, OK, OK, Ashed, OK, Dead}, "B C E A F D"},
		{[]Status{Afraid, OK, Stoned, OK, Lost, OK}, "B D F A C E"},
		// Not stable: B and C are both paralyzed, but C ends up ahead of B.
		{[]Status{Dead, Paralyzed, Paralyzed, OK, OK, OK}, "D E F C B A"},
		// A cured member never moves forward past an OK member.
		{[]Status{OK, OK, OK, OK, OK, OK}, "A B C D E F"},
	}
	for _, c := range cases {
		var ms []*Character
		for i, s := range c.st {
			ms = append(ms, &Character{Name: string(rune('A' + i)), Status: s})
		}
		sortPartyByStatus(ms)
		if got := names(ms); got != c.want {
			t.Errorf("%v: got %s, want %s", c.st, got, c.want)
		}
	}
}

// P1c (C2). Timing: ExecuteRound no longer re-sorts. A status change in a
// round moves no one while that round's results are shown; BeginNextRound
// re-sorts before the next orders are entered. Death and paralysis both move
// a member back.
func TestCharResortTiming(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, OK})
	cs.ExecuteRound(g)
	g.Town.Party.Members[0].Status = Paralyzed // as if paralyzed in this round
	g.Town.Party.Members[1].Status = Dead      // and slain
	if got := names(g.Town.Party.Members); got != "ALDA BRAM CORA DREW EDDA FINN" {
		t.Fatalf("the round's formation is kept until the transition: %s", got)
	}
	cs.BeginNextRound(g)
	if got := names(g.Town.Party.Members); got != "CORA DREW EDDA FINN ALDA BRAM" {
		t.Fatalf("BeginNextRound re-sorts: got %s", got)
	}
	if cs.Round != 3 || cs.Phase != CombatChoose || len(cs.Messages) != 0 {
		t.Fatalf("transition: round %d, phase %v, messages %d", cs.Round, cs.Phase, len(cs.Messages))
	}
	cs.ExecuteRound(g)
	if got := names(g.Town.Party.Members); got != "CORA DREW EDDA FINN ALDA BRAM" {
		t.Fatalf("ExecuteRound does not re-sort: got %s", got)
	}
	if cs.DisplayPartySnap[4].Status != Paralyzed || cs.DisplayPartySnap[5].Status != Dead {
		t.Fatalf("display snapshot follows the formation the orders were given for: %+v", cs.DisplayPartySnap)
	}
}

// P1c (C2). A round's results stay valid until the explicit transition:
// messages, events, the display snapshot and the formation they refer to.
func TestCharRoundResultsKeptUntilTransition(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, OK})
	cs.Actions[2] = PartyAction{Action: ActionParry}
	cs.ExecuteRound(g)
	g.Town.Party.Members[1].Status = Dead
	msgs, evs := strings.Join(cs.Messages, "|"), cs.Events()
	if len(evs) != 1 || evs[0].Actor.Member != 2 || evs[0].Actor.Name != "CORA" || g.Town.Party.Members[2].Name != "CORA" {
		t.Fatalf("event refers to slot 3, CORA, in the shown formation: %+v", evs)
	}
	if strings.Join(cs.Messages, "|") != msgs || cs.DisplayPartySnap == nil {
		t.Fatalf("results changed before the transition")
	}
	cs.BeginNextRound(g)
	if len(cs.Messages) != 0 || len(cs.Events()) != 0 {
		t.Fatalf("the transition clears the shown round")
	}
}

// P1c (C1). Orders are carried out by the character they were given for.
// After a death the transition re-sorts first, so orders are given against
// the new lines: CORA, now in slot 2, parries.
func TestCharOrdersStayWithCharacters(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, Dead, OK, OK, OK, OK})
	cs.BeginNextRound(g)
	if got := names(g.Town.Party.Members); got != "ALDA CORA DREW EDDA FINN BRAM" {
		t.Fatalf("re-sort: %s", got)
	}
	cs.Actions[1] = PartyAction{Action: ActionParry} // given for CORA in slot 2
	cs.ExecuteRound(g)
	if got := lines(cs); got != "CORA PARRIED." {
		t.Fatalf("CORA's order is carried out by CORA: got %q", got)
	}
}

// P1c (C1). Even if positions change between order entry and execution, an
// order stays with its character: the binding is to the character, not to
// the slot.
func TestCharOrdersFollowCharacterIfPositionsChange(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, OK})
	cs.BeginNextRound(g)                             // order entry begins with this formation
	cs.Actions[2] = PartyAction{Action: ActionParry} // CORA, slot 3
	m := g.Town.Party.Members
	m[2], m[3] = m[3], m[2] // positions change before execution
	cs.ExecuteRound(g)
	if got := lines(cs); got != "CORA PARRIED." {
		t.Fatalf("got %q", got)
	}
}

// P1c (C3). Last round's orders are cleared at the transition and never
// carried out later, by their member or by anyone who moves into the slot.
func TestCharStaleOrdersNeverExecute(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, OK})
	for i := range cs.Actions {
		cs.Actions[i] = PartyAction{Action: ActionParry}
	}
	cs.ExecuteRound(g)
	g.Town.Party.Members[1].Status = Dead // BRAM dies in round 2
	cs.BeginNextRound(g)
	for i, a := range cs.Actions {
		if a != (PartyAction{}) {
			t.Fatalf("slot %d keeps an order after the transition: %+v", i+1, a)
		}
	}
	cs.Actions[0] = PartyAction{Action: ActionParry} // only ALDA is given an order
	cs.ExecuteRound(g)
	if got := lines(cs); got != "ALDA PARRIED." {
		t.Fatalf("only the new order is carried out: got %q", got)
	}
}

// P1c (C1). Ally targets follow the character. After the transition moves
// CORA (paralyzed) to slot 5, DIALKO aimed at her cures her; DIOS aimed at
// DREW heals DREW, even when positions change before execution.
func TestCharAllyTargetsFollowCharacters(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, Dead, Paralyzed, OK, OK, OK})
	priest := g.Town.Party.Members[0]
	priest.Class = Priest
	priest.PriestSpells[0], priest.MaxPriestSpells[0] = 1, 1
	priest.PriestSpells[2], priest.MaxPriestSpells[2] = 1, 1
	priest.SpellKnown[SpellIndex["DIALKO"]] = true
	priest.SpellKnown[SpellIndex["DIOS"]] = true
	cs.BeginNextRound(g)
	if got := names(g.Town.Party.Members); got != "ALDA DREW EDDA FINN CORA BRAM" {
		t.Fatalf("re-sort: %s", got)
	}
	cs.Actions[0] = PartyAction{Action: ActionSpell, SpellName: "DIALKO", TargetAlly: 4} // CORA, slot 5
	cs.ExecuteRound(g)
	if got := lines(cs); got != "ALDA CASTS DIALKO! | CORA IS CURED!" {
		t.Fatalf("DIALKO: got %q", got)
	}
	evs := cs.Events()
	if len(evs) != 1 || evs[0].Kind != EventPartySpell || evs[0].Target.Member != 4 || evs[0].Target.Name != "CORA" {
		t.Fatalf("event: %+v", evs)
	}

	// DIOS at DREW (slot 2); then DREW and EDDA swap before execution.
	dreW := g.Town.Party.Members[1]
	dreW.HP = 3
	cs.BeginNextRound(g)
	cs.Actions[0] = PartyAction{Action: ActionSpell, SpellName: "DIOS", TargetAlly: 1}
	m := g.Town.Party.Members
	m[1], m[2] = m[2], m[1]
	cs.ExecuteRound(g)
	if !strings.Contains(lines(cs), "ALDA CASTS DIOS! | DREW IS") || dreW.HP <= 3 {
		t.Fatalf("DIOS heals DREW: %q, DREW HP %d", lines(cs), dreW.HP)
	}
}

// P1c. A cured character keeps their current position: the transition never
// moves an OK member forward, so CORA, cured in slot 5, stays there.
func TestCharCuredCharacterKeepsPosition(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, Paralyzed, OK, OK, OK})
	cs.BeginNextRound(g)
	if got := names(g.Town.Party.Members); got != "ALDA BRAM DREW EDDA FINN CORA" {
		t.Fatalf("paralyzed CORA moves back: %s", got)
	}
	g.Town.Party.Members[5].Status = OK // cured during the round
	cs.BeginNextRound(g)
	if got := names(g.Town.Party.Members); got != "ALDA BRAM DREW EDDA FINN CORA" {
		t.Fatalf("cured CORA stays where she is: %s", got)
	}
}

// Orders for members who cannot act. Paralyzed and dead members are skipped
// silently even if an order is present. Asleep and afraid members count as
// alive (IsAlive): an asleep member's Parry is carried out, an asleep
// caster's spell is disrupted.
func TestCharOrdersForIncapacitated(t *testing.T) {
	cases := []struct {
		st   Status
		a    PartyAction
		want string
	}{
		{Paralyzed, PartyAction{Action: ActionParry}, ""},
		{Dead, PartyAction{Action: ActionParry}, ""},
		{Asleep, PartyAction{Action: ActionParry}, "FINN PARRIED."},
		{Afraid, PartyAction{Action: ActionParry}, "FINN PARRIED."},
	}
	for _, c := range cases {
		seed(1)
		g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, c.st})
		cs.Actions[5] = c.a
		cs.ExecuteRound(g)
		if got := lines(cs); got != c.want {
			t.Errorf("%v member with %v: got %q, want %q", c.st, c.a.Action, got, c.want)
		}
	}
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, Asleep})
	mage := g.Town.Party.Members[5]
	mage.Class = Mage
	mage.MageSpells[0], mage.MaxMageSpells[0] = 1, 1
	mage.SpellKnown[SpellIndex["HALITO"]] = true
	cs.Actions[5] = PartyAction{Action: ActionSpell, SpellName: "HALITO", TargetGroup: 0}
	cs.ExecuteRound(g)
	if got := lines(cs); got != "FINN: SPELL DISRUPTED" {
		t.Errorf("asleep caster: got %q", got)
	}
}

// Message blocks: each action ends with ActionSeparator. A Use with nothing
// usable writes an empty block (a separator alone). The surprise line has no
// separator of its own and shares the first monster's block.
func TestCharMessageBlocks(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, OK})
	cs.Actions[3] = PartyAction{Action: ActionUse, UseItemIdx: -1}
	cs.Actions[4] = PartyAction{Action: ActionParry}
	cs.ExecuteRound(g)
	seps := 0
	for _, m := range cs.Messages {
		if m == ActionSeparator {
			seps++
		}
	}
	if seps != 2 || len(cs.Messages) != 3 || cs.Messages[len(cs.Messages)-1] != ActionSeparator {
		t.Fatalf("one empty block and one parry block: %q", cs.Messages)
	}

	seed(4)
	g, cs = combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, OK})
	cs.Groups[0].Members[0].Status = 0
	cs.Surprised, cs.Round = 2, 1
	cs.ExecuteRound(g)
	if len(cs.Messages) < 2 || cs.Messages[0] != "THE MONSTERS SURPRISED YOU!" || cs.Messages[1] == ActionSeparator {
		t.Fatalf("the surprise line opens the first monster block: %q", cs.Messages)
	}
}

// Correction of compatibility finding #17 (M5b). A successful Run ends the
// round normally: the round is ready for playback (CombatExecute) and its
// last block is the flight message. Before the correction the phase stayed
// at CombatConfirm, so Return played a further round.
func TestRunEndsTheRoundForPlayback(t *testing.T) {
	fled, failed := false, false
	for s := int64(1); s < 80 && !(fled && failed); s++ {
		seed(s)
		g, cs := combatFixture(t, []string{"A", "B", "C", "D", "E", "F"}, []Status{OK, OK, OK, OK, OK, OK})
		cs.Groups[0].Members[0].Status = 0
		cs.Actions[0] = PartyAction{Action: ActionRun}
		cs.ExecuteRound(g)
		if cs.Phase != CombatExecute {
			t.Fatalf("seed %d: phase %v after the round (fled %v)", s, cs.Phase, cs.Fled)
		}
		text := strings.Join(cs.Messages, "|")
		if cs.Fled {
			fled = true
			if !strings.HasSuffix(text, "THE PARTY FLED!|"+ActionSeparator) {
				t.Fatalf("seed %d: the flight message is the last block: %q", s, cs.Messages)
			}
		} else {
			failed = true
			if !strings.Contains(text, "A FAILED TO RUN!") {
				t.Fatalf("seed %d: failed run: %q", s, cs.Messages)
			}
		}
	}
	if !fled || !failed {
		t.Fatalf("seeds did not cover both outcomes (fled %v, failed %v)", fled, failed)
	}
}
