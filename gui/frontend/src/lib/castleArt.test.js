// ART-3C and ART-3D: the Training Grounds and the Temple of Cant are
// illustrated (their pictures in scenes.js, checked with the others in
// frameViews.test.js; their stories in content1a.test.js). Each principal
// figure stands at the right: the Training Grounds' picture keeps its right
// edge, the Temple's is a little right of centre.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { scenes, illustrated } from './scenes.js'

test('the Training Grounds’ picture, kept to its right edge, described for screen readers', () => {
  assert.ok(illustrated('training'))
  assert.match(new URL(scenes.training.src).pathname, /\/src\/assets\/scenes\/training\.webp$/)
  assert.equal(scenes.training.focus, '100% 50%')
  assert.match(scenes.training.alt, /^The Training Grounds: Sergeant Major Brannock Ironheel, a grey-bearded dwarf with a double-headed axe .*recruits at practice.*castle walls$/)
})

test('the Temple’s picture, a little right of centre, described for screen readers', () => {
  assert.ok(illustrated('temple'))
  assert.match(new URL(scenes.temple.src).pathname, /\/src\/assets\/scenes\/temple\.webp$/)
  assert.equal(scenes.temple.focus, '75% 50%')
  assert.match(scenes.temple.alt, /^The Temple of Cant: Father Caelion, a silver-haired elven priest .*stone bier.*censer.*stained-glass window of a hooded figure in darkness and another rising toward the light$/)
})

test('the other pictures are as they were; only Settings has none', () => {
  assert.deepEqual(Object.fromEntries(Object.entries(scenes).filter(([, s]) => s.src).map(([p, s]) => [p, s.focus])),
    { castle: '45% 50%', tavern: '50% 50%', inn: '100% 50%', trading: '75% 50%', temple: '75% 50%', edge: '50% 50%', training: '100% 50%' })
  assert.deepEqual(Object.keys(scenes).filter((p) => !scenes[p].src), ['settings', 'maze'])
})
