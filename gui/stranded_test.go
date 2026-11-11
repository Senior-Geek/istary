package main

// Stranded parties through the bridge (R-1): a real expedition that ends
// with no one able to go on leaves its party in the maze, saved in the same
// action; a restart, a failed save and its retry, and an encounter saved
// after its defeat was decided all keep that one outcome.

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// pitAhead sets out on a real expedition, walks out of Camp and leaves the
// party facing a pit (a square of this game's copy of the scenario made
// one) that will finish it: everyone at 1 HP with no save against the
// fall. It returns the pit's square and the party.
func pitAhead(t *testing.T, s *Session) (engine.Place, []*engine.Character) {
	t.Helper()
	if v := s.RealExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	if v := s.DungeonAction(app.ActNo, 0); v.Dungeon == nil || v.Dungeon.Prompt != "" {
		t.Fatalf("not walking: %+v", v.Dungeon)
	}
	g := s.ctrl.Game
	party := append([]*engine.Character(nil), g.Town.Party.Members...)
	for _, m := range party {
		m.HP, m.Agility = 1, 0
	}
	// From the entrance (0,0) north to (0,1), if open; else east to (1,0).
	g.Facing = engine.North
	if g.WallAhead() != data.WallOpen {
		g.Facing = engine.East
	}
	x, y := (g.PlayerX+g.Facing.DX()+20)%20, (g.PlayerY+g.Facing.DY()+20)%20
	cell := &g.Scenario.Mazes.Levels[0].Cells[y][x]
	cell.Type, cell.Encounter = data.SqPit, false
	g.FightMap = [20][20]bool{}
	if err := errorsJoin(s.checkpoint(), s.save()); err != nil { // the walk so far, as after any action
		t.Fatal(err)
	}
	p, _ := engine.PlaceOf(len(g.Scenario.Mazes.Levels), 0, x, y)
	return p, party
}

// strandedOnDisk fails unless the save holds names stranded at want, with
// no castle party and no expedition.
func strandedOnDisk(t *testing.T, dir string, names []string, want engine.Place) {
	t.Helper()
	st := onDisk(t, dir)
	if len(st.PartyNames) != 0 || len(st.Expedition) != 0 {
		t.Fatalf("on disk: party %v, expedition %d bytes", st.PartyNames, len(st.Expedition))
	}
	seen := map[string]int{}
	for _, ch := range st.Roster {
		seen[ch.Name]++
		for _, n := range names {
			if ch.Name == n && (ch.Stranded == nil || *ch.Stranded != want || !ch.InMaze) {
				t.Fatalf("%s on disk: stranded %v in maze %v, want %v", n, ch.Stranded, ch.InMaze, want)
			}
		}
	}
	for n, k := range seen {
		if k != 1 {
			t.Fatalf("%s is in the roster %d times", n, k)
		}
	}
}

func namesOf(party []*engine.Character) []string {
	var out []string
	for _, m := range party {
		out = append(out, m.Name)
	}
	return out
}

// strandedView checks the roster as the GUI shows it: each of names
// stranded at want, unable to join (OUT, or a full party first).
func strandedView(t *testing.T, v View, names []string, want engine.Place) {
	t.Helper()
	for _, n := range names {
		found := false
		for _, cv := range v.Roster {
			if cv.Name == n {
				found = true
				refusal := "** OUT **"
				if len(v.Party) == app.MaxPartySize {
					refusal = "** PARTY IS FULL **" // the Tavern's first answer
				}
				if cv.Stranded != want.String() || cv.InParty || cv.JoinRefusal != refusal || cv.MazeLevel != want.Level {
					t.Fatalf("%s in the view: stranded %q in party %v refusal %q level %d", n, cv.Stranded, cv.InParty, cv.JoinRefusal, cv.MazeLevel)
				}
			}
		}
		if !found {
			t.Fatalf("%s is not in the roster view", n)
		}
	}
}

// A real expedition that falls into a pit and cannot go on ends at once:
// the notice says where the party remains, the save (in the same action)
// holds it stranded, and a restart finds it there, once. (28, 34)
func TestStrandedPartyThroughTheBridge(t *testing.T) {
	rand.Seed(31)
	s, dir := partySession(t, false)
	pit, party := pitAhead(t, s)
	names := namesOf(party)
	v := s.DungeonAction(app.ActForward, 0)
	if v.Error != "" || v.Real || v.Expedition || v.Screen != "roster" {
		t.Fatalf("after the fall: error %q real %v expedition %v screen %q", v.Error, v.Real, v.Expedition, v.Screen)
	}
	if !strings.Contains(v.Notice, "THEY REMAIN AT "+pit.String()) || !strings.Contains(v.Notice, "not in the castle") {
		t.Fatalf("notice %q", v.Notice)
	}
	if len(v.Party) != 0 {
		t.Fatalf("a castle party of %d", len(v.Party))
	}
	strandedView(t, v, names, pit)
	strandedOnDisk(t, dir, names, pit)
	mustMatchDisk(t, s, dir, "after the fall")

	for i := 0; i < 2; i++ {
		r := reopened(t, s, dir)
		rv := r.State()
		if rv.Fatal != "" || rv.Recovery != "" || rv.Real || r.ctrl.Game.Town.Party.Size() != 0 || len(r.ctrl.Game.Town.Roster.Characters) != 6 {
			t.Fatalf("after restart: fatal %q recovery %q real %v party %d", rv.Fatal, rv.Recovery, rv.Real, r.ctrl.Game.Town.Party.Size())
		}
		strandedView(t, rv, names, pit)
	}
	strandedOnDisk(t, dir, names, pit)
}

// A failed save at the fatal step is reported; the file keeps the last
// checkpoint, so a restart before any retry resumes the party alive one
// step back, as if the step never happened (the file and the window never
// disagree silently: the window says Unsaved). Retrying writes the stranded
// party once. (31)
func TestStrandingWithAFailedSave(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	rand.Seed(32)
	s, dir := partySession(t, false)
	pit, party := pitAhead(t, s)
	names := namesOf(party)
	before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))

	os.Chmod(dir, 0o555)
	v := s.DungeonAction(app.ActForward, 0)
	os.Chmod(dir, 0o755)
	if !strings.Contains(v.Error, "Could not save") || v.Unsaved == "" || v.Real {
		t.Fatalf("the failed save: error %q unsaved %q real %v", v.Error, v.Unsaved, v.Real)
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, before) {
		t.Fatal("the file changed although the save failed")
	}
	for _, m := range party {
		if m.Stranded == nil || *m.Stranded != pit {
			t.Fatalf("%s in the window: %v", m.Name, m.Stranded)
		}
	}

	// A restart now: the file's last checkpoint, the party alive.
	r := reopened(t, s, dir)
	if !r.real || r.ctrl.Game.Town.Party.Size() != 6 {
		t.Fatalf("restart after the failed save: real %v party %d", r.real, r.ctrl.Game.Town.Party.Size())
	}
	for _, m := range r.ctrl.Game.Town.Party.Members {
		if m.Stranded != nil || m.Status != engine.OK {
			t.Fatalf("%s after the restart: %v %v", m.Name, m.Stranded, m.Status)
		}
	}

	// Retrying in the window instead: written once, nothing done again.
	for i := 0; i < 2; i++ {
		if v := s.SaveNow(); v.Error != "" || v.Unsaved != "" {
			t.Fatalf("retry %d: error %q unsaved %q", i, v.Error, v.Unsaved)
		}
		strandedOnDisk(t, dir, names, pit)
		mustMatchDisk(t, s, dir, "after the retry")
	}
}

// The app closing on the defeat screen (the defeat decided, combat not yet
// left) or just before it (a party still able to fight) is picked up on
// restart: decided, the party is stranded where it fought before anything
// else, and that is saved; undecided, the encounter is over as before and
// the party walks on. (29, 30)
func TestRestartOnTheDefeatScreen(t *testing.T) {
	for _, decided := range []bool{false, true} {
		rand.Seed(33)
		s, dir := partySession(t, false)
		realEncounter(t, s)
		g := s.ctrl.Game
		party := append([]*engine.Character(nil), g.Town.Party.Members...)
		here, _ := g.Place()
		if decided {
			for i, m := range party {
				m.Status, m.HP = engine.Dead, 0
				if i == 2 {
					m.Status = engine.Stoned
				}
			}
			g.Combat.Phase = engine.CombatDefeat
		}
		if err := errorsJoin(s.checkpoint(), s.save()); err != nil {
			t.Fatal(err)
		}
		if g.Expedition == nil || g.Expedition.Interrupted != app.InterruptedCombat {
			t.Fatal("no interrupted encounter saved")
		}
		r := reopened(t, s, dir)
		rv := r.State()
		if !decided {
			if !r.real || r.ctrl.Game.Phase != engine.PhaseMaze || r.ctrl.Game.Town.Party.Size() != 6 {
				t.Fatalf("undecided: real %v phase %v", r.real, r.ctrl.Game.Phase)
			}
			continue
		}
		if r.real || rv.Real || r.ctrl.Game.Phase != engine.PhaseTown || !strings.Contains(rv.Notice, "closed after the party was defeated") {
			t.Fatalf("decided: real %v phase %v notice %q", r.real, r.ctrl.Game.Phase, rv.Notice)
		}
		strandedOnDisk(t, dir, namesOf(party), here) // the resume saved it
		strandedView(t, rv, namesOf(party), here)
		for _, ch := range r.ctrl.Game.Town.Roster.Characters {
			if ch.Name == party[2].Name && ch.Status != engine.Stoned {
				t.Fatalf("%s became %v", ch.Name, ch.Status)
			}
		}
		// Again: the same.
		r2 := reopened(t, s, dir)
		strandedView(t, r2.State(), namesOf(party), here)
		strandedOnDisk(t, dir, namesOf(party), here)
	}
}

func errorsJoin(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// A save holding an expedition from another version keeps it untouched,
// and stranded characters beside it keep their squares. (33)
func TestStrandedBesideAnotherVersionsExpedition(t *testing.T) {
	rand.Seed(34)
	s, dir := partySession(t, false)
	pit, party := pitAhead(t, s)
	s.DungeonAction(app.ActForward, 0)
	s.shutdown()
	editSave(t, dir, func(st map[string]any) {
		st["expedition"] = map[string]any{"version": 2.0, "party": []any{"SOMEONE"}, "future": "kept"}
	})
	r := NewSession(dir)
	defer r.shutdown()
	v := r.State()
	if !strings.Contains(v.Recovery, "another version") {
		t.Fatalf("recovery %q", v.Recovery)
	}
	strandedView(t, v, namesOf(party), pit)
	createMage(t, r, "radagast") // a castle action that saves
	raw, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if !bytes.Contains(raw, []byte(`"future": "kept"`)) || bytes.Count(raw, []byte(`"stranded"`)) != 6 {
		t.Fatal("the save lost the other expedition or a stranded record")
	}
}

// A practice expedition that cannot go on strands only its copies: the
// roster and the save are untouched.
func TestPracticeDefeatStrandsNoOne(t *testing.T) {
	rand.Seed(35)
	s, dir := partySession(t, false)
	before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	walkTo(t, s, "edge")
	if v := s.PracticeExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	pg := s.practice.Game
	for _, m := range pg.Town.Party.Members {
		m.HP, m.Agility = 1, 0
	}
	pg.Facing = engine.North
	if pg.WallAhead() != data.WallOpen {
		pg.Facing = engine.East
	}
	x, y := (pg.PlayerX+pg.Facing.DX()+20)%20, (pg.PlayerY+pg.Facing.DY()+20)%20
	cell := &pg.Scenario.Mazes.Levels[0].Cells[y][x]
	cell.Type, cell.Encounter = data.SqPit, false
	pg.FightMap = [20][20]bool{}
	v := s.DungeonAction(app.ActForward, 0)
	if s.practice != nil || !strings.Contains(v.Notice, "perished") || !strings.Contains(v.Notice, "roster is unchanged") {
		t.Fatalf("practice: still running %v, notice %q", s.practice != nil, v.Notice)
	}
	for _, ch := range s.ctrl.Game.Town.Roster.Characters {
		if ch.Stranded != nil {
			t.Fatalf("%s stranded by practice", ch.Name)
		}
	}
	if s.ctrl.Game.Town.Party.Size() != 6 {
		t.Fatalf("castle party %d", s.ctrl.Game.Town.Party.Size())
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, before) {
		t.Fatal("practice changed the save")
	}
}
