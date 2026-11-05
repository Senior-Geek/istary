package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// sampleCharacter builds a fully specified character without randomness.
func sampleCharacter() *Character {
	c := NewCharacter("ALDRIC", Dwarf, Priest, Good)
	c.Password = "SECRET"
	c.Level = 4
	c.XP = 4321
	c.HP, c.MaxHP = 18, 22
	c.Gold = 777
	c.Age = 1000
	c.Strength, c.IQ, c.Piety, c.Vitality, c.Agility, c.Luck = 12, 8, 16, 14, 6, 9
	c.AC = 4
	c.PriestSpells = [7]int{3, 1, 0, 0, 0, 0, 0}
	c.MaxPriestSpells = [7]int{4, 2, 0, 0, 0, 0, 0}
	c.SpellKnown[SpellIndex["DIOS"]] = true
	c.SpellKnown[SpellIndex["BADIOS"]] = true
	c.MaxLevAC = 4
	c.PoisonAmt = 1
	c.AddItem(3, true)
	c.AddItem(7, false)
	c.EquipItem(0)
	return c
}

func newTestGame(t *testing.T, num int) *GameState {
	t.Helper()
	return New(loadScenario(t, num))
}

func TestRosterPathKeepsUpstreamDirectory(t *testing.T) {
	home := freshHome(t)
	for _, key := range []string{"1", "2", "3"} {
		got, err := RosterPath(key)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(home, ".config", "wizardry", "roster"+key+".json")
		if got != want {
			t.Errorf("RosterPath(%q) = %s, want %s", key, got, want)
		}
	}
	if fi, err := os.Stat(filepath.Join(home, ".config", "wizardry")); err != nil || !fi.IsDir() {
		t.Errorf("RosterPath should create the save directory: %v", err)
	}
}

// The on-disk JSON format is the save-compatibility contract. Any change to
// field names, tags, or omitempty rules shows up as a golden diff.
func TestSaveFileFormat(t *testing.T) {
	home := freshHome(t)
	g := newTestGame(t, 1)
	c := sampleCharacter()
	other := NewCharacter("BENCHWARMER", Human, Fighter, Neutral)
	g.Town.Roster.Add(c)
	g.Town.Roster.Add(other)
	g.Town.Party.Members = []*Character{c}

	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".config", "wizardry", "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "save_roster1.golden.json", raw)
}

func TestSaveLoadRoundTripAllScenarios(t *testing.T) {
	for num := 1; num <= 3; num++ {
		freshHome(t)
		g := newTestGame(t, num)
		c := sampleCharacter()
		g.Town.Roster.Add(c)
		g.Town.Party.Members = []*Character{c}
		if err := g.Save(); err != nil {
			t.Fatal(err)
		}

		g2 := newTestGame(t, num)
		if err := g2.Load(); err != nil {
			t.Fatalf("wiz%d load: %v", num, err)
		}
		if len(g2.Town.Roster.Characters) != 1 {
			t.Fatalf("wiz%d: roster has %d characters, want 1", num, len(g2.Town.Roster.Characters))
		}
		got := g2.Town.Roster.Characters[0]
		if got.Name != c.Name || got.Password != c.Password || got.XP != c.XP ||
			got.HP != c.HP || got.Gold != c.Gold || got.PoisonAmt != c.PoisonAmt ||
			got.Items != c.Items || got.ItemCount != c.ItemCount ||
			got.PriestSpells != c.PriestSpells || got.MaxPriestSpells != c.MaxPriestSpells {
			t.Errorf("wiz%d: character changed in round trip:\n got %+v\nwant %+v", num, got, c)
		}
		if g2.Town.Party.Size() != 1 || g2.Town.Party.Members[0] != got {
			t.Errorf("wiz%d: party not rebuilt from roster pointer", num)
		}
	}
}

func TestLoadWithoutSaveIsFreshGame(t *testing.T) {
	freshHome(t)
	g := newTestGame(t, 1)
	if err := g.Load(); err != nil {
		t.Fatalf("Load with no save file: %v", err)
	}
	if len(g.Town.Roster.Characters) != 0 || g.Town.Party.Size() != 0 {
		t.Error("fresh game should have empty roster and party")
	}
}

func TestLoadCorruptSaveReturnsError(t *testing.T) {
	home := freshHome(t)
	dir := filepath.Join(home, ".config", "wizardry")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "roster1.json"), []byte("{not json"), 0644)
	if err := newTestGame(t, 1).Load(); err == nil {
		t.Error("Load of corrupt JSON should return an error")
	}
}

func installLegacySave(t *testing.T, home, key string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "legacy_roster.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".config", "wizardry", "wiz"+key)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "roster.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
}

// Older builds saved to ~/.config/wizardry/wiz<N>/roster.json with
// Equipment/Inventory arrays instead of the flat Items model.
func TestLoadLegacySave(t *testing.T) {
	home := freshHome(t)
	installLegacySave(t, home, "1")
	g := newTestGame(t, 1)
	if err := g.Load(); err != nil {
		t.Fatal(err)
	}

	roster := g.Town.Roster.Characters
	if len(roster) != 2 {
		t.Fatalf("roster has %d characters, want 2", len(roster))
	}
	fighter, mage := roster[0], roster[1]

	// Equipment entries become equipped, identified items; Inventory follows.
	want := [8]Possession{
		{ItemIndex: 5, Equipped: true, Identified: true},
		{ItemIndex: 12, Equipped: true, Identified: true},
		{ItemIndex: 1, Equipped: false, Identified: true},
	}
	if fighter.Items != want || fighter.ItemCount != 3 {
		t.Errorf("fighter items = %+v (count %d), want %+v (count 3)", fighter.Items, fighter.ItemCount, want)
	}
	if fighter.Inventory != nil || fighter.Equipment != [8]int{-1, -1, -1, -1, -1, -1, -1, -1} {
		t.Error("legacy Equipment/Inventory fields should be cleared after migration")
	}

	// Missing MaxMageSpells are backfilled from current slots, and spell
	// knowledge is regenerated from slots: a level-1 mage slot grants every
	// level-1 mage spell.
	if mage.MaxMageSpells != [7]int{3, 0, 0, 0, 0, 0, 0} {
		t.Errorf("mage MaxMageSpells = %v, want backfilled [3 0 ...]", mage.MaxMageSpells)
	}
	for _, sp := range MageSpellsByLevel[0] {
		if !mage.SpellKnown[SpellIndex[sp.Name]] {
			t.Errorf("legacy mage should know level-1 spell %s", sp.Name)
		}
	}
	if mage.Status != Dead {
		t.Errorf("mage status = %v, want DEAD preserved", mage.Status)
	}

	// Party rebuilt from names; unknown names are dropped.
	if g.Town.Party.Size() != 2 || g.Town.Party.Members[0] != fighter || g.Town.Party.Members[1] != mage {
		t.Errorf("party = %v, want [OLDFIGHTER OLDMAGE]", g.Town.Party.Members)
	}

	// The next save goes to the flat path; the legacy file is left in place.
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(home, ".config", "wizardry", "roster1.json"),
		filepath.Join(home, ".config", "wizardry", "wiz1", "roster.json"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
}

func TestCurrentSaveTakesPrecedenceOverLegacy(t *testing.T) {
	home := freshHome(t)
	installLegacySave(t, home, "2")
	g := newTestGame(t, 2)
	c := sampleCharacter()
	g.Town.Roster.Add(c)
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	g2 := newTestGame(t, 2)
	if err := g2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(g2.Town.Roster.Characters) != 1 || g2.Town.Roster.Characters[0].Name != "ALDRIC" {
		t.Errorf("Load should prefer roster2.json over wiz2/roster.json")
	}
}

// Characterizes current behavior: Load discards saved SpellKnown and
// regenerates it from MaxMageSpells/MaxPriestSpells, so a freshly created
// mage (HALITO + KATINO) knows all level-1 mage spells after a reload.
func TestLoadRegeneratesSpellKnowledge(t *testing.T) {
	freshHome(t)
	g := newTestGame(t, 1)
	cs := &CreationState{Name: "MERLIN", Race: Elf, Class: Mage, Alignment: Neutral,
		Stats: [6]int{7, 15, 10, 6, 9, 6}}
	seed(1)
	c := cs.FinalizeCharacter()
	known := func(c *Character) (n int) {
		for _, k := range c.SpellKnown {
			if k {
				n++
			}
		}
		return n
	}
	if known(c) != 2 {
		t.Fatalf("new mage knows %d spells, want 2 (HALITO, KATINO)", known(c))
	}
	g.Town.Roster.Add(c)
	g.Save()

	g2 := newTestGame(t, 1)
	g2.Load()
	if got, want := known(g2.Town.Roster.Characters[0]), len(MageSpellsByLevel[0]); got != want {
		t.Errorf("reloaded mage knows %d spells, want %d (all level-1 mage spells)", got, want)
	}
}
