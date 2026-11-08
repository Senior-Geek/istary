// Display helpers for the party bar (Castle-0). view.partyBar is Go's: the
// six positions of the party being played (the practice copies during
// practice), empty ones included. These only arrange and word it.

// formation splits the six positions into the front row (which can fight
// hand to hand) and the back row.
export function formation(bar = [], frontRow = 3) {
  return [
    { label: 'Front', note: 'fights hand to hand', slots: bar.slice(0, frontRow) },
    { label: 'Back', note: 'cannot reach the enemy', slots: bar.slice(frontRow) },
  ]
}

// condition words a member's state: the game's status, and poison.
export function condition(slot) {
  const words = [slot.status]
  if (slot.poisoned) words.push('POISONED')
  return words.join(' · ')
}

// unwell says whether the condition is anything but OK.
export function unwell(slot) {
  return slot.status !== 'OK' || slot.poisoned
}

// hpShare is the part of maximum HP left, from 0 to 1, for the HP bar.
export function hpShare(slot) {
  if (!slot.maxHp) return 0
  return Math.max(0, Math.min(1, slot.hp / slot.maxHp))
}

// canOpen says whether the member's character sheet can be opened: a
// character of the saved roster. Practice copies (index -1) are not.
export function canOpen(slot) {
  return !!slot.name && slot.index >= 0
}
