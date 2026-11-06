package app

// Combat view (M5c): a read-only description of combat for a front end,
// built from the controller's presentation state. It holds only values and
// copies, never pointers into the engine, and is suitable for serialization.
//
// What a View shows follows Classic's display timing and the player's
// knowledge:
//   - During playback the party, monster counts and monster names are the
//     round-start display (Pascal DSPPARTY/DSPENEMY run once, in CUTIL).
//     They change only when the round ends. Group names use the
//     identification frozen at round start (a LATUMAPIC cast mid-round
//     renames groups at the next round, not before its message).
//   - Events and messages are those released by Commit and Advance, no more.
//     Party effects carry the change in HP, not absolute HP, so silent
//     changes (round-start healing or poison) stay hidden until the round's
//     display update.
//   - Monster IDs, HP, initiative, victims of unshown actions, chest traps
//     and the round's outcome are not in a View.
// No timers, no random numbers.

import (
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// CombatView is the state of combat as the player may see it now.
type CombatView struct {
	Phase   string             `json:"phase"`           // opening, friendly, orders, confirm, playback, chest, victory, defeat
	Next    string             `json:"next"`            // what the front end does next; see the Next* constants
	Round   int                `json:"round"`           // round number, from 1
	Frozen  bool               `json:"frozen"`          // party and monster values are the round-start display (playback)
	Party   []CombatMemberView `json:"party"`           // in formation order
	Groups  []CombatGroupView  `json:"groups"`          // groups the player can see (count above zero)
	Actor   int                `json:"actor"`           // slot (from 1) being given an order; 0 when none
	Menu    []CombatOptionView `json:"menu"`            // the actor's orders, during order entry
	Orders  []CombatOrderView  `json:"orders"`          // orders given this round, during order entry and confirmation
	Events  []CombatEventView  `json:"events"`          // released events of the round being played back, in order
	Current int                `json:"current"`         // index into Events of the block being shown; -1 when none
	Lines   []string           `json:"lines"`           // the message lines on screen now
	Boons   []string           `json:"boons,omitempty"` // HAMAN's boons, once the block that offers them is on screen
	Chest   CombatChestView    `json:"chest"`           // the chest, in the chest phase only
	XP      int                `json:"xp,omitempty"`    // victory: experience for each survivor
	Gold    int                `json:"gold,omitempty"`  // victory: gold in each share
}

// CombatChestView is the chest as the party knows it. The true trap, the
// chances and the treasure are not in it; reports are what the party was
// told, right or wrong.
type CombatChestView struct {
	Present  bool               `json:"present"`
	Stage    string             `json:"stage,omitempty"` // menu, who-open, who-calfo, who-inspect, who-disarm, result
	Actions  []CombatOptionView `json:"actions,omitempty"`
	Members  []ChestMemberView  `json:"members,omitempty"`
	Reports  []ChestReportView  `json:"reports,omitempty"`
	Complete bool               `json:"complete"` // opened or left: the result leads to the rewards
}

// ChestMemberView says what one party member can do at the chest.
type ChestMemberView struct {
	Slot         int    `json:"slot"`
	Name         string `json:"name"`
	CanAct       bool   `json:"canAct"`
	Inspected    bool   `json:"inspected"`    // a second look gets "YOU ALREADY LOOKED!"
	CanCastCalfo bool   `json:"canCastCalfo"` // knows CALFO and has the spell points
}

// ChestReportView is one thing the party was told about the trap.
type ChestReportView struct {
	Slot   int    `json:"slot"`
	Member string `json:"member"`
	How    string `json:"how"`            // inspect, calfo, disarm, open
	Said   string `json:"said,omitempty"` // the trap named; empty when nothing was learned
	SetOff bool   `json:"setOff,omitempty"`
}

// What the front end does next.
const (
	NextStart       = "start"        // after the opening pause: StartCombat
	NextFriendly    = "friendly"     // fight or leave in peace (front end)
	NextOrders      = "orders"       // give the actor an order: SetOrder
	NextConfirm     = "confirm"      // Commit, or ReturnToOrders
	NextAdvance     = "advance"      // after the pause: Advance
	NextChoice      = "choice"       // HAMAN's boon: ChooseBoon
	NextEndRound    = "end-round"    // every block shown: EndRound
	NextChest       = "chest"        // choose a chest action and who acts: OpenChest, InspectChest, CastCalfo, DisarmChest, LeaveChest
	NextChestResult = "chest-result" // after the pause: AdvanceChest, then EndChestResult
	NextLeave       = "leave"        // after the victory or defeat pause (front end)
)

// CombatMemberView is one party member as displayed.
type CombatMemberView struct {
	Slot     int    `json:"slot"` // from 1; 1-3 are the front row
	Name     string `json:"name"`
	Class    string `json:"class"` // alignment and class, as in the party list ("G-FIG")
	AC       int    `json:"ac"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	Status   string `json:"status"`
	Poisoned bool   `json:"poisoned"`
	CanAct   bool   `json:"canAct"` // from the displayed status
}

// CombatGroupView is one visible monster group.
type CombatGroupView struct {
	Number int    `json:"number"` // from 1, as the player targets it
	Count  int    `json:"count"`
	Name   string `json:"name"` // the identified or unidentified name, singular or plural by Count
}

// CombatOptionView is one entry of the actor's order menu.
type CombatOptionView struct {
	Action    string `json:"action"` // fight, spell, parry, run, use, dispel
	Key       string `json:"key"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
	Note      string `json:"note,omitempty"`
	Target    string `json:"target"` // group, member, spell, item, none
}

// CombatOrderView is an order given this round.
type CombatOrderView struct {
	Slot   int    `json:"slot"` // from 1
	Member string `json:"member"`
	Action string `json:"action"`
	Group  int    `json:"group,omitempty"` // from 1
	Spell  string `json:"spell,omitempty"`
	Ally   string `json:"ally,omitempty"`
	Item   string `json:"item,omitempty"` // as the player knows it (unidentified items by their unknown name)
}

// CombatRefView identifies a participant in an event.
type CombatRefView struct {
	Side   string `json:"side"`             // party, monster, or empty
	Slot   int    `json:"slot,omitempty"`   // party member, from 1
	Group  int    `json:"group,omitempty"`  // monster group, from 1
	Member int    `json:"member,omitempty"` // monster within the group this round, from 1
	Name   string `json:"name,omitempty"`
}

// CombatEffectView is one change shown by an event.
type CombatEffectView struct {
	Who        CombatRefView `json:"who"`
	HPChange   int           `json:"hpChange,omitempty"` // party members only
	StatusFrom string        `json:"statusFrom"`
	StatusTo   string        `json:"statusTo"`
}

// CombatEventView is one released message block.
type CombatEventView struct {
	Kind    string             `json:"kind"`
	Actor   CombatRefView      `json:"actor"`
	Target  CombatRefView      `json:"target"`
	Lines   []string           `json:"lines"`
	Part    int                `json:"part,omitempty"`
	Silent  bool               `json:"silent,omitempty"`
	Effects []CombatEffectView `json:"effects,omitempty"`
}

var actionNames = map[engine.CombatAction]string{
	engine.ActionFight: "fight", engine.ActionSpell: "spell", engine.ActionParry: "parry",
	engine.ActionRun: "run", engine.ActionUse: "use", engine.ActionDispel: "dispel",
}

var targetNames = map[OrderTarget]string{
	TargetNothing: "none", TargetGroup: "group", TargetMember: "member", TargetSpell: "spell", TargetItem: "item",
}

// CombatView describes combat as the player may see it now.
func (c *Controller) CombatView() (CombatView, error) {
	g := c.Game
	cs := g.Combat
	if g.Phase != engine.PhaseCombat || cs == nil {
		return CombatView{}, ErrNotInCombat
	}
	v := CombatView{Round: cs.Round, Current: -1}
	switch cs.Phase {
	case engine.CombatInit:
		v.Phase, v.Next = "opening", NextStart
	case engine.CombatFriendly:
		v.Phase, v.Next = "friendly", NextFriendly
	case engine.CombatChoose:
		v.Phase, v.Next = "orders", NextOrders
	case engine.CombatConfirm:
		v.Phase, v.Next = "confirm", NextConfirm
	case engine.CombatExecute:
		v.Phase, v.Next = "playback", NextAdvance
		if atBoonPrompt(cs) {
			v.Next = NextChoice
			v.Boons = cs.HamanChoiceNames()
		} else if playbackDone(cs) {
			v.Next = NextEndRound
		}
	case engine.CombatChest:
		v.Phase, v.Next = "chest", NextChest
	case engine.CombatChestResult:
		v.Phase, v.Next = "chest", NextChestResult
	case engine.CombatVictory:
		v.Phase, v.Next = "victory", NextLeave
	case engine.CombatDefeat:
		v.Phase, v.Next = "defeat", NextLeave
	}
	v.Frozen = cs.Phase == engine.CombatExecute && cs.DisplayPartySnap != nil
	v.Party = c.partyView(cs, v.Frozen)
	v.Groups = c.groupsView(cs, v.Frozen)

	switch cs.Phase {
	case engine.CombatChoose:
		v.Lines = messageLines(cs.Messages) // a refusal such as "** CANT CAST **"
		party := g.Town.Party.Members
		if cs.CurrentActor >= 0 && cs.CurrentActor < len(party) {
			v.Actor = cs.CurrentActor + 1
			if menu, err := c.CombatOrderMenu(party[cs.CurrentActor]); err == nil {
				for _, o := range menu {
					v.Menu = append(v.Menu, CombatOptionView{Action: actionNames[o.Action], Key: string(o.Key),
						Available: o.Available, Reason: o.Reason, Note: o.Note, Target: targetNames[o.Target]})
				}
			}
		}
		v.Orders = c.ordersView(cs)
	case engine.CombatConfirm:
		v.Orders = c.ordersView(cs)
	case engine.CombatExecute:
		evs := c.RevealedEvents()
		for _, e := range evs {
			v.Events = append(v.Events, eventView(e))
		}
		if !playbackDone(cs) && len(v.Events) > 0 {
			v.Current = len(v.Events) - 1
			v.Lines = append([]string(nil), evs[v.Current].Lines...)
		}
	case engine.CombatChest, engine.CombatChestResult:
		v.Chest = c.chestView(cs)
		if cs.Phase == engine.CombatChestResult {
			v.Lines = blockLines(cs.Messages, cs.MessageIndex)
		}
	case engine.CombatVictory:
		v.Lines = messageLines(cs.Messages) // the rewards
		v.XP, v.Gold = cs.TotalXP, cs.TotalGold
	}
	return v, nil
}

// blockLines returns the lines of the block starting at from.
func blockLines(msgs []string, from int) []string {
	var out []string
	for i := from; i >= 0 && i < len(msgs) && msgs[i] != engine.ActionSeparator; i++ {
		out = append(out, msgs[i])
	}
	return out
}

// chestView describes the chest from what the party can know.
func (c *Controller) chestView(cs *engine.CombatState) CombatChestView {
	v := CombatChestView{Present: true, Complete: cs.ChestOpened || cs.ChestLeft}
	v.Stage = map[engine.ChestSubPhase]string{engine.ChestMenu: "menu", engine.ChestWhoOpen: "who-open",
		engine.ChestWhoCalfo: "who-calfo", engine.ChestWhoInspect: "who-inspect", engine.ChestWhoDisarm: "who-disarm"}[cs.ChestSubPhase]
	if cs.Phase == engine.CombatChestResult {
		v.Stage = "result"
	}
	calfo := engine.LookupSpell("CALFO")
	anyone, caster, uninspected := false, false, false
	for i, m := range c.Game.Town.Party.Members {
		if m == nil {
			continue
		}
		mv := ChestMemberView{Slot: i + 1, Name: m.Name, CanAct: CanGiveOrder(m), CanCastCalfo: calfo != nil && m.CanCastSpell(calfo)}
		if i < len(cs.ChestInspected) {
			mv.Inspected = cs.ChestInspected[i]
		}
		anyone = anyone || mv.CanAct
		caster = caster || (mv.CanAct && mv.CanCastCalfo)
		uninspected = uninspected || (mv.CanAct && !mv.Inspected)
		v.Members = append(v.Members, mv)
	}
	why := ""
	if !anyone {
		why = "no one can act"
	}
	act := func(action, key, note string) CombatOptionView {
		return CombatOptionView{Action: action, Key: key, Available: anyone, Reason: why, Note: note, Target: "member"}
	}
	calfoNote, inspectNote := "", ""
	if !caster {
		calfoNote = "no one here can cast CALFO"
	}
	if !uninspected {
		inspectNote = "everyone has already looked"
	}
	if v.Stage != "result" {
		v.Actions = []CombatOptionView{act("open", "O", ""), act("calfo", "C", calfoNote),
			{Action: "leave", Key: "L", Available: true, Target: "none"}, act("inspect", "I", inspectNote), act("disarm", "D", "")}
	}
	for _, r := range cs.ChestReports {
		v.Reports = append(v.Reports, ChestReportView{Slot: r.Member + 1, Member: r.Name, How: r.How, Said: r.Said, SetOff: r.SetOff})
	}
	return v
}

func messageLines(msgs []string) []string {
	var out []string
	for _, m := range msgs {
		if m != engine.ActionSeparator {
			out = append(out, m)
		}
	}
	return out
}

// partyView lists the party in formation order. During playback the values
// are the round-start snapshot (formation does not change until EndRound).
func (c *Controller) partyView(cs *engine.CombatState, frozen bool) []CombatMemberView {
	var out []CombatMemberView
	for i, m := range c.Game.Town.Party.Members {
		if m == nil {
			continue
		}
		mv := CombatMemberView{Slot: i + 1, Name: m.Name, Class: engine.AlignClass(m.Alignment, m.Class),
			AC: m.AC, HP: m.HP, MaxHP: m.MaxHP, Status: m.Status.String(), Poisoned: m.PoisonAmt > 0}
		st := m.Status
		if i < len(cs.PartyACMod) {
			mv.AC += cs.PartyACMod[i]
		}
		if frozen && i < len(cs.DisplayPartySnap) {
			snap := cs.DisplayPartySnap[i]
			mv.AC, mv.HP, mv.MaxHP, mv.Status, mv.Poisoned = snap.AC, snap.HP, snap.MaxHP, snap.Status.String(), snap.PoisonAmt > 0
			st = snap.Status
		}
		mv.CanAct = st == engine.OK || st == engine.Asleep || st == engine.Afraid
		out = append(out, mv)
	}
	return out
}

// groupsView lists the visible groups by the name the player knows. During
// playback the counts and identification are the round-start display.
func (c *Controller) groupsView(cs *engine.CombatState, frozen bool) []CombatGroupView {
	monsters := c.Game.Scenario.Monsters
	var out []CombatGroupView
	for i, grp := range cs.Groups {
		count, known := grp.AliveCount(), grp.Identified
		if frozen && i < len(cs.DisplayAliveCounts) && i < len(cs.DisplayIdentified) {
			count, known = cs.DisplayAliveCounts[i], cs.DisplayIdentified[i]
		}
		if count == 0 || grp.MonsterID < 0 || grp.MonsterID >= len(monsters) {
			continue
		}
		mon := &monsters[grp.MonsterID]
		name := mon.NameUnknown
		switch {
		case known && count == 1:
			name = mon.Name
		case known:
			name = mon.NamePlural
		case count != 1:
			name = mon.NameUnknownPlural
		}
		out = append(out, CombatGroupView{Number: i + 1, Count: count, Name: name})
	}
	return out
}

// ordersView lists the orders given this round, by member.
func (c *Controller) ordersView(cs *engine.CombatState) []CombatOrderView {
	party := c.Game.Town.Party.Members
	items := c.Game.Scenario.Items
	var out []CombatOrderView
	for i, a := range cs.Actions {
		if a.Action == engine.ActionNone || i >= len(party) || party[i] == nil {
			continue
		}
		m := party[i]
		o := CombatOrderView{Slot: i + 1, Member: m.Name, Action: actionNames[a.Action]}
		switch a.Action {
		case engine.ActionFight, engine.ActionDispel:
			o.Group = a.TargetGroup + 1
		case engine.ActionSpell:
			o.Spell = strings.ToUpper(a.SpellName)
			if sp := engine.LookupSpell(a.SpellName); sp != nil {
				switch sp.Target {
				case engine.TargetPartyMember:
					if a.TargetAlly >= 0 && a.TargetAlly < len(party) && party[a.TargetAlly] != nil {
						o.Ally = party[a.TargetAlly].Name
					}
				case engine.TargetMonsterGroup, engine.TargetSingleMonster:
					o.Group = a.TargetGroup + 1
				}
			}
		case engine.ActionUse:
			if a.UseItemIdx >= 0 && a.UseItemIdx < m.ItemCount {
				poss := m.Items[a.UseItemIdx]
				if poss.ItemIndex > 0 && poss.ItemIndex < len(items) {
					o.Item = items[poss.ItemIndex].NameUnknown
					if poss.Identified {
						o.Item = items[poss.ItemIndex].Name
					}
				}
			}
		}
		out = append(out, o)
	}
	return out
}

func refView(r engine.CombatRef) CombatRefView {
	switch r.Side {
	case engine.SideParty:
		return CombatRefView{Side: "party", Slot: r.Member + 1, Name: r.Name}
	case engine.SideMonster:
		v := CombatRefView{Side: "monster", Group: r.Group + 1, Name: r.Name}
		if r.Member >= 0 {
			v.Member = r.Member + 1
		}
		return v
	}
	return CombatRefView{}
}

// eventView copies an event. Party effects keep the HP change only.
func eventView(e engine.CombatEvent) CombatEventView {
	v := CombatEventView{Kind: string(e.Kind), Actor: refView(e.Actor), Target: refView(e.Target),
		Lines: append([]string{}, e.Lines...), Part: e.Part, Silent: e.Silent}
	for _, f := range e.Effects {
		ev := CombatEffectView{Who: refView(f.Who), StatusFrom: f.StatusFrom, StatusTo: f.StatusTo}
		if f.Who.Side == engine.SideParty {
			ev.HPChange = f.HPAfter - f.HPBefore
		}
		v.Effects = append(v.Effects, ev)
	}
	return v
}
