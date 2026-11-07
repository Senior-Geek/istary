package app

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// Camp saved at any point (its menu or any of its screens, mid-equip,
// mid-trade, a spell waiting for its target, DUMAPIC, MALOR's screen),
// reloaded from disk and resumed, goes on exactly as uninterrupted play:
// same states, same random numbers.
func TestCampResumeFollowsUninterruptedPlay(t *testing.T) {
	orig := pristine(t)
	screens := map[string]int{}
	for s := int64(1); s <= 30; s++ {
		const steps = 250
		run := func(resume bool) []string {
			seed(s)
			rng := rand.New(rand.NewSource(s))
			c := campers(t, rng)
			var trace []string
			for i := 0; i < steps && c.Game.Phase == engine.PhaseCamp; i++ {
				c.press(campKeyFor(c.Game, rng)...)
				if resume && c.Game.Phase == engine.PhaseCamp {
					if v, err := c.CampView(); err == nil {
						screens[v.Screen]++
					}
					c = reload(t, c, orig)
				}
				trace = append(trace, campTrace(c.Game))
			}
			return append(trace, fmt.Sprint("rng ", rand.Int63()))
		}
		want, got := run(false), run(true)
		for i := range want {
			if i >= len(got) || got[i] != want[i] {
				t.Fatalf("seed %d step %d: resumed\n%s\nuninterrupted\n%s", s, i, got[min(i, len(got)-1)], want[i])
			}
		}
	}
	t.Logf("resumed at: %v", screens)
	for _, k := range []string{"menu", "reorder", "inspect", "equip", "drop", "trade", "tradeGold", "tradeItem",
		"spellBooks", "spellList", "castSpell", "spellTarget", "dumapic", "malor"} {
		if screens[k] == 0 {
			t.Errorf("never resumed at %s", k)
		}
	}
}

// An equip interrupted half-way resumes on the same screen and category
// with the same items unequipped and the same AC, and finishes as it
// would have.
func TestInterruptedEquipResumesExactly(t *testing.T) {
	orig := pristine(t)
	for s := int64(1); s <= 20; s++ {
		seed(s)
		rng := rand.New(rand.NewSource(s))
		c := campers(t, rng)
		must(t, c.CampInspect(0))
		must(t, c.BeginEquip())
		if c.Game.Town.InputMode != engine.InputEquip {
			continue // nothing to equip
		}
		before := campTrace(c.Game)
		c2 := reload(t, c, orig)
		if campTrace(c2.Game) != before {
			t.Fatalf("seed %d: resumed\n%s\nwant\n%s", s, campTrace(c2.Game), before)
		}
		for c2.Game.Town.InputMode == engine.InputEquip {
			must(t, c2.EquipChoose(1))
			must(t, c.EquipChoose(1))
		}
		if campTrace(c2.Game) != campTrace(c.Game) {
			t.Fatalf("seed %d: finished differently", s)
		}
		return
	}
	t.Fatal("no seed had anything to equip")
}
