package app

// Stranded parties (R-1): a party that can no longer go on stays on its
// square, its members' conditions as they were; the player is back in the
// castle with no party. Numbers in the comments are the R-1 brief's test
// scenarios (section 12).

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// record is everything about a character that being stranded must leave
// alone: the whole saved record but where they are.
func record(t *testing.T, ch *engine.Character) string {
	t.Helper()
	cp := *ch
	cp.Stranded, cp.InMaze = nil, false
	raw, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func records(t *testing.T, party []*engine.Character) []string {
	var out []string
	for _, m := range party {
		out = append(out, record(t, m))
	}
	return out
}

// mustBeStranded fails unless the game is in the castle with no party and
// every one of party is stranded at want, unchanged otherwise (before).
func mustBeStranded(t *testing.T, c *Controller, party []*engine.Character, before []string, want engine.Place) {
	t.Helper()
	g := c.Game
	if g.Phase != engine.PhaseTown || g.Town.Location != engine.Castle || g.Combat != nil || g.Town.Party.Size() != 0 {
		t.Fatalf("phase %v location %v combat %v party %d, want the castle with no party", g.Phase, g.Town.Location, g.Combat != nil, g.Town.Party.Size())
	}
	if g.Town.Message != engine.StrandMessage || g.Town.Message2 != "THEY REMAIN AT "+want.String()+"." {
		t.Fatalf("castle message %q / %q", g.Town.Message, g.Town.Message2)
	}
	for i, m := range party {
		if m.Status == engine.Lost {
			if m.Stranded != nil || m.InMaze {
				t.Fatalf("LOST %s: stranded %v in maze %v", m.Name, m.Stranded, m.InMaze)
			}
		} else if m.Stranded == nil || *m.Stranded != want || !m.InMaze {
			t.Fatalf("%s: stranded %v in maze %v, want %v", m.Name, m.Stranded, m.InMaze, want)
		}
		if got := record(t, m); before != nil && got != before[i] {
			t.Fatalf("%s changed:\n%s\nwas\n%s", m.Name, got, before[i])
		}
		if !slicesContain(g.Town.Roster.Characters, m) && len(g.Town.Roster.Characters) > 0 {
			t.Fatalf("%s left the roster", m.Name)
		}
	}
}

func slicesContain(list []*engine.Character, ch *engine.Character) bool {
	for _, c := range list {
		if c == ch {
			return true
		}
	}
	return false
}

func noneStranded(t *testing.T, chars []*engine.Character) {
	t.Helper()
	for _, m := range chars {
		if m != nil && m.Stranded != nil {
			t.Fatalf("%s stranded at %v", m.Name, m.Stranded)
		}
	}
}

// The one continuation rule: someone who can act. ASLEEP and AFRAID can
// (they recover), everything from PARALYZED on cannot. (1, 4, 5, 6)
func TestPartyCanContinueRule(t *testing.T) {
	with := func(sts ...engine.Status) []*engine.Character {
		var p []*engine.Character
		for i, st := range sts {
			ch := engine.NewCharacter(fmt.Sprint(i), engine.Human, engine.Fighter, engine.Neutral)
			ch.Status = st
			p = append(p, ch)
		}
		return p
	}
	for _, tc := range []struct {
		sts  []engine.Status
		want bool
	}{
		{[]engine.Status{engine.OK, engine.Dead, engine.Dead}, true},
		{[]engine.Status{engine.Asleep, engine.Dead}, true},
		{[]engine.Status{engine.Afraid, engine.Stoned}, true},
		{[]engine.Status{engine.Dead, engine.Dead}, false},
		{[]engine.Status{engine.Dead, engine.Ashed}, false},
		{[]engine.Status{engine.Paralyzed, engine.Stoned}, false},
		{[]engine.Status{engine.Paralyzed, engine.Dead, engine.Ashed, engine.Lost}, false},
		{nil, false},
	} {
		if got := engine.PartyCanContinue(with(tc.sts...)); got != tc.want {
			t.Errorf("%v: %v, want %v", tc.sts, got, tc.want)
		}
	}
	if engine.PartyCanContinue([]*engine.Character{nil, nil}) {
		t.Error("an empty party can go on")
	}
}

// One member falling does not end the fight; ASLEEP and AFRAID members
// keep it going. (1, 6)
func TestOneFallenMemberDoesNotEndCombat(t *testing.T) {
	for _, keep := range []engine.Status{engine.OK, engine.Asleep, engine.Afraid} {
		seed(3)
		c, p := confirmed(t, 1)
		first, err := c.Commit()
		must(t, err)
		playAll(t, c, first)
		for i, m := range p {
			m.Status = engine.Dead
			if i == 2 {
				m.Status = keep
			}
		}
		if out, err := c.EndRound(); err != nil || out != RoundContinues || c.Game.Phase != engine.PhaseCombat {
			t.Fatalf("%v left: %v %v phase %v", keep, out, err, c.Game.Phase)
		}
		noneStranded(t, p)
	}
}

// A party that cannot go on in combat stays on its square, conditions as
// they were, all on the same square; the player is in the castle with no
// party. (2, 3, 4, 5, 16, 17)
func TestCombatDefeatStrandsTheParty(t *testing.T) {
	for _, sts := range [][]engine.Status{
		{engine.Dead, engine.Dead, engine.Dead, engine.Dead, engine.Dead, engine.Dead},
		{engine.Dead, engine.Ashed, engine.Dead, engine.Ashed, engine.Ashed, engine.Dead},
		{engine.Paralyzed, engine.Stoned, engine.Paralyzed, engine.Stoned, engine.Paralyzed, engine.Stoned},
		{engine.Stoned, engine.Dead, engine.Paralyzed, engine.Ashed, engine.Lost, engine.Dead},
	} {
		seed(3)
		c, p := confirmed(t, 1)
		g := c.Game
		g.MazeLevel, g.PlayerX, g.PlayerY = 2, 7, 12
		for _, m := range p {
			g.Town.Roster.Add(m)
			m.InMaze = true
		}
		first, err := c.Commit()
		must(t, err)
		playAll(t, c, first)
		for i, m := range p {
			m.Status = sts[i]
			m.Gold, m.XP = 100*i, 1000*i
		}
		out, err := c.EndRound()
		if err != nil || out != RoundDefeated {
			t.Fatalf("%v: %v %v", sts, out, err)
		}
		before := records(t, p)
		explored, _ := json.Marshal(g.Exploration)
		must(t, c.LeaveCombat())
		want := engine.Place{Level: 3, Row: 7, Col: 7}
		mustBeStranded(t, c, p, before, want)
		for i, m := range p {
			if m.Status != sts[i] {
				t.Fatalf("%s is %v, was %v", m.Name, m.Status, sts[i])
			}
		}
		if after, _ := json.Marshal(g.Exploration); string(after) != string(explored) {
			t.Fatal("the exploration record changed") // (35)
		}
		// Again: nothing more happens. (30)
		if err := c.LeaveCombat(); !errors.Is(err, ErrNotEnding) {
			t.Fatalf("leaving again: %v", err)
		}
		if _, done, err := g.StrandParty(); done || err != nil {
			t.Fatalf("stranding again: %v %v", done, err)
		}
		mustBeStranded(t, c, p, before, want)
	}
}

// A successful R)UN leaves no one stranded. (7)
func TestEscapeStrandsNoOne(t *testing.T) {
	for s := int64(1); s < 60; s++ {
		seed(s)
		c, p := confirmed(t, 1, engine.PartyAction{Action: engine.ActionRun})
		first, err := c.Commit()
		must(t, err)
		playAll(t, c, first)
		if out, _ := c.EndRound(); out == RoundEscaped {
			if c.Game.Phase != engine.PhaseMaze || c.Game.Town.Party.Size() != 6 {
				t.Fatalf("after fleeing: phase %v party %d", c.Game.Phase, c.Game.Town.Party.Size())
			}
			noneStranded(t, p)
			return
		}
	}
	t.Fatal("no seed escaped")
}

// LOKTOFEIT and MALOR cast in combat set the combat to "defeat" for a
// moment (finding #24); the round's end sets it back and the fight goes on.
// That is no defeat: nobody is stranded. (8)
func TestEscapeSpellFlagsAreNoDefeat(t *testing.T) {
	for _, spell := range []string{"LOKTOFEIT", "MALOR"} {
		seed(5)
		c, p := confirmed(t, 1, engine.PartyAction{Action: engine.ActionSpell, SpellName: spell})
		caster := p[0]
		caster.Level = 60 // LOKTOFEIT works when RANDOM MOD 100 <= 2 x level
		if spell == "LOKTOFEIT" {
			caster.Class, caster.PriestSpells[5], caster.MaxPriestSpells[5] = engine.Priest, 1, 1
		} else {
			caster.Class, caster.MageSpells[6], caster.MaxMageSpells[6] = engine.Mage, 1, 1
		}
		caster.SpellKnown[engine.SpellIndex[spell]] = true
		first, err := c.Commit()
		must(t, err)
		steps := playAll(t, c, first)
		cast := false
		for _, st := range steps {
			cast = cast || strings.Contains(strings.Join(st.Event.Lines, " "), spell+"!")
		}
		if !cast {
			t.Fatalf("%s was not cast", spell)
		}
		out, err := c.EndRound()
		if err != nil || out == RoundDefeated || c.Game.Phase != engine.PhaseCombat {
			t.Fatalf("%s: %v %v phase %v", spell, out, err, c.Game.Phase)
		}
		noneStranded(t, p)
	}
}

// walkers returns a controller with the six WALKERs in the maze at the
// entrance, out of Camp, weak enough for any damage to kill them.
func walkers(t *testing.T) (*Controller, []*engine.Character) {
	t.Helper()
	c := inMaze(t)
	party := append([]*engine.Character(nil), c.Game.Town.Party.Members...)
	for _, m := range party {
		m.HP, m.Agility, m.InMaze = 1, 0, true // no save against a fall
	}
	return c, party
}

// trap turns a square into a pit in this game's copy of the scenario (the
// data has no OUCH or SOLID ROCK square on the first floors).
func trap(c *Controller, level, x, y int, typ data.SquareType) {
	cell := &c.Game.Scenario.Mazes.Levels[level].Cells[y][x]
	cell.Type, cell.BaseDamage, cell.NumDice, cell.DieSize = typ, 5, 1, 4
}

// stepOnto puts the party on a neighbour of (x, y) on level from which a
// step forward reaches it, and steps. It fails if there is none.
func stepOnto(t *testing.T, c *Controller, level, x, y int) {
	t.Helper()
	g := c.Game
	g.MazeLevel = level
	g.InitFightMap()
	for _, dir := range []engine.Direction{engine.North, engine.East, engine.South, engine.West} {
		nx, ny := (x-dir.DX()+20)%20, (y-dir.DY()+20)%20
		at(c, nx, ny, dir)
		if g.WallAhead() == data.WallOpen {
			must(t, c.Forward())
			return
		}
	}
	t.Fatalf("no way onto L%d (%d,%d)", level, x, y)
}

// place is (x, y) on level in L:R:C.
func place(c *Controller, level, x, y int) engine.Place {
	p, _ := engine.PlaceOf(len(c.Game.Scenario.Mazes.Levels), level, x, y)
	return p
}

// A step onto a pit, an OUCH square or SOLID ROCK that leaves no one able
// to go on strands the party on that square; SOLID ROCK no longer carries
// it to the castle. (9, 10, 12, 16)
func TestHazardDefeatStrandsTheParty(t *testing.T) {
	pit := func(c *Controller) (int, int, int) {
		for l, lv := range c.Game.Scenario.Mazes.Levels {
			for y, row := range lv.Cells {
				for x, cell := range row {
					if cell.Type == data.SqPit {
						return l, x, y // the first pit of the data
					}
				}
			}
		}
		t.Fatal("no pit in the data")
		return 0, 0, 0
	}
	for _, kind := range []string{"pit", "ouch", "rock"} {
		seed(7)
		c, party := walkers(t)
		level, x, y := 0, 5, 9
		switch kind {
		case "pit":
			level, x, y = pit(c)
		case "ouch":
			trap(c, level, x, y, data.SqOuchy)
		case "rock":
			trap(c, level, x, y, data.SqRockwater)
		}
		before := records(t, party)
		stepOnto(t, c, level, x, y)
		if c.Game.Phase == engine.PhaseTown {
			for _, m := range party {
				if m.Status != engine.Dead {
					t.Fatalf("%s: %s is %v", kind, m.Name, m.Status)
				}
			}
		}
		for i, m := range party {
			// The fall's damage is the hazard's: HP and status change, nothing else.
			m2 := *m
			m2.HP, m2.Status = 1, engine.OK
			if record(t, &m2) != before[i] {
				t.Fatalf("%s: %s changed beyond the damage", kind, m.Name)
			}
		}
		mustBeStranded(t, c, party, nil, place(c, level, x, y))
	}
}

// Poison that kills the last one standing strands the party where it
// walks. (11)
func TestPoisonDefeatStrandsTheParty(t *testing.T) {
	seed(11)
	c, party := walkers(t)
	for _, m := range party {
		m.PoisonAmt = 5
	}
	g := c.Game
	for step := 0; step < 400 && g.Phase != engine.PhaseTown; step++ {
		if g.Phase == engine.PhaseCombat { // an encounter: not this test's
			g.Combat, g.Phase = nil, engine.PhaseMaze
		}
		for g.WallAhead() != data.WallOpen {
			must(t, c.TurnRight())
		}
		x, y := (g.PlayerX+g.Facing.DX()+20)%20, (g.PlayerY+g.Facing.DY()+20)%20
		if cell := g.Scenario.Mazes.Levels[0].Cells[y][x]; cell.Type != data.SqNormal {
			must(t, c.TurnRight())
			continue
		}
		g.FightMap = [20][20]bool{}
		must(t, c.Forward())
		for c.mazePrompt() != "" {
			c.HandleKey(Key{Kind: KeyEnter})
		}
		if g.Phase == engine.PhaseTown {
			mustBeStranded(t, c, party, nil, place(c, 0, x, y))
			return
		}
	}
	t.Fatal("the poison never finished the party")
}

// A chute that drops the party and kills it strands it where it landed,
// not where it fell in. (13)
func TestChuteDefeatStrandsWhereThePartyLands(t *testing.T) {
	seed(13)
	c, party := walkers(t)
	var cl, cx, cy int
	found := false
	for l, lv := range c.Game.Scenario.Mazes.Levels {
		for y, row := range lv.Cells {
			for x, cell := range row {
				if cell.Type == data.SqChute && cell.DestLevel > 0 && !found {
					cl, cx, cy, found = l, x, y, true
				}
			}
		}
	}
	if !found {
		t.Fatal("no chute in the data")
	}
	cell := c.Game.Scenario.Mazes.Levels[cl].Cells[cy][cx]
	stepOnto(t, c, cl, cx, cy)
	want := place(c, cell.DestLevel-1, cell.DestX, cell.DestY)
	if want == place(c, cl, cx, cy) {
		t.Fatal("the chute does not move the party")
	}
	mustBeStranded(t, c, party, nil, want)
}

// The stairs home strand no one. (14)
func TestStairsHomeStrandsNoOne(t *testing.T) {
	c := explorers(t)
	party := append([]*engine.Character(nil), c.Game.Town.Party.Members...)
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(true))
	if c.Game.Phase != engine.PhaseTown || c.Game.Town.Party.Size() != 6 {
		t.Fatalf("after the stairs: phase %v party %d", c.Game.Phase, c.Game.Town.Party.Size())
	}
	noneStranded(t, party)
	for _, m := range party {
		if m.InMaze {
			t.Fatalf("%s still in the maze", m.Name)
		}
	}
}

// Camp MALOR into rock makes the party LOST and puts it in the castle, as
// before: LOST is not stranded. A LOST member of a defeated party is not
// left on the square either. (15)
func TestLostHazardsKeepTheirMeaning(t *testing.T) {
	c := explorers(t)
	party := append([]*engine.Character(nil), c.Game.Town.Party.Members...)
	must(t, c.EnterMaze())
	c.Game.Town.MalorDeltaUD = 30
	handleMalorInput(c.Game, Key{Kind: KeyEnter})
	if c.Game.Phase != engine.PhaseTown || c.Game.Town.Message2 != "YOU ARE LOST FOREVER!" {
		t.Fatalf("MALOR into rock: phase %v %q", c.Game.Phase, c.Game.Town.Message2)
	}
	for _, m := range party {
		if m.Status != engine.Lost || m.InMaze {
			t.Fatalf("%s: %v in maze %v", m.Name, m.Status, m.InMaze)
		}
	}
	noneStranded(t, party)
}

// A defeat just after crossing the level's edge, or a one-way wall,
// strands the party on the square it reached. (19, 20)
func TestDefeatAfterWrapAndOneWayRecordsTheSquareReached(t *testing.T) {
	seed(19)
	c, party := walkers(t)
	g := c.Game
	done := false
	// An open east edge, from column 19 to column 0 of the same row.
	for l, lv := range g.Scenario.Mazes.Levels {
		for y := 0; y < 20 && !done; y++ {
			if lv.Cells[y][19].E == data.WallOpen && lv.Cells[y][19].Type == data.SqNormal && lv.Cells[y][0].Type == data.SqNormal {
				trap(c, l, 0, y, data.SqPit)
				g.MazeLevel = l
				at(c, 19, y, engine.East)
				must(t, c.Forward())
				mustBeStranded(t, c, party, nil, place(c, l, 0, y))
				if got := party[0].Stranded.String(); got != fmt.Sprintf("L%d:R%d:C0", l+1, 19-y) {
					t.Fatalf("stranded at %s", got)
				}
				done = true
			}
		}
	}
	if !done {
		t.Fatal("no open east edge in the data")
	}

	seed(20)
	c, party = walkers(t)
	g = c.Game
	for l, lv := range g.Scenario.Mazes.Levels {
		for y := 0; y < 20; y++ {
			for x := 0; x < 20; x++ {
				for _, dir := range []engine.Direction{engine.North, engine.East, engine.South, engine.West} {
					nx, ny := (x+dir.DX()+20)%20, (y+dir.DY()+20)%20
					from, to := lv.Cells[y][x], lv.Cells[ny][nx]
					back := map[engine.Direction]data.WallType{engine.North: to.S, engine.South: to.N, engine.East: to.W, engine.West: to.E}[dir]
					ahead := map[engine.Direction]data.WallType{engine.North: from.N, engine.South: from.S, engine.East: from.E, engine.West: from.W}[dir]
					if ahead != data.WallOpen || back == data.WallOpen || to.Type != data.SqNormal {
						continue
					}
					trap(c, l, nx, ny, data.SqPit)
					g.MazeLevel = l
					at(c, x, y, dir)
					must(t, c.Forward())
					mustBeStranded(t, c, party, nil, place(c, l, nx, ny))
					return
				}
			}
		}
	}
	t.Fatal("no one-way wall in the data")
}

// toTavern puts the game in the Tavern with no party.
func toTavern(t *testing.T, c *Controller) {
	t.Helper()
	c.Game.Town.Location = engine.Castle
	must(t, c.EnterTavern())
}

// A stranded party stays in the roster and out of the castle: it cannot
// join a party, be treated at the Temple, be renamed, deleted or equipped
// in town, or pass the Rite; others can set out. A square on the first
// floor (internal level 0) is not the castle. Two defeats leave two
// parties on their own squares. (17, 18, 21, 22, 23, 24, 25, 26)
func TestStrandedCharactersAreNotInTheCastle(t *testing.T) {
	seed(21)
	c, first := walkers(t)
	g := c.Game
	for i := 0; i < 6; i++ {
		ch := member(fmt.Sprintf("SPARE %d", i+1), engine.Neutral)
		ch.HP, ch.MaxHP, ch.Agility = 1, 10, 0
		g.Town.Roster.Add(ch)
	}
	trap(c, 0, 4, 4, data.SqPit)
	stepOnto(t, c, 0, 4, 4)
	firstAt := place(c, 0, 4, 4)
	mustBeStranded(t, c, first, nil, firstAt)
	if first[0].MazeLevel != 0 || firstAt.Level != 1 {
		t.Fatalf("legacy level %d, place %v", first[0].MazeLevel, firstAt)
	}

	toTavern(t, c)
	for _, m := range first {
		if got := c.JoinRefusal(m); got != "** OUT **" {
			t.Fatalf("%s may join: %q", m.Name, got)
		}
		if _, err := c.AddMember(m.Name); err == nil || c.Game.Town.Party.Size() != 0 {
			t.Fatalf("%s joined: %v", m.Name, err)
		}
		if got := engine.RiteCanPerform(m); got == "" {
			t.Fatalf("%s may pass the Rite", m.Name)
		}
	}
	for i := 1; i <= 6; i++ {
		if pw, err := c.AddMember(fmt.Sprintf("SPARE %d", i)); err != nil || !pw {
			t.Fatalf("SPARE %d: %v", i, err)
		}
		must(t, c.SubmitTavernPassword(""))
	}
	second := append([]*engine.Character(nil), c.Game.Town.Party.Members...)
	if len(second) != 6 {
		t.Fatalf("the new party has %d", len(second))
	}

	// The Temple, as the GUI and the terminal take it.
	must(t, c.LeaveTavern())
	must(t, c.GoTo(engine.Temple))
	for i, ch := range g.Town.Roster.Characters {
		if !ch.IsStranded() {
			continue
		}
		q, err := c.TempleQuote(i, 0)
		if err != nil || !strings.Contains(q.Refusal, "not at the castle") {
			t.Fatalf("%s may be treated: %v %q", ch.Name, err, q.Refusal)
		}
		status, hp := ch.Status, ch.HP
		if _, err := c.Treat(i, 0, true); err == nil || ch.Status != status || ch.HP != hp {
			t.Fatalf("%s treated: %v", ch.Name, err)
		}
	}
	must(t, c.LeaveTemple())
	gold := first[0].Gold
	status := first[0].Status
	c.HandleKey(Rune('C')) // the terminal: the Temple, then the name
	for _, r := range first[0].Name {
		c.HandleKey(Rune(r))
	}
	c.HandleKey(Key{Kind: KeyEnter})
	if first[0].Status != status || first[0].Gold != gold || !strings.Contains(g.Town.Message, "IS OUT") {
		t.Fatalf("terminal Temple: %v %q", first[0].Status, g.Town.Message)
	}

	// Training Grounds and town equipping.
	g.Town.Location, g.Town.InputMode, g.Town.TempleStep = engine.Castle, engine.InputNone, 0
	for i, ch := range g.Town.Roster.Characters {
		if !ch.IsStranded() {
			continue
		}
		if err := c.RenameCharacter(i, "RENAMED"); !errors.Is(err, ErrStranded) {
			t.Fatalf("rename %s: %v", ch.Name, err)
		}
		if _, err := c.Equipment(i); !errors.Is(err, ErrStranded) {
			t.Fatalf("equip %s: %v", ch.Name, err)
		}
		g.Town.EditChar, g.Town.InputMode = ch, engine.InputConfirmDelete
		if err := c.ConfirmDeleteCharacter(true); !errors.Is(err, ErrStranded) || !slicesContain(g.Town.Roster.Characters, ch) {
			t.Fatalf("delete %s: %v", ch.Name, err)
		}
	}
	g.Town.EditChar, g.Town.InputMode = nil, engine.InputNone

	// The second party sets out and is defeated elsewhere.
	g.Town.Location = engine.EdgeOfTown
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(false))
	trap(c, 0, 9, 9, data.SqPit)
	stepOnto(t, c, 0, 9, 9)
	mustBeStranded(t, c, second, nil, place(c, 0, 9, 9))
	for _, m := range first {
		if *m.Stranded != firstAt {
			t.Fatalf("%s moved to %v", m.Name, m.Stranded)
		}
	}
	if n := len(g.Town.Roster.Characters); n != 12 {
		t.Fatalf("roster of %d", n)
	}
}

// Inspect finds stranded characters on their own square only, by their
// record (not the legacy fields, which say L1 (0,0) for everyone); P)ICK
// UP refuses them until rescue is designed (R-2), changing nothing.
func TestInspectFindsStrandedCharactersAndPickUpWaits(t *testing.T) {
	seed(22)
	c, fallen := walkers(t)
	g := c.Game
	trap(c, 0, 4, 4, data.SqPit)
	stepOnto(t, c, 0, 4, 4)
	here := place(c, 0, 4, 4)
	mustBeStranded(t, c, fallen, nil, here)

	rescuers := explorers(t).Game.Town.Party.Members
	for _, m := range rescuers {
		m.Name = "R" + m.Name
		g.Town.Roster.Add(m)
	}
	g.Town.Party.Members = rescuers
	g.Town.Location = engine.EdgeOfTown
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(false))
	g.MazeMessage, g.MazeMessage2 = "", ""

	// At the entrance (the legacy fields' square): nobody.
	must(t, c.Inspect())
	if len(g.MazeInspectFound) != 0 {
		t.Fatalf("found %d at the entrance", len(g.MazeInspectFound))
	}
	must(t, c.LeaveInspect())

	g.PlayerX, g.PlayerY = 4, 4
	must(t, c.Inspect())
	if len(g.MazeInspectFound) != 6 {
		t.Fatalf("found %d on the square", len(g.MazeInspectFound))
	}
	before := records(t, fallen)
	if err := c.PickUp(); !errors.Is(err, ErrPickUpStranded) {
		t.Fatalf("pick up: %v", err)
	}
	c.HandleKey(Rune('P')) // the terminal's key: refused the same way
	if g.Town.Party.Size() != 6 || !g.MazeInspecting {
		t.Fatalf("party %d inspecting %v", g.Town.Party.Size(), g.MazeInspecting)
	}
	mustBeStrandedStill(t, fallen, before, here)
	must(t, c.LeaveInspect())
}

func mustBeStrandedStill(t *testing.T, chars []*engine.Character, before []string, here engine.Place) {
	t.Helper()
	for i, m := range chars {
		if m.Stranded == nil || *m.Stranded != here || !m.InMaze || record(t, m) != before[i] {
			t.Fatalf("%s: stranded %v in maze %v", m.Name, m.Stranded, m.InMaze)
		}
	}
}

// Stranded characters are saved and loaded with their squares, once each;
// the castle party stays empty. A second save and load changes nothing.
// (27, 34)
func TestStrandedSurviveSaveAndLoad(t *testing.T) {
	engine.SetSaveDir(t.TempDir())
	t.Cleanup(func() { engine.SetSaveDir("") })
	seed(27)
	c, party := walkers(t)
	trap(c, 0, 4, 4, data.SqPit)
	stepOnto(t, c, 0, 4, 4)
	here := place(c, 0, 4, 4)
	for round := 0; round < 2; round++ {
		must(t, c.Game.Save())
		g := engine.New(pristine(t))
		g.Title, g.Phase = nil, engine.PhaseTown
		must(t, g.Load())
		if g.Town.Party.Size() != 0 || len(g.Town.Roster.Characters) != 6 || g.Expedition != nil {
			t.Fatalf("loaded: party %d roster %d expedition %v", g.Town.Party.Size(), len(g.Town.Roster.Characters), g.Expedition != nil)
		}
		names := map[string]bool{}
		for i, ch := range g.Town.Roster.Characters {
			if names[ch.Name] {
				t.Fatalf("%s twice", ch.Name)
			}
			names[ch.Name] = true
			if ch.Stranded == nil || *ch.Stranded != here || !ch.InMaze {
				t.Fatalf("%s loaded: stranded %v", ch.Name, ch.Stranded)
			}
			// Loading rebuilds known spells (finding #32); the rest is as saved.
			cp, was := *ch, *party[i]
			cp.SpellKnown, was.SpellKnown = [50]bool{}, [50]bool{}
			if record(t, &cp) != record(t, &was) {
				t.Fatalf("%s changed by saving", ch.Name)
			}
		}
		c = New(g)
		party = g.Town.Roster.Characters
	}
}

// A saved encounter whose defeat was already decided is finished on
// resume: the party is stranded at the checkpoint's square before anything
// else can happen, and resuming the same save again gives the same result.
// An encounter not yet lost resumes as before. (29, 30)
func TestResumeFinishesADecidedDefeat(t *testing.T) {
	orig := pristine(t)
	setUp := func(lost bool) (*Controller, []*engine.Character) {
		c := explorersWithSaves(t)
		party := append([]*engine.Character(nil), c.Game.Town.Party.Members...)
		must(t, c.EnterMaze())
		must(t, c.LeaveCamp())
		must(t, c.Answer(false))
		g := c.Game
		g.PlayerX, g.PlayerY = 3, 2
		g.Combat, g.Phase = engine.NewCombat(g), engine.PhaseCombat
		for _, m := range party {
			m.InMaze = true
			if lost {
				m.Status, m.HP = engine.Dead, 0
			}
		}
		if lost {
			party[1].Status = engine.Paralyzed
			g.Combat.Phase = engine.CombatDefeat
		}
		exp, err := c.Checkpoint(orig)
		must(t, err)
		if exp.Interrupted != InterruptedCombat {
			t.Fatalf("interrupted %q", exp.Interrupted)
		}
		g.Expedition = exp
		must(t, g.Save())
		return c, party
	}
	load := func() *Controller {
		g := engine.New(pristine(t))
		g.Title, g.Phase = nil, engine.PhaseTown
		must(t, g.Load())
		c := New(g)
		must(t, c.Resume(orig))
		return c
	}

	_, party := setUp(true)
	want := records(t, party)
	for restart := 0; restart < 2; restart++ { // the same save, twice: the same result
		c := load()
		g := c.Game
		var loaded []*engine.Character
		for _, p := range party {
			for _, ch := range g.Town.Roster.Characters {
				if ch.Name == p.Name {
					loaded = append(loaded, ch)
				}
			}
		}
		if len(loaded) != 6 || g.Expedition != nil {
			t.Fatalf("loaded %d, expedition %v", len(loaded), g.Expedition != nil)
		}
		for i := range loaded {
			cp := *loaded[i]
			cp.SpellKnown = party[i].SpellKnown
			if record(t, &cp) != want[i] {
				t.Fatalf("%s changed by resuming", cp.Name)
			}
		}
		mustBeStranded(t, c, loaded, nil, place(c, 0, 3, 2))
		if loaded[1].Status != engine.Paralyzed {
			t.Fatalf("paralysis became %v", loaded[1].Status)
		}
		if err := c.Resume(orig); !errors.Is(err, ErrNotOnExpedition) {
			t.Fatalf("resume again: %v", err)
		}
	}

	setUp(false)
	c := load()
	if c.Game.Phase != engine.PhaseMaze || c.Game.Town.Party.Size() != 6 {
		t.Fatalf("an undecided encounter: phase %v party %d", c.Game.Phase, c.Game.Town.Party.Size())
	}
	noneStranded(t, c.Game.Town.Party.Members)
}

// An older save's dead party left walking (#28) cannot walk on after a
// restart: it is stranded at its checkpoint.
func TestResumeStrandsAnIncapacitatedWalkingParty(t *testing.T) {
	orig := pristine(t)
	c := explorersWithSaves(t)
	party := append([]*engine.Character(nil), c.Game.Town.Party.Members...)
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(false))
	c.Game.PlayerX, c.Game.PlayerY = 6, 1
	for _, m := range party {
		m.Status, m.InMaze = engine.Stoned, true
	}
	exp, err := c.Checkpoint(orig)
	must(t, err)
	c.Game.Expedition = exp
	must(t, c.Game.Save())
	g := engine.New(pristine(t))
	g.Title, g.Phase = nil, engine.PhaseTown
	must(t, g.Load())
	c2 := New(g)
	must(t, c2.Resume(orig))
	mustBeStranded(t, c2, g.Town.Roster.Characters, nil, place(c2, 0, 6, 1))
}

// Old saves have no stranded record: none is made up. A character an older
// version left out (InMaze, or a legacy LOSTXYL level) stays OUT with no
// square, and Inspect finds them by the legacy fields as before. (26, 32)
func TestLegacyOutCharactersGetNoSquare(t *testing.T) {
	dir := t.TempDir()
	engine.SetSaveDir(dir)
	t.Cleanup(func() { engine.SetSaveDir("") })
	legacy := `{"roster":[
	  {"Name":"DISBANDED","Status":0,"in_maze":true,"Level":1,"HP":5,"MaxHP":5},
	  {"Name":"OLDLEVEL","Status":5,"maze_level":3,"maze_x":4,"maze_y":5,"Level":1},
	  {"Name":"HOME","Status":0,"Level":1,"HP":5,"MaxHP":5}],"party":[]}`
	c := newController(t, 1)
	path, err := engine.RosterPath("1")
	must(t, err)
	must(t, os.WriteFile(path, []byte(legacy), 0o644))
	must(t, c.Game.Load())
	r := c.Game.Town.Roster.Characters
	for _, ch := range r {
		if ch.Stranded != nil {
			t.Fatalf("%s given a square %v", ch.Name, ch.Stranded)
		}
	}
	if !r[0].OutOfCastle() || !r[1].OutOfCastle() || r[2].OutOfCastle() {
		t.Fatalf("out of castle: %v %v %v", r[0].OutOfCastle(), r[1].OutOfCastle(), r[2].OutOfCastle())
	}
	if c.JoinRefusal(r[0]) != "** OUT **" || c.JoinRefusal(r[2]) != "" {
		t.Fatalf("join: %q %q", c.JoinRefusal(r[0]), c.JoinRefusal(r[2]))
	}
	// Renaming is refused only for the stranded and the expedition: a
	// legacy OUT character is handled as before.
	if err := c.RenameCharacter(0, "RENAMED"); err != nil {
		t.Fatalf("legacy rename: %v", err)
	}
	// Saved again, the file has no stranded record.
	must(t, c.Game.Save())
	raw, err := os.ReadFile(path)
	must(t, err)
	if strings.Contains(string(raw), "stranded") {
		t.Fatal("a stranded record was written for an old save")
	}
}
