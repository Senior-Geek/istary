package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/Senior-Geek/istary/app"
)

// The front end's tests (frontend/src/lib/combatViews.test.js) run its
// display helpers over real views. This records those views: one seeded
// practice encounter with a chest, driven through the bridge. Run with
// ISTARY_UPDATE_FIXTURES=1 to rewrite the file after a deliberate change.
func TestCombatViewFixture(t *testing.T) {
	var views []*app.CombatView
	for seed := int64(1); seed <= 400; seed++ {
		rand.Seed(seed)
		s, _ := partySession(t, false)
		s.PracticeEncounter(true)
		_, seen := drive(t, s, rand.New(rand.NewSource(seed)))
		var vs []*app.CombatView
		chest, rounds := false, 0
		for _, v := range seen {
			if v.Combat != nil {
				vs = append(vs, v.Combat)
				chest = chest || v.Combat.Phase == "chest"
				if v.Combat.Next == app.NextEndRound {
					rounds++
				}
			}
		}
		if chest && rounds >= 2 && len(vs) <= 120 { // a short fight of two rounds or more that reached the chest
			views = vs
			break
		}
	}
	if views == nil {
		t.Fatal("no seed reached a chest after two rounds")
	}
	got, _ := json.Marshal(views)
	path := filepath.Join("frontend", "src", "lib", "testdata", "combat-views.json")
	if os.Getenv("ISTARY_UPDATE_FIXTURES") != "" {
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with ISTARY_UPDATE_FIXTURES=1)", err)
	}
	if !bytes.Equal(append(got, '\n'), want) {
		t.Fatal("combat-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
