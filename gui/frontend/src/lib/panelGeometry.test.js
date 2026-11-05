// Run with: npm test (node's built-in test runner; no extra dependencies).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { MARGIN, MIN_WIDTH, MIN_HEIGHT, fitSize, clampPosition, centered, constrain } from './panelGeometry.js'

const vp = { width: 980, height: 720 }

test('opens centered within the viewport', () => {
  const g = centered({ width: 560, height: 560 }, vp)
  assert.deepEqual(g, { width: 560, height: 560, left: 210, top: 80 })
})

test('a panel larger than the viewport is shrunk to fit and placed at the margin', () => {
  const g = centered({ width: 2000, height: 1500 }, vp)
  assert.deepEqual(g, { width: 980 - 2 * MARGIN, height: 720 - 2 * MARGIN, left: MARGIN, top: MARGIN })
})

test('minimum dimensions are enforced', () => {
  assert.deepEqual(fitSize({ width: 10, height: 10 }, vp), { width: MIN_WIDTH, height: MIN_HEIGHT })
})

test('the viewport wins over the minimum when it is smaller', () => {
  assert.deepEqual(fitSize({ width: 500, height: 500 }, { width: 300, height: 200 }),
    { width: 300 - 2 * MARGIN, height: 200 - 2 * MARGIN })
})

test('dragging cannot move the panel past any edge', () => {
  const size = { width: 400, height: 300 }
  const cases = [
    [{ left: -500, top: 100 }, { left: MARGIN, top: 100 }],          // off the left
    [{ left: 5000, top: 100 }, { left: 980 - 400 - MARGIN, top: 100 }], // off the right
    [{ left: 100, top: -500 }, { left: 100, top: MARGIN }],          // above: title bar stays visible
    [{ left: 100, top: 5000 }, { left: 100, top: 720 - 300 - MARGIN }], // below
  ]
  for (const [pos, want] of cases) {
    assert.deepEqual(clampPosition(pos, size, vp), want, JSON.stringify(pos))
  }
})

test('title bar and close button stay inside the viewport for any drag', () => {
  const size = { width: 560, height: 560 }
  for (let left = -2000; left <= 3000; left += 137) {
    for (let top = -2000; top <= 3000; top += 211) {
      const p = clampPosition({ left, top }, size, vp)
      assert.ok(p.left >= 0 && p.left + size.width <= vp.width, `left ${left} -> ${p.left}`)
      assert.ok(p.top >= 0 && p.top + 32 <= vp.height, `top ${top} -> ${p.top}`)
    }
  }
})

test('shrinking the window pulls the panel back inside and shrinks it if needed', () => {
  const g = { left: 500, top: 300, width: 560, height: 560 }
  const small = { width: 760, height: 560 }
  const c = constrain(g, small)
  assert.deepEqual(c, { width: 560, height: 560 - 2 * MARGIN, left: 760 - 560 - MARGIN, top: MARGIN })
  assert.ok(c.left + c.width <= small.width && c.top + c.height <= small.height)
})

test('a user resize past the edge keeps the panel inside', () => {
  // Resized wider than the room to its right: it moves left rather than overflow.
  const c = constrain({ left: 600, top: 100, width: 700, height: 400 }, vp)
  assert.equal(c.left + c.width, vp.width - MARGIN)
  assert.equal(c.width, 700)
})

test('constrain leaves an in-bounds panel alone', () => {
  const g = { left: 100, top: 100, width: 500, height: 400 }
  assert.deepEqual(constrain(g, vp), g)
})
