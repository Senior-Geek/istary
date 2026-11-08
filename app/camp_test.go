package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// campers is a party in Camp with spells, gold, poison and mixed items
// (some cursed, some equipped, some unidentified), chosen by rng, at a
// random square; saves go to a temporary folder.
func campers(t *testing.T, rng *rand.Rand) *Controller {
	t.Helper()
	engine.SetSaveDir(t.TempDir())
	t.Cleanup(func() { engine.SetSaveDir("") })
	c := newController(t, 1)
	g := c.Game
	for i, cls := range []engine.Class{engine.Fighter, engine.Samurai, engine.Priest, engine.Bishop, engine.Mage, engine.Lord} {
		ch := engine.NewCharacter(fmt.Sprintf("CAMPER %d", i+1), engine.Human, cls, engine.Neutral)
		ch.Level, ch.MaxHP, ch.HP = 13, 80, 20+rng.Intn(60)
		ch.Gold, ch.PoisonAmt = rng.Intn(500), rng.Intn(3)
		for l := 0; l < 7; l++ {
			ch.MageSpells[l], ch.MaxMageSpells[l] = rng.Intn(3), 3
			ch.PriestSpells[l], ch.MaxPriestSpells[l] = rng.Intn(3), 3
		}
		// Known spells as Load leaves them (it rebuilds them from the
		// spell slots, finding #32), so a reload changes nothing.
		ch.SpellKnown = [50]bool{}
		ch.MigrateSpellKnown()
		for n := 0; n < 3+rng.Intn(5); n++ {
			ch.AddItem(1+rng.Intn(len(g.Scenario.Items)-1), rng.Intn(2) == 0)
			if rng.Intn(5) == 0 {
				ch.Items[ch.ItemCount-1].Cursed = true
			}
			if rng.Intn(3) == 0 {
				ch.EquipItem(ch.ItemCount - 1)
			}
		}
		ch.InMaze = true
		g.Town.Roster.Add(ch)
		g.Town.Party.Members = append(g.Town.Party.Members, ch)
	}
	must(t, c.EnterMaze())
	g.MazeLevel, g.PlayerX, g.PlayerY = rng.Intn(10), rng.Intn(20), rng.Intn(20)
	g.InitFightMap()
	return c
}

// campTrace describes everything Camp can change: its screen and progress,
// the maze state, the party order and every roster character in full.
func campTrace(g *engine.GameState) string {
	t := g.Town
	name := func(c *engine.Character) string {
		if c == nil {
			return "-"
		}
		return c.Name
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%v mode%v %q/%q buf%q edit%s cat%d %v pm%v/%d ro%d", g.Phase, t.InputMode, t.Message, t.Message2,
		t.InputBuf, name(t.EditChar), t.EquipCategory, t.EquipChoices, t.EquipPartyMode, t.EquipPartyIdx, t.ReorderPos)
	for _, r := range t.ReorderResult {
		b.WriteString(" r" + name(r))
	}
	fmt.Fprintf(&b, " tt%s sp%v mal%d/%d/%d", name(t.TradeTarget), t.InspectSpells, t.MalorDeltaEW, t.MalorDeltaNS, t.MalorDeltaUD)
	if t.PendingSpell != nil {
		b.WriteString(" pending " + t.PendingSpell.Name)
	}
	fmt.Fprintf(&b, " L%d(%d,%d)%v li%d pr%d", g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing, g.LightLevel, g.ProtectLevel)
	b.WriteString(" party")
	for _, m := range t.Party.Members {
		b.WriteString(" " + name(m))
	}
	raw, _ := json.Marshal(t.Roster.Characters)
	b.Write(raw)
	return b.String()
}

var campWords = []string{"dios", "dial", "dialma", "madi", "latumofis", "milwa", "lomilwa", "dumapic", "malor",
	"katino", "halito", "badios", "di", "kadorto", "maporfic", "xyz"}

// campKeyFor picks a key for the open screen, as a player might: spell
// names at the cast prompt, targets, MALOR directions, menu choices, and
// otherwise anything.
func campKeyFor(g *engine.GameState, rng *rand.Rand) []Key {
	runes := func(s string) []Key {
		var ks []Key
		for _, r := range s {
			ks = append(ks, Rune(r))
		}
		return ks
	}
	switch mode := g.Town.InputMode; {
	case mode == engine.InputCastSpell && rng.Intn(5) > 0:
		return append(runes(campWords[rng.Intn(len(campWords))]), Key{Kind: KeyEnter})
	case mode == engine.InputSpellTarget && rng.Intn(5) > 0:
		return runes(string(rune('1' + rng.Intn(6))))
	case mode == engine.InputMalor && rng.Intn(4) > 0:
		return runes(string("NSEWUD"[rng.Intn(6)]))
	case mode == engine.InputInspect && rng.Intn(3) == 0:
		return runes("S")
	case mode == engine.InputNone && rng.Intn(2) == 0:
		return runes(string("123456RE"[rng.Intn(8)]))
	}
	switch r := rng.Intn(20); {
	case r < 3:
		return []Key{{Kind: KeyEnter}}
	case r < 4:
		return []Key{{Kind: KeyEscape}}
	case r < 5:
		return []Key{{Kind: KeyBackspace}}
	case r < 6:
		return runes(campWords[rng.Intn(len(campWords))])
	}
	return runes(string("0123456789REDLSUITMPNW"[rng.Intn(22)]))
}

// partyGoods lists the party's gold and every item (index and flags).
func partyGoods(g *engine.GameState) (int, []string) {
	gold, items := 0, []string(nil)
	for _, m := range g.Town.Party.Members {
		if m == nil {
			continue
		}
		gold += m.Gold
		for i := 0; i < m.ItemCount; i++ {
			items = append(items, fmt.Sprintf("%d/%v", m.Items[i].ItemIndex, m.Items[i].Cursed))
		}
	}
	sort.Strings(items)
	return gold, items
}

// Camp never creates or loses gold or items: across random play, gold is
// constant and items go only by D)ROP, one at a time.
func TestCampKeepsGoldAndItems(t *testing.T) {
	traded, dropped := 0, 0
	for s := int64(1); s <= 80; s++ {
		seed(s)
		rng := rand.New(rand.NewSource(s))
		c := campers(t, rng)
		for i := 0; i < 300 && c.Game.Phase == engine.PhaseCamp; i++ {
			gold, items := partyGoods(c.Game)
			mode := c.Game.Town.InputMode
			c.press(campKeyFor(c.Game, rng)...)
			if c.Game.Phase != engine.PhaseCamp {
				break
			}
			gold2, items2 := partyGoods(c.Game)
			if gold2 != gold {
				t.Fatalf("seed %d: party gold %d became %d", s, gold, gold2)
			}
			switch {
			case slices.Equal(items, items2):
				if c.Game.Town.Message == "TRADED" && mode == engine.InputTradeTarget {
					traded++
				}
			case mode == engine.InputDrop && len(items2) == len(items)-1:
				dropped++
			default:
				t.Fatalf("seed %d: items %v became %v (mode %v)", s, items, items2, mode)
			}
		}
	}
	if traded == 0 || dropped == 0 {
		t.Fatalf("traded %d, dropped %d", traded, dropped)
	}
}

// opFor performs a CampView action with arguments chosen by rng.
func opFor(c *Controller, act string, rng *rand.Rand) error {
	slot, n := rng.Intn(6), 1+rng.Intn(8)
	switch act {
	case ActCampInspect:
		return c.CampInspect(slot)
	case ActReorder:
		return c.BeginReorder()
	case ActReorderPick:
		return c.ReorderPick(slot)
	case ActEquipParty:
		return c.EquipParty()
	case ActDisband:
		return c.Disband()
	case ActLeaveCamp:
		return c.LeaveCamp()
	case ActLeaveInspect:
		return c.LeaveInspection()
	case ActEquip:
		return c.BeginEquip()
	case ActEquipChoose:
		return c.EquipChoose(1 + rng.Intn(max(1, len(c.Game.Town.EquipChoices))))
	case ActEquipSkip:
		return c.EquipSkip()
	case ActDrop:
		return c.BeginDrop()
	case ActDropItem:
		return c.DropItem(n)
	case ActEndDrop:
		return c.EndDrop()
	case ActTrade:
		return c.BeginTrade()
	case ActTradeWith:
		return c.TradeWith(slot)
	case ActTradeGold:
		if rng.Intn(3) == 0 {
			return c.TradeGold(rng.Intn(300))
		}
		return c.TradeGold(0)
	case ActTradeItem:
		return c.TradeItem(n)
	case ActEndTrade:
		return c.EndTrade()
	case ActSpellBooks:
		return c.OpenSpellBooks()
	case ActSpellBook:
		return c.ShowSpellBook(rng.Intn(2) == 0)
	case ActCloseBook:
		return c.CloseSpellBook()
	case ActCast:
		switch rng.Intn(4) {
		case 0:
			return c.CastSpell("dumapic")
		case 1:
			return c.CastSpell("dios")
		}
		return c.CastSpell(campWords[rng.Intn(len(campWords))])
	case ActCastOn:
		return c.CastOn(slot)
	case ActCancelCast:
		return c.CancelCast()
	case ActIdentify:
		return c.Identify()
	case ActUse:
		return c.UseItem(n)
	case ActLeaveDumapic:
		return c.LeaveDumapic()
	}
	return fmt.Errorf("unknown action %q", act)
}

var allCampActs = []string{ActCampInspect, ActReorder, ActReorderPick, ActEquipParty, ActDisband, ActLeaveCamp,
	ActLeaveInspect, ActEquip, ActEquipChoose, ActEquipSkip, ActDrop, ActDropItem, ActEndDrop, ActTrade,
	ActTradeWith, ActTradeGold, ActTradeItem, ActEndTrade, ActSpellBooks, ActSpellBook, ActCloseBook, ActCast,
	ActCastOn, ActCancelCast, ActIdentify, ActUse, ActLeaveDumapic}

// The operations are the boundary: on every screen, each action CampView
// lists is offered (never ErrWrongState), and every other one is refused
// without changing anything or drawing a random number. Refused arguments
// (ErrInvalidChoice, ErrNoSuchPartyMember, MALOR) change nothing either.
func TestCampViewActionsAreTheOperations(t *testing.T) {
	used := map[string]int{}
	for s := int64(1); s <= 60; s++ {
		seed(s)
		rng := rand.New(rand.NewSource(s))
		c := campers(t, rng)
		for i := 0; i < 300 && c.Game.Phase == engine.PhaseCamp; i++ {
			v, err := c.CampView()
			must(t, err)
			for _, act := range allCampActs {
				if slices.Contains(v.Actions, act) {
					continue
				}
				before, probe := campTrace(c.Game), rand.Int63()
				seed(probe)
				if err := opFor(c, act, rng); !errors.Is(err, ErrWrongState) {
					t.Fatalf("seed %d: %s not listed on %s but gave %v", s, act, v.Screen, err)
				}
				if campTrace(c.Game) != before || rand.Int63() != rand.New(rand.NewSource(probe)).Int63() {
					t.Fatalf("seed %d: refused %s changed the game", s, act)
				}
			}
			act := v.Actions[rng.Intn(len(v.Actions))]
			if act == ActLeaveCamp && rng.Intn(4) > 0 {
				continue
			}
			before := campTrace(c.Game)
			err = opFor(c, act, rng)
			if errors.Is(err, ErrWrongState) {
				t.Fatalf("seed %d: listed %s on %s refused", s, act, v.Screen)
			}
			if err != nil && campTrace(c.Game) != before {
				t.Fatalf("seed %d: %s failed (%v) but changed the game", s, act, err)
			}
			if err == nil {
				used[act]++
			}
		}
	}
	t.Logf("operations used: %v", used)
	for _, act := range allCampActs {
		if used[act] == 0 {
			t.Errorf("never used: %s", act)
		}
	}
}

// DUMAPIC's result is in CampView only while its screen is open; the
// dungeon view never has the position.
func TestDumapicIsConfinedToItsScreen(t *testing.T) {
	seed(1)
	c := campers(t, rand.New(rand.NewSource(1)))
	g := c.Game
	g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = 2, 7, 11, engine.South
	priest := g.Town.Party.Members[2]
	priest.PriestSpells, priest.SpellKnown = [7]int{3, 3, 3, 3, 3, 3, 3}, [50]bool{}
	priest.SpellKnown[engine.SpellIndex["DUMAPIC"]] = true
	priest.Class = engine.Mage
	priest.MageSpells = [7]int{3, 3, 3, 3, 3, 3, 3}
	v, _ := c.CampView()
	if v.Dumapic != nil {
		t.Fatal("DUMAPIC result at the menu")
	}
	must(t, c.CampInspect(2))
	must(t, c.CastSpell("dumapic"))
	v, _ = c.CampView()
	if v.Screen != "dumapic" || v.Dumapic == nil || *v.Dumapic != (DumapicView{Facing: "SOUTH", East: 7, North: 11, Level: 3}) {
		t.Fatalf("dumapic: %+v %+v (%q)", v.Screen, v.Dumapic, g.Town.Message)
	}
	raw, _ := json.Marshal(c.DungeonView())
	for _, k := range []string{`"level"`, `"x"`, `"y"`, `"facing"`, `"east"`, `"north"`, `"dumapic"`} {
		if strings.Contains(string(raw), k) {
			t.Fatalf("dungeon view has %s: %s", k, raw)
		}
	}
	must(t, c.LeaveDumapic())
	if v, _ := c.CampView(); v.Dumapic != nil || v.Screen != "inspect" {
		t.Fatalf("after leaving: %+v", v)
	}
}

// MALOR is refused by CastSpell (finding #25): nothing is spent and no
// screen opens. The terminal's keys still cast it, unchanged.
func TestCastSpellRefusesMalor(t *testing.T) {
	seed(2)
	c := campers(t, rand.New(rand.NewSource(2)))
	mage := c.Game.Town.Party.Members[4]
	mage.SpellKnown[engine.SpellIndex["MALOR"]] = true
	mage.MageSpells[6] = 2
	must(t, c.CampInspect(4))
	before := campTrace(c.Game)
	if err := c.CastSpell("Malor"); !errors.Is(err, ErrMalorUnavailable) || campTrace(c.Game) != before {
		t.Fatalf("CastSpell(MALOR) = %v", err)
	}
	c.press(Rune('S'), Rune('m'), Rune('a'), Rune('l'), Rune('o'), Rune('r'), Key{Kind: KeyEnter})
	if c.Game.Town.InputMode != engine.InputMalor || mage.MageSpells[6] != 1 {
		t.Fatalf("the terminal's MALOR changed: mode %v slots %d", c.Game.Town.InputMode, mage.MageSpells[6])
	}
}

// Use is the engine's stub (finding #30): every item is "** POWERLESS **"
// and nothing changes but the message.
func TestUseItemIsTheEngineStub(t *testing.T) {
	seed(3)
	c := campers(t, rand.New(rand.NewSource(3)))
	must(t, c.CampInspect(0))
	_, items := partyGoods(c.Game)
	roster, _ := json.Marshal(c.Game.Town.Roster.Characters)
	must(t, c.UseItem(1))
	after, _ := json.Marshal(c.Game.Town.Roster.Characters)
	_, items2 := partyGoods(c.Game)
	if c.Game.Town.Message != "** POWERLESS **" || string(after) != string(roster) || !slices.Equal(items, items2) {
		t.Fatalf("use: %q", c.Game.Town.Message)
	}
}

// Disband is the engine's (finding #26): members leave the party still
// marked as in the maze, age 25 weeks, their position is not recorded,
// and the game goes to the castle.
func TestDisbandIsTheEngines(t *testing.T) {
	seed(4)
	c := campers(t, rand.New(rand.NewSource(4)))
	members := slices.Clone(c.Game.Town.Party.Members)
	ages := map[string]int{}
	for _, m := range members {
		ages[m.Name] = m.Age
	}
	must(t, c.Disband())
	g := c.Game
	if g.Phase != engine.PhaseTown || g.Town.Location != engine.Castle || g.Town.Party.Size() != 0 {
		t.Fatalf("after disband: %v %v party %d", g.Phase, g.Town.Location, g.Town.Party.Size())
	}
	for _, m := range members {
		if !m.InMaze || m.Age != ages[m.Name]+25 || m.MazeLevel != 0 || m.MazeX != 0 || m.MazeY != 0 {
			t.Fatalf("%s: inMaze %v age %d position L%d (%d,%d)", m.Name, m.InMaze, m.Age, m.MazeLevel, m.MazeX, m.MazeY)
		}
	}
}

// keysFor are the terminal keys an operation stands for (the inverse of
// camp_ops.go, written out independently).
func keysFor(c *Controller, act string, slot, n, gold int, priest bool, spell string) []Key {
	d := func(i int) Key { return Rune(rune('0' + i)) }
	enter, esc := Key{Kind: KeyEnter}, Key{Kind: KeyEscape}
	switch act {
	case ActCampInspect, ActReorderPick, ActTradeWith, ActCastOn:
		return []Key{d(slot + 1)}
	case ActReorder, ActSpellBooks:
		return []Key{Rune('R')}
	case ActEquipParty, ActEquip:
		return []Key{Rune('E')}
	case ActDisband, ActDrop:
		return []Key{Rune('D')}
	case ActLeaveCamp, ActLeaveInspect, ActCloseBook, ActLeaveDumapic:
		return []Key{Rune('L')}
	case ActEquipChoose, ActDropItem, ActTradeItem:
		return []Key{d(n)}
	case ActEquipSkip:
		return []Key{enter}
	case ActEndDrop:
		return []Key{Rune('0')}
	case ActTrade:
		return []Key{Rune('T')}
	case ActTradeGold:
		var ks []Key
		for range c.Game.Town.InputBuf {
			ks = append(ks, Key{Kind: KeyBackspace})
		}
		if gold > 0 {
			for _, r := range fmt.Sprint(gold) {
				ks = append(ks, Rune(r))
			}
		}
		return append(ks, enter)
	case ActEndTrade:
		if c.Game.Town.InputMode == engine.InputTradeGold {
			return []Key{esc}
		}
		return []Key{enter}
	case ActSpellBook:
		if priest {
			return []Key{Rune('P')}
		}
		return []Key{Rune('M')}
	case ActCast:
		ks := []Key{Rune('S')}
		for _, r := range spell {
			ks = append(ks, Rune(r))
		}
		return append(ks, enter)
	case ActCancelCast:
		return []Key{esc}
	case ActIdentify:
		return []Key{Rune('I')}
	case ActUse:
		return []Key{Rune('U'), d(n)}
	}
	return nil
}

// Each operation is exactly its terminal keys: random operations on one
// game and their keys through HandleKey on another give the same states
// and the same random numbers.
func TestCampOperationsAreTheTerminalKeys(t *testing.T) {
	for s := int64(1); s <= 40; s++ {
		pick := rand.New(rand.NewSource(s))
		seed(s)
		a := campers(t, rand.New(rand.NewSource(s)))
		seed(s)
		b := campers(t, rand.New(rand.NewSource(s)))
		for i := 0; i < 200 && a.Game.Phase == engine.PhaseCamp; i++ {
			v, err := a.CampView()
			must(t, err)
			act := v.Actions[pick.Intn(len(v.Actions))]
			slot, n, gold, priest := pick.Intn(6), 1+pick.Intn(8), pick.Intn(3)*pick.Intn(100), pick.Intn(2) == 0
			spell := []string{"dios", "dialma", "latumofis", "milwa", "lomilwa", "dumapic", "katino", "di"}[pick.Intn(8)]
			if act == ActEquipChoose {
				n = 1 + pick.Intn(max(1, len(a.Game.Town.EquipChoices)))
			}
			keys := keysFor(b, act, slot, n, gold, priest, spell)
			// Both games use the global random source: each side starts the
			// step from the same seed, and must end it at the same draw.
			step := pick.Int63()
			seed(step)
			var err2 error
			switch act {
			case ActCampInspect:
				err2 = a.CampInspect(slot)
			case ActReorder:
				err2 = a.BeginReorder()
			case ActReorderPick:
				err2 = a.ReorderPick(slot)
			case ActEquipParty:
				err2 = a.EquipParty()
			case ActDisband:
				err2 = a.Disband()
			case ActLeaveCamp:
				err2 = a.LeaveCamp()
			case ActLeaveInspect:
				err2 = a.LeaveInspection()
			case ActEquip:
				err2 = a.BeginEquip()
			case ActEquipChoose:
				err2 = a.EquipChoose(n)
			case ActEquipSkip:
				err2 = a.EquipSkip()
			case ActDrop:
				err2 = a.BeginDrop()
			case ActDropItem:
				err2 = a.DropItem(n)
			case ActEndDrop:
				err2 = a.EndDrop()
			case ActTrade:
				err2 = a.BeginTrade()
			case ActTradeWith:
				err2 = a.TradeWith(slot)
			case ActTradeGold:
				err2 = a.TradeGold(gold)
			case ActTradeItem:
				err2 = a.TradeItem(n)
			case ActEndTrade:
				err2 = a.EndTrade()
			case ActSpellBook:
				err2 = a.ShowSpellBook(priest)
			case ActCloseBook:
				err2 = a.CloseSpellBook()
			case ActSpellBooks:
				err2 = a.OpenSpellBooks()
			case ActCast:
				err2 = a.CastSpell(spell)
			case ActCastOn:
				err2 = a.CastOn(slot)
			case ActCancelCast:
				err2 = a.CancelCast()
			case ActIdentify:
				err2 = a.Identify()
			case ActUse:
				err2 = a.UseItem(n)
			case ActLeaveDumapic:
				err2 = a.LeaveDumapic()
			}
			drawA := rand.Int63()
			seed(step)
			if err2 == nil {
				b.press(keys...)
			}
			drawB := rand.Int63()
			if ga, gb := campTrace(a.Game), campTrace(b.Game); ga != gb || drawA != drawB {
				t.Fatalf("seed %d step %d (%s): operation\n%s\nkeys\n%s\nnext draw %d / %d", s, i, act, ga, gb, drawA, drawB)
			}
		}
	}
}

// campSpells is exactly the spells Camp gives an effect: cast through the
// terminal's keys, every other spell answers "NOT IN CAMP" (finding #33),
// and MALOR opens its own screen.
func TestCampSpellsAreApplyCampSpells(t *testing.T) {
	seed(4)
	c := campers(t, rand.New(rand.NewSource(4)))
	bishop := c.Game.Town.Party.Members[3]
	for _, sp := range engine.SpellTable {
		bishop.SpellKnown[engine.SpellIndex[sp.Name]] = true
		bishop.MageSpells[sp.Level-1], bishop.PriestSpells[sp.Level-1] = 3, 3
		c.Game.Fizzles = 0
		must(t, c.CampInspect(3))
		c.press(Rune('S'))
		for _, r := range sp.Name {
			c.press(Rune(r))
		}
		c.press(Key{Kind: KeyEnter})
		if c.mode() == engine.InputSpellTarget {
			c.press(Rune('1'))
		}
		town := c.Game.Town
		switch {
		case sp.Name == "MALOR":
			if c.mode() != engine.InputMalor || !campSpells[sp.Name] {
				t.Fatalf("MALOR: mode %v", c.mode())
			}
			c.press(Key{Kind: KeyEscape})
		case (town.Message == "NOT IN CAMP") == campSpells[sp.Name]:
			t.Fatalf("%s: %q, campSpells %v", sp.Name, town.Message, campSpells[sp.Name])
		}
		if c.mode() == engine.InputDumapic {
			must(t, c.LeaveDumapic())
		}
		if c.Game.Phase != engine.PhaseCamp {
			t.Fatalf("%s left Camp", sp.Name)
		}
		must(t, c.LeaveInspection())
	}
}

// CampView's spells are the game's answers: a spell listed as castable is
// cast (never "** CANT CAST **" or "NOT IN CAMP"), and one that is not
// would be refused or wasted, or is MALOR.
func TestCampSpellViewsMatchTheGame(t *testing.T) {
	seen := map[string]bool{}
	for s := int64(1); s <= 12; s++ {
		seed(s)
		rng := rand.New(rand.NewSource(s))
		c := campers(t, rng)
		slot := rng.Intn(6)
		must(t, c.CampInspect(slot))
		v, _ := c.CampView()
		ch := c.Game.Town.EditChar
		var known []string
		for i, k := range ch.SpellKnown {
			if k && i < len(engine.SpellTable) {
				known = append(known, engine.SpellTable[i].Name)
			}
		}
		if len(v.Spells) != len(known) {
			t.Fatalf("seed %d: %d spells listed, %d known", s, len(v.Spells), len(known))
		}
		for i, sv := range v.Spells {
			if sv.Name != known[i] || sv.Castable == (sv.Reason != "") {
				t.Fatalf("seed %d: %+v", s, sv)
			}
			if sv.Name == "MALOR" {
				continue
			}
			g := cloneCamp(t, c)
			g.Game.Fizzles = 0
			g.press(Rune('S'))
			for _, r := range sv.Name {
				g.press(Rune(r))
			}
			g.press(Key{Kind: KeyEnter})
			if g.mode() == engine.InputSpellTarget {
				if !sv.Target {
					t.Fatalf("seed %d: %s asks for a target", s, sv.Name)
				}
				g.press(Rune('1'))
			} else if sv.Target && sv.Castable {
				t.Fatalf("seed %d: %s did not ask for a target", s, sv.Name)
			}
			msg := g.Game.Town.Message
			refused := msg == "** CANT CAST **" || msg == "NOT IN CAMP"
			if refused == sv.Castable {
				t.Fatalf("seed %d: %s castable %v, the game says %q", s, sv.Name, sv.Castable, msg)
			}
			seen[fmt.Sprintf("%v %v %s", sv.Castable, sv.Target, strings.Fields(sv.Reason+" -")[0])] = true
		}
	}
	for _, k := range []string{"true true -", "true false -", "false false No", "false true No", "false false A"} {
		if !seen[k] {
			t.Errorf("no spell was %q", k)
		}
	}
}

// cloneCamp is a copy of c's game, through the save, at the same Camp
// screen (Checkpoint and Resume).
func cloneCamp(t *testing.T, c *Controller) *Controller {
	t.Helper()
	pristine := newController(t, 1).Game.Scenario
	exp, err := c.Checkpoint(pristine)
	must(t, err)
	c.Game.Expedition = exp
	must(t, c.Save())
	r := newController(t, 1)
	must(t, r.Game.Load())
	must(t, r.Resume(pristine))
	return r
}
