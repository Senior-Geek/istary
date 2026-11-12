package main

// UX-2 at Boltac's through the Session, as the Trading Post page drives
// it: the party bar chooses the customer, the Selected Character panel is
// the customer's roster entry, and equipping beside the counter is the
// character sheet's EquipAction.

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// panelOf is the Selected Character panel's character, which must be the
// customer.
func panelOf(t *testing.T, v View) *CharacterView {
	t.Helper()
	ch := customerOf(v)
	if ch == nil {
		t.Fatalf("no customer (step %q)", v.Shop.View.Step)
	}
	sv := v.Shop.View
	if ch.Name != sv.Name || ch.Gold != sv.Gold || 8-len(ch.Items) != sv.Room || ch.PartySlot != sv.Customer+1 {
		t.Fatalf("the panel %s %d GP %d items, Boltac %s %d GP room %d", ch.Name, ch.Gold, len(ch.Items), sv.Name, sv.Gold, sv.Room)
	}
	return ch
}

// Clicking another party member serves them instead, at once: Boltac's
// catalog, gold and room are theirs, and so is the panel.
func TestSwitchingCustomers(t *testing.T) {
	s, _ := shopSession(t)
	g := s.ctrl.Game
	g.Town.Party.Members[2].Gold = 7
	v := s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 0})
	if ch := panelOf(t, v); ch.Name != "AGUI" {
		t.Fatalf("customer %s", ch.Name)
	}
	v = s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 2})
	ch := panelOf(t, v)
	if ch.Name != "CGUI" || ch.Gold != 7 {
		t.Fatalf("after switching: %s %d GP", ch.Name, ch.Gold)
	}
	for _, o := range v.Shop.View.Catalog {
		if o.Price > 7 && o.Refusal == "" {
			t.Fatalf("%s at %d GP offered to a customer with 7", o.Name, o.Price)
		}
	}
	v = s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: mageSlot})
	if ch := panelOf(t, v); ch.Class != "MAGE" {
		t.Fatalf("the mage: %+v", ch)
	}
	for _, o := range v.Shop.View.Catalog {
		if o.Name == "LONG SWORD" && o.Usable {
			t.Fatal("a long sword usable by a mage")
		}
	}
}

// Boltac serves the party only: the counter takes a party slot, never a
// roster character outside the party. (The floating character sheet is
// another matter: see TestSheetsOutsideThePartyAtBoltacs.)
func TestBoltacServesThePartyOnly(t *testing.T) {
	s, dir := shopSession(t)
	outsider := hero("OUTSIDER", engine.Neutral, "")
	rosterWith(t, s, outsider)
	for _, slot := range []int{6, 7, 19, -1} {
		v, wrote := shopAct(t, s, dir, ShopRequest{Action: ShopActEnter, Slot: slot})
		if v.Error == "" || wrote || v.Shop.View.Customer != -1 {
			t.Fatalf("slot %d: %q wrote %v customer %d", slot, v.Error, wrote, v.Shop.View.Customer)
		}
	}
	walkTo(t, s, "tavern")
	s.LeaveParty(5)
	walkTo(t, s, "trading")
	if v := s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 5}); v.Error == "" {
		t.Fatal("served an empty party slot")
	}
}

// Equipping beside the counter is the sheet's EquipAction: it changes the
// item, the AC, Boltac's view of it and the save at once; it is refused,
// writing nothing, while Boltac waits for an answer.
func TestEquippingBesideTheCounter(t *testing.T) {
	s, dir := shopSession(t)
	s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 0})
	done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: chainMail}, "buy", "CHAIN MAIL", -90)
	v := s.State()
	ch := panelOf(t, v)
	if len(ch.Items) != 1 || ch.Items[0].Equipped || ch.Items[0].EquipRefusal != "" || ch.AC != 10 {
		t.Fatalf("bought, carried: %+v AC %d", ch.Items, ch.AC)
	}
	v = s.EquipAction(EquipRequest{Index: ch.Index, N: 1, Equip: true})
	ch = panelOf(t, v)
	if v.Error != "" || !ch.Items[0].Equipped || ch.AC == 10 || !v.Shop.View.Items[0].Equipped || v.PartyBar[0].AC != ch.AC {
		t.Fatalf("equipped: %q %+v AC %d, Boltac %+v", v.Error, ch.Items[0], ch.AC, v.Shop.View.Items[0])
	}
	if c := onDisk(t, dir).Roster[ch.Index]; !c.Items[0].Equipped || c.AC != ch.AC {
		t.Fatalf("saved: %+v AC %d", c.Items[0], c.AC)
	}

	// Boltac asks before buying it back: no equipping until he is answered.
	s.ShopAction(ShopRequest{Action: ShopActSell, N: 1})
	saved, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	v = s.EquipAction(EquipRequest{Index: ch.Index, N: 1, Equip: false})
	if v.Error != errAnswerBoltac.Error() || !panelOf(t, v).Items[0].Equipped || panelOf(t, v).EquipNote != errAnswerBoltac.Error() {
		t.Fatalf("while Boltac asks: %q %+v", v.Error, panelOf(t, v))
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, saved) {
		t.Fatal("a refused equip wrote the save")
	}
	s.ShopAction(ShopRequest{Action: ShopActConfirmSale, Yes: false})
	if v = s.EquipAction(EquipRequest{Index: ch.Index, N: 1, Equip: false}); v.Error != "" || panelOf(t, v).Items[0].Equipped {
		t.Fatalf("after answering: %q", v.Error)
	}
}

// The class rule and curses hold beside the counter as on the sheet: a
// mage cannot put on armour; a cursed item, once worn, stays until Boltac
// uncurses it.
func TestEquipRulesBesideTheCounter(t *testing.T) {
	s, dir := shopSession(t)
	g := s.ctrl.Game
	mage := g.Town.Party.Members[mageSlot]
	mage.AddItem(chainMail, true)
	mage.AddItem(itemStaff, true)
	mage.Items[1].Cursed = true
	s.ctrl.Save()
	v := s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: mageSlot})
	ch := panelOf(t, v)
	if ch.Items[0].EquipRefusal != "A MAGE cannot use it." {
		t.Fatalf("chain mail: %q", ch.Items[0].EquipRefusal)
	}
	if v, wrote := equipAct(t, s, dir, EquipRequest{Index: ch.Index, N: 1, Equip: true}); v.Error == "" || wrote {
		t.Fatalf("a mage in chain mail: %q", v.Error)
	}
	v = s.EquipAction(EquipRequest{Index: ch.Index, N: 2, Equip: true})
	if ch = panelOf(t, v); !ch.Items[1].Equipped || !ch.Items[1].Cursed || ch.Items[1].UnequipRefusal == "" {
		t.Fatalf("a cursed staff: %+v", ch.Items[1])
	}
	if v, wrote := equipAct(t, s, dir, EquipRequest{Index: ch.Index, N: 2, Equip: false}); v.Error == "" || wrote {
		t.Fatalf("took off a cursed staff: %q", v.Error)
	}
	if o := v.Shop.View.Items[1]; o.UncurseRefusal != "" || !o.Cursed {
		t.Fatalf("Boltac would not uncurse it: %+v", o)
	}
}

// Purchases, sales, equipping and pooling are in the save: after a
// restart the panel's character is as it was, through the party bar.
func TestTheCounterAfterARestart(t *testing.T) {
	s, dir := shopSession(t)
	s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 1})
	done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: chainMail}, "buy", "CHAIN MAIL", -90)
	done(t, s, dir, ShopRequest{Action: ShopActBuy, Item: itemStaff}, "buy", "STAFF", -10)
	ch := panelOf(t, s.State())
	s.EquipAction(EquipRequest{Index: ch.Index, N: 1, Equip: true})
	if v := s.ShopAction(ShopRequest{Action: ShopActPool}); v.Error != "" {
		t.Fatal(v.Error)
	}
	before := panelOf(t, s.State())
	s.shutdown()

	s = NewSession(dir)
	defer s.shutdown()
	if s.State().Location != "trading" {
		walkTo(t, s, "trading")
	}
	v := s.State()
	if v.Shop.View.Customer >= 0 {
		v = s.ShopAction(ShopRequest{Action: ShopActLeaveCustomer})
	}
	v = s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 1})
	after := panelOf(t, v)
	if after.Gold != before.Gold || after.AC != before.AC || len(after.Items) != 2 || !after.Items[0].Equipped || after.Items[1].Equipped {
		t.Fatalf("after a restart: %d GP AC %d %+v; before %d GP AC %d", after.Gold, after.AC, after.Items, before.Gold, before.AC)
	}
	if v.PartyBar[1].AC != after.AC {
		t.Fatalf("party bar AC %d, panel %d", v.PartyBar[1].AC, after.AC)
	}
}

// What the brief asked to be checked and reported, not removed: Boltac's
// counter serves only the party, but the floating character sheet, which
// equips any roster character anywhere in town (M8), can still be open on
// a character outside the party when the player walks into Boltac's (the
// page keeps the sheet open across places). EquipAction takes a roster
// index and is open everywhere in town, so equipping that character works
// at Boltac's too. It is a sheet's operation, not a transaction: no gold
// or item changes hands.
func TestSheetsOutsideThePartyAtBoltacs(t *testing.T) {
	s, _ := shopSession(t)
	outsider := hero("OUTSIDER", engine.Neutral, "")
	outsider.AddItem(chainMail, true)
	rosterWith(t, s, outsider)
	index := len(s.ctrl.Game.Town.Roster.Characters) - 1
	if s.ctrl.Game.Town.Roster.Characters[index] != outsider || app.PartySlot(s.ctrl.Game.Town, outsider) >= 0 {
		t.Fatal("the outsider is not the last roster character, outside the party")
	}
	v := s.EquipAction(EquipRequest{Index: index, N: 1, Equip: true})
	if v.Error != "" || !outsider.Items[0].Equipped || v.Location != "trading" {
		t.Fatalf("equipping outside the party at Boltac's: %q", v.Error)
	}
}

// Arriving at Boltac's serves the first party member at once (UX-2): no
// "WHO WILL ENTER" first, nothing written. Who may be served is the
// counter's rule, unchanged: any party member, whatever their condition.
// With nobody to serve the counter waits at "WHO WILL ENTER".
func TestArrivingServesTheFirstMember(t *testing.T) {
	s, dir := partySession(t, false)
	walkTo(t, s, "castle")
	v, wrote := navigateWrote(t, s, dir, "trading")
	if wrote || v.Error != "" || v.Shop.View.Step != "main" || v.Shop.View.Customer != 0 || panelOf(t, v).Name != "AGUI" {
		t.Fatalf("arriving: %q wrote %v %+v", v.Error, wrote, v.Shop.View.Step)
	}

	// The first member cannot act: Boltac serves them all the same, as the
	// original's counter does.
	walkTo(t, s, "castle")
	s.ctrl.Game.Town.Party.Members[0].Status = engine.Dead
	if v = walkTo(t, s, "trading"); v.Shop.View.Customer != 0 {
		t.Fatalf("a dead first member: customer %d", v.Shop.View.Customer)
	}

	// Nobody to serve (the castle refuses Boltac's to no party, so only an
	// emptied party gets here): the counter waits.
	s.ctrl.ShopLeaveCustomer()
	s.ctrl.Game.Town.Party.Members = nil
	s.serveFirst()
	if v = s.State(); v.Shop.View.Step != "who" || v.Shop.View.Customer != -1 {
		t.Fatalf("no party: %+v", v.Shop.View)
	}
}

// navigateWrote is Navigate, reporting whether the save folder changed.
func navigateWrote(t *testing.T, s *Session, dir, dest string) (View, bool) {
	t.Helper()
	before := folder(t, dir)
	v := s.Navigate(dest)
	return v, !maps.Equal(before, folder(t, dir))
}
