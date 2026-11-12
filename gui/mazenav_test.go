package main

// The maze navigation follow-up: a resumed expedition is headed EXPEDITION
// RESUMED, a new one is not; the stairs suggest an answer by how the party
// came to them (app/stairs.go), and never take it.

import (
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// sitesOf names the squares where characters are known to lie, and who.
func sitesOf(g *engine.GameState) []string {
	var out []string
	for _, site := range g.StrandedSites() {
		for _, c := range site.Characters {
			out = append(out, site.Place.String()+" "+c.Name)
		}
	}
	return out
}

// A new expedition, even one setting out while others lie stranded, is not
// a resumed one, and the entrance stairs suggest staying in the maze.
func TestANewExpeditionIsNotResumed(t *testing.T) {
	rand.Seed(5)
	s, dir, _, pit := strandedAndRescuers(t, 2)
	defer s.shutdown()
	if v := s.State(); v.Title != "" || v.Notice != "" {
		t.Fatalf("rescuers set out with %q %q", v.Title, v.Notice)
	}
	stranded := strandedOnDiskAt(t, dir, pit)

	s.DungeonAction(app.ActCamp, 0)
	v := s.DungeonAction(app.ActLeaveCamp, 0)
	if v.Title != "" || v.Dungeon == nil || v.Dungeon.Prompt != "stairs" || v.Dungeon.Default != app.ActNo {
		t.Fatalf("at the entrance after Camp: %q %+v", v.Title, v.Dungeon)
	}
	if got := strandedOnDiskAt(t, dir, pit); !slices.Equal(got, stranded) {
		t.Fatalf("stranded %v, were %v", got, stranded)
	}
}

// A party's first expedition is not headed either.
func TestAFirstExpeditionIsNotResumed(t *testing.T) {
	rand.Seed(5)
	s, _ := partySession(t, false)
	defer s.shutdown()
	walkTo(t, s, "edge")
	if v := s.RealExpedition(); v.Error != "" || v.Title != "" {
		t.Fatalf("setting out: %q %q", v.Error, v.Title)
	}
	if v := s.DungeonAction(app.ActLeaveCamp, 0); v.Title != "" || v.Dungeon == nil || v.Dungeon.Default != app.ActNo {
		t.Fatalf("leaving Camp: %q %+v", v.Title, v.Dungeon)
	}
}

// Istary restarted during an expedition resumes it under EXPEDITION
// RESUMED, once: the party, where it stands, the dungeon and the stranded
// are as they were. The heading goes with the next action. Nothing of how
// the party came to the entrance is saved, so its stairs then suggest yes.
func TestAResumedExpeditionIsHeaded(t *testing.T) {
	rand.Seed(5)
	s, dir, _, pit := strandedAndRescuers(t, 2)
	before := s.State()
	g := s.ctrl.Game
	where := [3]int{g.MazeLevel, g.PlayerX, g.PlayerY}
	party := partyStanding(g)
	stranded := strandedOnDiskAt(t, dir, pit)
	sites := sitesOf(g)

	r := reopened(t, s, dir)
	defer r.shutdown()
	v := r.State()
	rg := r.ctrl.Game
	if v.Title != "EXPEDITION RESUMED" || v.Notice != "Your party has returned to the dungeon. Stranded adventurers remain at their recorded locations." {
		t.Fatalf("resumed with %q %q", v.Title, v.Notice)
	}
	if !v.Real || v.Screen != "dungeon" || [3]int{rg.MazeLevel, rg.PlayerX, rg.PlayerY} != where ||
		!reflect.DeepEqual(v.Dungeon, before.Dungeon) || !reflect.DeepEqual(partyStanding(rg), party) {
		t.Fatalf("resumed elsewhere or otherwise: %+v", v.Dungeon)
	}
	if got := strandedOnDiskAt(t, dir, pit); !slices.Equal(got, stranded) || !slices.Equal(sitesOf(rg), sites) || len(sites) == 0 {
		t.Fatalf("stranded %v, were %v", got, stranded)
	}

	v = r.DungeonAction(app.ActCamp, 0)
	if v.Title != "" || v.Notice != "" {
		t.Fatalf("still headed after an action: %q %q", v.Title, v.Notice)
	}
	v = r.DungeonAction(app.ActLeaveCamp, 0)
	if v.Dungeon == nil || v.Dungeon.Prompt != "stairs" || v.Dungeon.Default != app.ActYes {
		t.Fatalf("at the entrance after the restart: %+v", v.Dungeon)
	}
	// Both answers stay open; no keeps the party in the maze.
	if v = r.DungeonAction(app.ActNo, 0); v.Error != "" || v.Dungeon == nil || v.Dungeon.Prompt != "" {
		t.Fatalf("no: %q %+v", v.Error, v.Dungeon)
	}
}

// Back at the entrance from inside the maze, the stairs suggest going home,
// and yes takes the party there with the stranded left where they lie.
func TestReturningToTheEntranceSuggestsGoingHome(t *testing.T) {
	rand.Seed(5)
	s, dir, _, pit := strandedAndRescuers(t, 2)
	defer s.shutdown()
	stranded := strandedOnDiskAt(t, dir, pit)
	g := s.ctrl.Game
	for i := 0; i < 4 && g.WallAhead() != data.WallOpen; i++ {
		s.DungeonAction(app.ActRight, 0)
	}
	for _, a := range []string{app.ActForward, app.ActRight, app.ActRight} {
		if v := s.DungeonAction(a, 0); v.Error != "" || v.Dungeon.Prompt != "" {
			t.Fatalf("%s: %q %+v", a, v.Error, v.Dungeon)
		}
	}
	v := s.DungeonAction(app.ActForward, 0)
	if v.Dungeon == nil || v.Dungeon.Prompt != "stairs" || v.Dungeon.Default != app.ActYes || !slices.Equal(v.Dungeon.Actions, []string{app.ActYes, app.ActNo}) {
		t.Fatalf("back at the entrance: %+v", v.Dungeon)
	}
	if v = s.DungeonAction(app.ActYes, 0); v.Real || v.Location != "castle" {
		t.Fatalf("yes: real %v at %q (%q)", v.Real, v.Location, v.Error)
	}
	if got := strandedOnDiskAt(t, dir, pit); !slices.Equal(got, stranded) {
		t.Fatalf("stranded %v, were %v", got, stranded)
	}
}

// With no one stranded, a resumed expedition is headed as ever but says
// nothing of stranded adventurers; a new expedition is not headed at all.
func TestAResumeWithNoOneStrandedSaysNothingOfThem(t *testing.T) {
	rand.Seed(5)
	s, dir := partySession(t, false)
	walkTo(t, s, "edge")
	if v := s.RealExpedition(); v.Error != "" || v.Title != "" || v.Notice != "" {
		t.Fatalf("setting out: %q %q %q", v.Error, v.Title, v.Notice)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	if sites := s.ctrl.Game.StrandedSites(); len(sites) != 0 {
		t.Fatalf("stranded already: %v", sitesOf(s.ctrl.Game))
	}
	r := reopened(t, s, dir)
	defer r.shutdown()
	if v := r.State(); !v.Real || v.Title != "EXPEDITION RESUMED" || v.Notice != "Your party has returned to the dungeon." {
		t.Fatalf("resumed with %q %q", v.Title, v.Notice)
	}
}
