package main

// Rescue through the bridge (R-2): Inspect and P)ICK UP on a real
// expedition, each pickup one action saved with the expedition; a failed
// save and its retry; a restart; practice, which keeps nothing. Numbers in
// the comments are the R-2 brief's test scenarios (section 12).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// strandedAndRescuers strands the session's party of six in the pit beside
// the entrance (pitAhead), then forms a party of n sturdy rescuers and
// takes it out of Camp to the entrance, which lies in the same area as the
// pit. Encounters are switched off for the walk.
func strandedAndRescuers(t *testing.T, n int) (s *Session, dir string, fallen []*engine.Character, pit engine.Place) {
	t.Helper()
	s, dir = partySession(t, false)
	pit, fallen = pitAhead(t, s)
	if v := s.DungeonAction(app.ActForward, 0); v.Real || !strings.Contains(v.Notice, "THEY REMAIN AT "+pit.String()) {
		t.Fatalf("the first party did not fall: %q", v.Notice)
	}
	var rescuers []*engine.Character
	for i := 0; i < n; i++ {
		ch := engine.NewCharacter(fmt.Sprintf("RESCUER%d", i+1), engine.Human, engine.Fighter, engine.Neutral)
		ch.HP, ch.MaxHP, ch.Level, ch.Gold = 300, 300, 8, 50000
		rescuers = append(rescuers, ch)
	}
	rosterWith(t, s, rescuers...)
	walkTo(t, s, "tavern")
	for i := range rescuers {
		if v := s.JoinParty(len(fallen) + i); v.PendingJoin != "" {
			s.SubmitJoinPassword("")
		}
	}
	walkTo(t, s, "edge")
	if v := s.RealExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	if v := s.DungeonAction(app.ActNo, 0); v.Dungeon == nil || v.Dungeon.Prompt != "" || len(v.Party) != n {
		t.Fatalf("rescuers not walking: %+v", v.Dungeon)
	}
	s.ctrl.Game.FightMap = [20][20]bool{}
	return s, dir, fallen, pit
}

// foundNamed is the number Inspect gives the character named name, or 0.
func foundNamed(v View, name string) int {
	if v.Dungeon == nil || v.Dungeon.Inspect == nil {
		return 0
	}
	for _, f := range v.Dungeon.Inspect.Found {
		if f.Name == name {
			return f.N
		}
	}
	return 0
}

// diskParty is the expedition's party in the save, or the castle party
// when no expedition is saved.
func diskParty(t *testing.T, dir string) (names []string, expedition bool) {
	t.Helper()
	st := onDisk(t, dir)
	if len(st.Expedition) == 0 {
		return st.PartyNames, false
	}
	var e engine.Expedition
	if err := json.Unmarshal(st.Expedition, &e); err != nil {
		t.Fatal(err)
	}
	return e.Party, true
}

// strandedOnDiskAt is who the save has stranded at p.
func strandedOnDiskAt(t *testing.T, dir string, p engine.Place) []string {
	t.Helper()
	var out []string
	for _, ch := range onDisk(t, dir).Roster {
		if ch.Stranded != nil && *ch.Stranded == p {
			out = append(out, ch.Name)
		}
	}
	return out
}

// (35 and the atomic pickup) Each pickup through the bridge is one action
// saved with the expedition: the save then holds the character in the
// expedition's party and no longer stranded, the rest stranded. A full
// party is refused with the reason, and the save does not change. Home by
// the stairs, the rescued are in the castle party, stranded nowhere.
func TestRescueThroughTheBridge(t *testing.T) {
	rand.Seed(51)
	s, dir, fallen, pit := strandedAndRescuers(t, 3)
	v := s.DungeonAction(app.ActInspect, 0)
	if v.Error != "" || v.Dungeon.Inspect == nil || len(v.Dungeon.Inspect.Found) != 6 || v.Dungeon.Inspect.Room != 3 {
		t.Fatalf("inspect: %q %+v", v.Error, v.Dungeon.Inspect)
	}
	for i, name := range []string{fallen[1].Name, fallen[3].Name, fallen[5].Name} {
		v = s.DungeonAction(app.ActPickUp, foundNamed(v, name))
		if v.Error != "" || !strings.Contains(strings.Join(v.Dungeon.Lines, " "), name+" JOINS THE PARTY.") {
			t.Fatalf("pick %s: %q %v", name, v.Error, v.Dungeon.Lines)
		}
		party, out := diskParty(t, dir)
		if !out || len(party) != 4+i || party[len(party)-1] != name {
			t.Fatalf("on disk after %s: expedition %v party %v", name, out, party)
		}
		if got := strandedOnDiskAt(t, dir, pit); len(got) != 5-i {
			t.Fatalf("stranded on disk after %s: %v", name, got)
		}
		mustMatchDisk(t, s, dir, "after a pickup")
	}
	before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	v = s.DungeonAction(app.ActPickUp, 1)
	if !strings.Contains(v.Error, "YOUR PARTY HAS 6; "+fallen[0].Name+" STAYS HERE.") || len(v.Party) != 6 {
		t.Fatalf("a seventh: %q", v.Error)
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, before) {
		t.Fatal("the refused pickup changed the save")
	}
	for _, f := range v.Dungeon.Inspect.Found {
		if f.CanPick || f.Refusal == "" {
			t.Fatalf("full party, %+v", f)
		}
	}
	v = s.DungeonAction(app.ActLeave, 0)
	if v.Dungeon.Prompt != "" {
		t.Fatalf("after leaving: prompt %q", v.Dungeon.Prompt)
	}
	v = homeByTheStairs(t, s)
	if v.Real || len(v.Party) != 6 || v.Screen == "dungeon" {
		t.Fatalf("not home: real %v party %d screen %q", v.Real, len(v.Party), v.Screen)
	}
	party, out := diskParty(t, dir)
	if out || len(party) != 6 {
		t.Fatalf("home on disk: expedition %v party %v", out, party)
	}
	for _, ch := range onDisk(t, dir).Roster {
		rescued := ch.Name == fallen[1].Name || ch.Name == fallen[3].Name || ch.Name == fallen[5].Name
		if rescued && (ch.Stranded != nil || ch.InMaze) {
			t.Fatalf("%s home: stranded %v in maze %v", ch.Name, ch.Stranded, ch.InMaze)
		}
	}
	if got := strandedOnDiskAt(t, dir, pit); len(got) != 3 {
		t.Fatalf("left behind on disk: %v", got)
	}
	mustMatchDisk(t, s, dir, "home")
}

// homeByTheStairs walks the party from the entrance's area back onto the
// entrance (by way of a neighbour, turning round) and takes the stairs up.
func homeByTheStairs(t *testing.T, s *Session) View {
	t.Helper()
	g := s.ctrl.Game
	if g.MazeLevel != 0 || g.PlayerX != 0 || g.PlayerY != 0 {
		t.Fatalf("not at the entrance: L%d (%d,%d)", g.MazeLevel+1, g.PlayerX, g.PlayerY)
	}
	for i := 0; i < 4 && g.WallAhead() != data.WallOpen; i++ {
		s.DungeonAction(app.ActRight, 0)
	}
	steps := []string{app.ActForward, app.ActRight, app.ActRight, app.ActForward}
	var v View
	for _, a := range steps {
		v = s.DungeonAction(a, 0)
		for v.Dungeon != nil && v.Dungeon.Prompt == "messages" {
			v = s.DungeonAction(app.ActContinue, 0)
		}
		if v.Dungeon != nil && v.Dungeon.Prompt == "stairs" && a != steps[len(steps)-1] {
			v = s.DungeonAction(app.ActNo, 0)
		}
	}
	if v.Dungeon == nil || v.Dungeon.Prompt != "stairs" {
		t.Fatalf("no stairs: %+v (error %q)", v.Dungeon, v.Error)
	}
	return s.DungeonAction(app.ActYes, 0)
}

// (37, 38) A pickup whose save fails is reported; the file keeps the last
// safe point, so a restart before a retry finds the character still
// stranded and still listed. Retrying in the window writes the pickup once.
func TestRescueWithAFailedSave(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	rand.Seed(52)
	s, dir, fallen, pit := strandedAndRescuers(t, 2)
	s.DungeonAction(app.ActInspect, 0)
	before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	os.Chmod(dir, 0o555)
	v := s.DungeonAction(app.ActPickUp, 1)
	os.Chmod(dir, 0o755)
	if !strings.Contains(v.Error, "Could not save") || v.Unsaved == "" || len(v.Party) != 3 {
		t.Fatalf("the failed save: error %q unsaved %q party %d", v.Error, v.Unsaved, len(v.Party))
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, before) {
		t.Fatal("the file changed although the save failed")
	}

	r := reopened(t, s, dir)
	rg := r.ctrl.Game
	if !r.real || rg.Town.Party.Size() != 2 || !rg.MazeInspecting || len(rg.MazeInspectFound) != 6 {
		t.Fatalf("restart before the retry: real %v party %d inspecting %v found %d", r.real, rg.Town.Party.Size(), rg.MazeInspecting, len(rg.MazeInspectFound))
	}
	for _, ch := range rg.Town.Roster.Characters {
		if ch.Name == fallen[0].Name && (ch.Stranded == nil || *ch.Stranded != pit) {
			t.Fatalf("%s after the restart: %v", ch.Name, ch.Stranded)
		}
	}

	for i := 0; i < 2; i++ {
		if v := s.SaveNow(); v.Error != "" || v.Unsaved != "" || len(v.Party) != 3 {
			t.Fatalf("retry %d: error %q unsaved %q party %d", i, v.Error, v.Unsaved, len(v.Party))
		}
		party, _ := diskParty(t, dir)
		if len(party) != 3 || party[2] != fallen[0].Name {
			t.Fatalf("retry %d on disk: %v", i, party)
		}
		if got := strandedOnDiskAt(t, dir, pit); len(got) != 5 {
			t.Fatalf("retry %d stranded on disk: %v", i, got)
		}
		mustMatchDisk(t, s, dir, "after the retry")
	}
}

// (36) A restart while carrying the rescued resumes the expedition with
// them in the party, the rest stranded and still listed by Inspect.
func TestRestartWhileCarryingTheRescued(t *testing.T) {
	rand.Seed(53)
	s, dir, fallen, pit := strandedAndRescuers(t, 1)
	s.DungeonAction(app.ActInspect, 0)
	s.DungeonAction(app.ActPickUp, 1)
	s.DungeonAction(app.ActPickUp, 1)
	for i := 0; i < 2; i++ {
		r := reopened(t, s, dir)
		rv := r.State()
		rg := r.ctrl.Game
		if !r.real || rv.Dungeon == nil || len(rv.Party) != 3 || rv.Dungeon.Inspect == nil || len(rv.Dungeon.Inspect.Found) != 4 || rv.Dungeon.Inspect.Room != 3 {
			t.Fatalf("restart %d: real %v party %d inspect %+v", i, r.real, len(rv.Party), rv.Dungeon)
		}
		if rg.Town.Party.Members[1].Name != fallen[0].Name || rg.Town.Party.Members[1].Status != fallen[0].Status || rg.Town.Party.Members[1].Stranded != nil {
			t.Fatalf("restart %d: carried %+v", i, rg.Town.Party.Members[1])
		}
		if got := strandedOnDiskAt(t, dir, pit); len(got) != 4 {
			t.Fatalf("restart %d stranded: %v", i, got)
		}
	}
}

// (44) A practice expedition may rescue copies: they join the practice
// party; the roster, the castle party and the save are untouched, during
// and after.
func TestPracticeRescueKeepsNothing(t *testing.T) {
	rand.Seed(54)
	s, dir := partySession(t, false)
	pit, fallen := pitAhead(t, s)
	s.DungeonAction(app.ActForward, 0)
	helper := engine.NewCharacter("HELPER", engine.Human, engine.Fighter, engine.Neutral)
	helper.HP, helper.MaxHP = 50, 50
	rosterWith(t, s, helper)
	walkTo(t, s, "tavern")
	if v := s.JoinParty(6); v.PendingJoin != "" {
		s.SubmitJoinPassword("")
	}
	walkTo(t, s, "edge")
	before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if v := s.PracticeExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	v := s.DungeonAction(app.ActInspect, 0)
	if v.Dungeon.Inspect == nil || len(v.Dungeon.Inspect.Found) != 6 {
		t.Fatalf("practice inspect: %+v", v.Dungeon.Inspect)
	}
	for i := 0; i < 5; i++ {
		if v = s.DungeonAction(app.ActPickUp, 1); v.Error != "" {
			t.Fatal(v.Error)
		}
	}
	if len(v.Dungeon.Party) != 6 || s.practice.Game.Town.Party.Size() != 6 {
		t.Fatalf("practice party %d", len(v.Dungeon.Party))
	}
	check := func(when string) {
		for _, ch := range fallen {
			if ch.Stranded == nil || *ch.Stranded != pit || !ch.InMaze {
				t.Fatalf("%s: %s in the roster: %v", when, ch.Name, ch.Stranded)
			}
		}
		if s.ctrl.Game.Town.Party.Size() != 1 {
			t.Fatalf("%s: castle party %d", when, s.ctrl.Game.Town.Party.Size())
		}
		if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, before) {
			t.Fatalf("%s: practice changed the save", when)
		}
	}
	check("during practice")
	s.AbandonPractice()
	check("after practice")
}
