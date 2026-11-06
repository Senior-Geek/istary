package engine

import "testing"

// Silent blocks (no message lines) are redacted in events. Pin that no
// state change happens in one, so the redaction hides nothing that a
// message would have shown later. Seeded B1F encounters, fight or parry.
func TestCombatEventsSilentBlocksChangeNothing(t *testing.T) {
	silent, rounds := 0, 0
	for s := int64(1); s <= 80; s++ {
		seed(s)
		g := New(loadScenario(t, 1))
		for i, cls := range []Class{Fighter, Fighter, Fighter, Thief, Priest, Mage} {
			c := NewCharacter(string(rune('A'+i)), Human, cls, Neutral)
			c.HP, c.MaxHP, c.Level, c.AC, c.Strength = 15, 15, 2, 8, 16
			g.Town.Party.Members = append(g.Town.Party.Members, c)
		}
		g.Phase = PhaseCombat
		cs := NewCombat(g)
		g.Combat = cs
		for r := 0; r < 10 && !cs.AllMonstersDead() && !cs.Fled && !cs.allPartyDead(g.Town.Party.Members); r++ {
			for i := range cs.Actions {
				cs.Actions[i] = PartyAction{Action: ActionParry}
				if i < 3 {
					cs.Actions[i] = PartyAction{Action: ActionFight, TargetGroup: 0}
				}
			}
			cs.ExecuteRound(g)
			rounds++
			for _, e := range cs.Events() {
				if e.Silent {
					silent++
				}
			}
			if cs.ev.withheld != 0 {
				t.Fatalf("seed %d round %d: %d state changes in silent blocks", s, cs.Round, cs.ev.withheld)
			}
			cs.CompactGroups()
			cs.Round++
		}
	}
	if silent == 0 {
		t.Fatalf("no silent block was exercised in %d rounds", rounds)
	}
	t.Logf("%d rounds, %d silent blocks, no state change in any", rounds, silent)
}

// A surprise round where the first monster misses: its block holds only the
// surprise line. The event is narration; the monster and its victim are not
// named (seed 2 found by search; fixed during P1 integration).
func TestCombatEventsSurpriseThenSilentMiss(t *testing.T) {
	for s := int64(1); s <= 400; s++ {
		seed(s)
		g := New(loadScenario(t, 1))
		for i, cls := range []Class{Fighter, Fighter, Fighter, Thief, Priest, Mage} {
			c := NewCharacter(string(rune('A'+i)), Human, cls, Neutral)
			c.HP, c.MaxHP, c.Level, c.AC = 15, 15, 2, 2
			g.Town.Party.Members = append(g.Town.Party.Members, c)
		}
		cs := NewCombat(g)
		cs.Surprised, cs.Round = 2, 1
		cs.ExecuteRound(g)
		evs := cs.Events()
		if len(evs) == 0 || len(evs[0].Lines) != 1 || evs[0].Lines[0] != "THE MONSTERS SURPRISED YOU!" {
			continue
		}
		e := evs[0]
		if e.Kind != EventNarration || e.Actor != NoRef || e.Target != NoRef || e.Effects != nil {
			t.Fatalf("seed %d: the surprise line alone exposes an action: %+v", s, e)
		}
		return
	}
	t.Fatal("no surprise round with a silent first attack was found")
}
