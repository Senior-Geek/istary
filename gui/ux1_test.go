package main

// UX-1 through the Session as the front end drives it: the coordinate
// readout in the maze and at Camp, the stranded nudge on the party's
// square, and the equipment check at the Edge of Town.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// readout checks the view's position is the minimap's party square and
// facing, and returns it.
func readout(t *testing.T, v View) app.PositionView {
	t.Helper()
	if v.Position == nil || v.Minimap == nil {
		t.Fatalf("no position (%v) or minimap (%v)", v.Position, v.Minimap)
	}
	if v.Position.Place != v.Minimap.Party.Place || v.Position.Facing != v.Minimap.Party.Facing {
		t.Fatalf("readout %+v, minimap %+v", v.Position, v.Minimap.Party)
	}
	return *v.Position
}

// The readout follows the party's moves and turns, is the same at Camp as
// in the maze, and is gone back in town.
func TestCoordinatesOnTheScreen(t *testing.T) {
	s, _ := partySession(t, false)
	if v := s.State(); v.Position != nil {
		t.Fatalf("a position in town: %+v", v.Position)
	}
	v := s.RealExpedition()
	if v.Error != "" || v.Dungeon.Phase != "camp" {
		t.Fatalf("setting out: %q %+v", v.Error, v.Dungeon)
	}
	if p := readout(t, v); p.Place != "L1:R19:C0" || p.Facing != "N" {
		t.Fatalf("Camp at the entrance: %+v", p)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	v = s.DungeonAction(app.ActNo, 0)
	if p := readout(t, v); p.Place != "L1:R19:C0" || p.Facing != "N" {
		t.Fatalf("the maze at the entrance: %+v", p)
	}
	for _, step := range []struct{ act, facing string }{{app.ActRight, "E"}, {app.ActRight, "S"}, {app.ActLeft, "E"}, {app.ActLeft, "N"}} {
		v = s.DungeonAction(step.act, 0)
		if p := readout(t, v); p.Place != "L1:R19:C0" || p.Facing != step.facing {
			t.Fatalf("after %s: %+v", step.act, p)
		}
	}
	s.ctrl.Game.FightMap = [20][20]bool{}
	v = s.DungeonAction(app.ActForward, 0)
	if p := readout(t, v); p.Place != "L1:R18:C0" || p.Facing != "N" {
		t.Fatalf("a step north: %+v", p)
	}
	maze := readout(t, v)
	v = s.DungeonAction(app.ActCamp, 0)
	if v.Dungeon.Phase != "camp" {
		t.Fatalf("not in Camp: %+v", v.Dungeon)
	}
	if camp := readout(t, v); camp != maze {
		t.Fatalf("Camp %+v, the maze %+v", camp, maze)
	}
	v = s.DungeonAction(app.ActLeaveCamp, 0)
	if again := readout(t, v); again != maze {
		t.Fatalf("after Camp %+v, before %+v", again, maze)
	}
}

// The nudge counts those on the party's square, as the minimap's marker
// under the party does, and follows each pickup.
func TestStrandedHereFollowsPickups(t *testing.T) {
	s, _, _, pit := strandedAndRescuers(t, 3)
	g := s.ctrl.Game
	_, g.PlayerX, g.PlayerY = pit.Internal()
	v := s.State()
	under := func(v View) int {
		for _, m := range v.Minimap.Stranded {
			if m.Row == app.LocalRadius && m.Col == app.LocalRadius {
				return m.Count
			}
		}
		return 0
	}
	if p := readout(t, v); p.Place != pit.String() || p.Stranded != 6 || under(v) != 6 {
		t.Fatalf("on the square: %+v, marker %d", p, under(v))
	}
	v = s.DungeonAction(app.ActInspect, 0)
	for want := 5; want >= 3; want-- {
		if v = s.DungeonAction(app.ActPickUp, 1); v.Error != "" {
			t.Fatal(v.Error)
		}
		if p := readout(t, v); p.Stranded != want || under(v) != want {
			t.Fatalf("after a pickup: %d stranded here, marker %d, want %d", p.Stranded, under(v), want)
		}
	}
}

// equipNamed equips (or carries) the named scenario item for the roster
// character at index, as the character sheet does.
func equipNamed(t *testing.T, s *Session, index int, name string, equip bool) {
	t.Helper()
	ch := s.ctrl.Game.Town.Roster.Characters[index]
	for i, it := range s.scenario.Items {
		if it.Name == name {
			ch.AddItem(i, true)
			if equip {
				if v := s.EquipAction(EquipRequest{Index: index, N: ch.ItemCount, Equip: true}); v.Error != "" {
					t.Fatalf("equip %s: %s", name, v.Error)
				}
			}
			return
		}
	}
	t.Fatalf("no item %s", name)
}

// The equipment check at the Edge of Town: who has no usable weapon or no
// armor equipped, carried items not counting, a Mage never asked to wear
// armor. It is a caution: reading it saves nothing, Return to Town only
// goes to the castle, and Continue sets out as before.
func TestEquipmentCheckAtTheEdge(t *testing.T) {
	s, dir := partySession(t, false) // AGUI-FGUI: Fighter, Fighter, Thief, Priest, Mage, Fighter; nothing worn
	r := s.State().Readiness
	if r == nil || len(r.Notes) != 6 {
		t.Fatalf("readiness %+v", r)
	}
	for _, want := range []string{"Equipment check: AGUI has no weapon equipped and has no armor equipped;", "; EGUI has no weapon equipped;"} {
		if !strings.Contains(r.Line, want) {
			t.Fatalf("%q lacks %q", r.Line, want)
		}
	}

	// Carried is not equipped.
	equipNamed(t, s, 0, "LONG SWORD", false)
	equipNamed(t, s, 0, "CHAIN MAIL", false)
	if r := s.State().Readiness; r.Notes[0].Name != "AGUI" || len(r.Notes[0].Problems) != 2 {
		t.Fatalf("carried, not worn: %+v", r.Notes[0])
	}
	// Worn by the rules, the warning goes; any usable weapon will do.
	s.ctrl.Game.Town.Roster.Characters[0].ItemCount = 0
	equipNamed(t, s, 0, "LONG SWORD", true)
	equipNamed(t, s, 0, "CHAIN MAIL", true)
	equipNamed(t, s, 3, "STAFF", true)
	equipNamed(t, s, 3, "LEATHER ARMOR", true)
	equipNamed(t, s, 4, "DAGGER", true)
	r = s.State().Readiness
	var names []string
	for _, n := range r.Notes {
		names = append(names, n.Name)
	}
	if strings.Join(names, ",") != "BGUI,CGUI,FGUI" {
		t.Fatalf("still warned: %v (%s)", names, r.Line)
	}

	// Reading the check writes nothing.
	saved, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	for i := 0; i < 3; i++ {
		s.State()
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, saved) {
		t.Fatal("reading the check changed the save")
	}

	// Return to Town: the castle, the party and its kit as they were.
	party := strings.Join(savedParty(t, dir), ",")
	v := s.Navigate("castle")
	if v.Error != "" || v.Location != "castle" || v.Readiness != nil {
		t.Fatalf("return to town: %q %s %+v", v.Error, v.Location, v.Readiness)
	}
	if got := strings.Join(savedParty(t, dir), ","); got != party {
		t.Fatalf("party %s, was %s", got, party)
	}
	if ch := s.ctrl.Game.Town.Roster.Characters[1]; ch.ItemCount != 0 || s.ctrl.Game.Expedition != nil {
		t.Fatalf("something changed: %d items, expedition %v", ch.ItemCount, s.ctrl.Game.Expedition)
	}

	// Continue into Maze: setting out is not refused for it.
	walkTo(t, s, "edge")
	if v := s.State(); v.Readiness == nil || v.Departures.Real != "" {
		t.Fatalf("back at the edge: %+v %+v", v.Readiness, v.Departures)
	}
	v = s.RealExpedition()
	if v.Error != "" || !v.Real || v.Readiness != nil {
		t.Fatalf("continue into maze: %q real %v readiness %+v", v.Error, v.Real, v.Readiness)
	}
	if ch := s.ctrl.Game.Town.Roster.Characters[1]; ch.ItemCount != 0 {
		t.Fatal("setting out equipped someone")
	}
}

// The incapacitated are carried, not checked; a party whose able members
// are ready has nothing to say.
func TestEquipmentCheckSkipsTheIncapacitated(t *testing.T) {
	s, _ := partySession(t, false)
	for i := 0; i < 6; i++ {
		switch i {
		case 3: // the Priest, shielded but unarmed
			equipNamed(t, s, i, "SMALL SHIELD", true)
		case 4: // the Mage
			equipNamed(t, s, i, "STAFF", true)
		case 5:
			s.ctrl.Game.Town.Roster.Characters[i].Status = engine.Paralyzed
		default:
			equipNamed(t, s, i, "SHORT SWORD", true)
			equipNamed(t, s, i, "SMALL SHIELD", true)
		}
	}
	if s.ctrl.Game.Town.Roster.Characters[3].Class != engine.Priest {
		t.Fatal("DGUI is not the Priest")
	}
	if r := s.State().Readiness; r == nil || len(r.Notes) != 1 || r.Notes[0].Name != "DGUI" || r.Line != "Equipment check: DGUI has no weapon equipped." {
		t.Fatalf("readiness %+v", r)
	}
}
