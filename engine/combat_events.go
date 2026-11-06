package engine

// Structured combat events (spike P1b, docs/design/m2-exploration/
// COMBAT-EVENT-REQUIREMENTS.md E1).
//
// One CombatEvent describes one message block: the messages between two
// ActionSeparators. Events are recorded by the same calls that write the
// messages (addMessage, endAction), so the two cannot disagree. The actor and
// target are noted where an action begins; effects are the changes to
// combatant state made while the block was open, found by comparing state
// when the block opened with state when it closed. Nothing is parsed from
// message text, and recording consumes no random numbers.
//
// What an event exposes: party members by slot and name; monsters by group,
// member index and the name the messages use (unidentified names stay
// unidentified); party HP and status changes; monster status changes. It
// does not expose monster IDs, monster HP, initiative, or victims of actions
// not yet shown. Events hold authoritative changes for a block; when to show
// them (Classic freezes the display until round end) is for the controller
// and the presentation to decide.

import "fmt"

// CombatEventKind names what happened in a block.
type CombatEventKind string

const (
	EventNarration     CombatEventKind = "narration" // no actor noted: surprise, flight, victory, chest, round end
	EventPartyFight    CombatEventKind = "party-fight"
	EventPartySpell    CombatEventKind = "party-spell"
	EventPartyUse      CombatEventKind = "party-use"
	EventPartyParry    CombatEventKind = "party-parry"
	EventPartyRun      CombatEventKind = "party-run"
	EventPartyDispel   CombatEventKind = "party-dispel"
	EventMonsterMelee  CombatEventKind = "monster-melee"
	EventMonsterSpell  CombatEventKind = "monster-spell"
	EventMonsterBreath CombatEventKind = "monster-breath"
	EventMonsterCall   CombatEventKind = "monster-call-for-help"
)

// CombatSide says which side a participant is on.
type CombatSide string

const (
	SideNone    CombatSide = ""
	SideParty   CombatSide = "party"
	SideMonster CombatSide = "monster"
)

// CombatRef identifies one participant at the time of the event. For the
// party, Member is the slot (0-5). For monsters, Group is the group index and
// Member the index within the group; both are positions for this round only
// (groups are compacted at round end). Name is the name the messages use.
type CombatRef struct {
	Side   CombatSide
	Group  int
	Member int
	Name   string
}

// NoRef is the empty reference.
var NoRef = CombatRef{Group: -1, Member: -1}

// CombatEffect is one change made during a block. Party effects carry HP and
// status; monster effects carry status only (monster HP is not known to the
// player).
type CombatEffect struct {
	Who        CombatRef
	HPBefore   int
	HPAfter    int
	StatusFrom string
	StatusTo   string
}

// CombatEvent is one message block.
type CombatEvent struct {
	Kind CombatEventKind
	// Actor is the party member or monster whose action the block shows.
	Actor CombatRef
	// Target is the group or member as ordered (party) or the victim the
	// monster attacks (monsters). The engine may hit something else if the
	// target is gone; Effects show what actually changed.
	Target CombatRef
	// Lines are the block's messages, exactly as in CombatState.Messages.
	Lines []string
	// Start and End index Messages: Lines == Messages[Start:End], and
	// Messages[End] is the separator when the block is closed.
	Start, End int
	// Part numbers the blocks of one action: 1 for the first. A monster's
	// melee writes one block per attack (a miss writes an empty block), and
	// some spells end a block early, so one action can span several blocks.
	Part int
	// Silent marks a block with no lines. The terminal shows it as a pause
	// with no text (a monster attack that misses writes one), so the player
	// is not told who acted or at whom: Actor, Target and Effects are
	// cleared. Kind and Part remain.
	Silent  bool
	Effects []CombatEffect
	// Merged counts further actions noted while this block was open (a block
	// whose action wrote no separator). Expected to be zero.
	Merged int
}

// eventLog is the recorder's state. Unexported and not saved.
type eventLog struct {
	party  []*Character
	events []CombatEvent
	open   *CombatEvent
	action *CombatEvent // the action whose later blocks inherit its actor
	snap   combatSnapshot
	mark   int // len(Messages) when the last event closed
	// withheld counts state changes made in silent blocks (none expected;
	// checked by TestCombatEventsSilentBlocksChangeNothing).
	withheld int
}

type combatSnapshot struct {
	party    map[*Character][2]int // HP, status
	monsters map[*CombatMonster]int
}

var monsterStatusNames = [...]string{"OK", "ASLEEP", "PARALYZED", "AFRAID", "SILENCED", "DEAD"}

func monsterStatusName(s int) string {
	if s >= 0 && s < len(monsterStatusNames) {
		return monsterStatusNames[s]
	}
	return fmt.Sprintf("STATUS %d", s)
}

// Events returns the events for the current Messages. Events whose messages
// have since been cleared are not returned.
func (cs *CombatState) Events() []CombatEvent {
	cs.syncEvents()
	out := append([]CombatEvent(nil), cs.ev.events...)
	if o := cs.ev.open; o != nil {
		e := *o
		e.End = len(cs.Messages)
		e.Lines = append([]string(nil), cs.Messages[e.Start:e.End]...)
		out = append(out, e)
	}
	return out
}

// syncEvents drops the log when Messages were cleared or replaced since the
// last event (ExecuteRound, the chest and the terminal all reset Messages).
func (cs *CombatState) syncEvents() {
	if len(cs.Messages) < cs.ev.mark || (cs.ev.open != nil && len(cs.Messages) < cs.ev.open.Start) {
		party := cs.ev.party
		cs.ev = eventLog{party: party}
	}
}

func (cs *CombatState) takeSnapshot() combatSnapshot {
	s := combatSnapshot{party: map[*Character][2]int{}, monsters: map[*CombatMonster]int{}}
	for _, m := range cs.ev.party {
		if m != nil {
			s.party[m] = [2]int{m.HP, int(m.Status)}
		}
	}
	for _, g := range cs.Groups {
		for _, cm := range g.Members {
			s.monsters[cm] = cm.Status
		}
	}
	return s
}

// openEvent starts a block if none is open.
func (cs *CombatState) openEvent() *CombatEvent {
	cs.syncEvents()
	if cs.ev.open == nil {
		cs.ev.open = &CombatEvent{Kind: EventNarration, Actor: NoRef, Target: NoRef, Start: len(cs.Messages)}
		if a := cs.ev.action; a != nil {
			a.Part++
			cs.ev.open.Kind, cs.ev.open.Actor, cs.ev.open.Target, cs.ev.open.Part = a.Kind, a.Actor, a.Target, a.Part
		}
		cs.ev.snap = cs.takeSnapshot()
	}
	return cs.ev.open
}

// beginEvent notes the action a block shows. Called where an action starts,
// before it changes anything.
func (cs *CombatState) beginEvent(kind CombatEventKind, actor, target CombatRef) {
	if cs.ev.open == nil {
		cs.ev.action = nil // a new action starts a new block
	}
	e := cs.openEvent()
	if e.Kind != EventNarration || e.Actor.Side != SideNone {
		e.Merged++
		return
	}
	e.Kind, e.Actor, e.Target, e.Part = kind, actor, target, 1
	a := *e
	cs.ev.action = &a
}

// noteAttack records the target of one attack in a multi-attack action.
func (cs *CombatState) noteAttack(target CombatRef) {
	if cs.ev.action != nil {
		cs.ev.action.Target = target
	}
	if e := cs.ev.open; e != nil && e.Merged == 0 && e.Actor.Side != SideNone {
		e.Target = target
	}
}

// endEventAction stops later blocks inheriting the last action (round end).
func (cs *CombatState) endEventAction() { cs.ev.action = nil }

// noteEvent refines the kind or target of the open block once the action's
// branch is known (a monster decides between melee, spell and breath).
func (cs *CombatState) noteEvent(kind CombatEventKind, target *CombatRef) {
	for _, e := range []*CombatEvent{cs.ev.open, cs.ev.action} {
		if e != nil && e.Merged == 0 {
			if kind != "" {
				e.Kind = kind
			}
			if target != nil {
				e.Target = *target
			}
		}
	}
}

// closeEvent ends the open block at the separator about to be written.
func (cs *CombatState) closeEvent() {
	e := cs.openEvent()
	e.End = len(cs.Messages)
	e.Lines = append([]string(nil), cs.Messages[e.Start:e.End]...)
	e.Effects = cs.diffSnapshot(cs.ev.snap)
	if len(e.Lines) == 0 {
		cs.ev.withheld += len(e.Effects)
		e.Silent, e.Actor, e.Target, e.Effects = true, NoRef, NoRef, nil
	}
	cs.ev.events = append(cs.ev.events, *e)
	cs.ev.open = nil
	cs.ev.mark = e.End + 1
}

func (cs *CombatState) diffSnapshot(before combatSnapshot) []CombatEffect {
	var out []CombatEffect
	for i, m := range cs.ev.party {
		if m == nil {
			continue
		}
		b, ok := before.party[m]
		if !ok || (b[0] == m.HP && b[1] == int(m.Status)) {
			continue
		}
		out = append(out, CombatEffect{
			Who:      CombatRef{Side: SideParty, Group: -1, Member: i, Name: m.Name},
			HPBefore: b[0], HPAfter: m.HP,
			StatusFrom: Status(b[1]).String(), StatusTo: m.Status.String(),
		})
	}
	for gi, g := range cs.Groups {
		for mi, cm := range g.Members {
			b, ok := before.monsters[cm]
			if !ok || b == cm.Status {
				continue
			}
			out = append(out, CombatEffect{
				Who:        CombatRef{Side: SideMonster, Group: gi, Member: mi},
				StatusFrom: monsterStatusName(b), StatusTo: monsterStatusName(cm.Status),
			})
		}
	}
	return out
}

// partyActionEvent maps an order to its event kind and target. Targets are
// as ordered: a monster group, or an ally slot, named after the member in
// that slot now (which, after a re-sort, may not be the member the order was
// given for; see engine/combat_characterization_test.go). A monster's spell
// target is never set: the pre-assigned victim is not shown to the player.
func partyActionEvent(a PartyAction, party []*Character) (CombatEventKind, CombatRef) {
	group := CombatRef{Side: SideMonster, Group: a.TargetGroup, Member: -1}
	switch a.Action {
	case ActionFight:
		return EventPartyFight, group
	case ActionDispel:
		return EventPartyDispel, group
	case ActionParry:
		return EventPartyParry, NoRef
	case ActionRun:
		return EventPartyRun, NoRef
	case ActionUse:
		return EventPartyUse, NoRef
	case ActionSpell:
		if sp := LookupSpell(a.SpellName); sp != nil {
			switch sp.Target {
			case TargetPartyMember:
				ally := CombatRef{Side: SideParty, Group: -1, Member: a.TargetAlly}
				if a.TargetAlly >= 0 && a.TargetAlly < len(party) && party[a.TargetAlly] != nil {
					ally.Name = party[a.TargetAlly].Name
				}
				return EventPartySpell, ally
			case TargetMonsterGroup, TargetSingleMonster:
				return EventPartySpell, group
			}
		}
		return EventPartySpell, NoRef
	}
	return EventNarration, NoRef
}
