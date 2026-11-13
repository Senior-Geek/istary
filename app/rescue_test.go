package app

// Rescue (R-2): Inspect finds stranded characters in the party's area, and
// P)ICK UP takes them one at a time while there is room. Numbers in the
// comments are the R-2 brief's test scenarios (section 12).

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// rescueParty returns a controller in the maze with n sturdy RESCUERs at
// the entrance (L1:R19:C0), out of Camp, the stairs declined.
func rescueParty(t *testing.T, n int) *Controller {
	t.Helper()
	c := newController(t, 1)
	for i := 0; i < n; i++ {
		ch := member(fmt.Sprintf("RESCUER %d", i+1), engine.Neutral)
		ch.HP, ch.MaxHP, ch.Level, ch.Gold = 300, 300, 5, 100000
		ch.InMaze = true // as the Tavern sets it on joining
		c.Game.Town.Roster.Add(ch)
		c.Game.Town.Party.Members = append(c.Game.Town.Party.Members, ch)
	}
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(false))
	c.Game.MazeMessage, c.Game.MazeMessage2 = "", ""
	return c
}

// strandedAt adds n characters named prefix1, prefix2 ... to the roster,
// stranded at p, each different: DEAD, PARALYZED, STONED and ASHED in turn,
// with their own HP, gold, experience, age, item and spells.
func strandedAt(c *Controller, p engine.Place, prefix string, n int) []*engine.Character {
	sts := []engine.Status{engine.Dead, engine.Paralyzed, engine.Stoned, engine.Ashed}
	var out []*engine.Character
	for i := 0; i < n; i++ {
		ch := member(fmt.Sprintf("%s%d", prefix, i+1), engine.Neutral)
		ch.Status = sts[i%len(sts)]
		ch.HP = 0
		if ch.Status == engine.Paralyzed || ch.Status == engine.Stoned {
			ch.HP = 3 + i
		}
		ch.Gold, ch.XP, ch.Age, ch.Level = 100*(i+1), 1000*(i+1), 20*52+i, 2+i
		ch.Items[0], ch.ItemCount = engine.Possession{ItemIndex: 1 + i, Equipped: true, Identified: true}, 1
		ch.PriestSpells[0] = i
		ch.InMaze = true
		pp := p
		ch.Stranded = &pp
		if !c.Game.Town.Roster.Add(ch) {
			panic("roster full")
		}
		out = append(out, ch)
	}
	return out
}

// here is where the party stands.
func herePlace(t *testing.T, c *Controller) engine.Place {
	t.Helper()
	p, ok := c.Game.Place()
	if !ok {
		t.Fatal("the party is not on a square")
	}
	return p
}

// walledLevel walls in every square of level (0-based) in this game's copy
// of the scenario.
func walledLevel(c *Controller, level int) {
	for y, row := range c.Game.Scenario.Mazes.Levels[level].Cells {
		for x := range row {
			cell := &c.Game.Scenario.Mazes.Levels[level].Cells[y][x]
			cell.N, cell.S, cell.E, cell.W = data.WallWall, data.WallWall, data.WallWall, data.WallWall
		}
	}
}

// wall sets the two sides of the wall between (x, y) on level and its
// neighbour in direction d.
func wall(c *Controller, level, x, y int, d engine.Direction, w data.WallType) {
	cells := c.Game.Scenario.Mazes.Levels[level].Cells
	set := func(cell *data.MazeCell, d engine.Direction) {
		switch d {
		case engine.North:
			cell.N = w
		case engine.East:
			cell.E = w
		case engine.South:
			cell.S = w
		case engine.West:
			cell.W = w
		}
	}
	set(&cells[y][x], d)
	nx, ny := (x+d.DX()+20)%20, (y+d.DY()+20)%20
	set(&cells[ny][nx], (d+2)%4)
}

// onLevel puts the party at (x, y) on level (0-based), facing north.
func onLevel(c *Controller, level, x, y int) {
	c.Game.MazeLevel = level
	at(c, x, y, engine.North)
	c.Game.InitFightMap()
}

func charNames(chars []*engine.Character) []string {
	var out []string
	for _, ch := range chars {
		if ch != nil {
			out = append(out, ch.Name)
		}
	}
	return out
}

func sameNames(a []*engine.Character, want ...string) bool {
	got := charNames(a)
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// rescueState is the rescue's whole state: every roster record, the party and
// the found list.
func rescueState(t *testing.T, c *Controller) string {
	t.Helper()
	g := c.Game
	raw, err := json.Marshal(struct {
		Roster []*engine.Character
		Party  []string
		Found  []string
		Pick   bool
	}{g.Town.Roster.Characters, charNames(g.Town.Party.Members), charNames(g.MazeInspectFound), g.MazePicking})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// timesInParty counts ch in the party.
func timesInParty(g *engine.GameState, ch *engine.Character) int {
	n := 0
	for _, m := range g.Town.Party.Members {
		if m == ch {
			n++
		}
	}
	return n
}

// mustBeRescued fails unless every one of chars is in the party once, no
// longer stranded, still out with it, and otherwise as before.
func mustBeRescued(t *testing.T, c *Controller, chars []*engine.Character, before []string) {
	t.Helper()
	for i, ch := range chars {
		if timesInParty(c.Game, ch) != 1 || ch.Stranded != nil || !ch.InMaze || ch.MazeLevel != 0 || ch.MazeX != 0 || ch.MazeY != 0 {
			t.Fatalf("%s: in party %d times, stranded %v, in maze %v, legacy L%d (%d,%d)", ch.Name, timesInParty(c.Game, ch), ch.Stranded, ch.InMaze, ch.MazeLevel, ch.MazeX, ch.MazeY)
		}
		if got := record(t, ch); got != before[i] {
			t.Fatalf("%s changed by the pickup:\n%s\nwas\n%s", ch.Name, got, before[i])
		}
	}
}

// (1, 2, 3, 4, 8) Inspect finds the stranded on the party's square and on
// any square it could walk to through open walls; not past a door or a
// hidden door, and not on another level. The search reveals nothing on the
// player's map.
func TestInspectFindsTheStrandedInTheArea(t *testing.T) {
	c := rescueParty(t, 1)
	g := c.Game
	walledLevel(c, 1)
	wall(c, 1, 5, 5, engine.East, data.WallOpen)
	wall(c, 1, 6, 5, engine.East, data.WallDoor)
	wall(c, 1, 5, 5, engine.North, data.WallHidden)
	at2 := func(x, y int) engine.Place { return place(c, 1, x, y) }
	strandedAt(c, at2(5, 5), "SAME", 1)
	strandedAt(c, at2(6, 5), "NEXT", 1)
	strandedAt(c, at2(7, 5), "DOOR", 1)
	strandedAt(c, at2(5, 6), "HIDDEN", 1)
	strandedAt(c, place(c, 0, 5, 5), "UPSTAIRS", 1)
	strandedAt(c, place(c, 2, 5, 5), "DOWNSTAIRS", 1)
	onLevel(c, 1, 5, 5)
	explored, _ := json.Marshal(g.Exploration)
	must(t, c.Inspect())
	if !sameNames(g.MazeInspectFound, "SAME1", "NEXT1") {
		t.Fatalf("found %v", charNames(g.MazeInspectFound))
	}
	if after, _ := json.Marshal(g.Exploration); string(after) != string(explored) {
		t.Fatal("the search changed the player's map")
	}
	must(t, c.LeaveInspect())
	onLevel(c, 1, 7, 5) // beyond the door
	must(t, c.Inspect())
	if !sameNames(g.MazeInspectFound, "DOOR1") {
		t.Fatalf("beyond the door: %v", charNames(g.MazeInspectFound))
	}
}

// (9) A party stranded on SOLID ROCK is found, and picked up, from a square
// joined to the rock square by an open wall; the rescuers never step on it.
func TestSolidRockBodiesAreRecoveredFromNextDoor(t *testing.T) {
	seed(9)
	c, fallen := walkers(t)
	g := c.Game
	trap(c, 0, 4, 4, data.SqRockwater)
	stepOnto(t, c, 0, 4, 4)
	rock := place(c, 0, 4, 4)
	mustBeStranded(t, c, fallen, nil, rock)

	rescuer := explorers(t).Game.Town.Party.Members[0]
	rescuer.Name = "LONE RESCUER"
	g.Town.Roster.Add(rescuer)
	g.Town.Party.Members = []*engine.Character{rescuer}
	g.Town.Location = engine.EdgeOfTown
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(false))
	var nx, ny int
	for _, d := range []engine.Direction{engine.North, engine.East, engine.South, engine.West} {
		nx, ny = (4-d.DX()+20)%20, (4-d.DY()+20)%20
		at(c, nx, ny, d)
		if g.WallAhead() == data.WallOpen {
			break
		}
	}
	before := records(t, fallen)
	must(t, c.Inspect())
	if len(g.MazeInspectFound) != 6 {
		t.Fatalf("found %d from beside the rock", len(g.MazeInspectFound))
	}
	for i := 0; i < 5; i++ {
		must(t, c.PickUp(1))
	}
	mustBeRescued(t, c, fallen[:5], before[:5])
	if g.PlayerX != nx || g.PlayerY != ny || rescuer.Status != engine.OK || g.Phase != engine.PhaseMaze {
		t.Fatalf("the rescuer moved or died: (%d,%d) %v %v", g.PlayerX, g.PlayerY, rescuer.Status, g.Phase)
	}
	if fallen[5].Stranded == nil || *fallen[5].Stranded != rock {
		t.Fatalf("the sixth: %v", fallen[5].Stranded)
	}
}

// (11, 12, 13, 19, 20) One rescuer takes five, three take three, six take
// none: the party holds six, the incapacitated included. A refusal changes
// nothing, and everyone found is still listed, with the reason.
func TestRescueCapacity(t *testing.T) {
	for _, n := range []int{1, 3, 6} {
		c := rescueParty(t, n)
		g := c.Game
		fallen := strandedAt(c, herePlace(t, c), "FALLEN", 6)
		before := records(t, fallen)
		must(t, c.Inspect())
		taken := 0
		for {
			err := c.PickUp(1)
			if err != nil {
				var r *Refusal
				want := fmt.Sprintf("YOUR PARTY HAS 6; %s STAYS HERE.", g.MazeInspectFound[0].Name)
				if !errors.As(err, &r) || r.Message != want {
					t.Fatalf("%d rescuers: refusal %v, want %q", n, err, want)
				}
				break
			}
			taken++
		}
		if taken != 6-n || g.Town.Party.Size() != 6 {
			t.Fatalf("%d rescuers took %d, party %d", n, taken, g.Town.Party.Size())
		}
		mustBeRescued(t, c, fallen[:taken], before[:taken])
		snap := rescueState(t, c)
		if err := c.PickUp(1); err == nil || rescueState(t, c) != snap {
			t.Fatalf("%d rescuers: a full party picked up (%v)", n, err)
		}
		if err := g.RecoverStranded(fallen[taken]); !errors.Is(err, engine.ErrRescuePartyFull) || rescueState(t, c) != snap {
			t.Fatalf("engine: %v", err)
		}
		v := c.DungeonView()
		if v.Inspect == nil || v.Inspect.Room != 0 || len(v.Inspect.Found) != n || len(v.Found) != n {
			t.Fatalf("%d rescuers: view %+v", n, v.Inspect)
		}
		for _, f := range v.Inspect.Found {
			if f.CanPick || f.Refusal != "YOUR PARTY HAS 6; "+f.Name+" STAYS HERE." {
				t.Fatalf("full party: %+v", f)
			}
		}
		for _, a := range v.Actions {
			if a == ActPickUp {
				t.Fatal("pick-up offered to a full party")
			}
		}
		for _, ch := range fallen[taken:] {
			if ch.Stranded == nil || *ch.Stranded != herePlace(t, c) {
				t.Fatalf("%s left behind: %v", ch.Name, ch.Stranded)
			}
		}
	}
}

// (14) Empty entries in the party list (Camp D)ISBAND leaves them, #26) take
// no room; a pickup closes them up.
func TestEmptyPartyEntriesTakeNoRoom(t *testing.T) {
	c := rescueParty(t, 3)
	g := c.Game
	g.Town.Party.Members = append(g.Town.Party.Members, nil, nil)
	strandedAt(c, herePlace(t, c), "FALLEN", 4)
	must(t, c.Inspect())
	if c.DungeonView().Inspect.Room != 3 {
		t.Fatalf("room %d", c.DungeonView().Inspect.Room)
	}
	for i := 0; i < 3; i++ {
		must(t, c.PickUp(1))
	}
	if len(g.Town.Party.Members) != 6 || g.Town.Party.Size() != 6 {
		t.Fatalf("party list %d, size %d", len(g.Town.Party.Members), g.Town.Party.Size())
	}
	if err := c.PickUp(1); err == nil {
		t.Fatal("a seventh")
	}
}

// (15, 16, 17, 18) Several on one square are taken one at a time, by number;
// whoever is not picked stays stranded, unchanged. A number not on the
// list, or the same character twice, changes nothing.
func TestPickUpOneAtATime(t *testing.T) {
	c := rescueParty(t, 1)
	g := c.Game
	p := herePlace(t, c)
	fallen := strandedAt(c, p, "FALLEN", 4)
	before := records(t, fallen)
	must(t, c.Inspect())
	snap := rescueState(t, c)
	for _, n := range []int{0, -1, 5, 99} {
		if err := c.PickUp(n); !errors.Is(err, ErrInvalidChoice) || rescueState(t, c) != snap {
			t.Fatalf("pick %d: %v", n, err)
		}
	}
	must(t, c.PickUp(2))
	if g.MazeMessage != "FALLEN2 JOINS THE PARTY." || !sameNames(g.MazeInspectFound, "FALLEN1", "FALLEN3", "FALLEN4") || !g.MazeInspecting {
		t.Fatalf("after one: %q found %v", g.MazeMessage, charNames(g.MazeInspectFound))
	}
	if v := c.DungeonView(); v.Inspect.Room != 4 || len(v.Inspect.Found) != 3 || v.Inspect.Found[1].N != 2 || v.Inspect.Found[1].Name != "FALLEN3" {
		t.Fatalf("view %+v", v.Inspect)
	}
	snap = rescueState(t, c)
	if err := g.RecoverStranded(fallen[1]); !errors.Is(err, engine.ErrNotFoundHere) || rescueState(t, c) != snap {
		t.Fatalf("FALLEN2 again: %v", err)
	}
	mustBeRescued(t, c, fallen[1:2], before[1:2])
	must(t, c.LeaveInspect())
	for i, ch := range []*engine.Character{fallen[0], fallen[2], fallen[3]} {
		if ch.Stranded == nil || *ch.Stranded != p || !ch.InMaze || record(t, ch) != before[[]int{0, 2, 3}[i]] {
			t.Fatalf("%s: stranded %v", ch.Name, ch.Stranded)
		}
	}
	must(t, c.Inspect())
	if !sameNames(g.MazeInspectFound, "FALLEN1", "FALLEN3", "FALLEN4") {
		t.Fatalf("inspect again: %v", charNames(g.MazeInspectFound))
	}
}

// (21-28) Picked up, a character is no longer stranded, still out with the
// party, and exactly as before: condition, HP, items, gold, experience, age
// and spells. The incapacitated still cannot act; the party goes on.
func TestRescuedTravelAsTheyAre(t *testing.T) {
	seed(21)
	c := rescueParty(t, 1)
	g := c.Game
	fallen := strandedAt(c, herePlace(t, c), "FALLEN", 5)
	before := records(t, fallen)
	must(t, c.Inspect())
	for range fallen {
		must(t, c.PickUp(1))
	}
	mustBeRescued(t, c, fallen, before)
	for _, ch := range fallen {
		if ch.IsAlive() || ch.OutOfCastle() != true {
			t.Fatalf("%s: alive %v out %v", ch.Name, ch.IsAlive(), ch.OutOfCastle())
		}
	}
	if !engine.PartyCanContinue(g.Town.Party.Members) || g.Town.Party.ActiveCount() != 1 {
		t.Fatalf("active %d", g.Town.Party.ActiveCount())
	}
	must(t, c.LeaveInspect())
	g.FightMap = [20][20]bool{}
	from, _ := g.Place()
	for _, d := range []engine.Direction{engine.North, engine.East, engine.South, engine.West} {
		g.Facing = d
		if g.WallAhead() == data.WallOpen {
			break
		}
	}
	must(t, c.Forward())
	if to, _ := g.Place(); to == from || g.Town.Party.Size() != 6 || g.Phase == engine.PhaseTown {
		t.Fatalf("the party did not go on: %v -> %v, %v", from, to, g.Phase)
	}
	if v := c.DungeonView(); len(v.Party) != 6 || v.Party[1].Status != "DEAD" {
		t.Fatalf("party view %+v", v.Party)
	}
}

// (29) A rescue party that cannot go on is stranded where it falls, the
// rescued with it: one record each, on the new square; none at the old.
func TestRescuePartyDefeatStrandsEveryoneAnew(t *testing.T) {
	seed(29)
	c := rescueParty(t, 2)
	g := c.Game
	old := herePlace(t, c)
	fallen := strandedAt(c, old, "FALLEN", 4)
	must(t, c.Inspect())
	for range fallen {
		must(t, c.PickUp(1))
	}
	must(t, c.LeaveInspect())
	party := append([]*engine.Character(nil), g.Town.Party.Members...)
	for _, m := range party[:2] {
		m.HP, m.Agility = 1, 0
	}
	type kept struct {
		gold, xp int
		items    [8]engine.Possession
	}
	var before []kept
	for _, m := range party {
		before = append(before, kept{m.Gold, m.XP, m.Items})
	}
	trap(c, 0, 4, 4, data.SqPit)
	stepOnto(t, c, 0, 4, 4)
	pit := place(c, 0, 4, 4)
	// The fall hurts whoever is not dead, the carried PARALYZED too (as any
	// pit does); possessions, gold and experience stay.
	for i, m := range party {
		if (kept{m.Gold, m.XP, m.Items}) != before[i] {
			t.Fatalf("%s lost something", m.Name)
		}
	}
	mustBeStranded(t, c, party, nil, pit)
	if len(g.Town.Roster.Characters) != 6 {
		t.Fatalf("roster %d", len(g.Town.Roster.Characters))
	}
	seen := map[string]bool{}
	for _, ch := range g.Town.Roster.Characters {
		if seen[ch.Name] || *ch.Stranded == old {
			t.Fatalf("%s twice, or still at %v", ch.Name, old)
		}
		seen[ch.Name] = true
	}
}

// (30, 31, 33) The stairs, a square that sends the party home and Camp
// MALOR to the castle all bring the rescued back: no longer out, stranded
// nowhere, not healed or raised, still in the castle party (Q8).
func TestTheRescuedComeHome(t *testing.T) {
	ways := map[string]func(t *testing.T, c *Controller){
		"stairs": func(t *testing.T, c *Controller) {
			stepOnto(t, c, 0, 0, 0)
			if c.mazePrompt() != "stairs" {
				t.Fatalf("prompt %q", c.mazePrompt())
			}
			must(t, c.Answer(true))
		},
		"back to the castle square": func(t *testing.T, c *Controller) {
			g := c.Game
			g.MazeMessages, g.MazeMsgScroll, g.MazeMsgWait = []string{"A VOICE..."}, 0, false
			g.MazePendingBack2Shop = true
			must(t, c.ContinueMessage())
		},
		"Camp MALOR": func(t *testing.T, c *Controller) {
			must(t, c.Camp())
			onLevel(c, 0, 0, 0)
			handleMalorInput(c.Game, Key{Kind: KeyEnter})
		},
	}
	for name, home := range ways {
		t.Run(name, func(t *testing.T) {
			// A step can meet a wandering party (global math/rand); a fixed
			// seed keeps that from depending on the tests run before.
			rand.Seed(1)
			c := rescueParty(t, 1)
			g := c.Game
			onLevel(c, 0, 1, 0)
			fallen := strandedAt(c, herePlace(t, c), "FALLEN", 3)
			must(t, c.Inspect())
			for range fallen {
				must(t, c.PickUp(1))
			}
			must(t, c.LeaveInspect())
			before := records(t, fallen)
			home(t, c)
			if g.Phase != engine.PhaseTown || g.Town.Location != engine.Castle {
				t.Fatalf("not home: %v %v", g.Phase, g.Town.Location)
			}
			for i, ch := range fallen {
				if ch.InMaze || ch.Stranded != nil || ch.OutOfCastle() || !InParty(g.Town, ch) {
					t.Fatalf("%s: in maze %v stranded %v in party %v", ch.Name, ch.InMaze, ch.Stranded, InParty(g.Town, ch))
				}
				if name != "Camp MALOR" && record(t, ch) != before[i] { // MALOR to the castle also cures poison (#25)
					t.Fatalf("%s changed on the way home", ch.Name)
				}
				if ch.Status == engine.OK || ch.IsAlive() {
					t.Fatalf("%s healed: %v", ch.Name, ch.Status)
				}
			}
		})
	}
}

// (32, 34) Back at the castle the rescued are at the castle: the Temple
// will treat them. A party in which no one can act cannot set out (Q8);
// one led by someone who can may take them along.
func TestTheRescuedAtTheCastle(t *testing.T) {
	c := rescueParty(t, 1)
	g := c.Game
	leader := g.Town.Party.Members[0]
	onLevel(c, 0, 1, 0)
	fallen := strandedAt(c, herePlace(t, c), "FALLEN", 2)
	must(t, c.Inspect())
	must(t, c.PickUp(1))
	must(t, c.PickUp(1))
	must(t, c.LeaveInspect())
	stepOnto(t, c, 0, 0, 0)
	must(t, c.Answer(true))

	g.Town.Location = engine.Temple
	for _, ch := range fallen {
		idx := -1
		for i, r := range g.Town.Roster.Characters {
			if r == ch {
				idx = i
			}
		}
		q, err := c.TempleQuote(idx, 0)
		if err != nil || q.Refusal != "" || q.Donation == 0 {
			t.Fatalf("Temple for %s: %+v %v", ch.Name, q, err)
		}
	}

	g.Town.Location = engine.EdgeOfTown
	g.Town.Party.Members = []*engine.Character{fallen[0], fallen[1]}
	if err := c.EnterMaze(); !errors.Is(err, ErrNoOneCanGoOn) || g.Phase != engine.PhaseTown {
		t.Fatalf("set out with no one able: %v, %v", err, g.Phase)
	}
	if g.Town.Message != ErrNoOneCanGoOn.Message {
		t.Fatalf("message %q", g.Town.Message)
	}
	g.Town.Party.Members = []*engine.Character{leader, fallen[0], fallen[1]}
	must(t, c.EnterMaze())
}

// (43) LOST characters are never found, even with a square.
func TestLostAreNeverFound(t *testing.T) {
	c := rescueParty(t, 1)
	lost := strandedAt(c, herePlace(t, c), "LOST", 1)[0]
	lost.Status = engine.Lost
	must(t, c.Inspect())
	if len(c.Game.MazeInspectFound) != 0 {
		t.Fatalf("found %v", charNames(c.Game.MazeInspectFound))
	}
	if err := c.Game.RecoverStranded(lost); !errors.Is(err, engine.ErrNotFoundHere) || lost.Stranded == nil {
		t.Fatalf("recovered the LOST: %v", err)
	}
}

// (41) A character an older version left out, with no Stranded record, is
// found only on the exact square of its legacy fields, and only when they
// name a square of the maze; picked up, the fields are cleared. Characters
// out on another version's expedition are not lost here (42).
func TestLegacyOutCharactersAndOtherExpeditions(t *testing.T) {
	c := rescueParty(t, 1)
	g := c.Game
	walledLevel(c, 1)
	wall(c, 1, 3, 4, engine.North, data.WallOpen)
	old := member("OLDTIMER", engine.Neutral)
	old.InMaze, old.MazeLevel, old.MazeX, old.MazeY, old.Status = true, 1, 3, 4, engine.Dead
	bad := member("NOWHERE", engine.Neutral)
	bad.InMaze, bad.MazeLevel, bad.MazeX, bad.MazeY = true, 50, 3, 4
	away := member("AWAY", engine.Neutral)
	away.InMaze = true // L1 (0,0) by its legacy fields, but out on another version's expedition
	for _, ch := range []*engine.Character{old, bad, away} {
		g.Town.Roster.Add(ch)
	}
	g.OtherExpedition = json.RawMessage(`{"version":99,"party":["AWAY"]}`)

	onLevel(c, 1, 3, 5) // in the area, but not the square
	must(t, c.Inspect())
	if len(g.MazeInspectFound) != 0 {
		t.Fatalf("next door: %v", charNames(g.MazeInspectFound))
	}
	must(t, c.LeaveInspect())
	onLevel(c, 0, 0, 0)
	must(t, c.Inspect())
	if len(g.MazeInspectFound) != 0 {
		t.Fatalf("at the entrance: %v", charNames(g.MazeInspectFound))
	}
	must(t, c.LeaveInspect())
	for l := range g.Scenario.Mazes.Levels {
		onLevel(c, l, 3, 4)
		for _, ch := range g.FindStranded() {
			if ch == bad {
				t.Fatalf("NOWHERE found on L%d", l+1)
			}
		}
	}
	onLevel(c, 1, 3, 4)
	must(t, c.Inspect())
	if !sameNames(g.MazeInspectFound, "OLDTIMER") {
		t.Fatalf("on the square: %v", charNames(g.MazeInspectFound))
	}
	must(t, c.PickUp(1))
	if old.MazeLevel != 0 || old.MazeX != 0 || old.MazeY != 0 || !old.InMaze || old.Stranded != nil || timesInParty(g, old) != 1 {
		t.Fatalf("OLDTIMER: L%d (%d,%d) in maze %v", old.MazeLevel, old.MazeX, old.MazeY, old.InMaze)
	}
}

// (Q6) The terminal: P)ICK UP asks GET WHO (0=EXIT) and takes one character
// a key; a number not on the list does nothing; 0 or [ESC] stops; a full
// party is told YOU HAVE 6. With more than nine found, the number is typed
// and [RET] takes it. The rules are PickUp's.
func TestTerminalGetWho(t *testing.T) {
	c := rescueParty(t, 4)
	g := c.Game
	fallen := strandedAt(c, herePlace(t, c), "FALLEN", 3)
	keys(t, c, "I")
	if !g.MazeInspecting || len(g.MazeInspectFound) != 3 {
		t.Fatalf("inspect: %v %d", g.MazeInspecting, len(g.MazeInspectFound))
	}
	keys(t, c, "P")
	if !g.MazePicking {
		t.Fatal("no GET WHO prompt")
	}
	snap := rescueState(t, c)
	keys(t, c, "7X")
	if rescueState(t, c) != snap {
		t.Fatal("an invalid choice changed something")
	}
	keys(t, c, "2")
	if !sameNames(g.MazeInspectFound, "FALLEN1", "FALLEN3") || g.MazeMessage != "FALLEN2 JOINS THE PARTY." || !g.MazePicking {
		t.Fatalf("after 2: %v %q picking %v", charNames(g.MazeInspectFound), g.MazeMessage, g.MazePicking)
	}
	keys(t, c, "1") // the party is full now: the prompt closes
	if g.MazePicking || g.Town.Party.Size() != 6 || !sameNames(g.MazeInspectFound, "FALLEN3") {
		t.Fatalf("full: picking %v party %d found %v", g.MazePicking, g.Town.Party.Size(), charNames(g.MazeInspectFound))
	}
	snap = rescueState(t, c)
	keys(t, c, "P")
	if g.MazeMessage != PartyFullMessage || g.MazePicking || rescueState(t, c) != snap {
		t.Fatalf("full party P: %q picking %v", g.MazeMessage, g.MazePicking)
	}
	keys(t, c, "L")
	if g.MazeInspecting || fallen[2].Stranded == nil {
		t.Fatal("leave")
	}

	c = rescueParty(t, 1)
	g = c.Game
	strandedAt(c, herePlace(t, c), "MANY", 12)
	keys(t, c, "IP")
	keys(t, c, "1")
	if len(g.MazeInspectFound) != 12 || g.MazePickBuf != "1" {
		t.Fatalf("typed 1: found %d buffer %q", len(g.MazeInspectFound), g.MazePickBuf)
	}
	keys(t, c, "2")
	press(t, c, KeyEnter)
	if len(g.MazeInspectFound) != 11 || g.Town.Party.Members[1].Name != "MANY12" || g.MazePickBuf != "" {
		t.Fatalf("12: found %d party %v", len(g.MazeInspectFound), charNames(g.Town.Party.Members))
	}
	keys(t, c, "13")
	press(t, c, KeyBackspace)
	press(t, c, KeyBackspace)
	keys(t, c, "99")
	press(t, c, KeyEnter) // 99 is not on the list: nothing is taken
	if len(g.MazeInspectFound) != 11 || g.Town.Party.Size() != 2 || !g.MazePicking {
		t.Fatalf("99: found %d party %d", len(g.MazeInspectFound), g.Town.Party.Size())
	}
	keys(t, c, "0")
	if g.MazePicking {
		t.Fatal("0 did not stop")
	}
	keys(t, c, "P")
	press(t, c, KeyEscape)
	if g.MazePicking || !g.MazeInspecting {
		t.Fatal("[ESC] did not stop")
	}
}

// (35, 36, 39, 40) A partial rescue, saved while the Inspect screen is
// open, resumes after a restart with the rescued in the party, carried,
// and the rest still stranded and listed; twice from the same file, the
// same. Saved finds that no longer stand are dropped. An expedition that
// lists a member still marked stranded is refused, changing nothing.
func TestPartialRescueSurvivesARestart(t *testing.T) {
	orig := pristine(t)
	engine.SetSaveDir(t.TempDir())
	t.Cleanup(func() { engine.SetSaveDir("") })
	c := rescueParty(t, 2)
	g := c.Game
	p := herePlace(t, c)
	strandedAt(c, p, "FALLEN", 4)
	must(t, c.Inspect())
	must(t, c.PickUp(1))
	must(t, c.PickUp(2)) // FALLEN3
	exp, err := c.Checkpoint(orig)
	must(t, err)
	g.Expedition = exp
	must(t, g.Save())
	load := func() (*Controller, error) {
		g := engine.New(pristine(t))
		g.Title, g.Phase = nil, engine.PhaseTown
		must(t, g.Load())
		c := New(g)
		return c, c.Resume(orig)
	}
	for restart := 0; restart < 2; restart++ {
		c2, err := load()
		must(t, err)
		g2 := c2.Game
		if !sameNames(g2.Town.Party.Members, "RESCUER 1", "RESCUER 2", "FALLEN1", "FALLEN3") || !g2.MazeInspecting ||
			!sameNames(g2.MazeInspectFound, "FALLEN2", "FALLEN4") {
			t.Fatalf("resumed: party %v found %v", charNames(g2.Town.Party.Members), charNames(g2.MazeInspectFound))
		}
		for _, ch := range g2.Town.Roster.Characters {
			stranded := ch.Name == "FALLEN2" || ch.Name == "FALLEN4"
			if (ch.Stranded != nil) != stranded || (stranded && *ch.Stranded != p) || !ch.InMaze {
				t.Fatalf("%s: stranded %v in maze %v", ch.Name, ch.Stranded, ch.InMaze)
			}
		}
		if restart == 1 {
			must(t, c2.PickUp(2))
			if !sameNames(g2.MazeInspectFound, "FALLEN2") {
				t.Fatalf("after a resume: %v", charNames(g2.MazeInspectFound))
			}
		}
	}

	// Stale finds: a name no longer on the roster, one picked up since, and
	// one twice.
	g.Expedition.Where.InspectFound = []string{"FALLEN1", "NOBODY", "FALLEN4", "FALLEN4"}
	must(t, g.Save())
	c3, err := load()
	must(t, err)
	if !sameNames(c3.Game.MazeInspectFound, "FALLEN4") {
		t.Fatalf("stale finds kept: %v", charNames(c3.Game.MazeInspectFound))
	}

	// Contradictory: a member of the expedition still marked stranded.
	g.Expedition.Where.InspectFound = nil
	for _, ch := range g.Town.Roster.Characters {
		if ch.Name == "FALLEN1" {
			ch.Stranded = &p
		}
	}
	must(t, g.Save())
	g4 := engine.New(pristine(t))
	g4.Title, g4.Phase = nil, engine.PhaseTown
	must(t, g4.Load())
	c4 := New(g4)
	raw, _ := json.Marshal(g4.Town.Roster.Characters)
	var problem *ExpeditionProblem
	if err := c4.Resume(orig); !errors.As(err, &problem) {
		t.Fatalf("resume with a stranded member: %v", err)
	}
	if after, _ := json.Marshal(g4.Town.Roster.Characters); string(after) != string(raw) || g4.Phase != engine.PhaseTown || g4.Town.Party.Size() != 0 || g4.Expedition == nil {
		t.Fatal("the refused resume changed something")
	}
}
