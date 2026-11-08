package main

import (
	"io/fs"
	"maps"
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

// folder is every file under the save folder: its contents and when it was
// last written.
func folder(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[path] = info.ModTime().String() + "\n" + string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// navigate calls Navigate and fails if it wrote anything in the save folder
// or drew a random number.
func navigate(t *testing.T, s *Session, dir, dest string) View {
	t.Helper()
	before := folder(t, dir)
	rand.Seed(11)
	v := s.Navigate(dest)
	next := rand.Int63()
	rand.Seed(11)
	if next != rand.Int63() {
		t.Fatalf("going to %q drew a random number", dest)
	}
	if !maps.Equal(before, folder(t, dir)) {
		t.Fatalf("going to %q wrote to the save folder", dest)
	}
	return v
}

// sameView reports whether two views are the same apart from the last
// action's error and notice.
func sameView(a, b View) bool {
	a.Error, a.Notice, b.Error, b.Notice = "", "", "", ""
	return reflect.DeepEqual(a, b)
}

func dests(exits []ExitView) []string {
	var out []string
	for _, e := range exits {
		out = append(out, e.Dest)
	}
	return out
}

// The town as the original lays it out: the Castle is the hub; the
// Training Grounds and Settings are at the Edge of Town; every location
// but the Castle leads back to the Castle.
var townMap = map[string][]string{
	"castle":   {"tavern", "inn", "trading", "temple", "edge"},
	"edge":     {"training", "castle", "settings"},
	"training": {"castle"},
	"tavern":   {"castle"},
	"settings": {"castle"},
}

// Walking the town, from every location to every destination, goes exactly
// where the exits say, refuses everything else without changing the view,
// and never saves or draws a random number.
func TestNavigationWalksTheTown(t *testing.T) {
	everywhere := []string{"castle", "tavern", "inn", "trading", "temple", "edge", "training", "settings", "maze", "", "CASTLE"}
	for _, party := range []bool{false, true} {
		s, dir := newTestSession(t)
		rosterWith(t, s, hero("WALKER", engine.Good, ""))
		if party {
			join(t, s, 0, "")
		}
		paths := map[string][]string{"castle": nil}
		queue := []string{"castle"}
		for len(queue) > 0 {
			at := queue[0]
			queue = queue[1:]
			for _, dest := range everywhere {
				if v := s.State(); v.Location != "castle" {
					if v = navigate(t, s, dir, "castle"); v.Location != "castle" {
						t.Fatalf("back to the castle from %s: %q", v.Location, v.Error)
					}
				}
				for _, step := range paths[at] {
					navigate(t, s, dir, step)
				}
				before := s.State()
				if before.Location != at || !slices.Equal(dests(before.Exits), townMap[at]) {
					t.Fatalf("at %s (want %s): exits %v", before.Location, at, dests(before.Exits))
				}
				i := slices.IndexFunc(before.Exits, func(e ExitView) bool { return e.Dest == dest })
				v := navigate(t, s, dir, dest)
				if i >= 0 && before.Exits[i].Open {
					if v.Error != "" || v.Location != dest {
						t.Fatalf("party %v: %s to %s: at %s, %q", party, at, dest, v.Location, v.Error)
					}
					if _, seen := paths[dest]; !seen {
						paths[dest] = append(slices.Clone(paths[at]), dest)
						queue = append(queue, dest)
					}
					continue
				}
				if v.Error == "" || !sameView(v, before) {
					t.Fatalf("party %v: %s to %s was not refused cleanly: %q", party, at, dest, v.Error)
				}
				if i >= 0 && v.Error != before.Exits[i].Reason {
					t.Fatalf("party %v: %s to %s refused with %q, the exit says %q", party, at, dest, v.Error, before.Exits[i].Reason)
				}
			}
		}
		if got := slices.Sorted(maps.Keys(paths)); !slices.Equal(got, []string{"castle", "edge", "settings", "tavern", "training"}) {
			t.Fatalf("party %v: reached %v", party, got)
		}
		v := navigate(t, s, dir, "castle")
		for _, e := range v.Exits[1:4] { // the Inn, Boltac's, the Temple
			want := "not open in Istary's window"
			if !party {
				want = "YOU HAVE NO PARTY!"
			}
			if e.Open || !strings.Contains(e.Reason, want) {
				t.Errorf("party %v: %+v", party, e)
			}
		}
	}
}

// While a character is being created, or an expedition or practice is
// under way, nobody walks around town, and asking changes nothing: not the
// view, not the save or the expedition's checkpoint, not the random
// numbers. After a restart the resumed expedition is refused the same way.
func TestNavigationRefusedOutOfTown(t *testing.T) {
	rand.Seed(5)
	s, dir := partySession(t, false)
	refused := func(when, location, why string) {
		t.Helper()
		before := s.State()
		if before.Location != location {
			t.Fatalf("%s: location %q, want %q", when, before.Location, location)
		}
		for _, e := range before.Exits {
			if e.Open || e.Reason != why {
				t.Fatalf("%s: exit %+v", when, e)
			}
		}
		for _, dest := range []string{"castle", "tavern", "edge", "training", "settings"} {
			if v := navigate(t, s, dir, dest); v.Error != why || !sameView(v, before) {
				t.Fatalf("%s: going to %s: %q", when, dest, v.Error)
			}
		}
	}

	mustOK(t, s.StartCreation("NEWCOMER"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	refused("creating", "training", errCreating.Error())
	mustOK(t, s.SetPassword("", ""), "creation")
	mustOK(t, s.Back(), "creation")
	mustOK(t, s.CancelCreation(), "roster")

	s.PracticeExpedition()
	refused("practice expedition", "maze", errPracticeOut.Error())
	if v := s.State(); v.Exits != nil {
		t.Fatalf("exits in the maze: %v", v.Exits)
	}
	s.AbandonPractice()
	s.PracticeEncounter(false)
	refused("practice encounter", "maze", errPracticeOut.Error())
	s.AbandonPractice()

	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	refused("real expedition", "maze", errExpeditionOut.Error())
	if v := s.State(); !v.Real || v.Screen != "dungeon" {
		t.Fatalf("not out: %q", v.Screen)
	}

	s.shutdown()
	s = NewSession(dir)
	defer s.shutdown()
	if v := s.State(); !v.Real || v.Recovery != "" {
		t.Fatalf("not resumed: recovery %q", v.Recovery)
	}
	refused("resumed expedition", "maze", errExpeditionOut.Error())
}

// A saved expedition that cannot be resumed stays saved, untouched, while
// the player walks around town.
func TestNavigationLeavesAnUnresumableExpeditionAlone(t *testing.T) {
	dir := outAndRestarted(t)
	editSave(t, dir, func(st map[string]any) {
		st["expedition"].(map[string]any)["version"] = 2.0
	})
	s := NewSession(dir)
	defer s.shutdown()
	if v := s.State(); v.Recovery == "" || v.Location != "castle" {
		t.Fatalf("recovery %q at %q", v.Recovery, v.Location)
	}
	for _, dest := range []string{"tavern", "castle", "edge", "settings", "castle", "edge", "training", "castle"} {
		if v := navigate(t, s, dir, dest); v.Error != "" || v.Location != dest || v.Recovery == "" {
			t.Fatalf("going to %s: %q, recovery %q", dest, v.Error, v.Recovery)
		}
	}
}

// An open town prompt is closed by the game's own cancel steps before
// moving: a join waiting for its password, a creation not yet confirmed.
func TestNavigationClosesAnOpenPrompt(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("GUARDED", engine.Good, "pw"))
	if v := s.JoinParty(0); v.PendingJoin != "GUARDED" || v.Location != "tavern" {
		t.Fatalf("pending %q at %q", v.PendingJoin, v.Location)
	}
	if v := navigate(t, s, dir, "castle"); v.Error != "" || v.Location != "castle" || v.PendingJoin != "" || len(v.Party) != 0 {
		t.Fatalf("leaving the Tavern: %q at %q, pending %q", v.Error, v.Location, v.PendingJoin)
	}
	if v := mustOK(t, s.StartCreation("NEWCOMER"), "confirmCreate"); v.Location != "training" {
		t.Fatalf("creating at %q", v.Location)
	}
	if v := navigate(t, s, dir, "castle"); v.Error != "" || v.Location != "castle" || v.Screen != "roster" || len(v.Roster) != 1 {
		t.Fatalf("leaving the Training Grounds: %q at %q on %q", v.Error, v.Location, v.Screen)
	}
}

// Settings is left by going back to the Castle; anything else that moves
// the game (an action that goes to another location, a reload) leaves it
// too.
func TestSettingsIsLeftWithTheEdgeOfTown(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	navigate(t, s, dir, "edge")
	if v := navigate(t, s, dir, "settings"); v.Location != "settings" || !slices.Equal(dests(v.Exits), []string{"castle"}) {
		t.Fatalf("settings: at %q, exits %v", v.Location, dests(v.Exits))
	}
	if v := s.JoinParty(0); v.Location != "tavern" {
		t.Fatalf("joining from Settings: at %q", v.Location)
	}
	s.SubmitJoinPassword("")
	navigate(t, s, dir, "castle")
	if v := navigate(t, s, dir, "edge"); v.Location != "edge" {
		t.Fatalf("back at the Edge of Town: at %q", v.Location)
	}
	navigate(t, s, dir, "settings")
	if v := s.ReloadRoster(); v.Location != "castle" {
		t.Fatalf("after a reload: at %q", v.Location)
	}
	if v := navigate(t, s, dir, "edge"); v.Location != "edge" {
		t.Fatalf("after a reload, the Edge of Town: at %q", v.Location)
	}
}

// The party bar shows the six positions of the party being played: the
// saved party in town and out on a real expedition (where it agrees with
// the dungeon view), the practice copies during practice.
func TestPartyBarIsThePartyBeingPlayed(t *testing.T) {
	s, _ := newTestSession(t)
	lead := hero("LEAD", engine.Good, "")
	lead.Level, lead.HP, lead.MaxHP = 3, 11, 12
	rosterWith(t, s, hero("OTHER", engine.Good, ""), lead)
	join(t, s, 1, "")
	join(t, s, 0, "")
	lead.PoisonAmt = 2

	v := s.State()
	if len(v.PartyBar) != app.MaxPartySize {
		t.Fatalf("%d positions", len(v.PartyBar))
	}
	for i, p := range v.PartyBar {
		if p.Slot != i+1 || p.Front != (i < app.FrontRowSize) {
			t.Fatalf("position %d: %+v", i, p)
		}
		if i >= len(v.Party) {
			if p != (PartyBarSlot{Slot: i + 1, Index: -1, Front: p.Front}) {
				t.Fatalf("empty position %d: %+v", i, p)
			}
			continue
		}
		q := v.Party[i]
		if p.Name != q.Name || p.Class != q.Short || p.HP != q.HP || p.MaxHP != q.MaxHP || p.Status != q.Status || p.Index != q.Index {
			t.Fatalf("position %d: %+v, the Tavern shows %+v", i, p, q)
		}
	}
	if p := v.PartyBar[0]; p.Name != "LEAD" || p.Level != 3 || !p.Poisoned || v.PartyBar[1].Poisoned {
		t.Fatalf("lead: %+v", p)
	}

	s.PracticeExpedition()
	s.practice.Game.Town.Party.Members[0].HP = 1
	if v := s.State(); v.PartyBar[0].HP != 1 || v.PartyBar[0].Index != -1 || v.Party[0].HP != 11 {
		t.Fatalf("practice: bar %+v, saved party %+v", v.PartyBar[0], v.Party[0])
	}
	s.AbandonPractice()
	if v := s.State(); v.PartyBar[0].HP != 11 || v.PartyBar[0].Index != 1 {
		t.Fatalf("after practice: %+v", v.PartyBar[0])
	}

	rand.Seed(3)
	s.RealExpedition()
	for _, act := range []string{app.ActLeaveCamp, app.ActNo, app.ActForward, app.ActRight, app.ActForward} {
		v := s.DungeonAction(act, 0)
		for _, m := range v.Dungeon.Party {
			p := v.PartyBar[m.Slot-1]
			if p.Name != m.Name || p.Class != m.Class || p.HP != m.HP || p.MaxHP != m.MaxHP || p.Status != m.Status || p.Poisoned != m.Poisoned {
				t.Fatalf("after %s: bar %+v, dungeon %+v", act, p, m)
			}
		}
	}
}
