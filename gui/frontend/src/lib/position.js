// The party's coordinates (UX-1), over Go's PositionView (view.position):
// the canonical L:R:C and the facing, as the maps and the rescue records
// give them. Nothing here works out a square; with no position there is
// nothing to show.

/** @typedef {{ place: string, level: number, row: number, col: number, facing: string, stranded: number }} Position */

/** @type {Record<string, string>} */
const facings = { N: 'north', E: 'east', S: 'south', W: 'west' }

// where is the compact readout, "L1:R19:C0 · N", or '' with no position.
/** @param {Position | null | undefined} pos */
export function where(pos) {
  return pos ? `${pos.place} · ${pos.facing}` : ''
}

// whereTitle spells the readout out: "Level 1, row 19, column 0, facing north".
/** @param {Position | null | undefined} pos */
export function whereTitle(pos) {
  if (!pos) return 'The party is not on a square of the maze.'
  return `Level ${pos.level}, row ${pos.row}, column ${pos.col}, facing ${facings[pos.facing] ?? pos.facing}`
}

// strandedHint is the nudge on the party's square, "2 stranded adventurers
// here", or '' when no one lies there (by the square's map marker).
/** @param {Position | null | undefined} pos */
export function strandedHint(pos) {
  const n = pos?.stranded ?? 0
  if (n < 1) return ''
  return `${n} stranded adventurer${n === 1 ? '' : 's'} here`
}
