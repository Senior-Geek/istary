// ART-3B-2B: the Adventurer's Inn is illustrated (its picture in scenes.js,
// checked with the others in frameViews.test.js; its story in
// content1a.test.js). Its picture keeps its right edge: centred, the
// actions would cover the sign and, with a guest, Mistress Elspeth.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { scenes, illustrated } from './scenes.js'

test('the Inn’s picture, kept to its right edge, described for screen readers', () => {
  assert.ok(illustrated('inn'))
  assert.match(new URL(scenes.inn.src).pathname, /\/src\/assets\/scenes\/inn\.webp$/)
  assert.equal(scenes.inn.focus, '100% 50%')
  assert.match(scenes.inn.alt, /^The Adventurer's Inn: Mistress Elspeth, .*reception desk.*sign/)
  // The other pictures are as they were.
  assert.equal(scenes.castle.focus, '45% 50%')
  assert.equal(scenes.edge.focus, '50% 50%')
  assert.equal(scenes.tavern.focus, '50% 50%')
  // (ART-3B-3B) Boltac's has its own since: boltac.test.js.
  assert.equal(scenes.temple.src, null, 'temple: no picture yet')
})
