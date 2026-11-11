package main

// The R-2 acceptance walkthrough (brief section 13), through the Session as
// the front end drives it, on a scratch save folder. Run it with
//
//	go test -run TestR2Walkthrough -v
//
// to read each step. Expedition A (six weak adventurers) meets a real
// encounter and is wiped out; rescue expedition B (three) walks until A is
// in its area, takes three of them, is closed and started again, and
// brings them home by the stairs; rescue expedition C (three others) takes
// the other three home. Every adventurer is then accounted for once, and
// again after a restart. TestR2RescuersDefeatedWhileCarrying shows a rescue
// party wiped out with the rescued.

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

func TestR2Walkthrough(t *testing.T) {
	rand.Seed(41)
	rng := rand.New(rand.NewSource(41))
	s, dir := partySession(t, true) // A: six weak characters, at the Edge of Town
	var b, c []*engine.Character
	for i := 0; i < 6; i++ {
		ch := engine.NewCharacter(fmt.Sprintf("B%d", i+1), engine.Human, engine.Fighter, engine.Neutral)
		if i >= 3 {
			ch.Name = fmt.Sprintf("C%d", i-2)
		}
		ch.HP, ch.MaxHP, ch.Level, ch.AC = 400, 400, 12, -2
		if i < 3 {
			b = append(b, ch)
		} else {
			c = append(c, ch)
		}
	}
	rosterWith(t, s, append(append([]*engine.Character(nil), b...), c...)...)
	g := s.ctrl.Game
	partyA := append([]*engine.Character(nil), g.Town.Party.Members...)
	for i, m := range partyA { // something to keep: gold and an unequipped item each
		m.Gold = 100 * (i + 1)
		m.Items[m.ItemCount] = engine.Possession{ItemIndex: 1 + i, Identified: true}
		m.ItemCount++
	}
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	t.Logf("A: %v (HP 2, AC 10, gold and an item each); rescuers B %v, C %v", namesOf(partyA), namesOf(b), namesOf(c))

	// Expedition A.
	fought := defeatA(t, s, rng)
	kept := map[string]standing{}
	for _, m := range partyA {
		kept[m.Name] = standingOf(m)
	}
	v := s.State()
	t.Logf("A is stranded at %s: %s", fought, partyLine(partyA))
	strandedView(t, v, namesOf(partyA), fought)
	strandedOnDisk(t, dir, namesOf(partyA), fought)

	// Rescue expedition B: three rescuers.
	setOut(t, s, b, 6)
	v = walkUntil(t, s, rng, func(g *engine.GameState) bool { return inAreaOf(g, fought) }, false)
	here, _ := g.Place()
	t.Logf("B (%v) finds A's square %s in its area at %s", namesOf(b), fought, here)
	v = s.DungeonAction(app.ActInspect, 0)
	t.Logf("Inspect: %s", foundLine(v))
	if len(v.Dungeon.Inspect.Found) != 6 || v.Dungeon.Inspect.Room != 3 {
		t.Fatalf("B's inspect: %+v", v.Dungeon.Inspect)
	}
	for _, name := range []string{"BGUI", "DGUI", "FGUI"} {
		v = s.DungeonAction(app.ActPickUp, foundNamed(v, name))
		if v.Error != "" {
			t.Fatalf("pick %s: %s", name, v.Error)
		}
		t.Logf("  %s", strings.Join(v.Dungeon.Lines, " "))
	}
	v = s.DungeonAction(app.ActPickUp, 1)
	t.Logf("a fourth: %q; Inspect: %s", v.Error, foundLine(v))
	if !strings.Contains(v.Error, "YOUR PARTY HAS 6") || len(v.Party) != 6 {
		t.Fatalf("a fourth: %q party %d", v.Error, len(v.Party))
	}
	s.DungeonAction(app.ActLeave, 0)
	leftBehind := strandedOnDiskAt(t, dir, fought)
	t.Logf("B's party now %v; still stranded at %s: %v", namesOf(g.Town.Party.Members), fought, leftBehind)

	// Close and start again, carrying the rescued.
	s.shutdown()
	s = NewSession(dir)
	defer s.shutdown()
	g = s.ctrl.Game
	v = s.State()
	if !s.real || len(v.Party) != 6 || v.Screen != "dungeon" {
		t.Fatalf("after the restart: real %v party %d screen %q", s.real, len(v.Party), v.Screen)
	}
	here, _ = g.Place()
	t.Logf("after a restart: the expedition resumes at %s with %v; stranded at %s: %v", here, namesOf(g.Town.Party.Members), fought, strandedOnDiskAt(t, dir, fought))
	if got := strandedOnDiskAt(t, dir, fought); strings.Join(got, ",") != strings.Join(leftBehind, ",") {
		t.Fatalf("stranded after the restart: %v, was %v", got, leftBehind)
	}

	v = walkUntil(t, s, rng, func(g *engine.GameState) bool { return false }, true)
	v = s.DungeonAction(app.ActYes, 0)
	t.Logf("B climbs out: screen %q, castle party %v", v.Screen, namesOf(g.Town.Party.Members))
	mustBeHome(t, s, []string{"BGUI", "DGUI", "FGUI"}, kept)

	// Rescue expedition C: the other three rescuers, for the rest.
	walkTo(t, s, "tavern")
	for len(g.Town.Party.Members) > 0 {
		s.LeaveParty(0)
	}
	setOut(t, s, c, 9)
	walkUntil(t, s, rng, func(g *engine.GameState) bool { return inAreaOf(g, fought) }, false)
	v = s.DungeonAction(app.ActInspect, 0)
	t.Logf("C (%v) inspects: %s", namesOf(c), foundLine(v))
	if len(v.Dungeon.Inspect.Found) != 3 || v.Dungeon.Inspect.Room != 3 {
		t.Fatalf("C's inspect: %+v", v.Dungeon.Inspect)
	}
	for len(v.Dungeon.Inspect.Found) > 0 {
		v = s.DungeonAction(app.ActPickUp, 1)
		if v.Error != "" {
			t.Fatal(v.Error)
		}
		t.Logf("  %s", strings.Join(v.Dungeon.Lines, " "))
	}
	s.DungeonAction(app.ActLeave, 0)
	walkUntil(t, s, rng, func(g *engine.GameState) bool { return false }, true)
	v = s.DungeonAction(app.ActYes, 0)
	t.Logf("C climbs out: castle party %v", namesOf(g.Town.Party.Members))
	mustBeHome(t, s, []string{"AGUI", "CGUI", "EGUI"}, kept)

	// Everyone accounted for, once; and again after a restart.
	for round := 0; round < 2; round++ {
		rg := s.ctrl.Game
		seen := map[string]int{}
		for _, ch := range rg.Town.Roster.Characters {
			seen[ch.Name]++
			if ch.Stranded != nil || ch.OutOfCastle() && !app.InParty(rg.Town, ch) {
				t.Fatalf("%s: stranded %v, out %v", ch.Name, ch.Stranded, ch.OutOfCastle())
			}
		}
		for _, m := range partyA {
			if seen[m.Name] != 1 {
				t.Fatalf("%s is in the roster %d times", m.Name, seen[m.Name])
			}
		}
		if len(rg.Town.Roster.Characters) != 12 || rg.Expedition != nil {
			t.Fatalf("roster %d, expedition %v", len(rg.Town.Roster.Characters), rg.Expedition != nil)
		}
		t.Logf("round %d: roster of %d, each once; none stranded; castle party %v", round, len(rg.Town.Roster.Characters), namesOf(rg.Town.Party.Members))
		if round == 0 {
			s.shutdown()
			s = NewSession(dir)
			defer s.shutdown()
		}
	}
}

// defeatA walks the session's party until an encounter wipes it out, and
// returns the square where it fought.
func defeatA(t *testing.T, s *Session, rng *rand.Rand) engine.Place {
	t.Helper()
	g := s.ctrl.Game
	v := s.RealExpedition()
	s.DungeonAction(app.ActLeaveCamp, 0)
	v = s.DungeonAction(app.ActNo, 0)
	var fought engine.Place
	defeated := false
	for step := 0; step < 3000 && s.real; step++ {
		switch v.Screen {
		case "dungeon":
			d := v.Dungeon
			switch {
			case d.Prompt == "stairs", d.Prompt == "search":
				v = s.DungeonAction(app.ActNo, 0)
			case d.Prompt == "messages":
				v = s.DungeonAction(app.ActContinue, 0)
			case d.Prompt != "":
				t.Fatalf("prompt %q", d.Prompt)
			case g.WallAhead() == data.WallOpen && rng.Intn(4) > 0:
				v = s.DungeonAction(app.ActForward, 0)
			default:
				v = s.DungeonAction(app.ActRight, 0)
			}
		case "combat":
			fought, _ = g.Place()
			v, _ = fightTo(t, s, rng)
			defeated = v.Screen == "combat" && v.Combat.Phase == "defeat"
			if v.Screen == "combat" {
				v = s.CombatLeave()
			}
		default:
			t.Fatalf("screen %q", v.Screen)
		}
	}
	if s.real || !defeated {
		t.Fatalf("A was not defeated (real %v)", s.real)
	}
	return fought
}

// setOut forms the castle party from chars (roster indices from first) and
// takes it out of Camp to the entrance on a real expedition.
func setOut(t *testing.T, s *Session, chars []*engine.Character, first int) {
	t.Helper()
	walkTo(t, s, "tavern")
	for i := range chars {
		v := s.JoinParty(first + i)
		if v.PendingJoin != "" {
			v = s.SubmitJoinPassword("")
		}
		if v.Error != "" {
			t.Logf("join %d: %s (location %q)", first+i, v.Error, v.Location)
		}
	}
	if names := namesOf(s.ctrl.Game.Town.Party.Members); strings.Join(names, ",") != strings.Join(namesOf(chars), ",") {
		t.Fatalf("party %v", names)
	}
	walkTo(t, s, "edge")
	if v := s.RealExpedition(); v.Error != "" {
		t.Fatal(v.Error)
	}
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
}

// inAreaOf reports whether p is in the party's rescue area.
func inAreaOf(g *engine.GameState, p engine.Place) bool {
	level, x, y := p.Internal()
	if level != g.MazeLevel {
		return false
	}
	area, ok := engine.RescueArea(g.Scenario, g.MazeLevel, g.PlayerX, g.PlayerY)
	return ok && area[x][y]
}

// walkUntil walks the real expedition on its level, fighting what it
// meets, until done says so; with home, it walks to the entrance instead
// and stops at the stairs' question there. Each step is the first of a
// shortest way through open walls (found afresh after every step, so a
// spinner or a teleporter only changes the way).
func walkUntil(t *testing.T, s *Session, rng *rand.Rand, done func(*engine.GameState) bool, home bool) View {
	t.Helper()
	g := s.ctrl.Game
	v := s.State()
	for step := 0; step < 4000; step++ {
		if v.Screen == "combat" {
			v, _ = fightTo(t, s, rng)
			if v.Screen == "combat" {
				v = s.CombatLeave()
			}
			if !s.real {
				t.Fatalf("the rescuers fell: %s", v.Notice)
			}
			continue
		}
		if v.Screen != "dungeon" {
			t.Fatalf("screen %q (%s)", v.Screen, v.Error)
		}
		d := v.Dungeon
		atEntrance := g.MazeLevel == 0 && g.PlayerX == 0 && g.PlayerY == 0
		switch {
		case d.Prompt == "stairs" && home && atEntrance:
			return v
		case d.Prompt == "stairs", d.Prompt == "search":
			v = s.DungeonAction(app.ActNo, 0)
			continue
		case d.Prompt == "messages":
			v = s.DungeonAction(app.ActContinue, 0)
			continue
		case d.Prompt == "buttons":
			v = s.DungeonAction(app.ActButton, 1)
			continue
		case d.Prompt != "":
			t.Fatalf("prompt %q", d.Prompt)
		}
		if !home && done(g) {
			return v
		}
		tx, ty := 0, 0
		if !home {
			tx, ty = -1, -1 // any square: done decides
		}
		dir, ok := firstStep(g, tx, ty, done, home)
		if home && atEntrance { // the stairs ask only on arriving: step off first
			for _, d := range []engine.Direction{engine.North, engine.East, engine.South, engine.West} {
				c := g.Scenario.Mazes.Levels[0].Cells[0][0]
				if [...]data.WallType{c.N, c.E, c.S, c.W}[d] == data.WallOpen {
					dir, ok = d, true
					break
				}
			}
		}
		if !ok {
			t.Fatalf("no way on from L%d (%d,%d)", g.MazeLevel+1, g.PlayerX, g.PlayerY)
		}
		for i := 0; i < 4 && g.Facing != dir; i++ {
			v = s.DungeonAction(app.ActRight, 0)
		}
		if g.Facing != dir {
			continue // something interrupted the turn
		}
		v = s.DungeonAction(app.ActForward, 0)
	}
	t.Fatal("never got there")
	return v
}

// firstStep is the direction of the first step of a shortest way through
// open walls on the party's level, to (tx, ty) when home, otherwise to the
// nearest square at which done holds.
func firstStep(g *engine.GameState, tx, ty int, done func(*engine.GameState) bool, home bool) (engine.Direction, bool) {
	cells := g.Scenario.Mazes.Levels[g.MazeLevel].Cells
	type sq struct{ x, y int }
	first := map[sq]engine.Direction{}
	start := sq{g.PlayerX, g.PlayerY}
	seen := map[sq]bool{start: true}
	queue := []sq{start}
	goal := func(p sq) bool {
		if home {
			return p.x == tx && p.y == ty
		}
		x, y := g.PlayerX, g.PlayerY
		g.PlayerX, g.PlayerY = p.x, p.y
		ok := done(g)
		g.PlayerX, g.PlayerY = x, y
		return ok
	}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p != start && goal(p) {
			return first[p], true
		}
		c := cells[p.y][p.x]
		for _, d := range []engine.Direction{engine.North, engine.East, engine.South, engine.West} {
			w := [...]data.WallType{c.N, c.E, c.S, c.W}[d]
			if w != data.WallOpen {
				continue
			}
			n := sq{(p.x + d.DX() + 20) % 20, (p.y + d.DY() + 20) % 20}
			if seen[n] {
				continue
			}
			seen[n] = true
			if p == start {
				first[n] = d
			} else {
				first[n] = first[p]
			}
			queue = append(queue, n)
		}
	}
	return 0, false
}

func foundLine(v View) string {
	if v.Dungeon == nil || v.Dungeon.Inspect == nil {
		return "(not inspecting)"
	}
	var parts []string
	for _, f := range v.Dungeon.Inspect.Found {
		what := "can be picked up"
		if !f.CanPick {
			what = f.Refusal
		}
		parts = append(parts, fmt.Sprintf("%d) %s %s %s (%s)", f.N, f.Name, f.Class, f.Status, what))
	}
	return fmt.Sprintf("room for %d; %s", v.Dungeon.Inspect.Room, strings.Join(parts, "; "))
}

// mustBeHome fails unless the game is in the castle with names in the
// castle party, back from the maze, stranded nowhere, with the condition
// and possessions they were stranded with.
func mustBeHome(t *testing.T, s *Session, names []string, kept map[string]standing) {
	t.Helper()
	g := s.ctrl.Game
	if s.real || g.Phase != engine.PhaseTown || g.Expedition != nil {
		t.Fatalf("not home: real %v phase %v", s.real, g.Phase)
	}
	for _, n := range names {
		var ch *engine.Character
		for _, m := range g.Town.Party.Members {
			if m.Name == n {
				ch = m
			}
		}
		if ch == nil || ch.Stranded != nil || ch.InMaze {
			t.Fatalf("%s not home: %+v", n, ch)
		}
		was, now := kept[n], standingOf(ch)
		if now.Status != was.Status || now.Items != was.Items || now.Gold != was.Gold || now.XP != was.XP {
			t.Fatalf("%s changed: %+v, was %+v", n, now, was)
		}
		t.Logf("  %s home: %v, HP %d, gold %d, %d items, as when stranded", n, ch.Status, ch.HP, ch.Gold, ch.ItemCount)
	}
}

// A rescue party wiped out while carrying the rescued is stranded where it
// falls, the rescued with it: they are not put back where they were found.
// Those not picked up stay where they were.
func TestR2RescuersDefeatedWhileCarrying(t *testing.T) {
	rand.Seed(61)
	s, dir, fallen, pit := strandedAndRescuers(t, 3)
	g := s.ctrl.Game
	v := s.DungeonAction(app.ActInspect, 0)
	for i := 0; i < 3; i++ {
		v = s.DungeonAction(app.ActPickUp, 1)
	}
	s.DungeonAction(app.ActLeave, 0)
	carried := namesOf(g.Town.Party.Members)
	t.Logf("B picked up three at the entrance: party %v", carried)

	// Onto the pit (the rescuers can take it), then on to a square made a
	// second pit, with the rescuers at 1 HP.
	pl, px, py := pit.Internal()
	for i := 0; i < 4 && (g.PlayerX+g.Facing.DX()+20)%20 != px || (g.PlayerY+g.Facing.DY()+20)%20 != py; i++ {
		s.DungeonAction(app.ActRight, 0)
	}
	s.DungeonAction(app.ActForward, 0)
	if g.MazeLevel != pl || g.PlayerX != px || g.PlayerY != py || !s.real {
		t.Fatalf("not on the pit: L%d (%d,%d) real %v", g.MazeLevel+1, g.PlayerX, g.PlayerY, s.real)
	}
	var qx, qy int
	found := false
	for _, d := range []engine.Direction{engine.North, engine.East, engine.South, engine.West} {
		g.Facing = d
		qx, qy = (px+d.DX()+20)%20, (py+d.DY()+20)%20
		if g.WallAhead() == data.WallOpen && (qx != 0 || qy != 0) {
			found = true
			break
		}
	}
	if !found {
		t.Skip("the pit square has no other way on")
	}
	cell := &g.Scenario.Mazes.Levels[pl].Cells[qy][qx]
	cell.Type, cell.Encounter = data.SqPit, false
	for _, m := range g.Town.Party.Members[:3] {
		m.HP, m.Agility = 1, 0
	}
	second, _ := engine.PlaceOf(len(g.Scenario.Mazes.Levels), pl, qx, qy)
	v = s.DungeonAction(app.ActForward, 0)
	t.Logf("B falls at %s: %s", second, v.Notice)
	if v.Real || !strings.Contains(v.Notice, "THEY REMAIN AT "+second.String()) {
		t.Fatalf("not defeated: real %v notice %q", v.Real, v.Notice)
	}
	got := strandedOnDiskAt(t, dir, second)
	if strings.Join(got, ",") != strings.Join(sortedLike(carried, onDiskNames(t, dir)), ",") {
		t.Fatalf("stranded at %s: %v, want %v", second, got, carried)
	}
	rest := strandedOnDiskAt(t, dir, pit)
	if len(rest) != 3 {
		t.Fatalf("still at %s: %v", pit, rest)
	}
	for _, ch := range fallen[:3] {
		if ch.Stranded == nil || *ch.Stranded != second {
			t.Fatalf("%s: %v", ch.Name, ch.Stranded)
		}
	}
	t.Logf("stranded at %s: %v; still at %s: %v", second, got, pit, rest)
	mustMatchDisk(t, s, dir, "after the rescuers fell")
}

// onDiskNames is the roster order in the save.
func onDiskNames(t *testing.T, dir string) []string {
	var out []string
	for _, ch := range onDisk(t, dir).Roster {
		out = append(out, ch.Name)
	}
	return out
}

// sortedLike returns names in the order of order.
func sortedLike(names, order []string) []string {
	in := map[string]bool{}
	for _, n := range names {
		in[n] = true
	}
	var out []string
	for _, n := range order {
		if in[n] {
			out = append(out, n)
		}
	}
	return out
}
