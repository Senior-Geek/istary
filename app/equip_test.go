package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// Equipping in town (M8) is the equip walk, one item at a time: random
// characters equipping and taking off random items in town end exactly as
// a copy does through Camp's walk choosing the same, refusals included,
// with no random number drawn.

func cloneChar(t *testing.T, ch *engine.Character) *engine.Character {
	t.Helper()
	raw, err := json.Marshal(ch)
	if err != nil {
		t.Fatal(err)
	}
	var out engine.Character
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return &out
}

// wearing is ch's items and AC, to compare.
func wearing(ch *engine.Character) string {
	return fmt.Sprintf("AC%d %v", ch.AC, ch.Items[:ch.ItemCount])
}

func TestTownEquippingIsTheWalk(t *testing.T) {
	classes := []engine.Class{engine.Fighter, engine.Mage, engine.Priest, engine.Thief, engine.Bishop, engine.Samurai, engine.Lord, engine.Ninja}
	outcomes := map[string]int{}
	for seedN := int64(1); seedN <= 150; seedN++ {
		rng := rand.New(rand.NewSource(seedN))
		town := newController(t, 1)
		ch := engine.NewCharacter("WEARER", engine.Human, classes[rng.Intn(len(classes))], engine.Alignment(rng.Intn(3)))
		for n := 3 + rng.Intn(6); n > 0; n-- {
			ch.AddItem(1+rng.Intn(len(town.Game.Scenario.Items)-1), rng.Intn(2) == 0)
		}
		town.Game.Town.Roster.Add(ch)
		walker := wearer(t, cloneChar(t, ch), nil)
		other := walker.Game.Town.Party.Members[0]

		for step := 0; step < 12; step++ {
			n := 1 + rng.Intn(ch.ItemCount)
			equip := rng.Intn(3) > 0
			items := town.Game.Scenario.Items
			kind := items[ch.Items[n-1].ItemIndex].TypeID

			// What the walk is told to choose: what is worn now, with n
			// equipped in its category, or without n.
			var want []int
			for i := 0; i < other.ItemCount; i++ {
				p := other.Items[i]
				if !p.Equipped || i == n-1 || (equip && items[p.ItemIndex].TypeID == kind && !p.Cursed) {
					continue
				}
				want = append(want, i)
			}
			if equip {
				want = append(want, n-1)
			}

			seed(seedN*100 + int64(step))
			var err error
			if equip {
				_, err = town.TownEquip(0, n)
			} else {
				err = town.TownUnequip(0, n)
			}
			next := rand.Int63()
			seed(seedN*100 + int64(step))
			if next != rand.Int63() {
				t.Fatal("equipping drew a random number")
			}
			if err != nil { // refused: the walk keeps everything as it is
				want = want[:0]
				for i := 0; i < other.ItemCount; i++ {
					if other.Items[i].Equipped {
						want = append(want, i)
					}
				}
			}
			equipWalk(t, walker, 0, want...)
			if a, b := wearing(ch), wearing(other); a != b {
				t.Fatalf("seed %d step %d: equip %v item %d (%v)\n town %s\n walk %s", seedN, step, equip, n, err, a, b)
			}
			var r *Refusal
			switch {
			case err == nil:
				outcomes[fmt.Sprint("done ", equip)]++
			case errors.As(err, &r):
				outcomes[r.Message]++
			default:
				t.Fatalf("seed %d: %v", seedN, err)
			}
		}
	}
	t.Logf("outcomes: %v", outcomes)
	for _, want := range []string{"done true", "done false", "It is not something to equip.", "It is equipped.",
		"It is not equipped.", "It is cursed: only Boltac can remove it."} {
		if outcomes[want] == 0 {
			t.Errorf("never: %q", want)
		}
	}
	classOrCurse := 0
	for k, n := range outcomes {
		if strings.HasPrefix(k, "A ") {
			classOrCurse += n
		}
	}
	if classOrCurse == 0 {
		t.Errorf("never refused for the class or a cursed category: %v", outcomes)
	}
}

// The view says what equipping would answer, and the answers come with it.
func TestEquipmentViewIsTheOperations(t *testing.T) {
	ch := engine.NewCharacter("MAGE", engine.Human, engine.Mage, engine.Good)
	c := newController(t, 1)
	for _, it := range []int{itemLongSword, 5 /* STAFF */, 28 /* SCROLL/HALITO */, itemCursedRobe, 9 /* ROBES */} {
		ch.AddItem(it, true)
	}
	c.Game.Town.Roster.Add(ch)
	v, err := c.Equipment(0)
	must(t, err)
	want := []struct{ kind, equip string }{
		{"weapon", "A MAGE cannot use it."}, {"weapon", ""}, {"", "It is not something to equip."}, {"armour", ""}, {"armour", ""},
	}
	for i, w := range want {
		if v[i].Kind != w.kind || v[i].EquipRefusal != w.equip || v[i].UnequipRefusal != "It is not equipped." {
			t.Fatalf("item %d: %+v, want %+v", i+1, v[i], w)
		}
	}
	if _, err := c.TownEquip(0, 1); err == nil || err.Error() != "A MAGE cannot use it." {
		t.Fatalf("the long sword: %v", err)
	}
	if cursed, err := c.TownEquip(0, 4); err != nil || !cursed || ch.AC != 12 {
		t.Fatalf("the cursed robe: cursed %v, %v, AC %d", cursed, err, ch.AC)
	}
	v, _ = c.Equipment(0)
	if !v[3].Cursed || v[3].UnequipRefusal == "" || v[4].EquipRefusal != "A cursed armour is stuck in its place: only Boltac can remove it." {
		t.Fatalf("after the robe: %+v / %+v", v[3], v[4])
	}
	if _, err := c.TownEquip(0, 5); err == nil {
		t.Fatal("robes over a stuck robe")
	}
	if err := c.TownUnequip(0, 4); err == nil || !ch.Items[3].Equipped {
		t.Fatalf("took off a cursed robe: %v", err)
	}
	if _, err := c.TownEquip(0, 2); err != nil || !ch.Items[1].Equipped || ch.Items[1].Cursed {
		t.Fatalf("the staff: %v", err)
	}
	must(t, c.TownUnequip(0, 2))
	if ch.Items[1].Equipped || ch.AC != 12 {
		t.Fatalf("staff off: %v, AC %d", ch.Items[1].Equipped, ch.AC)
	}
	for _, bad := range []int{0, 6} {
		if _, err := c.TownEquip(0, bad); !errors.Is(err, ErrInvalidChoice) {
			t.Fatalf("item %d: %v", bad, err)
		}
	}
	if _, err := c.Equipment(1); !errors.Is(err, ErrNoSuchCharacter) {
		t.Fatalf("no such character: %v", err)
	}
}

// Equipping in town is refused for a member of a saved expedition and
// outside town, and what it does is saved and loaded.
func TestTownEquippingLimitsAndPersistence(t *testing.T) {
	engine.SetSaveDir(t.TempDir())
	t.Cleanup(func() { engine.SetSaveDir("") })
	c := newController(t, 1)
	ch := engine.NewCharacter("KEEPER", engine.Human, engine.Fighter, engine.Good)
	ch.AddItem(itemChainMail, true)
	ch.AddItem(itemEvilShield3, true)
	c.Game.Town.Roster.Add(ch)
	if _, err := c.TownEquip(0, 1); err != nil {
		t.Fatal(err)
	}
	if cursed, err := c.TownEquip(0, 2); err != nil || !cursed {
		t.Fatalf("an evil shield on a good fighter: cursed %v, %v", cursed, err)
	}
	must(t, c.Save())
	g := engine.New(c.Game.Scenario)
	must(t, g.Load())
	if got := wearing(g.Town.Roster.Characters[0]); got != wearing(ch) {
		t.Fatalf("loaded %s, saved %s", got, wearing(ch))
	}

	c.Game.Expedition = &engine.Expedition{Version: 1, Party: []string{"KEEPER"}}
	if _, err := c.TownEquip(0, 1); err != ErrOnExpedition {
		t.Fatalf("on an expedition: %v", err)
	}
	c.Game.Expedition = nil
	c.Game.Phase = engine.PhaseCamp
	if _, err := c.Equipment(0); !errors.Is(err, ErrWrongState) {
		t.Fatalf("in Camp: %v", err)
	}
}
