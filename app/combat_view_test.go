package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// A View holds only values: no pointer, map, interface, func or channel
// anywhere in its type, so nothing in it can reach into the engine.
func TestCombatViewHoldsOnlyValues(t *testing.T) {
	var walk func(reflect.Type, string)
	walk = func(ty reflect.Type, path string) {
		switch ty.Kind() {
		case reflect.Struct:
			for i := 0; i < ty.NumField(); i++ {
				walk(ty.Field(i).Type, path+"."+ty.Field(i).Name)
			}
		case reflect.Slice:
			walk(ty.Elem(), path+"[]")
		case reflect.String, reflect.Int, reflect.Bool:
		default:
			t.Errorf("%s is a %v", path, ty.Kind())
		}
	}
	walk(reflect.TypeOf(CombatView{}), "CombatView")
}

// expedition puts six adventurers in the maze and opens an encounter.
func expedition(t *testing.T, weak bool) *Controller {
	t.Helper()
	c := newController(t, 1)
	g := c.Game
	for i, cls := range []engine.Class{engine.Fighter, engine.Fighter, engine.Thief, engine.Priest, engine.Mage, engine.Fighter} {
		ch := engine.NewCharacter(string(rune('A'+i)), engine.Human, cls, engine.Neutral)
		ch.HP, ch.MaxHP, ch.Level, ch.AC, ch.Strength = 14, 14, 2, 7, 16
		if weak {
			ch.HP, ch.AC = 2, 10
		}
		switch cls {
		case engine.Priest:
			ch.PriestSpells[0], ch.MaxPriestSpells[0] = 2, 2
			ch.SpellKnown[engine.SpellIndex["DIOS"]] = true
		case engine.Mage:
			ch.MageSpells[0], ch.MaxMageSpells[0] = 2, 2
			ch.SpellKnown[engine.SpellIndex["KATINO"]] = true
		}
		g.Town.Party.Members = append(g.Town.Party.Members, ch)
	}
	g.Combat = engine.NewCombat(g)
	g.Phase = engine.PhaseCombat
	return c
}

// giveOrders gives every member who can act an order (varied by seed and
// round) and moves to confirmation, as the terminal's navigation does.
func giveOrders(t *testing.T, c *Controller, s int64, round int) {
	t.Helper()
	cs := c.Game.Combat
	for i, m := range c.Game.Town.Party.Members {
		if !CanGiveOrder(m) {
			continue
		}
		g := cs.FirstAliveGroup()
		tries := []Order{{Action: engine.ActionParry}}
		switch (int(s) + round + i) % 5 {
		case 0, 1:
			tries = append([]Order{{Action: engine.ActionFight, Group: g}}, tries...)
		case 2:
			if s%3 == 0 {
				tries = append([]Order{{Action: engine.ActionRun}}, tries...)
			}
		case 3:
			tries = append([]Order{{Action: engine.ActionSpell, Spell: "KATINO", Group: g}, {Action: engine.ActionSpell, Spell: "DIOS", Ally: m}}, tries...)
		case 4:
			tries = append([]Order{{Action: engine.ActionUse, Item: -1}}, tries...)
		}
		for _, o := range tries {
			if c.SetOrder(m, o) == nil {
				break
			}
		}
	}
	cs.Phase = engine.CombatConfirm
}

func view(t *testing.T, c *Controller) CombatView {
	t.Helper()
	v, err := c.CombatView()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// The boundary, inspected at every phase and after every Advance over many
// seeded encounters: during playback only released events appear; the
// party, monster counts and names are the round-start display and do not
// change until EndRound; nothing else in the View changes during playback,
// so the round's outcome cannot show early; formation changes appear only
// after EndRound.
func TestCombatViewBoundaryEveryStep(t *testing.T) {
	seen := map[string]int{}
	for s := int64(1); s <= 120; s++ {
		seed(s)
		c := expedition(t, s%5 == 0)
		g := c.Game
		if g.Combat.Surprised == 2 {
			seen["surprise"]++
		}
		if v := view(t, c); v.Phase != "opening" || v.Next != NextStart || len(v.Events) != 0 {
			t.Fatalf("seed %d opening: %+v", s, v)
		}
		playback, err := c.StartCombat()
		if err != nil {
			t.Fatal(err)
		}
		for round := 0; round < 12 && g.Phase == engine.PhaseCombat && g.Combat != nil; round++ {
			cs := g.Combat
			var before CombatView
			if !playback {
				if cs.Phase != engine.CombatChoose {
					break
				}
				if v := view(t, c); v.Next != NextOrders || v.Actor == 0 || len(v.Menu) != 6 {
					t.Fatalf("seed %d orders view: next %s actor %d menu %d", s, v.Next, v.Actor, len(v.Menu))
				}
				giveOrders(t, c, s, round)
				before = view(t, c)
				if before.Next != NextConfirm || len(before.Orders) == 0 || len(before.Events) != 0 {
					t.Fatalf("seed %d confirm view: %+v", s, before)
				}
				if _, err := c.Commit(); err != nil {
					t.Fatal(err)
				}
			}
			playback = false

			first := view(t, c)
			if first.Phase != "playback" || !first.Frozen {
				t.Fatalf("seed %d: playback view %+v", s, first)
			}
			if before.Party != nil && !reflect.DeepEqual(first.Party, before.Party) {
				t.Fatalf("seed %d: the party changed at Commit:\n%+v\n%+v", s, before.Party, first.Party)
			}
			if before.Groups != nil {
				for i, gv := range first.Groups {
					if i >= len(before.Groups) || gv.Count != before.Groups[i].Count || gv.Number != before.Groups[i].Number {
						t.Fatalf("seed %d: group counts changed at Commit", s)
					}
				}
			}
			still := func(v CombatView) CombatView {
				v.Events, v.Current, v.Lines, v.Next = nil, 0, nil, ""
				return v
			}
			for step := 0; ; step++ {
				v := view(t, c)
				rev := c.RevealedEvents()
				if len(v.Events) != len(rev) {
					t.Fatalf("seed %d step %d: %d events in the view, %d released", s, step, len(v.Events), len(rev))
				}
				for i := range rev {
					if !reflect.DeepEqual(v.Events[i], eventView(rev[i])) {
						t.Fatalf("seed %d step %d: event %d differs from the released one", s, step, i)
					}
					if rev[i].Silent {
						seen["silent"]++
					}
					for _, f := range rev[i].Effects {
						if f.Who.Side == engine.SideParty && f.StatusTo != "OK" && f.StatusFrom == "OK" {
							seen["incapacitated"]++
						}
					}
				}
				if len(rev) > 0 && len(rev) < len(g.Combat.Events()) && v.Current != len(rev)-1 {
					t.Fatalf("seed %d step %d: current %d", s, step, v.Current)
				}
				if !reflect.DeepEqual(still(v), still(first)) {
					t.Fatalf("seed %d step %d: the view changed during playback beyond its events:\n%+v\n%+v", s, step, still(first), still(v))
				}
				if v.Next == NextChoice {
					t.Skip("HAMAN is not reachable with this party")
				}
				st, err := c.Advance()
				if err != nil {
					t.Fatal(err)
				}
				if st.Done {
					break
				}
				if v.Next != NextAdvance {
					t.Fatalf("seed %d: next %s with blocks unshown", s, v.Next)
				}
			}
			end := view(t, c)
			if end.Next != NextEndRound || end.Current != -1 || end.Lines != nil || !reflect.DeepEqual(still(end), still(first)) {
				t.Fatalf("seed %d: view at the end of playback: %+v", s, end)
			}
			order := names(first.Party)
			out, err := c.EndRound()
			if err != nil {
				t.Fatal(err)
			}
			switch out {
			case RoundContinues:
				seen["continues"]++
				v := view(t, c)
				if v.Phase != "orders" || v.Frozen || v.Round != first.Round+1 {
					t.Fatalf("seed %d: next round view %+v", s, v)
				}
				if names(v.Party) != order {
					seen["formation change"]++
				}
			case RoundEscaped:
				seen["escaped"]++
				if _, err := c.CombatView(); !errors.Is(err, ErrNotInCombat) {
					t.Fatalf("seed %d: view after escape: %v", s, err)
				}
			case RoundVictory:
				seen["victory"]++
				if v := view(t, c); v.Phase != "victory" && v.Phase != "chest" {
					t.Fatalf("seed %d: victory view %s", s, v.Phase)
				}
			case RoundDefeated:
				seen["defeat"]++
				if v := view(t, c); v.Phase != "defeat" || v.Next != NextLeave {
					t.Fatalf("seed %d: defeat view %+v", s, v)
				}
			}
		}
	}
	t.Logf("covered: %v", seen)
	for _, k := range []string{"surprise", "silent", "incapacitated", "continues", "escaped", "victory", "defeat", "formation change"} {
		if seen[k] == 0 {
			t.Errorf("not covered: %s", k)
		}
	}
}

func names(p []CombatMemberView) string {
	var out []string
	for _, m := range p {
		out = append(out, m.Name)
	}
	return strings.Join(out, " ")
}

// Order entry: the actor, its menu as the controller gives it, and the
// orders given so far, with items by the name the player knows.
func TestCombatViewOrderEntry(t *testing.T) {
	c, p := combatController(t, 2)
	priest, user := p[4], p[1]
	priest.Class = engine.Priest
	priest.PriestSpells[0], priest.MaxPriestSpells[0] = 1, 1
	priest.SpellKnown[engine.SpellIndex["DIOS"]] = true
	scroll := itemNamed(t, c, "SCROLL/DILTO")
	user.Items[0] = engine.Possession{ItemIndex: scroll, Equipped: true}
	user.ItemCount = 1
	c.Game.Combat.CurrentActor = 3
	v := view(t, c)
	menu, _ := c.CombatOrderMenu(p[3])
	if v.Actor != 4 || len(v.Menu) != len(menu) || v.Menu[0].Action != "fight" || v.Menu[0].Available {
		t.Fatalf("menu: actor %d %+v", v.Actor, v.Menu)
	}
	for _, o := range []struct {
		ch *engine.Character
		o  Order
	}{
		{p[0], Order{Action: engine.ActionFight, Group: 1}},
		{priest, Order{Action: engine.ActionSpell, Spell: "DIOS", Ally: p[2]}},
		{user, Order{Action: engine.ActionUse, Item: 0}},
	} {
		if err := c.SetOrder(o.ch, o.o); err != nil {
			t.Fatal(err)
		}
	}
	want := []CombatOrderView{
		{Slot: 1, Member: "A", Action: "fight", Group: 2},
		{Slot: 2, Member: "B", Action: "use", Item: c.Game.Scenario.Items[scroll].NameUnknown},
		{Slot: 5, Member: "E", Action: "spell", Spell: "DIOS", Ally: "C"},
	}
	if got := view(t, c).Orders; !reflect.DeepEqual(got, want) {
		t.Fatalf("orders:\n%+v\nwant\n%+v", got, want)
	}
	c.Game.Combat.Phase = engine.CombatConfirm
	if v := view(t, c); v.Next != NextConfirm || v.Menu != nil || len(v.Orders) != 3 {
		t.Fatalf("confirm view: %+v", v)
	}
}

// A LATUMAPIC cast mid-round identifies the groups in the engine at once,
// but the View keeps the round-start names until the round ends.
func TestCombatViewFreezesIdentification(t *testing.T) {
	for s := int64(1); s <= 200; s++ {
		seed(s)
		c := expedition(t, false)
		g := c.Game
		cs := g.Combat
		if cs.Surprised != 0 {
			continue
		}
		for _, m := range g.Town.Party.Members { // no identification roll
			m.IQ, m.Piety, m.Level = 0, 0, 0
		}
		priest := g.Town.Party.Members[3]
		priest.PriestSpells[2], priest.MaxPriestSpells[2] = 1, 1
		priest.SpellKnown[engine.SpellIndex["LATUMAPIC"]] = true
		c.StartCombat()
		for i, m := range g.Town.Party.Members {
			c.SetOrder(m, Order{Action: engine.ActionParry})
			if i == 3 {
				if err := c.SetOrder(m, Order{Action: engine.ActionSpell, Spell: "LATUMAPIC"}); err != nil {
					t.Fatal(err)
				}
			}
		}
		cs.Phase = engine.CombatConfirm
		before := view(t, c).Groups
		c.Commit()
		if cs.DisplayIdentified == nil || cs.DisplayIdentified[0] || !cs.Groups[0].Identified {
			continue // identified before the round, or the spell did not land
		}
		for {
			if v := view(t, c); !reflect.DeepEqual(v.Groups, before) {
				t.Fatalf("seed %d: names changed during playback: %+v -> %+v", s, before, v.Groups)
			}
			if st, _ := c.Advance(); st.Done {
				break
			}
		}
		if out, _ := c.EndRound(); out == RoundContinues {
			if after := view(t, c).Groups; reflect.DeepEqual(after, before) {
				t.Fatalf("seed %d: the identified names did not appear after the round", s)
			}
		}
		return
	}
	t.Fatal("no seed with a mid-round LATUMAPIC")
}

// Phases outside the round, and no combat.
func TestCombatViewOtherPhases(t *testing.T) {
	c, _ := combatController(t, 1)
	cs := c.Game.Combat
	for ph, want := range map[engine.CombatPhase][2]string{
		engine.CombatInit:        {"opening", NextStart},
		engine.CombatFriendly:    {"friendly", NextFriendly},
		engine.CombatChest:       {"chest", NextChest},
		engine.CombatChestResult: {"chest", NextChest},
		engine.CombatVictory:     {"victory", NextLeave},
		engine.CombatDefeat:      {"defeat", NextLeave},
	} {
		cs.Phase = ph
		if v := view(t, c); v.Phase != want[0] || v.Next != want[1] || v.Frozen || len(v.Events) != 0 || v.Menu != nil || v.Orders != nil {
			t.Errorf("phase %v: %+v", ph, v)
		}
	}
	c.Game.Combat = nil
	if _, err := c.CombatView(); !errors.Is(err, ErrNotInCombat) {
		t.Fatalf("no combat: %v", err)
	}
}
