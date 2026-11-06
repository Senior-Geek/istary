package engine

// Round transition and order binding: Istary corrections C1–C3
// (docs/design/m2-exploration/COMBAT-DESIGN-BASELINE.md §3, finding #15 in
// docs/WIZARDRY-COMPATIBILITY.md).
//
// The inherited engine re-sorted the party at the start of ExecuteRound,
// after orders had been entered by slot, so an order or an ally target could
// pass to whoever moved into the slot, and orders were never cleared. Now:
//   - C2: the re-sort happens in BeginNextRound, when a shown round gives way
//     to the next round's order entry, as Classic does at the end of the round;
//   - C3: BeginNextRound clears the previous round's orders;
//   - C1: orders and ally targets are bound to the characters they were given
//     for (the formation at order entry) and follow them if positions change.
// The sorting rules themselves are unchanged.

// BeginNextRound moves a combat whose round has been shown to the next
// round's order entry: groups are compacted, the round number advances, the
// party is re-sorted by status, and last round's orders are cleared. Call it
// only after the round's messages (and events) have been shown; until then
// they refer to the formation of the round they describe.
func (cs *CombatState) BeginNextRound(game *GameState) {
	cs.CompactGroups()
	cs.Round++
	// Pascal DSPPARTY (COMBAT3.TEXT lines 267-270): sort party by status.
	// Dead and incapacitated characters move back; members behind move up.
	// A cured character is not moved forward again.
	sortPartyByStatus(game.Town.Party.Members)
	for i := range cs.Actions {
		cs.Actions[i] = PartyAction{}
	}
	cs.orderFor = append(cs.orderFor[:0:0], game.Town.Party.Members...)
	cs.Phase = CombatChoose
	cs.Messages = nil
	cs.MessageIndex = 0
}

// bindOrders re-indexes cs.Actions from the formation the orders were given
// against (cs.orderFor) to the current formation, so each order, and the
// ally an order targets, stays with its character. When nothing has moved
// it changes nothing. Orders for characters no longer in the party are
// dropped.
func (cs *CombatState) bindOrders(party []*Character) {
	who := cs.orderFor
	if len(who) != len(cs.Actions) {
		// No record of the formation at order entry (a combat built
		// directly, as some tests do): the orders were given against the
		// current formation.
		cs.orderFor = append([]*Character(nil), party...)
		return
	}
	moved := len(party) != len(who)
	for i := range who {
		if i >= len(party) || party[i] != who[i] {
			moved = true
		}
	}
	if !moved {
		return
	}
	index := func(c *Character) int {
		for i, m := range party {
			if c != nil && m == c {
				return i
			}
		}
		return -1
	}
	bound := make([]PartyAction, len(party))
	for i, a := range cs.Actions {
		j := index(who[i])
		if j < 0 || j >= len(bound) {
			continue
		}
		if (a.Action == ActionSpell || a.Action == ActionUse) && a.TargetAlly >= 0 && a.TargetAlly < len(who) {
			if k := index(who[a.TargetAlly]); k >= 0 {
				a.TargetAlly = k
			}
		}
		bound[j] = a
	}
	cs.Actions = bound
	cs.orderFor = append([]*Character(nil), party...)
}
