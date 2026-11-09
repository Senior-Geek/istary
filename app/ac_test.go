package app

import (
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// Armor class (finding #35): AC is what is worn, worked out again after
// every equipment change (equipping, selling, uncursing, dropping) and on
// every load, so no stale value is shown or saved.

const (
	itemChainMail   = 11 // CHAIN MAIL
	itemCursedRobe  = 73 // CURSED ROBE: a cursed type with a negative modifier
	itemLargeShield = 8
	itemHelm        = 14
)

// wornAC adds up what ch wears, independently of RecalcAC.
func wornAC(c *Controller, ch *engine.Character) int {
	ac := 10
	for i := 0; i < ch.ItemCount; i++ {
		if ch.Items[i].Equipped {
			ac -= c.Game.Scenario.Items[ch.Items[i].ItemIndex].ACMod
		}
	}
	return ac
}

// toBoltac takes the controller from Camp to Boltac's, ch at the counter.
func toBoltac(t *testing.T, c *Controller) {
	t.Helper()
	g := c.Game
	g.Phase, g.Town.InputMode, g.Town.Location, g.Town.EditChar = engine.PhaseTown, engine.InputNone, engine.Castle, nil
	must(t, c.GoTo(engine.Trading))
	must(t, c.ShopEnter(0))
}

// Equipping any mix of armour gives the AC of what is worn, whatever was
// worn before; choosing nothing takes it off.
func TestACFollowsTheEquipWalk(t *testing.T) {
	kit := []int{itemLeather, itemChainMail, itemSmallShield, itemLargeShield, itemHelm, itemLongSword}
	rng := rand.New(rand.NewSource(35))
	ch := engine.NewCharacter("ARMOURED", engine.Human, engine.Fighter, engine.Good)
	c := wearer(t, ch, kit)
	for round := 0; round < 40; round++ {
		var want []int
		for pos := range kit {
			if rng.Intn(2) == 0 {
				want = append(want, pos)
			}
		}
		equipWalk(t, c, 0, want...)
		if ch.AC != wornAC(c, ch) {
			t.Fatalf("round %d wearing %v: AC %d, worn %d", round, want, ch.AC, wornAC(c, ch))
		}
	}
	equipWalk(t, c, 0)
	if ch.AC != 10 {
		t.Fatalf("wearing nothing: AC %d", ch.AC)
	}
}

// Selling, dropping and uncursing worn armour each change the AC at once.
func TestACAfterEquipmentLeaves(t *testing.T) {
	t.Run("selling", func(t *testing.T) {
		ch := engine.NewCharacter("SELLER", engine.Human, engine.Fighter, engine.Good)
		c := wearer(t, ch, []int{itemChainMail, itemSmallShield})
		equipWalk(t, c, 0, 0, 1)
		worn := ch.AC
		toBoltac(t, c)
		must(t, c.Sell(1))
		if ch.AC != wornAC(c, ch) || ch.AC == worn {
			t.Fatalf("after selling the chain mail: AC %d (was %d), worn %d", ch.AC, worn, wornAC(c, ch))
		}
	})
	t.Run("dropping", func(t *testing.T) {
		ch := engine.NewCharacter("DROPPER", engine.Human, engine.Fighter, engine.Good)
		c := wearer(t, ch, []int{itemChainMail, itemSmallShield})
		equipWalk(t, c, 0, 0, 1)
		worn := ch.AC
		must(t, c.BeginDrop())
		must(t, c.DropItem(2))
		if ch.AC != wornAC(c, ch) || ch.AC == worn {
			t.Fatalf("after dropping the shield: AC %d (was %d), worn %d", ch.AC, worn, wornAC(c, ch))
		}
	})
	t.Run("uncursing", func(t *testing.T) {
		ch := engine.NewCharacter("ROBED", engine.Human, engine.Fighter, engine.Good)
		c := wearer(t, ch, []int{itemCursedRobe, itemHelm})
		equipWalk(t, c, 0, 0, 1)
		if !ch.Items[0].Cursed || ch.AC != wornAC(c, ch) {
			t.Fatalf("the cursed robe: cursed %v, AC %d, worn %d", ch.Items[0].Cursed, ch.AC, wornAC(c, ch))
		}
		cursedAC := ch.AC
		toBoltac(t, c)
		must(t, c.Uncurse(1))
		if ch.AC != wornAC(c, ch) || ch.AC == cursedAC {
			t.Fatalf("after uncursing: AC %d (was %d), worn %d", ch.AC, cursedAC, wornAC(c, ch))
		}
	})
	t.Run("selling what is not worn", func(t *testing.T) {
		ch := engine.NewCharacter("SPARE", engine.Human, engine.Fighter, engine.Good)
		c := wearer(t, ch, []int{itemChainMail, itemSmallShield})
		equipWalk(t, c, 0, 0)
		worn := ch.AC
		toBoltac(t, c)
		must(t, c.Sell(2))
		if ch.AC != worn {
			t.Fatalf("selling a spare shield changed AC %d -> %d", worn, ch.AC)
		}
	})
}

// A load works out every AC from what is worn: a stale stored value is
// not kept, and a right one survives a save and a load unchanged.
func TestACOnLoad(t *testing.T) {
	ch := engine.NewCharacter("SAVED", engine.Human, engine.Fighter, engine.Good)
	c := wearer(t, ch, []int{itemChainMail, itemLargeShield, itemHelm})
	equipWalk(t, c, 0, 0, 1, 2)
	want := ch.AC
	must(t, c.Save())
	for _, stale := range []int{want, -3, 10, 99} {
		ch.AC = stale
		must(t, c.Save())
		g := engine.New(c.Game.Scenario)
		must(t, g.Load())
		if got := g.Town.Roster.Characters[0].AC; got != want {
			t.Errorf("saved with AC %d: loaded %d, want %d", stale, got, want)
		}
	}
}
