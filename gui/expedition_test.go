package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// Real expeditions (M6b-1): the saved party in the saved game, saved after
// every action. Until M6b-2 the save holds no position, so a restart during
// an expedition finds the party in the castle with everything saved up to
// the last action.

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
	if !slices.Equal(names, st.PartyNames) {
		t.Fatalf("%s: party on disk %v, in the game %v", when, st.PartyNames, names)
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

// restart opens a new session on the same folder, as when the app starts
// again, and checks the M6b-1 outcome: the party is in the castle, as saved.
func restart(t *testing.T, dir string, want []standing, when string) *Session {
	t.Helper()
	r := NewSession(dir)
	v := r.State()
	if v.Fatal != "" || v.Screen != "roster" || v.Expedition || r.ctrl.Game.Phase != engine.PhaseTown {
		t.Fatalf("%s: after restart: screen %q expedition %v phase %v fatal %q", when, v.Screen, v.Expedition, r.ctrl.Game.Phase, v.Fatal)
	}
	if got := partyStanding(r.ctrl.Game); !slices.Equal(got, want) {
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
// (the stairs, or the engine's defeat), and a restart at any point finds
// the party in the castle exactly as last saved.
func TestRealExpeditionsThroughTheBridge(t *testing.T) {
	seen := map[string]int{}
	for seed := int64(1); seed <= 12; seed++ {
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
				restart(t, dir, partyStanding(s.ctrl.Game), "mid-expedition")
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
		restart(t, dir, partyStanding(s.ctrl.Game), "at the end")
	}
	t.Logf("covered: %v", seen)
	for _, k := range []string{"encounter", "back in the maze after combat", "prompt stairs", "camp",
		"perished", "climbed out", "experience gained", "gold gained", "party changed", "restarted mid-expedition"} {
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

// Interruptions under the M6b-1 rule: wherever the app stops (walking, in
// the middle of a round, at the chest, after the rewards), the party is in
// the castle after a restart, with every committed loss kept, and with the
// encounter's rewards only if they had been given, never twice.
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
	restart(t, dir, partyStanding(s.ctrl.Game), "walking")

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
		r := restart(t, dir, partyStanding(s.ctrl.Game), "mid-round")
		if xp, gold := partyXP(r.ctrl.Game); xp != xp0 || gold != gold0 {
			t.Fatalf("seed %d: rewards after an interrupted round", seed)
		}
		reached["mid-round"] = true

		// At the chest, before it is opened or left: no reward yet.
		v, ok = fightTo(t, s, rng, app.NextChest)
		if !ok {
			continue
		}
		r = restart(t, dir, partyStanding(s.ctrl.Game), "at the chest")
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
		restart(t, dir, partyStanding(s.ctrl.Game), "after the rewards")
		v = s.CombatLeave()
		if v.Screen != "dungeon" || !s.real {
			t.Fatalf("seed %d: not back in the maze: %q", seed, v.Screen)
		}
		r = restart(t, dir, partyStanding(s.ctrl.Game), "back in the maze")
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
