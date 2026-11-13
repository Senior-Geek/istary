package main

import (
	"math/rand"
	"os"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// TestWriteAcceptanceLibrary writes the library the live acceptance checks
// open (scripts/acceptance/live.mjs, docs/TESTING.md): one game, a party of
// six carrying things worn and packed, scrolls and potions among them, and
// two more on the roster. It writes only when ISTARY_ACCEPTANCE_LIBRARY
// names a new or empty directory; otherwise it is skipped.
func TestWriteAcceptanceLibrary(t *testing.T) {
	root := os.Getenv("ISTARY_ACCEPTANCE_LIBRARY")
	if root == "" {
		t.Skip("ISTARY_ACCEPTANCE_LIBRARY is not set")
	}
	if entries, err := os.ReadDir(root); err == nil && len(entries) > 0 {
		t.Fatalf("%s is not empty", root)
	}
	rand.Seed(11)
	s := newLibraryAt(t, root)
	fine(t, s.NewGame(1))
	fillParty(t, s, false, engine.Neutral)

	const (
		dagger, smallShield, leather, chainMail = 6, 7, 10, 11
		diosPotion, scrollKatino, scrollHalito  = 15, 21, 28
	)
	carry := [][]struct {
		item int
		worn bool
	}{
		{{chainMail, true}, {smallShield, true}, {diosPotion, false}, {dagger, false}},
		{{leather, true}, {itemStaff, false}},
		{{leather, true}, {dagger, true}, {scrollHalito, false}},
		{{diosPotion, false}},
		{{scrollKatino, false}, {dagger, true}},
		{},
	}
	g := s.ctrl.Game
	for i, m := range g.Town.Party.Members {
		for _, c := range carry[i] {
			m.AddItem(c.item, true)
		}
	}
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	// Worn the way the window equips: by the controller's rules, saved.
	for i, m := range g.Town.Party.Members {
		for n, c := range carry[i] {
			if c.worn {
				fine(t, s.EquipAction(EquipRequest{Index: rosterIndex(g, m), N: n + 1, Equip: true}))
			}
		}
	}
	createMage(t, s, "spare")
	rosterWith(t, s, hero("RESERVE", engine.Good, ""))
	walkTo(t, s, "castle")
	s.shutdown()

	// Opened and saved once more, as the window would: loading completes
	// what a save leaves implicit (the spells a caster knows), so the file
	// is what the window itself writes, and its first save changes nothing.
	r := newLibraryAt(t, root)
	defer r.shutdown()
	v := fine(t, r.ContinueGame(1))
	if err := r.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	if v.Error != "" || len(v.Roster) != 8 || len(v.Party) != 6 || v.Location != "castle" {
		t.Fatalf("library: error %q, roster %d, party %d, at %q", v.Error, len(v.Roster), len(v.Party), v.Location)
	}
}
