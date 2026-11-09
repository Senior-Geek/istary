package app

import (
	"fmt"

	"github.com/Senior-Geek/istary/engine"
)

// The Temple's operations (T-1, T-2), for front ends without keys. Treat is
// the terminal's keys pressed for the player (the name, RETURN, the payer's
// number, a key at the ritual), so the two cannot differ: the donation is
// taken, then engine.TempleHeal treats the patient and draws its numbers,
// exactly as the keys do, and the donation is kept whatever the outcome. A
// refusal is decided before any key and changes nothing, drawing nothing.
//
// Treat is stricter than the keys in three cases, on purpose (the Temple
// assignment; the terminal keeps the original's behaviour, pinned by
// cmd/istary/temple_test.go):
//   - the patient must be at the castle: not a member of a saved
//     expedition, and not out in the maze. A character outside the party
//     who is marked in the maze (INMAZE or a maze level, the Tavern's own
//     "** OUT **", Pascal ADDPARTY) is somewhere below; the party's own
//     members are marked INMAZE by joining (Pascal ADDPARTY line 179) but
//     stand in the castle with it;
//   - the payer must be OK: an incapacitated member's gold can still help,
//     by Pool Gold (app/gold.go), but they do not tithe themselves;
//   - a payer short of the donation is refused before anything happens
//     (the keys refuse too, "CHEAP APOSTATES! OUT!", after a step).

// TempleKind is what the Temple does for a condition.
const (
	TempleCure  = "cure"  // paralysis or stone: assured
	TempleRaise = "raise" // dead: may fail into ashes
	TempleAshes = "ashes" // ashes: may fail, and then LOST for good
)

func templeKind(st engine.Status) string {
	switch st {
	case engine.Paralyzed, engine.Stoned:
		return TempleCure
	case engine.Dead:
		return TempleRaise
	case engine.Ashed:
		return TempleAshes
	}
	return ""
}

// atTemple reports that the player is at the Temple.
func (c *Controller) atTemple() bool {
	g := c.Game
	return g.Phase == engine.PhaseTown && g.Town.Location == engine.Temple
}

// patientRefusal is why ch cannot be treated now, or "": as the Temple
// answers (LOST, OK), or because ch is not at the castle.
func (c *Controller) patientRefusal(ch *engine.Character) string {
	switch {
	case ch.Status == engine.Lost:
		return ch.Name + " IS LOST"
	case engine.TempleDonation(ch.Status, ch.Level) == 0:
		return ch.Name + " IS OK"
	case c.Game.OnExpedition(ch.Name):
		return fmt.Sprintf("%s is out on an expedition, not at the castle.", ch.Name)
	case !InParty(c.Game.Town, ch) && (ch.InMaze || ch.MazeLevel != 0):
		return fmt.Sprintf("%s is OUT in the maze, not at the castle; only a rescue can bring them back.", ch.Name)
	}
	return ""
}

// payerRefusal is why the party member in slot cannot tithe donation, or "".
func payerRefusal(m *engine.Character, donation int) string {
	switch {
	case m.Status != engine.OK:
		return fmt.Sprintf("%s IS NOT OK and cannot tithe.", m.Name)
	case m.Gold < donation:
		return "CHEAP APOSTATES! OUT!"
	}
	return ""
}

// TempleReceipt is what a treatment did: the patient before and after, the
// donation and who paid it.
type TempleReceipt struct {
	Patient      string   `json:"patient"`
	Index        int      `json:"index"` // roster index
	Kind         string   `json:"kind"`  // cure, raise or ashes
	StatusBefore string   `json:"statusBefore"`
	StatusAfter  string   `json:"statusAfter"`
	Healed       bool     `json:"healed"` // the treatment succeeded
	Donation     int      `json:"donation"`
	Payer        string   `json:"payer"`
	PayerBefore  int      `json:"payerBefore"` // the payer's gold
	PayerAfter   int      `json:"payerAfter"`
	HPBefore     int      `json:"hpBefore"`
	HPAfter      int      `json:"hpAfter"`
	MaxHP        int      `json:"maxHp"`
	AgeBefore    int      `json:"ageBefore"` // weeks
	AgeAfter     int      `json:"ageAfter"`
	Messages     []string `json:"messages"` // the Temple's words
}

// Treat treats the roster character at index, the donation tithed by the
// party member in slot (0-based): one action, as the keys take it.
func (c *Controller) Treat(index, slot int) (*TempleReceipt, error) {
	town := c.Game.Town
	if !c.atTemple() || town.TempleStep == engine.TempleRitual {
		return nil, ErrWrongState
	}
	roster := town.Roster.Characters
	if index < 0 || index >= len(roster) || roster[index] == nil {
		return nil, ErrNoSuchCharacter
	}
	if slot < 0 || slot >= len(town.Party.Members) || town.Party.Members[slot] == nil {
		return nil, ErrNoSuchPartyMember
	}
	ch, payer := roster[index], town.Party.Members[slot]
	if why := c.patientRefusal(ch); why != "" {
		return nil, &Refusal{Message: why}
	}
	donation := engine.TempleDonation(ch.Status, ch.Level)
	if why := payerRefusal(payer, donation); why != "" {
		return nil, &Refusal{Message: why}
	}
	r := &TempleReceipt{
		Patient: ch.Name, Index: index, Kind: templeKind(ch.Status), StatusBefore: ch.Status.String(),
		Donation: donation, Payer: payer.Name, PayerBefore: payer.Gold,
		HPBefore: ch.HP, AgeBefore: ch.Age,
	}
	c.toTempleWho()
	for _, k := range ch.Name {
		c.press(Rune(k))
	}
	c.press(Key{Kind: KeyEnter})
	c.press(Rune('1' + rune(slot)))
	c.press(Key{Kind: KeyEnter}) // at the ritual: the treatment
	r.StatusAfter, r.Healed = ch.Status.String(), ch.Status == engine.OK
	r.PayerAfter, r.HPAfter, r.MaxHP, r.AgeAfter = payer.Gold, ch.HP, ch.MaxHP, ch.Age
	r.Messages = append([]string(nil), town.TempleMessages...)
	return r, nil
}

// toTempleWho brings the Temple back to "WHO ARE YOU HELPING" with nothing
// typed, by RETURN as the keys do (a result or a donation shown).
func (c *Controller) toTempleWho() {
	town := c.Game.Town
	for town.TempleStep == engine.TempleResult || town.TempleStep == engine.TempleTithe {
		c.press(Key{Kind: KeyEnter})
	}
	for town.InputBuf != "" {
		c.press(Key{Kind: KeyBackspace})
	}
}

// LeaveTemple leaves the Temple for the castle from wherever the player is
// in it, by RETURN as the keys do. Refused at the ritual, where the keys'
// RETURN would carry out a paid treatment (Treat never stops there).
func (c *Controller) LeaveTemple() error {
	if !c.atTemple() || c.Game.Town.TempleStep == engine.TempleRitual {
		return ErrWrongState
	}
	c.toTempleWho()
	c.press(Key{Kind: KeyEnter}) // nothing typed: the castle
	return nil
}
