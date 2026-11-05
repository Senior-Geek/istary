// The game draws from the global math/rand source. Go 1.24+ makes rand.Seed a
// no-op by default; this test-only directive restores seeding so regression
// tests can replay a fixed random sequence without touching production code.
//go:debug randseednop=0

package engine

import (
	"bytes"
	"flag"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/scenarios/wiz1"
	"github.com/Senior-Geek/istary/scenarios/wiz2"
	"github.com/Senior-Geek/istary/scenarios/wiz3"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/")

// TestMain points HOME at a throwaway directory so no test can touch the
// player's real ~/.config/wizardry saves.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "istary-engine-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

// seed makes the global math/rand sequence deterministic for one test.
func seed(n int64) {
	rand.Seed(n)
}

// freshHome gives a test its own empty HOME directory.
func freshHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// loadScenario loads scenario 1, 2 or 3. Each call returns a fresh copy so
// tests that mutate maze or item data cannot affect each other.
func loadScenario(t *testing.T, num int) *data.Scenario {
	t.Helper()
	loaders := map[int]func() (*data.Scenario, error){1: wiz1.Load, 2: wiz2.Load, 3: wiz3.Load}
	s, err := loaders[num]()
	if err != nil {
		t.Fatalf("load scenario %d: %v", num, err)
	}
	return s
}

// checkGolden compares got with testdata/<name>; run `go test -update` to
// regenerate after an intentional change.
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
