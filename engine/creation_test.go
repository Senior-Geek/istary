package engine

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func TestInitStatsUsesRaceBase(t *testing.T) {
	for r := Human; r <= Hobbit; r++ {
		seed(int64(r))
		cs := &CreationState{Race: r}
		cs.InitStats()
		if cs.Stats != RaceBaseStats[r] {
			t.Errorf("%v: stats = %v, want race base %v", r, cs.Stats, RaceBaseStats[r])
		}
		if cs.StatCursor != 0 {
			t.Errorf("%v: StatCursor = %d, want 0", r, cs.StatCursor)
		}
	}
}

// Bonus points: 7-10, then while < 20 a 1-in-11 chance of +10.
func TestRollBonusPointsRange(t *testing.T) {
	seed(7)
	seen := map[int]int{}
	for i := 0; i < 20000; i++ {
		b := RollBonusPoints()
		ok := (b >= 7 && b <= 10) || (b >= 17 && b <= 20) || (b >= 27 && b <= 29)
		if !ok {
			t.Fatalf("RollBonusPoints() = %d, outside 7-10/17-20/27-29", b)
		}
		seen[b]++
	}
	for b := 7; b <= 10; b++ {
		if seen[b] == 0 {
			t.Errorf("never rolled %d in 20000 tries", b)
		}
	}
	if seen[17]+seen[18]+seen[19]+seen[20] == 0 {
		t.Error("never rolled a +10 bonus in 20000 tries")
	}
}

func TestStatPointAllocation(t *testing.T) {
	cs := &CreationState{Race: Hobbit}
	cs.Stats = RaceBaseStats[Hobbit]
	cs.BonusPoints = 5

	// Cannot drop below the race base.
	cs.StatCursor = 0
	cs.RemoveStatPoint()
	if cs.Stats[0] != 5 || cs.BonusPoints != 5 {
		t.Errorf("remove below base: STR=%d points=%d, want 5/5", cs.Stats[0], cs.BonusPoints)
	}

	// Adding spends points; removing refunds them.
	cs.AddStatPoint()
	cs.AddStatPoint()
	cs.RemoveStatPoint()
	if cs.Stats[0] != 6 || cs.BonusPoints != 4 {
		t.Errorf("add/add/remove: STR=%d points=%d, want 6/4", cs.Stats[0], cs.BonusPoints)
	}

	// Hobbit LUCK starts at 15 and caps at 18.
	cs.StatCursor = 5
	for i := 0; i < 4; i++ {
		cs.AddStatPoint()
	}
	if cs.Stats[5] != 18 || cs.BonusPoints != 1 {
		t.Errorf("LUCK cap: LCK=%d points=%d, want 18/1", cs.Stats[5], cs.BonusPoints)
	}

	// No points left: adding does nothing.
	cs.StatCursor = 1
	cs.AddStatPoint()
	if cs.BonusPoints != 0 {
		t.Fatalf("points = %d, want 0", cs.BonusPoints)
	}
	cs.AddStatPoint()
	if cs.Stats[1] != 8 || cs.BonusPoints != 0 {
		t.Errorf("add with no points: IQ=%d points=%d, want 8/0", cs.Stats[1], cs.BonusPoints)
	}
}

func classSet(avail [8]bool) string {
	var names []string
	for c, ok := range avail {
		if ok {
			names = append(names, ClassAbbrev[c])
		}
	}
	return strings.Join(names, ",")
}

func TestClassEligibility(t *testing.T) {
	tests := []struct {
		name  string
		stats [6]int // STR IQ PIE VIT AGI LCK
		align Alignment
		want  string
	}{
		{"all 10 good", [6]int{10, 10, 10, 10, 10, 10}, Good, ""},
		{"fighter threshold", [6]int{11, 10, 10, 10, 10, 10}, Neutral, "FIG"},
		{"mage threshold", [6]int{10, 11, 10, 10, 10, 10}, Evil, "MAG"},
		{"priest good", [6]int{10, 10, 11, 10, 10, 10}, Good, "PRI"},
		{"priest not neutral", [6]int{10, 10, 11, 10, 10, 10}, Neutral, ""},
		{"thief neutral", [6]int{10, 10, 10, 10, 11, 10}, Neutral, "THI"},
		{"thief not good", [6]int{10, 10, 10, 10, 11, 10}, Good, ""},
		{"bishop evil", [6]int{10, 12, 12, 10, 10, 10}, Evil, "MAG,PRI,BIS"},
		{"bishop not neutral", [6]int{10, 12, 12, 10, 10, 10}, Neutral, "MAG"},
		{"samurai neutral", [6]int{15, 11, 10, 14, 10, 10}, Neutral, "FIG,MAG,SAM"},
		{"samurai not evil", [6]int{15, 11, 10, 14, 10, 10}, Evil, "FIG,MAG"},
		{"lord good", [6]int{15, 12, 12, 15, 14, 15}, Good, "FIG,MAG,PRI,BIS,SAM,LOR"},
		{"lord not neutral", [6]int{15, 12, 12, 15, 14, 15}, Neutral, "FIG,MAG,THI,SAM"},
		{"ninja evil", [6]int{15, 15, 15, 15, 15, 15}, Evil, "FIG,MAG,PRI,THI,BIS,NIN"},
		{"ninja one short", [6]int{15, 15, 15, 15, 15, 14}, Evil, "FIG,MAG,PRI,THI,BIS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &CreationState{Stats: tt.stats, Alignment: tt.align}
			if got := classSet(cs.ClassAvailability()); got != tt.want {
				t.Errorf("ClassAvailability = %q, want %q", got, tt.want)
			}
			var list []string
			for _, c := range cs.CalculateAvailableClasses() {
				list = append(list, ClassAbbrev[c])
			}
			if got := strings.Join(list, ","); got != tt.want {
				t.Errorf("CalculateAvailableClasses = %q, want %q", got, tt.want)
			}
		})
	}
}

// Training Grounds class change uses the same rules as creation.
func TestCharClassQualifiesMatchesCreation(t *testing.T) {
	r := rand.New(rand.NewSource(99))
	for i := 0; i < 2000; i++ {
		var stats [6]int
		for j := range stats {
			stats[j] = 3 + r.Intn(16)
		}
		align := Alignment(r.Intn(3))
		cs := &CreationState{Stats: stats, Alignment: align}
		c := &Character{Alignment: align, Strength: stats[0], IQ: stats[1], Piety: stats[2],
			Vitality: stats[3], Agility: stats[4], Luck: stats[5]}
		if a, b := cs.ClassAvailability(), CharClassQualifies(c); a != b {
			t.Fatalf("stats %v %v: creation %s, class change %s", stats, align, classSet(a), classSet(b))
		}
	}
}

// Starting HP is the class die + VIT modifier, then two 50% passes of x9/10, min 2.
func TestRollHPRange(t *testing.T) {
	seed(3)
	for class := Fighter; class <= Ninja; class++ {
		for _, vit := range []int{3, 5, 10, 16, 18} {
			base := ClassHPDie[class] + vitMod(vit)
			allowed := map[int]bool{}
			for _, v := range []int{base, base * 9 / 10, base * 9 / 10 * 9 / 10} {
				if v < 2 {
					v = 2
				}
				allowed[v] = true
			}
			for i := 0; i < 200; i++ {
				if hp := RollHP(class, vit); !allowed[hp] {
					t.Fatalf("RollHP(%v, %d) = %d, allowed %v", class, vit, hp, allowed)
				}
			}
		}
	}
}

func TestFinalizeCharacterRules(t *testing.T) {
	for class := Fighter; class <= Ninja; class++ {
		seed(int64(class) + 100)
		cs := &CreationState{Name: "X", Race: Human, Class: class, Alignment: Good,
			Stats: [6]int{15, 15, 15, 15, 15, 15}, Password: "PW"}
		c := cs.FinalizeCharacter()
		if c.Level != 1 || c.AC != 10 || c.Status != OK || c.MaxLevAC != 1 {
			t.Errorf("%v: level/AC/status = %d/%d/%v", class, c.Level, c.AC, c.Status)
		}
		if c.Gold < 90 || c.Gold > 189 {
			t.Errorf("%v: gold %d outside 90-189", class, c.Gold)
		}
		if c.Age < 18*52 || c.Age >= 18*52+300 {
			t.Errorf("%v: age %d outside %d-%d", class, c.Age, 18*52, 18*52+299)
		}
		if c.HP != c.MaxHP || c.HP < 2 {
			t.Errorf("%v: HP %d/%d", class, c.HP, c.MaxHP)
		}
		if c.Password != "PW" || c.Strength != 15 || c.Luck != 15 {
			t.Errorf("%v: password/stats not copied", class)
		}
		mage := class == Mage || class == Bishop
		priest := class == Priest
		if got := c.SpellKnown[SpellIndex["HALITO"]] && c.SpellKnown[SpellIndex["KATINO"]]; got != mage || (c.MageSpells[0] == 2) != mage {
			t.Errorf("%v: mage spells known=%v slots=%v", class, got, c.MageSpells)
		}
		if got := c.SpellKnown[SpellIndex["DIOS"]] && c.SpellKnown[SpellIndex["BADIOS"]]; got != priest || (c.PriestSpells[0] == 2) != priest {
			t.Errorf("%v: priest spells known=%v slots=%v", class, got, c.PriestSpells)
		}
	}
}

// Golden record of seeded creation, so a refactor that changes the order or
// number of random draws is caught.
func TestFinalizeCharacterSeededGolden(t *testing.T) {
	var out strings.Builder
	for class := Fighter; class <= Ninja; class++ {
		seed(2026)
		cs := &CreationState{Name: ClassNames[class], Race: Race(int(class) % 5), Class: class,
			Alignment: Evil, Stats: [6]int{15, 15, 15, 15, 15, 15}}
		cs.BonusPoints = RollBonusPoints()
		c := cs.FinalizeCharacter()
		fmt.Fprintf(&out, "%-8s bonus=%d hp=%d gold=%d age=%d\n", ClassAbbrev[class], cs.BonusPoints, c.HP, c.Gold, c.Age)
	}
	checkGolden(t, "creation_seeded.golden", []byte(out.String()))
}

func TestRosterLimitAndRemove(t *testing.T) {
	var r Roster
	for i := 0; i < 20; i++ {
		if !r.Add(NewCharacter(fmt.Sprintf("C%02d", i), Human, Fighter, Good)) {
			t.Fatalf("Add #%d failed, roster should hold 20", i+1)
		}
	}
	if r.Add(NewCharacter("EXTRA", Human, Fighter, Good)) {
		t.Error("Add should fail when roster has 20 characters")
	}
	r.Remove("C05")
	if len(r.Characters) != 19 {
		t.Fatalf("after Remove, roster has %d, want 19", len(r.Characters))
	}
	for _, c := range r.Characters {
		if c.Name == "C05" {
			t.Error("C05 still in roster after Remove")
		}
	}
	r.Remove("NOBODY") // no-op
	if len(r.Characters) != 19 {
		t.Error("removing an unknown name should not change the roster")
	}
}

func TestPartyCounts(t *testing.T) {
	mk := func(s Status) *Character {
		c := NewCharacter("X", Human, Fighter, Good)
		c.Status = s
		return c
	}
	p := Party{Members: []*Character{mk(OK), mk(Asleep), mk(Afraid), mk(Paralyzed), mk(Dead), nil}}
	if p.Size() != 5 {
		t.Errorf("Size = %d, want 5 (nil slots ignored)", p.Size())
	}
	if p.ActiveCount() != 3 {
		t.Errorf("ActiveCount = %d, want 3 (OK, asleep, afraid)", p.ActiveCount())
	}
	for _, tt := range []struct {
		s           Status
		alive, dead bool
	}{
		{OK, true, false}, {Asleep, true, false}, {Afraid, true, false},
		{Paralyzed, false, false}, {Stoned, false, false},
		{Dead, false, true}, {Ashed, false, true}, {Lost, false, true},
	} {
		c := mk(tt.s)
		if c.IsAlive() != tt.alive || c.IsDead() != tt.dead {
			t.Errorf("%v: IsAlive=%v IsDead=%v, want %v/%v", tt.s, c.IsAlive(), c.IsDead(), tt.alive, tt.dead)
		}
	}
}

func TestItemManagement(t *testing.T) {
	a := NewCharacter("A", Human, Fighter, Good)
	b := NewCharacter("B", Human, Fighter, Good)
	for i := 0; i < 8; i++ {
		if !a.AddItem(10+i, true) {
			t.Fatalf("AddItem #%d failed", i+1)
		}
	}
	if a.AddItem(99, true) {
		t.Error("AddItem should fail with 8 items")
	}

	if !a.EquipItem(0) || a.EquipItem(0) {
		t.Error("EquipItem should succeed once, then report already equipped")
	}
	a.Items[0].Cursed = true
	if a.UnequipItem(0) || a.DropItem(0) || a.TradeItem(0, b) {
		t.Error("cursed equipped item must not unequip, drop or trade")
	}

	// Drop shifts later items down.
	if !a.DropItem(1) || a.ItemCount != 7 || a.Items[1].ItemIndex != 12 || a.Items[7] != (Possession{}) {
		t.Errorf("DropItem(1): count=%d items=%v", a.ItemCount, a.Items)
	}

	// Trade moves the item, unequipped, to the target.
	a.EquipItem(1)
	if !a.TradeItem(1, b) {
		t.Fatal("TradeItem failed")
	}
	if b.ItemCount != 1 || b.Items[0].ItemIndex != 12 || b.Items[0].Equipped {
		t.Errorf("target got %+v", b.Items[0])
	}
	if a.ItemCount != 6 || a.Items[1].ItemIndex != 13 {
		t.Errorf("source after trade: count=%d items=%v", a.ItemCount, a.Items)
	}
}

func TestNewCharacterJSONDefaults(t *testing.T) {
	c := NewCharacter("NEW", Gnome, Thief, Neutral)
	raw, _ := json.Marshal(c)
	var back Character
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Level != 1 || back.Status != OK || back.Equipment != [8]int{-1, -1, -1, -1, -1, -1, -1, -1} {
		t.Errorf("NewCharacter defaults lost in JSON: %+v", back)
	}
}
