// Display helpers for the combat screen. They only reshape what Go sent in
// view.combat (app.CombatView); no rule is decided here. The Go side decides
// what is legal, what is known, and what happens.

// condition is the treatment a member's displayed status gets.
export function condition(member) {
  switch (member?.status) {
    case 'DEAD':
    case 'ASHED':
    case 'LOST':
      return 'dead' // translucent: still one of the six
    case 'PARALYZE':
    case 'STONED':
      return 'paralyzed'
    case 'ASLEEP':
      return 'asleep'
    case 'AFRAID':
      return 'afraid'
    default:
      return 'ok'
  }
}

// ranks splits the party into the front row (slots 1-3) and the back row.
export function ranks(party = []) {
  return {
    front: party.filter((m) => m.slot <= 3),
    back: party.filter((m) => m.slot > 3),
  }
}

// Each value of combat.next, what the front end does, and whether automatic
// pacing may do it. Steps that need a choice are never automatic.
export const steps = {
  start: { method: 'CombatStart', auto: true, label: 'Begin' },
  friendly: { method: null, auto: false, label: '' },
  orders: { method: null, auto: false, label: '' },
  confirm: { method: null, auto: false, label: '' },
  advance: { method: 'CombatAdvance', auto: true, label: 'Next' },
  choice: { method: null, auto: false, label: '' },
  'end-round': { method: 'CombatEndRound', auto: true, label: 'End round' },
  chest: { method: null, auto: false, label: '' },
  'chest-result': { method: 'ChestContinue', auto: true, label: 'Continue' },
  leave: { method: 'CombatLeave', auto: false, label: 'Leave' },
}

// step returns what to do for a view's next step.
export function step(combat) {
  return steps[combat?.next] ?? { method: null, auto: false, label: '' }
}

// Automatic pacing delays in milliseconds. The opening and the end of a
// chest result pause longer (Classic PAUSE2).
export const speeds = { slow: 2400, normal: 1500, fast: 700 }
export function delay(next, speed = 'normal') {
  const base = speeds[speed] ?? speeds.normal
  return next === 'start' || next === 'chest-result' ? base * 1.5 : base
}

// needsOrder lists the slots of members who can act and have no order yet.
export function needsOrder(combat) {
  const ordered = new Set((combat?.orders ?? []).map((o) => o.slot))
  return (combat?.party ?? []).filter((m) => m.canAct && !ordered.has(m.slot)).map((m) => m.slot)
}

// orderFor returns a member's order this round, if any.
export function orderFor(combat, slot) {
  return (combat?.orders ?? []).find((o) => o.slot === slot)
}

// orderLabel describes an order in a few words.
export function orderLabel(o) {
  if (!o) return ''
  const act = o.action[0].toUpperCase() + o.action.slice(1)
  switch (o.action) {
    case 'fight':
    case 'dispel':
      return `${act} → group ${o.group}`
    case 'spell':
      return o.ally ? `${o.spell} → ${o.ally}` : o.group ? `${o.spell} → group ${o.group}` : o.spell
    case 'use':
      return o.item ? `Use ${o.item}` : 'Use'
    default:
      return act
  }
}

// current returns the event on screen during playback, or null.
export function current(combat) {
  if (!combat || combat.current < 0) return null
  return combat.events?.[combat.current] ?? null
}

// highlight says which party slots and monster groups the event on screen
// marks: the actor in gold, the target in white. Silent blocks mark nobody.
export function highlight(event) {
  const h = { actorSlot: 0, actorGroup: 0, targetSlot: 0, targetGroup: 0 }
  if (!event || event.silent) return h
  if (event.actor?.side === 'party') h.actorSlot = event.actor.slot
  if (event.actor?.side === 'monster') h.actorGroup = event.actor.group
  if (event.target?.side === 'party') h.targetSlot = event.target.slot
  if (event.target?.side === 'monster') h.targetGroup = event.target.group
  return h
}

// chips lists the visible effects of the event on screen: HP changes and
// conditions for members (by slot), and monster conditions (by group).
export function chips(event) {
  const out = []
  for (const f of event?.effects ?? []) {
    if (f.who.side === 'party') {
      if (f.hpChange) out.push({ slot: f.who.slot, text: f.hpChange > 0 ? `+${f.hpChange}` : `${f.hpChange}`, kind: f.hpChange > 0 ? 'heal' : 'hurt' })
      if (f.statusTo !== f.statusFrom) out.push({ slot: f.who.slot, text: f.statusTo, kind: 'status' })
    } else if (f.who.side === 'monster' && f.statusTo !== f.statusFrom) {
      out.push({ group: f.who.group, text: f.statusTo === 'DEAD' ? 'slain' : f.statusTo.toLowerCase(), kind: 'monster' })
    }
  }
  return out
}

// targetKind says what an order option still needs once chosen: a group, a
// member, a spell, an item, or nothing.
export function targetKind(option, combat) {
  if (!option) return 'none'
  if (option.target === 'item' && !(combat?.items?.length)) return 'none'
  return option.target
}
