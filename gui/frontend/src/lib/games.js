// Game selection (UX-3A), over Go's GamesView (view.games): six slots, each
// one independent adventure, and the save of an earlier Istary offered for
// import. What a slot holds, and whether it can be continued, is Go's;
// this only words it and lists the slot's choices.

/** @typedef {{ characters: number, party: string[] | null, expedition: string[] | null, depth: number, stranded: number }} Summary */
/** @typedef {Summary & { slot: number, state: string, saved: string, created: string, imported: boolean, problem: string }} Slot */
/** @typedef {Summary & { saved: string, problem: string }} Legacy */
/** @typedef {{ slots: Slot[], legacy: Legacy | null, replacing: number }} Games */

const plural = (n, one, many = one + 's') => `${n} ${n === 1 ? one : many}`

// summary words what a save holds: "3 characters", "Party: A, B", "Out
// in the maze: 6 on B1F", "2 stranded in the maze". Lines, in that order.
/** @param {Summary} s */
export function summary(s) {
  const lines = [s.characters ? plural(s.characters, 'character') : 'No characters yet']
  if (s.party?.length) lines.push(`Party: ${s.party.join(', ')}`)
  if (s.expedition?.length) lines.push(`Out in the maze: ${s.expedition.length} on B${s.depth}F`)
  if (s.stranded) lines.push(`${s.stranded} stranded in the maze`)
  return lines
}

// choices are a slot's buttons, in order: the first is the one to take.
// An empty slot begins a new game, or takes the earlier save; a game
// continues, or is replaced once confirmed; a damaged game can only be
// replaced. While its replacement awaits confirmation, a slot offers only
// that and keeping the game.
/** @param {Slot} slot @param {Games} games */
export function choices(slot, games) {
  if (games.replacing === slot.slot) {
    return [
      { label: 'Replace This Game', act: 'confirmReplace', kind: 'danger' },
      { label: 'Keep It', act: 'cancelReplace', kind: '' },
    ]
  }
  switch (slot.state) {
    case 'empty': {
      const out = [{ label: 'New Game', act: 'new', kind: 'primary' }]
      if (games.legacy && !games.legacy.problem) out.push({ label: 'Import Earlier Game Here', act: 'import', kind: '' })
      return out
    }
    case 'game':
      return [
        { label: 'Continue', act: 'continue', kind: 'primary' },
        { label: 'Replace with a New Game…', act: 'replace', kind: '' },
      ]
    default:
      return [{ label: 'Replace with a New Game…', act: 'replace', kind: '' }]
  }
}

// heading is a slot's title line: "Game 2", "Game 2 · imported".
/** @param {Slot} slot */
export function heading(slot) {
  return `Game ${slot.slot}${slot.imported ? ' · imported' : ''}`
}

// state words a slot's state for its badge.
/** @param {Slot} slot */
export function state(slot) {
  return { empty: 'Empty', game: 'Saved game', damaged: 'Cannot be opened' }[slot.state] ?? slot.state
}
