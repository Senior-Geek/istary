package main

import (
	"fmt"
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
	"trading":  {"castle"},
	"inn":      {"castle"},
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
		reached := []string{"castle", "edge", "settings", "tavern", "training"}
		if party {
			reached = []string{"castle", "edge", "inn", "settings", "tavern", "trading", "training"}
		}
		if got := slices.Sorted(maps.Keys(paths)); !slices.Equal(got, reached) {
			t.Fatalf("party %v: reached %v", party, got)
		}
		v := navigate(t, s, dir, "castle")
		for _, e := range v.Exits[1:4] { // the Inn, Boltac's, the Temple
			want := "not open in Istary's window"
			if e.Dest == "trading" || e.Dest == "inn" {
				want = ""
			}
			if !party {
				want = "YOU HAVE NO PARTY!"
			}
			if e.Open != (want == "") || !strings.Contains(e.Reason, want) || (want == "" && e.Reason != "") {
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

	walkTo(t, s, "training")
	mustOK(t, s.StartCreation("NEWCOMER"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	refused("creating", "training", errCreating.Error())
	mustOK(t, s.SetPassword("", ""), "creation")
	mustOK(t, s.Back(), "creation")
	mustOK(t, s.CancelCreation(), "roster")

	walkTo(t, s, "edge")
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
	walkTo(t, s, "tavern")
	if v := s.JoinParty(0); v.PendingJoin != "GUARDED" || v.Location != "tavern" {
		t.Fatalf("pending %q at %q", v.PendingJoin, v.Location)
	}
	if v := navigate(t, s, dir, "castle"); v.Error != "" || v.Location != "castle" || v.PendingJoin != "" || len(v.Party) != 0 {
		t.Fatalf("leaving the Tavern: %q at %q, pending %q", v.Error, v.Location, v.PendingJoin)
	}
	walkTo(t, s, "training")
	if v := mustOK(t, s.StartCreation("NEWCOMER"), "confirmCreate"); v.Location != "training" {
		t.Fatalf("creating at %q", v.Location)
	}
	if v := navigate(t, s, dir, "castle"); v.Error != "" || v.Location != "castle" || v.Screen != "roster" || len(v.Roster) != 1 {
		t.Fatalf("leaving the Training Grounds: %q at %q on %q", v.Error, v.Location, v.Screen)
	}
}

// Settings is left only by going back to the Castle; reading the roster
// again there (a reload) keeps the player in Settings. Coming back to the
// Edge of Town later is the Edge of Town, not Settings.
func TestSettingsIsLeftForTheCastle(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""), hero("SPARE", engine.Good, ""))
	join(t, s, 0, "")
	walkTo(t, s, "edge")
	if v := navigate(t, s, dir, "settings"); v.Location != "settings" || !slices.Equal(dests(v.Exits), []string{"castle"}) {
		t.Fatalf("settings: at %q, exits %v", v.Location, dests(v.Exits))
	}
	if v := s.JoinParty(1); v.Error != notAt(engine.Tavern).Error() || v.Location != "settings" {
		t.Fatalf("joining from Settings: %q at %q", v.Error, v.Location)
	}
	for _, act := range []func() View{s.RealExpedition, s.PracticeExpedition} {
		if v := act(); v.Error != notAt(engine.EdgeOfTown).Error() || v.Location != "settings" {
			t.Fatalf("setting out from Settings: %q at %q", v.Error, v.Location)
		}
	}
	if v := s.ReloadRoster(); v.Error != "" || v.Location != "settings" {
		t.Fatalf("after a reload: %q at %q", v.Error, v.Location)
	}
	navigate(t, s, dir, "castle")
	if v := navigate(t, s, dir, "edge"); v.Location != "edge" {
		t.Fatalf("back at the Edge of Town: at %q", v.Location)
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
	walkTo(t, s, "edge")

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
		if p.Name != q.Name || p.Class != q.Short || p.AC != q.AC || p.HP != q.HP || p.MaxHP != q.MaxHP || p.Status != q.Status || p.Index != q.Index {
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
			if p.Name != m.Name || p.Class != m.Class || p.AC != m.AC || p.HP != m.HP || p.MaxHP != m.MaxHP || p.Status != m.Status || p.Poisoned != m.Poisoned {
				t.Fatalf("after %s: bar %+v, dungeon %+v", act, p, m)
			}
		}
	}
}

// Every action that belongs to a location is accepted there and refused
// everywhere else, with the reason, without changing the view, the saved
// roster or the random numbers, and without moving the player.
func TestActionsOnlyAtTheirLocations(t *testing.T) {
	type action struct {
		name, at string
		do       func(s *Session) View
	}
	actions := []action{
		{"join", "tavern", func(s *Session) View { return s.JoinParty(1) }},
		{"leave party", "tavern", func(s *Session) View { return s.LeaveParty(0) }},
		{"create", "training", func(s *Session) View { return s.StartCreation("NEWBIE") }},
		{"rename", "training", func(s *Session) View { return s.RenameCharacter(1, "RENAMED") }},
		{"delete", "training", func(s *Session) View { return s.DeleteCharacter("SPARE", "") }},
		{"real expedition", "edge", func(s *Session) View { return s.RealExpedition() }},
		{"practice expedition", "edge", func(s *Session) View { return s.PracticeExpedition() }},
		{"practice encounter", "edge", func(s *Session) View { return s.PracticeEncounter(false) }},
		{"reload", "settings", func(s *Session) View { return s.ReloadRoster() }},
		{"reset roster", "settings", func(s *Session) View { return s.ResetRoster() }},
		{"restore backup", "settings", func(s *Session) View { return s.RestoreBackup("none.json") }},
	}
	why := map[string]string{
		"tavern":   notAt(engine.Tavern).Error(),
		"training": notAt(engine.Training).Error(),
		"edge":     notAt(engine.EdgeOfTown).Error(),
		"settings": errNotInSettings.Error(),
	}
	for _, a := range actions {
		for place := range routes {
			s, dir := newTestSession(t)
			rosterWith(t, s, hero("WALKER", engine.Good, ""), hero("SPARE", engine.Good, ""))
			join(t, s, 0, "")
			walkTo(t, s, place)
			before, saved := s.State(), onDisk(t, dir)
			rand.Seed(13)
			v := a.do(s)
			next := rand.Int63()
			if place == a.at {
				if v.Error == why[a.at] {
					t.Errorf("%s at %s: refused", a.name, place)
				}
				s.shutdown()
				continue
			}
			rand.Seed(13)
			if v.Error != why[a.at] || !sameView(v, before) || next != rand.Int63() ||
				!reflect.DeepEqual(onDisk(t, dir), saved) {
				t.Errorf("%s at %s: not refused cleanly: %q", a.name, place, v.Error)
			}
			s.shutdown()
		}
	}
}

// barAgrees fails unless the party bar shows the same members, in the same
// positions, with the same HP, AC and condition, as the screen's own party.
func barAgrees(t *testing.T, when string, bar []PartyBarSlot, slot int, name, class string, ac, hp, maxHP int, status string, poisoned bool) {
	t.Helper()
	p := bar[slot-1]
	if p.Name != name || p.Class != class || p.AC != ac || p.HP != hp || p.MaxHP != maxHP || p.Status != status || p.Poisoned != poisoned {
		t.Fatalf("%s: position %d: bar %+v, the screen %s %s AC %d %d/%d %s poisoned %v", when, slot, p, name, class, ac, hp, maxHP, status, poisoned)
	}
}

// Through random play the party bar stays exactly the party each screen
// shows: in Camp (whose member picks go by position, through reorders and
// trades), in the maze and back in the castle; in combat, the formation.
func TestPartyBarFollowsPlay(t *testing.T) {
	for seed := int64(1); seed <= 4; seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, _ := campSession(t, rng, seed%2 == 0)
		for step := 0; step < 120; step++ {
			v := s.State()
			if v.Camp == nil {
				break
			}
			named := 0
			for _, p := range v.PartyBar {
				if p.Name != "" {
					named++
				}
			}
			if named != len(v.Camp.Party) {
				t.Fatalf("seed %d step %d: %d on the bar, %d in Camp", seed, step, named, len(v.Camp.Party))
			}
			for i, m := range v.Camp.Party {
				barAgrees(t, fmt.Sprintf("seed %d camp step %d", seed, step), v.PartyBar, i+1, m.Name, m.Short, m.AC, m.HP, m.MaxHP, m.Status, m.Poison > 0)
			}
			s.CampAction(pickCamp(v.Camp, rng, step > 100))
		}
		s.shutdown()
	}

	check := func(when string, v View) {
		t.Helper()
		switch {
		case v.Combat != nil:
			// Combat shows the party as of the event being played back; the
			// bar has the outcome already, which is why the page does not
			// show it in combat. The formation is the same.
			for _, m := range v.Combat.Party {
				if p := v.PartyBar[m.Slot-1]; p.Name != m.Name || p.Class != m.Class || p.MaxHP != m.MaxHP {
					t.Fatalf("%s combat: position %d: bar %+v, combat %+v", when, m.Slot, p, m)
				}
			}
		case v.Dungeon != nil:
			for _, m := range v.Dungeon.Party {
				barAgrees(t, when+" maze", v.PartyBar, m.Slot, m.Name, m.Class, m.AC, m.HP, m.MaxHP, m.Status, m.Poisoned)
			}
		}
	}
	for seed := int64(1); seed <= 3; seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, _ := partySession(t, false)
		s.PracticeEncounter(false)
		_, views := drive(t, s, rng)
		for i, v := range views {
			check(fmt.Sprintf("seed %d encounter view %d", seed, i), v)
		}
		s.AbandonPractice()
		walkTo(t, s, "edge")
		s.RealExpedition()
		v := explore(t, s, rng, map[string]int{}, func(v View) { check(fmt.Sprintf("seed %d", seed), v) })
		if !v.Real {
			saved := s.ctrl.Game.Town.Party.Members
			for i, p := range v.PartyBar {
				if (i < len(saved) && saved[i] != nil && (p.Name != saved[i].Name || p.HP != saved[i].HP)) || (i >= len(saved) && p.Name != "") {
					t.Fatalf("seed %d: back in the castle, position %d: %+v", seed, i+1, p)
				}
			}
		}
		s.shutdown()
	}
}

// returnToCastle calls ReturnToCastle and fails if it wrote anything in the
// save folder or drew a random number.
func returnToCastle(t *testing.T, s *Session, dir string) View {
	t.Helper()
	before := folder(t, dir)
	rand.Seed(23)
	v := s.ReturnToCastle()
	next := rand.Int63()
	rand.Seed(23)
	if next != rand.Int63() {
		t.Fatal("returning to the Castle drew a random number")
	}
	if !maps.Equal(before, folder(t, dir)) {
		t.Fatal("returning to the Castle wrote to the save folder")
	}
	return v
}

// "← Castle" and Escape return from every town place, abandoning only
// what is unfinished there: a character being created, a join waiting for
// its password. What was done stays done; at the Castle nothing happens.
func TestReturnToCastle(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""), hero("GUARDED", engine.Good, "pw"))
	join(t, s, 0, "")
	for _, place := range []string{"tavern", "edge", "training", "settings"} {
		v := walkTo(t, s, place)
		if v.Return == nil || !v.Return.Open || v.Return.Note != "" {
			t.Fatalf("at %s: return %+v", place, v.Return)
		}
		if v = returnToCastle(t, s, dir); v.Error != "" || v.Location != "castle" || v.Return != nil || partyNames(v) != "WALKER" {
			t.Fatalf("from %s: %q at %q, party %q", place, v.Error, v.Location, partyNames(v))
		}
	}
	before := s.State()
	if v := returnToCastle(t, s, dir); v.Error != "" || !sameView(v, before) {
		t.Fatalf("at the Castle: %q", v.Error)
	}

	// A join waiting for its password is cancelled; the earlier join stays.
	walkTo(t, s, "tavern")
	if v := s.JoinParty(1); v.PendingJoin != "GUARDED" || v.Return == nil || v.Return.Note == "" {
		t.Fatalf("pending join: %q, return %+v", v.PendingJoin, v.Return)
	}
	if v := returnToCastle(t, s, dir); v.Error != "" || v.Location != "castle" || v.PendingJoin != "" || partyNames(v) != "WALKER" {
		t.Fatalf("with a join pending: %q at %q, pending %q, party %q", v.Error, v.Location, v.PendingJoin, partyNames(v))
	}

	// A character being created is discarded, at any step; the roster stays.
	saved := onDisk(t, dir)
	for _, steps := range []int{0, 1, 2, 4} {
		walkTo(t, s, "training")
		mustOK(t, s.StartCreation("NEWCOMER"), "confirmCreate")
		if steps > 0 {
			mustOK(t, s.AnswerCreate(true), "creation")
		}
		if steps > 1 {
			mustOK(t, s.SetPassword("", ""), "creation")
		}
		if steps > 2 {
			mustOK(t, s.ChooseRace(0), "creation")
			mustOK(t, s.ChooseAlignment(0), "creation")
		}
		v := s.State()
		if v.Return == nil || !v.Return.Open || (steps > 0 && v.Return.Note == "") {
			t.Fatalf("creating (%d steps): return %+v", steps, v.Return)
		}
		v = returnToCastle(t, s, dir)
		if v.Error != "" || v.Location != "castle" || v.Screen != "roster" || v.Creation != nil || len(v.Roster) != 2 {
			t.Fatalf("creating (%d steps): %q at %q on %q, roster %d", steps, v.Error, v.Location, v.Screen, len(v.Roster))
		}
		if steps > 0 && v.Notice != "The character being created was discarded." {
			t.Fatalf("creating (%d steps): notice %q", steps, v.Notice)
		}
	}
	if !reflect.DeepEqual(saved, onDisk(t, dir)) {
		t.Fatal("abandoning creation changed the save")
	}
}

// Out of town, returning to the Castle is refused and changes nothing:
// practice, a real expedition, and one resumed after a restart.
func TestReturnToCastleRefusedOutOfTown(t *testing.T) {
	rand.Seed(8)
	s, dir := partySession(t, false)
	refused := func(when, why string) {
		t.Helper()
		before := s.State()
		if before.Return != nil {
			t.Fatalf("%s: return offered %+v", when, before.Return)
		}
		saved := onDisk(t, dir)
		if v := s.ReturnToCastle(); v.Error != why || !sameView(v, before) {
			t.Fatalf("%s: %q", when, v.Error)
		}
		if !reflect.DeepEqual(saved, onDisk(t, dir)) {
			t.Fatalf("%s: the save changed", when)
		}
	}
	s.PracticeExpedition()
	refused("practice expedition", errPracticeOut.Error())
	s.AbandonPractice()
	s.PracticeEncounter(false)
	refused("practice encounter", errPracticeOut.Error())
	s.AbandonPractice()
	s.RealExpedition()
	refused("in Camp", errExpeditionOut.Error())
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	refused("in the maze", errExpeditionOut.Error())
	s.shutdown()
	s = NewSession(dir)
	defer s.shutdown()
	refused("resumed", errExpeditionOut.Error())
}

// While a saved expedition cannot be resumed, returning to the Castle works
// as anywhere in town and leaves the expedition as it was saved.
func TestReturnToCastleLeavesRecoveryAlone(t *testing.T) {
	dir := outAndRestarted(t)
	editSave(t, dir, func(st map[string]any) { st["expedition"].(map[string]any)["version"] = 2.0 })
	s := NewSession(dir)
	defer s.shutdown()
	for _, place := range []string{"tavern", "settings", "training"} {
		walkTo(t, s, place)
		if v := returnToCastle(t, s, dir); v.Error != "" || v.Location != "castle" || v.Recovery == "" {
			t.Fatalf("from %s: %q at %q, recovery %q", place, v.Error, v.Location, v.Recovery)
		}
	}
}

// Boltac's opens from the Castle for a party (without one the castle's
// answer stays), and "← Castle", Escape and its exit leave it from every
// step, as RETURN does: a question waiting is answered no, the customer
// leaves. Nothing is bought, undone, saved or drawn on the way out.
func TestReturnFromEveryStepOfBoltacs(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("WALKER", engine.Good, ""))
	walkTo(t, s, "castle")
	if v := navigate(t, s, dir, "trading"); v.Error != "YOU HAVE NO PARTY!" || v.Location != "castle" {
		t.Fatalf("no party: %q at %q", v.Error, v.Location)
	}
	s.shutdown()

	rand.Seed(4)
	s, dir = partySession(t, false)
	mage := s.ctrl.Game.Town.Party.Members[4]
	mage.Gold = 500
	steps := []struct {
		step string
		to   func(c *app.Controller) error
	}{
		{"who", func(c *app.Controller) error { return nil }},
		{"main", func(c *app.Controller) error { return c.ShopEnter(4) }},
		{"buy", func(c *app.Controller) error {
			c.ShopEnter(4)
			c.HandleKey(app.Rune('B'))
			return nil
		}},
		{"buy", func(c *app.Controller) error { // the purchase prompt open
			c.ShopEnter(4)
			c.HandleKey(app.Rune('B'))
			c.HandleKey(app.Rune('P'))
			return nil
		}},
		{"confirm", func(c *app.Controller) error {
			c.ShopEnter(4)
			c.Buy(1) // a long sword: the mage cannot use it
			return nil
		}},
		{"sell", func(c *app.Controller) error {
			c.ShopEnter(4)
			c.Buy(5) // a staff, to have something
			c.HandleKey(app.Key{Kind: app.KeyEnter})
			c.HandleKey(app.Rune('S'))
			return nil
		}},
		{"uncurse", func(c *app.Controller) error {
			c.ShopEnter(4)
			c.HandleKey(app.Rune('U'))
			return nil
		}},
		{"identify", func(c *app.Controller) error {
			c.ShopEnter(4)
			c.HandleKey(app.Rune('I'))
			return nil
		}},
	}
	for _, way := range []string{"return", "exit"} {
		for _, st := range steps {
			walkTo(t, s, "castle")
			if v := navigate(t, s, dir, "trading"); v.Error != "" || v.Location != "trading" || v.Return == nil || !v.Return.Open {
				t.Fatalf("into Boltac's: %q at %q, return %+v", v.Error, v.Location, v.Return)
			}
			if err := st.to(s.ctrl); err != nil {
				t.Fatal(err)
			}
			c, err := s.ctrl.ShopView()
			if err != nil || c.Step != st.step {
				t.Fatalf("at %s: step %q (%v)", st.step, c.Step, err)
			}
			if v := s.State(); (v.Return.Note != "") != (st.step == "confirm") {
				t.Fatalf("at %s: note %q", st.step, v.Return.Note)
			}
			gold, items := mage.Gold, mage.ItemCount
			var v View
			if way == "return" {
				v = returnToCastle(t, s, dir)
			} else {
				v = navigate(t, s, dir, "castle")
			}
			g := s.ctrl.Game
			if v.Error != "" || v.Location != "castle" || g.Town.InputMode != engine.InputNone || g.Town.ShopChar != nil {
				t.Fatalf("%s from %s: %q at %q, mode %v", way, st.step, v.Error, v.Location, g.Town.InputMode)
			}
			if mage.Gold != gold || mage.ItemCount != items {
				t.Fatalf("%s from %s: gold %d items %d, were %d %d", way, st.step, mage.Gold, mage.ItemCount, gold, items)
			}
		}
	}
}
