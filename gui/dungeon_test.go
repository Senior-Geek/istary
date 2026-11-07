package main

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// explore plays a practice expedition as a front end would, reading only
// the View and calling only Session methods: in the dungeon it picks one of
// the view's actions (mostly forward), and in combat it fights with drive.
// It checks at every view that the screen matches the practice game: the
// combat screen exactly when the engine has started an encounter.
func explore(t *testing.T, s *Session, rng *rand.Rand, seen map[string]int) View {
	t.Helper()
	v := s.State()
	for step := 0; step < 1500; step++ {
		if v.Error != "" {
			t.Fatalf("step %d: %s", step, v.Error)
		}
		if s.practice == nil {
			return v
		}
		g := s.practice.Game
		engineCombat := g.Phase == engine.PhaseCombat && g.Combat != nil
		if (v.Screen == "combat") != engineCombat || (v.Screen == "dungeon") == engineCombat {
			t.Fatalf("step %d: screen %q with engine phase %v", step, v.Screen, g.Phase)
		}
		if (v.Combat != nil) != (v.Screen == "combat") || (v.Dungeon != nil) != (v.Screen == "dungeon") || !v.Expedition {
			t.Fatalf("step %d: screen %q combat %v dungeon %v expedition %v", step, v.Screen, v.Combat != nil, v.Dungeon != nil, v.Expedition)
		}
		if v.Screen == "combat" {
			seen["encounter"]++
			_, views := drive(t, s, rng)
			v = views[len(views)-1]
			if v.Screen == "dungeon" {
				seen["back in the maze after combat"]++
			}
			continue
		}
		d := v.Dungeon
		seen["prompt "+d.Prompt]++
		act := d.Actions[rng.Intn(len(d.Actions))]
		switch {
		case d.Phase == "maze" && d.Prompt == "" && rng.Intn(3) > 0:
			act = app.ActForward
		case d.Prompt == "stairs" && rng.Intn(8) > 0:
			act = app.ActNo // mostly walk on
		case act == app.ActCamp:
			seen["camp"]++
		}
		button := 0
		if act == app.ActButton {
			button = 1 + rng.Intn(len(d.Buttons))
		}
		v = s.DungeonAction(act, button)
	}
	return v
}

func TestPracticeExpeditionsThroughTheBridge(t *testing.T) {
	seen := map[string]int{}
	for seed := int64(1); seed <= 60; seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir := partySession(t, false)
		before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
		party := s.State().Party

		v := s.PracticeExpedition()
		if v.Screen != "dungeon" || v.Dungeon == nil || v.Dungeon.Phase != "camp" || !slices.Equal(v.Dungeon.Actions, []string{app.ActLeaveCamp}) {
			t.Fatalf("seed %d: start: %+v", seed, v)
		}
		v = explore(t, s, rng, seen)
		switch {
		case s.practice != nil:
			seen["still walking"]++
			v = s.AbandonPractice()
			if !strings.Contains(v.Notice, "expedition abandoned") {
				t.Fatalf("abandon: %q", v.Notice)
			}
		case strings.Contains(v.Notice, "perished"):
			seen["perished"]++
		case strings.Contains(v.Notice, "back in the castle"):
			seen["climbed out"]++
		default:
			t.Fatalf("seed %d: ended with %q", seed, v.Notice)
		}
		if v.Screen != "roster" || v.Dungeon != nil || v.Combat != nil || v.Expedition || s.practice != nil {
			t.Fatalf("seed %d: after the expedition: %+v", seed, v)
		}
		after, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
		if !bytes.Equal(before, after) || !reflect.DeepEqual(s.State().Party, party) {
			t.Fatalf("seed %d: the practice expedition changed the roster", seed)
		}
	}
	t.Logf("covered: %v", seen)
	for _, k := range []string{"encounter", "back in the maze after combat", "prompt ", "prompt stairs", "camp", "perished", "climbed out"} {
		if seen[k] == 0 {
			t.Errorf("not covered: %s", k)
		}
	}
}

func TestExpeditionRefusals(t *testing.T) {
	s, _ := newTestSession(t)
	if v := s.PracticeExpedition(); v.Error == "" || s.practice != nil {
		t.Fatal("an expedition with no party started")
	}
	if v := s.DungeonAction(app.ActForward, 0); v.Error == "" {
		t.Fatal("a dungeon action with no expedition succeeded")
	}

	s, _ = partySession(t, false)
	s.PracticeEncounter(false)
	if v := s.DungeonAction(app.ActForward, 0); v.Error == "" {
		t.Fatal("a dungeon action during a practice encounter succeeded")
	}
	s.AbandonPractice()

	v := s.PracticeExpedition()
	if v.Error != "" {
		t.Fatal(v.Error)
	}
	if v := s.PracticeExpedition(); v.Error == "" {
		t.Fatal("a second expedition started")
	}
	if v := s.PracticeEncounter(false); v.Error == "" {
		t.Fatal("a practice encounter started during an expedition")
	}
	snap := func() string {
		return s.practice.DungeonView().Phase + strings.Join(s.practice.DungeonView().Lines, "/")
	}
	before := snap()
	for name, op := range map[string]func() View{
		"forward at camp": func() View { return s.DungeonAction(app.ActForward, 0) },
		"nonsense":        func() View { return s.DungeonAction("teleport", 0) },
		"combat start":    s.CombatStart,
		"combat leave":    s.CombatLeave,
	} {
		if v := op(); v.Error == "" || v.Screen != "dungeon" {
			t.Errorf("%s: error %q screen %q", name, v.Error, v.Screen)
		}
	}
	if snap() != before {
		t.Fatal("refused actions changed the expedition")
	}
	v = s.DungeonAction(app.ActLeaveCamp, 0)
	if v.Dungeon.Prompt != "stairs" {
		t.Fatalf("leaving camp at the entrance: %+v", v.Dungeon)
	}
	v = s.AbandonPractice()
	if v.Screen != "roster" || s.practice != nil || v.Expedition {
		t.Fatalf("abandon: %+v", v)
	}
}
