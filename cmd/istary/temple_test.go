package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
	"github.com/gdamore/tcell/v2"
)

// Characterization tests for the Temple of Cant (T-0): they pin what the
// terminal does today, before the Temple moves into the controller. They
// change no rule. Pinned as found: the patient is found anywhere on the
// roster by name, in the party or not, and even out in the maze; any party
// member may pay, dead or not; a payer short of the donation is refused
// ("CHEAP APOSTATES! OUT!"; the guide says the party's gold is pooled,
// finding #5); the donation is taken before the treatment and kept when it
// fails; paying and the treatment are two keys.

// templeGame is a scenario-1 game at the castle with a party of four (a
// rich fighter, a poor mage, a paralysed priest, a dead thief who can still
// pay) and, outside the party, a dead fighter, a mage in ashes, a stoned
// fighter, a LOST one, a sleeping one, a well one and one out in the maze.
func templeGame(t *testing.T) (*engine.GameState, *driver) {
	t.Helper()
	g := newGame(t, 1)
	mk := func(name string, class engine.Class, st engine.Status, level, gold int) *engine.Character {
		c := character(name, engine.Human, class, engine.Good)
		c.Status, c.Level, c.Gold, c.Age = st, level, gold, 25*52
		if st >= engine.Dead {
			c.HP = 0
		}
		g.Town.Roster.Add(c)
		return c
	}
	for _, c := range []*engine.Character{
		mk("RICH", engine.Fighter, engine.OK, 1, 5000),
		mk("POOR", engine.Mage, engine.OK, 1, 10),
		mk("PARA", engine.Priest, engine.Paralyzed, 3, 0),
		mk("GHOST", engine.Thief, engine.Dead, 1, 2000),
	} {
		g.Town.Party.Members = append(g.Town.Party.Members, c)
	}
	mk("DEADGUY", engine.Fighter, engine.Dead, 2, 0)
	mk("ASHY", engine.Mage, engine.Ashed, 4, 0)
	mk("STONE", engine.Fighter, engine.Stoned, 1, 0)
	mk("LOSTY", engine.Fighter, engine.Lost, 1, 0)
	mk("SLEEPY", engine.Fighter, engine.Asleep, 1, 0)
	mk("OKAY", engine.Fighter, engine.OK, 1, 0)
	out := mk("OUTSIDE", engine.Fighter, engine.Dead, 1, 0)
	out.InMaze, out.MazeLevel = true, 2
	return g, newDriver(t, g)
}

var templeSteps = map[engine.TempleStep]string{
	engine.TempleWho: "who", engine.TempleStatus: "status", engine.TempleTithe: "tithe",
	engine.TempleRitual: "ritual", engine.TempleResult: "result",
}

// templeState is one line of what the Temple holds: where it is, the prompt
// and message, the party's gold, the patient, and every roster entry's
// condition, HP and age.
func templeState(g *engine.GameState) string {
	town := g.Town
	var b strings.Builder
	fmt.Fprintf(&b, "%v/%s input=%d buf=%q msg=%q", town.Location, templeSteps[town.TempleStep], town.InputMode, town.InputBuf, town.Message)
	for _, m := range town.Party.Members {
		fmt.Fprintf(&b, " %s:%d", m.Name, m.Gold)
	}
	if c := town.TempleChar; c != nil && town.Location == engine.Temple {
		fmt.Fprintf(&b, " | patient %s cost %d", c.Name, town.TempleCost)
	}
	if len(town.TempleMessages) > 0 {
		fmt.Fprintf(&b, " | %q", town.TempleMessages)
	}
	b.WriteString(" | roster")
	for _, c := range town.Roster.Characters {
		fmt.Fprintf(&b, " %s:%v/%d/%d", c.Name, c.Status, c.HP, c.Age)
	}
	return b.String()
}

// templePress presses keys: "\r" RETURN, "\x1b" Escape, "\b" Backspace;
// anything else is typed.
func templePress(d *driver, keys string) {
	for _, r := range keys {
		switch r {
		case '\r':
			d.enter()
		case '\x1b':
			d.esc()
		case '\b':
			d.key(tcell.KeyBackspace2)
		default:
			d.keys(string(r))
		}
	}
}

// templeSeed gives the transcript a raise from the dead (DEADGUY), ashes
// lost for good (ASHY) and a raise that fails into ashes (OUTSIDE).
const templeSeed = 5

var templeScript = []struct{ keys, why string }{
	{"C", "enter: who are you helping"},
	{"NOBODY\r", "no such name: nothing"},
	{"losty\r", "LOST: beyond help"},
	{"okay\r", "well: nothing to do"},
	{"sleepy\r", "asleep: no donation, so OK"},
	{"PARX\bA\r", "the paralysed priest, typed with a correction: 100 x level 3"},
	{"2", "the poor mage pays: refused, nothing taken"},
	{"\r", "back to who"},
	{"PARA\r", ""},
	{"9", "no such member: nothing"},
	{"1", "the rich fighter pays 300: the ritual"},
	{"Z", "any key: treated"},
	{"\r", ""},
	{"DEADGUY\r", "dead, not in the party: 250 x level 2"},
	{"4", "the dead thief pays 500"},
	{"\r", "RETURN at the ritual treats too"},
	{"\r", ""},
	{"ASHY\r", "ashes: 500 x level 4 (it fails: LOST, the donation kept)"},
	{"1", ""},
	{"Z", ""},
	{"\r", ""},
	{"OUTSIDE\r", "dead and out in the maze, treated from the castle (it fails: ashes)"},
	{"1", ""},
	{"Z", ""},
	{"\r", ""},
	{"STONE\r", "stoned: 200"},
	{"\r", "RETURN at the tithe: back to who"},
	{"\r", "RETURN with nothing typed: the castle"},
}

// The whole session, state by state, is pinned in a golden file.
func TestTempleTranscript(t *testing.T) {
	g, d := templeGame(t)
	var out strings.Builder
	out.WriteString("start " + templeState(g) + "\n")
	seed(templeSeed)
	for _, step := range templeScript {
		templePress(d, step.keys)
		key := strings.NewReplacer("\r", "RET", "\b", "BS", "\x1b", "ESC").Replace(step.keys)
		fmt.Fprintf(&out, "%-10s %s\n", key, templeState(g))
		if step.why != "" {
			fmt.Fprintf(&out, "           # %s\n", step.why)
		}
	}
	checkGolden(t, "temple_transcript.golden", []byte(out.String()))
}

// Every Temple screen as the terminal draws it.
func TestTempleScreenSnapshots(t *testing.T) {
	g, _ := templeGame(t)
	d := newDriverWithScreen(t, g)
	var out strings.Builder
	snap := func(name string) {
		d.screen.Clear()
		redraw(d.screen.Screen, g)
		out.WriteString("=== " + name + "\n" + d.screen.text() + "\n")
	}
	seed(templeSeed)
	templePress(d, "C")
	snap("who are you helping")
	templePress(d, "LOS")
	snap("typing")
	templePress(d, "TY\r")
	snap("LOST")
	templePress(d, "PARA\r")
	snap("the donation, who will tithe")
	templePress(d, "2")
	snap("cheap apostates")
	templePress(d, "\rPARA\r1")
	snap("the ritual")
	templePress(d, "Z")
	snap("the result")
	checkGolden(t, "screens_temple.golden", []byte(out.String()))
}

// treat takes patient name to the Temple's result with payer slot (1-based)
// from the castle, and back to "who".
func treat(d *driver, name string, payer int) {
	templePress(d, "C"+name+"\r"+fmt.Sprint(payer)+"Z")
}

func roster(g *engine.GameState, name string) *engine.Character {
	for _, c := range g.Town.Roster.Characters {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// The rules the transcript shows, stated one by one.
func TestTempleRules(t *testing.T) {
	t.Run("donations by level", func(t *testing.T) {
		for _, level := range []int{1, 3, 7} {
			g, d := templeGame(t)
			roster(g, "DEADGUY").Level = level
			templePress(d, "CDEADGUY\r")
			if g.Town.TempleStep != engine.TempleTithe || g.Town.TempleCost != 250*level {
				t.Errorf("level %d: %v cost %d", level, g.Town.TempleStep, g.Town.TempleCost)
			}
		}
	})
	t.Run("short of the donation: refused, nothing taken, nothing drawn", func(t *testing.T) {
		g, d := templeGame(t)
		templePress(d, "CPARA\r")
		seed(5)
		templePress(d, "2")
		next := rand.Int63()
		seed(5)
		if g.Town.TempleStep != engine.TempleResult || fmt.Sprint(g.Town.TempleMessages) != "[CHEAP APOSTATES! OUT!]" ||
			g.Town.Party.Members[1].Gold != 10 || roster(g, "PARA").Status != engine.Paralyzed || next != rand.Int63() {
			t.Errorf("%v %q gold %d", g.Town.TempleStep, g.Town.TempleMessages, g.Town.Party.Members[1].Gold)
		}
	})
	t.Run("paid before the treatment, kept when it fails", func(t *testing.T) {
		g, d := templeGame(t)
		ash := roster(g, "ASHY")
		ash.Vitality = 3 // 40+9: fails above 49
		for s := int64(1); ; s++ {
			rand.Seed(s)
			if rand.Intn(100) > 49 {
				seed(s)
				break
			}
		}
		templePress(d, "CASHY\r1")
		if g.Town.Party.Members[0].Gold != 3000 || g.Town.TempleStep != engine.TempleRitual || ash.Status != engine.Ashed {
			t.Fatalf("before the ritual: gold %d %v", g.Town.Party.Members[0].Gold, ash.Status)
		}
		templePress(d, "Z")
		if ash.Status != engine.Lost || g.Town.Party.Members[0].Gold != 3000 || fmt.Sprint(g.Town.TempleMessages) != "[ASHY WILL BE BURIED]" {
			t.Errorf("after: %v gold %d %q", ash.Status, g.Town.Party.Members[0].Gold, g.Town.TempleMessages)
		}
		if roster(g, "ASHY") == nil {
			t.Error("the LOST was removed from the roster")
		}
	})
	t.Run("anyone on the roster, anywhere; any member pays", func(t *testing.T) {
		g, d := templeGame(t)
		roster(g, "OUTSIDE").Vitality = 18 // never fails
		treat(d, "OUTSIDE", 4)             // the dead thief pays
		if o := roster(g, "OUTSIDE"); o.Status != engine.OK || o.HP != 1 || g.Town.Party.Members[3].Gold != 1750 {
			t.Errorf("outside: %v HP %d, thief %d", o.Status, o.HP, g.Town.Party.Members[3].Gold)
		}
	})
	t.Run("the result is saved after the key, as the main loop saves", func(t *testing.T) {
		g, d := templeGame(t)
		roster(g, "DEADGUY").Vitality = 18
		treat(d, "DEADGUY", 1)
		if err := d.ctrl.Save(); err != nil {
			t.Fatal(err)
		}
		g2 := newGame(t, 1)
		if err := g2.Load(); err != nil {
			t.Fatal(err)
		}
		if c := roster(g2, "DEADGUY"); c == nil || c.Status != engine.OK || c.HP != 1 || c.Age == 25*52 || roster(g2, "RICH").Gold != 4500 {
			t.Errorf("loaded: %+v", c)
		}
	})
	t.Run("escape and an empty name leave", func(t *testing.T) {
		g, d := templeGame(t)
		templePress(d, "CAB\x1b")
		if g.Town.Location != engine.Castle || g.Town.InputMode != engine.InputNone {
			t.Errorf("escape: %v", g.Town.Location)
		}
		templePress(d, "C\r")
		if g.Town.Location != engine.Castle {
			t.Errorf("empty name: %v", g.Town.Location)
		}
		g.Town.Party.Members = nil
		templePress(d, "C")
		if g.Town.Location != engine.Castle || g.Town.Message != "YOU HAVE NO PARTY!" {
			t.Errorf("no party: %v %q", g.Town.Location, g.Town.Message)
		}
	})
}
