// Display helpers for the Camp screen (M6b-3B). They only reshape what Go
// sent in view.camp (gui.CampPanel: the controller's app.CampView, the
// party's sheets, and why actions are unavailable); no rule is decided
// here. What is open, what a spell does and what an item is called are all
// Go's.

// Labels for the actions app.CampView lists, and for those it says are
// unavailable.
export const labels = {
  inspect: 'Inspect',
  reorder: 'Reorder party',
  'reorder-pick': 'Place',
  'equip-party': 'Equip everyone',
  disband: 'Disband',
  'leave-camp': 'Return to the maze',
  leave: 'Done',
  equip: 'Equip',
  'equip-choose': 'Equip',
  'equip-skip': 'Equip nothing',
  drop: 'Drop an item',
  'drop-item': 'Drop',
  'end-drop': 'Done',
  trade: 'Trade',
  'trade-with': 'Trade with',
  'trade-gold': 'Give gold',
  'trade-item': 'Give',
  'end-trade': 'End trade',
  'spell-books': 'Spell books',
  'spell-book': 'Open',
  'close-book': 'Back',
  cast: 'Cast a spell',
  'cast-on': 'Cast on',
  'cancel-cast': 'Cancel',
  identify: 'Identify',
  use: 'Use an item',
  'leave-dumapic': 'Close',
}

const nameOf = (camp, slot) => camp.party?.[slot]?.name ?? ''

// operation says what Camp is doing and waiting for, in words.
export function operation(camp) {
  const v = camp.view
  const who = nameOf(camp, v.member)
  switch (v.screen) {
    case 'menu':
      return 'Camp. Choose a party member to inspect, or an action.'
    case 'reorder':
      return `Reordering the party: who goes in place ${(v.placed?.length ?? 0) + 1}?`
    case 'inspect':
      return `Inspecting ${who}`
    case 'equip':
      return `Equipping ${who}: ${v.category}`
    case 'drop':
      return `${who}: which item to drop?`
    case 'trade':
      return `${who} trades: with whom?`
    case 'tradeGold':
      return `${who} trades with ${nameOf(camp, v.tradeWith)}: how much gold?`
    case 'tradeItem':
      return `${who} trades with ${nameOf(camp, v.tradeWith)}: which item?`
    case 'spellBooks':
      return `${who}'s spell books`
    case 'spellList':
      return `${who}'s ${v.spellBook} spells`
    case 'spellTarget':
      return `${who} casts ${v.spell}: on whom?`
    case 'dumapic':
      return `${who} casts DUMAPIC`
    case 'malor':
      return 'MALOR is cast in the terminal only.'
  }
  return ''
}

// The action a click on a party member performs on each screen.
const picks = { menu: 'inspect', reorder: 'reorder-pick', trade: 'trade-with', spellTarget: 'cast-on' }

// partyPick returns what clicking the party member in slot (0-based) does:
// { action } when it can be clicked, or null. The members already placed
// in a reorder and the character trading are not offered (Go refuses them).
export function partyPick(camp, slot) {
  const v = camp.view
  const action = picks[v.screen]
  if (!action || !v.actions?.includes(action)) return null
  if (action === 'reorder-pick' && v.placed?.includes(slot)) return null
  if (action === 'trade-with' && slot === v.member) return null
  return { action }
}

// back is the action that leaves the open screen (the Escape key), or null.
const backs = {
  inspect: 'leave', drop: 'end-drop', trade: 'end-trade', tradeGold: 'end-trade', tradeItem: 'end-trade',
  spellBooks: 'close-book', spellList: 'close-book', spellTarget: 'cancel-cast', dumapic: 'leave-dumapic',
}
export function back(camp) {
  const a = backs[camp.view.screen]
  return a && camp.view.actions?.includes(a) ? a : null
}

// buttons lists a screen's buttons in order: each open action, and each
// action Go says is unavailable, disabled, with Go's reason.
export function buttons(camp, order) {
  const v = camp.view
  const out = []
  for (const action of order) {
    const why = camp.unavailable?.find((u) => u.action === action)
    if (v.actions?.includes(action)) out.push({ action, label: labels[action], enabled: true, reason: '' })
    else if (why) out.push({ action, label: labels[action], enabled: false, reason: why.reason })
  }
  return out
}

export const menuOrder = ['reorder', 'equip-party', 'disband', 'leave-camp']
export const inspectOrder = ['equip', 'drop', 'trade', 'spell-books', 'cast', 'use', 'identify', 'leave']

// spellBook groups the known spells of one book ("mage" or "priest") by
// level, in the order Go lists them.
export function spellBook(spells = [], book) {
  const levels = []
  for (const sp of spells) {
    if (sp.book !== book) continue
    let l = levels.find((x) => x.level === sp.level)
    if (!l) levels.push((l = { level: sp.level, spells: [] }))
    l.spells.push(sp)
  }
  return levels
}

// slots formats spell slots left and their maximum, by level.
export function slots(cur = [], max = []) {
  return max.map((m, i) => (m ? `L${i + 1} ${cur[i]}/${m}` : '')).filter(Boolean).join('  ') || 'none'
}

// whyNot groups the spells that cannot be cast by Go's reason, in order:
// [{ reason, names }].
export function whyNot(spells = []) {
  const out = []
  for (const sp of spells) {
    if (sp.castable) continue
    let g = out.find((x) => x.reason === sp.reason)
    if (!g) out.push((g = { reason: sp.reason, names: [] }))
    g.names.push(sp.name)
  }
  return out
}
