package main

// The R-3 acceptance walkthrough (brief section 10): the markers through a
// two-trip rescue, through the Session as the front end drives it, on a
// scratch save folder. Run it with
//
//	go test -run TestR3Walkthrough -v
//
// to read each step. Expedition A (six weak adventurers) is wiped out: its
// square is marked, x6. Rescue expedition B (three) sees the marker on the
// F10 map from the entrance, walks until A is in its area, takes three
// (x3), is closed and started again (still x3), and brings them home.
// Rescue expedition C takes the rest: the marker goes, and stays gone
// after a restart.

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

func TestR3Walkthrough(t *testing.T) {
	rand.Seed(41)
	rng := rand.New(rand.NewSource(41))
	s, dir := partySession(t, true)
	var b, c []*engine.Character
	for i := 0; i < 6; i++ {
		ch := engine.NewCharacter(fmt.Sprintf("B%d", i+1), engine.Human, engine.Fighter, engine.Neutral)
		if i >= 3 {
			ch.Name = fmt.Sprintf("C%d", i-2)
		}
		ch.HP, ch.MaxHP, ch.Level, ch.AC = 400, 400, 12, -2
		if i < 3 {
			b = append(b, ch)
		} else {
			c = append(c, ch)
		}
	}
	rosterWith(t, s, append(append([]*engine.Character(nil), b...), c...)...)
	g := s.ctrl.Game
	partyA := append([]*engine.Character(nil), g.Town.Party.Members...)
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}

	// Expedition A: one marker, six names, each with its condition.
	fought := defeatA(t, s, rng)
	marks := mustMarkers(t, s, fought, 6)
	t.Logf("A (%v) is stranded at %s; the map marks it: %s", namesOf(partyA), fought, marks)

	// Rescue expedition B sees it from the entrance.
	setOut(t, s, b, 6)
	g = s.ctrl.Game
	v := s.State()
	here, _ := g.Place()
	t.Logf("B (%v) at the entrance %s; F10 map L%d: %v", namesOf(b), here, fought.Level, automapMarkers(t, s, fought.Level))
	mustMarkers(t, s, fought, 6)
	minimapAgrees(t, s, v)

	v = walkUntil(t, s, rng, func(g *engine.GameState) bool { return inAreaOf(g, fought) }, false)
	here, _ = g.Place()
	t.Logf("B finds A's square %s in its area at %s; minimap markers %v", fought, here, minimapLines(v))
	minimapAgrees(t, s, v)
	v = s.DungeonAction(app.ActInspect, 0)
	t.Logf("Inspect: %s", progressLine(v))
	for _, name := range []string{"BGUI", "DGUI", "FGUI"} {
		v = s.DungeonAction(app.ActPickUp, foundNamed(v, name))
		if v.Error != "" {
			t.Fatalf("pick %s: %s", name, v.Error)
		}
		t.Logf("  %s  map: %v", progressLine(v), automapMarkers(t, s, fought.Level))
	}
	mustMarkers(t, s, fought, 3)
	minimapAgrees(t, s, v)
	s.DungeonAction(app.ActLeave, 0)

	s.shutdown()
	s = NewSession(dir)
	defer s.shutdown()
	g = s.ctrl.Game
	v = s.State()
	t.Logf("after a restart: map %v; minimap %v", automapMarkers(t, s, fought.Level), minimapLines(v))
	mustMarkers(t, s, fought, 3)
	minimapAgrees(t, s, v)

	walkUntil(t, s, rng, func(g *engine.GameState) bool { return false }, true)
	s.DungeonAction(app.ActYes, 0)
	t.Logf("B climbs out with %v; the marker: %v", namesOf(g.Town.Party.Members), mustMarkers(t, s, fought, 3))

	// Rescue expedition C takes the rest.
	walkTo(t, s, "tavern")
	for len(g.Town.Party.Members) > 0 {
		s.LeaveParty(0)
	}
	setOut(t, s, c, 9)
	walkUntil(t, s, rng, func(g *engine.GameState) bool { return inAreaOf(g, fought) }, false)
	v = s.DungeonAction(app.ActInspect, 0)
	t.Logf("C (%v) inspects: %s", namesOf(c), progressLine(v))
	for len(v.Dungeon.Inspect.Found) > 0 {
		if v = s.DungeonAction(app.ActPickUp, 1); v.Error != "" {
			t.Fatal(v.Error)
		}
		t.Logf("  %s  map: %v", progressLine(v), automapMarkers(t, s, fought.Level))
	}
	if got := automapMarkers(t, s, fought.Level); len(got) != 0 || len(v.Minimap.Stranded) != 0 {
		t.Fatalf("markers after the last pickup: %v %v", got, v.Minimap.Stranded)
	}
	s.DungeonAction(app.ActLeave, 0)
	walkUntil(t, s, rng, func(g *engine.GameState) bool { return false }, true)
	s.DungeonAction(app.ActYes, 0)
	s.shutdown()
	s = NewSession(dir)
	defer s.shutdown()
	if sites := s.ctrl.Game.StrandedSites(); len(sites) != 0 {
		t.Fatalf("after C and a restart: %+v", sites)
	}
	t.Logf("C climbs out with %v; after a restart no one is marked", namesOf(s.ctrl.Game.Town.Party.Members))
}

// mustMarkers checks the real game's map marks p alone on its level, with
// n characters, and returns the marker.
func mustMarkers(t *testing.T, s *Session, p engine.Place, n int) string {
	t.Helper()
	m, err := s.ctrl.Map(p.Level)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Stranded) != 1 || m.Stranded[0].Place != p.String() || m.Stranded[0].Count != n || len(m.Stranded[0].Characters) != n {
		t.Fatalf("want %s x%d, markers %+v", p, n, m.Stranded)
	}
	return markerLine(m.Stranded[0])
}

func minimapLines(v View) []string {
	out := []string{}
	if v.Minimap != nil {
		for _, m := range v.Minimap.Stranded {
			out = append(out, fmt.Sprintf("%s at window (%d,%d) x%d", m.Place, m.Row, m.Col, m.Count))
		}
	}
	return out
}

// progressLine is Inspect's progress as the screen shows it.
func progressLine(v View) string {
	if v.Dungeon == nil || v.Dungeon.Inspect == nil {
		return "(not inspecting)"
	}
	i := v.Dungeon.Inspect
	var found, carried []string
	for _, f := range i.Found {
		found = append(found, f.Name+" "+f.Condition)
	}
	for _, ch := range i.Carried {
		carried = append(carried, ch.Name+" "+ch.Condition)
	}
	return fmt.Sprintf("%q party %d, room %d; here [%s]; elsewhere %d; unable to act [%s]",
		strings.Join(v.Dungeon.Lines, " "), i.Size, i.Room, strings.Join(found, ", "), i.Elsewhere, strings.Join(carried, ", "))
}
