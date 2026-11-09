package engine

import "github.com/Senior-Geek/istary/data"

// Armor class (finding #35). A character's AC is derived from what they
// wear: 10, less each equipped item's armor modifier (lower is better).
// Character.AC holds the result; RecalcAC is the only place it is worked
// out, and it is called whenever equipment changes (the equip walk,
// selling, uncursing, dropping) and when a game is loaded, so the stored
// value is never stale. Combat's spell modifiers are kept apart (combat
// state), not in Character.AC.

// RecalcAC sets c.AC from c's equipped items.
func (c *Character) RecalcAC(items []data.Item) {
	ac := 10
	for i := 0; i < c.ItemCount; i++ {
		if !c.Items[i].Equipped {
			continue
		}
		if idx := c.Items[i].ItemIndex; idx >= 0 && idx < len(items) {
			ac -= items[idx].ACMod
		}
	}
	c.AC = ac
}
