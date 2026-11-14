package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// Real expeditions: the saved party in the saved game, saved after every
// action (M6b-1) with the expedition's checkpoint (M6b-2), so a restart
// during an expedition resumes it at its last safe point.

// onDisk reads the roster file as saved.
func onDisk(t *testing.T, dir string) engine.SaveState {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var st engine.SaveState
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatal(err)
	}
	return st
}

// mustMatchDisk fails unless the roster file holds exactly the session's
// roster and party: the last action was saved.
func mustMatchDisk(t *testing.T, s *Session, dir string, when string) {
	t.Helper()
	st := onDisk(t, dir)
	mem, _ := json.Marshal(s.ctrl.Game.Town.Roster.Characters)
	disk, _ := json.Marshal(st.Roster)
	if !bytes.Equal(mem, disk) {
		t.Fatalf("%s: the roster on disk differs from the game", when)
	}
	var names []string
	for _, m := range s.ctrl.Game.Town.Party.Members {
		if m != nil {
			names = append(names, m.Name)
		}
	}
	// Out on an expedition the party is in the expedition section, and the
	// castle party list is empty.
	var exp engine.Expedition
	if len(st.Expedition) > 0 {
		json.Unmarshal(st.Expedition, &exp)
	}
	switch {
	case s.real && (len(st.PartyNames) != 0 || !slices.Equal(exp.Party, names)):
		t.Fatalf("%s: on disk castle party %v, expedition %v; in the game %v", when, st.PartyNames, exp.Party, names)
	case !s.real && (len(st.Expedition) != 0 || !slices.Equal(names, st.PartyNames)):
		t.Fatalf("%s: party on disk %v (expedition %d bytes), in the game %v", when, st.PartyNames, len(st.Expedition), names)
	}
}

// standing is what a restart must bring back for each party member: what
// the save holds for them (known spells are rebuilt on load, so they are
// left out).
type standing struct {
	Name                     string
	HP, XP, Gold, Poison     int
	Status                   engine.Status
	Items                    [8]engine.Possession
	ItemCount                int
	MageSpells, PriestSpells [7]int
}

func partyStanding(g *engine.GameState) []standing {
	var out []standing
	for _, m := range g.Town.Party.Members {
		if m != nil {
			out = append(out, standing{m.Name, m.HP, m.XP, m.Gold, m.PoisonAmt, m.Status, m.Items, m.ItemCount, m.MageSpells, m.PriestSpells})
		}
	}
	return out
}

// restart is the app quitting and starting again while s plays: a new
// session opens the folder (reopened) and resumes. While s is out on a real
// expedition the party must be back at its last safe point: in the maze or
// Camp exactly as s shows it, or, if s is in an encounter, where s stands
// with the encounter over. Otherwise the party is in the castle. Either
// way every member is exactly as saved.
func restart(t *testing.T, s *Session, dir string, when string) *Session {
	t.Helper()
	g := s.ctrl.Game
	want := partyStanding(g)
	inCombat := g.Phase == engine.PhaseCombat
	before := s.State()
	where := fmt.Sprintf("L%d (%d,%d) %v", g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing)
	r := reopened(t, s, dir)
	v := r.State()
	rg := r.ctrl.Game
	switch {
	case v.Fatal != "" || v.Recovery != "":
		t.Fatalf("%s: after restart: fatal %q recovery %q", when, v.Fatal, v.Recovery)
	case s.real:
		if !r.real || !v.Real || v.Screen != "dungeon" || rg.Combat != nil {
			t.Fatalf("%s: not resumed: real %v screen %q", when, r.real, v.Screen)
		}
		if got := fmt.Sprintf("L%d (%d,%d) %v", rg.MazeLevel, rg.PlayerX, rg.PlayerY, rg.Facing); got != where {
			t.Fatalf("%s: resumed at %s, was at %s", when, got, where)
		}
		if !inCombat && !reflect.DeepEqual(v.Dungeon, before.Dungeon) {
			t.Fatalf("%s: the dungeon after restart\n%+v\nwas\n%+v", when, v.Dungeon, before.Dungeon)
		}
		if inCombat && !strings.Contains(v.Notice, "unclaimed rewards are lost") && !strings.Contains(v.Notice, "chest and its rewards are lost") {
			t.Fatalf("%s: notice %q", when, v.Notice)
		}
		if v.Title != resumedTitle || !strings.HasPrefix(v.Notice, "Your party has returned to the dungeon.") ||
			strings.HasSuffix(v.Notice, resumedStranded) != (len(rg.StrandedSites()) > 0) {
			t.Fatalf("%s: resumed with %q %q", when, v.Title, v.Notice)
		}
	default:
		if r.real || v.Screen != "roster" || v.Expedition || rg.Phase != engine.PhaseTown {
			t.Fatalf("%s: after restart: screen %q expedition %v phase %v", when, v.Screen, v.Expedition, rg.Phase)
		}
		if v.Title != "" {
			t.Fatalf("%s: in town headed %q", when, v.Title)
		}
	}
	if got := partyStanding(rg); !slices.Equal(got, want) {
		t.Fatalf("%s: after restart the party is\n %+v\nwant\n %+v", when, got, want)
	}
	return r
}

func partyXP(g *engine.GameState) (xp, gold int) {
	for _, m := range g.Town.Party.Members {
		if m != nil {
			xp, gold = xp+m.XP, gold+m.Gold
		}
	}
	return
}

// Real expeditions played through the bridge: every action is saved, the
// party's losses and rewards are kept, the expedition ends in the castle
// (the stairs, or the engine's defeat), and a restart at any point resumes
// it exactly where it was saved.
//
// At least twelve seeds are played, and more, up to sixty, until every
// outcome below has been seen: a win (experience and gold) comes in about
// one encounter in ten, and which seeds have one changes whenever the
// orders a front end may give do (COMBAT-1 stopped offering CALFO in
// combat, and the first twelve no longer included a win).
func TestRealExpeditionsThroughTheBridge(t *testing.T) {
	seen := map[string]int{}
	want := []string{"encounter", "back in the maze after combat", "prompt stairs", "camp",
		"perished", "climbed out", "experience gained", "gold gained", "party changed", "restarted mid-expedition"}
	covered := func() bool {
		for _, k := range want {
			if seen[k] == 0 {
				return false
			}
		}
		return true
	}
	for seed := int64(1); seed <= 60 && (seed <= 12 || !covered()); seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir := partySession(t, seed%4 == 0) // some weak parties, to see defeats
		xp0, gold0 := partyXP(s.ctrl.Game)
		hp0 := partyStanding(s.ctrl.Game)

		v := s.RealExpedition()
		if v.Error != "" || v.Screen != "dungeon" || !v.Real || !v.Expedition || v.Dungeon.Phase != "camp" {
			t.Fatalf("seed %d: start: error %q screen %q real %v", seed, v.Error, v.Screen, v.Real)
		}
		steps := 0
		v = explore(t, s, rng, seen, func(v View) {
			steps++
			if v.Unsaved != "" {
				t.Fatalf("seed %d: unsaved: %s", seed, v.Unsaved)
			}
			mustMatchDisk(t, s, dir, "after an action")
			if s.real && steps%151 == 0 {
				restart(t, s, dir, "mid-expedition")
				seen["restarted mid-expedition"]++
			}
		})
		xp, gold := partyXP(s.ctrl.Game)
		if xp > xp0 {
			seen["experience gained"]++
		}
		if gold > gold0 {
			seen["gold gained"]++
		}
		if !slices.Equal(partyStanding(s.ctrl.Game), hp0) {
			seen["party changed"]++
		}
		switch {
		case s.real:
			seen["still walking"]++
		case strings.Contains(v.Notice, "PERISHED"):
			seen["perished"]++
			for _, m := range s.ctrl.Game.Town.Party.Members {
				if m.IsAlive() {
					t.Fatalf("seed %d: perished with %s alive", seed, m.Name)
				}
			}
		case strings.Contains(v.Notice, "back in the castle"):
			seen["climbed out"]++
		default:
			t.Fatalf("seed %d: ended with %q", seed, v.Notice)
		}
		if !s.real && (v.Screen != "roster" || v.Expedition || v.Real) {
			t.Fatalf("seed %d: after the expedition: screen %q", seed, v.Screen)
		}
		restart(t, s, dir, "at the end")
	}
	t.Logf("covered: %v", seen)
	for _, k := range want {
		if seen[k] == 0 {
			t.Errorf("not covered: %s", k)
		}
	}
}

// realEncounter starts a real expedition, walks out of Camp and puts the
// party in an encounter guarding a chest (test setup, as a practice
// encounter does; the entrance's encounters never have one).
func realEncounter(t *testing.T, s *Session) {
	t.Helper()
	if v := s.RealExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	if v := s.DungeonAction(app.ActNo, 0); v.Dungeon == nil || v.Dungeon.Prompt != "" {
		t.Fatalf("not walking: %+v", v.Dungeon)
	}
	g := s.ctrl.Game
	for try := 0; try < 50 && (g.Combat == nil || len(g.Combat.Groups) == 0); try++ {
		g.Combat = engine.NewCombat(g)
	}
	g.Combat.HasChest = true
	g.Phase = engine.PhaseCombat
}

// fightTo plays the encounter until the combat view's next step is one of
// stop, and returns that view (ok false if combat ended first).
func fightTo(t *testing.T, s *Session, rng *rand.Rand, stop ...string) (View, bool) {
	t.Helper()
	v := s.State()
	for i := 0; i < 2000; i++ {
		if v.Screen != "combat" {
			return v, false
		}
		c := v.Combat
		if slices.Contains(stop, c.Next) {
			return v, true
		}
		switch c.Next {
		case app.NextStart:
			v = s.CombatStart()
		case app.NextFriendly:
			v = s.CombatFriendly(true)
		case app.NextOrders:
			v = giveOrder(t, s, c, rng)
		case app.NextConfirm:
			v = s.CombatCommit()
		case app.NextAdvance:
			v = s.CombatAdvance()
		case app.NextChoice:
			v = s.CombatChooseBoon(1)
		case app.NextEndRound:
			v = s.CombatEndRound()
		case app.NextChestResult:
			v = s.ChestContinue()
		case app.NextLeave: // over before reaching stop (a defeat, or no chest)
			return v, false
		default:
			t.Fatalf("unexpected next %q", c.Next)
		}
	}
	t.Fatal("combat never got there")
	return v, false
}

// Interruptions: wherever the app stops (walking, in the middle of a round,
// at the chest, after the rewards), a restart resumes the expedition at its
// last safe point with every committed loss kept, the encounter over, and
// its rewards only if they had been given, never twice.
func TestRealExpeditionInterruptions(t *testing.T) {
	// Walking.
	rand.Seed(7)
	s, dir := partySession(t, false)
	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	for i := 0; i < 3; i++ {
		s.DungeonAction(app.ActForward, 0)
		s.DungeonAction(app.ActRight, 0)
	}
	if !s.real {
		t.Fatal("the walk ended early")
	}
	restart(t, s, dir, "walking")

	reached := map[string]bool{}
	for seed := int64(1); seed <= 30 && len(reached) < 3; seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir := partySession(t, false)
		xp0, gold0 := partyXP(s.ctrl.Game)
		realEncounter(t, s)

		// In the middle of a round: the round's damage and spells are
		// committed; no reward.
		v, ok := fightTo(t, s, rng, app.NextAdvance)
		if !ok {
			continue
		}
		mustMatchDisk(t, s, dir, "mid-round")
		r := restart(t, s, dir, "mid-round")
		if xp, gold := partyXP(r.ctrl.Game); xp != xp0 || gold != gold0 {
			t.Fatalf("seed %d: rewards after an interrupted round", seed)
		}
		reached["mid-round"] = true

		// At the chest, before it is opened or left: no reward yet.
		v, ok = fightTo(t, s, rng, app.NextChest)
		if !ok {
			continue
		}
		r = restart(t, s, dir, "at the chest")
		if xp, gold := partyXP(r.ctrl.Game); xp != xp0 || gold != gold0 {
			t.Fatalf("seed %d: rewards before the chest was resolved", seed)
		}
		reached["at the chest"] = true

		// Open it; once the rewards are given (the victory screen), a
		// restart keeps them, once.
		for _, m := range v.Combat.Chest.Members {
			if m.CanAct {
				v = s.ChestAction("open", m.Slot)
				break
			}
		}
		v, ok = fightTo(t, s, rng, app.NextLeave)
		if !ok || v.Combat.Phase != "victory" {
			continue
		}
		xp1, gold1 := partyXP(s.ctrl.Game)
		if xp1 <= xp0 {
			t.Fatalf("seed %d: no experience at victory", seed)
		}
		restart(t, s, dir, "after the rewards")
		v = s.CombatLeave()
		if v.Screen != "dungeon" || !s.real {
			t.Fatalf("seed %d: not back in the maze: %q", seed, v.Screen)
		}
		r = restart(t, s, dir, "back in the maze")
		if xp, gold := partyXP(r.ctrl.Game); xp != xp1 || gold != gold1 {
			t.Fatalf("seed %d: rewards changed after leaving combat: %d/%d, want %d/%d", seed, xp, gold, xp1, gold1)
		}
		reached["after the rewards"] = true
	}
	for _, k := range []string{"mid-round", "at the chest", "after the rewards"} {
		if !reached[k] {
			t.Errorf("never interrupted %s", k)
		}
	}
}

// A failed save is reported, stays reported until a save succeeds, and the
// action is not repeated: the mage's spell, cast in a round whose save
// failed, is spent once when the next save succeeds.
func TestRealExpeditionSaveFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	for seed := int64(1); seed <= 20; seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir := partySession(t, false)
		realEncounter(t, s)
		v, ok := fightTo(t, s, rng, app.NextOrders)
		if !ok {
			continue
		}
		var mage *engine.Character
		for _, m := range s.ctrl.Game.Town.Party.Members {
			if m.Class == engine.Mage {
				mage = m
			}
		}
		slots := mage.MageSpells[0]
		// Every member parries but the mage, who casts KATINO.
		for _, m := range v.Combat.Party {
			s.CombatSelect(m.Slot)
			r := OrderRequest{Slot: m.Slot, Action: "parry", Item: -1}
			if m.Name == mage.Name {
				r = OrderRequest{Slot: m.Slot, Action: "spell", Spell: "KATINO", Group: v.Combat.Groups[0].Number, Item: -1}
			}
			if v = s.CombatOrder(r); v.Error != "" {
				t.Fatalf("order for %s: %s", m.Name, v.Error)
			}
		}
		if v = s.CombatFinishOrders(); v.Error != "" {
			t.Fatal(v.Error)
		}
		before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))

		os.Chmod(dir, 0o555)
		v = s.CombatCommit()
		os.Chmod(dir, 0o755)
		if !strings.Contains(v.Error, "Could not save") || v.Unsaved == "" {
			t.Fatalf("failed save not reported: error %q unsaved %q", v.Error, v.Unsaved)
		}
		if after, _ := os.ReadFile(filepath.Join(dir, "roster1.json")); !bytes.Equal(after, before) {
			t.Fatal("the file changed although the save failed")
		}
		if mage.MageSpells[0] != slots-1 {
			t.Fatalf("spell slots %d, want %d (the round ran once)", mage.MageSpells[0], slots-1)
		}
		if v := s.State(); v.Unsaved == "" || v.Combat.Next != app.NextAdvance {
			t.Fatalf("after the failure: unsaved %q next %q", v.Unsaved, v.Combat.Next)
		}

		v = s.CombatAdvance()
		if v.Error != "" || v.Unsaved != "" {
			t.Fatalf("next save: error %q unsaved %q", v.Error, v.Unsaved)
		}
		mustMatchDisk(t, s, dir, "after the next save")
		for _, ch := range onDisk(t, dir).Roster {
			if ch.Name == mage.Name && ch.MageSpells[0] != slots-1 {
				t.Fatalf("saved spell slots %d, want %d", ch.MageSpells[0], slots-1)
			}
		}
		return
	}
	t.Fatal("no seed reached order entry")
}

// While the party is out, town actions are refused and change nothing on
// disk; practice cannot start; a real expedition cannot be abandoned or
// started twice. Practice still leaves the roster byte-identical.
func TestRealExpeditionRefusals(t *testing.T) {
	rand.Seed(3)
	s, dir := partySession(t, false)
	if v := s.PracticeExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	if v := s.RealExpedition(); v.Error == "" || s.real {
		t.Fatal("a real expedition started during practice")
	}
	s.AbandonPractice()

	if v := s.RealExpedition(); v.Error != "" || !s.real {
		t.Fatalf("start: %q", v.Error)
	}
	path := filepath.Join(dir, "roster1.json")
	before, _ := os.ReadFile(path)
	for name, op := range map[string]func() View{
		"second expedition":  s.RealExpedition,
		"practice":           s.PracticeExpedition,
		"practice encounter": func() View { return s.PracticeEncounter(false) },
		"abandon":            s.AbandonPractice,
		"rename":             func() View { return s.RenameCharacter(0, "RENAMED") },
		"join":               func() View { return s.JoinParty(0) },
		"leave party":        func() View { return s.LeaveParty(0) },
		"delete":             func() View { return s.DeleteCharacter("AGUI", "") },
		"create":             func() View { return s.StartCreation("NEWBIE") },
		"reset roster":       s.ResetRoster,
		"restore backup":     func() View { return s.RestoreBackup("x.json") },
		"reload":             s.ReloadRoster,
	} {
		v := op()
		if v.Error == "" || !s.real || v.Screen != "dungeon" {
			t.Errorf("%s: error %q real %v screen %q", name, v.Error, s.real, v.Screen)
		}
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, before) {
		t.Fatal("a refused action changed the save")
	}
	if s.ctrl.Game.Town.Roster.Characters[0].Name != "AGUI" {
		t.Fatal("a refused rename changed the roster")
	}
}

// outAndRestarted starts a real expedition, walks out of Camp and returns
// the folder with the expedition saved and the session closed, as if the
// app had quit there.
func outAndRestarted(t *testing.T) string {
	t.Helper()
	rand.Seed(5)
	s, dir := partySession(t, false)
	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	s.DungeonAction(app.ActRight, 0)
	s.shutdown()
	return dir
}

// editSave changes the saved file as JSON.
func editSave(t *testing.T, dir string, edit func(st map[string]any)) {
	t.Helper()
	path := filepath.Join(dir, "roster1.json")
	raw, _ := os.ReadFile(path)
	var st map[string]any
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatal(err)
	}
	edit(st)
	raw, _ = json.MarshalIndent(st, "", "  ")
	os.WriteFile(path, raw, 0o644)
}

// A saved expedition that cannot be resumed (a member deleted outside
// Istary) is shown, kept, and blocks only what would overwrite it; the
// player can return its party to the castle, after a backup.
func TestUnresumableExpeditionIsKeptAndRecoverable(t *testing.T) {
	dir := outAndRestarted(t)
	editSave(t, dir, func(st map[string]any) {
		roster := st["roster"].([]any)
		st["roster"] = roster[1:] // AGUI, the first member, is gone
	})
	path := filepath.Join(dir, "roster1.json")
	s := NewSession(dir)
	defer s.shutdown()
	v := s.State()
	if !strings.Contains(v.Recovery, "AGUI is no longer on the roster") || !v.CanReturn || v.Real || v.Screen != "roster" {
		t.Fatalf("recovery %q canReturn %v real %v screen %q", v.Recovery, v.CanReturn, v.Real, v.Screen)
	}
	for name, op := range map[string]func() View{
		"new expedition": func() View { walkTo(t, s, "edge"); return s.RealExpedition() },
		"reset":          func() View { walkTo(t, s, "settings"); return s.ResetRoster() },
	} {
		if v := op(); !strings.Contains(v.Error, "already saved") {
			t.Errorf("%s: %q", name, v.Error)
		}
	}
	walkTo(t, s, "training")
	if v := s.DeleteCharacter("BGUI", ""); !strings.Contains(v.Error, "OUT ON AN EXPEDITION") {
		t.Errorf("deleting a member: %q", v.Error)
	}
	walkTo(t, s, "tavern")
	if v := s.JoinParty(0); !strings.Contains(v.Error, "OUT") && v.PendingJoin != "" {
		t.Errorf("a member joined the castle party: %+v", v.Party)
	}
	raw, _ := os.ReadFile(path)
	if !bytes.Contains(raw, []byte(`"expedition"`)) {
		t.Fatal("the expedition was dropped")
	}

	v = s.ReturnExpeditionToCastle()
	if v.Error != "" || v.Recovery != "" || len(v.Party) != 5 || !strings.Contains(v.Notice, "backed up") {
		t.Fatalf("return: error %q recovery %q party %d notice %q", v.Error, v.Recovery, len(v.Party), v.Notice)
	}
	raw, _ = os.ReadFile(path)
	if bytes.Contains(raw, []byte(`"expedition"`)) {
		t.Fatal("the section stayed after the return")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "backups"))
	if len(entries) != 1 || !strings.Contains(entries[0].Name(), "before-return") {
		t.Fatalf("backups %v", entries)
	}
}

// A newer version's expedition is never resumed or ended here, and it is
// written back unchanged when the castle saves.
func TestOtherVersionExpeditionIsLeftAlone(t *testing.T) {
	dir := outAndRestarted(t)
	editSave(t, dir, func(st map[string]any) {
		e := st["expedition"].(map[string]any)
		e["version"] = 2.0
		e["future"] = "kept"
	})
	s := NewSession(dir)
	defer s.shutdown()
	v := s.State()
	if !strings.Contains(v.Recovery, "another version") || v.CanReturn || v.Real {
		t.Fatalf("recovery %q canReturn %v", v.Recovery, v.CanReturn)
	}
	if v := s.ReturnExpeditionToCastle(); v.Error == "" {
		t.Fatal("a newer expedition was ended")
	}
	createMage(t, s, "radagast") // a castle action that saves
	raw, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if !bytes.Contains(raw, []byte(`"future": "kept"`)) || !bytes.Contains(raw, []byte(`"version": 2`)) {
		t.Fatal("the newer expedition was not written back unchanged")
	}
}

// A failed save at a maze step is reported, the file keeps the previous
// checkpoint (a restart then resumes there, not at the unsaved step), and
// the step is not replayed: the next save records where the party is.
func TestFailedSaveKeepsThePreviousCheckpoint(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	rand.Seed(9)
	s, dir := partySession(t, false)
	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	saved := s.ctrl.Game.Facing

	os.Chmod(dir, 0o555)
	v := s.DungeonAction(app.ActRight, 0)
	os.Chmod(dir, 0o755)
	if !strings.Contains(v.Error, "Could not save") || v.Unsaved == "" {
		t.Fatalf("error %q unsaved %q", v.Error, v.Unsaved)
	}
	turned := s.ctrl.Game.Facing
	if turned == saved {
		t.Fatal("the turn did not happen")
	}
	r := reopened(t, s, dir)
	if r.ctrl.Game.Facing != saved {
		t.Fatalf("a restart resumed facing %v; the last saved checkpoint faces %v", r.ctrl.Game.Facing, saved)
	}
	v = s.State()
	if s.ctrl.Game.Facing != turned || v.Unsaved == "" {
		t.Fatal("the failed step was undone or forgotten")
	}
	v = s.DungeonAction(app.ActLeft, 0)
	if v.Error != "" || v.Unsaved != "" {
		t.Fatalf("next save: %q %q", v.Error, v.Unsaved)
	}
	r = reopened(t, s, dir)
	if r.ctrl.Game.Facing != s.ctrl.Game.Facing || s.ctrl.Game.Facing != saved {
		t.Fatalf("after the next save: resumed %v, playing %v", r.ctrl.Game.Facing, s.ctrl.Game.Facing)
	}
}

// Practice never writes an expedition: with a real expedition saved and
// not resumable, or with none, practice leaves the file byte-identical.
func TestPracticeWritesNoExpedition(t *testing.T) {
	rand.Seed(4)
	s, dir := partySession(t, false)
	path := filepath.Join(dir, "roster1.json")
	before, _ := os.ReadFile(path)
	s.PracticeExpedition()
	for i := 0; i < 30 && s.practice != nil; i++ {
		s.DungeonAction(app.ActLeaveCamp, 0)
		s.DungeonAction(app.ActNo, 0)
		s.DungeonAction(app.ActForward, 0)
	}
	s.AbandonPractice()
	if after, _ := os.ReadFile(path); !bytes.Equal(after, before) {
		t.Fatal("practice changed the save")
	}
}

// "Try Resuming Again" works from anywhere in town while an expedition
// cannot be resumed, and resumes it once it can; it is refused otherwise.
func TestTryResumeAgain(t *testing.T) {
	dir := outAndRestarted(t)
	path := filepath.Join(dir, "roster1.json")
	good, _ := os.ReadFile(path)
	editSave(t, dir, func(st map[string]any) {
		st["roster"] = st["roster"].([]any)[1:] // AGUI, the first member, is gone
	})
	s := NewSession(dir)
	defer s.shutdown()
	walkTo(t, s, "tavern")
	if v := s.TryResumeAgain(); v.Error != "" || v.Recovery == "" || v.Location != "castle" {
		t.Fatalf("still unresumable: %q, recovery %q at %q", v.Error, v.Recovery, v.Location)
	}
	os.WriteFile(path, good, 0o644) // the member is back
	walkTo(t, s, "edge")
	if v := s.TryResumeAgain(); v.Error != "" || v.Recovery != "" || !v.Real || v.Screen != "dungeon" {
		t.Fatalf("resumable again: %q, recovery %q, real %v, screen %q", v.Error, v.Recovery, v.Real, v.Screen)
	}
	if v := s.TryResumeAgain(); v.Error == "" || !v.Real {
		t.Fatalf("with nothing to resume: %q", v.Error)
	}
}

// On a real expedition, SaveNow after a failed save records where the
// party now stands and saves it: a restart resumes there, not at the
// previous checkpoint, and the step is not taken twice.
func TestSaveNowOnAnExpedition(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	rand.Seed(9)
	s, dir := partySession(t, false)
	s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	os.Chmod(dir, 0o555)
	v := s.DungeonAction(app.ActRight, 0)
	os.Chmod(dir, 0o755)
	if v.Unsaved == "" {
		t.Fatal("the failed save is not reported")
	}
	turned := s.ctrl.Game.Facing
	if v = s.SaveNow(); v.Error != "" || v.Unsaved != "" || s.ctrl.Game.Facing != turned {
		t.Fatalf("saving again: %q %q, facing %v (was %v)", v.Error, v.Unsaved, s.ctrl.Game.Facing, turned)
	}
	mustMatchDisk(t, s, dir, "saved again")
	if r := reopened(t, s, dir); r.ctrl.Game.Facing != turned || !r.real {
		t.Fatalf("a restart resumes facing %v (real %v), want %v", r.ctrl.Game.Facing, r.real, turned)
	}
}
