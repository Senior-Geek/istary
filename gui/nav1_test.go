package main

// NAV-1: a game opens at the Castle, whether new, continued after Select
// Game or continued after a restart, without its save being written; the
// Castle's ways out are its five destinations, and the Edge of Town's are
// the Castle, the Training Grounds and Settings (the page shows the last
// two among the Edge's actions, the Castle in the top bar). An expedition
// under way resumes in the maze (TestAnExpeditionContinuesWhereItWasSaved).

import (
	"bytes"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

func exitDests(v View) []string {
	var out []string
	for _, e := range v.Exits {
		out = append(out, e.Dest)
	}
	return out
}

func TestAGameOpensAtTheCastle(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	v := fine(t, s.NewGame(1))
	if v.Location != "castle" {
		t.Fatalf("a new game opened at %q", v.Location)
	}
	rosterWith(t, s, hero("ALDA", engine.Good, ""))
	join(t, s, 0, "")

	// Left from Settings' place, the Edge of Town, and from the Training
	// Grounds by way of it: wherever the game was left, it opens at the
	// Castle.
	for _, last := range []string{"settings", "training"} {
		walkTo(t, s, last)
		changeGame(t, s)
		save := readFile(t, filepath.Join(s.slotDir(1), saveFile))
		if v = fine(t, s.ContinueGame(1)); v.Location != "castle" || v.Screen != "roster" || v.Real {
			t.Fatalf("after %s, continued at %q (screen %q)", last, v.Location, v.Screen)
		}
		if !bytes.Equal(readFile(t, filepath.Join(s.slotDir(1), saveFile)), save) {
			t.Fatal("continuing wrote the save")
		}
	}

	walkTo(t, s, "edge")
	save := readFile(t, filepath.Join(s.slotDir(1), saveFile))
	r := relibrary(t, s, root)
	defer r.shutdown()
	if v = fine(t, r.ContinueGame(1)); v.Location != "castle" || partyNames(v) != "ALDA" {
		t.Fatalf("after a restart: at %q, party %q", v.Location, partyNames(v))
	}
	if !bytes.Equal(readFile(t, filepath.Join(r.slotDir(1), saveFile)), save) {
		t.Fatal("continuing after a restart wrote the save")
	}
}

func TestTheCastleAndTheEdgeWaysOut(t *testing.T) {
	s, _ := newLibrary(t)
	defer s.shutdown()
	v := fine(t, s.NewGame(1))
	if got := exitDests(v); !slices.Equal(got, []string{"tavern", "inn", "trading", "temple", "edge"}) {
		t.Fatalf("the Castle's ways out: %v", got)
	}
	if v.Return != nil {
		t.Fatal("a way back to the Castle at the Castle")
	}
	v = walkTo(t, s, "edge")
	if got := exitDests(v); !slices.Equal(got, []string{"training", "castle", "settings"}) {
		t.Fatalf("the Edge's ways out: %v", got)
	}
	if v.Return == nil || !v.Return.Open {
		t.Fatal("no way back to the Castle (Esc) from the Edge")
	}
	// Select Game is offered here, and refused elsewhere with its new name.
	walkTo(t, s, "training")
	if v = s.ChangeGame(); v.Error != "Select Game is at the Edge of Town." {
		t.Fatalf("Select Game from the Training Grounds: %q", v.Error)
	}
}
