// The game draws from the global math/rand source. Go 1.24+ makes rand.Seed a
// no-op by default; this test-only directive restores seeding so tests can
// replay a fixed random sequence without touching production code.
//go:debug randseednop=0

package app

import (
	"math/rand"
	"os"
	"testing"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
	"github.com/Senior-Geek/istary/scenarios/wiz1"
	"github.com/Senior-Geek/istary/scenarios/wiz2"
	"github.com/Senior-Geek/istary/scenarios/wiz3"
)

// TestMain points HOME at a throwaway directory so no test can touch the
// player's real ~/.config/wizardry saves.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "istary-app-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

func seed(n int64) { rand.Seed(n) }

// newController returns a controller for a scenario, in town with the title
// screen skipped.
func newController(t *testing.T, num int) *Controller {
	t.Helper()
	loaders := map[int]func() (*data.Scenario, error){1: wiz1.Load, 2: wiz2.Load, 3: wiz3.Load}
	s, err := loaders[num]()
	if err != nil {
		t.Fatal(err)
	}
	g := engine.New(s)
	g.Title = nil
	g.Phase = engine.PhaseTown
	return New(g)
}

// keys sends each rune through HandleKey and fails if one is not handled.
func keys(t *testing.T, c *Controller, s string) {
	t.Helper()
	for _, r := range s {
		if !c.HandleKey(Rune(r)) {
			t.Fatalf("key %q not handled (phase %v)", r, c.Game.Phase)
		}
	}
}

func press(t *testing.T, c *Controller, kind KeyKind) {
	t.Helper()
	if !c.HandleKey(Key{Kind: kind}) {
		t.Fatalf("key %v not handled (phase %v)", kind, c.Game.Phase)
	}
}

func member(name string, align engine.Alignment) *engine.Character {
	ch := engine.NewCharacter(name, engine.Human, engine.Fighter, align)
	ch.HP, ch.MaxHP = 10, 10
	return ch
}
