package main

// Spike P1b: structured combat events agree with the messages.
//
// The events are recorded by the same calls that write the messages, so
// these tests check the bookkeeping rather than re-derive anything: the
// events' lines rebuild the message list exactly, block for block; every
// party action is attributed to the member who acted; every party change an
// event reports is about a member its own block names (no silent change is
// exposed); and no block mixes two actions. The existing combat_trace golden,
// unchanged, shows that messages and random-number use did not change.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// narration lists the lines no action writes: surprise and the party's end.
var narration = map[string]bool{
	"THE MONSTERS SURPRISED YOU!":  true,
	"YOU SURPRISED THE MONSTERS!":  true,
	"** THE PARTY HAS PERISHED **": true,
}

// checkEvents compares c.Events() with c.Messages and returns the problems.
func checkEvents(c *engine.CombatState, party []*engine.Character) []string {
	var bad []string
	evs := c.Events()
	var rebuilt []string
	for i, e := range evs {
		if e.Start != len(rebuilt) {
			bad = append(bad, fmt.Sprintf("event %d starts at %d, expected %d", i, e.Start, len(rebuilt)))
		}
		rebuilt = append(rebuilt, e.Lines...)
		if e.End < len(c.Messages) {
			rebuilt = append(rebuilt, engine.ActionSeparator)
		}
		if e.Merged != 0 {
			bad = append(bad, fmt.Sprintf("event %d merges %d further actions: %q", i, e.Merged, e.Lines))
		}
		text := strings.Join(e.Lines, " ")
		// Blocks no action claims must be the known narration lines.
		if e.Actor.Side == engine.SideNone {
			for _, l := range e.Lines {
				if !narration[l] {
					bad = append(bad, fmt.Sprintf("event %d: unattributed line %q", i, l))
				}
			}
		} else if e.Part < 1 {
			bad = append(bad, fmt.Sprintf("event %d: actor without a part number", i))
		}
		if e.Actor.Side == engine.SideParty {
			if e.Actor.Member >= len(party) || party[e.Actor.Member].Name != e.Actor.Name {
				bad = append(bad, fmt.Sprintf("event %d actor %+v is not the member in that slot", i, e.Actor))
			}
			if len(e.Lines) > 0 && !strings.Contains(text, e.Actor.Name) {
				bad = append(bad, fmt.Sprintf("event %d actor %s not named in %q", i, e.Actor.Name, e.Lines))
			}
		}
		// A monster that acts is named, as the messages name it, in its own
		// block: an event never names an actor the player was not shown.
		if e.Actor.Side == engine.SideMonster && !strings.Contains(text, e.Actor.Name) {
			bad = append(bad, fmt.Sprintf("event %d monster actor %s not named in %q", i, e.Actor.Name, e.Lines))
		}
		for _, f := range e.Effects {
			if f.Who.Side == engine.SideParty && !strings.Contains(text, f.Who.Name) {
				bad = append(bad, fmt.Sprintf("event %d reports a change to %s that its block does not mention: %q %+v", i, f.Who.Name, e.Lines, f))
			}
			if f.Who.Side == engine.SideMonster && len(e.Lines) == 0 {
				bad = append(bad, fmt.Sprintf("event %d reports a monster change in an empty block", i))
			}
		}
	}
	if strings.Join(rebuilt, "\n") != strings.Join(c.Messages, "\n") {
		bad = append(bad, fmt.Sprintf("events rebuild %q, messages are %q", rebuilt, c.Messages))
	}
	return bad
}

// formatEvents renders events for the golden file.
func formatEvents(evs []engine.CombatEvent) string {
	ref := func(r engine.CombatRef) string {
		switch r.Side {
		case engine.SideParty:
			return fmt.Sprintf("party %d %s", r.Member+1, r.Name)
		case engine.SideMonster:
			if r.Member < 0 {
				return fmt.Sprintf("group %d", r.Group+1)
			}
			return fmt.Sprintf("group %d #%d %s", r.Group+1, r.Member+1, r.Name)
		}
		return "-"
	}
	var b strings.Builder
	for _, e := range evs {
		part := ""
		if e.Part > 1 {
			part = fmt.Sprintf(" (part %d)", e.Part)
		}
		if e.Silent {
			fmt.Fprintf(&b, "  [%s%s] silent block (no message; actor and target withheld)\n", e.Kind, part)
			continue
		}
		fmt.Fprintf(&b, "  [%s%s] actor: %s; target: %s\n", e.Kind, part, ref(e.Actor), ref(e.Target))
		for _, l := range e.Lines {
			fmt.Fprintf(&b, "      | %s\n", l)
		}
		for _, f := range e.Effects {
			if f.Who.Side == engine.SideParty {
				fmt.Fprintf(&b, "      > %s: HP %d→%d, %s→%s\n", ref(f.Who), f.HPBefore, f.HPAfter, f.StatusFrom, f.StatusTo)
			} else {
				fmt.Fprintf(&b, "      > %s: %s→%s\n", ref(f.Who), f.StatusFrom, f.StatusTo)
			}
		}
	}
	return b.String()
}

// Terminal-driven: the same policy and seeds as the combat_trace golden,
// plus more seeds. Checked at every round, including surprise rounds.
func TestCombatEventsMatchMessages(t *testing.T) {
	var golden strings.Builder
	rounds, events := 0, 0
	for s := int64(1); s <= 60; s++ {
		seed(s)
		g, d := partyInMaze(t)
		g.Combat = engine.NewCombat(g)
		g.Phase = engine.PhaseCombat
		logged := 0
		for step := 0; step < 5000 && g.Combat != nil && g.Phase == engine.PhaseCombat; step++ {
			c := g.Combat
			switch c.Phase {
			case engine.CombatFriendly:
				d.keys("F")
			case engine.CombatChoose:
				actor := c.CurrentActor
				d.keys("F")
				if c.SelectingGroup {
					d.keys("1")
				}
				if c.Phase == engine.CombatChoose && c.CurrentActor == actor {
					d.keys("P")
				}
			case engine.CombatConfirm:
				d.enter()
			case engine.CombatChest:
				d.keys("L")
			default:
				if c.Phase == engine.CombatExecute && c.Round != logged {
					logged = c.Round
					rounds++
					events += len(c.Events())
					for _, p := range checkEvents(c, g.Town.Party.Members) {
						t.Errorf("seed %d round %d: %s", s, c.Round, p)
					}
					if s >= 21 && s <= 23 {
						fmt.Fprintf(&golden, "== seed %d round %d\n%s", s, c.Round, formatEvents(c.Events()))
					}
				}
				combatTick(g)
			}
		}
	}
	t.Logf("%d rounds, %d events checked", rounds, events)
	checkGolden(t, "combat_events.golden", []byte(golden.String()))
}

// Engine-driven: orders set directly so that spells, dispel, use and run
// are exercised too (the terminal policy above only fights and parries).
func TestCombatEventsMatchMessagesWithSpells(t *testing.T) {
	kinds := map[engine.CombatEventKind]int{}
	for s := int64(1); s <= 80; s++ {
		seed(s)
		g, _ := partyInMaze(t)
		g.Combat = engine.NewCombat(g)
		g.Phase = engine.PhaseCombat
		c := g.Combat
		party := g.Town.Party.Members
		for _, m := range party {
			switch m.Class {
			case engine.Mage:
				m.SpellKnown[engine.SpellIndex["HALITO"]] = true
				m.SpellKnown[engine.SpellIndex["KATINO"]] = true
			case engine.Priest:
				m.SpellKnown[engine.SpellIndex["DIOS"]] = true
				m.SpellKnown[engine.SpellIndex["BADIOS"]] = true
			}
		}
		for r := 0; r < 8 && !c.AllMonstersDead() && !c.Fled; r++ {
			for i, m := range party {
				a := engine.PartyAction{Action: engine.ActionParry, TargetGroup: 0}
				switch {
				case m.Class == engine.Mage:
					a = engine.PartyAction{Action: engine.ActionSpell, SpellName: []string{"KATINO", "HALITO"}[r%2], TargetGroup: 0}
				case m.Class == engine.Priest && r%3 == 0:
					a = engine.PartyAction{Action: engine.ActionSpell, SpellName: "DIOS", TargetAlly: 0}
				case m.Class == engine.Priest && r%3 == 1:
					a = engine.PartyAction{Action: engine.ActionDispel, TargetGroup: 0}
				case m.Class == engine.Priest:
					a = engine.PartyAction{Action: engine.ActionSpell, SpellName: "BADIOS", TargetGroup: 0}
				case i < 3:
					a = engine.PartyAction{Action: engine.ActionFight, TargetGroup: 0}
				case m.Class == engine.Thief && r == 5:
					a = engine.PartyAction{Action: engine.ActionRun}
				case r == 2:
					a = engine.PartyAction{Action: engine.ActionUse, UseItemIdx: 0}
				}
				c.Actions[i] = a
			}
			c.ExecuteRound(g)
			for _, e := range c.Events() {
				kinds[e.Kind]++
			}
			for _, p := range checkEvents(c, party) {
				t.Errorf("seed %d round %d: %s", s, c.Round, p)
			}
			if c.AllMonstersDead() || c.Fled {
				break
			}
			c.CompactGroups()
			c.Round++
		}
	}
	t.Logf("event kinds seen: %v", kinds)
	for _, k := range []engine.CombatEventKind{engine.EventPartyFight, engine.EventPartySpell, engine.EventPartyParry, engine.EventPartyDispel, engine.EventPartyUse, engine.EventMonsterMelee} {
		if kinds[k] == 0 {
			t.Errorf("no %s event was exercised", k)
		}
	}
}
