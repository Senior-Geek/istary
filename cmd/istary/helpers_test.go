// The game draws from the global math/rand source. Go 1.24+ makes rand.Seed a
// no-op by default; this test-only directive restores seeding so regression
// tests can replay a fixed random sequence without touching production code.
//go:debug randseednop=0

package main

import (
	"bytes"
	"flag"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
	"github.com/Senior-Geek/istary/render"
	"github.com/gdamore/tcell/v2"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/")

// TestMain points HOME at a throwaway directory so no test can touch the
// player's real ~/.config/wizardry saves.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "istary-main-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

func seed(n int64) { rand.Seed(n) }

// newGame loads a scenario and skips the title screen, as pressing S at the
// title menu does.
func newGame(t *testing.T, num int) *engine.GameState {
	t.Helper()
	s, err := loadScenario([]string{"", "1", "2", "3"}[num])
	if err != nil {
		t.Fatal(err)
	}
	g := engine.New(s)
	g.Title = nil
	g.Phase = engine.PhaseTown
	return g
}

// driver feeds key events through the same per-phase dispatch as the main
// event loop. Keep dispatch in sync with main() until the controller is
// extracted.
type driver struct {
	t      *testing.T
	game   *engine.GameState
	screen *simScreen // nil unless the test needs one
	quit   bool
}

func (d *driver) dispatch(ev *tcell.EventKey) {
	g := d.game
	switch g.Phase {
	case engine.PhaseTown:
		if handleTownInput(g, ev) {
			d.quit = true
		}
	case engine.PhaseCamp:
		handleCampInput(g, ev)
	case engine.PhaseMaze:
		if d.screen == nil {
			d.t.Fatal("maze input needs a screen; use newDriverWithScreen")
		}
		if handleMazeInput(d.screen.Screen, g, ev) {
			d.quit = true
		}
	case engine.PhaseCombat:
		handleCombatInput(g, ev)
	case engine.PhaseUtilities:
		handleUtilInput(g, ev)
	case engine.PhaseCreation:
		handleCreationInput(g, ev)
	default:
		d.t.Fatalf("dispatch: unsupported phase %v", g.Phase)
	}
}

// keys sends each rune as a key press.
func (d *driver) keys(s string) *driver {
	for _, r := range s {
		d.dispatch(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	return d
}

func (d *driver) key(k tcell.Key) *driver {
	d.dispatch(tcell.NewEventKey(k, 0, tcell.ModNone))
	return d
}

func (d *driver) enter() *driver { return d.key(tcell.KeyEnter) }
func (d *driver) esc() *driver   { return d.key(tcell.KeyEscape) }

func newDriver(t *testing.T, g *engine.GameState) *driver {
	return &driver{t: t, game: g}
}

func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run go test -update to create)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs from golden file.\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func character(name string, race engine.Race, class engine.Class, align engine.Alignment) *engine.Character {
	c := engine.NewCharacter(name, race, class, align)
	c.HP, c.MaxHP = 10, 10
	c.Strength, c.IQ, c.Piety, c.Vitality, c.Agility, c.Luck = 15, 10, 10, 12, 10, 10
	c.AC = 10
	return c
}

// simScreen is an 80x24 in-memory terminal for driver and snapshot tests.
type simScreen struct {
	*render.Screen
	sim tcell.SimulationScreen
}

func newSimScreen(t *testing.T) *simScreen {
	t.Helper()
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	sim.SetSize(80, 24)
	t.Cleanup(sim.Fini)
	return &simScreen{Screen: render.NewScreenFrom(sim), sim: sim}
}

// lines returns the visible text, one string per row. The renderer draws
// 40 logical columns at 2x scale, so only even physical columns hold text.
func (s *simScreen) lines() []string {
	s.Show()
	cells, w, h := s.sim.GetContents()
	out := make([]string, h)
	for y := 0; y < h; y++ {
		var row []rune
		for x := 0; x < w; x += 2 {
			r := ' '
			if c := cells[y*w+x]; len(c.Runes) > 0 && c.Runes[0] != 0 {
				r = c.Runes[0]
			}
			row = append(row, r)
		}
		out[y] = strings.TrimRight(string(row), " ")
	}
	return out
}

func (s *simScreen) text() string { return strings.Join(s.lines(), "\n") }

func newDriverWithScreen(t *testing.T, g *engine.GameState) *driver {
	return &driver{t: t, game: g, screen: newSimScreen(t)}
}
