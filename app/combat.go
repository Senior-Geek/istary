package app

// Combat order entry (M5a): which orders a party member may be given, and
// SetOrder, which validates an order and records it for the round. The rules
// are the ones the terminal applied in handleCombatChoose (Pascal CUTIL);
// none is new. Order entry's prompts and navigation stay in the front end.
//
// Orders are given to a character, not a slot: the controller finds the
// character's current slot when it records the order, and the engine binds
// it to that character (correction C1, engine/combat_rounds.go).

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// Errors returned by combat order entry.
var (
	ErrNotInCombat      = errors.New("no combat order entry is in progress")
	ErrCannotAct        = errors.New("cannot be given an order")
	ErrOrderUnavailable = errors.New("order is not available")
	ErrInvalidTarget    = errors.New("invalid target")
	ErrNoSuchItem       = errors.New("no usable item in that slot")
)

// OrderTarget says what must be chosen after an order before it is complete.
type OrderTarget int

const (
	TargetNothing OrderTarget = iota // the order is complete as it is
	TargetGroup                      // a living monster group (asked only when more than one is alive)
	TargetMember                     // a party member
	TargetSpell                      // a spell name; the spell then decides its own target
	TargetItem                       // one of the character's usable items
)

// OrderOption is one entry of a character's combat order menu.
type OrderOption struct {
	Action    engine.CombatAction
	Key       rune // the terminal key, for front ends that show one
	Available bool
	Reason    string      // why it is unavailable; empty when available
	Note      string      // what an available order will do that is not obvious
	Target    OrderTarget // what must be chosen next
}

// Order is an order for one character. Only the fields its action needs are
// read: Group for Fight, Dispel and group spells; Spell for spells; Ally for
// spells cast on a party member; Item (an index into the character's items)
// for Use. Use with Item < 0 is allowed only when nothing is usable, and is
// then recorded as Parry, as Classic does.
type Order struct {
	Action engine.CombatAction
	Group  int
	Spell  string
	Ally   *engine.Character
	Item   int
}

// combatChoosing returns the combat when orders are being entered.
func (c *Controller) combatChoosing() (*engine.CombatState, error) {
	cs := c.Game.Combat
	if c.Game.Phase != engine.PhaseCombat || cs == nil || cs.Phase != engine.CombatChoose {
		return nil, ErrNotInCombat
	}
	return cs, nil
}

// partySlot returns ch's current position in the party, or -1.
func (c *Controller) partySlot(ch *engine.Character) int {
	if ch == nil {
		return -1
	}
	return PartySlot(c.Game.Town, ch)
}

// CanGiveOrder reports whether ch is offered an order this round: members
// the engine counts as alive (OK, asleep or afraid). Paralyzed, stoned, dead,
// ashed and lost members are skipped.
func CanGiveOrder(ch *engine.Character) bool {
	return ch != nil && ch.IsAlive()
}

// canDispel is the class rule for D)ISPELL (game.pas lines 4387-4393):
// priests always, bishops above level 3, lords above level 8.
func canDispel(ch *engine.Character) bool {
	return ch.Class == engine.Priest ||
		(ch.Class == engine.Lord && ch.Level > 8) ||
		(ch.Class == engine.Bishop && ch.Level > 3)
}

// CombatUsableItems returns the indexes of ch's items that U)SE offers:
// equipped items that cast a spell (Pascal COMBAT2.TEXT lines 51-173).
func (c *Controller) CombatUsableItems(ch *engine.Character) []int {
	var out []int
	items := c.Game.Scenario.Items
	for i := 0; i < ch.ItemCount; i++ {
		poss := ch.Items[i]
		if poss.ItemIndex > 0 && poss.ItemIndex < len(items) && items[poss.ItemIndex].SpellPower > 0 && poss.Equipped {
			out = append(out, i)
		}
	}
	return out
}

// CombatOrderMenu returns the orders ch may be given now, in the terminal's
// menu order, each with whether it is available and why not.
func (c *Controller) CombatOrderMenu(ch *engine.Character) ([]OrderOption, error) {
	cs, err := c.combatChoosing()
	if err != nil {
		return nil, err
	}
	slot := c.partySlot(ch)
	if slot < 0 {
		return nil, ErrNoSuchPartyMember
	}
	if !CanGiveOrder(ch) {
		return nil, fmt.Errorf("%s %w: %s", ch.Name, ErrCannotAct, strings.ToLower(ch.Status.String()))
	}
	noGroup := cs.FirstAliveGroup() < 0
	unless := func(ok bool, why string) (bool, string) {
		if ok {
			return true, ""
		}
		return false, why
	}

	fightOK, fightWhy := unless(slot < FrontRowSize, fmt.Sprintf("only the front row (1-%d) can fight", FrontRowSize))
	if fightOK && noGroup {
		fightOK, fightWhy = false, "no monster group is left"
	}
	spellOK, spellWhy := unless(ch.IsCaster(), "knows no spells")
	dispelOK, dispelWhy := unless(canDispel(ch), "only priests, bishops above level 3 and lords above level 8 can dispel")
	if dispelOK && noGroup {
		dispelOK, dispelWhy = false, "no monster group is left"
	}
	use := OrderOption{Action: engine.ActionUse, Key: 'U', Available: true, Target: TargetItem}
	if len(c.CombatUsableItems(ch)) == 0 {
		use.Note, use.Target = "nothing usable: counts as Parry", TargetNothing
	}
	return []OrderOption{
		{Action: engine.ActionFight, Key: 'F', Available: fightOK, Reason: fightWhy, Target: TargetGroup},
		{Action: engine.ActionSpell, Key: 'S', Available: spellOK, Reason: spellWhy, Target: TargetSpell},
		{Action: engine.ActionParry, Key: 'P', Available: true},
		{Action: engine.ActionRun, Key: 'R', Available: true},
		use,
		{Action: engine.ActionDispel, Key: 'D', Available: dispelOK, Reason: dispelWhy, Target: TargetGroup},
	}, nil
}

// CombatSpellTarget checks that ch may cast the named spell now and says
// what it needs: a group, a member, or nothing. The refusals are the
// in-game messages "** UNKNOWN SPELL **" and "** CANT CAST **".
func (c *Controller) CombatSpellTarget(ch *engine.Character, name string) (OrderTarget, error) {
	if _, err := c.combatChoosing(); err != nil {
		return TargetNothing, err
	}
	if c.partySlot(ch) < 0 {
		return TargetNothing, ErrNoSuchPartyMember
	}
	if !CanGiveOrder(ch) || !ch.IsCaster() {
		return TargetNothing, ErrOrderUnavailable
	}
	sp := engine.LookupSpell(name)
	if sp == nil {
		return TargetNothing, &Refusal{Message: "** UNKNOWN SPELL **"}
	}
	if !ch.CanCastSpell(sp) {
		return TargetNothing, &Refusal{Message: "** CANT CAST **"}
	}
	switch sp.Target {
	case engine.TargetSingleMonster, engine.TargetMonsterGroup:
		return TargetGroup, nil
	case engine.TargetPartyMember:
		return TargetMember, nil
	}
	return TargetNothing, nil
}

// SetOrder validates o for ch and records it as ch's order for this round.
// It replaces any earlier order for ch. It does not move order entry on;
// that is the front end's navigation.
func (c *Controller) SetOrder(ch *engine.Character, o Order) error {
	cs, err := c.combatChoosing()
	if err != nil {
		return err
	}
	slot := c.partySlot(ch)
	if slot < 0 {
		return ErrNoSuchPartyMember
	}
	menu, err := c.CombatOrderMenu(ch)
	if err != nil {
		return err
	}
	var opt *OrderOption
	for i := range menu {
		if menu[i].Action == o.Action {
			opt = &menu[i]
		}
	}
	if opt == nil {
		return ErrInvalidChoice
	}
	if !opt.Available {
		return fmt.Errorf("%w: %s", ErrOrderUnavailable, opt.Reason)
	}
	livingGroup := func(g int) bool {
		return g >= 0 && g < len(cs.Groups) && cs.Groups[g].AliveCount() > 0
	}

	var a engine.PartyAction
	switch o.Action {
	case engine.ActionFight, engine.ActionDispel:
		if !livingGroup(o.Group) {
			return ErrInvalidTarget
		}
		a = engine.PartyAction{Action: o.Action, TargetGroup: o.Group}
	case engine.ActionParry, engine.ActionRun:
		a = engine.PartyAction{Action: o.Action}
	case engine.ActionSpell:
		target, err := c.CombatSpellTarget(ch, o.Spell)
		if err != nil {
			return err
		}
		switch target {
		case TargetGroup:
			// Kept as found: if no group is alive, the terminal recorded the
			// spell with no group (-1). The known route to that state, a
			// successful Run leaving the round at confirmation
			// (docs/WIZARDRY-COMPATIBILITY.md #17), is corrected in M5b.
			noneAlive := cs.FirstAliveGroup() < 0 && o.Group == -1
			if !livingGroup(o.Group) && !noneAlive {
				return ErrInvalidTarget
			}
			a = engine.PartyAction{Action: engine.ActionSpell, SpellName: o.Spell, TargetGroup: o.Group}
		case TargetMember:
			// Any member of the party, as the terminal allows.
			ally := c.partySlot(o.Ally)
			if ally < 0 {
				return ErrInvalidTarget
			}
			a = engine.PartyAction{Action: engine.ActionSpell, SpellName: o.Spell, TargetAlly: ally}
		default:
			a = engine.PartyAction{Action: engine.ActionSpell, SpellName: o.Spell}
		}
	case engine.ActionUse:
		usable := c.CombatUsableItems(ch)
		if len(usable) == 0 {
			if o.Item >= 0 {
				return ErrNoSuchItem
			}
			a = engine.PartyAction{Action: engine.ActionParry} // nothing usable: Parry
			break
		}
		ok := false
		for _, i := range usable {
			ok = ok || i == o.Item
		}
		if !ok {
			return ErrNoSuchItem
		}
		// The item's spell aims itself: group spells at the first living
		// group, any other spell at the user.
		item := &c.Game.Scenario.Items[ch.Items[o.Item].ItemIndex]
		sp := engine.SpellTable[item.SpellPower-1]
		targetGroup, targetAlly := cs.FirstAliveGroup(), slot
		if sp.Target == engine.TargetMonsterGroup || sp.Target == engine.TargetAllMonsters {
			targetAlly = -1
		} else {
			targetGroup = -1
		}
		a = engine.PartyAction{Action: engine.ActionUse, UseItemIdx: o.Item, TargetGroup: targetGroup, TargetAlly: targetAlly}
	default:
		return ErrInvalidChoice
	}
	cs.Actions[slot] = a
	return nil
}
