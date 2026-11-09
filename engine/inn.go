package engine

import (
	"fmt"
	"math/rand"
)

// InnRoomQuality is the HP healed per week for each room type.
// Traced from CASTLE segment p-code (byte 5141-5171):
//   Stables=0, Cot=1, Economy=3, Merchant=7, Royal=10
// The Inn itself is app/inn.go (its rooms, the heal loop and the level
// check).
var InnRoomQuality = [...]int{0, 1, 3, 7, 10}

// restoreSpells sets current spell slots to max based on character class.
// From p-code proc 18/17 (IC 2700/2778) — CIP 22 dispatches by class:
//   Fighter/Thief/Ninja: no spells
//   Mage: mage spells
//   Priest: priest spells
//   Bishop: both
//   Samurai: mage only
//   Lord: priest only
// The original restores spells silently (no display message).
func RestoreSpells(c *Character) {
	switch c.Class {
	case Mage, Bishop, Samurai:
		for i := 0; i < 7; i++ {
			c.MageSpells[i] = c.MaxMageSpells[i]
		}
	}

	switch c.Class {
	case Priest, Bishop, Lord:
		for i := 0; i < 7; i++ {
			c.PriestSpells[i] = c.MaxPriestSpells[i]
		}
	}
}

// innStatChanges applies age-dependent stat modifications.
// From p-code proc 34 (byte 3900, lex 6):
//   For each of 6 stats (STR, IQ, PIE, VIT, AGI, LCK):
//     if random()%130 < AGE/52: stat may decrease
//     else: stat may increase
//   Decrease from 18: only 1/6 chance
//   If VIT drops to 2: old age death
func InnStatChanges(c *Character) []string {
	var msgs []string
	statNames := [6]string{"STRENGTH", "I.Q.", "PIETY", "VITALITY", "AGILITY", "LUCK"}
	stats := [6]*int{&c.Strength, &c.IQ, &c.Piety, &c.Vitality, &c.Agility, &c.Luck}

	ageYears := c.Age / 52

	for i := 0; i < 6; i++ {
		// From p-code proc 6 (IC 3911-3920): random()%4 == 0 → skip this stat
		// (backward jump target in .pseudo is wrong; must be forward skip)
		if rand.Intn(4) == 0 {
			continue
		}
		// From p-code: random()%130 < ageYears → decrease path
		if rand.Intn(130) < ageYears {
			// Decrease path (bytes 3961-4016)
			if *stats[i] == 18 {
				// At max: only 1/6 chance of decrease
				if rand.Intn(6) != 4 {
					continue
				}
			}
			if *stats[i] > 1 {
				*stats[i]--
				msgs = append(msgs, fmt.Sprintf("YOU LOST %s", statNames[i]))

				// VIT drop to 2 = old age death (proc 7, IC 3816)
				// P-code IC 4009: EQUI (==), not LEQI (<=)
				if i == 3 && c.Vitality == 2 {
					msgs = append(msgs, "** YOU HAVE DIED OF OLD AGE **")
					c.Status = Lost
					c.HP = 0
					return msgs
				}
			}
		} else {
			// Increase path (bytes 4018-4049)
			if *stats[i] < 18 {
				*stats[i]++
				msgs = append(msgs, fmt.Sprintf("YOU GAINED %s", statNames[i]))
			}
		}
	}

	return msgs
}
