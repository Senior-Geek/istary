// The game draws from the global math/rand source. Go 1.24+ makes rand.Seed a
// no-op by default; this test-only directive restores seeding so these tests
// replay fixed random sequences (as in engine, app and cmd/istary tests).
//go:debug randseednop=0

package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// partySession returns a session whose party is six adventurers, joined
// through the Tavern as the GUI does.
// partySession is a session with a party of six, at the Edge of Town.
func partySession(t *testing.T, weak bool) (*Session, string) {
	return alignedPartySession(t, weak, engine.Neutral)
}

func alignedPartySession(t *testing.T, weak bool, align engine.Alignment) (*Session, string) {
	t.Helper()
	s, dir := newTestSession(t)
	fillParty(t, s, weak, align)
	return s, dir
}

// fillParty puts six adventurers on s's roster and in its party, joined
// through the Tavern as the GUI does, and leaves them at the Edge of Town.
func fillParty(t *testing.T, s *Session, weak bool, align engine.Alignment) {
	t.Helper()
	var chars []*engine.Character
	for i, cls := range []engine.Class{engine.Fighter, engine.Fighter, engine.Thief, engine.Priest, engine.Mage, engine.Fighter} {
		ch := engine.NewCharacter(string(rune('A'+i))+"GUI", engine.Human, cls, align)
		ch.HP, ch.MaxHP, ch.Level, ch.AC, ch.Strength, ch.Agility = 14, 14, 2, 7, 16, 12
		if weak {
			ch.HP, ch.AC = 2, 10
		}
		switch cls {
		case engine.Priest:
			ch.PriestSpells[0], ch.MaxPriestSpells[0] = 2, 2
			ch.PriestSpells[1], ch.MaxPriestSpells[1] = 1, 1
			ch.SpellKnown[engine.SpellIndex["DIOS"]] = true
			ch.SpellKnown[engine.SpellIndex["CALFO"]] = true
		case engine.Mage:
			ch.MageSpells[0], ch.MaxMageSpells[0] = 2, 2
			ch.SpellKnown[engine.SpellIndex["KATINO"]] = true
			ch.SpellKnown[engine.SpellIndex["HALITO"]] = true
		}
		chars = append(chars, ch)
	}
	rosterWith(t, s, chars...)
	walkTo(t, s, "tavern")
	for i := range chars {
		if v := s.JoinParty(i); v.PendingJoin != "" {
			s.SubmitJoinPassword("") // the Tavern asks even for no password
		}
	}
	walkTo(t, s, "edge") // where the party sets out
	if n := len(s.State().Party); n != 6 {
		t.Fatalf("party of %d", n)
	}
}

// drive plays a practice encounter as a front end would: it reads only the
// View, and calls only Session methods. It returns how combat ended and the
// views seen.
func drive(t *testing.T, s *Session, rng *rand.Rand) (string, []View) {
	t.Helper()
	var views []View
	v := s.State()
	for step := 0; step < 3000; step++ {
		if v.Error != "" && !strings.Contains(v.Error, "not available") {
			t.Fatalf("step %d: %s", step, v.Error)
		}
		views = append(views, v)
		if v.Screen != "combat" {
			return v.Notice, views
		}
		c := v.Combat
		switch c.Next {
		case app.NextStart:
			v = s.CombatStart()
		case app.NextFriendly:
			v = s.CombatFriendly(rng.Intn(4) != 0)
		case app.NextOrders:
			v = giveOrder(t, s, c, rng)
		case app.NextConfirm:
			v = s.CombatCommit()
		case app.NextAdvance:
			v = s.CombatAdvance()
		case app.NextChoice:
			v = s.CombatChooseBoon(1 + rng.Intn(3))
		case app.NextEndRound:
			v = s.CombatEndRound()
		case app.NextChest:
			acts := []string{"open", "inspect", "calfo", "disarm", "leave", "inspect", "calfo"}
			slot := 1 + rng.Intn(len(c.Party))
			v = s.ChestAction(acts[rng.Intn(len(acts))], slot)
			if v.Error != "" { // that member cannot act: anyone who can
				for _, m := range c.Chest.Members {
					if m.CanAct {
						v = s.ChestAction("open", m.Slot)
						break
					}
				}
			}
		case app.NextChestResult:
			v = s.ChestContinue()
		case app.NextLeave:
			v = s.CombatLeave()
		default:
			t.Fatalf("unknown next %q", c.Next)
		}
	}
	t.Fatal("the encounter never ended")
	return "", nil
}

// giveOrder gives the next member without an order an order chosen from
// the view's menu, spells and items; or finishes orders.
func giveOrder(t *testing.T, s *Session, c *app.CombatView, rng *rand.Rand) View {
	t.Helper()
	ordered := map[int]bool{}
	for _, o := range c.Orders {
		ordered[o.Slot] = true
	}
	for _, m := range c.Party {
		if !m.CanAct || ordered[m.Slot] {
			continue
		}
		if c.Actor != m.Slot {
			return s.CombatSelect(m.Slot)
		}
		group := c.Groups[rng.Intn(len(c.Groups))].Number
		var tries []OrderRequest
		for _, opt := range c.Menu {
			if !opt.Available {
				continue
			}
			switch opt.Action {
			case "fight", "dispel":
				tries = append(tries, OrderRequest{Slot: m.Slot, Action: opt.Action, Group: group, Item: -1})
			case "spell":
				for _, sp := range c.Spells {
					if !sp.Castable { // listed with its reason, not offered (COMBAT-1)
						continue
					}
					r := OrderRequest{Slot: m.Slot, Action: "spell", Spell: sp.Name, Item: -1}
					switch sp.Target {
					case "group":
						r.Group = group
					case "member":
						r.Ally = 1 + rng.Intn(len(c.Party))
					}
					tries = append(tries, r)
				}
			case "use":
				r := OrderRequest{Slot: m.Slot, Action: "use", Item: -1}
				if len(c.Items) > 0 {
					r.Item = c.Items[0].Index
				}
				tries = append(tries, r)
			case "run":
				if rng.Intn(40) == 0 { // rarely, so most fights are fought out
					tries = append(tries, OrderRequest{Slot: m.Slot, Action: "run", Item: -1})
				}
			default:
				tries = append(tries, OrderRequest{Slot: m.Slot, Action: opt.Action, Item: -1})
			}
		}
		rng.Shuffle(len(tries), func(i, j int) { tries[i], tries[j] = tries[j], tries[i] })
		for _, r := range tries {
			if v := s.CombatOrder(r); v.Error == "" {
				return v
			}
		}
		t.Fatalf("no order accepted for slot %d: %+v", m.Slot, c.Menu)
	}
	return s.CombatFinishOrders()
}

// Practice encounters played to the end through the bridge: every outcome,
// with the front end seeing only the View; the roster on disk is unchanged.
func TestPracticeEncountersThroughTheBridge(t *testing.T) {
	seen := map[string]int{}
	for seed := int64(1); seed <= 150; seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir := alignedPartySession(t, seed%4 == 0, []engine.Alignment{engine.Neutral, engine.Good, engine.Evil}[seed%3])
		before, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
		party := s.State().Party
		v := s.PracticeEncounter(seed%3 == 0)
		if v.Screen != "combat" || v.Combat == nil || (v.Combat.Phase != "opening" && v.Combat.Phase != "friendly") {
			t.Fatalf("seed %d: start: %+v", seed, v)
		}
		if v.Combat.Surprise != "" {
			seen["surprise"]++
		}
		end, views := drive(t, s, rng)
		for _, w := range views {
			if w.Combat == nil {
				continue
			}
			seen["phase "+w.Combat.Phase]++
			for _, e := range w.Combat.Events {
				if e.Silent {
					seen["silent"]++
				}
				for _, f := range e.Effects {
					if f.Who.Side == "party" && f.StatusFrom == "OK" && f.StatusTo != "OK" {
						seen["incapacitated"]++
					}
				}
			}
		}
		switch {
		case strings.Contains(end, "FLED"):
			seen["escape"]++
		case strings.Contains(end, "perished"):
			seen["defeat"]++
		default:
			seen["victory or left"]++
		}
		after, _ := os.ReadFile(filepath.Join(dir, "roster1.json"))
		if !bytes.Equal(before, after) || !reflect.DeepEqual(s.State().Party, party) {
			t.Fatalf("seed %d: the practice encounter changed the roster", seed)
		}
	}
	t.Logf("covered: %v", seen)
	for _, k := range []string{"phase orders", "phase confirm", "phase playback", "phase chest", "phase victory", "phase friendly", "escape", "defeat", "silent", "incapacitated"} {
		if seen[k] == 0 {
			t.Errorf("not covered: %s", k)
		}
	}
}

// The front end receives combat data only as app.CombatView and maze data
// only as app.DungeonView: no engine type is reachable from View, and both
// are plain values.
func TestViewCarriesNoEngineState(t *testing.T) {
	var walk func(reflect.Type, string)
	seen := map[reflect.Type]bool{}
	walk = func(ty reflect.Type, path string) {
		if seen[ty] {
			return
		}
		seen[ty] = true
		if strings.Contains(ty.PkgPath(), "/engine") {
			t.Errorf("%s is engine type %s", path, ty)
		}
		switch ty.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array:
			walk(ty.Elem(), path)
		case reflect.Struct:
			for i := 0; i < ty.NumField(); i++ {
				walk(ty.Field(i).Type, path+"."+ty.Field(i).Name)
			}
		case reflect.Map, reflect.Interface, reflect.Func, reflect.Chan:
			t.Errorf("%s is a %v", path, ty.Kind())
		}
	}
	walk(reflect.TypeOf(View{}), "View")
	if f, _ := reflect.TypeOf(View{}).FieldByName("Combat"); f.Type != reflect.TypeOf(&app.CombatView{}) {
		t.Fatalf("View.Combat is %v", f.Type)
	}
	if f, _ := reflect.TypeOf(View{}).FieldByName("Dungeon"); f.Type != reflect.TypeOf(&app.DungeonView{}) {
		t.Fatalf("View.Dungeon is %v", f.Type)
	}
}

// Hidden facts never reach the front end's JSON: unidentified monsters by
// their true names, the chest's trap before anything said it, monster HP.
func TestPracticeJSONHidesEngineFacts(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		rand.Seed(seed)
		s, _ := partySession(t, false)
		s.PracticeEncounter(true)
		rng := rand.New(rand.NewSource(seed))
		cs := s.practice.Game.Combat
		var hidden []string
		for _, g := range cs.Groups {
			mon := s.scenario.Monsters[g.MonsterID]
			if !g.Identified && mon.Name != mon.NameUnknown {
				hidden = append(hidden, `"`+mon.Name+`"`, `"`+mon.NamePlural+`"`)
			}
		}
		for i := 0; i < 12 && s.practice != nil; i++ {
			v := s.State()
			raw, _ := json.Marshal(v)
			for _, h := range hidden {
				if s.practice.Game.Combat != nil && s.practice.Game.Combat.Phase != engine.CombatExecute && strings.Contains(string(raw), h) {
					stillHidden := true
					for _, g := range s.practice.Game.Combat.Groups {
						if g.Identified {
							stillHidden = false
						}
					}
					if stillHidden {
						t.Fatalf("seed %d: unidentified monster name %s in the view", seed, h)
					}
				}
			}
			for _, k := range []string{`"monsterId"`, `"initiative"`, `"victim"`, `"trapType"`, `"totalGold"`} {
				if strings.Contains(string(raw), k) {
					t.Fatalf("seed %d: %s in the view", seed, k)
				}
			}
			c := v.Combat
			if c == nil {
				break
			}
			switch c.Next {
			case app.NextStart:
				s.CombatStart()
			case app.NextFriendly:
				s.CombatFriendly(true)
			case app.NextOrders:
				giveOrder(t, s, c, rng)
			case app.NextConfirm:
				s.CombatCommit()
			default:
				i = 12
			}
		}
	}
}

// Refusals reach the front end as messages; nothing runs without an
// encounter; abandoning discards it.
func TestPracticeRefusals(t *testing.T) {
	s, _ := newTestSession(t)
	if v := s.PracticeEncounter(false); !strings.Contains(v.Error, "Form a party") || v.Screen == "combat" {
		t.Fatalf("no party: %+v", v)
	}
	if v := s.CombatCommit(); v.Error != errNoPractice.Error() {
		t.Fatalf("no encounter: %q", v.Error)
	}
	s, _ = partySession(t, false)
	rand.Seed(5)
	v := s.PracticeEncounter(false)
	if v2 := s.PracticeEncounter(false); v2.Error == "" {
		t.Fatal("a second encounter started")
	}
	for v.Combat.Next != app.NextOrders {
		switch v.Combat.Next {
		case app.NextStart:
			v = s.CombatStart()
		case app.NextFriendly:
			v = s.CombatFriendly(true)
		default:
			v = s.CombatAdvance()
			if v.Combat.Next == app.NextEndRound {
				v = s.CombatEndRound()
			}
		}
	}
	v = s.CombatSelect(4)
	if v = s.CombatOrder(OrderRequest{Slot: 4, Action: "fight", Group: 1, Item: -1}); !strings.Contains(v.Error, "front row") {
		t.Fatalf("back-row fight: %q", v.Error)
	}
	if v = s.CombatOrder(OrderRequest{Slot: 1, Action: "dance", Item: -1}); v.Error == "" {
		t.Fatal("unknown action accepted")
	}
	if v = s.CombatFinishOrders(); v.Error == "" {
		t.Fatal("orders finished with members unordered")
	}
	if v = s.CombatCommit(); v.Error == "" {
		t.Fatal("committed during order entry")
	}
	v = s.AbandonPractice()
	if v.Screen != "roster" || v.Combat != nil || !strings.Contains(v.Notice, "abandoned") {
		t.Fatalf("abandon: %+v", v)
	}
}
