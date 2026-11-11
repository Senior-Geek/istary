package main

// The R-1 acceptance walkthrough (brief section 13), through the Session as
// the front end drives it, on a scratch save folder. Run it with
//
//	go test -run TestR1Walkthrough -v
//
// to read each step. Expedition A sets out weak, meets a real encounter
// and is wiped out in genuine combat; Expedition B, formed from the others,
// sets out and climbs home; the app then closes and starts again.

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

func TestR1Walkthrough(t *testing.T) {
	rand.Seed(41)
	rng := rand.New(rand.NewSource(41))
	s, dir := partySession(t, true) // A: six weak characters, at the Edge of Town
	var spares []*engine.Character
	for i := 0; i < 6; i++ {
		ch := engine.NewCharacter(fmt.Sprintf("B%d", i+1), engine.Human, engine.Fighter, engine.Neutral)
		ch.HP, ch.MaxHP, ch.Level = 30, 30, 3
		spares = append(spares, ch)
	}
	rosterWith(t, s, spares...)
	g := s.ctrl.Game
	partyA := append([]*engine.Character(nil), g.Town.Party.Members...)
	t.Logf("A: %v (HP 2, AC 10); also in the roster: %v", namesOf(partyA), namesOf(spares))

	// Expedition A.
	v := s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	v = s.DungeonAction(app.ActNo, 0)
	start, _ := g.Place()
	t.Logf("A sets out: %s at %s facing %s", v.Screen, start, g.Facing.Letter())
	var fought engine.Place
	var before []string
	for step := 0; step < 3000 && s.real; step++ {
		switch v.Screen {
		case "dungeon":
			d := v.Dungeon
			switch {
			case d.Prompt == "stairs":
				v = s.DungeonAction(app.ActNo, 0)
			case d.Prompt == "messages":
				v = s.DungeonAction(app.ActContinue, 0)
			case d.Prompt != "":
				t.Fatalf("prompt %q", d.Prompt)
			case g.WallAhead() == data.WallOpen && rng.Intn(4) > 0:
				v = s.DungeonAction(app.ActForward, 0)
			default:
				v = s.DungeonAction(app.ActRight, 0)
			}
		case "combat":
			fought, _ = g.Place()
			t.Logf("A meets an encounter at %s", fought)
			v, _ = fightTo(t, s, rng)
			if v.Screen == "combat" && v.Combat.Phase == "defeat" {
				t.Logf("A is defeated: %s", partyLine(g.Town.Party.Members))
				before = nil
				for _, m := range partyA {
					raw, _ := json.Marshal(standingOf(m))
					before = append(before, string(raw))
				}
			}
			if v.Screen == "combat" {
				v = s.CombatLeave()
			}
		default:
			t.Fatalf("screen %q", v.Screen)
		}
	}
	if s.real || before == nil {
		t.Fatalf("A was not defeated (real %v)", s.real)
	}
	t.Logf("the castle: screen %q, party %d; notice: %s", v.Screen, len(v.Party), v.Notice)
	if v.Screen != "roster" || len(v.Party) != 0 || v.Expedition || v.Real {
		t.Fatalf("not back to the castle: %q party %d", v.Screen, len(v.Party))
	}
	for i, m := range partyA {
		raw, _ := json.Marshal(standingOf(m))
		if string(raw) != before[i] {
			t.Fatalf("%s changed by the defeat:\n%s\nwas\n%s", m.Name, raw, before[i])
		}
	}
	strandedView(t, v, namesOf(partyA), fought)
	strandedOnDisk(t, dir, namesOf(partyA), fought)
	for _, cv := range v.Roster {
		if cv.Stranded != "" {
			t.Logf("  roster: %-5s %-8s stranded %s, Tavern says %q", cv.Name, cv.Status, cv.Stranded, cv.JoinRefusal)
		}
	}
	walkTo(t, s, "tavern")
	for i := range partyA {
		if v := s.JoinParty(i); len(v.Party) != 0 || v.PendingJoin != "" {
			t.Fatalf("%s joined", partyA[i].Name)
		}
	}

	// Expedition B, from the others.
	for i := range spares {
		if v := s.JoinParty(6 + i); v.PendingJoin != "" {
			s.SubmitJoinPassword("")
		}
	}
	// The Temple (it takes a party to enter) will not treat A.
	walkTo(t, s, "temple")
	for i, m := range partyA {
		q, err := s.TempleQuote(i, 0)
		if err != nil || q.Refusal == "" {
			t.Fatalf("%s may be treated: %v", m.Name, err)
		}
		if i == 0 {
			t.Logf("Temple for %s: %q", m.Name, q.Refusal)
		}
	}
	walkTo(t, s, "edge")
	partyB := append([]*engine.Character(nil), g.Town.Party.Members...)
	if len(partyB) != 6 || partyB[0].Name != "B1" {
		t.Fatalf("B: %v", namesOf(partyB))
	}
	v = s.RealExpedition()
	v = s.DungeonAction(app.ActLeaveCamp, 0)
	entrance, _ := g.Place()
	t.Logf("B sets out: %v; at %s the stairs ask %q", namesOf(partyB), entrance, v.Dungeon.Prompt)
	for _, ch := range onDisk(t, dir).Roster { // A still stranded in the save
		for _, m := range partyA {
			if ch.Name == m.Name && (ch.Stranded == nil || *ch.Stranded != fought) {
				t.Fatalf("%s on disk: %v", ch.Name, ch.Stranded)
			}
		}
	}
	for _, m := range partyA {
		if m.Stranded == nil || *m.Stranded != fought {
			t.Fatalf("%s moved: %v", m.Name, m.Stranded)
		}
	}
	v = s.DungeonAction(app.ActYes, 0)
	t.Logf("B climbs out: %s; party %v", v.Notice, namesOf(g.Town.Party.Members))
	if s.real || len(v.Party) != 6 {
		t.Fatalf("B not home: real %v party %d", s.real, len(v.Party))
	}
	noneStrandedAmong(t, partyB)

	// Close, start again.
	s.shutdown()
	r := NewSession(dir)
	defer r.shutdown()
	rv := r.State()
	rg := r.ctrl.Game
	t.Logf("after restart: screen %q, castle party %v, roster %d", rv.Screen, namesOf(rg.Town.Party.Members), len(rg.Town.Roster.Characters))
	strandedView(t, rv, namesOf(partyA), fought)
	if len(rg.Town.Roster.Characters) != 12 || len(rv.Party) != 6 || rv.Party[0].Name != "B1" || rg.Expedition != nil {
		t.Fatalf("after restart: roster %d party %d", len(rg.Town.Roster.Characters), len(rv.Party))
	}
	seen := map[string]bool{}
	for _, ch := range rg.Town.Roster.Characters {
		if seen[ch.Name] {
			t.Fatalf("%s twice", ch.Name)
		}
		seen[ch.Name] = true
		if app.InParty(rg.Town, ch) == ch.IsStranded() {
			t.Fatalf("%s: in party %v, stranded %v", ch.Name, app.InParty(rg.Town, ch), ch.IsStranded())
		}
	}
	t.Logf("A remains at %s after the restart; no one twice; B in the castle", fought)
}

func partyLine(members []*engine.Character) string {
	var out string
	for _, m := range members {
		out += fmt.Sprintf("%s %v; ", m.Name, m.Status)
	}
	return out
}

// standingOf is a character's condition, equipment, gold and experience.
func standingOf(m *engine.Character) standing {
	return standing{m.Name, m.HP, m.XP, m.Gold, m.PoisonAmt, m.Status, m.Items, m.ItemCount, m.MageSpells, m.PriestSpells}
}

func noneStrandedAmong(t *testing.T, chars []*engine.Character) {
	t.Helper()
	for _, m := range chars {
		if m.Stranded != nil || m.InMaze {
			t.Fatalf("%s: stranded %v in maze %v", m.Name, m.Stranded, m.InMaze)
		}
	}
}
