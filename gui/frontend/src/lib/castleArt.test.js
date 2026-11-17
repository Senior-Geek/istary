// ART-3C: the Training Grounds are illustrated (the picture in scenes.js,
// checked with the others in frameViews.test.js; the story in
// content1a.test.js). Brannock stands at the right: the picture keeps its
// right edge.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { scenes, illustrated } from './scenes.js'

test('the Training Grounds’ picture, kept to its right edge, described for screen readers', () => {
  assert.ok(illustrated('training'))
  assert.match(new URL(scenes.training.src).pathname, /\/src\/assets\/scenes\/training\.webp$/)
  assert.equal(scenes.training.focus, '100% 50%')
  assert.match(scenes.training.alt, /^The Training Grounds: Sergeant Major Brannock Ironheel, a grey-bearded dwarf with a double-headed axe .*recruits at practice.*castle walls$/)
})

test('the other pictures are as they were; the Temple and Settings have none', () => {
  assert.deepEqual(Object.fromEntries(Object.entries(scenes).filter(([, s]) => s.src).map(([p, s]) => [p, s.focus])),
    { castle: '45% 50%', tavern: '50% 50%', inn: '100% 50%', trading: '75% 50%', edge: '50% 50%', training: '100% 50%' })
  assert.deepEqual(Object.keys(scenes).filter((p) => !scenes[p].src), ['temple', 'settings', 'maze'])
})
