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
