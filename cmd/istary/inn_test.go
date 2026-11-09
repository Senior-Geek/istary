package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// Characterization tests for the Adventurer's Inn (I-0): they pin what the
// terminal does today, before the Inn moves into the controller. They change
// no rule. Where today's behaviour is a known question (findings #10, #11,
// N2, N3) it is pinned as it is and said so: spells come back in any room,
// a stay ages no one, a stay cannot be cut short, and there is no Pool Gold.

// innGame is a scenario-1 game at the castle with a party of five: a
// wounded fighter ready for level 2 (and short of level 3), a wounded mage
// with spells spent and little gold, a dead priest, an old fighter with
// experience for two levels, and a rich, badly hurt fighter.
func innGame(t *testing.T) (*engine.GameState, *driver) {
	t.Helper()
	g := newGame(t, 1)
	arne := character("ARNE", engine.Dwarf, engine.Fighter, engine.Good)
	arne.HP, arne.MaxHP, arne.Gold, arne.XP, arne.Age = 3, 20, 1000, 1000, 20*52
	mage := character("MAGE", engine.Elf, engine.Mage, engine.Good)
	mage.HP, mage.MaxHP, mage.Gold, mage.Age = 2, 8, 30, 19*52
	mage.MaxMageSpells = [7]int{2, 1}
	mage.MageSpells = [7]int{0, 0}
	priest := character("PRIEST", engine.Human, engine.Priest, engine.Good)
	priest.Status, priest.HP, priest.Gold = engine.Dead, 0, 500
	old := character("OLD", engine.Human, engine.Fighter, engine.Good)
	old.Age, old.Vitality, old.XP, old.Gold = 200*52, 3, 1724, 5
	rich := character("RICH", engine.Human, engine.Fighter, engine.Good)
	rich.HP, rich.MaxHP, rich.Gold, rich.Age = 1, 40, 10000, 25*52
	for _, c := range []*engine.Character{arne, mage, priest, old, rich} {
		g.Town.Roster.Add(c)
		g.Town.Party.Members = append(g.Town.Party.Members, c)
	}
	return g, newDriver(t, g)
}

var innSteps = map[engine.InnStep]string{
	engine.InnWho: "who", engine.InnSelectRoom: "rooms", engine.InnHealing: "healing", engine.InnLevelUp: "levelUp",
}

// innState is one line of what the Inn holds: where it is, the message,
// every member's gold, and the guest in full.
func innState(g *engine.GameState) string {
	town := g.Town
	var b strings.Builder
	fmt.Fprintf(&b, "%v/%s input=%d msg=%q", town.Location, innSteps[town.InnStep], town.InputMode, town.Message)
	for _, m := range town.Party.Members {
		fmt.Fprintf(&b, " %s:%d", m.Name, m.Gold)
	}
	if c := town.InnChar; c != nil && town.Location == engine.Inn {
		fmt.Fprintf(&b, " | %s %v L%d XP%d HP%d/%d age%d stats%v mage%v/%v priest%v/%v",
			c.Name, c.Status, c.Level, c.XP, c.HP, c.MaxHP, c.Age,
			[6]int{c.Strength, c.IQ, c.Piety, c.Vitality, c.Agility, c.Luck},
			c.MageSpells, c.MaxMageSpells, c.PriestSpells, c.MaxPriestSpells)
		if town.InnStep == engine.InnHealing {
			fmt.Fprintf(&b, " | heal %d per %d GP", town.InnHealAmt, town.InnHealCost)
		}
		if len(town.InnMessages) > 0 {
			fmt.Fprintf(&b, " | %q", town.InnMessages)
		}
	}
	return b.String()
}

// innPress presses keys: "\r" is RETURN and "~" is one tick of the Inn's
// healing animation (the terminal's 0.7-second timer).
func innPress(d *driver, keys string) {
	for _, r := range keys {
		switch r {
		case '~':
			d.ctrl.InnTick()
		case '\r':
			d.enter()
		default:
			d.keys(string(r))
		}
	}
}

// innSeed makes OLD die of old age at his level-up in the transcript.
const innSeed = 1

// innScript is a session at the Inn, key by key and tick by tick.
var innScript = []struct{ keys, why string }{
	{"A", "enter: who will stay"},
	{"9", "no such member: nothing"},
	{"3", "the dead priest: IS NOT OK"},
	{"2", "the mage: the rooms"},
	{"A", "the stables: spells already back, no healing"},
	{"Z", "a key while napping: nothing"},
	{"~", "the tick: the level check, not enough experience"},
	{"Z", "a key at the result: nothing"},
	{"\r", "RETURN: the rooms again, same guest"},
	{"B", "a cot, 10 GP a week, with 30 GP"},
	{"\r", "RETURN while healing: nothing"},
	{"~", "week 1"},
	{"~", "week 2"},
	{"~", "week 3: the gold is gone"},
	{"~", "out of gold: the level check"},
	{"\r", ""},
	{"\r", "RETURN at the rooms: who will stay"},
	{"1", "the fighter, 3/20 HP, 1000 GP, experience for level 2"},
	{"C", "economy, 3 HP for 50 GP a week"},
	{"~~~~~", "five weeks"},
	{"~", "week 6 heals the last 2 HP and charges a full week"},
	{"~", "healed: the level check makes level 2 (random)"},
	{"\r", ""},
	{"A", "the stables: another level check; not enough for level 3"},
	{"~", ""},
	{"\r", ""},
	{"\r", ""},
	{"5", "the rich fighter, 1/40 HP"},
	{"E", "the royal suite, 10 HP for 500 GP a week"},
	{"~~~", ""},
	{"~", "week 4: healed"},
	{"~", "the level check"},
	{"\r", ""},
	{"D", "a merchant suite when already healed: no charge"},
	{"~", ""},
	{"\r", ""},
	{"\r", ""},
	{"4", "the old fighter, 200 years, vitality 3, experience for level 3"},
	{"A", "the stables"},
	{"~", "level 2 only, and he dies of old age: LOST"},
	{"\r", "RETURN: the rooms again for a LOST guest"},
	{"\r", ""},
	{"\r", "RETURN at who will stay: the castle"},
}

// The whole session, state by state, is pinned in a golden file. Healing
// draws no random number; the level checks do.
func TestInnTranscript(t *testing.T) {
	g, d := innGame(t)
	var out strings.Builder
	out.WriteString("start " + innState(g) + "\n")
	seed(innSeed)
	for _, step := range innScript {
		innPress(d, step.keys)
		key := strings.ReplaceAll(step.keys, "\r", "RET")
		fmt.Fprintf(&out, "%-5s %s\n", key, innState(g))
		if step.why != "" {
			fmt.Fprintf(&out, "      # %s\n", step.why)
		}
	}
	checkGolden(t, "inn_transcript.golden", []byte(out.String()))
}

// Every Inn screen as the terminal draws it.
func TestInnScreenSnapshots(t *testing.T) {
	g, _ := innGame(t)
	d := newDriverWithScreen(t, g)
	var out strings.Builder
	snap := func(name string) {
		d.screen.Clear()
		redraw(d.screen.Screen, g)
		out.WriteString("=== " + name + "\n" + d.screen.text() + "\n")
	}
	seed(innSeed)
	innPress(d, "A")
	snap("who will stay")
	innPress(d, "3")
	snap("a guest who is not OK")
	innPress(d, "2")
	snap("the rooms")
	innPress(d, "A")
	snap("napping in the stables")
	innPress(d, "~")
	snap("not enough experience")
	innPress(d, "\rB~")
	snap("healing in a cot")
	innPress(d, "~~~\r\r1C~~~~~~~")
	snap("made a level")
	checkGolden(t, "screens_inn.golden", []byte(out.String()))
}

// stay checks guest slot (1-based) into room (A-E) and ticks until the
// level check is shown. It fails if the healing drew a random number.
func stay(t *testing.T, d *driver, slot int, room string) {
	t.Helper()
	innPress(d, "A"+fmt.Sprint(slot)+room)
	town := d.game.Town
	for town.InnStep == engine.InnHealing {
		c := town.InnChar
		hp, gold := c.HP, c.Gold
		seed(7)
		if d.ctrl.InnTick(); town.InnStep == engine.InnHealing {
			next := rand.Int63()
			seed(7)
			if next != rand.Int63() {
				t.Fatal("a week of healing drew a random number")
			}
			if c.HP == hp && c.Gold == gold {
				t.Fatal("a tick changed nothing and did not end the stay")
			}
		}
	}
}

// The rules the transcript shows, stated one by one.
func TestInnRules(t *testing.T) {
	rates := []struct {
		room     string
		hp, cost int
	}{{"B", 1, 10}, {"C", 3, 50}, {"D", 7, 200}, {"E", 10, 500}}
	t.Run("weeks, healing and charges", func(t *testing.T) {
		// A week heals the room's rate (to the maximum at most) and charges
		// a full week; the stay goes on while hurt and able to pay.
		for _, r := range rates {
			for _, start := range []struct{ hp, max, gold int }{
				{1, 40, 100000}, {39, 40, 100000}, {1, 40, 0}, {1, 40, r.cost - 1}, {1, 40, 3*r.cost + 1}, {40, 40, 1000},
			} {
				g, d := innGame(t)
				c := g.Town.Party.Members[4]
				c.HP, c.MaxHP, c.Gold = start.hp, start.max, start.gold
				stay(t, d, 5, r.room)
				weeks := min((start.max-start.hp+r.hp-1)/r.hp, start.gold/r.cost)
				if c.HP != min(start.max, start.hp+weeks*r.hp) || c.Gold != start.gold-weeks*r.cost {
					t.Errorf("room %s from %+v: HP %d, gold %d after %d weeks", r.room, start, c.HP, c.Gold, weeks)
				}
				if c.Age != 25*52 {
					t.Errorf("room %s: age %d; a stay ages no one today (#11)", r.room, c.Age)
				}
			}
		}
	})
	t.Run("stables", func(t *testing.T) {
		g, d := innGame(t)
		m := g.Town.Party.Members[1]
		stay(t, d, 2, "A")
		if m.HP != 2 || m.Gold != 30 || m.MageSpells != m.MaxMageSpells {
			t.Errorf("stables: HP %d, gold %d, spells %v of %v", m.HP, m.Gold, m.MageSpells, m.MaxMageSpells)
		}
	})
	t.Run("spells come back when the room is chosen, in any room", func(t *testing.T) {
		for _, room := range []string{"A", "B", "C", "D", "E"} {
			g, d := innGame(t)
			m := g.Town.Party.Members[1]
			m.Gold = 0
			innPress(d, "A2"+room)
			if m.MageSpells != m.MaxMageSpells {
				t.Errorf("room %s: spells %v before any tick", room, m.MageSpells)
			}
		}
	})
	t.Run("only OK guests", func(t *testing.T) {
		for _, st := range []engine.Status{engine.Asleep, engine.Afraid, engine.Paralyzed, engine.Stoned, engine.Dead, engine.Ashed, engine.Lost} {
			g, d := innGame(t)
			g.Town.Party.Members[0].Status = st
			innPress(d, "A1")
			if g.Town.InnStep != engine.InnWho || g.Town.Message != "ARNE IS NOT OK" {
				t.Errorf("%v: %v %q", st, g.Town.InnStep, g.Town.Message)
			}
		}
	})
	t.Run("no party", func(t *testing.T) {
		g, d := innGame(t)
		g.Town.Party.Members = nil
		innPress(d, "A")
		if g.Town.Location != engine.Castle || g.Town.Message != "YOU HAVE NO PARTY!" {
			t.Errorf("%v %q", g.Town.Location, g.Town.Message)
		}
	})
	t.Run("one level per stay", func(t *testing.T) {
		g, d := innGame(t)
		a := g.Town.Party.Members[0]
		a.XP = 1724 // enough for level 3
		seed(1)
		stay(t, d, 1, "A")
		if a.Level != 2 || a.MaxLevAC != 2 || g.Town.InnMessages[0] != "YOU MADE A LEVEL!" {
			t.Fatalf("first stay: level %d, %q", a.Level, g.Town.InnMessages)
		}
		innPress(d, "\rA~")
		if a.Level != 3 {
			t.Fatalf("second stay: level %d", a.Level)
		}
	})
	t.Run("experience still needed", func(t *testing.T) {
		g, d := innGame(t)
		stay(t, d, 2, "A")
		want := []string{"", "YOU NEED 1100 MORE", "EXPERIENCE POINTS TO MAKE LEVEL"}
		if fmt.Sprint(g.Town.InnMessages) != fmt.Sprint(want) {
			t.Errorf("%q", g.Town.InnMessages)
		}
	})
	t.Run("death of old age", func(t *testing.T) {
		g, d := innGame(t)
		o := g.Town.Party.Members[3]
		seed(innSeed)
		innPress(d, "A1C~~~~~~~\r\r4A~") // ARNE's level first, as in the transcript
		msgs := g.Town.InnMessages
		if o.Status != engine.Lost || o.HP != 0 || msgs[len(msgs)-1] != "** YOU HAVE DIED OF OLD AGE **" || o.Vitality != 2 {
			t.Errorf("status %v HP %d VIT %d: %q", o.Status, o.HP, o.Vitality, msgs)
		}
		if !inParty(g, o) || g.Town.InnChar != o {
			t.Error("the dead guest left the party or the Inn")
		}
	})
	t.Run("nothing to cut a stay short", func(t *testing.T) {
		// The guide's SPACE checks out early (N2); today no key does.
		g, d := innGame(t)
		c := g.Town.Party.Members[4]
		innPress(d, "A5B~ \r")
		innPress(d, "~")
		if c.HP != 3 || g.Town.InnStep != engine.InnHealing {
			t.Errorf("after SPACE and RETURN: HP %d, %v", c.HP, g.Town.InnStep)
		}
	})
}

func inParty(g *engine.GameState, c *engine.Character) bool {
	for _, m := range g.Town.Party.Members {
		if m == c {
			return true
		}
	}
	return false
}
