package app

import (
	"errors"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// SelectCombatActor: any member who can be given an order, during order
// entry only; the view's menu then follows the chosen member.
func TestCombatSelectActor(t *testing.T) {
	c, p := combatController(t, 1)
	if err := c.SelectCombatActor(p[4]); err != nil || c.Game.Combat.CurrentActor != 4 {
		t.Fatalf("select E: %v", err)
	}
	if v := view(t, c); v.Actor != 5 || v.Menu[0].Available {
		t.Fatalf("menu for E (back row): %+v", v.Menu)
	}
	p[2].Status = engine.Paralyzed
	if err := c.SelectCombatActor(p[2]); !errors.Is(err, ErrCannotAct) {
		t.Fatalf("paralyzed: %v", err)
	}
	if err := c.SelectCombatActor(engine.NewCharacter("Z", engine.Human, engine.Fighter, engine.Neutral)); !errors.Is(err, ErrNoSuchPartyMember) {
		t.Fatalf("stranger: %v", err)
	}
	c.Game.Combat.Phase = engine.CombatConfirm
	if err := c.SelectCombatActor(p[0]); !errors.Is(err, ErrNotInCombat) {
		t.Fatalf("at confirmation: %v", err)
	}
}

// The actor's castable spells (known, with points) with their targets, and
// usable items by the name the player knows.
func TestCombatViewSpellsAndItems(t *testing.T) {
	c, p := combatController(t, 1)
	pr := p[4]
	pr.Class = engine.Priest
	pr.PriestSpells[0], pr.MaxPriestSpells[0] = 1, 1
	pr.SpellKnown[engine.SpellIndex["DIOS"]] = true
	pr.SpellKnown[engine.SpellIndex["BADIOS"]] = true
	pr.SpellKnown[engine.SpellIndex["DIALKO"]] = true // level 3: no points
	pr.Items[0] = engine.Possession{ItemIndex: itemNamed(t, c, "SCROLL/DILTO"), Equipped: true}
	pr.Items[1] = engine.Possession{ItemIndex: itemNamed(t, c, "DIOS POTION"), Equipped: true, Identified: true}
	pr.ItemCount = 2
	c.SelectCombatActor(pr)
	v := view(t, c)
	if len(v.Spells) != 2 || v.Spells[0] != (CombatSpellView{Name: "DIOS", Level: 1, Kind: "priest", Target: "member"}) ||
		v.Spells[1] != (CombatSpellView{Name: "BADIOS", Level: 1, Kind: "priest", Target: "group"}) {
		t.Fatalf("spells: %+v", v.Spells)
	}
	items := c.Game.Scenario.Items
	if len(v.Items) != 2 || v.Items[0].Name != items[pr.Items[0].ItemIndex].NameUnknown || v.Items[1].Name != "DIOS POTION" {
		t.Fatalf("items: %+v", v.Items)
	}
	c.SelectCombatActor(p[0])
	if v := view(t, c); v.Spells != nil || v.Items != nil {
		t.Fatalf("fighter: %+v %+v", v.Spells, v.Items)
	}
}
