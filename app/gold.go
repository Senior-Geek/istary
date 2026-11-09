package app

import "github.com/Senior-Geek/istary/engine"

// Pooling and sharing the party's gold (I-3), shared by the town's
// services. The Player's Guide documents both:
//
//   - P)OOL GOLD "moves all the party's gold to the character" (inspect,
//     p.28); at Boltac's it "transfers the gold possessed by party members
//     to the person shopping" (p.32); at the Inn it "strips all the other
//     party members of all their gold and gives it to the character in the
//     Inn" (p.34).
//   - D)IVVY GOLD at the Tavern "pools the gold belonging to the party, and
//     then evenly splits it up between the party members" (p.24).
//
// The guide names no exception, so every party member takes part, whatever
// their condition (a dead member's gold is pooled too). It does not say
// where the remainder of an uneven split goes; Istary gives it out one
// gold piece each in party order, so no gold is lost. Neither draws a
// random number, and the party's total is the same before and after.

// GoldMove is what a pool or a split did: every party member's gold before
// and after, in party order.
type GoldMove struct {
	Names  []string `json:"names"`
	Before []int    `json:"before"`
	After  []int    `json:"after"`
	Total  int      `json:"total"` // the party's gold, the same before and after
	To     int      `json:"to"`    // the member pooled to (0-based), -1 for a split
}

// poolPlaces are where the guide offers pooling: the Tavern (by inspecting
// a member), Boltac's and the Inn. Sharing is offered at the Tavern.
var poolPlaces = map[engine.TownLocation]bool{engine.Tavern: true, engine.Trading: true, engine.Inn: true}

// goldReady reports that the party is in town at loc with no prompt open.
func (c *Controller) goldReady(places map[engine.TownLocation]bool) bool {
	g := c.Game
	return g.Phase == engine.PhaseTown && places[g.Town.Location] && g.Town.InputMode == engine.InputNone &&
		g.Town.Party.Size() > 0
}

func (c *Controller) goldMove(to int) *GoldMove {
	m := &GoldMove{To: to}
	for _, ch := range c.Game.Town.Party.Members {
		m.Names = append(m.Names, ch.Name)
		m.Before = append(m.Before, ch.Gold)
		m.Total += ch.Gold
	}
	return m
}

func (m *GoldMove) done(c *Controller) *GoldMove {
	for _, ch := range c.Game.Town.Party.Members {
		m.After = append(m.After, ch.Gold)
	}
	return m
}

// PoolGold gives the member in slot (0-based) every other member's gold.
func (c *Controller) PoolGold(slot int) (*GoldMove, error) {
	if !c.goldReady(poolPlaces) {
		return nil, ErrWrongState
	}
	members := c.Game.Town.Party.Members
	if slot < 0 || slot >= len(members) || members[slot] == nil {
		return nil, ErrNoSuchPartyMember
	}
	m := c.goldMove(slot)
	for i, ch := range members {
		if i != slot {
			members[slot].Gold += ch.Gold
			ch.Gold = 0
		}
	}
	return m.done(c), nil
}

// DivvyGold pools the party's gold and shares it evenly, the remainder one
// gold piece each from the first member on.
func (c *Controller) DivvyGold() (*GoldMove, error) {
	if !c.goldReady(map[engine.TownLocation]bool{engine.Tavern: true}) {
		return nil, ErrWrongState
	}
	members := c.Game.Town.Party.Members
	m := c.goldMove(-1)
	n := len(members)
	for i, ch := range members {
		ch.Gold = m.Total / n
		if i < m.Total%n {
			ch.Gold++
		}
	}
	return m.done(c), nil
}
