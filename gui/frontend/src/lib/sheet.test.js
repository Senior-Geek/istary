// The embedded character sheet's helpers (lib/sheet.js), over real views
// recorded from Go (testdata/frame-views.json, creation-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { subject, toggleLabel, available } from './sheet.js'

const frames = JSON.parse(readFileSync(new URL('./testdata/frame-views.json', import.meta.url)))
const creations = JSON.parse(readFileSync(new URL('./testdata/creation-views.json', import.meta.url)))

test('the toggle says what it does', () => {
  assert.equal(toggleLabel(false), 'Character Sheet')
  assert.equal(toggleLabel(true), 'Hide Character Sheet')
})

test('the sheet is offered in town, and not in the maze or while a character is created', () => {
  for (const v of frames) {
    assert.equal(available(v), v.screen === 'roster' && v.location !== 'maze', v.case)
  }
  for (const v of creations) assert.equal(available({ ...v, location: 'training' }), false, v.case)
  assert.equal(available(null), false)
})

test('the sheet is the chosen character’s, else the first party member’s, else the roster’s first', () => {
  const roster = [{ index: 0, name: 'A' }, { index: 1, name: 'B' }, { index: 2, name: 'C' }]
  const party = [{ slot: 1, name: '', index: -1 }, { slot: 2, name: 'C', index: 2 }]
  assert.equal(subject({ roster, partyBar: party }, 1).name, 'B')
  assert.equal(subject({ roster, partyBar: party }, -1).name, 'C')
  assert.equal(subject({ roster, partyBar: party }, 7).name, 'C', 'a character no longer on the roster')
  assert.equal(subject({ roster, partyBar: [] }, -1).name, 'A')
  assert.equal(subject({ roster: [], partyBar: [] }, -1), null)
  // Practice copies (index -1) have no sheet.
  assert.equal(subject({ roster, partyBar: [{ slot: 1, name: 'X', index: -1 }] }, -1).name, 'A')
})
