package app

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// COMBAT-1: spell allowances, the combat spell list and its reasons, the
// spell descriptions, and Party Retreat.

// knownNames lists the spells ch knows, in table order.
func knownNames(ch *engine.Character) []string {
	var out []string
	for i, k := range ch.SpellKnown {
		if k {
			out = append(out, engine.SpellTable[i].Name)
		}
	}
	return out
}

// A new character's casts are the original's (ROLLER KEEPCHYN, engine
// FinalizeCharacter): mages and bishops two level 1 mage casts with HALITO
// and KATINO, priests two level 1 priest casts with DIOS and BADIOS, and
// they start with every cast they have. The other classes start with none.
func TestNewCastersStartWithTheirCasts(t *testing.T) {
	want := map[engine.Class]struct {
		mage, priest int
		known        []string
	}{
		engine.Mage:    {2, 0, []string{"HALITO", "KATINO"}},
		engine.Bishop:  {2, 0, []string{"HALITO", "KATINO"}},
		engine.Priest:  {0, 2, []string{"DIOS", "BADIOS"}},
		engine.Fighter: {}, engine.Thief: {}, engine.Samurai: {}, engine.Lord: {}, engine.Ninja: {},
	}
	for class, w := range want {
		seed(int64(class) + 1)
		cs := &engine.CreationState{Name: "NEW", Race: engine.Elf, Alignment: engine.Good, Class: class,
			Stats: [6]int{14, 14, 14, 14, 14, 14}}
		ch := cs.FinalizeCharacter()
		if ch.MageSpells != [7]int{w.mage} || ch.MaxMageSpells != [7]int{w.mage} ||
			ch.PriestSpells != [7]int{w.priest} || ch.MaxPriestSpells != [7]int{w.priest} {
			t.Errorf("%v: mage %v/%v priest %v/%v, want %d and %d", class, ch.MageSpells, ch.MaxMageSpells,
				ch.PriestSpells, ch.MaxPriestSpells, w.mage, w.priest)
		}
		if got := knownNames(ch); !slices.Equal(got, w.known) {
			t.Errorf("%v: knows %v, want %v", class, got, w.known)
		}
	}
}

// caster turns party member i of a combat controller into a level 1 mage
// knowing the four level 1 mage spells (as one whose saved spells an
// earlier Istary widened, #32), with left of 2 casts.
func caster(p []*engine.Character, i, left int) *engine.Character {
	m := p[i]
	m.Class = engine.Mage
	m.MageSpells[0], m.MaxMageSpells[0] = left, 2
	for _, n := range []string{"HALITO", "MOGREF", "KATINO", "DUMAPIC"} {
		m.SpellKnown[engine.SpellIndex[n]] = true
	}
	return m
}

func spellNamed(t *testing.T, v CombatView, name string) CombatSpellView {
	t.Helper()
	for _, sp := range v.Spells {
		if sp.Name == name {
			return sp
		}
	}
	t.Fatalf("%s not listed: %+v", name, v.Spells)
	return CombatSpellView{}
}

// The combat list is every spell the actor knows, with its description and
// points; what cannot be ordered says why, and is refused if ordered. The
// spells and their points are the engine's: nothing is filtered but by
// the casting rules, and spells with no effect in combat.
func TestCombatSpellListAndReasons(t *testing.T) {
	c, p := combatController(t, 1)
	m := caster(p, 4, 2)
	c.SelectCombatActor(m)
	v := view(t, c)
	if got := len(v.Spells); got != 4 {
		t.Fatalf("listed %d: %+v", got, v.Spells)
	}
	for _, sp := range v.Spells {
		if sp.Summary != SpellSummary(sp.Name) || sp.Summary == "" || sp.Left != 2 || sp.Max != 2 || sp.Level != 1 || sp.Kind != "mage" {
			t.Errorf("%+v", sp)
		}
	}
	for name, target := range map[string]string{"HALITO": "group", "MOGREF": "none", "KATINO": "group"} {
		if sp := spellNamed(t, v, name); !sp.Castable || sp.Target != target || sp.Reason != "" {
			t.Errorf("%s: %+v", name, sp)
		}
	}
	if sp := spellNamed(t, v, "DUMAPIC"); sp.Castable || sp.Reason != "Exploration only: cast it in Camp." {
		t.Errorf("DUMAPIC: %+v", sp)
	}
	for _, o := range v.Menu {
		if o.Action == "spell" && (!o.Available || o.Note != "") {
			t.Errorf("spell order with casts left: %+v", o)
		}
	}

	// No casts left: every spell says so (DUMAPIC keeps its own reason), the
	// order says none can be cast, and an order is refused as before.
	m.MageSpells[0] = 0
	v = view(t, c)
	for _, n := range []string{"HALITO", "MOGREF", "KATINO"} {
		sp := spellNamed(t, v, n)
		if sp.Castable || sp.Left != 0 || sp.Reason != "No level 1 mage casts left (0 of 2); a stay at the Inn restores them." {
			t.Errorf("%s exhausted: %+v", n, sp)
		}
	}
	if sp := spellNamed(t, v, "DUMAPIC"); sp.Reason != "Exploration only: cast it in Camp." {
		t.Errorf("DUMAPIC exhausted: %+v", sp)
	}
	for _, o := range v.Menu {
		if o.Action == "spell" && (!o.Available || o.Note != "no spell can be cast now: see why") {
			t.Errorf("spell order when exhausted: %+v", o)
		}
	}
	var r *Refusal
	if err := c.SetOrder(m, Order{Action: engine.ActionSpell, Spell: "HALITO", Group: 0}); !errors.As(err, &r) || r.Message != "** CANT CAST **" {
		t.Fatalf("exhausted HALITO ordered: %v", err)
	}

	// A bishop who knows a priest spell before having priest casts; a mage
	// who knows one (class changed) and may not cast priest spells.
	m.MageSpells[0] = 2
	m.SpellKnown[engine.SpellIndex["DIOS"]] = true
	if sp := spellNamed(t, view(t, c), "DIOS"); sp.Castable || sp.Reason != "A mage cannot cast priest spells." {
		t.Errorf("mage's DIOS: %+v", sp)
	}
	m.Class = engine.Bishop
	if sp := spellNamed(t, view(t, c), "DIOS"); sp.Castable || sp.Reason != "No level 1 priest casts yet." || sp.Kind != "priest" {
		t.Errorf("bishop's DIOS: %+v", sp)
	}

	// A fighter (once a mage) who still knows spells casts none, as before.
	m.Class = engine.Fighter
	v = view(t, c)
	if v.Spells != nil {
		t.Errorf("fighter's spells: %+v", v.Spells)
	}
	for _, o := range v.Menu {
		if o.Action == "spell" && (o.Available || o.Reason != "knows no spells") {
			t.Errorf("fighter's spell order: %+v", o)
		}
	}
}

// A spell point is spent when the spell is cast in the round, once, and not
// when it is ordered, taken back, or disrupted.
func TestCastingSpendsAPointOnlyWhenCast(t *testing.T) {
	cast, kept := 0, 0
	for s := int64(1); s <= 40; s++ {
		seed(s)
		c, p := confirmed(t, 1)
		m := caster(p, 4, 2)
		c.Game.Combat.Actions[4] = engine.PartyAction{Action: engine.ActionSpell, SpellName: "HALITO", TargetGroup: 0}
		if err := c.ReturnToOrders(); err != nil {
			t.Fatal(err)
		}
		if m.MageSpells[0] != 2 {
			t.Fatalf("seed %d: ordering and taking back spent a point: %v", s, m.MageSpells)
		}
		if err := c.SetOrder(m, Order{Action: engine.ActionSpell, Spell: "HALITO", Group: 0}); err != nil {
			t.Fatal(err)
		}
		for _, o := range p {
			if o != m {
				c.SetOrder(o, Order{Action: engine.ActionParry})
			}
		}
		if err := c.FinishOrders(); err != nil {
			t.Fatal(err)
		}
		asleep := s%5 == 0
		if asleep {
			m.Status = engine.Asleep
		}
		first, err := c.Commit()
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for _, st := range playAll(t, c, first) {
			lines = append(lines, st.Event.Lines...)
		}
		casts := slices.Contains(lines, m.Name+" CASTS HALITO!")
		switch {
		case asleep && (casts || m.MageSpells[0] != 2 || !slices.Contains(lines, m.Name+": SPELL DISRUPTED")):
			t.Fatalf("seed %d: asleep caster: %v, %v", s, m.MageSpells, lines)
		case casts && m.MageSpells[0] != 1:
			t.Fatalf("seed %d: cast once, points %v", s, m.MageSpells)
		case !casts && m.MageSpells[0] != 2:
			t.Fatalf("seed %d: not cast, points %v: %v", s, m.MageSpells, lines)
		}
		if casts {
			cast++
		} else {
			kept++
		}
	}
	if cast == 0 || kept == 0 {
		t.Fatalf("cast %d, kept %d: both should be seen", cast, kept)
	}
}

// The Inn restores every cast, in any room, the stables included (finding
// #10: the Inn, not entering the maze), and the restored spells can be
// cast in the next fight.
func TestTheInnRestoresCasts(t *testing.T) {
	c := newController(t, 1)
	m := engine.NewCharacter("ELEN", engine.Elf, engine.Mage, engine.Neutral)
	m.HP, m.MaxHP, m.Gold = 3, 3, 100
	caster([]*engine.Character{m}, 0, 0)
	c.Game.Town.Party.Members = []*engine.Character{m}
	if err := c.GoTo(engine.Inn); err != nil {
		t.Fatal(err)
	}
	if err := c.InnEnter(0); err != nil {
		t.Fatal(err)
	}
	stay, err := c.Stay(engine.Stables, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.MageSpells[0] != 2 || !stay.Spells || stay.MageBefore[0] != 0 || stay.MageAfter[0] != 2 {
		t.Fatalf("after the stables: %v, stay %+v", m.MageSpells, stay)
	}
	if err := c.LeaveInn(); err != nil {
		t.Fatal(err)
	}
	cc, p := combatController(t, 1)
	p[4] = m
	cc.Game.Town.Party.Members = p
	cc.SelectCombatActor(m)
	if sp := spellNamed(t, view(t, cc), "HALITO"); !sp.Castable || sp.Left != 2 {
		t.Fatalf("HALITO after the Inn: %+v", sp)
	}
}

// Every spell has a description and says what it acts on; the numbers in a
// description are the engine's dice; the combat notes name real spells;
// the guide is the table, in order, with the same words.
func TestSpellDescriptionsMatchTheEngine(t *testing.T) {
	rng := regexp.MustCompile(`(\d+)–(\d+) (damage|hit points)`)
	ac := regexp.MustCompile(`armour class (improves|worsens) by (\d+)`)
	guide := SpellGuide()
	if len(guide) != len(engine.SpellTable) || len(spellSummaries) != len(engine.SpellTable) {
		t.Fatalf("guide %d, descriptions %d, spells %d", len(guide), len(spellSummaries), len(engine.SpellTable))
	}
	for i, sp := range engine.SpellTable {
		g := guide[i]
		if g.Name != sp.Name || g.Level != sp.Level || g.Book != spellBook(sp) || g.Summary == "" || g.Affects == "" ||
			!strings.HasSuffix(g.Summary, ".") || g.Combat != CombatSpellNote(sp.Name) {
			t.Errorf("%s: %+v", sp.Name, g)
		}
		if g.Camp != (campSpells[sp.Name] && sp.Name != "MALOR") {
			t.Errorf("%s: camp %v", sp.Name, g.Camp)
		}
		if mm := rng.FindStringSubmatch(g.Summary); mm != nil && (sp.Effect == engine.EffectDamage || sp.Effect == engine.EffectHeal || sp.Name == "ZILWAN") {
			lo, _ := strconv.Atoi(mm[1])
			hi, _ := strconv.Atoi(mm[2])
			n, d := sp.DiceNum, sp.DiceSides
			if sp.Name == "ZILWAN" {
				n, d = 10, 200 // its dice are in executeSpell (DOHITS 10d200)
			}
			if lo != n+sp.DiceBonus || hi != n*d+sp.DiceBonus {
				t.Errorf("%s says %d–%d, the engine rolls %dd%d+%d", sp.Name, lo, hi, n, d, sp.DiceBonus)
			}
		} else if (sp.Effect == engine.EffectDamage || sp.Effect == engine.EffectHeal) && sp.DiceNum > 0 {
			t.Errorf("%s: no range in %q", sp.Name, g.Summary)
		}
		if mm := ac.FindStringSubmatch(g.Summary); mm != nil {
			by, _ := strconv.Atoi(mm[2])
			better := mm[1] == "improves"
			mod := sp.ACMod
			if sp.Name == "MAPORFIC" {
				mod = -2 // in executeSpell (ACMOD2), checked by TestMaporficImprovesArmourClass
			}
			if (sp.Effect != engine.EffectBuff && sp.Effect != engine.EffectDebuff && sp.Name != "MAPORFIC") || by != abs(mod) || better != (mod < 0) {
				t.Errorf("%s says armour class %s by %d, the engine's change is %+d", sp.Name, mm[1], by, sp.ACMod)
			}
		} else if sp.Effect == engine.EffectBuff || sp.Effect == engine.EffectDebuff {
			t.Errorf("%s: no armour class change in %q", sp.Name, g.Summary)
		}
	}
	for name := range combatNotes {
		if engine.SpellsByName[name] == nil {
			t.Errorf("combat note for no spell %q", name)
		}
	}
	if SpellSummary("NOSUCH") != "" {
		t.Error("a description for no spell")
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Party Retreat gives one Run order, to the most agile member who can act,
// and Parry to everyone else who can act, replacing earlier orders, and
// goes to confirmation, where the orders can be taken back.
func TestPartyRetreatOrders(t *testing.T) {
	c, p := combatController(t, 2)
	for i, agi := range []int{10, 14, 18, 9, 14, 12} {
		p[i].Agility = agi
	}
	p[2].Status = engine.Paralyzed // the most agile cannot act
	p[0].Status = engine.Dead
	p[5].Status = engine.Asleep // asleep members are given orders, as Classic (finding #16)
	c.SetOrder(p[1], Order{Action: engine.ActionFight, Group: 1})
	if v := view(t, c); v.Retreat != p[1].Name {
		t.Fatalf("retreat offered by %q, want %s (agility 14, before %s)", v.Retreat, p[1].Name, p[4].Name)
	}
	if err := c.PartyRetreat(); err != nil {
		t.Fatal(err)
	}
	cs := c.Game.Combat
	want := []engine.CombatAction{engine.ActionNone, engine.ActionRun, engine.ActionNone, engine.ActionParry, engine.ActionParry, engine.ActionParry}
	for i, a := range cs.Actions {
		if a.Action != want[i] {
			t.Errorf("slot %d: %v, want %v", i+1, a.Action, want[i])
		}
	}
	if cs.Phase != engine.CombatConfirm {
		t.Fatalf("phase %v", cs.Phase)
	}
	if v := view(t, c); v.Retreat != "" || len(v.Orders) != 4 {
		t.Fatalf("at confirmation: retreat %q, orders %+v", v.Retreat, v.Orders)
	}
	if err := c.PartyRetreat(); !errors.Is(err, ErrNotInCombat) {
		t.Fatalf("retreat again at confirmation: %v", err)
	}
	if err := c.ReturnToOrders(); err != nil || cs.Phase != engine.CombatChoose {
		t.Fatalf("taken back: %v %v", err, cs.Phase)
	}

	// Nobody who can act: no retreat, and nothing changes.
	for _, m := range p {
		m.Status = engine.Paralyzed
	}
	before := slices.Clone(cs.Actions)
	if err := c.PartyRetreat(); !errors.Is(err, ErrOrderUnavailable) || !slices.Equal(cs.Actions, before) {
		t.Fatalf("nobody to run: %v %+v", err, cs.Actions)
	}
	if c.RetreatRunner() != nil {
		t.Fatal("a runner when nobody can act")
	}
}

// playRound commits, plays back and ends the round, returning every line,
// the party-run events' escape outcomes as the view showed them, and the
// outcome.
func playRound(t *testing.T, c *Controller) ([]string, []string, RoundOutcome) {
	t.Helper()
	first, err := c.Commit()
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, st := range playAll(t, c, first) {
		lines = append(lines, st.Event.Lines...)
	}
	var escapes []string
	for _, e := range view(t, c).Events {
		if e.Kind == "party-run" {
			escapes = append(escapes, e.Escape)
		} else if e.Escape != "" {
			t.Fatalf("escape on a %s event", e.Kind)
		}
	}
	out, err := c.EndRound()
	if err != nil {
		t.Fatal(err)
	}
	return lines, escapes, out
}

// Party Retreat is the Run order, no more: with the same random numbers it
// plays exactly as the same orders given one by one, so its chance and its
// consequences are Run's. Both outcomes are seen; a failed retreat loses
// the runner's turn and the fight goes on, and a success leaves combat.
func TestPartyRetreatIsTheRunOrder(t *testing.T) {
	seen := map[RoundOutcome]int{}
	for s := int64(1); s <= 80; s++ {
		play := func(retreat bool) ([]string, []string, RoundOutcome, *Controller) {
			c, p := confirmed(t, 2)
			c.ReturnToOrders()
			rng := rand.New(rand.NewSource(s))
			for _, m := range p {
				m.Agility = 3 + rng.Intn(16)
			}
			p[rng.Intn(6)].Status = engine.Dead
			if retreat {
				if err := c.PartyRetreat(); err != nil {
					t.Fatal(err)
				}
			} else {
				runner := c.RetreatRunner()
				for _, m := range p {
					if !CanGiveOrder(m) {
						continue
					}
					o := Order{Action: engine.ActionParry}
					if m == runner {
						o.Action = engine.ActionRun
					}
					if err := c.SetOrder(m, o); err != nil {
						t.Fatal(err)
					}
				}
				if err := c.FinishOrders(); err != nil {
					t.Fatal(err)
				}
			}
			seed(s)
			lines, escapes, out := playRound(t, c)
			return lines, escapes, out, c
		}
		l1, e1, o1, c1 := play(true)
		l2, e2, o2, _ := play(false)
		if !slices.Equal(l1, l2) || !slices.Equal(e1, e2) || o1 != o2 {
			t.Fatalf("seed %d: retreat played %v %v %v; the same orders by hand %v %v %v", s, o1, e1, l1, o2, e2, l2)
		}
		switch o1 {
		case RoundEscaped:
			if len(e1) != 1 || e1[0] != "escaped" || !slices.Contains(l1, "THE PARTY FLED!") || c1.Game.Phase != engine.PhaseMaze {
				t.Fatalf("seed %d: escaped: %v %v", s, e1, l1)
			}
		case RoundContinues:
			if len(e1) != 1 || e1[0] != "failed" || !slices.ContainsFunc(l1, func(l string) bool { return strings.HasSuffix(l, " FAILED TO RUN!") }) {
				t.Fatalf("seed %d: failed: %v %v", s, e1, l1)
			}
		}
		seen[o1]++
	}
	if seen[RoundEscaped] == 0 || seen[RoundContinues] == 0 {
		t.Fatalf("outcomes seen: %v", seen)
	}
	t.Logf("outcomes: %v", fmt.Sprint(seen))
}
