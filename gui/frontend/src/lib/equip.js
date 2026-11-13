// Display helpers for equipping in town (M8) on the character sheet. They
// only reshape what Go sent on each item (gui.ItemView: the controller's
// answers for equipping and taking off) and on the character (EquipNote:
// why equipping is closed now); no rule is decided here. A button is
// enabled exactly when Go gives no refusal, and a disabled one shows Go's
// reason.

// itemActions is what the sheet offers for one item: Equip or Take off
// (whichever applies), enabled or with Go's reason, or null when the
// character's equipment cannot be changed now (note) or the item is not
// something to equip.
export function itemActions(item, note) {
  if (note || !item.kind) return null
  if (item.equipped) {
    return { label: 'Take off', equip: false, enabled: !item.unequipRefusal, reason: item.unequipRefusal }
  }
  return { label: 'Equip', equip: true, enabled: !item.equipRefusal, reason: item.equipRefusal }
}

// kindLabel words an item's category for the sheet.
export function kindLabel(item) {
  return item.kind || 'carried'
}

// The sheet's two lists (UX-3B): what is worn or wielded, and the pack
// (everything carried and not equipped). They are one inventory, split by
// Go's equipped flag; each item is listed once, with its place in the
// character's items as pos (from 1), and the total counts both against
// the eight a character can carry.
export const CAPACITY = 8

export function wornAndPack(ch) {
  const items = (ch?.items ?? []).map((it, i) => ({ ...it, pos: i + 1 }))
  return {
    worn: items.filter((it) => it.equipped),
    pack: items.filter((it) => !it.equipped),
    carried: `${items.length} / ${CAPACITY}`,
  }
}

// removeAction is Remove for a worn item: Go's take-off, enabled exactly
// when Go gives no refusal (a cursed item says why). null when equipment
// cannot be changed now (note).
export function removeAction(item, note) {
  const a = itemActions(item, note)
  if (!a || a.equip) return null
  return { label: 'Remove', enabled: a.enabled, reason: a.reason }
}

// useAction is Use for an item in the pack. Only equipping is a use
// today: an item that is not something to equip has no Use at all (no
// button that would do nothing), and one that cannot be equipped by this
// character has a disabled Use with Go's reason. null when equipment
// cannot be changed now (note).
export function useAction(item, note) {
  const a = itemActions(item, note)
  if (!a || !a.equip) return null
  return { label: 'Use', enabled: a.enabled, reason: a.reason }
}
