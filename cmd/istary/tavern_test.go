package main

import (
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// tavernGame returns a scenario-1 game at Gilgamesh's Tavern with the given
// characters in the roster.
func tavernGame(t *testing.T, num int, roster ...*engine.Character) (*engine.GameState, *driver) {
	t.Helper()
	g := newGame(t, num)
	for _, c := range roster {
		g.Town.Roster.Add(c)
	}
	d := newDriver(t, g)
	d.keys("G")
	if g.Town.Location != engine.Tavern {
		t.Fatalf("G from castle should enter the tavern, got %v", g.Town.Location)
	}
	return g, d
}

func addMember(d *driver, name, password string) {
	d.keys("A").keys(name).enter()
	if d.game.Town.InputMode == engine.InputTavernPassword {
		d.keys(password).enter()
	}
}

func TestTavernAddMemberWithPassword(t *testing.T) {
	c := character("ARAGORN", engine.Human, engine.Fighter, engine.Good)
	c.Password = "STRIDER"
	g, d := tavernGame(t, 1, c)

	d.keys("A").keys("aragorn").enter()
	if g.Town.InputMode != engine.InputTavernPassword {
		t.Fatalf("known name should prompt for password, mode=%v msg=%q", g.Town.InputMode, g.Town.Message)
	}
	d.keys("strider").enter()
	if g.Town.Party.Size() != 1 || g.Town.Party.Members[0] != c {
		t.Fatalf("party = %v, want ARAGORN", g.Town.Party.Members)
	}
	if g.Town.Message != "ARAGORN JOINS THE PARTY!" {
		t.Errorf("message = %q", g.Town.Message)
	}
	// Current behavior: joining sets InMaze (Pascal INMAZE := TRUE).
	if !c.InMaze {
		t.Error("joining the party should set InMaze")
	}
}

func TestTavernWrongPassword(t *testing.T) {
	c := character("SAM", engine.Hobbit, engine.Thief, engine.Neutral)
	c.Password = "ROSIE"
	g, d := tavernGame(t, 1, c)
	addMember(d, "SAM", "WRONG")
	if g.Town.Party.Size() != 0 || g.Town.Message != "** THATS NOT IT **" {
		t.Errorf("party=%d message=%q", g.Town.Party.Size(), g.Town.Message)
	}
}

func TestTavernJoinRules(t *testing.T) {
	good := character("GOOD", engine.Human, engine.Fighter, engine.Good)
	evil := character("EVIL", engine.Human, engine.Thief, engine.Evil)
	neutral := character("NEUTRAL", engine.Human, engine.Mage, engine.Neutral)
	lost := character("LOST", engine.Human, engine.Fighter, engine.Good)
	lost.Status = engine.Lost
	out := character("OUT", engine.Human, engine.Fighter, engine.Good)
	out.InMaze = true

	g, d := tavernGame(t, 1, good, evil, neutral, lost, out)
	steps := []struct {
		name, msg string
		size      int
	}{
		{"GOOD", "GOOD JOINS THE PARTY!", 1},
		{"GOOD", "GOOD IS ALREADY IN THE PARTY!", 1},
		{"EVIL", "** BAD ALIGNMENT **", 1},
		{"NEUTRAL", "NEUTRAL JOINS THE PARTY!", 2},
		{"LOST", "** WHO? **", 2}, // lost characters are skipped in the search
		{"OUT", "** OUT **", 2},
		{"NOBODY", "** WHO? **", 2},
	}
	for _, s := range steps {
		addMember(d, s.name, "")
		if g.Town.Message != s.msg || g.Town.Party.Size() != s.size {
			t.Errorf("add %s: message=%q size=%d, want %q size %d", s.name, g.Town.Message, g.Town.Party.Size(), s.msg, s.size)
		}
	}
}

// Party alignment is the last non-neutral member's alignment.
func TestTavernNeutralPartyAcceptsEvil(t *testing.T) {
	n := character("N", engine.Human, engine.Fighter, engine.Neutral)
	e := character("E", engine.Human, engine.Fighter, engine.Evil)
	gd := character("G", engine.Human, engine.Fighter, engine.Good)
	g, d := tavernGame(t, 1, n, e, gd)
	addMember(d, "N", "")
	addMember(d, "E", "")
	addMember(d, "G", "")
	if g.Town.Party.Size() != 2 || g.Town.Message != "** BAD ALIGNMENT **" {
		t.Errorf("size=%d message=%q, want 2 and BAD ALIGNMENT", g.Town.Party.Size(), g.Town.Message)
	}
}

func TestTavernPartyLimitSix(t *testing.T) {
	var roster []*engine.Character
	for _, name := range []string{"P1", "P2", "P3", "P4", "P5", "P6", "P7"} {
		roster = append(roster, character(name, engine.Human, engine.Fighter, engine.Neutral))
	}
	g, d := tavernGame(t, 1, roster...)
	for _, c := range roster[:6] {
		addMember(d, c.Name, "")
	}
	if g.Town.Party.Size() != 6 {
		t.Fatalf("party size = %d, want 6", g.Town.Party.Size())
	}
	d.keys("A")
	if g.Town.Message != "** PARTY IS FULL **" || g.Town.InputMode != engine.InputNone {
		t.Errorf("message=%q mode=%v", g.Town.Message, g.Town.InputMode)
	}
}

func TestTavernRemoveMember(t *testing.T) {
	a := character("A", engine.Human, engine.Fighter, engine.Good)
	b := character("B", engine.Human, engine.Fighter, engine.Good)
	g, d := tavernGame(t, 1, a, b)
	addMember(d, "A", "")
	addMember(d, "B", "")

	d.keys("R1")
	if g.Town.Party.Size() != 1 || g.Town.Party.Members[0] != b {
		t.Fatalf("party = %v, want [B]", g.Town.Party.Members)
	}
	if g.Town.Message != "A LEAVES THE PARTY." || a.InMaze {
		t.Errorf("message=%q InMaze=%v", g.Town.Message, a.InMaze)
	}
	// Removed character stays in the roster.
	if len(g.Town.Roster.Characters) != 2 {
		t.Error("removing from the party must not delete from the roster")
	}
	// RETURN at "WHO WILL LEAVE" exits without removing.
	d.keys("R").enter()
	if g.Town.Party.Size() != 1 || g.Town.InputMode != engine.InputNone {
		t.Error("RETURN should cancel removal")
	}
}

func TestWiz3OnlyLegaciesMayJoin(t *testing.T) {
	plain := character("PLAIN", engine.Human, engine.Fighter, engine.Good)
	heir := character("HEIR", engine.Human, engine.Fighter, engine.Good)
	heir.IsLegacy = true
	g, d := tavernGame(t, 3, plain, heir)
	addMember(d, "PLAIN", "")
	if g.Town.Message != "** ONLY A MEMORY **" {
		t.Errorf("non-legacy in Wiz 3: message = %q", g.Town.Message)
	}
	addMember(d, "HEIR", "")
	if g.Town.Party.Size() != 1 {
		t.Errorf("legacy character should join in Wiz 3")
	}
}

func TestCastleShopsRequireParty(t *testing.T) {
	g := newGame(t, 1)
	d := newDriver(t, g)
	for _, k := range []string{"A", "B", "C"} {
		d.keys(k)
		if g.Town.Location != engine.Castle || g.Town.Message != "YOU HAVE NO PARTY!" {
			t.Errorf("%s with no party: location=%v message=%q", k, g.Town.Location, g.Town.Message)
		}
	}
	d.keys("E")
	d.keys("M")
	if g.Phase != engine.PhaseTown {
		t.Error("M at Edge of Town should not enter the maze without a party")
	}
}

func TestDeleteCharacterRemovesFromRosterAndParty(t *testing.T) {
	c := character("DOOMED", engine.Human, engine.Fighter, engine.Good)
	g, d := tavernGame(t, 1, c)
	addMember(d, "DOOMED", "")
	d.enter() // back to castle
	toTrainingName(d, "DOOMED").enter().enter() // empty password
	if g.Town.InputMode != engine.InputCharEdit {
		t.Fatalf("mode = %v, want character edit", g.Town.InputMode)
	}
	d.keys("D").keys("Y")
	if len(g.Town.Roster.Characters) != 0 || g.Town.Party.Size() != 0 {
		t.Errorf("delete: roster=%d party=%d", len(g.Town.Roster.Characters), g.Town.Party.Size())
	}
}
