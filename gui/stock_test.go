package main

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// Boltac's stock in the GUI (M8-1): it belongs to each game, so a reload
// or a restore (a new game on the same scenario) starts again from the
// scenario's values, practice games never touch the saved game's stock,
// and stock never reaches the save.

const (
	shortSwordPlus1 = 18 // 2 in stock in Wizardry I
	scrollHalito    = 28 // 27 in stock
)

func TestReloadAndRestoreResetStock(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	walkTo(t, s, "settings")
	s.ResetRoster() // a backup to restore
	rosterWith(t, s, hero("WALKER", engine.Good, ""))

	for _, reread := range []struct {
		name string
		do   func() View
	}{
		{"reload", s.ReloadRoster},
		{"restore", func() View { return s.RestoreBackup(s.ListBackups()[0].File) }},
	} {
		g := s.ctrl.Game
		g.SetStock(shortSwordPlus1, 0)
		g.SetStock(scrollHalito, 30)
		before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
		rand.Seed(51)
		v := reread.do()
		next := rand.Int63()
		rand.Seed(51)
		if v.Error != "" || next != rand.Int63() {
			t.Fatalf("%s: %q (or a random number was drawn)", reread.name, v.Error)
		}
		g = s.ctrl.Game
		if g.Stock(shortSwordPlus1) != 2 || g.Stock(scrollHalito) != 27 {
			t.Fatalf("%s: stock %d / %d, want the scenario's 2 / 27", reread.name, g.Stock(shortSwordPlus1), g.Stock(scrollHalito))
		}
		if s.scenario.Items[shortSwordPlus1].Stock != 2 || s.scenario.Items[scrollHalito].Stock != 27 {
			t.Fatalf("%s: the scenario's stock changed", reread.name)
		}
		if reread.name == "reload" {
			after, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("a reload changed the save")
			}
		}
	}
}

func TestPracticeKeepsItsOwnStock(t *testing.T) {
	rand.Seed(3)
	s, _ := partySession(t, false)
	s.ctrl.Game.SetStock(shortSwordPlus1, 1)
	s.PracticeExpedition()
	p := s.practice.Game
	if p.Scenario != s.ctrl.Game.Scenario {
		t.Fatal("the practice game is expected to share the scenario")
	}
	if p.Stock(shortSwordPlus1) != 2 {
		t.Fatalf("the practice game starts with %d, want the scenario's 2", p.Stock(shortSwordPlus1))
	}
	p.SetStock(shortSwordPlus1, 0)
	p.SetStock(scrollHalito, 1)
	if s.ctrl.Game.Stock(shortSwordPlus1) != 1 || s.ctrl.Game.Stock(scrollHalito) != 27 {
		t.Fatalf("practice changed the saved game's stock: %d / %d", s.ctrl.Game.Stock(shortSwordPlus1), s.ctrl.Game.Stock(scrollHalito))
	}
	s.AbandonPractice()
	if s.ctrl.Game.Stock(shortSwordPlus1) != 1 {
		t.Fatal("abandoning practice changed the saved game's stock")
	}
}

// Stock is not saved: a save after the stock changed is the save it would
// have been, and a restart starts from the scenario's stock.
func TestStockIsNotSaved(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	path := filepath.Join(dir, "roster1.json")
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	plain, _ := os.ReadFile(path)
	s.ctrl.Game.SetStock(shortSwordPlus1, 0)
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(plain, after) {
		t.Fatal("the stock reached the save")
	}
	if r := reopened(t, s, dir); r.ctrl.Game.Stock(shortSwordPlus1) != 2 {
		t.Fatalf("after a restart: stock %d", r.ctrl.Game.Stock(shortSwordPlus1))
	}
}
