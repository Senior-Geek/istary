package main

import (
	"maps"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// departuresAgree checks that each departure the Edge of Town shows closed
// is refused by its action with the same words, and changes nothing saved
// (out on an expedition every action re-saves, so contents are compared).
func departuresAgree(t *testing.T, s *Session, dir, when string) {
	t.Helper()
	v := s.State()
	for _, d := range []struct {
		name, why string
		act       func() View
	}{
		{"real expedition", v.Departures.Real, s.RealExpedition},
		{"practice expedition", v.Departures.Practice, s.PracticeExpedition},
		{"practice encounter", v.Departures.Practice, func() View { return s.PracticeEncounter(false) }},
	} {
		if d.why == "" {
			continue
		}
		before := onDisk(t, dir)
		if got := d.act(); got.Error != d.why || !sameView(got, v) {
			t.Fatalf("%s, %s: shown %q, refused with %q", when, d.name, d.why, got.Error)
		}
		if !reflect.DeepEqual(before, onDisk(t, dir)) {
			t.Fatalf("%s, %s: the refusal changed the save", when, d.name)
		}
	}
}

// The departures shown at the Edge of Town are the actions' own answers:
// open with a party, closed without one, while out, and while an
// expedition is saved.
func TestDeparturesAreTheActionsAnswers(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	walkTo(t, s, "edge")
	if v := s.State(); v.Departures.Real != errNoParty.Error() || v.Departures.Practice != errNoParty.Error() {
		t.Fatalf("no party: %+v", v.Departures)
	}
	departuresAgree(t, s, dir, "no party")
	s.shutdown()

	rand.Seed(4)
	s, dir = partySession(t, false)
	if v := s.State(); v.Departures != (DeparturesView{}) {
		t.Fatalf("a party at the Edge of Town: %+v", v.Departures)
	}
	if v := s.PracticeExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	departuresAgree(t, s, dir, "practice")
	s.AbandonPractice()
	if v := s.RealExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	departuresAgree(t, s, dir, "out on an expedition")
	s.shutdown()

	dir = outAndRestarted(t)
	editSave(t, dir, func(st map[string]any) { st["expedition"].(map[string]any)["version"] = 2.0 })
	s = NewSession(dir)
	defer s.shutdown()
	walkTo(t, s, "edge")
	if v := s.State(); v.Departures.Real != errExpeditionSaved.Error() {
		t.Fatalf("an expedition saved: %+v", v.Departures)
	}
	departuresAgree(t, s, dir, "an expedition saved")
}

// Leave Game closes the window from the Edge of Town only, never while the
// party is out, a character is being created or the last save failed; it
// writes nothing.
func TestLeaveGame(t *testing.T) {
	rand.Seed(6)
	s, dir := partySession(t, false)
	quits := make(chan struct{}, 4)
	s.setQuit(func() { quits <- struct{}{} })
	refused := func(when, why string) {
		t.Helper()
		before := folder(t, dir)
		if v := s.LeaveGame(); v.Error != why {
			t.Fatalf("%s: %q", when, v.Error)
		}
		if !maps.Equal(before, folder(t, dir)) {
			t.Fatalf("%s: the save folder changed", when)
		}
		select {
		case <-quits:
			t.Fatalf("%s: the window closed", when)
		case <-time.After(20 * time.Millisecond):
		}
	}
	for _, place := range []string{"castle", "tavern", "training", "settings"} {
		walkTo(t, s, place)
		refused("at "+place, notAt(engine.EdgeOfTown).Error())
	}
	walkTo(t, s, "training")
	s.StartCreation("NEWCOMER")
	s.AnswerCreate(true)
	refused("creating", errCreating.Error())
	s.SetPassword("", "")
	s.Back()
	s.CancelCreation()

	walkTo(t, s, "edge")
	s.PracticeExpedition()
	refused("practice", errPracticeOut.Error())
	s.AbandonPractice()
	s.unsaved = "disk full"
	refused("unsaved", errUnsavedLeave.Error())
	s.unsaved = ""

	before := folder(t, dir)
	if v := s.LeaveGame(); v.Error != "" || v.Notice != "Leaving Istary." {
		t.Fatalf("at the Edge of Town: %q %q", v.Error, v.Notice)
	}
	select {
	case <-quits:
	case <-time.After(time.Second):
		t.Fatal("the window did not close")
	}
	if !maps.Equal(before, folder(t, dir)) {
		t.Fatal("leaving wrote to the save folder")
	}

	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	refused("out on an expedition", errExpeditionOut.Error())
}
