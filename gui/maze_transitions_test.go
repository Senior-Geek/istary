package main

// The maze picture across phase transitions (M-ART-1A.3). The dungeon
// screen draws the perspective (MazeView) whenever view.dungeon.phase is
// "maze", and Camp's own screen in Camp; the combat screen has none. These
// tests walk every way into and out of an encounter through the Session
// methods the window calls (by a step onto a fight square; escape, victory
// with the chest opened or left, defeat; a restart at each point) and check
// what the view says at every step:
//   - the dungeon screen is shown only in the maze or in Camp: never with a
//     phase it would have no picture for;
//   - in the maze, the corridor is the engine's Sight at that moment;
//   - after an encounter, the party is back in the maze with the corridor
//     of where it stands, and no action of the picture's is needed.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// sightCorridor is the corridor the dungeon view should hold now.
func sightCorridor(g *engine.GameState) ([]app.CorridorView, bool) {
	sight, dark := g.Sight()
	var want []app.CorridorView
	for _, q := range sight {
		want = append(want, app.CorridorView{Left: q.Left, Right: q.Right, Front: q.Front, LeftAhead: q.LeftAhead, RightAhead: q.RightAhead})
	}
	return want, dark
}

// screens records every (screen, dungeon phase) a view shows, and fails on
// a dungeon screen in a phase other than the maze or Camp, or a maze view
// whose corridor is not Sight's.
type screens struct {
	t    *testing.T
	seen map[string]int
}

func (r *screens) check(s *Session, v View, when string) View {
	r.t.Helper()
	key := v.Screen
	if v.Screen == "dungeon" {
		if v.Dungeon == nil {
			r.t.Fatalf("%s: a dungeon screen with no dungeon view", when)
		}
		key += "/" + v.Dungeon.Phase
		switch v.Dungeon.Phase {
		case "maze":
			g := s.active().Game
			want, dark := sightCorridor(g)
			if !reflect.DeepEqual(v.Dungeon.Corridor, want) || v.Dungeon.Dark != dark {
				r.t.Fatalf("%s: corridor %+v dark %v, Sight %+v dark %v", when, v.Dungeon.Corridor, v.Dungeon.Dark, want, dark)
			}
		case "camp":
			if v.Camp == nil {
				r.t.Fatalf("%s: Camp with no Camp screen", when)
			}
		default:
			r.t.Fatalf("%s: the dungeon screen in phase %q, which it has no picture for", when, v.Dungeon.Phase)
		}
	}
	if v.Screen == "combat" && v.Combat != nil {
		key += "/" + v.Combat.Phase
	}
	r.seen[key]++
	return v
}

// playOut plays the encounter to its end, retreating if retreat is set,
// and checks every view on the way.
func playOut(t *testing.T, s *Session, r *screens, rng *rand.Rand, retreat bool, chest string, when string) View {
	t.Helper()
	v := s.State()
	for i := 0; i < 3000 && v.Screen == "combat"; i++ {
		c := v.Combat
		switch c.Next {
		case app.NextStart:
			v = s.CombatStart()
		case app.NextFriendly:
			v = s.CombatFriendly(true)
		case app.NextOrders:
			if retreat && c.Retreat != "" {
				v = s.CombatRetreat()
			} else {
				v = giveOrder(t, s, c, rng)
			}
		case app.NextConfirm:
			v = s.CombatCommit()
		case app.NextAdvance:
			v = s.CombatAdvance()
		case app.NextChoice:
			v = s.CombatChooseBoon(1)
		case app.NextEndRound:
			v = s.CombatEndRound()
		case app.NextChest:
			for _, m := range c.Chest.Members {
				if m.CanAct {
					v = s.ChestAction(chest, m.Slot)
					break
				}
			}
		case app.NextChestResult:
			v = s.ChestContinue()
		case app.NextLeave:
			v = s.CombatLeave()
		default:
			t.Fatalf("%s: next %q", when, c.Next)
		}
		v = r.check(s, v, fmt.Sprintf("%s, step %d (%s)", when, i, c.Next))
	}
	if v.Screen == "combat" {
		t.Fatalf("%s: the encounter never ended", when)
	}
	return v
}

// stepIntoFight walks the party (out of Camp, not up the stairs) onto a
// fight square ahead of it, with the encounter it starts.
func stepIntoFight(t *testing.T, s *Session, r *screens, real bool) View {
	t.Helper()
	if real {
		r.check(s, s.RealExpedition(), "set out (real)")
	} else {
		walkTo(t, s, "edge")
		r.check(s, s.PracticeExpedition(), "set out (practice)")
	}
	r.check(s, s.DungeonAction(app.ActLeaveCamp, 0), "leave Camp")
	r.check(s, s.DungeonAction(app.ActNo, 0), "not up the stairs")
	g := s.active().Game
	g.Facing = engine.North
	if g.WallAhead() != data.WallOpen {
		g.Facing = engine.East
	}
	x, y := (g.PlayerX+g.Facing.DX()+20)%20, (g.PlayerY+g.Facing.DY()+20)%20
	g.FightMap = [20][20]bool{}
	g.FightMap[x][y] = true
	v := r.check(s, s.DungeonAction(app.ActForward, 0), "step onto the fight square")
	if v.Screen != "combat" {
		t.Fatalf("the step started no encounter: %q", v.Screen)
	}
	return v
}

// resumed is the app started again on dir after s stopped: s lets go of the
// save folder and the new session, which holds it, carries on.
func resumed(t *testing.T, s *Session, dir string) *Session {
	t.Helper()
	s.lock.release()
	r := NewSession(dir)
	if v := r.State(); v.Fatal != "" || v.Recovery != "" {
		t.Fatalf("resumed: fatal %q recovery %q", v.Fatal, v.Recovery)
	}
	return r
}

func TestMazePictureAcrossTransitions(t *testing.T) {
	r := &screens{t: t, seen: map[string]int{}}
	outcomes := map[string]int{}
	for seed := int64(1); seed <= 4; seed++ {
		for _, c := range []struct {
			name          string
			real, retreat bool
			weak          bool
			chest         string
			guarded       bool // an encounter guarding a chest (the entrance's never do)
		}{
			{"real, guarded chest opened", true, false, false, "open", true},
			{"real, guarded chest left", true, false, false, "leave", true},
			{"real, retreat", true, true, false, "leave", false},
			{"real, fight", true, false, false, "open", false},
			{"practice, fight", false, false, false, "open", false},
			{"real, weak party", true, false, true, "open", false},
			{"practice, weak party", false, false, true, "open", false},
		} {
			rand.Seed(seed)
			rng := rand.New(rand.NewSource(seed))
			s, dir := partySession(t, c.weak)
			when := fmt.Sprintf("seed %d, %s", seed, c.name)
			if c.guarded {
				realEncounter(t, s)
				r.check(s, s.State(), when+", the encounter")
			} else {
				stepIntoFight(t, s, r, c.real)
			}
			if seed%2 == 0 && c.real { // interrupted mid-encounter, then resumed
				s = resumed(t, s, dir)
				r.check(s, s.State(), when+", restarted in the encounter")
			}
			v := playOut(t, s, r, rng, c.retreat, c.chest, when)
			outcome := v.Screen
			if v.Screen == "dungeon" {
				outcome += "/" + v.Dungeon.Phase
				if c.real {
					s = resumed(t, s, dir)
					r.check(s, s.State(), when+", restarted after the encounter")
				}
				// Back in the maze: a turn is drawn from where the party stands.
				r.check(s, s.DungeonAction(app.ActRight, 0), when+", a turn after")
			}
			outcomes[c.name+" -> "+outcome]++
		}
	}
	t.Logf("screens seen: %v", r.seen)
	t.Logf("outcomes: %v", outcomes)
	// Every way out was taken: escape, victory with the chest opened and
	// left, defeat (back to the Castle's roster), real and practice; and the
	// chest, victory and defeat screens on the way.
	for _, k := range []string{"combat/chest", "combat/victory", "combat/defeat", "dungeon/maze", "dungeon/camp", "roster"} {
		if r.seen[k] == 0 {
			t.Errorf("never saw %s", k)
		}
	}
	for _, k := range []string{"real, retreat -> dungeon/maze", "real, fight -> dungeon/maze", "real, guarded chest opened -> dungeon/maze", "real, guarded chest left -> dungeon/maze",
		"practice, fight -> dungeon/maze", "real, weak party -> roster", "practice, weak party -> roster"} {
		if outcomes[k] == 0 {
			t.Errorf("never ended %s", k)
		}
	}
	keys := slices.Sorted(maps.Keys(r.seen))
	for _, k := range keys {
		if strings.HasPrefix(k, "dungeon/") && k != "dungeon/maze" && k != "dungeon/camp" {
			t.Errorf("the dungeon screen in %s", k)
		}
	}
}

// transitionStep is one view of the recorded transitions, as much of it as
// the maze picture uses: the screen, and on the dungeon screen its phase,
// corridor, darkness and light, the window's maze look and the position.
type transitionStep struct {
	When      string             `json:"when"`
	Screen    string             `json:"screen"`
	Combat    string             `json:"combat,omitempty"` // the combat screen's phase
	Phase     string             `json:"phase,omitempty"`
	Corridor  []app.CorridorView `json:"corridor,omitempty"`
	Dark      bool               `json:"dark,omitempty"`
	Spells    []string           `json:"spells,omitempty"`
	Place     string             `json:"place,omitempty"`
	Facing    string             `json:"facing,omitempty"`
	MazeStyle string             `json:"mazeStyle"`
	MazeZoom  string             `json:"mazeZoom"`
}

func stepOf(when string, v View) transitionStep {
	st := transitionStep{When: when, Screen: v.Screen, MazeStyle: v.Settings.MazeStyle, MazeZoom: v.Settings.MazeZoom}
	if v.Combat != nil {
		st.Combat = v.Combat.Phase
	}
	if v.Screen == "dungeon" && v.Dungeon != nil {
		st.Phase, st.Corridor, st.Dark, st.Spells = v.Dungeon.Phase, v.Dungeon.Corridor, v.Dungeon.Dark, v.Dungeon.Spells
	}
	if v.Position != nil {
		st.Place, st.Facing = v.Position.Place, v.Position.Facing
	}
	return st
}

// TestMazeTransitionFixture records, for the frontend's tests
// (testdata/maze-transitions.json), the views of one real expedition, with
// the maze look set to Classic at Close: walking into an encounter and
// escaping it; into another, won, with its chest opened; a restart; Camp and
// back. Each phase change is recorded once. Run with ISTARY_UPDATE_FIXTURES=1
// to rewrite the file after a deliberate change.
func TestMazeTransitionFixture(t *testing.T) {
	r := &screens{t: t, seen: map[string]int{}}
	var steps []transitionStep
	fought := map[string]bool{} // the combat phases recorded in this encounter
	add := func(s *Session, when string, v View) View {
		t.Helper()
		v = r.check(s, v, when)
		st := stepOf(when, v)
		if st.Screen != "combat" {
			clear(fought)
			if n := len(steps); n == 0 || steps[n-1].Phase != st.Phase || st.Phase == "maze" {
				steps = append(steps, st)
			}
		} else if !fought[st.Combat] {
			fought[st.Combat] = true
			steps = append(steps, st)
		}
		return v
	}
	var s *Session
	var dir string
	for seed := int64(1); ; seed++ {
		if seed > 30 {
			t.Fatal("no seed escapes the first encounter and wins the second")
		}
		steps = nil
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir = partySession(t, false)
		add(s, "the look chosen: Classic at Close", s.SetMazeLook("classic", "close"))
		add(s, "set out", s.RealExpedition())
		add(s, "leave Camp", s.DungeonAction(app.ActLeaveCamp, 0))
		add(s, "not up the stairs", s.DungeonAction(app.ActNo, 0))
		g := s.ctrl.Game
		g.Facing = engine.North
		if g.WallAhead() != data.WallOpen {
			g.Facing = engine.East
		}
		x, y := (g.PlayerX+g.Facing.DX()+20)%20, (g.PlayerY+g.Facing.DY()+20)%20
		g.FightMap = [20][20]bool{}
		g.FightMap[x][y] = true
		add(s, "a step onto a fight square", s.DungeonAction(app.ActForward, 0))
		v := s.State()
		for i := 0; i < 400 && v.Screen == "combat"; i++ {
			switch v.Combat.Next {
			case app.NextStart:
				v = s.CombatStart()
			case app.NextOrders:
				v = s.CombatRetreat()
			case app.NextConfirm:
				v = s.CombatCommit()
			case app.NextAdvance:
				v = s.CombatAdvance()
			case app.NextEndRound:
				v = s.CombatEndRound()
			case app.NextLeave:
				v = s.CombatLeave()
			default:
				v = s.State()
			}
			v = add(s, "escaping", v)
		}
		if v.Screen != "dungeon" || v.Dungeon.Phase != "maze" {
			continue
		}
		add(s, "a turn after the escape", s.DungeonAction(app.ActRight, 0))
		// A second encounter, guarding a chest; won, the chest opened.
		for try := 0; try < 50 && (g.Combat == nil || len(g.Combat.Groups) == 0); try++ {
			g.Combat = engine.NewCombat(g)
		}
		g.Combat.HasChest = true
		g.Phase = engine.PhaseCombat
		v = add(s, "an encounter guarding a chest", s.State())
		for i := 0; i < 2000 && v.Screen == "combat"; i++ {
			c := v.Combat
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
			case app.NextChest:
				for _, m := range c.Chest.Members {
					if m.CanAct {
						v = s.ChestAction("open", m.Slot)
						break
					}
				}
			case app.NextChestResult:
				v = s.ChestContinue()
			case app.NextLeave:
				v = s.CombatLeave()
			}
			v = add(s, "fighting", v)
		}
		if v.Screen != "dungeon" || !slices.ContainsFunc(steps, func(st transitionStep) bool { return st.Combat == "victory" }) ||
			!slices.ContainsFunc(steps, func(st transitionStep) bool { return st.Combat == "chest" }) {
			continue
		}
		break
	}
	s = resumed(t, s, dir)
	add(s, "restarted", s.State())
	add(s, "Camp", s.DungeonAction(app.ActCamp, 0))
	add(s, "out of Camp", s.DungeonAction(app.ActLeaveCamp, 0))
	add(s, "a turn", s.DungeonAction(app.ActLeft, 0))
	add(s, "a step", s.DungeonAction(app.ActForward, 0))
	s.shutdown()

	got, _ := json.MarshalIndent(steps, "", " ")
	got = append(got, '\n')
	path := filepath.Join("frontend", "src", "lib", "testdata", "maze-transitions.json")
	if os.Getenv("ISTARY_UPDATE_FIXTURES") != "" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with ISTARY_UPDATE_FIXTURES=1)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("maze-transitions.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
