// How a character's condition is shown (R-3): Go names it
// (app.ConditionLabel); this gives each the same treatment wherever it
// appears, on Inspect and on the map's stranded markers. The word is
// always shown; the mark and colour only add to it.
const looks = {
  OK: { mark: '', cls: 'ok' },
  ASLEEP: { mark: 'z', cls: 'asleep' },
  AFRAID: { mark: '!', cls: 'afraid' },
  PARALYZED: { mark: '≈', cls: 'paralyzed' },
  STONED: { mark: '▲', cls: 'stoned' },
  DEAD: { mark: '✝', cls: 'dead' },
  ASHES: { mark: '∴', cls: 'ashes' },
  LOST: { mark: '×', cls: 'lost' },
}

// condition is how a condition label is shown: its word, a mark and a
// class; a label Go adds later is shown as its word alone.
export function condition(label) {
  const l = looks[label] ?? { mark: '', cls: 'other' }
  return { label, mark: l.mark, cls: l.cls }
}
