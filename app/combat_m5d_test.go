package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// FinishOrders: every member who can act needs an order; members who
// cannot act need none; only during order entry.
func TestCombatFinishOrders(t *testing.T) {
	c, p := combatController(t, 1)
	p[5].Status = engine.Dead
	for i := 0; i < 4; i++ {
		if err := c.SetOrder(p[i], Order{Action: engine.ActionParry}); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.FinishOrders(); !errors.Is(err, ErrOrdersIncomplete) {
		t.Fatalf("with E unordered: %v", err)
	}
	if c.Game.Combat.Phase != engine.CombatChoose {
		t.Fatal("phase changed on a refused FinishOrders")
	}
	c.SetOrder(p[4], Order{Action: engine.ActionRun})
	if err := c.FinishOrders(); err != nil || c.Game.Combat.Phase != engine.CombatConfirm {
		t.Fatalf("complete orders: %v", err)
	}
	if v := view(t, c); v.Next != NextConfirm || len(v.Orders) != 5 {
		t.Fatalf("view: %+v", v)
	}
	if err := c.FinishOrders(); !errors.Is(err, ErrNotInCombat) {
		t.Fatalf("outside order entry: %v", err)
	}
	// Nobody can act: nothing to order.
	c2, p2 := combatController(t, 1)
	for _, m := range p2 {
		m.Status = engine.Paralyzed
	}
	if err := c2.FinishOrders(); err != nil {
		t.Fatalf("no one can act: %v", err)
	}
}

// Friendly encounters: fight (order entry from the first member who can
// act) or leave in peace (back to the maze); nothing else in that phase.
func TestCombatFriendlyEncounter(t *testing.T) {
	c, p := combatController(t, 1)
	c.Game.Combat.Phase, c.Game.Combat.Friendly = engine.CombatFriendly, true
	if v := view(t, c); v.Phase != "friendly" || v.Next != NextFriendly || len(v.Groups) != 1 || v.Menu != nil {
		t.Fatalf("friendly view: %+v", v)
	}
	if _, err := c.Commit(); !errors.Is(err, ErrNotConfirming) {
		t.Fatalf("Commit in a friendly encounter: %v", err)
	}
	p[0].Status = engine.Dead
	if err := c.FightFriendly(); err != nil {
		t.Fatal(err)
	}
	cs := c.Game.Combat
	if cs.Friendly || cs.Phase != engine.CombatChoose || cs.CurrentActor != 1 {
		t.Fatalf("fight: friendly %v phase %v actor %d", cs.Friendly, cs.Phase, cs.CurrentActor)
	}
	if err := c.FightFriendly(); !errors.Is(err, ErrNotFriendly) {
		t.Fatalf("second FightFriendly: %v", err)
	}
	if err := c.LeaveInPeace(); !errors.Is(err, ErrNotFriendly) {
		t.Fatalf("LeaveInPeace during order entry: %v", err)
	}
	c2, _ := combatController(t, 1)
	c2.Game.Combat.Phase = engine.CombatFriendly
	if err := c2.LeaveInPeace(); err != nil || c2.Game.Combat != nil || c2.Game.Phase != engine.PhaseMaze {
		t.Fatalf("leave in peace: %v phase %v", err, c2.Game.Phase)
	}
	if _, err := c2.CombatView(); !errors.Is(err, ErrNotInCombat) {
		t.Fatalf("view after leaving: %v", err)
	}
}

// Leaving combat after victory (the maze) or defeat (the castle); refused
// in any other phase.
func TestCombatLeaveCombat(t *testing.T) {
	c, _ := combatController(t, 1)
	if err := c.LeaveCombat(); !errors.Is(err, ErrNotEnding) {
		t.Fatalf("during order entry: %v", err)
	}
	c.Game.Combat.Phase = engine.CombatVictory
	c.Game.MazeMessage, c.Game.MazeMessage2 = "X", "Y"
	if v := view(t, c); v.Next != NextLeave {
		t.Fatalf("victory view: %+v", v)
	}
	if err := c.LeaveCombat(); err != nil || c.Game.Combat != nil || c.Game.Phase != engine.PhaseMaze || c.Game.MazeMessage != "" || c.Game.MazeMessage2 != "" {
		t.Fatalf("victory: %v phase %v", err, c.Game.Phase)
	}
	if err := c.LeaveCombat(); !errors.Is(err, ErrNotEnding) {
		t.Fatalf("after leaving: %v", err)
	}
	c2, _ := combatController(t, 1)
	c2.Game.Combat.Phase = engine.CombatDefeat
	// A defeat with someone still able to act is refused, changing nothing
	// (R-1: only a party that cannot go on is stranded).
	if err := c2.LeaveCombat(); err == nil || c2.Game.Phase != engine.PhaseCombat || c2.Game.Town.Party.Size() == 0 {
		t.Fatalf("defeat with the party standing: %v phase %v", err, c2.Game.Phase)
	}
	party := append([]*engine.Character(nil), c2.Game.Town.Party.Members...)
	for _, m := range party {
		m.Status = engine.Dead
	}
	here, _ := c2.Game.Place()
	if err := c2.LeaveCombat(); err != nil || c2.Game.Phase != engine.PhaseTown || c2.Game.Town.Location != engine.Castle ||
		c2.Game.Town.Message != "YOUR PARTY HAS PERISHED IN THE MAZE..." || c2.Game.Combat != nil || c2.Game.Town.Party.Size() != 0 {
		t.Fatalf("defeat: %v phase %v %q", err, c2.Game.Phase, c2.Game.Town.Message)
	}
	for _, m := range party {
		if m.Stranded == nil || *m.Stranded != here || !m.InMaze {
			t.Fatalf("%s after defeat: stranded %v in maze %v, want %v", m.Name, m.Stranded, m.InMaze, here)
		}
	}
}

// HAMAN (Wiz 2): playback runs up to the block that offers the boons and
// waits there. Before that block, the View shows no boons and no choice; at
// it, the boons as the prompt lists them. After the choice playback goes on
// and the round ends normally.
func TestCombatHamanChoice(t *testing.T) {
	found := map[string]bool{}
	for s := int64(1); s <= 300 && !(found["later"] && found["first"]); s++ {
		seed(s)
		c := newController(t, 2)
		g := c.Game
		for i, cls := range []engine.Class{engine.Fighter, engine.Fighter, engine.Fighter, engine.Thief, engine.Mage, engine.Priest} {
			ch := engine.NewCharacter(string(rune('A'+i)), engine.Human, cls, engine.Neutral)
			ch.HP, ch.MaxHP, ch.Level, ch.AC, ch.Agility = 60, 60, 2, 2, 10
			if cls == engine.Mage {
				ch.Level, ch.IQ = 13, 18
				ch.MageSpells[5], ch.MaxMageSpells[5] = 1, 1
				ch.SpellKnown[engine.SpellIndex["HAMAN"]] = true
			}
			g.Town.Party.Members = append(g.Town.Party.Members, ch)
		}
		g.Combat = engine.NewCombat(g)
		g.Phase = engine.PhaseCombat
		if len(g.Combat.Groups) == 0 || g.Combat.Surprised != 0 || g.Combat.Phase != engine.CombatInit {
			continue
		}
		c.StartCombat()
		for i, m := range g.Town.Party.Members {
			o := Order{Action: engine.ActionParry}
			if i == 4 {
				o = Order{Action: engine.ActionSpell, Spell: "HAMAN"}
			}
			if err := c.SetOrder(m, o); err != nil {
				t.Fatalf("seed %d: %v", s, err)
			}
		}
		if err := c.FinishOrders(); err != nil {
			t.Fatal(err)
		}
		c.Commit()
		cs := g.Combat
		if !cs.HamanSelecting {
			continue
		}
		steps := 0
		for {
			v := view(t, c)
			if v.Next == NextChoice {
				break
			}
			if v.Boons != nil {
				t.Fatalf("seed %d: boons before their block: %+v", s, v)
			}
			if _, err := c.Advance(); err != nil {
				t.Fatalf("seed %d: Advance before the prompt: %v", s, err)
			}
			steps++
		}
		v := view(t, c)
		cur := v.Events[v.Current]
		if !strings.Contains(strings.Join(cur.Lines, "|"), "WHICH BOON WILL YOU INVOKE ?") || len(v.Boons) != 3 {
			t.Fatalf("seed %d: at the prompt: %+v", s, v)
		}
		for i, b := range v.Boons {
			if !strings.Contains(strings.Join(cur.Lines, "|"), fmt.Sprintf("%d) %s", i+1, b)) {
				t.Fatalf("seed %d: boon %q is not what the prompt lists", s, b)
			}
		}
		if len(v.Events) != v.Current+1 {
			t.Fatalf("seed %d: events beyond the prompt released", s)
		}
		if _, err := c.Advance(); !errors.Is(err, ErrAwaitingChoice) {
			t.Fatalf("seed %d: Advance at the prompt: %v", s, err)
		}
		if _, err := c.EndRound(); !errors.Is(err, ErrAwaitingChoice) {
			t.Fatalf("seed %d: EndRound at the prompt: %v", s, err)
		}
		if err := c.ChooseBoon(0); err != nil {
			t.Fatal(err)
		}
		if v := view(t, c); v.Boons != nil || v.Next == NextChoice {
			t.Fatalf("seed %d: after the choice: %+v", s, v)
		}
		for i := 0; i < 100; i++ {
			st, err := c.Advance()
			if err != nil {
				t.Fatal(err)
			}
			if st.Done {
				break
			}
		}
		if _, err := c.EndRound(); err != nil {
			t.Fatalf("seed %d: EndRound after the choice: %v", s, err)
		}
		if steps == 0 {
			found["first"] = true
		} else {
			found["later"] = true
		}
	}
	if !found["later"] {
		t.Fatal("no seed where HAMAN's block comes after another block")
	}
	t.Logf("HAMAN cases covered: %v", found)
}
