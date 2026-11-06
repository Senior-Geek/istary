package app

import (
	"encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// atChest returns a controller at the chest menu with the given trap.
func atChest(t *testing.T, trap int) (*Controller, []*engine.Character) {
	t.Helper()
	c, p := combatController(t, 1, 0) // the group is dead: combat won
	cs := c.Game.Combat
	cs.Phase, cs.HasChest, cs.TrapType, cs.ChestSubPhase = engine.CombatChest, true, trap, engine.ChestMenu
	p[3].Class, p[3].Level, p[3].Agility = engine.Thief, 1, 3
	return c, p
}

// finish steps through a result and ends it, as the terminal does.
func finish(t *testing.T, c *Controller) bool {
	t.Helper()
	for i := 0; i < 20; i++ {
		done, err := c.AdvanceChest()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			complete, err := c.EndChestResult()
			if err != nil {
				t.Fatal(err)
			}
			return complete
		}
	}
	t.Fatal("chest result never finished")
	return false
}

func lastLines(c *Controller) string {
	return strings.Join(c.Game.Combat.Messages, "|")
}

// Inspection: a right report, a wrong one, or a fumble that sets the trap
// off; one look each ("YOU ALREADY LOOKED!" after that). Reports appear in
// the View as told, without saying whether they are right.
func TestChestInspect(t *testing.T) {
	trap := 2 // GAS BOMB
	seen := map[string]bool{}
	for s := int64(1); s <= 400 && len(seen) < 3; s++ {
		seed(s)
		c, p := atChest(t, trap)
		if err := c.InspectChest(p[3]); err != nil {
			t.Fatal(err)
		}
		cs := c.Game.Combat
		r := cs.ChestReports
		if len(r) != 1 || r[0].How != "inspect" || r[0].Member != 3 {
			t.Fatalf("seed %d: reports %+v", s, r)
		}
		v := view(t, c)
		if v.Chest.Stage != "result" || v.Next != NextChestResult || len(v.Chest.Reports) != 1 || v.Chest.Reports[0].Said != r[0].Said || !v.Chest.Members[3].Inspected {
			t.Fatalf("seed %d: view %+v", s, v.Chest)
		}
		switch {
		case r[0].SetOff:
			seen["fumble"] = true
			if !strings.HasPrefix(lastLines(c), "OOPPS! A GAS BOMB!") || !cs.ChestOpened || !finish(t, c) {
				t.Fatalf("seed %d: fumble: %q", s, lastLines(c))
			}
		case r[0].Said == engine.TrapNames[trap]:
			seen["right"] = true
		default:
			seen["wrong"] = true
			if lastLines(c) != "IT'S A "+r[0].Said {
				t.Fatalf("seed %d: wrong report %q", s, lastLines(c))
			}
		}
		if r[0].SetOff {
			continue
		}
		if finish(t, c) || cs.Phase != engine.CombatChest || cs.ChestSubPhase != engine.ChestMenu {
			t.Fatalf("seed %d: back to the menu: phase %v", s, cs.Phase)
		}
		c.InspectChest(p[3])
		if lastLines(c) != "YOU ALREADY LOOKED!" || len(cs.ChestReports) != 1 {
			t.Fatalf("seed %d: second look: %q", s, lastLines(c))
		}
	}
	if len(seen) != 3 {
		t.Fatalf("outcomes covered: %v", seen)
	}
}

// CALFO: a caster names the trap (95%) or learns nothing; anyone else gets
// "CAN'T CAST CALFO!" and spends nothing.
func TestChestCalfo(t *testing.T) {
	c, p := atChest(t, 1)
	if err := c.CastCalfo(p[0]); err != nil {
		t.Fatal(err)
	}
	if lastLines(c) != "CAN'T CAST CALFO!" || len(c.Game.Combat.ChestReports) != 0 {
		t.Fatalf("fighter: %q", lastLines(c))
	}
	seen := map[string]bool{}
	for s := int64(1); s <= 400 && len(seen) < 2; s++ {
		seed(s)
		c, p := atChest(t, 1)
		pr := p[4]
		pr.Class = engine.Priest
		pr.PriestSpells[1], pr.MaxPriestSpells[1] = 1, 1
		pr.SpellKnown[engine.SpellIndex["CALFO"]] = true
		if v := view(t, c); !v.Chest.Members[4].CanCastCalfo || v.Chest.Actions[1].Note != "" {
			t.Fatalf("caster not shown: %+v", v.Chest)
		}
		c.CastCalfo(pr)
		r := c.Game.Combat.ChestReports
		if pr.PriestSpells[1] != 0 || len(r) != 1 || r[0].How != "calfo" {
			t.Fatalf("seed %d: points %d reports %+v", s, pr.PriestSpells[1], r)
		}
		if r[0].Said == "" {
			seen["nothing"] = true
			if lastLines(c) != "CALFO REVEALS NOTHING..." {
				t.Fatalf("seed %d: %q", s, lastLines(c))
			}
		} else {
			seen["named"] = true
			if r[0].Said != "POISON NEEDLE" || lastLines(c) != "IT'S A POISON NEEDLE" || !c.Game.Combat.CalfoUsed {
				t.Fatalf("seed %d: %q", s, lastLines(c))
			}
		}
		if finish(t, c) {
			t.Fatal("CALFO completed the chest")
		}
	}
	if len(seen) != 2 {
		t.Fatalf("outcomes covered: %v", seen)
	}
}

// Disarming: disarmed (the chest opens safely), failed (back to the menu),
// or set off (the trap's effect, then the chest is open).
func TestChestDisarm(t *testing.T) {
	seen := map[string]bool{}
	for s := int64(1); s <= 400 && len(seen) < 3; s++ {
		seed(s)
		c, p := atChest(t, 1) // POISON NEEDLE
		th := p[3]
		if s%2 == 0 {
			th.Level = 30 // a good thief, to cover success
		}
		c.DisarmChest(th)
		cs := c.Game.Combat
		switch lines := lastLines(c); {
		case lines == "YOU DISARMED IT!":
			seen["disarmed"] = true
			if !cs.ChestOpened || cs.TrapType != engine.TrapNone || !finish(t, c) || cs.Phase != engine.CombatVictory {
				t.Fatalf("seed %d: disarmed: phase %v", s, cs.Phase)
			}
		case lines == "DISARM FAILED!!":
			seen["failed"] = true
			if cs.ChestOpened || finish(t, c) || cs.Phase != engine.CombatChest {
				t.Fatalf("seed %d: failed: phase %v", s, cs.Phase)
			}
		case strings.HasPrefix(lines, "YOU SET IT OFF!|IT WAS A POISON NEEDLE!"):
			seen["set off"] = true
			r := cs.ChestReports
			if len(r) != 1 || !r[0].SetOff || r[0].How != "disarm" || th.PoisonAmt == 0 || !cs.ChestOpened {
				t.Fatalf("seed %d: set off: %+v poison %d", s, r, th.PoisonAmt)
			}
			if !finish(t, c) {
				t.Fatal("the chest is open after the trap")
			}
		default:
			t.Fatalf("seed %d: %q", s, lines)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("outcomes covered: %v", seen)
	}
}

// Opening: no trap, a trap that goes off (or is dodged); then the rewards.
// Leaving: experience only.
func TestChestOpenAndLeave(t *testing.T) {
	c, p := atChest(t, engine.TrapNone)
	c.OpenChest(p[0])
	if lastLines(c) != "THE CHEST WAS NOT TRAPPED" || !view(t, c).Chest.Complete {
		t.Fatalf("trapless: %q", lastLines(c))
	}
	if !finish(t, c) {
		t.Fatal("opening completes the chest")
	}
	if v := view(t, c); v.Phase != "victory" || v.Chest.Present {
		t.Fatalf("after opening: %+v", v)
	}
	set := false
	for s := int64(1); s <= 50 && !set; s++ {
		seed(s)
		c, p := atChest(t, 1)
		c.OpenChest(p[1])
		if strings.HasPrefix(lastLines(c), "OOPPS! A POISON NEEDLE!") {
			set = true
			if p[1].PoisonAmt == 0 || len(c.Game.Combat.ChestReports) != 1 || !c.Game.Combat.ChestReports[0].SetOff {
				t.Fatalf("trap effect: poison %d", p[1].PoisonAmt)
			}
		}
	}
	if !set {
		t.Fatal("no trap went off")
	}
	c, _ = atChest(t, 1)
	if err := c.LeaveChest(); err != nil || lastLines(c) != "YOU LEAVE THE CHEST." {
		t.Fatalf("leave: %v %q", err, lastLines(c))
	}
	if !finish(t, c) || c.Game.Combat.Phase != engine.CombatVictory || c.Game.Combat.TotalGold != 0 {
		t.Fatalf("after leaving: phase %v gold %d", c.Game.Combat.Phase, c.Game.Combat.TotalGold)
	}
	if v := view(t, c); v.Gold != 0 || v.XP != c.Game.Combat.TotalXP {
		t.Fatalf("victory view: xp %d gold %d", v.XP, v.Gold)
	}
}

// Who may act, and in which phase.
func TestChestInvalid(t *testing.T) {
	c, p := combatController(t, 1) // order entry, no chest
	for name, op := range map[string]func() error{
		"open": func() error { return c.OpenChest(p[0]) }, "inspect": func() error { return c.InspectChest(p[0]) },
		"calfo": func() error { return c.CastCalfo(p[0]) }, "disarm": func() error { return c.DisarmChest(p[0]) },
		"leave": c.LeaveChest,
	} {
		if err := op(); !errors.Is(err, ErrNotAtChest) {
			t.Errorf("%s outside the chest: %v", name, err)
		}
	}
	if _, err := c.AdvanceChest(); !errors.Is(err, ErrNoChestResult) {
		t.Errorf("AdvanceChest: %v", err)
	}
	if _, err := c.EndChestResult(); !errors.Is(err, ErrNoChestResult) {
		t.Errorf("EndChestResult: %v", err)
	}
	c, p = atChest(t, 1)
	p[0].Status = engine.Dead
	if err := c.OpenChest(p[0]); !errors.Is(err, ErrCannotAct) || c.Game.Combat.Phase != engine.CombatChest {
		t.Fatalf("dead opener: %v", err)
	}
	stranger := engine.NewCharacter("Z", engine.Human, engine.Thief, engine.Neutral)
	if err := c.DisarmChest(stranger); !errors.Is(err, ErrNoSuchPartyMember) {
		t.Fatalf("stranger: %v", err)
	}
	// A result with a further block cannot be ended before it is shown.
	c.InspectChest(p[3])
	cs := c.Game.Combat
	cs.Messages = append(cs.Messages, engine.ActionSeparator, "MORE")
	if _, err := c.EndChestResult(); !errors.Is(err, ErrChestResultOpen) {
		t.Fatalf("ending early: %v", err)
	}
	if done, _ := c.AdvanceChest(); done || cs.MessageIndex != len(cs.Messages)-1 {
		t.Fatal("AdvanceChest did not show the next block")
	}
	if v := view(t, c); strings.Join(v.Lines, "|") != "MORE" {
		t.Fatalf("view lines %v", v.Lines)
	}
}

// The controller draws exactly the random numbers the engine call does,
// and writes the same messages.
func TestChestSameRandomness(t *testing.T) {
	type op func(*Controller, *engine.Character) error
	type eng func(*engine.CombatState, *engine.GameState, int)
	pairs := []struct {
		ctrl op
		eng  eng
	}{
		{(*Controller).OpenChest, (*engine.CombatState).OpenChest},
		{(*Controller).InspectChest, (*engine.CombatState).InspectChest},
		{(*Controller).CastCalfo, (*engine.CombatState).CalfoChest},
		{(*Controller).DisarmChest, (*engine.CombatState).DisarmChest},
	}
	for i, pr := range pairs {
		for s := int64(1); s <= 20; s++ {
			seed(s)
			c1, p1 := atChest(t, 1)
			pr.ctrl(c1, p1[3])
			a, an := strings.Join(c1.Game.Combat.Messages, "|"), rand.Int63()
			seed(s)
			c2, p2 := atChest(t, 1)
			pr.eng(c2.Game.Combat, c2.Game, 3)
			b, bn := strings.Join(c2.Game.Combat.Messages, "|"), rand.Int63()
			if a != b || an != bn || p1[3].PoisonAmt != p2[3].PoisonAmt {
				t.Fatalf("operation %d seed %d: %q vs %q", i, s, a, b)
			}
		}
	}
}

// The boundary: the View never holds the true trap, a correctness verdict
// or the treasure. A trap name appears only if a report or a shown message
// said it; gold only at victory. The View's types have no trap or gold
// fields other than the revealed rewards.
func TestChestViewBoundary(t *testing.T) {
	for s := int64(1); s <= 150; s++ {
		seed(s)
		trap := 1 + int(s)%7
		c, p := atChest(t, trap)
		cs := c.Game.Combat
		cs.TotalGold = 9999 // not yet revealed
		check := func(stage string) {
			v := view(t, c)
			raw, _ := json.Marshal(v)
			told := strings.Join(v.Lines, "|")
			for _, r := range v.Chest.Reports {
				told += "|" + r.Said
			}
			for _, name := range engine.TrapNames {
				if strings.Contains(string(raw), name) && !strings.Contains(told, name) {
					t.Fatalf("seed %d %s: %q in the view without being said", s, stage, name)
				}
			}
			if v.Phase != "victory" && strings.Contains(string(raw), "9999") {
				t.Fatalf("seed %d %s: the treasure is in the view", s, stage)
			}
		}
		check("menu")
		for _, act := range []func() error{
			func() error { return c.InspectChest(p[int(s)%6]) },
			func() error { return c.CastCalfo(p[(int(s)+1)%6]) },
			func() error { return c.InspectChest(p[3]) },
		} {
			if cs.Phase != engine.CombatChest {
				break
			}
			act()
			check("result")
			if !finish(t, c) {
				check("menu again")
			}
		}
	}
	var walk func(reflect.Type)
	walk = func(ty reflect.Type) {
		switch ty.Kind() {
		case reflect.Struct:
			for i := 0; i < ty.NumField(); i++ {
				f := ty.Field(i)
				low := strings.ToLower(f.Name)
				if strings.Contains(low, "trap") || strings.Contains(low, "correct") || (strings.Contains(low, "gold") && ty != reflect.TypeOf(CombatView{})) {
					t.Errorf("%s.%s", ty.Name(), f.Name)
				}
				walk(f.Type)
			}
		case reflect.Slice:
			walk(ty.Elem())
		}
	}
	walk(reflect.TypeOf(CombatView{}))
}
