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
//     (the keys refuse too, "CHEAP APOSTATES! OUT!", after a step), unless
//     the party contributes (Treat's party, TempleQuote "party": guide
//     p.33), which the keys do not offer.

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
	case !InParty(c.Game.Town, ch) && ch.OutOfCastle():
		return fmt.Sprintf("%s is OUT in the maze, not at the castle; only a rescue can bring them back.", ch.Name)
	}
	return ""
}

// payerRefusal is why party member m may not tithe at all, or "".
func payerRefusal(m *engine.Character) string {
	if m.Status != engine.OK {
		return fmt.Sprintf("%s IS NOT OK and cannot tithe.", m.Name)
	}
	return ""
}

// TempleShare is what one other party member brings to the fund when the
// party contributes: all their gold (Pool Gold).
type TempleShare struct {
	Slot int    `json:"slot"` // 0-based
	Name string `json:"name"`
	Gold int    `json:"gold"`
}

// TempleQuote is how a donation would be met for a patient and a payer,
// worked out by the rules Treat follows, changing nothing:
//   - "payer": the payer has the donation and pays it;
//   - "party": the payer is short and the party's gold together covers it:
//     the entire party's gold goes into the fund (Player's Guide p.33),
//     that is, every other member's gold is pooled to the payer (the
//     accepted Pool Gold, app/gold.go: every member, whatever their
//     condition), the donation is taken from it, and the rest stays with
//     the payer, as after any pooling;
//   - "" with Refusal: the patient cannot be treated, the payer may not
//     tithe, or even the party together is short ("CHEAP APOSTATES! OUT!",
//     the original's words).
type TempleQuote struct {
	Patient   string        `json:"patient"`
	Donation  int           `json:"donation"`
	Payer     string        `json:"payer"`
	PayerGold int           `json:"payerGold"`
	PartyGold int           `json:"partyGold"` // every member's gold, the payer's included
	Mode      string        `json:"mode"`
	Shares    []TempleShare `json:"shares"` // "party": each other member's gold brought to the fund
	Refusal   string        `json:"refusal"`
}

// templeQuote quotes the treatment of the roster character at index paid
// by the party member in slot.
func (c *Controller) templeQuote(index, slot int) (TempleQuote, error) {
	town := c.Game.Town
	roster := town.Roster.Characters
	if index < 0 || index >= len(roster) || roster[index] == nil {
		return TempleQuote{}, ErrNoSuchCharacter
	}
	if slot < 0 || slot >= len(town.Party.Members) || town.Party.Members[slot] == nil {
		return TempleQuote{}, ErrNoSuchPartyMember
	}
	ch, payer := roster[index], town.Party.Members[slot]
	q := TempleQuote{Patient: ch.Name, Donation: engine.TempleDonation(ch.Status, ch.Level), Payer: payer.Name, PayerGold: payer.Gold}
	for _, m := range town.Party.Members {
		if m != nil {
			q.PartyGold += m.Gold
		}
	}
	switch {
	case c.patientRefusal(ch) != "":
		q.Refusal = c.patientRefusal(ch)
	case payer.Status != engine.OK:
		q.Refusal = payerRefusal(payer)
	case payer.Gold >= q.Donation:
		q.Mode = "payer"
	case q.PartyGold >= q.Donation:
		q.Mode = "party"
		for i, m := range town.Party.Members {
			if m != nil && i != slot && m.Gold > 0 {
				q.Shares = append(q.Shares, TempleShare{Slot: i, Name: m.Name, Gold: m.Gold})
			}
		}
	default:
		q.Refusal = "CHEAP APOSTATES! OUT!"
	}
	return q, nil
}

// TempleQuote quotes a treatment as Treat would take it; ErrWrongState away
// from the Temple. It changes nothing and draws no random number.
func (c *Controller) TempleQuote(index, slot int) (TempleQuote, error) {
	if !c.atTemple() {
		return TempleQuote{}, ErrWrongState
	}
	return c.templeQuote(index, slot)
}

// TempleReceipt is what a treatment did: the patient before and after, the
// donation and who paid it, and the party's contribution if it made one.
type TempleReceipt struct {
	Patient      string `json:"patient"`
	Index        int    `json:"index"` // roster index
	Kind         string `json:"kind"`  // cure, raise or ashes
	StatusBefore string `json:"statusBefore"`
	StatusAfter  string `json:"statusAfter"`
	Healed       bool   `json:"healed"` // the treatment succeeded
	Donation     int    `json:"donation"`
	Payer        string `json:"payer"`
	PayerBefore  int    `json:"payerBefore"` // the payer's own gold, before any contribution
	PayerAfter   int    `json:"payerAfter"`
	// Shares are the other members' gold brought to the fund when the
	// party contributed (nil when the payer paid alone); Party is that
	// pooling, every member before and after it.
	Shares    []TempleShare `json:"shares"`
	Party     *GoldMove     `json:"party"`
	HPBefore  int           `json:"hpBefore"`
	HPAfter   int           `json:"hpAfter"`
	MaxHP     int           `json:"maxHp"`
	AgeBefore int           `json:"ageBefore"` // weeks
	AgeAfter  int           `json:"ageAfter"`
	Messages  []string      `json:"messages"` // the Temple's words
}

// Treat treats the roster character at index, the donation tithed by the
// party member in slot (0-based): one action, as the keys take it. When
// the payer is short and the party's gold together covers the donation,
// party says to let the party contribute (TempleQuote "party"): the
// pooling, the donation, the treatment and its roll then happen together;
// without it, a short payer is refused. A payer who has the donation pays
// alone, party or not.
func (c *Controller) Treat(index, slot int, party bool) (*TempleReceipt, error) {
	town := c.Game.Town
	if !c.atTemple() || town.TempleStep == engine.TempleRitual {
		return nil, ErrWrongState
	}
	q, err := c.templeQuote(index, slot)
	if err != nil {
		return nil, err
	}
	if q.Refusal != "" {
		return nil, &Refusal{Message: q.Refusal}
	}
	if q.Mode == "party" && !party {
		return nil, &Refusal{Message: "CHEAP APOSTATES! OUT!"}
	}
	ch, payer := town.Roster.Characters[index], town.Party.Members[slot]
	r := &TempleReceipt{
		Patient: ch.Name, Index: index, Kind: templeKind(ch.Status), StatusBefore: ch.Status.String(),
		Donation: q.Donation, Payer: payer.Name, PayerBefore: payer.Gold,
		HPBefore: ch.HP, AgeBefore: ch.Age,
	}
	if q.Mode == "party" {
		m, err := c.PoolGold(slot) // the entire party's gold into the fund (p.33)
		if err != nil {
			return nil, err
		}
		r.Shares, r.Party = q.Shares, m
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
