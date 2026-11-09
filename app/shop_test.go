package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// shopController is a scenario-1 controller at the castle with a random
// party: random classes, gold, items (identified, equipped and cursed at
// random) and stock, all from rng.
func shopController(t *testing.T, rng *rand.Rand) *Controller {
	t.Helper()
	c := newController(t, 1)
	g := c.Game
	for i := 0; i < 1+rng.Intn(6); i++ {
		ch := engine.NewCharacter(fmt.Sprintf("M%d", i), engine.Race(rng.Intn(5)), engine.Class(rng.Intn(8)), engine.Alignment(rng.Intn(3)))
		ch.Gold = []int{0, 3, 12, 40, 600, 2000, 100000}[rng.Intn(7)]
		for k := rng.Intn(9); k > 0; k-- {
			ch.Items[ch.ItemCount] = engine.Possession{ItemIndex: rng.Intn(len(g.Scenario.Items)), Identified: rng.Intn(2) == 0, Equipped: rng.Intn(3) == 0, Cursed: rng.Intn(4) == 0}
			ch.ItemCount++
		}
		g.Town.Roster.Add(ch)
		g.Town.Party.Members = append(g.Town.Party.Members, ch)
	}
	for k := rng.Intn(10); k > 0; k-- {
		g.SetStock(rng.Intn(len(g.Scenario.Items)), rng.Intn(4)-1)
	}
	return c
}

// twins are two controllers built from the same seed.
func shopTwins(t *testing.T, seed int64) (*Controller, *Controller) {
	return shopController(t, rand.New(rand.NewSource(seed))), shopController(t, rand.New(rand.NewSource(seed)))
}

// shopState is everything Boltac's can change: the town (party, items and
// gold included) and the stock. The page shown and the catalog cursor are
// the terminal's paging, which Buy does not do; they are left out unless
// asked for, except the cursor while Boltac asks about an item (it names
// the item).
func shopStateOf(t *testing.T, c *Controller, paging bool) string {
	t.Helper()
	town := *c.Game.Town
	if !paging {
		town.ShopPage = [6]int{}
		if town.ShopStep != engine.ShopBuyConfirm {
			town.ShopCatalog = 0
		}
	}
	raw, err := json.Marshal(town)
	if err != nil {
		t.Fatal(err)
	}
	stock := make([]int, len(c.Game.Scenario.Items))
	for i := range stock {
		stock[i] = c.Game.Stock(i)
	}
	return fmt.Sprintf("%v %s %v", c.Game.Phase, raw, stock)
}

// around is a's text around its first difference from b.
func around(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return a[max(0, i-120):min(len(a), i+80)]
}

// keysToStep is what toShopStep presses.
func keysToStep(c *Controller, step engine.ShopStep, letter rune) {
	town := c.Game.Town
	if town.InputMode == engine.InputShopPurchase {
		c.HandleKey(Key{Kind: KeyEnter})
	}
	if town.ShopStep == step {
		return
	}
	for town.ShopStep != engine.ShopMain {
		c.HandleKey(Key{Kind: KeyEnter})
	}
	c.HandleKey(Rune(letter))
}

// Random operations at Boltac's leave the game exactly as the terminal's
// keys for them do, draw no random number, and a refused or invalid one
// changes nothing.
func TestShopOperationsAreTheKeys(t *testing.T) {
	seen := map[string]int{}
	for seedN := int64(1); seedN <= 300; seedN++ {
		op, key := shopTwins(t, seedN)
		rng := rand.New(rand.NewSource(seedN + 7000))
		check := func(what string, err error) {
			t.Helper()
			if a, b := shopStateOf(t, op, false), shopStateOf(t, key, false); a != b {
				t.Fatalf("seed %d, %s (%v): ops and keys differ at\n ops  ...%s\n keys ...%s", seedN, what, err, around(a, b), around(b, a))
			}
		}
		for step := 0; step < 80; step++ {
			town := op.Game.Town
			seed(seedN*100 + int64(step))
			var what string
			var err error
			switch {
			case town.Location == engine.Castle:
				what, err = "GoTo(Trading)", op.GoTo(engine.Trading)
				key.HandleKey(Rune('B'))
			case town.ShopStep == engine.ShopWho:
				switch r := rng.Intn(8); {
				case r == 0:
					what, err = "GoTo(Castle)", op.GoTo(engine.Castle)
					key.HandleKey(Key{Kind: KeyEnter})
				default:
					slot := rng.Intn(7) - 1
					what, err = fmt.Sprintf("ShopEnter(%d)", slot), op.ShopEnter(slot)
					if err == nil {
						key.HandleKey(Rune('1' + rune(slot)))
					}
				}
			case town.ShopStep == engine.ShopBuyConfirm:
				yes := rng.Intn(2) == 0
				what, err = fmt.Sprintf("ConfirmUnusable(%v)", yes), op.ConfirmUnusable(yes)
				key.HandleKey(Rune(map[bool]rune{true: 'Y', false: 'N'}[yes]))
			default:
				switch r := rng.Intn(20); {
				case r < 8:
					item := rng.Intn(len(op.Game.Scenario.Items)+2) - 1
					what, err = fmt.Sprintf("Buy(%d)", item), op.Buy(item)
					if !errors.Is(err, ErrInvalidChoice) {
						keysToStep(key, engine.ShopBuy, 'B')
						for i := 0; i < 40 && !slices.Contains(key.Game.Town.ShopPage[:], item); i++ {
							key.HandleKey(Rune('F'))
						}
						key.HandleKey(Rune('P'))
						key.HandleKey(Rune('1' + rune(slices.Index(key.Game.Town.ShopPage[:], item))))
					}
				case r < 16:
					n := rng.Intn(10)
					kind := []struct {
						name   string
						do     func(int) error
						step   engine.ShopStep
						letter rune
					}{
						{"Sell", op.Sell, engine.ShopSell, 'S'},
						{"Uncurse", op.Uncurse, engine.ShopUncurse, 'U'},
						{"ShopIdentify", op.ShopIdentify, engine.ShopIdentify, 'I'},
					}[rng.Intn(3)]
					what, err = fmt.Sprintf("%s(%d)", kind.name, n), kind.do(n)
					if !errors.Is(err, ErrInvalidChoice) {
						keysToStep(key, kind.step, kind.letter)
						key.HandleKey(Rune('0' + rune(n)))
					}
				case r < 18:
					what, err = "ShopLeaveCustomer", op.ShopLeaveCustomer()
					for key.Game.Town.ShopStep != engine.ShopWho {
						key.HandleKey(Key{Kind: KeyEnter})
					}
				default:
					what, err = "LeaveShop", op.LeaveShop()
					for key.Game.Town.Location == engine.Trading {
						key.HandleKey(Key{Kind: KeyEnter})
					}
				}
			}
			next := rand.Int63()
			seed(seedN*100 + int64(step))
			if next != rand.Int63() {
				t.Fatalf("seed %d: %s drew a random number", seedN, what)
			}
			var r *Refusal
			switch {
			case errors.As(err, &r):
				seen["refused: "+r.Message]++
			case err != nil:
				seen["error: "+err.Error()]++
			default:
				seen[what[:min(len(what), 4)]]++
			}
			check(what, err)
		}
	}
	for _, want := range []string{"refused: ** YOU CANNOT AFFORD IT **", "refused: ** YOU CANT CARRY ANYTHING MORE **",
		"refused: ** WE DONT BUY CURSED ITEMS **", "refused: ** THAT IS NOT A CURSED ITEM **",
		"refused: ** YOU CANT AFFORD THE FEE **", "refused: ** THAT HAS BEEN IDENTIFIED **",
		"error: invalid choice", "error: no party member in that slot", "Buy(", "Sell", "Uncu", "Shop", "Conf", "Leav", "GoTo"} {
		if seen[want] == 0 {
			t.Errorf("never reached: %q (seen %v)", want, seen)
		}
	}
}

// When the item is on the page shown, Buy is exactly P and its place, page
// and cursor included.
func TestBuyOnThePageIsPAndItsPlace(t *testing.T) {
	for seedN := int64(1); seedN <= 200; seedN++ {
		op, key := shopTwins(t, seedN)
		if op.Game.Town.Party.Size() == 0 {
			continue
		}
		for _, c := range []*Controller{op, key} {
			c.press(Rune('B'), Rune('1'), Rune('B'))
		}
		place := rand.New(rand.NewSource(seedN)).Intn(6)
		item := op.Game.Town.ShopPage[place]
		if item < 0 {
			continue
		}
		err := op.Buy(item)
		key.press(Rune('P'), Rune('1'+rune(place)))
		if a, b := shopStateOf(t, op, true), shopStateOf(t, key, true); a != b {
			t.Fatalf("seed %d, Buy(%d) (%v):\n ops  %s\n keys %s", seedN, item, err, a, b)
		}
	}
}

// ShopView's prices, fees and refusals are what the operations then do:
// through random visits, each prediction is checked against the action.
func TestShopViewPredictsTheOperations(t *testing.T) {
	checked := map[string]int{}
	for seedN := int64(1); seedN <= 300; seedN++ {
		rng := rand.New(rand.NewSource(seedN))
		c := shopController(t, rng)
		if c.Game.Town.Party.Size() == 0 {
			continue
		}
		c.GoTo(engine.Trading)
		if _, err := c.ShopView(); err != nil {
			t.Fatal(err)
		}
		c.ShopEnter(rng.Intn(c.Game.Town.Party.Size()))
		for step := 0; step < 30; step++ {
			v, err := c.ShopView()
			if err != nil {
				t.Fatal(err)
			}
			ch := c.Game.Town.ShopChar
			if v.Name != ch.Name || v.Gold != ch.Gold || v.Customer != PartySlot(c.Game.Town, ch) || v.Room != 8-ch.ItemCount {
				t.Fatalf("seed %d: customer %+v", seedN, v)
			}
			if v.Step == "confirm" {
				c.ConfirmUnusable(rng.Intn(2) == 0)
				continue
			}
			gold := ch.Gold
			if rng.Intn(2) == 0 && len(v.Catalog) > 0 {
				o := v.Catalog[rng.Intn(len(v.Catalog))]
				err := c.Buy(o.Item)
				switch {
				case o.Refusal != "":
					var r *Refusal
					if !errors.As(err, &r) || r.Message != o.Refusal || ch.Gold != gold {
						t.Fatalf("seed %d: Buy(%d) predicted %q, got %v", seedN, o.Item, o.Refusal, err)
					}
					checked["buy refused"]++
				case !o.Usable:
					if err != nil || c.Game.Town.ShopStep != engine.ShopBuyConfirm || c.Game.Town.ShopCatalog != o.Item {
						t.Fatalf("seed %d: Buy(%d) of an unusable item: %v", seedN, o.Item, err)
					}
					checked["buy asks"]++
				default:
					if err != nil || ch.Gold != gold-o.Price || ch.Items[ch.ItemCount-1].ItemIndex != o.Item {
						t.Fatalf("seed %d: Buy(%d) for %d: %v, gold %d -> %d", seedN, o.Item, o.Price, err, gold, ch.Gold)
					}
					checked["bought"]++
				}
				continue
			}
			if len(v.Items) == 0 {
				continue
			}
			o := v.Items[rng.Intn(len(v.Items))]
			// A curse the player has not seen (an imported item) is not
			// told in advance (#34): Boltac's answer is the game's own.
			if p := ch.Items[o.N-1]; p.Cursed && !engine.KnownCursed(p) {
				item := c.Game.Scenario.Items[p.ItemIndex]
				_, o.SellRefusal = sellQuote(item, p)
				_, o.UncurseRefusal = uncurseQuote(ch, item, p)
				checked["unseen curse"]++
			}
			switch rng.Intn(3) {
			case 0:
				err := c.Sell(o.N)
				if (o.SellRefusal != "") != (err != nil) || (err == nil && ch.Gold != gold+o.Price) {
					t.Fatalf("seed %d: Sell(%d) predicted %d %q, got %v, gold %d -> %d", seedN, o.N, o.Price, o.SellRefusal, err, gold, ch.Gold)
				}
				checked["sell "+fmt.Sprint(err != nil)]++
			case 1:
				err := c.Uncurse(o.N)
				if (o.UncurseRefusal != "") != (err != nil) || (err == nil && ch.Gold != gold-o.Fee) {
					t.Fatalf("seed %d: Uncurse(%d) predicted %d %q, got %v", seedN, o.N, o.Fee, o.UncurseRefusal, err)
				}
				checked["uncurse "+fmt.Sprint(err != nil)]++
			default:
				err := c.ShopIdentify(o.N)
				if (o.IdentifyRefusal != "") != (err != nil) || (err == nil && (ch.Gold != gold-o.Fee || !ch.Items[o.N-1].Identified)) {
					t.Fatalf("seed %d: ShopIdentify(%d) predicted %d %q, got %v", seedN, o.N, o.Fee, o.IdentifyRefusal, err)
				}
				checked["identify "+fmt.Sprint(err != nil)]++
			}
			if err := c.shopAnswer(); err != nil && v.Message == "" && c.Game.Town.Message == "" {
				t.Fatal("a refusal without a message")
			}
		}
	}
	for _, k := range []string{"buy refused", "buy asks", "bought", "sell true", "sell false", "uncurse true", "uncurse false", "identify true", "identify false"} {
		if checked[k] == 0 {
			t.Errorf("never checked: %s (%v)", k, checked)
		}
	}
}

// ShopView: names unknown until identified, the catalog in Boltac's order
// with only what he sells, the question's item, and nothing at "who" or
// away from Boltac's.
func TestShopViewDetails(t *testing.T) {
	c := newController(t, 1)
	if _, err := c.ShopView(); !errors.Is(err, ErrWrongState) {
		t.Fatalf("at the castle: %v", err)
	}
	mage := engine.NewCharacter("MAGE", engine.Elf, engine.Mage, engine.Good)
	mage.Gold = 100
	mage.Items[0] = engine.Possession{ItemIndex: 14} // HELM, unidentified
	mage.Items[1] = engine.Possession{ItemIndex: 11, Identified: true, Equipped: true}
	mage.ItemCount = 2
	c.Game.Town.Roster.Add(mage)
	c.Game.Town.Party.Members = []*engine.Character{mage}
	if err := c.GoTo(engine.Trading); err != nil {
		t.Fatal(err)
	}
	if v, _ := c.ShopView(); v.Step != "who" || v.Customer != -1 || v.Catalog != nil || v.Items != nil {
		t.Fatalf("at who: %+v", v)
	}
	c.ShopEnter(0)
	seed(61)
	v, _ := c.ShopView()
	next := rand.Int63()
	seed(61)
	if next != rand.Int63() {
		t.Error("ShopView drew a random number")
	}
	if v.Step != "main" || v.Class != "MAGE" || v.Room != 6 {
		t.Fatalf("main: %+v", v)
	}
	if h := v.Items[0]; h.Name != "HELM" || h.Identified || h.Price != 1 || h.Fee != 50 || h.IdentifyRefusal != "" || h.UncurseRefusal != "** THAT IS NOT A CURSED ITEM **" {
		t.Errorf("the helm: %+v", h)
	}
	if m := v.Items[1]; m.Name != "CHAIN MAIL" || !m.Equipped || m.Price != 45 || m.IdentifyRefusal != "** THAT HAS BEEN IDENTIFIED **" {
		t.Errorf("the chain mail: %+v", m)
	}
	var order []int
	for _, o := range v.Catalog {
		order = append(order, o.Item)
		if !inCatalog(c.Game, o.Item) || o.Stock != c.Game.Stock(o.Item) {
			t.Fatalf("offer %+v", o)
		}
	}
	if !slices.IsSorted(order) || order[0] != 1 || len(order) != 31 {
		t.Fatalf("catalog %v", order)
	}
	sword := v.Catalog[0] // LONG SWORD: 25 GP, not for a mage
	if sword.Name != "LONG SWORD" || sword.Usable || sword.Refusal != "" || sword.Stock != -1 {
		t.Errorf("the long sword: %+v", sword)
	}
	c.Buy(1)
	if v, _ := c.ShopView(); v.Step != "confirm" || v.Confirm != 1 || v.Message != "UNUSABLE ITEM - CONFIRM BUY (Y/N) ? >" {
		t.Errorf("asking: %+v", v)
	}
	for _, f := range []func() error{func() error { return c.Buy(1) }, func() error { return c.Sell(1) }} {
		if err := f(); !errors.Is(err, ErrWrongState) || c.Game.Town.ShopStep != engine.ShopBuyConfirm {
			t.Errorf("while asking: %v", err)
		}
	}
}

// Leaving: from every step of a visit, LeaveShop reaches the castle and
// ShopLeaveCustomer "WHO WILL ENTER", as RETURN does; GoTo(Castle) only
// from "WHO WILL ENTER"; nothing outside Boltac's.
func TestLeavingTheShop(t *testing.T) {
	setups := map[string]string{"main": "", "buy": "B", "prompt": "BP", "confirm": "BP1", "sell": "S", "uncurse": "U", "identify": "I"}
	for name, keys := range setups {
		for _, leave := range []string{"LeaveShop", "ShopLeaveCustomer", "GoTo"} {
			c := newController(t, 1)
			mage := engine.NewCharacter("MAGE", engine.Elf, engine.Mage, engine.Good)
			mage.Gold = 100
			mage.Items[0] = engine.Possession{ItemIndex: 14}
			mage.ItemCount = 1
			c.Game.Town.Roster.Add(mage)
			c.Game.Town.Party.Members = []*engine.Character{mage}
			c.GoTo(engine.Trading)
			c.ShopEnter(0)
			for _, r := range keys {
				c.HandleKey(Rune(r))
			}
			switch leave {
			case "LeaveShop":
				if err := c.LeaveShop(); err != nil || c.Game.Town.Location != engine.Castle || c.Game.Town.InputMode != engine.InputNone {
					t.Errorf("%s: LeaveShop: %v at %v", name, err, c.Game.Town.Location)
				}
			case "ShopLeaveCustomer":
				if err := c.ShopLeaveCustomer(); err != nil || c.Game.Town.ShopStep != engine.ShopWho || c.Game.Town.ShopChar != nil {
					t.Errorf("%s: ShopLeaveCustomer: %v", name, err)
				}
			default:
				if err := c.GoTo(engine.Castle); !errors.Is(err, ErrWrongState) || c.Game.Town.Location != engine.Trading {
					t.Errorf("%s: GoTo(Castle) with a customer: %v", name, err)
				}
			}
		}
	}
	c := newController(t, 1)
	for _, f := range []func() error{c.LeaveShop, c.ShopLeaveCustomer, func() error { return c.ShopEnter(0) }, func() error { return c.Buy(1) },
		func() error { return c.Sell(1) }, func() error { return c.ConfirmUnusable(true) }} {
		if err := f(); !errors.Is(err, ErrWrongState) {
			t.Errorf("at the castle: %v", err)
		}
	}
}
