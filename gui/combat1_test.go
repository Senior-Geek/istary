package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// COMBAT-1 through the bridge: a new mage's first expedition, the spell
// list's refusals, Party Retreat, the spell guide, and saves.

// toOrders plays the encounter on to order entry, member slot selected.
func toOrders(t *testing.T, s *Session, slot int) View {
	t.Helper()
	v := s.State()
	for step := 0; step < 20 && v.Combat != nil && v.Combat.Next != app.NextOrders; step++ {
		switch v.Combat.Next {
		case app.NextStart:
			v = s.CombatStart()
		case app.NextFriendly:
			v = s.CombatFriendly(true)
		case app.NextAdvance:
			v = s.CombatAdvance()
		case app.NextEndRound:
			v = s.CombatEndRound()
		default:
			t.Fatalf("on the way to orders: %s", v.Combat.Next)
		}
	}
	if v.Combat == nil || v.Combat.Next != app.NextOrders {
		t.Fatalf("not at orders: %+v", v.Combat)
	}
	return s.CombatSelect(slot)
}

// A mage made at the Training Grounds starts with two level 1 casts, keeps
// them through a restart, and on the first expedition finds HALITO, KATINO
// and MOGREF ready to cast in the first fight, each described, and DUMAPIC
// listed as exploration only, refused if ordered. (A restart teaches every
// level 1 mage spell: finding #32, kept.)
func TestFirstExpeditionSpells(t *testing.T) {
	s, dir := newTestSession(t)
	var err error
	spendAll(t, s, toPoints(t, s, "elen", 1), []int{1, 3, 0, 2, 4, 5})
	mustOK(t, s.PickClass(1), "creation")
	mustOK(t, s.Confirm(true), "roster")
	r := reopened(t, s, dir)
	s.shutdown()
	var mage *engine.Character
	for _, c := range r.ctrl.Game.Town.Roster.Characters {
		if c.Name == "ELEN" {
			mage = c
		}
	}
	if mage == nil || mage.MageSpells != [7]int{2} || mage.MaxMageSpells != [7]int{2} {
		t.Fatalf("after a restart: %+v", mage)
	}
	if r.lock, err = lockSaveFolder(dir); err != nil { // r plays on, as the restarted app
		t.Fatal(err)
	}
	var others []*engine.Character
	for i := 0; i < 5; i++ {
		ch := engine.NewCharacter(string(rune('A'+i))+"F", engine.Human, engine.Fighter, engine.Neutral)
		ch.HP, ch.MaxHP = 14, 14
		others = append(others, ch)
	}
	rosterWith(t, r, others...)
	walkTo(t, r, "tavern")
	for i := range r.ctrl.Game.Town.Roster.Characters {
		if v := r.JoinParty(i); v.PendingJoin != "" {
			r.SubmitJoinPassword("")
		}
	}
	walkTo(t, r, "edge")
	realEncounter(t, r)
	r.ctrl.Game.Combat.HasChest = false
	v := toOrders(t, r, 1) // ELEN joined first
	if v.Combat.Party[0].Name != "ELEN" {
		t.Fatalf("slot 1 is %s", v.Combat.Party[0].Name)
	}
	byName := map[string]app.CombatSpellView{}
	for _, sp := range v.Combat.Spells {
		byName[sp.Name] = sp
	}
	for _, n := range []string{"HALITO", "MOGREF", "KATINO"} {
		if sp := byName[n]; !sp.Castable || sp.Left != 2 || sp.Max != 2 || sp.Summary != app.SpellSummary(n) || sp.Summary == "" {
			t.Errorf("%s on the first expedition: %+v", n, sp)
		}
	}
	if sp := byName["DUMAPIC"]; sp.Castable || !strings.HasPrefix(sp.Reason, "Exploration only") {
		t.Errorf("DUMAPIC in combat: %+v", sp)
	}
	if v := r.CombatOrder(OrderRequest{Slot: 1, Action: "spell", Spell: "DUMAPIC", Item: -1}); !strings.Contains(v.Error, "Exploration only") {
		t.Fatalf("DUMAPIC ordered: %q", v.Error)
	}
	if mage.MageSpells[0] != 2 {
		t.Fatalf("a refused order spent a point: %v", mage.MageSpells)
	}
	if v := r.CombatOrder(OrderRequest{Slot: 1, Action: "spell", Spell: "KATINO", Group: 1, Item: -1}); v.Error != "" {
		t.Fatalf("KATINO: %s", v.Error)
	}
}

// Party Retreat through the bridge: one call orders the retreat and shows
// it for confirmation; it is refused where it does not apply; a practice
// encounter's retreat leaves the roster on disk as it was.
func TestCombatRetreatThroughTheBridge(t *testing.T) {
	s, dir := partySession(t, false)
	before, err := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if v := s.CombatRetreat(); v.Error == "" {
		t.Fatal("a retreat in town")
	}
	s.PracticeEncounter(false)
	v := toOrders(t, s, 1)
	if v.Combat.Retreat == "" {
		t.Fatalf("no retreat offered: %+v", v.Combat)
	}
	runner := v.Combat.Retreat
	v = s.CombatRetreat()
	if v.Error != "" || v.Combat.Next != app.NextConfirm {
		t.Fatalf("retreat: %q %s", v.Error, v.Combat.Next)
	}
	runs := 0
	for _, o := range v.Combat.Orders {
		switch {
		case o.Action == "run" && o.Member == runner:
			runs++
		case o.Action != "parry":
			t.Fatalf("order %+v", o)
		}
	}
	if runs != 1 {
		t.Fatalf("orders %+v", v.Combat.Orders)
	}
	if v := s.CombatRetreat(); v.Error == "" {
		t.Fatal("a retreat at confirmation")
	}
	for step := 0; step < 400 && s.State().Screen == "combat"; step++ {
		c := s.State().Combat
		switch c.Next {
		case app.NextOrders:
			s.CombatRetreat()
		case app.NextConfirm:
			s.CombatCommit()
		case app.NextAdvance:
			s.CombatAdvance()
		case app.NextEndRound:
			s.CombatEndRound()
		case app.NextChoice:
			s.CombatChooseBoon(1)
		case app.NextLeave:
			s.CombatLeave()
		default:
			t.Fatalf("next %s", c.Next)
		}
	}
	if v := s.State(); v.Screen == "combat" {
		t.Fatal("still in combat")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("a practice retreat changed the roster on disk")
	}
}

// The spell guide is Go's, the same in every call, and reading it, or the
// spells of a character from a save made before COMBAT-1, writes nothing:
// the save format is unchanged (no field was added to it).
func TestSpellGuideAndOldSaves(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "saves")
	os.MkdirAll(dir, 0o755)
	old, err := os.ReadFile(filepath.Join("..", "engine", "testdata", "save_roster1.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "roster1.json")
	if err := os.WriteFile(path, old, 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewSession(dir)
	defer s.shutdown()
	v := s.State()
	if v.Fatal != "" || len(v.Roster) == 0 {
		t.Fatalf("old save: fatal %q, roster %d", v.Fatal, len(v.Roster))
	}
	g := s.SpellGuide()
	if !reflect.DeepEqual(g, app.SpellGuide()) || len(g) != len(engine.SpellTable) {
		t.Fatalf("guide %d entries", len(g))
	}
	for _, ch := range v.Roster {
		for _, n := range ch.Spells {
			if app.SpellSummary(n) == "" {
				t.Errorf("%s knows %s, which the guide does not describe", ch.Name, n)
			}
		}
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(old, after) {
		t.Fatal("opening an old save and reading the guide rewrote it")
	}
}
