package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
	"github.com/gdamore/tcell/v2"
)

// Characterization tests for Boltac's Trading Post (M8-0): they pin what
// the terminal does today, before the shop moves into the controller. They
// change no code; where today's behaviour is a known question (findings
// #12, #34, #35, B6, B7) it is pinned as it is and said so.

// Wizardry I items the tests use.
const (
	itemLongSword    = 1  // 25 GP, unlimited, fighters, samurai, lords, ninja
	itemStaff        = 5  // 10 GP, unlimited, anyone
	itemChainMail    = 11 // 90 GP, unlimited, AC 3
	itemHelm         = 14 // 100 GP, unlimited, unknown name HELM
	itemShortSword1  = 18 // SHORT SWORD+1, 70536 GP, 2 in stock
	itemScrollHalito = 28 // 500 GP, 27 in stock, unknown name SCROLL
	itemLongSwordM1  = 29 // LONG SWORD -1, a cursed type, 1 in stock
	itemLeatherM1    = 35 // LEATHER -1, a cursed type
	itemPlateMail1   = 24 // PLATE MAIL + 1, not stocked (0)
)

// shopGame is a scenario-1 game at the castle with a party of three: a rich
// fighter, a poor mage and a bishop carrying the items the tests sell,
// identify and uncurse.
func shopGame(t *testing.T) (*engine.GameState, *driver) {
	t.Helper()
	g := newGame(t, 1)
	fighter := character("FIGHTER", engine.Human, engine.Fighter, engine.Good)
	fighter.Gold = 2000
	mage := character("MAGE", engine.Elf, engine.Mage, engine.Good)
	mage.Gold = 40
	bishop := character("BISHOP", engine.Human, engine.Bishop, engine.Good)
	bishop.Gold, bishop.AC = 600, 7
	for _, p := range []engine.Possession{
		{ItemIndex: itemStaff, Identified: true, Equipped: true},
		{ItemIndex: itemChainMail, Identified: true, Equipped: true},
		{ItemIndex: itemScrollHalito},
		{ItemIndex: itemLongSwordM1, Identified: true, Equipped: true, Cursed: true},
		{ItemIndex: itemHelm},
		{ItemIndex: itemLeatherM1, Cursed: true},
	} {
		bishop.Items[bishop.ItemCount] = p
		bishop.ItemCount++
	}
	for _, c := range []*engine.Character{fighter, mage, bishop} {
		g.Town.Roster.Add(c)
		g.Town.Party.Members = append(g.Town.Party.Members, c)
	}
	return g, newDriver(t, g)
}

// shopAs enters Boltac's (castle B) and chooses party member slot (1-based).
func shopAs(d *driver, slot int) *driver {
	return d.keys("B").keys(fmt.Sprint(slot))
}

var shopSteps = map[engine.ShopStep]string{
	engine.ShopWho: "who", engine.ShopMain: "main", engine.ShopBuy: "buy", engine.ShopBuyConfirm: "confirm",
	engine.ShopSell: "sell", engine.ShopUncurse: "uncurse", engine.ShopIdentify: "identify",
}

// shopState is one line of what the shop holds: where it is, Boltac's
// message, every member's gold, the customer's items and AC, the page shown
// and the stock of the items the tests touch.
func shopState(g *engine.GameState) string {
	town := g.Town
	var b strings.Builder
	fmt.Fprintf(&b, "%v/%s input=%d msg=%q", town.Location, shopSteps[town.ShopStep], town.InputMode, town.Message)
	for _, m := range town.Party.Members {
		fmt.Fprintf(&b, " %s:%d", m.Name, m.Gold)
	}
	if c := town.ShopChar; c != nil {
		fmt.Fprintf(&b, " | %s AC%d [", c.Name, c.AC)
		for i := 0; i < c.ItemCount; i++ {
			p := c.Items[i]
			fmt.Fprintf(&b, " %d", p.ItemIndex)
			for _, f := range []struct {
				on  bool
				tag string
			}{{p.Identified, "i"}, {p.Equipped, "e"}, {p.Cursed, "c"}} {
				if f.on {
					b.WriteString(f.tag)
				}
			}
		}
		b.WriteString(" ]")
	}
	if town.ShopStep == engine.ShopBuy || town.ShopStep == engine.ShopBuyConfirm {
		fmt.Fprintf(&b, " | cursor %d page %v", town.ShopCatalog, town.ShopPage)
	}
	b.WriteString(" | stock")
	for _, i := range []int{itemLongSword, itemChainMail, itemHelm, itemShortSword1, itemScrollHalito, itemLongSwordM1, itemLeatherM1, itemPlateMail1} {
		fmt.Fprintf(&b, " %d:%d", i, g.Stock(i))
	}
	return b.String()
}

// shopScript is a session at Boltac's, key by key ("\r" is RETURN). It goes
// through every screen and every refusal.
var shopScript = []struct{ keys, why string }{
	{"B", "enter: who will enter"},
	{"9", "no such member: nothing"},
	{"1", "the fighter shops"},
	{"B", "browse: page 1"},
	{"F", "forward"},
	{"F", "forward again"},
	{"B", "back"},
	{"S", "start"},
	{"P", "purchase prompt"},
	{"1", "buy a long sword (usable, unlimited)"},
	{"F", ""},
	{"F", ""},
	{"F", ""},
	{"F", ""},
	{"F", "to the stocked magic items"},
	{"S", ""},
	{"\r", "leave the catalog"},
	{"L", "leave: who will enter"},
	{"2", "the mage shops (40 GP)"},
	{"B", ""},
	{"P", ""},
	{"1", "a long sword: unusable by a mage, confirm"},
	{"N", "no"},
	{"P", ""},
	{"1", ""},
	{"Y", "yes: its your money; the catalog cursor is now the item"},
	{"F", "the page after a confirmed buy starts from it"},
	{"P", ""},
	{"3", "the flail (150 GP): can't afford"},
	{"P", ""},
	{"\r", "RETURN leaves the purchase prompt"},
	{"\r", "RETURN leaves the catalog"},
	{"S", "sell"},
	{"1", "sell the long sword: half of 25; the sell list stays open"},
	{"\r", "RETURN at a list: the main menu"},
	{"S", "sell with no items: nothing"},
	{"U", "uncurse with no items: nothing"},
	{"I", "identify with no items: nothing"},
	{"\r", "RETURN at the main menu: who will enter"},
	{"3", "the bishop shops"},
	{"S", "sell"},
	{"4", "the cursed long sword: refused, back to the main menu"},
	{"S", ""},
	{"3", "the unidentified scroll: 1 GP (B6)"},
	{"1", "the equipped staff: sold, no question (B7), AC unchanged (#35)"},
	{"\r", ""},
	{"U", "uncurse"},
	{"1", "the chain mail is not cursed: back to the main menu"},
	{"U", ""},
	{"4", "the cursed leather: a 750 GP fee, more than the bishop has"},
	{"I", "identify"},
	{"1", "the chain mail is identified already: back to the main menu"},
	{"I", ""},
	{"3", "the helm: identified for 50; the list stays open"},
	{"\r", ""},
	{"B", ""},
	{"P", ""},
	{"1", "a long sword: unusable by a bishop, confirm"},
	{"\r", "RETURN at the confirmation: no"},
	{"\r", "RETURN at the catalog: the main menu"},
	{"L", "leave: who will enter"},
	{"\r", "RETURN at who will enter: the castle"},
}

func pressAll(d *driver, keys string) {
	for _, r := range keys {
		if r == '\r' {
			d.enter()
		} else {
			d.dispatch(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
		}
	}
}

// The whole session, state by state, is pinned in a golden file, and draws
// no random number.
func TestShopTranscript(t *testing.T) {
	g, d := shopGame(t)
	var out strings.Builder
	out.WriteString("start " + shopState(g) + "\n")
	seed(31)
	for _, step := range shopScript {
		pressAll(d, step.keys)
		key := strings.ReplaceAll(step.keys, "\r", "RET")
		fmt.Fprintf(&out, "%-4s %s\n", key, shopState(g))
		if step.why != "" {
			fmt.Fprintf(&out, "     # %s\n", step.why)
		}
	}
	next := rand.Int63()
	seed(31)
	if next != rand.Int63() {
		t.Error("the shop drew a random number")
	}
	checkGolden(t, "shop_transcript.golden", []byte(out.String()))
}

// Every Boltac's screen as the terminal draws it.
func TestShopScreenSnapshots(t *testing.T) {
	g, _ := shopGame(t)
	d := newDriverWithScreen(t, g)
	var out strings.Builder
	snap := func(name string) {
		d.screen.Clear()
		redraw(d.screen.Screen, g)
		out.WriteString("=== " + name + "\n" + d.screen.text() + "\n")
	}
	pressAll(d, "B")
	snap("who will enter")
	pressAll(d, "3")
	snap("main menu")
	pressAll(d, "B")
	snap("catalog, page 1")
	pressAll(d, "FFFFF")
	snap("catalog, stocked magic items")
	pressAll(d, "SP")
	snap("purchase prompt")
	pressAll(d, "1")
	snap("unusable item confirmation")
	pressAll(d, "\r\r")
	pressAll(d, "S")
	snap("sell list")
	pressAll(d, "4")
	snap("cursed item refused")
	pressAll(d, "U")
	snap("uncurse list")
	pressAll(d, "\rI")
	snap("identify list")
	checkGolden(t, "screens_shop.golden", []byte(out.String()))
}

// The rules the transcript shows, stated one by one.
func TestShopRules(t *testing.T) {
	t.Run("buying", func(t *testing.T) {
		g, d := shopGame(t)
		f := g.Town.Party.Members[0]
		shopAs(d, 1).keys("BP1")
		if f.Gold != 2000-25 || f.ItemCount != 1 || f.Items[0] != (engine.Possession{ItemIndex: itemLongSword, Identified: true}) {
			t.Errorf("a long sword: gold %d, items %v", f.Gold, f.Items[:f.ItemCount])
		}
		if g.Town.Message != "** JUST WHAT YOU NEEDED **" || g.Stock(itemLongSword) != -1 {
			t.Errorf("message %q, unlimited stock %d", g.Town.Message, g.Stock(itemLongSword))
		}
		for f.ItemCount < 8 {
			d.keys("P1")
		}
		gold := f.Gold
		if d.keys("P1"); g.Town.Message != "** YOU CANT CARRY ANYTHING MORE **" || f.Gold != gold {
			t.Errorf("a ninth item: %q", g.Town.Message)
		}
	})
	t.Run("limited stock", func(t *testing.T) {
		g, d := shopGame(t)
		f := g.Town.Party.Members[0]
		f.Gold = 1 << 20
		shopAs(d, 1).keys("B")
		find := func(item int) int {
			for i := 0; i < 20; i++ {
				for slot, idx := range g.Town.ShopPage {
					if idx == item {
						return slot + 1
					}
				}
				d.keys("F")
			}
			return 0
		}
		for want := 1; want >= 0; want-- {
			slot := find(itemShortSword1)
			if slot == 0 {
				t.Fatal("SHORT SWORD+1 is not in the catalog")
			}
			d.keys("P" + fmt.Sprint(slot))
			if g.Stock(itemShortSword1) != want {
				t.Fatalf("stock %d, want %d", g.Stock(itemShortSword1), want)
			}
		}
		d.keys("S")
		if find(itemShortSword1) != 0 {
			t.Error("a sold-out item is still listed")
		}
		if find(itemLongSwordM1) != 0 || find(itemPlateMail1) != 0 {
			t.Error("a cursed type or an unstocked item is listed")
		}
	})
	t.Run("gold", func(t *testing.T) {
		g, d := shopGame(t)
		m := g.Town.Party.Members[1]
		m.Gold = 12
		shopAs(d, 2).keys("BP5") // STAFF, 10 GP: usable by a mage
		if m.Gold != 2 || g.Town.Message != "** JUST WHAT YOU NEEDED **" {
			t.Fatalf("a staff: gold %d, %q", m.Gold, g.Town.Message)
		}
		if d.keys("P5"); m.Gold != 2 || m.ItemCount != 1 || g.Town.Message != "** YOU CANNOT AFFORD IT **" {
			t.Errorf("unaffordable: gold %d, %q", m.Gold, g.Town.Message)
		}
	})
	t.Run("unusable", func(t *testing.T) {
		g, d := shopGame(t)
		m := g.Town.Party.Members[1]
		m.Gold = 100
		shopAs(d, 2).keys("BP1") // LONG SWORD, 25 GP: not for a mage
		if g.Town.ShopStep != engine.ShopBuyConfirm || m.Gold != 100 {
			t.Fatalf("no confirmation: %v %q", g.Town.ShopStep, g.Town.Message)
		}
		if d.keys("N"); m.Gold != 100 || m.ItemCount != 0 || g.Town.ShopStep != engine.ShopBuy {
			t.Errorf("no: gold %d", m.Gold)
		}
		d.keys("P1Y")
		if m.Gold != 75 || m.ItemCount != 1 || g.Town.Message != "** ITS YOUR MONEY **" || g.Town.ShopCatalog != itemLongSword {
			t.Errorf("yes: gold %d, %q, cursor %d", m.Gold, g.Town.Message, g.Town.ShopCatalog)
		}
	})
	t.Run("selling", func(t *testing.T) {
		g, d := shopGame(t)
		b := g.Town.Party.Members[2]
		shopAs(d, 3)
		if d.keys("S4"); g.Town.Message != "** WE DONT BUY CURSED ITEMS **" || b.ItemCount != 6 || g.Town.ShopStep != engine.ShopMain {
			t.Errorf("cursed: %q", g.Town.Message)
		}
		if d.keys("S3"); b.Gold != 601 || b.ItemCount != 5 || g.Stock(itemScrollHalito) != 28 {
			t.Errorf("unidentified: gold %d, stock %d (B6: 1 GP)", b.Gold, g.Stock(itemScrollHalito))
		}
		if d.keys("S1"); b.Gold != 606 || b.ItemCount != 4 || b.AC != 7 || g.Town.Message != "** ANYTHING ELSE, SIRE? **" {
			t.Errorf("equipped staff: gold %d, AC %d (B7 no question, #35 AC kept)", b.Gold, b.AC)
		}
	})
	t.Run("selling raises stock", func(t *testing.T) {
		g, d := shopGame(t)
		f := g.Town.Party.Members[0]
		f.Items[0] = engine.Possession{ItemIndex: itemPlateMail1, Identified: true}
		f.Items[1] = engine.Possession{ItemIndex: itemLongSword, Identified: true}
		f.ItemCount = 2
		shopAs(d, 1).keys("S1S1")
		if g.Stock(itemPlateMail1) != 1 || g.Stock(itemLongSword) != -1 || f.Gold != 2000+750+12 {
			t.Errorf("stock %d / %d, gold %d", g.Stock(itemPlateMail1), g.Stock(itemLongSword), f.Gold)
		}
	})
	t.Run("uncursing", func(t *testing.T) {
		g, d := shopGame(t)
		b := g.Town.Party.Members[2]
		shopAs(d, 3)
		if d.keys("U2"); g.Town.Message != "** THAT IS NOT A CURSED ITEM **" || b.ItemCount != 6 {
			t.Errorf("not cursed: %q", g.Town.Message)
		}
		b.Gold = 100
		if d.keys("U6"); g.Town.Message != "** YOU CANT AFFORD THE FEE **" || b.ItemCount != 6 || b.Gold != 100 {
			t.Errorf("fee: %q", g.Town.Message)
		}
		b.Gold = 750
		if d.keys("U6"); b.Gold != 0 || b.ItemCount != 5 || g.Stock(itemLeatherM1) != 2 {
			t.Errorf("uncursed: gold %d, items %d, stock %d", b.Gold, b.ItemCount, g.Stock(itemLeatherM1))
		}
	})
	t.Run("identifying", func(t *testing.T) {
		g, d := shopGame(t)
		b := g.Town.Party.Members[2]
		shopAs(d, 3)
		if d.keys("I2"); g.Town.Message != "** THAT HAS BEEN IDENTIFIED **" {
			t.Errorf("identified: %q", g.Town.Message)
		}
		b.Gold = 49
		if d.keys("I5"); g.Town.Message != "** YOU CANT AFFORD THE FEE **" || b.Items[4].Identified {
			t.Errorf("fee: %q", g.Town.Message)
		}
		b.Gold = 50
		if d.keys("I5"); b.Gold != 0 || !b.Items[4].Identified || g.Town.Message != "** ANYTHING ELSE, SIRE? **" {
			t.Errorf("identified for 50: gold %d", b.Gold)
		}
	})
	t.Run("stock is not saved", func(t *testing.T) { // finding #12
		g, d := shopGame(t)
		shopAs(d, 1).keys("BP1")
		raw := snapshotSave(t, g)
		if strings.Contains(raw, "stock") {
			t.Error("the save holds stock")
		}
	})
}

// snapshotSave is the save file the game would write.
func snapshotSave(t *testing.T, g *engine.GameState) string {
	t.Helper()
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	dir, err := engine.SaveDir()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// Stock is the game's own (M8-1): buying a limited item out in one game
// leaves the scenario, and another game on the same scenario, as they were;
// that game still lists and sells the item.
func TestShopStockIsTheGames(t *testing.T) {
	g, d := shopGame(t)
	g.Town.Party.Members[0].Gold = 1 << 20
	other := engine.New(g.Scenario)
	other.Title, other.Phase = nil, engine.PhaseTown
	for _, c := range g.Town.Party.Members {
		cp := *c
		other.Town.Roster.Add(&cp)
		other.Town.Party.Members = append(other.Town.Party.Members, &cp)
	}
	listed := func(g *engine.GameState, d *driver, item int) bool {
		d.keys("S")
		for i := 0; i < 8; i++ {
			for _, idx := range g.Town.ShopPage {
				if idx == item {
					return true
				}
			}
			d.keys("F")
		}
		return false
	}
	shopAs(d, 1).keys("B")
	for g.Stock(itemShortSword1) > 0 {
		if !listed(g, d, itemShortSword1) {
			t.Fatal("SHORT SWORD+1 not listed")
		}
		for slot, idx := range g.Town.ShopPage {
			if idx == itemShortSword1 {
				d.keys("P" + fmt.Sprint(slot+1))
				break
			}
		}
	}
	if listed(g, d, itemShortSword1) {
		t.Error("sold out, still listed")
	}
	if g.Scenario.Items[itemShortSword1].Stock != 2 {
		t.Errorf("the scenario's stock changed to %d", g.Scenario.Items[itemShortSword1].Stock)
	}
	od := newDriver(t, other)
	shopAs(od, 1).keys("B")
	if other.Stock(itemShortSword1) != 2 || !listed(other, od, itemShortSword1) {
		t.Errorf("the other game: stock %d", other.Stock(itemShortSword1))
	}
}
