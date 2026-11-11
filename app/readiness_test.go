package app

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// The equipment check (UX-1): the game's own rules on what is equipped and
// who may use it, a caution that changes nothing.

// kitted is a character of class carrying the named items, the equipped
// ones marked so (as the equip walk leaves them: worn, identified).
func kitted(t *testing.T, c *Controller, name string, class engine.Class, equipped []string, carried ...string) *engine.Character {
	t.Helper()
	ch := engine.NewCharacter(name, engine.Human, class, engine.Neutral)
	ch.HP, ch.MaxHP = 10, 10
	for _, n := range equipped {
		ch.AddItem(itemNamed(t, c, n), true)
		ch.Items[ch.ItemCount-1].Equipped = true
	}
	for _, n := range carried {
		ch.AddItem(itemNamed(t, c, n), true)
	}
	ch.RecalcAC(c.Game.Scenario.Items)
	return ch
}

// partyOf puts chars in the town's party, in order.
func partyOf(c *Controller, chars ...*engine.Character) {
	for _, ch := range chars {
		c.Game.Town.Roster.Add(ch)
		c.Game.Town.Party.Members = append(c.Game.Town.Party.Members, ch)
	}
}

func problemsOf(c *Controller) map[string][]string {
	out := map[string][]string{}
	for _, n := range c.EquipmentCheck() {
		out[n.Name] = n.Problems
	}
	return out
}

const (
	noWeapon = "has no weapon equipped"
	noArmor  = "has no armor equipped"
)

func TestEquipmentCheck(t *testing.T) {
	cases := []struct {
		what     string
		class    engine.Class
		equipped []string
		carried  []string
		want     []string
	}{
		{"a Fighter with nothing", engine.Fighter, nil, nil, []string{noWeapon, noArmor}},
		{"a Fighter carrying a sword and armor, not wearing them", engine.Fighter, nil, []string{"LONG SWORD", "LEATHER ARMOR"}, []string{noWeapon, noArmor}},
		{"a Fighter with a sword only", engine.Fighter, []string{"LONG SWORD"}, nil, []string{noArmor}},
		{"a Fighter in armor only", engine.Fighter, []string{"CHAIN MAIL"}, nil, []string{noWeapon}},
		{"a Fighter, sword and armor", engine.Fighter, []string{"LONG SWORD", "CHAIN MAIL"}, nil, nil},
		{"a Fighter with a shield for protection", engine.Fighter, []string{"STAFF", "SMALL SHIELD"}, nil, nil},
		{"a Fighter with a helm for protection", engine.Fighter, []string{"LONG SWORD", "HELM"}, nil, nil},
		{"a Priest, mace and armor", engine.Priest, []string{"ANOINTED MACE", "CHAIN MAIL"}, nil, nil},
		{"a Priest with a staff, not a sword", engine.Priest, []string{"STAFF", "LEATHER ARMOR"}, nil, nil},
		{"a Thief, short sword and leather", engine.Thief, []string{"SHORT SWORD", "LEATHER ARMOR"}, nil, nil},
		{"a Thief with the Thieves Dagger", engine.Thief, []string{"THIEVES DAGGER", "SMALL SHIELD"}, nil, nil},
		{"a Bishop, mace and leather", engine.Bishop, []string{"ANOINTED MACE", "LEATHER ARMOR"}, nil, nil},
		{"a Samurai in armor with a sword", engine.Samurai, []string{"LONG SWORD", "PLATE MAIL"}, nil, nil},
		{"a Lord in armor with a sword", engine.Lord, []string{"LONG SWORD", "BREAST PLATE"}, nil, nil},
		// A Mage may wear nothing heavier than robes: never told to wear armor.
		{"a Mage with a dagger, unarmored", engine.Mage, []string{"DAGGER"}, nil, nil},
		{"a Mage with a staff and robes", engine.Mage, []string{"STAFF", "ROBES"}, nil, nil},
		{"a Mage with nothing", engine.Mage, nil, nil, []string{noWeapon}},
		{"a Mage carrying a dagger, not wielding it", engine.Mage, nil, []string{"DAGGER"}, []string{noWeapon}},
		// A Ninja with nothing equipped fights unarmed on purpose.
		{"a Ninja with nothing equipped", engine.Ninja, nil, []string{"SHURIKEN"}, nil},
		{"a Ninja in armor, unarmed", engine.Ninja, []string{"LEATHER ARMOR"}, nil, []string{noWeapon}},
		{"a Ninja with a weapon, unarmored", engine.Ninja, []string{"SHURIKEN"}, nil, []string{noArmor}},
		// Equipped, but not by the rules (an older save, or a change of
		// class): it is not usable, and the check says so.
		{"a Priest wielding a sword", engine.Priest, []string{"LONG SWORD", "CHAIN MAIL"}, nil, []string{"has a weapon equipped that a PRIEST cannot use"}},
		{"a Mage in chain mail", engine.Mage, []string{"DAGGER", "CHAIN MAIL"}, nil, nil},
		{"a Thief in plate", engine.Thief, []string{"SHORT SWORD", "PLATE MAIL"}, nil, []string{"has only armor equipped that a THIEF cannot use"}},
	}
	for _, tc := range cases {
		c := newController(t, 1)
		ch := kitted(t, c, "HERO", tc.class, tc.equipped, tc.carried...)
		partyOf(c, ch)
		got := problemsOf(c)["HERO"]
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %q, want %q", tc.what, got, tc.want)
		}
	}
}

// Every class may use some weapon, and each finds a usable one counts: no
// class needs a dagger or a sword. Protection is asked of every class but
// the Mage, the one class the original arms with nothing heavier than robes.
func TestEquipmentCheckByClass(t *testing.T) {
	c := newController(t, 1)
	items := c.Game.Scenario.Items
	for k, name := range engine.ClassNames {
		class := engine.Class(k)
		if got, want := protectionExpected(items, class), class != engine.Mage; got != want {
			t.Errorf("%s: protection expected %v", name, got)
		}
		for i, it := range items {
			if it.TypeID != kindWeapon || !classUses(it, class) {
				continue
			}
			ch := engine.NewCharacter("ARMED", engine.Human, class, engine.Neutral)
			ch.AddItem(i, true)
			ch.Items[0].Equipped = true
			if p := equipmentProblems(items, ch); len(p) > 0 && p[0] != noArmor {
				t.Errorf("%s with %s: %q", name, it.Name, p)
			}
		}
	}
}

// The check is of those who can act, in party order, and as one sentence.
func TestEquipmentCheckLine(t *testing.T) {
	c := newController(t, 1)
	ready := kitted(t, c, "READY", engine.Fighter, []string{"LONG SWORD", "CHAIN MAIL"})
	bilbo := kitted(t, c, "BILBO", engine.Thief, []string{"LEATHER ARMOR"}, "SHORT SWORD")
	hades := kitted(t, c, "HADES", engine.Fighter, []string{"LONG SWORD"})
	dead := kitted(t, c, "GONE", engine.Fighter, nil)
	dead.Status = engine.Dead
	both := kitted(t, c, "BARE", engine.Priest, nil)
	partyOf(c, ready, bilbo, hades, dead, both)
	notes := c.EquipmentCheck()
	want := []ReadinessNote{
		{Slot: 2, Name: "BILBO", Problems: []string{noWeapon}},
		{Slot: 3, Name: "HADES", Problems: []string{noArmor}},
		{Slot: 5, Name: "BARE", Problems: []string{noWeapon, noArmor}},
	}
	if !reflect.DeepEqual(notes, want) {
		t.Fatalf("notes %+v", notes)
	}
	line := EquipmentCheckLine(notes)
	if line != "Equipment check: BILBO has no weapon equipped; HADES has no armor equipped; BARE has no weapon equipped and has no armor equipped." {
		t.Fatalf("line %q", line)
	}
	if EquipmentCheckLine(nil) != "" || EquipmentCheckLine(c.EquipmentCheck()[:0]) != "" {
		t.Fatal("a line with nothing to say")
	}
}

// Checking reads the game and changes nothing: no item equipped, bought or
// moved, no AC worked out, no random number drawn.
func TestEquipmentCheckChangesNothing(t *testing.T) {
	c := newController(t, 1)
	partyOf(c, kitted(t, c, "BILBO", engine.Thief, nil, "SHORT SWORD", "LEATHER ARMOR"), kitted(t, c, "MAGE", engine.Mage, []string{"DAGGER"}))
	before, _ := json.Marshal(c.Game.Town)
	seed(9)
	draw := rand.Int63()
	seed(9)
	for i := 0; i < 3; i++ {
		c.EquipmentCheck()
	}
	if after, _ := json.Marshal(c.Game.Town); string(after) != string(before) {
		t.Fatal("the check changed the town")
	}
	if rand.Int63() != draw {
		t.Fatal("the check drew a random number")
	}
}
