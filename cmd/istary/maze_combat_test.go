package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/Senior-Geek/istary/engine"
)

// adventurers returns a fixed six-member party for maze and combat tests.
func adventurers() []*engine.Character {
	mk := func(name string, race engine.Race, class engine.Class, hp, str, agi int) *engine.Character {
		c := character(name, race, class, engine.Neutral)
		c.HP, c.MaxHP = hp, hp
		c.Strength, c.Agility = str, agi
		c.Level = 2
		return c
	}
	priest := mk("PRIEST", engine.Dwarf, engine.Priest, 14, 12, 9)
	priest.Alignment = engine.Good
	priest.Piety = 15
	priest.PriestSpells, priest.MaxPriestSpells = [7]int{2}, [7]int{2}
	mage := mk("MAGE", engine.Elf, engine.Mage, 7, 8, 11)
	mage.IQ = 16
	mage.MageSpells, mage.MaxMageSpells = [7]int{2}, [7]int{2}
	return []*engine.Character{
		mk("FIGHTER A", engine.Human, engine.Fighter, 22, 17, 10),
		mk("FIGHTER B", engine.Dwarf, engine.Fighter, 24, 18, 8),
		mk("THIEF", engine.Hobbit, engine.Thief, 12, 11, 16),
		priest,
		mage,
		mk("FIGHTER C", engine.Human, engine.Fighter, 20, 16, 10),
	}
}

// partyInMaze puts the test party into the B1F entrance via Edge of Town
// and the camp, the way a player does.
func partyInMaze(t *testing.T) (*engine.GameState, *driver) {
	t.Helper()
	g := newGame(t, 1)
	for _, c := range adventurers() {
		g.Town.Roster.Add(c)
		g.Town.Party.Members = append(g.Town.Party.Members, c)
	}
	d := newDriverWithScreen(t, g)
	d.keys("E").keys("M")
	if g.Phase != engine.PhaseCamp {
		t.Fatalf("M at Edge of Town should open camp, phase = %v", g.Phase)
	}
	d.keys("L")
	if g.Phase != engine.PhaseMaze || g.PlayerX != 0 || g.PlayerY != 0 || g.Facing != engine.North {
		t.Fatalf("leaving camp: phase=%v pos=(%d,%d) facing=%v", g.Phase, g.PlayerX, g.PlayerY, g.Facing)
	}
	// The entrance is the up staircase, which asks before anything else.
	if g.MazeMessage2 != "TAKE THEM (Y/N) ?" {
		t.Fatalf("expected the stairs prompt at the entrance, got %q / %q", g.MazeMessage, g.MazeMessage2)
	}
	d.keys("N")
	return g, d
}

func TestEntranceStairsPrompt(t *testing.T) {
	g, d := partyInMaze(t)
	if g.MazeMessage2 != "" {
		t.Fatalf("N should dismiss the stairs prompt, got %q", g.MazeMessage2)
	}
	// Stepping back onto the stairs and answering Y returns to the castle.
	d.keys("F").keys("RR").keys("F")
	if g.PlayerX != 0 || g.PlayerY != 0 || g.MazeMessage2 != "TAKE THEM (Y/N) ?" {
		t.Fatalf("back on stairs: pos=(%d,%d) prompt=%q", g.PlayerX, g.PlayerY, g.MazeMessage2)
	}
	d.keys("Y")
	if g.Phase != engine.PhaseTown || g.Town.Location != engine.Castle {
		t.Errorf("Y at the up stairs: phase=%v location=%v", g.Phase, g.Town.Location)
	}
	for _, m := range g.Town.Party.Members {
		if m.InMaze {
			t.Errorf("%s still marked InMaze after leaving the dungeon", m.Name)
		}
	}
}

func TestWallBumpSaysOuch(t *testing.T) {
	g, d := partyInMaze(t)
	d.keys("RR") // face south: the B1F start square has a wall there
	d.keys("F")
	if g.ViewportMsg != "OUCH!" || g.PlayerX != 0 || g.PlayerY != 0 {
		t.Errorf("bump: msg=%q pos=(%d,%d)", g.ViewportMsg, g.PlayerX, g.PlayerY)
	}
	d.keys("K")
	if g.ViewportMsg != "OUCH!" || g.PlayerY != 0 {
		t.Errorf("kicking a solid wall should also fail: msg=%q pos=(%d,%d)", g.ViewportMsg, g.PlayerX, g.PlayerY)
	}
	d.keys("L")
	if g.ViewportMsg != "" {
		t.Errorf("any action should clear the viewport message, got %q", g.ViewportMsg)
	}
}

func TestMazeToggles(t *testing.T) {
	g, d := partyInMaze(t)
	d.keys("Q")
	if !g.QuickPlot || g.MazeMessage != "QUICK PLOT ON" {
		t.Errorf("Q: quick=%v msg=%q", g.QuickPlot, g.MazeMessage)
	}
	d.keys("M")
	if !g.ShowMap {
		t.Fatal("M should show the map")
	}
	d.keys("X")
	if g.ShowMap {
		t.Error("any key should close the map")
	}
	d.keys("T").keys("250").enter()
	if g.MazeDelay != 250 || g.MazeDelayInput {
		t.Errorf("T)IME: delay=%d input=%v", g.MazeDelay, g.MazeDelayInput)
	}
	d.keys("C")
	if g.Phase != engine.PhaseCamp {
		t.Errorf("C should camp, phase = %v", g.Phase)
	}
}

// Seeded walk through B1F. Each line is the state after one key press; the
// trace stops when the walk leaves maze mode (encounter, stairs, etc.).
func TestMazeWalkTrace(t *testing.T) {
	seed(11)
	g, d := partyInMaze(t)
	walk := "FFFFRFFFFFLFFFFKFFRFFFFLLFFFFRFFFFFFKFFFRFFFFLFFFF"
	var out strings.Builder
	for i, k := range walk {
		d.keys(string(k))
		fmt.Fprintf(&out, "%02d %c pos=(%d,%d) face=%-5v phase=%d light=%d view=%q msg=%q msgs=%d\n",
			i, k, g.PlayerX, g.PlayerY, g.Facing, g.Phase, g.LightLevel, g.ViewportMsg, g.MazeMessage, len(g.MazeMessages))
		if g.Phase != engine.PhaseMaze {
			break
		}
		// Page through any square messages so the walk can continue.
		for n := 0; len(g.MazeMessages) > 0 && n < 10; n++ {
			d.enter()
		}
	}
	checkGolden(t, "maze_walk.golden", []byte(out.String()))
}

// combatTick fires the main loop's combat timer with the delay already
// elapsed.
func combatTick(g *engine.GameState) {
	combatTimerTick(g, time.Hour)
}

// fightToTheEnd plays a combat with a fixed policy: front row fights group 1,
// everyone else parries, chests are left alone. It returns a trace of every
// round.
func fightToTheEnd(t *testing.T, g *engine.GameState, d *driver) string {
	t.Helper()
	var out strings.Builder
	party := func() string {
		var parts []string
		for _, m := range g.Town.Party.Members {
			parts = append(parts, fmt.Sprintf("%s %d/%d %v", m.Name, m.HP, m.MaxHP, m.Status))
		}
		return strings.Join(parts, "; ")
	}
	c := g.Combat
	for _, grp := range c.Groups {
		fmt.Fprintf(&out, "group: %d x %s\n", grp.AliveCount(), g.Scenario.Monsters[grp.MonsterID].Name)
	}
	loggedRound := 0
	for step := 0; step < 5000 && g.Combat != nil && g.Phase == engine.PhaseCombat; step++ {
		c = g.Combat
		switch c.Phase {
		case engine.CombatFriendly:
			d.keys("F")
		case engine.CombatChoose:
			actor := c.CurrentActor
			d.keys("F")
			if c.SelectingGroup {
				d.keys("1")
			}
			if c.Phase == engine.CombatChoose && c.CurrentActor == actor {
				d.keys("P")
			}
		case engine.CombatConfirm:
			d.enter()
		case engine.CombatChest:
			d.keys("L")
		default:
			if c.Phase == engine.CombatExecute && c.Round != loggedRound {
				loggedRound = c.Round
				fmt.Fprintf(&out, "round %d\n", c.Round)
				for _, m := range c.Messages {
					if m != engine.ActionSeparator {
						fmt.Fprintf(&out, "  %s\n", m)
					}
				}
				fmt.Fprintf(&out, "  party: %s\n", party())
			}
			combatTick(g)
		}
	}
	fmt.Fprintf(&out, "end: phase=%d town=%q maze=%q\n", g.Phase, g.Town.Message, g.MazeMessage)
	fmt.Fprintf(&out, "party: %s\n", party())
	for _, m := range g.Town.Party.Members {
		fmt.Fprintf(&out, "  %s xp=%d gold=%d items=%d\n", m.Name, m.XP, m.Gold, m.ItemCount)
	}
	return out.String()
}

// Seeded encounters on B1F played to completion. Pins combat order, hit and
// damage rolls, monster AI, rewards, and the end-of-combat transition.
func TestCombatTrace(t *testing.T) {
	var out strings.Builder
	for _, s := range []int64{21, 22, 23} {
		seed(s)
		g, d := partyInMaze(t)
		g.Combat = engine.NewCombat(g)
		g.Phase = engine.PhaseCombat
		if len(g.Combat.Groups) == 0 {
			t.Fatalf("seed %d: NewCombat produced no monsters", s)
		}
		fmt.Fprintf(&out, "== seed %d\n%s", s, fightToTheEnd(t, g, d))
	}
	checkGolden(t, "combat_trace.golden", []byte(out.String()))
}

// A seeded random encounter reached by walking, not by calling NewCombat.
func TestWalkIntoEncounter(t *testing.T) {
	g, d := partyInMaze(t)
	g.FightMap[0][1] = true // force a fight on the square north of the stairs
	seed(31)
	d.keys("F")
	if g.Phase != engine.PhaseCombat || g.Combat == nil {
		t.Fatalf("stepping onto a fight-map square should start combat, phase=%v", g.Phase)
	}
	if g.Combat.EncounterType != 0 && g.Combat.EncounterType != 2 {
		t.Errorf("EncounterType = %d", g.Combat.EncounterType)
	}
	if g.FightMap[0][1] {
		t.Error("the fight room should be cleared once combat starts")
	}
	trace := fightToTheEnd(t, g, d)
	if g.Combat != nil {
		t.Errorf("combat did not finish:\n%s", trace)
	}
}

// A longer wander driven by its own random source (so the game's global
// sequence is untouched). It covers special squares, poison/regeneration
// ticks and encounter triggers; each encounter is fought to the end and the
// wander resumes.
func TestMazeWanderTrace(t *testing.T) {
	seed(77)
	g, d := partyInMaze(t)
	picker := rand.New(rand.NewSource(77))
	moves := "FFFFLRK"
	var out strings.Builder
	for i := 0; i < 400 && g.Phase != engine.PhaseTown; i++ {
		k := moves[picker.Intn(len(moves))]
		d.keys(string(k))
		fmt.Fprintf(&out, "%03d %c L%d (%d,%d) %-5v phase=%d view=%q msg=%q|%q msgs=%d\n",
			i, k, g.MazeLevel+1, g.PlayerX, g.PlayerY, g.Facing, g.Phase, g.ViewportMsg, g.MazeMessage, g.MazeMessage2, len(g.MazeMessages))
		for n := 0; len(g.MazeMessages) > 0 && n < 10; n++ {
			d.enter()
		}
		switch {
		case g.MazeMessage2 == "TAKE THEM (Y/N) ?":
			d.keys("N")
		case g.MazeSearchYN:
			d.keys("N")
		case g.Phase == engine.PhaseCombat:
			out.WriteString(fightToTheEnd(t, g, d))
		case g.Phase == engine.PhaseCamp:
			d.keys("L")
		}
	}
	checkGolden(t, "maze_wander.golden", []byte(out.String()))
}

func TestCombatTimerDelay(t *testing.T) {
	seed(21)
	g, _ := partyInMaze(t)
	g.Combat = engine.NewCombat(g)
	g.Phase = engine.PhaseCombat
	g.Combat.Surprised = 0

	// Default PAUSE1 is 1500ms.
	if combatTimerTick(g, 1499*time.Millisecond) || g.Combat.Phase != engine.CombatInit {
		t.Fatal("combat advanced before the default 1500ms delay")
	}
	if !combatTimerTick(g, 1500*time.Millisecond) || g.Combat.Phase != engine.CombatChoose {
		t.Fatalf("combat should advance to Choose after 1500ms, phase=%v", g.Combat.Phase)
	}
	// Nothing to auto-advance while the player chooses actions.
	if combatTimerTick(g, time.Hour) {
		t.Error("timer should not advance during action selection")
	}

	// T)IME delay: MazeDelay*3/10 ms, minimum 50ms.
	g.Combat.Phase = engine.CombatInit
	g.MazeDelay = 100 // 30ms -> clamped to 50ms
	if combatTimerTick(g, 49*time.Millisecond) || !combatTimerTick(g, 50*time.Millisecond) {
		t.Error("T)IME delay should clamp to a 50ms minimum")
	}
}
