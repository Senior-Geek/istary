package engine

// Characterization tests (spike P1a). They pin what the engine does today,
// including known quirks, so that later changes are deliberate. They do not
// state what Istary intends: see docs/design/m2-exploration/
// COMBAT-DESIGN-BASELINE.md §3 (observations H1–H8, corrections C1–C3) and
// WIZARDRY-COMPATIBILITY.md finding #15.
//
// Historical (Player's Guide, 1987 printing): the dead move to the back "at
// the end of the round" (p.48), incapacitated members likewise (p.55).
// Observed here: the engine re-sorts at the start of ExecuteRound, after the
// orders were entered, and the orders stay with their slots.

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

// Timing: a status change between rounds does not move anyone until the
// next ExecuteRound, which re-sorts before it resolves anything (after the
// orders for that round were entered). The display snapshot is taken after
// the sort.
func TestCharResortTiming(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, OK, OK, OK, OK, OK})
	g.Town.Party.Members[1].Status = Dead // killed in the previous round
	if got := names(g.Town.Party.Members); got != "ALDA BRAM CORA DREW EDDA FINN" {
		t.Fatalf("before ExecuteRound the party is not re-sorted: %s", got)
	}
	cs.ExecuteRound(g)
	if got := names(g.Town.Party.Members); got != "ALDA CORA DREW EDDA FINN BRAM" {
		t.Fatalf("ExecuteRound re-sorts at its start: got %s", got)
	}
	if cs.DisplayPartySnap[5].Status != Dead || cs.DisplayPartySnap[1].Status != OK {
		t.Fatalf("display snapshot follows the sorted order: %+v", cs.DisplayPartySnap)
	}
	if cs.Phase != CombatExecute {
		t.Fatalf("round ends in CombatExecute, got %v", cs.Phase)
	}
}

// Orders are stored by slot and are not moved with the members: an order
// given for the member in slot 3 is carried out by whoever is in slot 3
// after the re-sort. (Go probe 2026-11-06 showed the same with Run.)
func TestCharOrdersStayWithSlots(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, Dead, OK, OK, OK, OK})
	cs.Actions[2] = PartyAction{Action: ActionParry} // entered for CORA in slot 3
	cs.ExecuteRound(g)
	if got := lines(cs); got != "DREW PARRIED." {
		t.Fatalf("slot 3's order is carried out by DREW: got %q", got)
	}
}

// Nothing clears cs.Actions between rounds. A slot whose member was not
// offered an order keeps the previous round's order, and a member who moves
// into that slot carries it out.
func TestCharStaleOrderCarriedOut(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, Dead, OK, OK, OK, OK})
	cs.Actions[1] = PartyAction{Action: ActionParry} // BRAM's order from the round he died in
	before := append([]PartyAction(nil), cs.Actions...)
	cs.ExecuteRound(g)
	if got := lines(cs); got != "CORA PARRIED." {
		t.Fatalf("CORA moved into slot 2 and carries out BRAM's old order: got %q", got)
	}
	for i := range cs.Actions {
		a, b := cs.Actions[i], before[i]
		a.Initiative, b.Initiative = 0, 0
		if a != b {
			t.Fatalf("ExecuteRound leaves orders in place; slot %d changed from %+v to %+v", i+1, b, a)
		}
	}
}

// Ally targets are slot indices too. DIALKO aimed at slot 3 (CORA,
// paralyzed) lands on whoever is in slot 3 after the re-sort.
func TestCharAllyTargetStaysWithSlot(t *testing.T) {
	seed(1)
	g, cs := combatFixture(t, []string{"ALDA", "BRAM", "CORA", "DREW", "EDDA", "FINN"}, []Status{OK, Dead, Paralyzed, OK, OK, OK})
	priest := g.Town.Party.Members[0]
	priest.Class = Priest
	priest.PriestSpells[2], priest.MaxPriestSpells[2] = 1, 1
	priest.SpellKnown[SpellIndex["DIALKO"]] = true
	cs.Actions[0] = PartyAction{Action: ActionSpell, SpellName: "DIALKO", TargetAlly: 2}
	cs.ExecuteRound(g)
	if got := names(g.Town.Party.Members); got != "ALDA DREW EDDA FINN CORA BRAM" {
		t.Fatalf("re-sort: %s", got)
	}
	if got := lines(cs); got != "ALDA CASTS DIALKO! | EDDA IS NOT HELPED!" {
		t.Fatalf("DIALKO lands on slot 3's new member: got %q", got)
	}
	if g.Town.Party.Members[4].Status != Paralyzed {
		t.Fatalf("CORA stays paralyzed")
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
