package app

// R-3: where stranded characters lie, on the maps, and the rescue's
// progress on Inspect. The numbers are the R-3 brief's tests.

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// markersOn is level l's markers as "place xN: NAME CONDITION, ...".
func markersOn(t *testing.T, c *Controller, l int) []string {
	t.Helper()
	m, err := c.Map(l)
	if err != nil {
		t.Fatalf("map L%d: %v", l, err)
	}
	return markerLines(m.Stranded)
}

func markerLines(ms []MapStranded) []string {
	out := []string{}
	for _, m := range ms {
		s := m.Place + " x" + string(rune('0'+m.Count)) + ":"
		for _, ch := range m.Characters {
			s += " " + ch.Name + " " + ch.Condition
		}
		out = append(out, s)
	}
	return out
}

func sameLines(a, b []string) bool { return reflect.DeepEqual(a, b) }

// geometry is a map view without its markers.
func geometry(m MapView) MapView {
	m.Stranded = nil
	return m
}

// (1, 2, 3, 16) One marker a square: one for one character, one with the
// count for several, one each for different squares, in order of row and
// column; each character listed with their condition, in roster order.
func TestStrandedMarkers(t *testing.T) {
	c := rescueParty(t, 1)
	strandedAt(c, engine.Place{Level: 1, Row: 5, Col: 8}, "ONE", 1)
	if got, want := markersOn(t, c, 1), []string{"L1:R5:C8 x1: ONE1 DEAD"}; !sameLines(got, want) {
		t.Fatalf("one: %v", got)
	}
	strandedAt(c, engine.Place{Level: 1, Row: 12, Col: 3}, "MANY", 4)
	strandedAt(c, engine.Place{Level: 1, Row: 2, Col: 17}, "FAR", 1)
	want := []string{
		"L1:R2:C17 x1: FAR1 DEAD",
		"L1:R5:C8 x1: ONE1 DEAD",
		"L1:R12:C3 x4: MANY1 DEAD MANY2 PARALYZED MANY3 STONED MANY4 ASHES",
	}
	if got := markersOn(t, c, 1); !sameLines(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
	m, _ := c.Map(1)
	if s := m.Stranded[2]; s.Row != 12 || s.Col != 3 || s.Count != 4 || s.Characters[0].Class != "N-FIG" || s.Characters[0].Level != 2 {
		t.Fatalf("marker: %+v", s)
	}
	for st, want := range map[engine.Status]string{engine.OK: "OK", engine.Asleep: "ASLEEP", engine.Afraid: "AFRAID",
		engine.Paralyzed: "PARALYZED", engine.Stoned: "STONED", engine.Dead: "DEAD", engine.Ashed: "ASHES", engine.Lost: "LOST"} {
		if got := ConditionLabel(st); got != want {
			t.Errorf("%v: %q", st, got)
		}
	}
}

// (4) Each level's map has its own markers only; a level where characters
// lie can be chosen even if nothing of it was explored, and shows them
// alone; a level with neither is not a map.
func TestStrandedMarkersByLevel(t *testing.T) {
	c := rescueParty(t, 1)
	c.Explore() // L1's entrance only
	strandedAt(c, engine.Place{Level: 1, Row: 5, Col: 8}, "FIRST", 1)
	strandedAt(c, engine.Place{Level: 3, Row: 9, Col: 9}, "THIRD", 2)
	m, err := c.Map(0)
	must(t, err)
	if m.Level != 1 || !reflect.DeepEqual(m.Levels, []int{1, 3}) || len(m.Stranded) != 1 {
		t.Fatalf("party's level: L%d levels %v markers %v", m.Level, m.Levels, markerLines(m.Stranded))
	}
	m3, err := c.Map(3)
	must(t, err)
	if got := markerLines(m3.Stranded); !sameLines(got, []string{"L3:R9:C9 x2: THIRD1 DEAD THIRD2 PARALYZED"}) {
		t.Fatalf("L3: %v", got)
	}
	if len(m3.Visited) != 0 || len(m3.Walls) != 0 || len(m3.Wraps) != 0 || len(m3.Features) != 0 || m3.Party != nil {
		t.Fatalf("L3 shows more than its markers: %+v", m3)
	}
	if _, err := c.Map(2); !errors.Is(err, ErrNoMap) {
		t.Fatalf("L2: %v", err)
	}
	lv, _ := c.LocalMap()
	for _, s := range lv.Stranded {
		t.Fatalf("the minimap shows %s", s.Place)
	}
}

// (5) A marker reveals its square's place and who lies there, nothing
// else: the map's floors, walls, features and ways across the edge are
// what they are without it, the exploration is untouched, and no square
// is marked explored, by the maps or by Inspect's search.
func TestMarkersRevealNoGeometry(t *testing.T) {
	c := rescueParty(t, 1)
	g := c.Game
	c.Explore()
	without, err := c.Map(1)
	must(t, err)
	before := g.Exploration.Clone()
	far := engine.Place{Level: 1, Row: 5, Col: 8}
	strandedAt(c, far, "FAR", 2)
	with, err := c.Map(1)
	must(t, err)
	lv, _ := c.LocalMap()
	if len(with.Stranded) != 1 || !reflect.DeepEqual(geometry(with), geometry(without)) {
		t.Fatalf("the marker changed the map: %v", markerLines(with.Stranded))
	}
	for _, q := range with.Visited {
		if q.Row == far.Row && q.Col == far.Col {
			t.Fatal("the marked square is shown visited")
		}
	}
	for _, w := range with.Walls {
		if (w.Row-far.Row)*(w.Row-far.Row)+(w.Col-far.Col)*(w.Col-far.Col) <= 1 {
			t.Fatalf("a wall by the marker: %+v", w)
		}
	}
	// Inspect searches the party's area: the exploration is untouched by it
	// too, and the maps after it are the same.
	must(t, c.Inspect())
	must(t, c.LeaveInspect())
	if !reflect.DeepEqual(g.Exploration, before) {
		t.Fatal("the exploration changed")
	}
	again, _ := c.Map(1)
	lv2, _ := c.LocalMap()
	if !reflect.DeepEqual(again, with) || !reflect.DeepEqual(lv2, lv) {
		t.Fatal("the maps changed")
	}
}

// (6, 7) The LOST, and characters with no certain square, are never
// marked: a legacy record (no Stranded record) whether its fields name a
// square or not, and a Stranded record that names no square of the maze.
// Members of the party and of another version's saved expedition neither.
func TestNoMarkerWithoutAKnownSquare(t *testing.T) {
	c := rescueParty(t, 1)
	g := c.Game
	lost := strandedAt(c, engine.Place{Level: 1, Row: 5, Col: 8}, "LOST", 1)[0]
	lost.Status = engine.Lost
	old := member("OLDTIMER", engine.Neutral)
	old.InMaze, old.MazeLevel, old.MazeX, old.MazeY, old.Status = true, 0, 3, 4, engine.Dead
	bad := member("NOWHERE", engine.Neutral)
	bad.InMaze, bad.MazeLevel, bad.MazeX, bad.MazeY = true, 50, 3, 4
	off := member("OFFMAP", engine.Neutral)
	off.InMaze, off.Stranded = true, &engine.Place{Level: 99, Row: 1, Col: 1}
	castle := member("CASTLE", engine.Neutral)
	castle.InMaze, castle.Stranded = true, &engine.Place{}
	away := member("AWAY", engine.Neutral)
	away.InMaze, away.Stranded = true, &engine.Place{Level: 1, Row: 6, Col: 6}
	for _, ch := range []*engine.Character{old, bad, off, castle, away} {
		g.Town.Roster.Add(ch)
	}
	g.OtherExpedition = json.RawMessage(`{"version":99,"party":["AWAY"]}`)
	if s := g.StrandedSites(); len(s) != 0 {
		t.Fatalf("sites: %+v", s)
	}
	if _, err := c.Map(0); !errors.Is(err, ErrNoMap) {
		t.Fatalf("a map with nothing to show: %v", err)
	}
	// The legacy character is still found on its very square, as in R-2.
	onLevel(c, 0, 3, 4)
	must(t, c.Inspect())
	if !sameNames(g.MazeInspectFound, "OLDTIMER") {
		t.Fatalf("found %v", charNames(g.MazeInspectFound))
	}
}

// (8, 9, 13) Picking up lowers the marker's count, on both maps alike,
// and the last pickup removes it; markers elsewhere are kept.
func TestPickupUpdatesTheMarkers(t *testing.T) {
	c := rescueParty(t, 2)
	p := herePlace(t, c)
	strandedAt(c, p, "HERE", 3)
	strandedAt(c, engine.Place{Level: 1, Row: 5, Col: 8}, "FAR", 1)
	both := func() ([]string, []string) {
		lv, _ := c.LocalMap()
		return markersOn(t, c, 1), markerLines(lv.Stranded)
	}
	full, local := both()
	if !sameLines(full, []string{"L1:R5:C8 x1: FAR1 DEAD", p.String() + " x3: HERE1 DEAD HERE2 PARALYZED HERE3 STONED"}) ||
		!sameLines(local, full[1:]) {
		t.Fatalf("before: %v / %v", full, local)
	}
	must(t, c.Inspect())
	must(t, c.PickUp(2))
	full, local = both()
	if !sameLines(full, []string{"L1:R5:C8 x1: FAR1 DEAD", p.String() + " x2: HERE1 DEAD HERE3 STONED"}) || !sameLines(local, full[1:]) {
		t.Fatalf("after one: %v / %v", full, local)
	}
	must(t, c.PickUp(1))
	must(t, c.PickUp(1))
	full, local = both()
	if !sameLines(full, []string{"L1:R5:C8 x1: FAR1 DEAD"}) || len(local) != 0 {
		t.Fatalf("after all: %v / %v", full, local)
	}
}

// (10) When the rescuers are stranded in turn, everyone they carried is
// marked with them where it happened; those they left are still marked
// where they were.
func TestSecondDefeatMovesTheMarkers(t *testing.T) {
	c := rescueParty(t, 2)
	g := c.Game
	p := herePlace(t, c)
	strandedAt(c, p, "FALLEN", 3)
	must(t, c.Inspect())
	must(t, c.PickUp(1))
	must(t, c.LeaveInspect())
	onLevel(c, 0, 8, 8)
	for _, m := range g.Town.Party.Members {
		m.Status = engine.Dead
	}
	there, ok, err := g.StrandParty()
	if err != nil || !ok {
		t.Fatalf("strand: %v %v", ok, err)
	}
	want := []string{
		there.String() + " x3: RESCUER 1 DEAD RESCUER 2 DEAD FALLEN1 DEAD", // roster order
		p.String() + " x2: FALLEN2 PARALYZED FALLEN3 STONED",
	}
	if got := markersOn(t, c, 1); !sameLines(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

// (14) The party standing on a marked square: both are on the map, the
// party where it stands, the marker on the same square (the front end
// draws the marker as a badge beside the arrow).
func TestPartyOnAMarkedSquare(t *testing.T) {
	c := rescueParty(t, 1)
	p := herePlace(t, c)
	strandedAt(c, p, "HERE", 2)
	m, err := c.Map(0)
	must(t, err)
	if m.Party == nil || len(m.Stranded) != 1 || m.Stranded[0].Row != m.Party.Row || m.Stranded[0].Col != m.Party.Col {
		t.Fatalf("party %+v markers %+v", m.Party, m.Stranded)
	}
	lv, _ := c.LocalMap()
	if len(lv.Stranded) != 1 || lv.Stranded[0].Row != LocalRadius || lv.Stranded[0].Col != LocalRadius {
		t.Fatalf("minimap: %+v", lv.Stranded)
	}
}

// (Section 6) Inspect's progress: the party's size and room, who else is
// found here, whether the message is a pickup's, how many are known to lie
// elsewhere, and the members who cannot act, each with their condition;
// nobody is said to be healed. All read from the game as it stands.
func TestInspectProgress(t *testing.T) {
	c := rescueParty(t, 3)
	g := c.Game
	strandedAt(c, herePlace(t, c), "HERE", 2)
	strandedAt(c, engine.Place{Level: 1, Row: 5, Col: 8}, "FAR", 3)
	must(t, c.Inspect())
	v := c.DungeonView().Inspect
	if v.Size != 3 || v.Room != 3 || v.Picked || v.Elsewhere != 3 || len(v.Carried) != 0 || len(v.Found) != 2 {
		t.Fatalf("before: %+v", v)
	}
	if v.Found[0].Condition != "DEAD" || v.Found[1].Condition != "PARALYZED" || v.Found[1].Status != "PARALYZE" {
		t.Fatalf("conditions: %+v", v.Found)
	}
	must(t, c.PickUp(2))
	v = c.DungeonView().Inspect
	if v.Size != 4 || v.Room != 2 || !v.Picked || v.Elsewhere != 3 || len(v.Found) != 1 ||
		!reflect.DeepEqual(v.Carried, []CharacterNote{{Name: "HERE2", Class: "N-FIG", Level: 3, Condition: "PARALYZED"}}) {
		t.Fatalf("after: %+v", v)
	}
	if g.Town.Party.Members[3].Status != engine.Paralyzed {
		t.Fatal("the pickup changed the condition")
	}
	must(t, c.PickUp(1))
	v = c.DungeonView().Inspect
	if len(v.Found) != 0 || !v.Picked || len(v.Carried) != 2 {
		t.Fatalf("all: %+v", v)
	}
	must(t, c.LeaveInspect())
	must(t, c.Inspect())
	if v = c.DungeonView().Inspect; v.Picked || v.Elsewhere != 3 {
		t.Fatalf("again: %+v", v)
	}
}

// (18) The maps and Inspect's progress only read: looking changes nothing,
// and what Inspect finds is what it found before R-3.
func TestLookingChangesNothing(t *testing.T) {
	c := rescueParty(t, 2)
	strandedAt(c, herePlace(t, c), "HERE", 2)
	strandedAt(c, engine.Place{Level: 2, Row: 5, Col: 8}, "FAR", 1)
	c.Explore()
	must(t, c.Inspect())
	snap := rescueState(t, c)
	exp := c.Game.Exploration.Clone()
	found := charNames(c.Game.FindStranded())
	for l := 0; l <= 3; l++ {
		c.Map(l)
	}
	c.LocalMap()
	c.DungeonView()
	if rescueState(t, c) != snap || !reflect.DeepEqual(c.Game.Exploration, exp) || !reflect.DeepEqual(charNames(c.Game.FindStranded()), found) {
		t.Fatal("looking changed the game")
	}
	if !sameNames(c.Game.MazeInspectFound, "HERE1", "HERE2") {
		t.Fatalf("found %v", charNames(c.Game.MazeInspectFound))
	}
}

// (13, 18) The markers and Inspect never disagree: standing on a marked
// square, Inspect finds everyone the marker lists; and everyone Inspect
// finds with a Stranded record lies on a marked square in the area.
func TestMarkersAndInspectAgree(t *testing.T) {
	for round := int64(1); round <= 5; round++ {
		c := rescueParty(t, 1)
		g := c.Game
		rng := rand.New(rand.NewSource(round))
		n := 0
		for len(g.Town.Roster.Characters) < 20 { // the roster's limit
			strandedAt(c, engine.Place{Level: 1 + rng.Intn(2), Row: rng.Intn(20), Col: rng.Intn(20)}, fmt.Sprintf("S%02d-", n), 1)
			n++
		}
		sites := g.StrandedSites()
		marked := map[*engine.Character]engine.Place{}
		for _, s := range sites {
			for _, ch := range s.Characters {
				marked[ch] = s.Place
			}
		}
		if len(marked) != n {
			t.Fatalf("%d of %d marked", len(marked), n)
		}
		for _, s := range sites {
			level, x, y := s.Place.Internal()
			onLevel(c, level, x, y)
			found := map[*engine.Character]bool{}
			for _, ch := range g.FindStranded() {
				found[ch] = true
				p, ok := marked[ch]
				if !ok || !inArea(g, p) {
					t.Fatalf("at %s Inspect finds %s, marked %v at %s", s.Place, ch.Name, ok, p)
				}
			}
			for _, ch := range s.Characters {
				if !found[ch] {
					t.Fatalf("at %s Inspect misses %s", s.Place, ch.Name)
				}
			}
		}
	}
}

// inArea reports whether p is in the area of the party's square.
func inArea(g *engine.GameState, p engine.Place) bool {
	level, x, y := p.Internal()
	area, ok := engine.RescueArea(g.Scenario, g.MazeLevel, g.PlayerX, g.PlayerY)
	return ok && level == g.MazeLevel && area[x][y]
}
