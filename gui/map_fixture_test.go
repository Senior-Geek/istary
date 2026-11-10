package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// mapCase is a recorded map: the minimap and the full map of one moment.
type mapCase struct {
	Case  string            `json:"case"`
	Local *app.LocalMapView `json:"local"`
	Full  *app.MapView      `json:"full"`
}

// The front end's map tests (frontend/src/lib/automap.test.js) draw real
// map views. This records them: the entrance as a practice expedition
// shows it, then levels 1 and 2 after seeded walks with and without light
// (the controller's own exploration record), from places beside the
// level's edges, with stairs, the elevator, doors, one-way walls and ways
// across the edge. Run with ISTARY_UPDATE_FIXTURES=1 to rewrite the file
// after a deliberate change.
func TestMapViewFixture(t *testing.T) {
	rand.Seed(21)
	s, _ := partySession(t, false)
	v := mustOK(t, s.PracticeExpedition(), "dungeon")
	full, err := s.Automap(0)
	if err != nil {
		t.Fatal(err)
	}
	cases := []mapCase{{Case: "entrance", Local: v.Minimap, Full: &full}}
	c := s.practice
	g := c.Game
	g.Phase = engine.PhaseMaze
	rng := rand.New(rand.NewSource(21))
	stand := func(level, x, y int, f engine.Direction) {
		g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = level, x, y, f
		c.Explore()
	}
	for level := 0; level < 2; level++ {
		stand(level, rng.Intn(20), rng.Intn(20), engine.North)
		for i := 0; i < 6000; i++ {
			g.LightLevel = map[bool]int{true: 30}[i%1000 > 400]
			switch r := rng.Intn(10); {
			case r < 6:
				if !g.MoveForward() && g.KickDoor() && rng.Intn(2) == 0 {
					g.PlayerX, g.PlayerY = engine.Shift(g.PlayerX, g.PlayerY, g.Facing, 0, 1)
				}
			case r < 8:
				g.TurnLeft()
			default:
				g.TurnRight()
			}
			c.Explore()
		}
	}
	// Both sides of a wall that is a door one way (L1:R19:C5 / R18:C5), and
	// of one across the north edge (L1:R0:C1 / R19:C1).
	stand(0, 5, 0, engine.North)
	stand(0, 5, 1, engine.South)
	stand(0, 1, 19, engine.North)
	stand(0, 1, 0, engine.South)
	// The features, stood on: stairs up and down and the elevator on L1,
	// the stairs on L2.
	stand(0, 0, 0, engine.North)
	stand(0, 0, 10, engine.North)
	stand(0, 10, 8, engine.South)
	stand(1, 12, 7, engine.North)
	stand(1, 16, 15, engine.North)
	record := func(name string, level, x, y int, f engine.Direction) {
		stand(level, x, y, f)
		lv, ok := c.LocalMap()
		full, err := c.Map(0)
		if !ok || err != nil {
			t.Fatalf("%s: %v %v", name, ok, err)
		}
		cases = append(cases, mapCase{Case: name, Local: &lv, Full: &full})
	}
	record("L1 entrance after the walks", 0, 0, 0, engine.North)
	record("L1 north edge", 0, 10, 19, engine.North)
	record("L1 one-way door", 0, 5, 1, engine.South)
	record("L1 elevator", 0, 10, 8, engine.South)
	record("L2 east edge", 1, 19, 2, engine.East)
	record("L2 west edge", 1, 0, 3, engine.West)
	record("L2 stairs", 1, 12, 7, engine.West)

	// The recording must cover what the maps draw.
	has := map[string]bool{}
	for _, mc := range cases {
		for _, w := range mc.Full.Walls {
			has["full "+w.Kind] = true
			if w.OneWay {
				has["full one way"] = true
			}
		}
		for _, w := range mc.Full.Wraps {
			has["wrap "+w.Kind] = true
			if w.OneWay {
				has["wrap one way"] = true
			}
		}
		for _, f := range mc.Full.Features {
			has[f.Kind] = true
		}
		for _, w := range mc.Local.Walls {
			has["local "+w.Kind] = true
			if w.OneWay {
				has["local one way"] = true
			}
		}
		if mc.Local.NorthEdge >= 0 {
			has["local north edge"] = true
		}
		if mc.Local.WestEdge >= 0 {
			has["local west edge"] = true
		}
		has["facing "+mc.Local.Party.Facing] = true
	}
	for _, k := range []string{"full wall", "full door", "full one way", "wrap open", "wrap one way", "local one way", "local wall", "local door",
		"local north edge", "local west edge", engine.FeatureStairsUp, engine.FeatureStairsDown, engine.FeatureElevator,
		"facing N", "facing E", "facing S", "facing W"} {
		if !has[k] {
			t.Errorf("the recording has no %s", k)
		}
	}
	got, _ := json.Marshal(cases)
	path := filepath.Join("frontend", "src", "lib", "testdata", "map-views.json")
	if os.Getenv("ISTARY_UPDATE_FIXTURES") != "" {
		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with ISTARY_UPDATE_FIXTURES=1)", err)
	}
	if !bytes.Equal(append(got, '\n'), want) {
		t.Fatal("map-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
	s.AbandonPractice()
}
