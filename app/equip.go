package app

import (
	"fmt"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// Equipping in town (M8): one item at a time, for front ends that show a
// character's items instead of walking the equipment categories. The rules
// are the equip walk's (Camp's E)QUIP, and the Training Grounds' Inspect in
// the terminal): an item is equipped in its category (weapon, armour,
// shield, helmet, gauntlets, miscellaneous; special items such as potions
// and scrolls are not equipped), only by a class that can use it, one per
// category; a cursed item holds its category and cannot be taken off; an
// item that curses when worn (finding #34) sticks; the AC is worked out
// again (#35). TownEquip of an item leaves the character exactly as the
// walk would, choosing that item and keeping the rest; TownUnequip as the
// walk choosing nothing for that category. Nothing draws a random number.

// kindNames names the equipment categories (data.Item.TypeID).
var kindNames = map[int]string{0: "weapon", 1: "armour", 2: "shield", 3: "helmet", 4: "gauntlets", 6: "miscellaneous"}

// EquipItem is one of a character's items, as equipping sees it.
type EquipItem struct {
	N          int    `json:"n"` // 1-based, as TownEquip and TownUnequip take
	Item       int    `json:"item"`
	Name       string `json:"name"` // its unknown name until identified
	Kind       string `json:"kind"` // its category, or "" for items not equipped
	Equipped   bool   `json:"equipped"`
	Cursed     bool   `json:"cursed"` // as far as the player knows (engine.KnownCursed)
	Identified bool   `json:"identified"`
	// EquipRefusal is why it cannot be equipped now ("" if it can; also
	// set when it is equipped already); UnequipRefusal why it cannot be
	// taken off ("" if it can).
	EquipRefusal   string `json:"equipRefusal"`
	UnequipRefusal string `json:"unequipRefusal"`
}

// canEquip is why c cannot equip an item of this kind: a special item, or
// one c's class cannot use (the walk's rule, p-code byte 5779); "" if it
// can.
func canEquip(item data.Item, c *engine.Character) string {
	if _, ok := kindNames[item.TypeID]; !ok {
		return "It is not something to equip."
	}
	if item.ClassUse != 0 && item.ClassUse&(1<<uint(c.Class)) == 0 {
		return fmt.Sprintf("A %s cannot use it.", c.Class)
	}
	return ""
}

// equipItemOf describes c's item at pos.
func equipItemOf(c *engine.Character, items []data.Item, pos int) EquipItem {
	p := c.Items[pos]
	e := EquipItem{N: pos + 1, Item: p.ItemIndex, Equipped: p.Equipped, Cursed: engine.KnownCursed(p), Identified: p.Identified}
	if p.ItemIndex < 0 || p.ItemIndex >= len(items) {
		e.EquipRefusal, e.UnequipRefusal = "It is not something to equip.", "It is not equipped."
		return e
	}
	item := items[p.ItemIndex]
	e.Name, e.Kind = item.Name, kindNames[item.TypeID]
	if !p.Identified && item.NameUnknown != "" {
		e.Name = item.NameUnknown
	}
	switch {
	case p.Equipped:
		e.EquipRefusal = "It is equipped."
	default:
		e.EquipRefusal = canEquip(item, c)
		if e.EquipRefusal == "" && cursedIn(c, items, item.TypeID) {
			e.EquipRefusal = fmt.Sprintf("A cursed %s is stuck in its place: only Boltac can remove it.", e.Kind)
		}
	}
	switch {
	case !p.Equipped:
		e.UnequipRefusal = "It is not equipped."
	case p.Cursed:
		e.UnequipRefusal = "It is cursed: only Boltac can remove it."
	}
	return e
}

// townChar is the roster character at index, if town equipping is open:
// in town, not while a character is being created, and not for a member
// of a saved expedition (it refers to them as they are).
func (c *Controller) townChar(index int) (*engine.Character, error) {
	g := c.Game
	if g.Phase != engine.PhaseTown {
		return nil, ErrWrongState
	}
	roster := g.Town.Roster.Characters
	if index < 0 || index >= len(roster) || roster[index] == nil {
		return nil, ErrNoSuchCharacter
	}
	if g.OnExpedition(roster[index].Name) {
		return nil, ErrOnExpedition
	}
	if roster[index].IsStranded() {
		return nil, ErrStranded
	}
	return roster[index], nil
}

// Equipment describes the roster character at index's items for equipping
// in town. It changes nothing.
func (c *Controller) Equipment(index int) ([]EquipItem, error) {
	ch, err := c.townChar(index)
	if err != nil {
		return nil, err
	}
	out := make([]EquipItem, 0, ch.ItemCount)
	for i := 0; i < ch.ItemCount; i++ {
		out = append(out, equipItemOf(ch, c.Game.Scenario.Items, i))
	}
	return out, nil
}

// TownEquip equips the roster character at index's item n (1-based),
// taking off the item worn in its category. It reports whether the item
// stuck (it is cursed now). A refusal is a *Refusal with the reason, and
// changes nothing.
func (c *Controller) TownEquip(index, n int) (cursed bool, err error) {
	ch, err := c.townChar(index)
	if err != nil {
		return false, err
	}
	if n < 1 || n > ch.ItemCount {
		return false, ErrInvalidChoice
	}
	items := c.Game.Scenario.Items
	if why := equipItemOf(ch, items, n-1).EquipRefusal; why != "" {
		return false, &Refusal{Message: why}
	}
	p := &ch.Items[n-1]
	item := items[p.ItemIndex]
	for i := 0; i < ch.ItemCount; i++ {
		q := &ch.Items[i]
		if q.Equipped && q.ItemIndex >= 0 && q.ItemIndex < len(items) && items[q.ItemIndex].TypeID == item.TypeID {
			q.Equipped = false // not cursed: a cursed one refused above
		}
	}
	p.Equipped = true
	if engine.CursesOnEquip(item, ch) {
		p.Cursed = true
	}
	ch.RecalcAC(items)
	return p.Cursed, nil
}

// TownUnequip takes off the roster character at index's item n (1-based).
// A refusal is a *Refusal with the reason, and changes nothing.
func (c *Controller) TownUnequip(index, n int) error {
	ch, err := c.townChar(index)
	if err != nil {
		return err
	}
	if n < 1 || n > ch.ItemCount {
		return ErrInvalidChoice
	}
	if why := equipItemOf(ch, c.Game.Scenario.Items, n-1).UnequipRefusal; why != "" {
		return &Refusal{Message: why}
	}
	ch.Items[n-1].Equipped = false
	ch.RecalcAC(c.Game.Scenario.Items)
	return nil
}
