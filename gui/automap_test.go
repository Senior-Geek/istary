package main

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// The maps on a real expedition: the minimap follows the party from the
// entrance (L1:R19:C0, facing north), every action's discoveries are saved
// with it, and a restart brings the same map back.
func TestMapsOnARealExpedition(t *testing.T) {
	rand.Seed(3)
	s, dir := partySession(t, false)
	v := mustOK(t, s.RealExpedition(), "dungeon")
	if v.Minimap == nil || v.Minimap.Party.Place != "L1:R19:C0" || v.Minimap.Party.Facing != "N" || !v.Settings.MinimapVisible {
		t.Fatalf("minimap at the start %+v", v.Minimap)
	}
	if len(v.Minimap.Features) != 1 || v.Minimap.Features[0].Kind != engine.FeatureStairsUp {
		t.Fatalf("entrance features %+v", v.Minimap.Features)
	}
	mustOK(t, s.DungeonAction(app.ActLeaveCamp, 0), "dungeon")
	mustOK(t, s.DungeonAction(app.ActNo, 0), "dungeon")
	v = mustOK(t, s.DungeonAction(app.ActRight, 0), "dungeon")
	if v.Minimap.Party.Facing != "E" || len(v.Minimap.Walls) == 0 {
		t.Fatalf("after turning %+v", v.Minimap)
	}
	if st := onDisk(t, dir); len(st.Exploration) == 0 {
		t.Fatal("the exploration was not saved with the action")
	}
	want := s.ctrl.Game.Exploration.Clone()
	r := reopened(t, s, dir)
	rv := r.State()
	if !reflect.DeepEqual(r.ctrl.Game.Exploration, want) {
		t.Fatal("the map changed across a restart")
	}
	if !reflect.DeepEqual(rv.Minimap, v.Minimap) {
		t.Fatalf("minimap after restart\n%+v\nwas\n%+v", rv.Minimap, v.Minimap)
	}
}

// Looking at a map is not an action: no turn, no move, no random number,
// no save; the view is the same before and after.
func TestAutomapChangesNothing(t *testing.T) {
	rand.Seed(4)
	s, dir := partySession(t, false)
	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	path := filepath.Join(dir, "roster1.json")
	file, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	before := s.State()
	ex := s.ctrl.Game.Exploration.Clone()
	rand.Seed(99)
	m, err := s.Automap(0)
	if err != nil || m.Level != 1 || m.Party == nil || m.Party.Place != "L1:R19:C0" {
		t.Fatalf("automap %+v %v", m, err)
	}
	if _, err := s.Automap(4); err == nil {
		t.Fatal("an unexplored level was shown")
	}
	next := rand.Int63()
	rand.Seed(99)
	if rand.Int63() != next {
		t.Fatal("the automap drew a random number")
	}
	after := s.State()
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(ex, s.ctrl.Game.Exploration) {
		t.Fatal("the automap changed the game")
	}
	file2, _ := os.ReadFile(path)
	info2, _ := os.Stat(path)
	if !bytes.Equal(file, file2) || !info.ModTime().Equal(info2.ModTime()) {
		t.Fatal("the automap saved")
	}
}

// The map belongs to the player: it survives the castle, changing the
// party, and going back in, which adds to it.
func TestMapSurvivesTheCastleAndTheParty(t *testing.T) {
	rand.Seed(6)
	s, _ := partySession(t, false)
	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActRight, 0) // the stairs prompt: decline by turning is refused, so answer
	s.DungeonAction(app.ActNo, 0)
	s.DungeonAction(app.ActRight, 0)
	s.DungeonAction(app.ActForward, 0)
	walls := countSides(s.ctrl.Game.Exploration)
	// Back to the stairs and up to the castle.
	g := s.ctrl.Game
	g.PlayerX, g.PlayerY, g.Facing = 1, 0, engine.West
	s.DungeonAction(app.ActForward, 0)
	v := s.DungeonAction(app.ActYes, 0)
	if g.Phase != engine.PhaseTown || v.Minimap != nil {
		t.Fatalf("not back in the castle: %v %+v", g.Phase, v.Minimap)
	}
	m, err := s.Automap(0)
	if err != nil || m.Party != nil || countSides(g.Exploration) < walls {
		t.Fatalf("map from the castle %+v %v", m.Party, err)
	}
	// Another party, then the maze again.
	walkTo(t, s, "tavern")
	before := g.Exploration.Clone()
	if v := s.LeaveParty(5); v.Error != "" {
		t.Fatal(v.Error)
	}
	if !reflect.DeepEqual(before, g.Exploration) {
		t.Fatal("changing the party changed the map")
	}
	walkTo(t, s, "edge")
	mustOK(t, s.RealExpedition(), "dungeon")
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	s.DungeonAction(app.ActForward, 0)
	s.DungeonAction(app.ActForward, 0)
	if g.Phase != engine.PhaseMaze || g.PlayerY == 0 {
		t.Fatalf("did not walk north: %v %d,%d", g.Phase, g.PlayerX, g.PlayerY)
	}
	if countSides(g.Exploration) <= countSides(before) {
		t.Fatal("going back in added nothing")
	}
}

func countSides(e *engine.Exploration) int {
	n := 0
	for _, l := range e.LevelsSeen() {
		m := e.Level(l)
		for r := range m.Walls {
			for c := range m.Walls[r] {
				for _, w := range m.Walls[r][c] {
					if w != "" {
						n++
					}
				}
			}
		}
	}
	return n
}

// Practice starts from the player's map and keeps nothing of what it shows.
func TestPracticeKeepsNoMap(t *testing.T) {
	rand.Seed(8)
	s, dir := partySession(t, false)
	walkTo(t, s, "edge")
	v := mustOK(t, s.PracticeExpedition(), "dungeon")
	if v.Minimap == nil || v.Minimap.Party.Place != "L1:R19:C0" {
		t.Fatalf("practice minimap %+v", v.Minimap)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	s.DungeonAction(app.ActRight, 0)
	if m, err := s.Automap(0); err != nil || m.Party == nil {
		t.Fatalf("practice automap %v", err)
	}
	if s.ctrl.Game.Exploration != nil {
		t.Fatal("practice recorded into the player's map")
	}
	if st := onDisk(t, dir); len(st.Exploration) != 0 {
		t.Fatal("practice saved a map")
	}
	s.AbandonPractice()
	if _, err := s.Automap(0); err == nil {
		t.Fatal("practice's map outlived it")
	}
}

// An expedition saved before maps existed resumes with an empty map and
// records only what is seen from where the party stands.
func TestOldSaveResumesWithoutAMap(t *testing.T) {
	dir := outAndRestarted(t)
	editSave(t, dir, func(st map[string]any) { delete(st, "exploration") })
	s := NewSession(dir)
	defer s.shutdown()
	v := s.State()
	if !s.real || v.Minimap == nil {
		t.Fatalf("not resumed: %+v", v.Recovery)
	}
	e := s.ctrl.Game.Exploration
	m := e.Level(1)
	visited := 0
	for r := range m.Visited {
		for c := range m.Visited[r] {
			if m.Visited[r][c] {
				visited++
			}
		}
	}
	if len(e.LevelsSeen()) != 1 || visited != 1 || !m.Visited[19][0] {
		t.Fatalf("made-up exploration: %d visited", visited)
	}
}

// The minimap can be hidden; the choice is kept.
func TestMinimapSetting(t *testing.T) {
	s, dir := newTestSession(t)
	if v := s.SetMinimapVisible(false); v.Settings.MinimapVisible || !v.Settings.PartyVisible {
		t.Fatalf("%+v", v.Settings)
	}
	if r := reopened(t, s, dir); r.State().Settings.MinimapVisible {
		t.Fatal("not kept")
	}
	if v := s.SetMinimapVisible(true); !v.Settings.MinimapVisible {
		t.Fatal("not shown again")
	}
}
