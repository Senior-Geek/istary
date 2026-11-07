package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/data"
)

// rosterFile reads the saved roster file.
func rosterFile(t *testing.T, home string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".config", "wizardry", "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// A game with a roster of three, two of them out on an expedition.
func expeditionGame(t *testing.T) (*GameState, string) {
	t.Helper()
	home := freshHome(t)
	g := newTestGame(t, 1)
	for _, name := range []string{"ALDRIC", "BRIN", "CARA"} {
		c := sampleCharacter()
		c.Name = name
		g.Town.Roster.Add(c)
	}
	g.Town.Party.Members = g.Town.Roster.Characters[:2]
	g.Expedition = &Expedition{
		Version: ExpeditionVersion,
		Party:   []string{"ALDRIC", "BRIN"},
		Where: Checkpoint{Phase: "maze", Level: 2, X: 5, Y: 9, Facing: East, Light: 12,
			FightMap: slices.Repeat([]string{strings.Repeat("0", 20)}, 20), SearchYN: true, SearchHere: true},
		Interrupted: "combat",
		Squares:     []SquareState{{Level: 1, X: 3, Y: 4, Type: data.SqNormal, Count: 0, SpclMonster: 7}},
	}
	return g, home
}

// Without an expedition the file is exactly as before (the save golden,
// TestSaveFileFormat, pins it); with one, the section is written, the
// members are not in the castle party list, and loading returns it as saved
// with no castle party.
func TestExpeditionSectionRoundTrip(t *testing.T) {
	g, home := expeditionGame(t)
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	var st SaveState
	if err := json.Unmarshal(rosterFile(t, home), &st); err != nil {
		t.Fatal(err)
	}
	if len(st.PartyNames) != 0 || len(st.Expedition) == 0 {
		t.Fatalf("party %v, expedition %d bytes", st.PartyNames, len(st.Expedition))
	}
	g2 := newTestGame(t, 1)
	if err := g2.Load(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g2.Expedition, g.Expedition) || g2.OtherExpedition != nil {
		t.Fatalf("loaded %+v\nwant %+v", g2.Expedition, g.Expedition)
	}
	if g2.Town.Party.Size() != 0 || !g2.OnExpedition("BRIN") || g2.OnExpedition("CARA") {
		t.Fatal("expedition members in the castle party, or membership wrong")
	}

	// The castle party is written beside it: CARA in the castle.
	g2.Town.Party.Members = []*Character{g2.Town.Roster.Characters[2]}
	if err := g2.Save(); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(rosterFile(t, home), &st)
	if !slices.Equal(st.PartyNames, []string{"CARA"}) {
		t.Fatalf("castle party %v", st.PartyNames)
	}

	// Ending the expedition removes the section: the file is as before.
	g2.Expedition = nil
	g2.Town.Party.Members = g2.Town.Roster.Characters[:2]
	if err := g2.Save(); err != nil {
		t.Fatal(err)
	}
	if raw := rosterFile(t, home); bytes.Contains(raw, []byte(`"expedition"`)) {
		t.Fatal("the section stayed after the expedition ended")
	}
}

// A section of another version (a newer Istary) is kept and written back,
// and its members stay out of the castle party list; a front end that does
// not resume expeditions (the terminal) never drops one.
func TestOtherExpeditionVersionIsKept(t *testing.T) {
	g, home := expeditionGame(t)
	g.Expedition = nil
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	var st map[string]json.RawMessage
	json.Unmarshal(rosterFile(t, home), &st)
	other := `{"version":2,"party":["ALDRIC"],"somethingNew":{"a":[1,2,3]}}`
	st["expedition"] = json.RawMessage(other)
	st["party"] = json.RawMessage(`["BRIN"]`)
	raw, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(home, ".config", "wizardry", "roster1.json"), raw, 0o644)

	g2 := newTestGame(t, 1)
	if err := g2.Load(); err != nil {
		t.Fatal(err)
	}
	if g2.Expedition != nil || g2.OtherExpedition == nil || !g2.OnExpedition("ALDRIC") || g2.OnExpedition("BRIN") {
		t.Fatalf("other version: expedition %v other %s", g2.Expedition, g2.OtherExpedition)
	}
	if err := g2.Save(); err != nil {
		t.Fatal(err)
	}
	var back SaveState
	json.Unmarshal(rosterFile(t, home), &back)
	var a, b any
	json.Unmarshal(back.Expedition, &a)
	json.Unmarshal([]byte(other), &b)
	if !reflect.DeepEqual(a, b) || !slices.Equal(back.PartyNames, []string{"BRIN"}) {
		t.Fatalf("written back: %s party %v", back.Expedition, back.PartyNames)
	}
}

// Old saves (no section) load with no expedition; an unreadable section is
// an error, not silently dropped.
func TestExpeditionSectionCompatibility(t *testing.T) {
	home := freshHome(t)
	dir := filepath.Join(home, ".config", "wizardry")
	os.MkdirAll(dir, 0o755)
	golden, _ := os.ReadFile(filepath.Join("testdata", "save_roster1.golden.json"))
	os.WriteFile(filepath.Join(dir, "roster1.json"), golden, 0o644)
	g := newTestGame(t, 1)
	if err := g.Load(); err != nil || g.Expedition != nil || g.OtherExpedition != nil {
		t.Fatalf("old save: %v %v", err, g.Expedition)
	}
	bad := bytes.Replace(golden, []byte(`"party": [`), []byte(`"expedition": {"version": "one"}, "party": [`), 1)
	os.WriteFile(filepath.Join(dir, "roster1.json"), bad, 0o644)
	if err := newTestGame(t, 1).Load(); err == nil {
		t.Fatal("an unreadable expedition section loaded without an error")
	}
}

// Changed squares: found against the scenario as loaded, and restored onto
// any copy of it (including one changed elsewhere); a square outside the
// maze is refused before anything changes.
func TestChangedSquaresRoundTrip(t *testing.T) {
	orig, sc, other := loadScenario(t, 1), loadScenario(t, 1), loadScenario(t, 1)
	if len(ChangedSquares(orig, sc)) != 0 {
		t.Fatal("a fresh scenario has changes")
	}
	sc.Mazes.Levels[0].Cells[5][13].SpclMonster = 77
	sc.Mazes.Levels[3].Cells[15][10].Count = 1
	sc.Mazes.Levels[3].Cells[15][10].Type = data.SqNormal
	other.Mazes.Levels[2].Cells[1][1].Count = 99 // a change that must be undone
	changes := ChangedSquares(orig, sc)
	if len(changes) != 2 {
		t.Fatalf("changes %v", changes)
	}
	if err := RestoreSquares(orig, other, changes); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ChangedSquares(orig, other), changes) {
		t.Fatalf("restored %v, want %v", ChangedSquares(orig, other), changes)
	}
	before := ChangedSquares(orig, other)
	if err := RestoreSquares(orig, other, []SquareState{{Level: 99}}); err == nil {
		t.Fatal("a square outside the maze was accepted")
	}
	if !reflect.DeepEqual(ChangedSquares(orig, other), before) {
		t.Fatal("a refused restore changed squares")
	}
}
