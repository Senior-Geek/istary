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

// frameView is what the location frame and the party bar read from a View.
type frameView struct {
	Case       string         `json:"case"`
	Screen     string         `json:"screen"`
	Location   string         `json:"location"`
	Exits      []ExitView     `json:"exits"`
	PartyBar   []PartyBarSlot `json:"partyBar"`
	FrontRow   int            `json:"frontRow"`
	Settings   SettingsView   `json:"settings"`
	Departures DeparturesView `json:"departures"`
	Expedition bool           `json:"expedition"`
	Real       bool           `json:"real"`
}

func frameOf(name string, v View) frameView {
	return frameView{name, v.Screen, v.Location, v.Exits, v.PartyBar, v.FrontRow, v.Settings, v.Departures, v.Expedition, v.Real}
}

// TestFrameViewFixture records the frame's part of real views, at every
// location and in every kind of party, for the page's tests
// (frontend/src/lib/frameViews.test.js). Run with ISTARY_UPDATE_FIXTURES=1
// to rewrite it.
func TestFrameViewFixture(t *testing.T) {
	var views []frameView
	s, _ := newTestSession(t)
	rosterWith(t, s, hero("LEAD", engine.Good, ""), hero("SECOND", engine.Good, ""))
	views = append(views, frameOf("castle, no party", s.State()))
	join(t, s, 0, "")
	join(t, s, 1, "")
	s.ctrl.Game.Town.Party.Members[1].PoisonAmt = 3
	s.ctrl.Game.Town.Party.Members[1].HP = 2
	for _, place := range []string{"castle", "tavern", "edge", "training", "settings"} {
		views = append(views, frameOf(place, walkTo(t, s, place)))
	}
	s.SetPartyVisible(false)
	views = append(views, frameOf("party hidden", walkTo(t, s, "castle")))
	s.SetPartyVisible(true)
	walkTo(t, s, "training")
	s.StartCreation("NEWCOMER")
	views = append(views, frameOf("confirming a new character", s.State()))
	s.AnswerCreate(true)
	views = append(views, frameOf("creating", s.State()))
	s.SetPassword("", "")
	s.Back()
	s.CancelCreation()
	s.shutdown()

	rand.Seed(2)
	s, _ = partySession(t, false)
	s.PracticeExpedition()
	s.practice.Game.Town.Party.Members[0].HP = 1
	views = append(views, frameOf("practice expedition", s.State()))
	s.AbandonPractice()
	s.RealExpedition()
	views = append(views, frameOf("real expedition", s.DungeonAction(app.ActLeaveCamp, 0)))
	s.shutdown()

	got, _ := json.MarshalIndent(views, "", " ")
	got = append(got, '\n')
	path := filepath.Join("frontend", "src", "lib", "testdata", "frame-views.json")
	if os.Getenv("ISTARY_UPDATE_FIXTURES") != "" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with ISTARY_UPDATE_FIXTURES=1)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("frame-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
