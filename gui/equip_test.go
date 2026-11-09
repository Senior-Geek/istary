package main

import (
	"bytes"
	"encoding/json"
	"maps"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// Equipping in town (M8): the sheet's items carry the controller's
// answers; EquipAction equips and takes off, saves a change at once, writes
// nothing for a refusal, draws no random number, and says when a change is
// not saved. It is not open during an expedition or practice.

const (
	evilShield  = 64 // EVIL SHIELD +3: curses a non-evil wearer
	smallShield = 7
)

// equipAct calls EquipAction, failing if it drew a random number, and
// reports whether it wrote to the save folder.
func equipAct(t *testing.T, s *Session, dir string, r EquipRequest) (View, bool) {
	t.Helper()
	before := folder(t, dir)
	rand.Seed(17)
	v := s.EquipAction(r)
	next := rand.Int63()
	rand.Seed(17)
	if next != rand.Int63() {
		t.Fatalf("%+v drew a random number", r)
	}
	return v, !maps.Equal(before, folder(t, dir))
}

func rosterChar(v View, name string) CharacterView {
	for _, ch := range v.Roster {
		if ch.Name == name {
			return ch
		}
	}
	return CharacterView{}
}

func TestEquippingInTown(t *testing.T) {
	s, dir := partySession(t, false) // at the Edge of Town
	g := s.ctrl.Game
	fighter := g.Town.Party.Members[0]
	for _, it := range []int{chainMail, smallShield, evilShield, scrollHalito, itemStaff} {
		fighter.AddItem(it, true)
	}
	fighter.Gold = 1 << 20 // for Boltac's uncursing fee
	s.ctrl.Save()
	index := rosterIndex(g, fighter)
	v := s.State()
	sheet := rosterChar(v, fighter.Name)
	if sheet.EquipNote != "" || sheet.Items[0].Kind != "armour" || sheet.Items[3].Kind != "" || sheet.Items[3].EquipRefusal == "" {
		t.Fatalf("the sheet: note %q items %+v", sheet.EquipNote, sheet.Items)
	}

	// Equip the chain mail: saved, the AC shown everywhere.
	v, wrote := equipAct(t, s, dir, EquipRequest{Index: index, N: 1, Equip: true})
	sheet = rosterChar(v, fighter.Name)
	if v.Error != "" || !wrote || !sheet.Items[0].Equipped || sheet.AC != 7 || v.PartyBar[0].AC != 7 || !strings.HasSuffix(v.Notice, " Saved.") {
		t.Fatalf("chain mail: %q wrote %v AC %d/%d notice %q", v.Error, wrote, sheet.AC, v.PartyBar[0].AC, v.Notice)
	}
	mustMatchDisk(t, s, dir, "chain mail")

	// A refusal writes nothing and says why.
	for _, r := range []EquipRequest{{Index: index, N: 4, Equip: true}, {Index: index, N: 2, Equip: false}, {Index: index, N: 9, Equip: true}} {
		if v, wrote = equipAct(t, s, dir, r); v.Error == "" || wrote {
			t.Fatalf("%+v: %q, wrote %v", r, v.Error, wrote)
		}
	}

	// The evil shield sticks to a neutral fighter, and says so; the small
	// shield cannot take its place, nor can it come off.
	v, _ = equipAct(t, s, dir, EquipRequest{Index: index, N: 3, Equip: true})
	sheet = rosterChar(v, fighter.Name)
	if !sheet.Items[2].Cursed || !strings.Contains(v.Notice, "CURSED") || sheet.Items[1].EquipRefusal == "" || sheet.Items[2].UnequipRefusal == "" {
		t.Fatalf("evil shield: %+v, notice %q", sheet.Items, v.Notice)
	}
	if v, wrote = equipAct(t, s, dir, EquipRequest{Index: index, N: 3, Equip: false}); v.Error == "" || wrote {
		t.Fatalf("took off a cursed shield: %q", v.Error)
	}

	// Boltac sees the same: the worn chain mail asks before it is sold,
	// the stuck shield is refused, and uncursing frees the place.
	walkTo(t, s, "trading")
	v = s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: 0})
	if it := v.Shop.View.Items; !it[0].SellAsks || !it[2].Cursed || it[2].SellRefusal == "" {
		t.Fatalf("Boltac's view: %+v", it)
	}
	if v = s.ShopAction(ShopRequest{Action: ShopActUncurse, N: 3}); v.Error != "" {
		t.Fatalf("uncursing: %q", v.Error)
	}
	walkTo(t, s, "castle")
	v = s.State()
	sheet = rosterChar(v, fighter.Name)
	if len(sheet.Items) != 4 || sheet.Items[1].EquipRefusal != "" || sheet.AC != 7 {
		t.Fatalf("after uncursing: %+v AC %d", sheet.Items, sheet.AC)
	}

	// Taking off is saved too, and a restart finds it as the window has it.
	v, wrote = equipAct(t, s, dir, EquipRequest{Index: index, N: 1, Equip: false})
	if v.Error != "" || !wrote || rosterChar(v, fighter.Name).AC != 10 {
		t.Fatalf("taking off the chain mail: %q", v.Error)
	}
	restart(t, s, dir, "after equipping")
}

// Equipping is not open during practice or an expedition, or while a
// character is being created; the sheet says why and nothing is changed.
func TestEquippingOnlyInTown(t *testing.T) {
	s, dir := partySession(t, false)
	g := s.ctrl.Game
	fighter := g.Town.Party.Members[0]
	fighter.AddItem(chainMail, true)
	s.ctrl.Save()
	index := rosterIndex(g, fighter)
	closed := func(when, why string) {
		t.Helper()
		if note := rosterChar(s.State(), fighter.Name).EquipNote; note != why {
			t.Fatalf("%s: note %q, want %q", when, note, why)
		}
		if v, wrote := equipAct(t, s, dir, EquipRequest{Index: index, N: 1, Equip: true}); v.Error != why || wrote || fighter.Items[0].Equipped {
			t.Fatalf("%s: %q, wrote %v", when, v.Error, wrote)
		}
	}
	s.PracticeExpedition()
	closed("practice", errPracticeOut.Error())
	s.AbandonPractice()
	s.RealExpedition()
	closed("expedition", errExpeditionOut.Error())
	s.shutdown()

	s, dir = newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	s.ctrl.Game.Town.Roster.Characters[0].AddItem(chainMail, true)
	walkTo(t, s, "training")
	s.StartCreation("NEWCOMER")
	s.AnswerCreate(true)
	if note := rosterChar(s.State(), "WALKER").EquipNote; note != errCreating.Error() {
		t.Fatalf("creating: note %q", note)
	}
	if v, wrote := equipAct(t, s, dir, EquipRequest{Index: 0, N: 1, Equip: true}); v.Error != errCreating.Error() || wrote {
		t.Fatalf("creating: %q", v.Error)
	}
}

// A change whose save fails stays in the window and is said to be NOT
// saved; a restart meanwhile finds the save as it was; the next change
// saves both.
func TestEquippingFailedSave(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	s, dir := partySession(t, false)
	g := s.ctrl.Game
	fighter := g.Town.Party.Members[0]
	fighter.AddItem(chainMail, true)
	fighter.AddItem(smallShield, true)
	s.ctrl.Save()
	index := rosterIndex(g, fighter)
	saved, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))

	os.Chmod(dir, 0o555)
	v := s.EquipAction(EquipRequest{Index: index, N: 1, Equip: true})
	os.Chmod(dir, 0o755)
	if !strings.Contains(v.Error, "Could not save") || v.Unsaved == "" || !strings.Contains(v.Notice, "NOT saved") || !fighter.Items[0].Equipped {
		t.Fatalf("error %q unsaved %q notice %q", v.Error, v.Unsaved, v.Notice)
	}
	if now, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(now, saved) {
		t.Fatal("the file changed although the save failed")
	}
	r := reopened(t, s, dir)
	if m := r.ctrl.Game.Town.Party.Members[0]; m.Items[0].Equipped || m.AC != 10 {
		t.Fatalf("a restart now finds the chain mail worn (%v) or AC %d", m.Items[0].Equipped, m.AC)
	}
	v = s.EquipAction(EquipRequest{Index: index, N: 2, Equip: true})
	if v.Error != "" || v.Unsaved != "" {
		t.Fatalf("the next change: %q %q", v.Error, v.Unsaved)
	}
	mustMatchDisk(t, s, dir, "after the next save")
}

// TestEquipViewFixture records real character sheets for the page's tests
// (frontend/src/lib/equipViews.test.js): open, with each refusal, a stuck
// item, and closed during practice. Run with ISTARY_UPDATE_FIXTURES=1 to
// rewrite it.
func TestEquipViewFixture(t *testing.T) {
	type sheet struct {
		Case  string        `json:"case"`
		Sheet CharacterView `json:"sheet"`
	}
	var sheets []sheet
	rand.Seed(5)
	s, _ := partySession(t, false)
	g := s.ctrl.Game
	fighter, mage := g.Town.Party.Members[0], g.Town.Party.Members[4]
	for _, it := range []int{chainMail, smallShield, evilShield, scrollHalito} {
		fighter.AddItem(it, true)
	}
	mage.AddItem(chainMail, true)
	mage.AddItem(itemStaff, false)
	s.ctrl.Save()
	add := func(name string, ch *engine.Character) {
		sheets = append(sheets, sheet{name, rosterChar(s.State(), ch.Name)})
	}
	add("fighter, nothing worn", fighter)
	add("mage: armour refused, an unknown staff", mage)
	s.EquipAction(EquipRequest{Index: rosterIndex(g, fighter), N: 1, Equip: true})
	s.EquipAction(EquipRequest{Index: rosterIndex(g, fighter), N: 3, Equip: true})
	add("fighter: chain mail worn, evil shield stuck", fighter)
	s.PracticeExpedition()
	add("during practice", fighter)
	s.AbandonPractice()
	s.shutdown()

	got, _ := json.MarshalIndent(sheets, "", " ")
	got = append(got, '\n')
	path := filepath.Join("frontend", "src", "lib", "testdata", "equip-views.json")
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
		t.Fatal("equip-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
