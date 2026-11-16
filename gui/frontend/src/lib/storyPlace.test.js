// ART-3B-3C: the story box, entering a place, stands clear of its controls
// (lib/storyPlace.js); the player's place is kept.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { clearOf } from './storyPlace.js'

const win = { w: 980, h: 692 }
// Boltac's at 980x692, roughly: the head's buttons, the Actions panel's,
// the shop's tabs, and the party bar.
const head = [{ x: 664, y: 28, w: 110, h: 32 }, { x: 784, y: 28, w: 60, h: 32 }, { x: 854, y: 28, w: 100, h: 32 }]
const actions = [{ x: 677, y: 112, w: 106, h: 24 }, { x: 791, y: 112, w: 122, h: 24 }]
const tabs = [{ x: 29, y: 308, w: 51, h: 32 }, { x: 84, y: 308, w: 49, h: 32 }, { x: 137, y: 308, w: 74, h: 32 }, { x: 215, y: 308, w: 80, h: 32 }]
const party = [{ x: 24, y: 580, w: 306, h: 44 }, { x: 337, y: 580, w: 306, h: 44 }]
const controls = [...head, ...actions, ...tabs, ...party]

test('a story box over no control stays where the player left it', () => {
  const r = { x: 28, y: 88, w: 284, h: 184 }
  assert.equal(clearOf(r, controls, win), r)
})

test('over the tabs, it moves up the least, above them, its size kept', () => {
  const r = { x: 28, y: 150, w: 300, h: 200 }
  assert.deepEqual(clearOf(r, controls, win), { x: 28, y: 302 - 200, w: 300, h: 200 })
})

test('where moving would meet other controls, it is made shorter instead', () => {
  // As tall as the window allows above the tabs: moving up would reach the
  // window's margin, so it keeps its top and ends above them.
  const r = { x: 28, y: 90, w: 300, h: 360 }
  const got = clearOf(r, controls, win)
  assert.deepEqual(got, { x: 28, y: 90, w: 300, h: 302 - 90 })
})

test('up, down or shorter first, its left and width kept; aside only if there is no such place', () => {
  // Just over the first tab's right edge: one step aside would be less,
  // but up keeps it off the place's text.
  const r = { x: 60, y: 260, w: 200, h: 120 }
  const near = [{ x: 29, y: 308, w: 40, h: 32 }]
  assert.deepEqual(clearOf(r, near, win), { x: 60, y: 308 - 6 - 120, w: 200, h: 120 })
  // A control above and below leave no room up or down: aside.
  const boxed = [{ x: 100, y: 8, w: 40, h: 260 }, { x: 100, y: 330, w: 40, h: 354 }, { x: 100, y: 270, w: 40, h: 50 }]
  const s = { x: 60, y: 200, w: 200, h: 120 }
  assert.deepEqual(clearOf(s, boxed, win), { x: 100 + 40 + 6, y: 200, w: 200, h: 120 })
})

test('with no clear place, null: it opens where it opens for a new player', () => {
  const r = { x: 8, y: 8, w: 964, h: 676 }
  const everywhere = [{ x: 400, y: 300, w: 100, h: 50 }]
  assert.equal(clearOf(r, everywhere, win, { minH: 600 }), null)
})

test('the story box asks for a clear place as a place is entered, and never moves the player’s', () => {
  const src = readFileSync(new URL('./StoryBox.svelte', import.meta.url), 'utf8')
  assert.match(src, /onMount\(\(\) => \{ place\(\); avoid\(\) \}\)/)
  assert.match(src, /const at = \$derived\(fit\(clear === 'opening' \? auto : clear \?\? rect \?\? auto, win\)\)/)
  // Only the player's own moves are sent up (onmove); the clear place is the box's.
  assert.equal(src.match(/onmove\?\.\(/g).length, 1)
  assert.match(src, /function moved\(r\) \{[\s\S]*?clear = null\s+onmove\?\.\(/)
})
