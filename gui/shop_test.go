package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// Boltac's Trading Post in the GUI (M8-3): ShopAction runs the
// controller's operations, a completed transaction is saved at once and
// nothing else writes, a failed save is reported and never passed off as
// saved, and nothing draws a random number.

const (
	itemLongSword = 1  // 25 GP, unlimited, not for mages
	itemStaff     = 5  // 10 GP, unlimited, anyone
	itemHelm      = 14 // 100 GP, unlimited, unknown name HELM
	itemCursed    = 29 // LONG SWORD -1, a cursed type: never in the catalog
	chainMail     = 11 // 90 GP, AC 3
	mageSlot      = 4  // partySession's mage
)

// shopSession is partySession's party, with gold, at Boltac's; the save
// holds what the game holds.
func shopSession(t *testing.T) (*Session, string) {
	t.Helper()
	rand.Seed(7)
	s, dir := partySession(t, false)
	for _, m := range s.ctrl.Game.Town.Party.Members {
		m.Gold = 1000
	}
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	walkTo(t, s, "trading")
	return s, dir
}

// shopAct calls ShopAction, failing if it drew a random number, and
// reports whether it wrote to the save folder.
func shopAct(t *testing.T, s *Session, dir string, r ShopRequest) (View, bool) {
	t.Helper()
	before := folder(t, dir)
	rand.Seed(13)
	v := s.ShopAction(r)
	next := rand.Int63()
	rand.Seed(13)
	if next != rand.Int63() {
		t.Fatalf("%+v drew a random number", r)
	}
	return v, !maps.Equal(before, folder(t, dir))
}

// done fails unless r completed a transaction, saved it, and said so.
func done(t *testing.T, s *Session, dir string, r ShopRequest, action, item string, gold int) View {
	t.Helper()
	v, wrote := shopAct(t, s, dir, r)
	if v.Error != "" || !wrote || v.Unsaved != "" {
		t.Fatalf("%+v: error %q wrote %v unsaved %q", r, v.Error, wrote, v.Unsaved)
	}
	want := ShopReceipt{Action: action, Customer: v.Shop.View.Name, Item: item, Gold: gold, Saved: true, Durable: true}
	if v.Shop.Last == nil || *v.Shop.Last != want {
		t.Fatalf("%+v: receipt %+v, want %+v", r, v.Shop.Last, want)
	}
	if !strings.HasSuffix(v.Notice, " Saved.") {
		t.Fatalf("%+v: notice %q", r, v.Notice)
	}
	mustMatchDisk(t, s, dir, r.Action)
	return v
}

// refused fails unless r was refused with why, changing no character and
// writing nothing (Boltac's own refusals send him back to his menu, as in
// the game).
func refused(t *testing.T, s *Session, dir string, r ShopRequest, why string) View {
	t.Helper()
	roster := rosterBytes(s.ctrl.Game)
	v, wrote := shopAct(t, s, dir, r)
	if v.Error != why || wrote || !bytes.Equal(roster, rosterBytes(s.ctrl.Game)) {
		t.Fatalf("%+v: error %q (want %q), wrote %v", r, v.Error, why, wrote)
	}
	if v.Shop != nil && v.Shop.Last != nil {
		t.Fatalf("%+v: a receipt for a refusal: %+v", r, v.Shop.Last)
	}
	return v
}

// unchanged fails unless r succeeded without changing any character, and
// wrote nothing.
func unchanged(t *testing.T, s *Session, dir string, r ShopRequest) View {
	t.Helper()
	roster := rosterBytes(s.ctrl.Game)
	v, wrote := shopAct(t, s, dir, r)
	if v.Error != "" || wrote || v.Shop.Last != nil || !bytes.Equal(roster, rosterBytes(s.ctrl.Game)) {
		t.Fatalf("%+v: error %q wrote %v receipt %+v", r, v.Error, wrote, v.Shop.Last)
	}
	return v
}

func offer(t *testing.T, v View, item int) app.ShopOffer {
	t.Helper()
	i := slices.IndexFunc(v.Shop.View.Catalog, func(o app.ShopOffer) bool { return o.Item == item })
	if i < 0 {
		t.Fatalf("item %d not in the catalog", item)
	}
	return v.Shop.View.Catalog[i]
}

func TestShopTransactions(t *testing.T) {
	s, dir := shopSession(t)
	g := s.ctrl.Game
	mage := g.Town.Party.Members[mageSlot]
	before := folder(t, dir)
	v := s.State()
	if !maps.Equal(before, folder(t, dir)) {
		t.Fatal("looking wrote to the save folder")
	}
	if v.Shop == nil || v.Shop.View.Step != "who" || v.Shop.View.Customer != -1 || v.Location != "trading" {
		t.Fatalf("at Boltac's: %+v at %q", v.Shop, v.Location)
	}

	refused(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemStaff}, errNoCustomer.Error())
	refused(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: 6}, "No party member in that slot.")
	v = unchanged(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: mageSlot})
	if sv := v.Shop.View; sv.Step != "main" || sv.Customer != mageSlot || sv.Name != mage.Name || sv.Gold != 1000 || sv.Room != 8 {
		t.Fatalf("the mage enters: %+v", sv)
	}
	if v = unchanged(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: mageSlot}); v.Shop.View.Customer != mageSlot || v.Shop.View.Step != "main" {
		t.Fatalf("the mage again: %+v", v.Shop.View)
	}
	refused(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: 6}, "No party member in that slot.")
	if v = s.State(); v.Shop.View.Customer != mageSlot {
		t.Fatalf("a bad slot sent the mage away: %+v", v.Shop.View)
	}

	// A purchase, saved; what Boltac cannot sell or the mage cannot pay
	// is refused as the offer says.
	done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemStaff}, "buy", "STAFF", -10)
	v = s.State()
	if o := offer(t, v, shortSwordPlus1); o.Refusal != "** YOU CANNOT AFFORD IT **" {
		t.Fatalf("offer %+v", o)
	}
	refused(t, s, dir, ShopRequest{Action: ShopActBuy, Item: shortSwordPlus1}, "** YOU CANNOT AFFORD IT **")
	refused(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemCursed}, errNoSuchOffer.Error())
	refused(t, s, dir, ShopRequest{Action: ShopActBuy, Item: 0}, errNoSuchOffer.Error())
	refused(t, s, dir, ShopRequest{Action: "haggle"}, errorText(app.ErrInvalidChoice))

	// An unusable item: Boltac asks; no is nothing, yes is a purchase.
	if o := offer(t, v, itemLongSword); o.Usable || o.Refusal != "" {
		t.Fatalf("offer %+v", o)
	}
	v = unchanged(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemLongSword})
	if v.Shop.View.Step != "confirm" || v.Shop.View.Confirm != itemLongSword || v.Return.Note == "" {
		t.Fatalf("Boltac asks: %+v, return %+v", v.Shop.View, v.Return)
	}
	refused(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemStaff}, errAnswerBoltac.Error())
	refused(t, s, dir, ShopRequest{Action: ShopActSell, N: 1}, errAnswerBoltac.Error())
	refused(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: 0}, errAnswerBoltac.Error())
	if v = unchanged(t, s, dir, ShopRequest{Action: ShopActConfirm, Yes: false}); v.Shop.View.Step != "buy" {
		t.Fatalf("after no: %q", v.Shop.View.Step)
	}
	refused(t, s, dir, ShopRequest{Action: ShopActConfirm, Yes: true}, errNotAsked.Error())
	unchanged(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemLongSword})
	v = done(t, s, dir, ShopRequest{Action: ShopActConfirm, Yes: true}, "buy", "LONG SWORD", -25)
	if v.Shop.View.Message != "** ITS YOUR MONEY **" || mage.ItemCount != 2 || mage.Gold != 965 {
		t.Fatalf("after yes: %q, %d items, %d GP", v.Shop.View.Message, mage.ItemCount, mage.Gold)
	}

	// Stock goes down with a purchase and up with a sale.
	stock := g.Stock(scrollHalito)
	done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: scrollHalito}, "buy", "SCROLL/HALITO", -500)
	if v = s.State(); offer(t, v, scrollHalito).Stock != stock-1 || g.Stock(scrollHalito) != stock-1 {
		t.Fatalf("stock %d, was %d", g.Stock(scrollHalito), stock)
	}
	done(t, s, dir, ShopRequest{Action: ShopActSell, N: 3}, "sell", "SCROLL/HALITO", 250)
	if g.Stock(scrollHalito) != stock {
		t.Fatalf("stock %d after the sale, want %d", g.Stock(scrollHalito), stock)
	}
	refused(t, s, dir, ShopRequest{Action: ShopActSell, N: 3}, errNoSuchItem.Error())
	refused(t, s, dir, ShopRequest{Action: ShopActSell, N: 0}, errNoSuchItem.Error())

	// Identifying and uncursing (a curse is set by hand: in play none
	// sticks yet, finding #34).
	mage.AddItem(itemHelm, false)
	mage.AddItem(itemStaff, true)
	mage.Items[3].Cursed = true
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	v = s.State()
	if it := v.Shop.View.Items[2]; it.Name != "HELM" || it.Identified || it.Price != 1 || it.Fee != 50 || it.IdentifyRefusal != "" {
		t.Fatalf("the unknown helm: %+v", it)
	}
	done(t, s, dir, ShopRequest{Action: ShopActIdentify, N: 3}, "identify", g.Scenario.Items[itemHelm].Name, -50)
	refused(t, s, dir, ShopRequest{Action: ShopActIdentify, N: 3}, "** THAT HAS BEEN IDENTIFIED **")
	refused(t, s, dir, ShopRequest{Action: ShopActUncurse, N: 3}, "** THAT IS NOT A CURSED ITEM **")
	refused(t, s, dir, ShopRequest{Action: ShopActSell, N: 4}, "** WE DONT BUY CURSED ITEMS **")
	done(t, s, dir, ShopRequest{Action: ShopActUncurse, N: 4}, "uncurse", "STAFF", -5)
	if mage.ItemCount != 3 || mage.Gold != 1000-10-25-500+250-50-5 {
		t.Fatalf("%d items, %d GP", mage.ItemCount, mage.Gold)
	}
	poor := mage.Gold
	mage.Gold = 0
	refused(t, s, dir, ShopRequest{Action: ShopActIdentify, N: 3}, "** THAT HAS BEEN IDENTIFIED **")
	mage.AddItem(itemHelm, false)
	refused(t, s, dir, ShopRequest{Action: ShopActIdentify, N: 4}, "** YOU CANT AFFORD THE FEE **")
	mage.Gold = poor
	s.ctrl.Save()

	// Room for eight items, no more.
	for mage.ItemCount < 8 {
		done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemStaff}, "buy", "STAFF", -10)
	}
	v = s.State()
	if v.Shop.View.Room != 0 || offer(t, v, itemStaff).Refusal != "** YOU CANT CARRY ANYTHING MORE **" {
		t.Fatalf("full: room %d, %+v", v.Shop.View.Room, offer(t, v, itemStaff))
	}
	refused(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemStaff}, "** YOU CANT CARRY ANYTHING MORE **")

	// Another customer.
	// Another customer, straight from the counter (the mage leaves first),
	// and back by "WHO WILL ENTER".
	fighter := g.Town.Party.Members[0]
	if v = unchanged(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: 0}); v.Shop.View.Name != fighter.Name || v.Shop.View.Customer != 0 || v.Shop.View.Step != "main" {
		t.Fatalf("the fighter in the mage's place: %+v", v.Shop.View)
	}
	if v = unchanged(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: mageSlot}); v.Shop.View.Customer != mageSlot || v.Shop.View.Room != 0 {
		t.Fatalf("the mage back: %+v", v.Shop.View)
	}
	if v = unchanged(t, s, dir, ShopRequest{Action: ShopActLeaveCustomer}); v.Shop.View.Step != "who" || v.Shop.View.Customer != -1 {
		t.Fatalf("after leaving: %+v", v.Shop.View)
	}
	refused(t, s, dir, ShopRequest{Action: ShopActLeaveCustomer}, errNoCustomer.Error())
	if v = unchanged(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: 0}); v.Shop.View.Name != fighter.Name || v.Shop.View.Room != 8 {
		t.Fatalf("the fighter enters: %+v", v.Shop.View)
	}
	if o := offer(t, v, itemLongSword); !o.Usable {
		t.Fatalf("offer %+v", o)
	}
	v = done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemLongSword}, "buy", "LONG SWORD", -25)
	if v.Shop.View.Step != "buy" || fighter.ItemCount != 1 || mage.ItemCount != 8 {
		t.Fatalf("the fighter's purchase: %+v", v.Shop.View)
	}

	// The receipt lasts until the next shop action, and a new visit
	// starts without one.
	if v = s.State(); v.Shop.Last == nil {
		t.Fatal("the receipt is gone")
	}
	walkTo(t, s, "castle")
	if v = walkTo(t, s, "trading"); v.Shop.Last != nil || v.Shop.View.Step != "who" {
		t.Fatalf("a new visit: %+v", v.Shop)
	}
}

// Shop actions are refused anywhere but Boltac's, practice and
// expeditions included, changing nothing.
func TestShopActionsOnlyAtBoltacs(t *testing.T) {
	s, dir := shopSession(t)
	s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 0})
	away := func(when, why string) {
		t.Helper()
		before := s.State()
		if before.Shop != nil {
			t.Fatalf("%s: a shop view away from Boltac's", when)
		}
		for _, r := range []ShopRequest{
			{Action: ShopActEnter, Slot: 1}, {Action: ShopActLeaveCustomer}, {Action: ShopActBuy, Item: itemStaff},
			{Action: ShopActConfirm, Yes: true}, {Action: ShopActSell, N: 1}, {Action: ShopActUncurse, N: 1}, {Action: ShopActIdentify, N: 1},
		} {
			if v := refused(t, s, dir, r, why); !sameView(v, before) {
				t.Fatalf("%s: %+v changed the view", when, r)
			}
		}
	}
	notHere := notAt(engine.Trading).Error()
	for _, place := range []string{"castle", "tavern", "edge", "training", "settings"} {
		walkTo(t, s, place)
		away(place, notHere)
	}
	walkTo(t, s, "training")
	mustOK(t, s.StartCreation("NEWCOMER"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	away("creating", errCreating.Error())
	s.ReturnToCastle()

	walkTo(t, s, "edge")
	s.PracticeExpedition()
	away("practice expedition", errPracticeOut.Error())
	s.AbandonPractice()
	s.PracticeEncounter(false)
	away("practice encounter", errPracticeOut.Error())
	s.AbandonPractice()
	s.RealExpedition()
	away("real expedition", errExpeditionOut.Error())
}

// A transaction whose save fails has happened in the window and stays; it
// is reported as not saved, the file keeps the last save (a restart finds
// the party as it was), refusals write nothing meanwhile, and the next
// transaction saves both.
func TestShopFailedSave(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	s, dir := shopSession(t)
	mage := s.ctrl.Game.Town.Party.Members[mageSlot]
	s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: mageSlot})
	file := filepath.Join(dir, "roster1.json")
	saved, _ := os.ReadFile(file)

	os.Chmod(dir, 0o555)
	v := s.ShopAction(ShopRequest{Action: ShopActBuy, Item: itemStaff})
	os.Chmod(dir, 0o755)
	if !strings.Contains(v.Error, "Could not save") || v.Unsaved == "" {
		t.Fatalf("error %q unsaved %q", v.Error, v.Unsaved)
	}
	if l := v.Shop.Last; l == nil || l.Saved || l.Durable || l.Item != "STAFF" || l.Gold != -10 {
		t.Fatalf("receipt %+v", v.Shop.Last)
	}
	if !strings.Contains(v.Notice, "NOT saved") || strings.Contains(v.Notice, "Saved.") {
		t.Fatalf("notice %q", v.Notice)
	}
	if mage.Gold != 990 || mage.ItemCount != 1 {
		t.Fatalf("in the window: %d GP, %d items", mage.Gold, mage.ItemCount)
	}
	if now, _ := os.ReadFile(file); !bytes.Equal(now, saved) {
		t.Fatal("the file changed although the save failed")
	}
	r := reopened(t, s, dir)
	if m := r.ctrl.Game.Town.Party.Members[mageSlot]; m.Gold != 1000 || m.ItemCount != 0 {
		t.Fatalf("a restart now finds %d GP and %d items, want the saved 1000 and 0", m.Gold, m.ItemCount)
	}

	v = refused(t, s, dir, ShopRequest{Action: ShopActBuy, Item: shortSwordPlus1}, "** YOU CANNOT AFFORD IT **")
	if v.Unsaved == "" || s.State().Unsaved == "" {
		t.Fatal("the failed save is no longer reported")
	}
	unchanged(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemLongSword})
	unchanged(t, s, dir, ShopRequest{Action: ShopActConfirm, Yes: false})

	done(t, s, dir, ShopRequest{Action: ShopActSell, N: 1}, "sell", "STAFF", 5)
	r = reopened(t, s, dir)
	if m := r.ctrl.Game.Town.Party.Members[mageSlot]; m.Gold != 995 || m.ItemCount != 0 {
		t.Fatalf("after the next save a restart finds %d GP and %d items, want 995 and 0", m.Gold, m.ItemCount)
	}
}

// After transactions a restart finds every customer exactly as the window
// had them, once; the save keeps its format; a reload starts Boltac's
// stock again from the scenario and keeps what was bought.
func TestShopRestartAndReload(t *testing.T) {
	s, dir := shopSession(t)
	g := s.ctrl.Game
	keys := func() []string {
		raw, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
		var top map[string]json.RawMessage
		var st struct{ Roster []map[string]json.RawMessage }
		json.Unmarshal(raw, &top)
		json.Unmarshal(raw, &st)
		out := slices.Sorted(maps.Keys(top))
		for _, ch := range st.Roster {
			out = append(out, slices.Sorted(maps.Keys(ch))...)
		}
		return out
	}
	format := keys()

	stock := g.Stock(scrollHalito)
	for _, slot := range []int{0, mageSlot} {
		s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: slot})
		done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: scrollHalito}, "buy", "SCROLL/HALITO", -500)
		done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemStaff}, "buy", "STAFF", -10)
		done(t, s, dir, ShopRequest{Action: ShopActSell, N: 2}, "sell", "STAFF", 5)
		s.ShopAction(ShopRequest{Action: ShopActLeaveCustomer})
	}
	if !slices.Equal(keys(), format) {
		t.Fatalf("the save's format changed:\n%v\nwas\n%v", keys(), format)
	}
	want := partyStanding(g)
	for _, m := range want {
		if (m.Name == "AGUI" || m.Name == "EGUI") != (m.ItemCount == 1 && m.Gold == 495) || m.ItemCount > 1 {
			t.Fatalf("%s: %d items, %d GP", m.Name, m.ItemCount, m.Gold)
		}
	}
	r := restart(t, s, dir, "after shopping")
	if rg := r.ctrl.Game; rg.Stock(scrollHalito) != stock {
		t.Fatalf("a restart's stock %d, want the scenario's %d", rg.Stock(scrollHalito), stock)
	}

	walkTo(t, s, "settings")
	if v := s.ReloadRoster(); v.Error != "" {
		t.Fatal(v.Error)
	}
	g = s.ctrl.Game
	if got := partyStanding(g); !slices.Equal(got, want) || g.Stock(scrollHalito) != stock {
		t.Fatalf("after a reload: %+v, stock %d", got, g.Stock(scrollHalito))
	}
}

// ShopAction is the controller's operation and nothing else: random
// requests leave the window's game exactly as the same operations leave a
// twin game loaded from the same save (and app's tests hold the operations
// to the terminal's keys). A request writes the save exactly when it
// changes a character, and the save then holds the game.
func TestShopActionsAreTheControllers(t *testing.T) {
	asked := 0 // requests that left Boltac asking before buying a worn item
	for seed := int64(1); seed <= 40; seed++ {
		s, dir := shopSession(t)
		rng := rand.New(rand.NewSource(seed))
		for _, m := range s.ctrl.Game.Town.Party.Members {
			m.Gold = rng.Intn(1500)
			for n := rng.Intn(4); n > 0; n-- {
				m.AddItem(1+rng.Intn(30), rng.Intn(2) == 0)
				m.Items[m.ItemCount-1].Cursed = rng.Intn(4) == 0
				m.Items[m.ItemCount-1].Equipped = rng.Intn(3) == 0
			}
			m.RecalcAC(s.scenario.Items)
		}
		if err := s.ctrl.Save(); err != nil {
			t.Fatal(err)
		}
		tg := engine.New(s.scenario)
		tg.Title, tg.Phase = nil, engine.PhaseTown
		if err := tg.Load(); err != nil {
			t.Fatal(err)
		}
		twin := app.New(tg)
		if err := twin.GoTo(engine.Trading); err != nil {
			t.Fatal(err)
		}
		state := func(g *engine.GameState) string {
			town := g.Town
			stock := make([]int, len(g.Scenario.Items))
			for i := range stock {
				stock[i] = g.Stock(i)
			}
			// Known spells are rebuilt by every load (finding #32), so the
			// twin's differ from the start; they are left out.
			var roster []map[string]any
			raw, _ := json.Marshal(town.Roster.Characters)
			json.Unmarshal(raw, &roster)
			for _, ch := range roster {
				delete(ch, "spell_known")
			}
			raw, _ = json.Marshal(struct {
				Roster   []map[string]any
				Stock    []int
				Step     engine.ShopStep
				Customer int
				Message  string
				Mode     engine.InputMode
				Location engine.TownLocation
			}{roster, stock, town.ShopStep, app.PartySlot(town, town.ShopChar), town.Message, town.InputMode, town.Location})
			return string(raw)
		}
		actions := []string{ShopActEnter, ShopActLeaveCustomer, ShopActBuy, ShopActBuy, ShopActBuy, ShopActConfirm, ShopActConfirmSale, ShopActSell, ShopActSell, ShopActUncurse, ShopActIdentify}
		for step := 0; step < 120; step++ {
			r := ShopRequest{Action: actions[rng.Intn(len(actions))], Slot: rng.Intn(7), N: rng.Intn(10), Yes: rng.Intn(2) == 0, Item: rng.Intn(40)}
			if v := s.State(); r.Action == ShopActBuy && len(v.Shop.View.Catalog) > 0 && rng.Intn(4) > 0 {
				r.Item = v.Shop.View.Catalog[rng.Intn(len(v.Shop.View.Catalog))].Item
			}
			roster := rosterBytes(s.ctrl.Game)
			v, wrote := shopAct(t, s, dir, r)
			switch r.Action {
			case ShopActEnter: // in place of the customer, as the GUI offers it
				tv, _ := twin.ShopView()
				members := tg.Town.Party.Members
				valid := r.Slot < len(members) && members[r.Slot] != nil
				asking := tv.Step == "confirm" || tv.Step == "confirmSell"
				if tv.Step != "who" && !asking && r.Slot != tv.Customer && valid {
					twin.ShopLeaveCustomer()
				}
				if r.Slot != tv.Customer && !asking {
					twin.ShopEnter(r.Slot)
				}
			case ShopActLeaveCustomer:
				twin.ShopLeaveCustomer()
			case ShopActBuy:
				twin.Buy(r.Item)
			case ShopActConfirm:
				twin.ConfirmUnusable(r.Yes)
			case ShopActConfirmSale:
				twin.ConfirmSale(r.Yes)
			case ShopActSell:
				twin.Sell(r.N)
			case ShopActUncurse:
				twin.Uncurse(r.N)
			case ShopActIdentify:
				twin.ShopIdentify(r.N)
			}
			if got, want := state(s.ctrl.Game), state(tg); got != want {
				t.Fatalf("seed %d step %d %+v: the window's game\n%s\nthe controller's\n%s", seed, step, r, got, want)
			}
			changed := !bytes.Equal(roster, rosterBytes(s.ctrl.Game))
			if wrote != changed || (v.Shop.Last != nil) != changed || v.Unsaved != "" {
				t.Fatalf("seed %d step %d %+v: changed %v, wrote %v, receipt %+v", seed, step, r, changed, wrote, v.Shop.Last)
			}
			mustMatchDisk(t, s, dir, r.Action)
			if v.Shop.View.Step == "confirmSell" {
				asked++
			}
		}
		s.shutdown()
	}
	if asked == 0 {
		t.Fatal("never asked about selling a worn item")
	}
}

// A save that replaced the file but could not sync the folder is saved,
// perhaps not durably, and the receipt and notice say so.
func TestShopSaveNotDurable(t *testing.T) {
	s, dir := shopSession(t)
	s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: mageSlot})
	saveGame = func(c *app.Controller) error {
		if err := c.Save(); err != nil {
			return err
		}
		return &engine.NotDurableError{Err: errors.New("no sync")}
	}
	defer func() { saveGame = (*app.Controller).Save }()
	v := s.ShopAction(ShopRequest{Action: ShopActBuy, Item: itemStaff})
	if l := v.Shop.Last; v.Error != "" || v.Unsaved != "" || l == nil || !l.Saved || l.Durable || !strings.Contains(v.Warning, "no sync") {
		t.Fatalf("error %q unsaved %q warning %q receipt %+v", v.Error, v.Unsaved, v.Warning, v.Shop.Last)
	}
	if !strings.HasSuffix(v.Notice, "Saved, but perhaps not durably.") {
		t.Fatalf("notice %q", v.Notice)
	}
	mustMatchDisk(t, s, dir, "not durable")
}

// shopCase is what the Trading Post screen reads from a View.
type shopCase struct {
	Case     string         `json:"case"`
	Location string         `json:"location"`
	Shop     *ShopPanel     `json:"shop"`
	PartyBar []PartyBarSlot `json:"partyBar"`
	FrontRow int            `json:"frontRow"`
	Return   *ReturnView    `json:"return"`
	Error    string         `json:"error"`
	Notice   string         `json:"notice"`
	Unsaved  string         `json:"unsaved"`
	Warning  string         `json:"warning"`
}

func shopCaseOf(name string, v View) shopCase {
	return shopCase{name, v.Location, v.Shop, v.PartyBar, v.FrontRow, v.Return, v.Error, v.Notice, v.Unsaved, v.Warning}
}

// TestShopViewFixture records real views at Boltac's, one for each state
// the Trading Post screen draws, for the page's tests
// (frontend/src/lib/shopViews.test.js). Run with ISTARY_UPDATE_FIXTURES=1
// to rewrite it.
func TestShopViewFixture(t *testing.T) {
	var cases []shopCase
	s, dir := shopSession(t)
	g := s.ctrl.Game
	mage := g.Town.Party.Members[mageSlot]
	mage.AddItem(itemLongSword, true)
	mage.Items[0].Equipped = true
	mage.AddItem(itemHelm, false)
	mage.AddItem(itemStaff, true)
	mage.Items[2].Cursed = true
	mage.AddItem(scrollHalito, false)
	mage.Items[3].Cursed = true // unidentified and cursed
	g.Town.Party.Members[2].Gold = 5
	priest := g.Town.Party.Members[3]
	priest.Gold = 0
	priest.AddItem(itemHelm, false)
	s.ctrl.Save()
	add := func(name string, v View) { cases = append(cases, shopCaseOf(name, v)) }

	add("who", s.State())
	add("mage at the counter", s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: mageSlot}))
	add("bought, saved", s.ShopAction(ShopRequest{Action: ShopActBuy, Item: itemStaff}))
	add("refused: cannot afford", s.ShopAction(ShopRequest{Action: ShopActBuy, Item: shortSwordPlus1}))
	add("Boltac asks: unusable", s.ShopAction(ShopRequest{Action: ShopActBuy, Item: itemLongSword}))
	add("bought anyway", s.ShopAction(ShopRequest{Action: ShopActConfirm, Yes: true}))
	add("identified", s.ShopAction(ShopRequest{Action: ShopActIdentify, N: 2}))
	add("uncursed", s.ShopAction(ShopRequest{Action: ShopActUncurse, N: 3}))
	add("Boltac asks: equipped", s.ShopAction(ShopRequest{Action: ShopActSell, N: 1}))
	add("sold", s.ShopAction(ShopRequest{Action: ShopActConfirmSale, Yes: true}))
	for mage.ItemCount < 8 {
		if v := s.ShopAction(ShopRequest{Action: ShopActBuy, Item: itemStaff}); v.Error != "" {
			t.Fatal(v.Error)
		}
	}
	add("no room", s.State())
	add("thief: little gold, nothing to sell", s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 2}))
	add("priest: no gold for a fee", s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 3}))

	s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 0})
	if os.Geteuid() != 0 {
		os.Chmod(dir, 0o555)
		v := s.ShopAction(ShopRequest{Action: ShopActBuy, Item: itemStaff})
		os.Chmod(dir, 0o755)
		// The message names the folder and a temporary file, which change
		// every run.
		tidy := func(msg string) string {
			msg = strings.ReplaceAll(msg, "/private"+dir, "<saves>")
			msg = strings.ReplaceAll(msg, dir, "<saves>")
			return regexp.MustCompile(`tmp-[0-9a-f]+`).ReplaceAllString(msg, "tmp-<random>")
		}
		v.Error, v.Unsaved = tidy(v.Error), tidy(v.Unsaved)
		add("bought, NOT saved", v)
	}
	saveGame = func(c *app.Controller) error {
		if err := c.Save(); err != nil {
			return err
		}
		return &engine.NotDurableError{Err: errors.New("no sync")}
	}
	add("bought, not durable", s.ShopAction(ShopRequest{Action: ShopActBuy, Item: itemStaff}))
	saveGame = (*app.Controller).Save
	s.shutdown()

	got, _ := json.MarshalIndent(cases, "", " ")
	got = append(got, '\n')
	path := filepath.Join("frontend", "src", "lib", "testdata", "shop-views.json")
	if os.Getenv("ISTARY_UPDATE_FIXTURES") != "" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with ISTARY_UPDATE_FIXTURES=1)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("shop-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}

// Selling worn armour at Boltac's changes the AC the party bar shows and
// the save holds (finding #35); a stale AC in a save is worked out again
// when the window loads it.
func TestShopSaleUpdatesAC(t *testing.T) {
	s, dir := shopSession(t)
	g := s.ctrl.Game
	fighter := g.Town.Party.Members[0]
	fighter.AddItem(chainMail, true)
	fighter.Items[0].Equipped = true
	fighter.RecalcAC(g.Scenario.Items)
	worn := fighter.AC
	s.ctrl.Save()
	s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 0})
	if v := unchanged(t, s, dir, ShopRequest{Action: ShopActSell, N: 1}); v.Shop.View.Step != "confirmSell" {
		t.Fatalf("a worn item sold without a question: %q", v.Shop.View.Step)
	}
	v := done(t, s, dir, ShopRequest{Action: ShopActConfirmSale, Yes: true}, "sell", "CHAIN MAIL", 45)
	if v.PartyBar[0].AC != 10 || worn == 10 {
		t.Fatalf("after selling worn chain mail: party bar AC %d (was %d)", v.PartyBar[0].AC, worn)
	}
	if ac := onDisk(t, dir).Roster[rosterIndex(g, fighter)].AC; ac != 10 {
		t.Fatalf("saved AC %d", ac)
	}

	editSave(t, dir, func(st map[string]any) {
		st["roster"].([]any)[0].(map[string]any)["AC"] = 2
	})
	walkTo(t, s, "settings")
	if v := s.ReloadRoster(); v.Error != "" || v.PartyBar[0].AC != 10 {
		t.Fatalf("reloaded with a stale AC: %q, AC %d", v.Error, v.PartyBar[0].AC)
	}
}

// Selling a worn item asks first (B7): the question writes nothing, no
// writes nothing, the other actions wait for the answer, and yes is a
// sale, saved, with the AC worked out again.
func TestShopSellingAWornItemAsks(t *testing.T) {
	s, dir := shopSession(t)
	g := s.ctrl.Game
	fighter := g.Town.Party.Members[0]
	fighter.AddItem(chainMail, true)
	fighter.AddItem(itemStaff, true)
	fighter.Items[0].Equipped = true
	fighter.RecalcAC(g.Scenario.Items)
	s.ctrl.Save()
	v := unchanged(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: 0})
	if !v.Shop.View.Items[0].SellAsks || v.Shop.View.Items[1].SellAsks {
		t.Fatalf("asks: %+v", v.Shop.View.Items)
	}
	refused(t, s, dir, ShopRequest{Action: ShopActConfirmSale, Yes: true}, errNotAsked.Error())
	v = unchanged(t, s, dir, ShopRequest{Action: ShopActSell, N: 1})
	if v.Shop.View.Step != "confirmSell" || v.Shop.View.SellConfirm != 1 || v.Return.Note == "" {
		t.Fatalf("Boltac asks: %+v, return %+v", v.Shop.View, v.Return)
	}
	for _, r := range []ShopRequest{{Action: ShopActSell, N: 2}, {Action: ShopActBuy, Item: itemStaff}, {Action: ShopActEnter, Slot: 1}, {Action: ShopActIdentify, N: 1}} {
		refused(t, s, dir, r, errAnswerBoltac.Error())
	}
	refused(t, s, dir, ShopRequest{Action: ShopActConfirm, Yes: true}, errNotAsked.Error())
	if v = unchanged(t, s, dir, ShopRequest{Action: ShopActConfirmSale, Yes: false}); v.Shop.View.Step != "sell" || !fighter.Items[0].Equipped {
		t.Fatalf("after no: %q", v.Shop.View.Step)
	}
	unchanged(t, s, dir, ShopRequest{Action: ShopActSell, N: 1})
	v = done(t, s, dir, ShopRequest{Action: ShopActConfirmSale, Yes: true}, "sell", "CHAIN MAIL", 45)
	if fighter.ItemCount != 1 || fighter.AC != 10 || v.PartyBar[0].AC != 10 {
		t.Fatalf("after yes: %d items, AC %d", fighter.ItemCount, fighter.AC)
	}
}
