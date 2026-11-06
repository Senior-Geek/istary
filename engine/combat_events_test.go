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
