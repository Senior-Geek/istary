package engine

import "github.com/Senior-Geek/istary/data"

// Curses (finding #34). A possession's Cursed flag means the item is stuck
// to its owner: "A cursed item is any item that cannot be voluntarily
// unequipped. If you equip such an item, you will be 'cursed to wear it'"
// (Player's Guide pp.4-5). An item becomes cursed when it is equipped, not
// when it is found: if its type is a cursed one (LONG SWORD -1, CURSED
// ROBE, DEADLY RING...) or it has an alignment other than its wearer's
// ("If a character equips an item that has a different alignment ... it
// will become cursed", p.4). A cursed item cannot be unequipped, dropped,
// traded or sold; it holds its equipment category; Boltac removes it for a
// fee (p.31), which ends the curse.
//
// What else a different alignment does ("the item will seriously impair
// him", p.4) is not described anywhere this engine was traced from, and is
// not modelled.

// CursesOnEquip reports whether c equipping item makes it cursed: a cursed
// type, or an alignment (1 good, 2 neutral, 3 evil; 0 none) other than c's.
func CursesOnEquip(item data.Item, c *Character) bool {
	return item.Cursed || (item.Alignment != 0 && item.Alignment != int(c.Alignment)+1)
}

// KnownCursed reports whether the player knows p is cursed: it is stuck,
// and the player has seen it identified or equipped (equipping a cursed
// item says so). A curse is never shown before then.
func KnownCursed(p Possession) bool {
	return p.Cursed && (p.Identified || p.Equipped)
}
