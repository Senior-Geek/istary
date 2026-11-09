package app

import (
	"slices"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// Curses (finding #34): equipping a cursed type or another alignment's item
// sticks it to the wearer, in Camp's equip walk, through a save and a load,
// until Boltac uncurses it (and takes it).

const (
	itemLongSword    = 1
	itemSmallShield  = 7
	itemLeather      = 10 // LEATHER ARMOR
	itemLongSwordM1  = 29 // a cursed type
	itemEvilShield3  = 64 // EVIL SHIELD +3: evil
	itemNeutPlate2   = 63 // NEUT P-MAIL + 2: neutral
	itemCursedHelmet = 78 // CURSED HELMET: a cursed type, AC -2
)

// wearer is a party of one in Camp: ch, carrying items (identified unless
// listed in unknown).
func wearer(t *testing.T, ch *engine.Character, items []int, unknown ...int) *Controller {
	t.Helper()
	engine.SetSaveDir(t.TempDir())
	t.Cleanup(func() { engine.SetSaveDir("") })
	c := newController(t, 1)
	for _, it := range items {
		ch.AddItem(it, !slices.Contains(unknown, it))
	}
	ch.Gold, ch.InMaze = 5000, true
	c.Game.Town.Roster.Add(ch)
	c.Game.Town.Party.Members = append(c.Game.Town.Party.Members, ch)
	must(t, c.EnterMaze())
	return c
}

// equipWalk runs Camp's E)QUIP for the party member in slot, choosing the
// items at positions want (0-based) where they are offered, nothing
// elsewhere; it returns the messages the walk showed.
func equipWalk(t *testing.T, c *Controller, slot int, want ...int) []string {
	t.Helper()
	town := c.Game.Town
	if town.InputMode != engine.InputInspect {
		must(t, c.CampInspect(slot))
	}
	must(t, c.BeginEquip())
	var said []string
	for town.InputMode == engine.InputEquip {
		n := slices.IndexFunc(town.EquipChoices, func(pos int) bool { return slices.Contains(want, pos) })
		if n >= 0 {
			must(t, c.EquipChoose(n+1))
		} else {
			must(t, c.EquipSkip())
		}
		if town.Message != "" {
			said = append(said, town.Message)
		}
	}
	return said
}

func TestEquippingACursedTypeSticks(t *testing.T) {
	ch := engine.NewCharacter("STUCK", engine.Human, engine.Fighter, engine.Good)
	c := wearer(t, ch, []int{itemLongSwordM1, itemLongSword}, itemLongSwordM1)
	if ch.Items[0].Cursed {
		t.Fatal("cursed before it was ever equipped")
	}
	said := equipWalk(t, c, 0, 0)
	if !ch.Items[0].Equipped || !ch.Items[0].Cursed || !slices.Contains(said, "** CURSED **") {
		t.Fatalf("equipped %v cursed %v, said %q", ch.Items[0].Equipped, ch.Items[0].Cursed, said)
	}
	if ch.Items[0].Identified {
		t.Fatal("equipping identified it")
	}

	// It holds the weapon's place: a second walk cannot change it.
	equipWalk(t, c, 0, 1)
	if !ch.Items[0].Equipped || ch.Items[1].Equipped {
		t.Fatalf("the long sword replaced the cursed one: %+v", ch.Items[:2])
	}
	// Nor can it be dropped.
	must(t, c.BeginDrop())
	must(t, c.DropItem(1))
	if ch.ItemCount != 2 || c.Game.Town.Message != "CURSED" {
		t.Fatalf("dropped: %d items, %q", ch.ItemCount, c.Game.Town.Message)
	}
	must(t, c.EndDrop())

	// It survives a save and a load.
	must(t, c.Save())
	g := engine.New(c.Game.Scenario)
	must(t, g.Load())
	back := g.Town.Roster.Characters[0]
	if p := back.Items[0]; !p.Equipped || !p.Cursed || p.Identified {
		t.Fatalf("after a load: %+v", p)
	}
}

func TestEquippingAnotherAlignmentsItemSticks(t *testing.T) {
	for _, tc := range []struct {
		align  engine.Alignment
		item   int
		cursed bool
	}{
		{engine.Good, itemEvilShield3, true},
		{engine.Neutral, itemEvilShield3, true},
		{engine.Evil, itemEvilShield3, false},
		{engine.Good, itemSmallShield, false}, // no alignment
		{engine.Good, itemNeutPlate2, true},
		{engine.Neutral, itemNeutPlate2, false},
		{engine.Evil, itemCursedHelmet, true}, // a cursed type, whatever the alignment
	} {
		ch := engine.NewCharacter("WEARER", engine.Human, engine.Fighter, tc.align)
		c := wearer(t, ch, []int{tc.item})
		equipWalk(t, c, 0, 0)
		if !ch.Items[0].Equipped || ch.Items[0].Cursed != tc.cursed {
			t.Errorf("%v wearing item %d: equipped %v cursed %v, want cursed %v", tc.align, tc.item, ch.Items[0].Equipped, ch.Items[0].Cursed, tc.cursed)
		}
	}
}

// Boltac's U)NCURSE takes a stuck item off and away: the curse ends with
// it, and the place it held is free again.
func TestBoltacUncursesAStuckItem(t *testing.T) {
	ch := engine.NewCharacter("STUCK", engine.Human, engine.Fighter, engine.Good)
	c := wearer(t, ch, []int{itemLongSwordM1, itemLongSword})
	equipWalk(t, c, 0, 0)
	c.Game.Phase = engine.PhaseTown
	c.Game.Town.InputMode = engine.InputNone
	c.Game.Town.Location = engine.Castle
	must(t, c.GoTo(engine.Trading))
	must(t, c.ShopEnter(0))
	v, _ := c.ShopView()
	if o := v.Items[0]; !o.Cursed || o.SellRefusal != "** WE DONT BUY CURSED ITEMS **" || o.UncurseRefusal != "" {
		t.Fatalf("Boltac's view of the stuck sword: %+v", o)
	}
	if err := c.Sell(1); err == nil || ch.ItemCount != 2 {
		t.Fatalf("sold a cursed item: %v", err)
	}
	gold := ch.Gold
	must(t, c.Uncurse(1))
	if ch.ItemCount != 1 || ch.Items[0].ItemIndex != itemLongSword || ch.Gold != gold-c.Game.Scenario.Items[itemLongSwordM1].Price/2 {
		t.Fatalf("after uncursing: %d items (%+v), gold %d", ch.ItemCount, ch.Items[0], ch.Gold)
	}
	for i := 0; i < ch.ItemCount; i++ {
		if ch.Items[i].Cursed || ch.Items[i].Equipped {
			t.Fatalf("still stuck: %+v", ch.Items[i])
		}
	}
}

// The view never tells a curse the player has not seen: an unidentified
// item that is not worn shows no curse and no refusal that depends on one
// (only an imported item can carry such a curse); trying says it.
func TestShopViewKeepsUnseenCursesHidden(t *testing.T) {
	ch := engine.NewCharacter("IMPORT", engine.Human, engine.Fighter, engine.Good)
	c := wearer(t, ch, []int{itemLongSwordM1, itemLongSwordM1, itemLongSwordM1}, itemLongSwordM1)
	ch.Items[0].Cursed = true                             // unseen
	ch.Items[1].Cursed, ch.Items[1].Equipped = true, true // worn: the player was told
	ch.Items[2].Cursed, ch.Items[2].Identified = true, true
	c.Game.Phase, c.Game.Town.Location = engine.PhaseTown, engine.Castle
	must(t, c.GoTo(engine.Trading))
	must(t, c.ShopEnter(0))
	v, _ := c.ShopView()
	if o := v.Items[0]; o.Cursed || o.SellRefusal != "" || o.UncurseRefusal != "" {
		t.Fatalf("an unseen curse is told: %+v", o)
	}
	for _, o := range v.Items[1:] {
		if !o.Cursed || o.SellRefusal == "" {
			t.Fatalf("a known curse is not told: %+v", o)
		}
	}
	if err := c.Sell(1); err == nil || err.Error() != "** WE DONT BUY CURSED ITEMS **" {
		t.Fatalf("trying to sell it: %v", err)
	}
}
