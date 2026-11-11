package app

import (
	"fmt"
	"strings"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// The equipment check (UX-1): before a party sets out, which members carry
// no usable weapon or no protection. It is a caution only: it buys, equips
// and changes nothing, and the party may set out regardless.
//
// The rules are the game's own, never a second set:
//
//   - Equipped means equipped (Possession.Equipped); an item merely carried
//     does not count.
//   - Usable is equipping's class rule (canEquip, Item.ClassUse). Nothing
//     else limits who may use an item: alignment only decides whether an
//     item curses when worn (finding #34).
//   - A weapon is an item of the weapon category (data.Item.TypeID 0);
//     combat swings the equipped one, or fights unarmed for 1d2. Any usable
//     weapon will do: a staff is as good as a sword here.
//   - Protection is armour, a shield, a helmet or gauntlets (TypeID 1-4).
//     It is asked of a class only when the scenario offers that class some
//     protection beyond what every class may wear (ROBES, in the original):
//     so a Mage, who may wear nothing heavier, is never told to.
//   - A Ninja with nothing equipped is fighting unarmed on purpose: combat
//     rewards it (WC030, initiative) and the Ninja's critical hits do not
//     need a weapon. Such a Ninja is not cautioned.
//   - Only members who can act are checked: the incapacitated carried home
//     do not fight.

// ReadinessNote is one party member's equipment problems.
type ReadinessNote struct {
	Slot     int      `json:"slot"` // from 1
	Name     string   `json:"name"`
	Problems []string `json:"problems"` // "has no weapon equipped", ...
}

// Equipment kinds the check looks at (data.Item.TypeID).
const (
	kindWeapon    = 0
	kindGauntlets = 4
)

// protective reports whether an item of type t protects (armour, shield,
// helmet, gauntlets).
func protective(t int) bool { return t > kindWeapon && t <= kindGauntlets }

// classUses reports whether class may use item (canEquip's class rule).
func classUses(item data.Item, class engine.Class) bool {
	return item.ClassUse == 0 || item.ClassUse&(1<<uint(class)) != 0
}

// protectionExpected reports whether the scenario offers class some
// protection not every class may wear.
func protectionExpected(items []data.Item, class engine.Class) bool {
	for _, it := range items {
		if !protective(it.TypeID) || !classUses(it, class) {
			continue
		}
		for k := range engine.ClassNames {
			if !classUses(it, engine.Class(k)) {
				return true // not for everyone, and this class may wear it
			}
		}
	}
	return false
}

// equipmentProblems is what ch lacks, in the check's words.
func equipmentProblems(items []data.Item, ch *engine.Character) []string {
	var weapon, unusableWeapon, protected, unusableProtection, anything bool
	for i := 0; i < ch.ItemCount && i < len(ch.Items); i++ {
		p := ch.Items[i]
		if !p.Equipped {
			continue
		}
		anything = true
		if p.ItemIndex < 0 || p.ItemIndex >= len(items) {
			continue
		}
		it := items[p.ItemIndex]
		usable := canEquip(it, ch) == ""
		switch {
		case it.TypeID == kindWeapon && usable:
			weapon = true
		case it.TypeID == kindWeapon:
			unusableWeapon = true
		case protective(it.TypeID) && usable:
			protected = true
		case protective(it.TypeID):
			unusableProtection = true
		}
	}
	if ch.Class == engine.Ninja && !anything {
		return nil
	}
	var out []string
	switch {
	case weapon:
	case unusableWeapon:
		out = append(out, fmt.Sprintf("has a weapon equipped that a %s cannot use", ch.Class))
	default:
		out = append(out, "has no weapon equipped")
	}
	if !protected && protectionExpected(items, ch.Class) {
		if unusableProtection {
			out = append(out, fmt.Sprintf("has only armor equipped that a %s cannot use", ch.Class))
		} else {
			out = append(out, "has no armor equipped")
		}
	}
	return out
}

// EquipmentCheck is the party's equipment problems, in party order; empty
// when everyone who can act is ready. It reads the game and changes
// nothing.
func (c *Controller) EquipmentCheck() []ReadinessNote {
	g := c.Game
	notes := []ReadinessNote{}
	for i, ch := range g.Town.Party.Members {
		if ch == nil || !ch.IsAlive() {
			continue
		}
		if p := equipmentProblems(g.Scenario.Items, ch); len(p) > 0 {
			notes = append(notes, ReadinessNote{Slot: i + 1, Name: ch.Name, Problems: p})
		}
	}
	return notes
}

// EquipmentCheckLine is the check as one sentence ("Equipment check: BILBO
// has no weapon equipped; HADES has no armor equipped."), or "" when there
// is nothing to say.
func EquipmentCheckLine(notes []ReadinessNote) string {
	var parts []string
	for _, n := range notes {
		parts = append(parts, n.Name+" "+strings.Join(n.Problems, " and "))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Equipment check: " + strings.Join(parts, "; ") + "."
}
