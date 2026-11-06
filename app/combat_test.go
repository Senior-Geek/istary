package app

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// combatController returns a controller in combat order entry with six
// fighters and the given number of monster groups, each with two living
// members unless dead is set for it.
func combatController(t *testing.T, groups int, dead ...int) (*Controller, []*engine.Character) {
	t.Helper()
	c := newController(t, 1)
	g := c.Game
	var party []*engine.Character
	for i := 0; i < 6; i++ {
		ch := engine.NewCharacter(string(rune('A'+i)), engine.Human, engine.Fighter, engine.Neutral)
		ch.HP, ch.MaxHP, ch.Level = 10, 10, 1
		party = append(party, ch)
	}
	g.Town.Party.Members = party
	cs := &engine.CombatState{Phase: engine.CombatChoose, Round: 1, Actions: make([]engine.PartyAction, 6)}
	for gi := 0; gi < groups; gi++ {
		grp := &engine.MonsterGroup{AliveCnt: 2, Members: []*engine.CombatMonster{{HP: 5}, {HP: 5}}}
		for _, d := range dead {
			if d == gi {
				grp.Members[0].Status, grp.Members[1].Status = 5, 5
			}
		}
		cs.Groups = append(cs.Groups, grp)
	}
	g.Combat, g.Phase = cs, engine.PhaseCombat
	return c, party
}

func option(t *testing.T, c *Controller, ch *engine.Character, a engine.CombatAction) OrderOption {
	t.Helper()
	menu, err := c.CombatOrderMenu(ch)
	if err != nil {
		t.Fatalf("menu for %s: %v", ch.Name, err)
	}
	for _, o := range menu {
		if o.Action == a {
			return o
		}
	}
	t.Fatalf("no %v in menu", a)
	return OrderOption{}
}

func itemNamed(t *testing.T, c *Controller, name string) int {
	t.Helper()
	for i, it := range c.Game.Scenario.Items {
		if it.Name == name {
			return i
		}
	}
	t.Fatalf("no item %s", name)
	return 0
}

// Fight: only the front row (slots 1-3), only while a group is alive, and
// only at a living group.
func TestCombatFightRules(t *testing.T) {
	c, p := combatController(t, 2, 1)
	if o := option(t, c, p[0], engine.ActionFight); !o.Available || o.Target != TargetGroup {
		t.Fatalf("front row: %+v", o)
	}
	if o := option(t, c, p[3], engine.ActionFight); o.Available || o.Reason == "" {
		t.Fatalf("back row: %+v", o)
	}
	if err := c.SetOrder(p[3], Order{Action: engine.ActionFight, Group: 0}); !errors.Is(err, ErrOrderUnavailable) {
		t.Fatalf("back row Fight: %v", err)
	}
	for _, g := range []int{1, 4, -1} { // group 2 is dead; 5 and 0 do not exist
		if err := c.SetOrder(p[0], Order{Action: engine.ActionFight, Group: g}); !errors.Is(err, ErrInvalidTarget) {
			t.Fatalf("Fight at group %d: %v", g+1, err)
		}
	}
	if err := c.SetOrder(p[2], Order{Action: engine.ActionFight, Group: 0}); err != nil {
		t.Fatal(err)
	}
	if got := c.Game.Combat.Actions[2]; got != (engine.PartyAction{Action: engine.ActionFight, TargetGroup: 0}) {
		t.Fatalf("recorded %+v", got)
	}
	c2, p2 := combatController(t, 1, 0)
	if o := option(t, c2, p2[0], engine.ActionFight); o.Available {
		t.Fatalf("no living group: %+v", o)
	}
}

// Dispel: priests; bishops above level 3; lords above level 8; at a living
// group.
func TestCombatDispelRules(t *testing.T) {
	cases := []struct {
		class engine.Class
		level int
		ok    bool
	}{
		{engine.Priest, 1, true}, {engine.Bishop, 3, false}, {engine.Bishop, 4, true},
		{engine.Lord, 8, false}, {engine.Lord, 9, true}, {engine.Fighter, 13, false}, {engine.Mage, 13, false},
	}
	for _, k := range cases {
		c, p := combatController(t, 1)
		p[4].Class, p[4].Level = k.class, k.level
		o := option(t, c, p[4], engine.ActionDispel)
		if o.Available != k.ok || (!k.ok && o.Reason == "") {
			t.Errorf("%v level %d: %+v", k.class, k.level, o)
		}
		err := c.SetOrder(p[4], Order{Action: engine.ActionDispel, Group: 0})
		if k.ok != (err == nil) {
			t.Errorf("%v level %d: SetOrder %v", k.class, k.level, err)
		}
	}
	c, p := combatController(t, 2, 0)
	p[5].Class = engine.Priest
	if err := c.SetOrder(p[5], Order{Action: engine.ActionDispel, Group: 0}); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("Dispel at a dead group: %v", err)
	}
}

// Spells: casters only; the name must be known and castable (in-game
// refusals); the spell decides its target; targets are validated.
func TestCombatSpellRules(t *testing.T) {
	c, p := combatController(t, 2, 1)
	if o := option(t, c, p[0], engine.ActionSpell); o.Available || o.Reason == "" {
		t.Fatalf("fighter: %+v", o)
	}
	mage, priest := p[3], p[4]
	mage.Class, priest.Class = engine.Mage, engine.Priest
	mage.MageSpells[0], mage.MaxMageSpells[0] = 1, 1
	mage.SpellKnown[engine.SpellIndex["HALITO"]] = true
	priest.PriestSpells[0], priest.MaxPriestSpells[0] = 1, 1
	priest.SpellKnown[engine.SpellIndex["DIOS"]] = true
	if o := option(t, c, mage, engine.ActionSpell); !o.Available || o.Target != TargetSpell {
		t.Fatalf("mage: %+v", o)
	}
	var r *Refusal
	if _, err := c.CombatSpellTarget(mage, "ZZZ"); !errors.As(err, &r) || r.Message != "** UNKNOWN SPELL **" {
		t.Fatalf("unknown spell: %v", err)
	}
	if _, err := c.CombatSpellTarget(mage, "KATINO"); !errors.As(err, &r) || r.Message != "** CANT CAST **" {
		t.Fatalf("spell not known: %v", err)
	}
	if tg, err := c.CombatSpellTarget(mage, "HALITO"); err != nil || tg != TargetGroup {
		t.Fatalf("HALITO: %v %v", tg, err)
	}
	if tg, err := c.CombatSpellTarget(priest, "DIOS"); err != nil || tg != TargetMember {
		t.Fatalf("DIOS: %v %v", tg, err)
	}
	if err := c.SetOrder(mage, Order{Action: engine.ActionSpell, Spell: "HALITO", Group: 1}); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("HALITO at a dead group: %v", err)
	}
	if err := c.SetOrder(mage, Order{Action: engine.ActionSpell, Spell: "HALITO", Group: 0}); err != nil {
		t.Fatal(err)
	}
	if got := c.Game.Combat.Actions[3]; got != (engine.PartyAction{Action: engine.ActionSpell, SpellName: "HALITO", TargetGroup: 0}) {
		t.Fatalf("recorded %+v", got)
	}
	stranger := engine.NewCharacter("Z", engine.Human, engine.Fighter, engine.Neutral)
	if err := c.SetOrder(priest, Order{Action: engine.ActionSpell, Spell: "DIOS", Ally: stranger}); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("DIOS on someone outside the party: %v", err)
	}
	// Any party member may be chosen, as the terminal allows, even the dead.
	p[1].Status = engine.Dead
	if err := c.SetOrder(priest, Order{Action: engine.ActionSpell, Spell: "DIOS", Ally: p[1]}); err != nil {
		t.Fatal(err)
	}
	if got := c.Game.Combat.Actions[4]; got != (engine.PartyAction{Action: engine.ActionSpell, SpellName: "DIOS", TargetAlly: 1}) {
		t.Fatalf("recorded %+v", got)
	}
	// Kept as found: with no living group, a group spell is recorded with
	// no group, as the terminal did.
	for _, g := range c.Game.Combat.Groups {
		for _, m := range g.Members {
			m.Status = 5
		}
	}
	if err := c.SetOrder(mage, Order{Action: engine.ActionSpell, Spell: "HALITO", Group: -1}); err != nil {
		t.Fatal(err)
	}
	if got := c.Game.Combat.Actions[3]; got != (engine.PartyAction{Action: engine.ActionSpell, SpellName: "HALITO", TargetGroup: -1}) {
		t.Fatalf("recorded %+v", got)
	}
	// Spell points spent: "** CANT CAST **".
	mage.MageSpells[0] = 0
	if _, err := c.CombatSpellTarget(mage, "HALITO"); !errors.As(err, &r) || r.Message != "** CANT CAST **" {
		t.Fatalf("no spell points: %v", err)
	}
}

// Use: equipped items that cast a spell; nothing usable records Parry;
// the item's spell aims itself.
func TestCombatUseRules(t *testing.T) {
	c, p := combatController(t, 2, 0)
	if err := c.SetOrder(p[0], Order{Action: engine.ActionUse, Item: -1}); err != nil {
		t.Fatal(err)
	}
	if got := c.Game.Combat.Actions[0]; got != (engine.PartyAction{Action: engine.ActionParry}) {
		t.Fatalf("nothing usable: %+v", got)
	}
	if o := option(t, c, p[0], engine.ActionUse); !o.Available || o.Note == "" || o.Target != TargetNothing {
		t.Fatalf("nothing usable: %+v", o)
	}
	u := p[1]
	u.Items[0] = engine.Possession{ItemIndex: itemNamed(t, c, "SCROLL/DILTO"), Equipped: true}
	u.Items[1] = engine.Possession{ItemIndex: itemNamed(t, c, "DIOS POTION"), Equipped: true}
	u.Items[2] = engine.Possession{ItemIndex: itemNamed(t, c, "ROD OF FLAME"), Equipped: false}
	u.Items[3] = engine.Possession{ItemIndex: itemNamed(t, c, "LONG SWORD"), Equipped: true}
	u.Items[4] = engine.Possession{ItemIndex: itemNamed(t, c, "SCROLL/HALITO"), Equipped: true}
	u.ItemCount = 5
	if got := c.CombatUsableItems(u); len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 4 {
		t.Fatalf("usable items %v, want [0 1 4]", got)
	}
	if o := option(t, c, u, engine.ActionUse); o.Target != TargetItem {
		t.Fatalf("usable: %+v", o)
	}
	for _, bad := range []int{-1, 2, 3, 7} {
		if err := c.SetOrder(u, Order{Action: engine.ActionUse, Item: bad}); !errors.Is(err, ErrNoSuchItem) {
			t.Errorf("item %d: %v", bad, err)
		}
	}
	if err := c.SetOrder(u, Order{Action: engine.ActionUse, Item: 0}); err != nil {
		t.Fatal(err)
	}
	if got := c.Game.Combat.Actions[1]; got != (engine.PartyAction{Action: engine.ActionUse, UseItemIdx: 0, TargetGroup: 1, TargetAlly: -1}) {
		t.Fatalf("group spell item aims at the first living group: %+v", got)
	}
	if err := c.SetOrder(u, Order{Action: engine.ActionUse, Item: 1}); err != nil {
		t.Fatal(err)
	}
	if got := c.Game.Combat.Actions[1]; got != (engine.PartyAction{Action: engine.ActionUse, UseItemIdx: 1, TargetGroup: -1, TargetAlly: 1}) {
		t.Fatalf("other items aim at the user: %+v", got)
	}
	// Kept as found (compatibility finding): HALITO targets a single monster,
	// and only group and all-monster spells count as group spells here, so a
	// SCROLL/HALITO is aimed at its user with no group.
	if err := c.SetOrder(u, Order{Action: engine.ActionUse, Item: 4}); err != nil {
		t.Fatal(err)
	}
	if got := c.Game.Combat.Actions[1]; got != (engine.PartyAction{Action: engine.ActionUse, UseItemIdx: 4, TargetGroup: -1, TargetAlly: 1}) {
		t.Fatalf("single-monster spell item: %+v", got)
	}
}

// Who may be given an order: members the engine counts as alive (OK, asleep,
// afraid); others are refused. Orders go to the character's current slot.
func TestCombatOrderEligibility(t *testing.T) {
	c, p := combatController(t, 1)
	for _, st := range []engine.Status{engine.Paralyzed, engine.Stoned, engine.Dead, engine.Ashed, engine.Lost} {
		p[0].Status = st
		if _, err := c.CombatOrderMenu(p[0]); !errors.Is(err, ErrCannotAct) {
			t.Errorf("%v: %v", st, err)
		}
		if err := c.SetOrder(p[0], Order{Action: engine.ActionParry}); !errors.Is(err, ErrCannotAct) {
			t.Errorf("%v SetOrder: %v", st, err)
		}
	}
	// Asleep and afraid members are offered orders today (open question Q-D8).
	for _, st := range []engine.Status{engine.Asleep, engine.Afraid} {
		p[0].Status = st
		if err := c.SetOrder(p[0], Order{Action: engine.ActionParry}); err != nil {
			t.Errorf("%v: %v", st, err)
		}
	}
	p[0].Status = engine.OK
	stranger := engine.NewCharacter("Z", engine.Human, engine.Fighter, engine.Neutral)
	if err := c.SetOrder(stranger, Order{Action: engine.ActionParry}); !errors.Is(err, ErrNoSuchPartyMember) {
		t.Fatalf("not in party: %v", err)
	}
	// By character: after positions change, the order lands in the
	// character's current slot.
	b := p[1]
	m := c.Game.Town.Party.Members
	m[1], m[4] = m[4], m[1]
	if err := c.SetOrder(b, Order{Action: engine.ActionRun}); err != nil {
		t.Fatal(err)
	}
	if c.Game.Combat.Actions[4].Action != engine.ActionRun || c.Game.Combat.Actions[1].Action != engine.ActionNone {
		t.Fatalf("order recorded in the wrong slot: %+v", c.Game.Combat.Actions)
	}
	if err := c.SetOrder(p[0], Order{Action: engine.CombatAction(99)}); !errors.Is(err, ErrInvalidChoice) {
		t.Fatalf("unknown action: %v", err)
	}
}

// Order entry is available only while orders are being chosen.
func TestCombatOrdersOnlyDuringOrderEntry(t *testing.T) {
	c, p := combatController(t, 1)
	for _, ph := range []engine.CombatPhase{engine.CombatInit, engine.CombatConfirm, engine.CombatExecute} {
		c.Game.Combat.Phase = ph
		if _, err := c.CombatOrderMenu(p[0]); !errors.Is(err, ErrNotInCombat) {
			t.Errorf("phase %v: %v", ph, err)
		}
	}
	c.Game.Combat = nil
	if err := c.SetOrder(p[0], Order{Action: engine.ActionParry}); !errors.Is(err, ErrNotInCombat) {
		t.Fatalf("no combat: %v", err)
	}
}

// The order-entry operations draw no random numbers.
func TestCombatOrdersUseNoRandomNumbers(t *testing.T) {
	seed(7)
	want := rand.Int63()
	c, p := combatController(t, 2)
	p[4].Class = engine.Priest
	seed(7)
	for _, ch := range p {
		c.CombatOrderMenu(ch)
		c.CombatUsableItems(ch)
		c.CombatSpellTarget(ch, "DIOS")
	}
	c.SetOrder(p[0], Order{Action: engine.ActionFight, Group: 1})
	c.SetOrder(p[4], Order{Action: engine.ActionDispel, Group: 0})
	c.SetOrder(p[5], Order{Action: engine.ActionUse, Item: -1})
	if got := rand.Int63(); got != want {
		t.Fatal("order entry consumed random numbers")
	}
}
