package main

// R-3 through the Session: the maps' stranded markers after pickups, saves,
// restarts and practice. The numbers are the R-3 brief's tests.

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// markerLine is a marker as "place xN: NAME CONDITION, ...".
func markerLine(m app.MapStranded) string {
	var b strings.Builder
	b.WriteString(m.Place)
	b.WriteString(" x" + string(rune('0'+m.Count)) + ":")
	for _, ch := range m.Characters {
		b.WriteString(" " + ch.Name + " " + ch.Condition)
	}
	return b.String()
}

// automapMarkers is level l's markers on the F10 map (Session.Automap).
func automapMarkers(t *testing.T, s *Session, l int) []string {
	t.Helper()
	m, err := s.Automap(l)
	if err != nil {
		t.Fatalf("automap L%d: %v", l, err)
	}
	out := []string{}
	for _, x := range m.Stranded {
		out = append(out, markerLine(x))
	}
	return out
}

// minimapAgrees checks the minimap's markers are the F10 map's, placed in
// its window (13).
func minimapAgrees(t *testing.T, s *Session, v View) {
	t.Helper()
	if v.Minimap == nil {
		t.Fatal("no minimap")
	}
	full, err := s.Automap(v.Minimap.Level)
	if err != nil {
		t.Fatal(err)
	}
	byPlace := map[string]app.MapStranded{}
	for _, m := range full.Stranded {
		byPlace[m.Place] = m
	}
	for _, m := range v.Minimap.Stranded {
		f, ok := byPlace[m.Place]
		if !ok || !reflect.DeepEqual(f.Characters, m.Characters) || f.Count != m.Count {
			t.Fatalf("minimap %s: %+v, full map %+v", m.Place, m, f)
		}
	}
}

// (8, 11, 13) Through the Session: the markers follow each pickup on both
// maps, and are the same after the game is closed and started again.
func TestMarkersSurviveARestart(t *testing.T) {
	rand.Seed(61)
	s, dir, fallen, pit := strandedAndRescuers(t, 3)
	want := pit.String() + " x6:"
	if got := automapMarkers(t, s, pit.Level); len(got) != 1 || !strings.HasPrefix(got[0], want) {
		t.Fatalf("before: %v", got)
	}
	v := s.DungeonAction(app.ActInspect, 0)
	minimapAgrees(t, s, v)
	for i := 0; i < 2; i++ {
		if v = s.DungeonAction(app.ActPickUp, 1); v.Error != "" {
			t.Fatal(v.Error)
		}
	}
	minimapAgrees(t, s, v)
	before := automapMarkers(t, s, pit.Level)
	if len(before) != 1 || !strings.HasPrefix(before[0], pit.String()+" x4:") || strings.Contains(before[0], fallen[0].Name+" ") {
		t.Fatalf("after two: %v", before)
	}
	beforeLocal := v.Minimap.Stranded

	s.shutdown()
	s = NewSession(dir)
	defer s.shutdown()
	v = s.State()
	if got := automapMarkers(t, s, pit.Level); !reflect.DeepEqual(got, before) {
		t.Fatalf("after the restart: %v, was %v", got, before)
	}
	if !reflect.DeepEqual(v.Minimap.Stranded, beforeLocal) {
		t.Fatalf("minimap after the restart: %+v, was %+v", v.Minimap.Stranded, beforeLocal)
	}
	if v.Dungeon.Inspect == nil || len(v.Dungeon.Inspect.Found) != 4 || v.Dungeon.Inspect.Elsewhere != 0 || !v.Dungeon.Inspect.Picked {
		t.Fatalf("Inspect after the restart: %+v", v.Dungeon.Inspect)
	}
	minimapAgrees(t, s, v)
}

// (12) A practice rescue picks up copies: the player's markers, read from
// the real game, do not change during practice or after it.
func TestPracticeLeavesTheMarkers(t *testing.T) {
	rand.Seed(54)
	s, dir := partySession(t, false)
	pit, _ := pitAhead(t, s)
	s.DungeonAction(app.ActForward, 0)
	helper := engine.NewCharacter("HELPER", engine.Human, engine.Fighter, engine.Neutral)
	helper.HP, helper.MaxHP = 50, 50
	rosterWith(t, s, helper)
	walkTo(t, s, "tavern")
	if v := s.JoinParty(6); v.PendingJoin != "" {
		s.SubmitJoinPassword("")
	}
	walkTo(t, s, "edge")
	real := func() []string {
		m, err := s.ctrl.Map(pit.Level)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, x := range m.Stranded {
			out = append(out, markerLine(x))
		}
		return out
	}
	want := real()
	saved, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if len(want) != 1 || !strings.HasPrefix(want[0], pit.String()+" x6:") {
		t.Fatalf("before: %v", want)
	}
	if v := s.PracticeExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	s.DungeonAction(app.ActInspect, 0)
	for i := 0; i < 5; i++ {
		if v := s.DungeonAction(app.ActPickUp, 1); v.Error != "" {
			t.Fatal(v.Error)
		}
	}
	if got := automapMarkers(t, s, pit.Level); len(got) != 1 || !strings.HasPrefix(got[0], pit.String()+" x1:") {
		t.Fatalf("the practice map: %v", got)
	}
	if got := real(); !reflect.DeepEqual(got, want) {
		t.Fatalf("during practice: %v", got)
	}
	s.AbandonPractice()
	if got := real(); !reflect.DeepEqual(got, want) {
		t.Fatalf("after practice: %v", got)
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, saved) {
		t.Fatal("practice changed the save")
	}
}
