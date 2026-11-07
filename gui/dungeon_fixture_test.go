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

// The front end's tests (frontend/src/lib/dungeonViews.test.js) run its
// display helpers over real views. This records the distinct dungeon views
// of seeded practice expeditions, enough to cover Camp, the walk and its
// prompts. Run with ISTARY_UPDATE_FIXTURES=1 to rewrite the file after a
// deliberate change.
func TestDungeonViewFixture(t *testing.T) {
	var views []*app.DungeonView
	have := map[string]bool{}
	kinds := map[string]bool{}
	for seed := int64(1); seed <= 40 && len(kinds) < 7; seed++ {
		rand.Seed(seed)
		s, _ := partySession(t, false)
		s.PracticeExpedition()
		explore(t, s, rand.New(rand.NewSource(seed)), map[string]int{}, func(v View) {
			d := v.Dungeon
			if d == nil || len(views) >= 80 {
				return
			}
			raw, _ := json.Marshal(d)
			kind := d.Phase + "/" + d.Prompt
			if have[string(raw)] || (kinds[kind] && len(views) >= 40) {
				return
			}
			have[string(raw)], kinds[kind] = true, true
			views = append(views, d)
		})
		s.AbandonPractice()
	}
	for _, k := range []string{"camp/", "maze/", "maze/stairs", "maze/messages", "maze/inspect"} {
		if !kinds[k] {
			t.Fatalf("the recording has no %s view", k)
		}
	}
	got, _ := json.Marshal(views)
	path := filepath.Join("frontend", "src", "lib", "testdata", "dungeon-views.json")
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
		t.Fatal("dungeon-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
