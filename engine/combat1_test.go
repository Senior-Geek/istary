package engine

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// COMBAT-1: known spells kept through loading (#32), MAPORFIC's armour
// class (#40) and LATUMOFIS in combat (#41).

func knownSpellNames(c *Character) []string {
	var out []string
	for i, k := range c.SpellKnown {
		if k {
			out = append(out, SpellTable[i].Name)
		}
	}
	return out
}

// reload saves g and loads it into a new game, returning its roster.
func reload(t *testing.T, g *GameState) []*Character {
	t.Helper()
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	g2 := newTestGame(t, 1)
	if err := g2.Load(); err != nil {
		t.Fatal(err)
	}
	return g2.Town.Roster.Characters
}

// Loading keeps the spells a character has learned, no more: a new mage
// knows HALITO and KATINO after any number of reloads (before COMBAT-1,
// every load taught all four level 1 spells).
func TestLoadKeepsKnownSpells(t *testing.T) {
	freshHome(t)
	g := newTestGame(t, 1)
	seed(1)
	mage := (&CreationState{Name: "MERLIN", Race: Elf, Class: Mage, Alignment: Neutral, Stats: [6]int{7, 15, 10, 6, 9, 6}}).FinalizeCharacter()
	priest := (&CreationState{Name: "ALDRIC", Race: Dwarf, Class: Priest, Alignment: Good, Stats: [6]int{12, 8, 16, 14, 6, 9}}).FinalizeCharacter()
	// A bishop of level 5 who has learned one level 2 mage spell, and a
	// fighter who was a mage and keeps HALITO (one point for it, SETSPELS).
	bishop := NewCharacter("BISHOP", Human, Bishop, Good)
	bishop.Level, bishop.MaxMageSpells, bishop.MaxPriestSpells = 5, [7]int{4, 2}, [7]int{2}
	for _, n := range []string{"HALITO", "KATINO", "MOGREF", "SOPIC", "DIOS"} {
		bishop.SpellKnown[SpellIndex[n]] = true
	}
	fighter := NewCharacter("WASMAGE", Human, Fighter, Good)
	fighter.SpellKnown[SpellIndex["HALITO"]] = true
	SetSpells(fighter)
	for _, c := range []*Character{mage, priest, bishop, fighter} {
		g.Town.Roster.Add(c)
	}
	want := map[string][]string{
		"MERLIN":  {"HALITO", "KATINO"},
		"ALDRIC":  {"DIOS", "BADIOS"},
		"BISHOP":  {"HALITO", "MOGREF", "KATINO", "SOPIC", "DIOS"},
		"WASMAGE": {"HALITO"},
	}
	for round := 1; round <= 3; round++ {
		roster := reload(t, g)
		for _, c := range roster {
			if got := knownSpellNames(c); !slices.Equal(got, want[c.Name]) {
				t.Errorf("reload %d: %s knows %v, want %v", round, c.Name, got, want[c.Name])
			}
		}
		g = newTestGame(t, 1)
		g.Load()
	}
}

// writeRoster writes a save of the given characters straight to disk.
func writeRoster(t *testing.T, chars ...*Character) {
	t.Helper()
	path, err := RosterPath("1")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(SaveState{Roster: chars})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Saves are read as before where the known spells cannot be trusted: none
// saved (the oldest saves, and imports) or ones that do not fit the
// character's points (saved in the spell table's old order) are rebuilt
// from the points. Saves written by earlier Istary, whose known spells the
// old load had already widened, are kept exactly as saved: loading changes
// no player's data.
func TestLoadRebuildsOnlyUntrustedKnownSpells(t *testing.T) {
	freshHome(t)
	none := NewCharacter("NONE", Elf, Mage, Neutral)
	none.MaxMageSpells = [7]int{2}
	odd := NewCharacter("ODD", Elf, Mage, Neutral)
	odd.MaxMageSpells = [7]int{2}
	odd.SpellKnown[SpellIndex["DIOS"]] = true // a priest spell, no priest points
	odd.SpellKnown[SpellIndex["DALTO"]] = true // mage level 4, no points there
	widened := NewCharacter("ELENARA", Elf, Mage, Neutral)
	widened.MaxMageSpells = [7]int{2}
	for _, sp := range MageSpellsByLevel[0] {
		widened.SpellKnown[SpellIndex[sp.Name]] = true
	}
	writeRoster(t, none, odd, widened)
	g := newTestGame(t, 1)
	if err := g.Load(); err != nil {
		t.Fatal(err)
	}
	all := []string{"HALITO", "MOGREF", "KATINO", "DUMAPIC"}
	for _, c := range g.Town.Roster.Characters {
		if got := knownSpellNames(c); !slices.Equal(got, all) {
			t.Errorf("%s knows %v, want %v", c.Name, got, all)
		}
	}
	for _, c := range []*Character{none, odd, widened} {
		if c.Name == "ODD" {
			if c.SpellKnownFits() {
				t.Error("ODD's spells fit")
			}
		} else if c.SpellKnownFits() != (c.Name == "ELENARA") {
			t.Errorf("%s: fits %v", c.Name, c.SpellKnownFits())
		}
	}
}

// MAPORFIC improves the whole party's armour class by 2 for the fight
// (ACMOD2 := 2, "BIG SHIELD"), as MOGREF improves the caster's by 2: the
// monsters' to-hit (20 - AC - level + mod) falls. Before COMBAT-1 it rose.
func TestMaporficImprovesArmourClass(t *testing.T) {
	g, cs := combatFixture(t, []string{"A", "B", "C"}, []Status{OK, OK, Dead})
	priest := g.Town.Party.Members[1]
	priest.Class, priest.Level = Priest, 7
	priest.MaxPriestSpells[3], priest.PriestSpells[3] = 1, 1
	priest.SpellKnown[SpellIndex["MAPORFIC"]] = true
	cs.PartyACMod = [6]int{0, -2, 0}
	cs.executeSpell(priest, &PartyAction{Action: ActionSpell, SpellName: "MAPORFIC"}, g)
	if cs.PartyACMod != [6]int{-2, -4, -2} || priest.PriestSpells[3] != 0 {
		t.Fatalf("after MAPORFIC: %v, points %v", cs.PartyACMod, priest.PriestSpells)
	}
	mogref := &CombatState{}
	g.Town.Party.Members[0].Class = Mage
	g.Town.Party.Members[0].MageSpells[0], g.Town.Party.Members[0].MaxMageSpells[0] = 1, 1
	g.Town.Party.Members[0].SpellKnown[SpellIndex["MOGREF"]] = true
	mogref.executeSpell(g.Town.Party.Members[0], &PartyAction{Action: ActionSpell, SpellName: "MOGREF"}, g)
	if mogref.PartyACMod[0] != -2 {
		t.Fatalf("MOGREF, the reference: %v", mogref.PartyACMod)
	}
}

// LATUMOFIS in combat cures its target's poison, as in Camp; it says so as
// before. Before COMBAT-1 it only said so.
func TestLatumofisCuresPoisonInCombat(t *testing.T) {
	g, cs := combatFixture(t, []string{"A", "B", "C"}, []Status{OK, OK, OK})
	priest, sick, other := g.Town.Party.Members[0], g.Town.Party.Members[2], g.Town.Party.Members[1]
	priest.Class, priest.Level = Priest, 7
	priest.MaxPriestSpells[3], priest.PriestSpells[3] = 1, 1
	priest.SpellKnown[SpellIndex["LATUMOFIS"]] = true
	sick.PoisonAmt, other.PoisonAmt = 3, 2
	cs.executeSpell(priest, &PartyAction{Action: ActionSpell, SpellName: "LATUMOFIS", TargetAlly: 2}, g)
	if sick.PoisonAmt != 0 || other.PoisonAmt != 2 || priest.PriestSpells[3] != 0 || !slices.Contains(cs.Messages, "POISON CURED!") {
		t.Fatalf("poison %d (target), %d (other), points %v, messages %q", sick.PoisonAmt, other.PoisonAmt, priest.PriestSpells, cs.Messages)
	}
}
