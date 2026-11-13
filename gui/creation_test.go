package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Senior-Geek/istary/app"
)

// toPoints walks a new character to the bonus-point step: no password, an
// elf, the given alignment (0 good, 1 neutral, 2 evil).
func toPoints(t *testing.T, s *Session, name string, align int) View {
	t.Helper()
	walkTo(t, s, "training")
	mustOK(t, s.StartCreation(name), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	mustOK(t, s.SetPassword("", ""), "creation")
	mustOK(t, s.ChooseRace(1), "creation") // ELF
	v := mustOK(t, s.ChooseAlignment(align), "creation")
	if v.Creation.Step != "stats" || v.Creation.Points == 0 {
		t.Fatalf("not at the bonus points: %+v", v.Creation)
	}
	return v
}

// TestAClassIsChosenFromThePoints is UX-3B's one creation screen: the
// class is chosen at the bonus-point step (PickClass), by the controller's
// own rules, and a refused class leaves the points as they were.
func TestAClassIsChosenFromThePoints(t *testing.T) {
	s, dir := newTestSession(t)
	v := toPoints(t, s, "picker", 1)

	// Points left: no class yet, and the character is still at the points.
	if got := s.PickClass(1); got.Error != errorText(app.ErrPointsRemaining) || got.Creation.Step != "stats" ||
		got.Creation.Points != v.Creation.Points {
		t.Fatalf("with points left: error %q, creation %+v", got.Error, got.Creation)
	}

	v = spendAll(t, s, v, []int{1, 3, 0, 2, 4, 5})
	spent := *v.Creation

	// Every class the view does not allow is refused, and leaves the
	// points, the statistics and the step as they were.
	for class, ok := range spent.ClassOK {
		if ok {
			continue
		}
		got := s.PickClass(class)
		if got.Error != errorText(app.ErrClassUnavailable) {
			t.Errorf("class %d: error %q", class, got.Error)
		}
		if !reflect.DeepEqual(*got.Creation, spent) {
			t.Errorf("class %d refused, creation %+v, want %+v", class, *got.Creation, spent)
		}
	}

	// The points can still be moved after every one is spent.
	v = mustOK(t, s.RemovePoint(1), "creation")
	if v.Creation.Points != 1 || v.Creation.Step != "stats" {
		t.Fatalf("after taking a point back: %+v", v.Creation)
	}
	v = mustOK(t, s.AddPoint(1), "creation")
	if !reflect.DeepEqual(*v.Creation, spent) {
		t.Fatalf("after putting it back: %+v, want %+v", *v.Creation, spent)
	}

	// An allowed class goes straight to the review, with the same
	// statistics, and keeping it makes that character.
	if !spent.ClassOK[1] {
		t.Fatalf("an elf with I.Q. first cannot be a mage: %+v", spent)
	}
	v = mustOK(t, s.PickClass(1), "creation")
	if v.Creation.Step != "confirm" || v.Creation.Class != "MAGE" || v.Creation.Stats != spent.Stats ||
		v.Creation.Race != "ELF" || v.Creation.Alignment != spent.Alignment {
		t.Fatalf("review: %+v", v.Creation)
	}
	v = mustOK(t, s.Confirm(true), "roster")
	if len(v.Roster) != 1 || v.Roster[0].Name != "PICKER" || v.Roster[0].Class != "MAGE" || v.Roster[0].Stats != spent.Stats {
		t.Fatalf("roster = %+v", v.Roster)
	}
	if _, err := os.Stat(filepath.Join(dir, "roster1.json")); err != nil {
		t.Fatalf("not saved: %v", err)
	}
}

// TestPickClassIsTheTwoStepsTogether: the character PickClass makes is the
// one FinishStats and ChooseClass make, from the same points.
func TestPickClassIsTheTwoStepsTogether(t *testing.T) {
	review := func(pick bool) (CreationView, CharacterView) {
		rand.Seed(7)
		s, _ := newTestSession(t)
		defer s.shutdown()
		v := spendAll(t, s, toPoints(t, s, "twin", 1), []int{1, 3, 0, 2, 4, 5})
		if pick {
			v = mustOK(t, s.PickClass(1), "creation")
		} else {
			mustOK(t, s.FinishStats(), "creation")
			v = mustOK(t, s.ChooseClass(1), "creation")
		}
		r := mustOK(t, s.Confirm(true), "roster")
		return *v.Creation, r.Roster[0]
	}
	c1, ch1 := review(true)
	c2, ch2 := review(false)
	if !reflect.DeepEqual(c1, c2) {
		t.Errorf("review %+v, want %+v", c1, c2)
	}
	if !reflect.DeepEqual(ch1, ch2) {
		t.Errorf("character %+v, want %+v", ch1, ch2)
	}
}

// TestDiscardingACreationStaysAtTheTrainingGrounds: Discard is open at
// every step, keeps nothing and does not leave.
func TestDiscardingACreationStaysAtTheTrainingGrounds(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "first")
	before := readFile(t, filepath.Join(dir, "roster1.json"))

	steps := []func(){
		func() { s.StartCreation("gone") },                            // confirming
		func() { s.AnswerCreate(true) },                               // password
		func() { s.SetPassword("", "") },                              // race
		func() { s.ChooseRace(1) },                                    // alignment
		func() { s.ChooseAlignment(1) },                               // points
		func() { spendAll(t, s, s.State(), []int{1, 3, 0, 2, 4, 5}) }, // spent
		func() { s.PickClass(1) },                                     // review
	}
	for n := 1; n < len(steps); n++ {
		walkTo(t, s, "training")
		for _, step := range steps[:n+1] {
			step()
		}
		at := s.State()
		if at.Screen != "creation" || at.Error != "" {
			t.Fatalf("step %d: %q %q", n, at.Screen, at.Error)
		}
		v := mustOK(t, s.DiscardCreation(), "roster")
		if v.Location != "training" || v.Creation != nil || len(v.Roster) != 1 {
			t.Fatalf("after discarding at %s: location %q, creation %+v, roster %d", at.Creation.Step, v.Location, v.Creation, len(v.Roster))
		}
		if !bytes.Equal(readFile(t, filepath.Join(dir, "roster1.json")), before) {
			t.Fatalf("discarding at %s wrote the save", at.Creation.Step)
		}
	}
	if v := s.DiscardCreation(); v.Error == "" {
		t.Error("discarding with nothing being created was accepted")
	}
}

// creationView is what the Training Grounds read from a View.
type creationView struct {
	Case       string        `json:"case"`
	Screen     string        `json:"screen"`
	Message    string        `json:"message"`
	Error      string        `json:"error"`
	Creation   *CreationView `json:"creation"`
	StatNames  []string      `json:"statNames"`
	Classes    []string      `json:"classes"`
	Races      []string      `json:"races"`
	Alignments []string      `json:"alignments"`
}

// TestCreationViewFixture records the creation steps' views for the page's
// tests (frontend/src/lib/creation.test.js). Run with
// ISTARY_UPDATE_FIXTURES=1 to rewrite it.
func TestCreationViewFixture(t *testing.T) {
	rand.Seed(3)
	s, _ := newTestSession(t)
	var views []creationView
	rec := func(name string, v View) {
		views = append(views, creationView{name, v.Screen, v.Message, v.Error, v.Creation, v.StatNames, v.Classes, v.Races, v.Alignments})
	}
	walkTo(t, s, "training")
	rec("confirming", s.StartCreation("record"))
	rec("password", s.AnswerCreate(true))
	rec("race", s.SetPassword("", ""))
	rec("alignment", s.ChooseRace(1))
	v := s.ChooseAlignment(1)
	rec("points left", v)
	v = spendAll(t, s, v, []int{1, 3, 0, 2, 4, 5})
	rec("points spent", v)
	rec("class refused", s.PickClass(7))
	rec("class step", s.FinishStats())
	s.Back()
	rec("review", s.PickClass(1))
	s.DiscardCreation()
	walkTo(t, s, "training")
	s.StartCreation("evil")
	s.AnswerCreate(true)
	s.SetPassword("", "")
	s.ChooseRace(0)
	v = s.ChooseAlignment(2)
	rec("evil, points spent", spendAll(t, s, v, []int{0, 1, 2, 3, 4, 5}))
	s.DiscardCreation()
	s.shutdown()

	got, _ := json.MarshalIndent(views, "", " ")
	got = append(got, '\n')
	path := filepath.Join("frontend", "src", "lib", "testdata", "creation-views.json")
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
		t.Fatal("creation-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
