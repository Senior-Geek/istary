// The coordinate readout and the stranded nudge (UX-1), over Go's
// PositionView, and the views recorded from Go.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { where, whereTitle, strandedHint } from './position.js'
import { actions, keyAction } from './dungeon.js'

const entrance = { place: 'L1:R19:C0', level: 1, row: 19, col: 0, facing: 'N', stranded: 0 }

test('the readout is Go’s place and facing, nothing made up', () => {
  assert.equal(where(entrance), 'L1:R19:C0 · N')
  assert.equal(where({ ...entrance, place: 'L3:R0:C12', facing: 'W' }), 'L3:R0:C12 · W')
  assert.equal(whereTitle(entrance), 'Level 1, row 19, column 0, facing north')
  assert.equal(where(null), '')
  assert.equal(where(undefined), '')
  assert.match(whereTitle(null), /not on a square/)
})

test('the stranded nudge counts the party’s square, singular and plural', () => {
  assert.equal(strandedHint(entrance), '')
  assert.equal(strandedHint(null), '')
  assert.equal(strandedHint({ ...entrance, stranded: 1 }), '1 stranded adventurer here')
  assert.equal(strandedHint({ ...entrance, stranded: 2 }), '2 stranded adventurers here')
})

test('Inspect is now Inspect / Rescue, still on I', () => {
  assert.equal(actions.inspect.label, 'Inspect / Rescue')
  assert.deepEqual(actions.inspect.keys, ['i'])
  assert.equal(actions.inspect.hint, 'I')
  const maze = { actions: ['forward', 'left', 'right', 'kick', 'camp', 'inspect'] }
  assert.deepEqual(keyAction('i', maze), { action: 'inspect', button: 0 })
})

test('the dungeon screen shows the readout in its header, in the maze and at Camp', () => {
  const src = readFileSync(new URL('./DungeonScreen.svelte', import.meta.url), 'utf8')
  const header = src.slice(src.indexOf('<header class="bar">'), src.indexOf('</header>'))
  assert.match(header, /where\(view\.position\)/, 'the readout is in the header, for both phases')
  assert.match(src, /strandedHint\(view\.position\)/)
})
