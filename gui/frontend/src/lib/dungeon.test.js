import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { scene, near, far, keyAction, actions } from './dungeon.js'

const wall = { left: 'wall', right: 'wall', front: 'open' }

test('each square is half the size of the one before', () => {
  assert.equal(far(0), 32)
  assert.equal(near(0), 64)
  assert.equal(near(1), far(0))
  assert.equal(far(4), 2)
})

test('a corridor of walls ending in a door', () => {
  const s = scene([wall, { left: 'wall', right: 'wall', front: 'door' }])
  // Farthest square first: square 1's walls and door, then square 0's walls.
  assert.deepEqual(s.map((x) => `${x.depth}${x.kind}`), ['1wall', '1wall', '1wall', '1door', '0wall', '0wall'])
  // The front door is drawn on the far plane of square 1.
  const door = s.find((x) => x.kind === 'door')
  assert.ok(door.points.every(([x, y]) => Math.abs(x) <= far(1) && Math.abs(y) <= far(1)))
})

test('an open side shows the wall ahead of the side square', () => {
  const s = scene([{ left: 'open', leftAhead: 'wall', right: 'open', rightAhead: 'door', front: 'wall' }])
  assert.deepEqual(s.map((x) => x.kind), ['wall', 'wall', 'door', 'wall'])
  const [l] = s
  assert.ok(l.points.every(([x]) => x <= -far(0)), 'left panel lies to the left')
  assert.deepEqual(scene([{ left: 'open', right: 'open', front: 'open' }]), [])
})

test('doors on side walls lie inside their wall', () => {
  const [w, d] = scene([{ left: 'door', right: 'open', front: 'open' }])
  assert.equal(d.kind, 'door')
  const xs = w.points.map(([x]) => x)
  assert.ok(d.points.every(([x]) => x >= Math.min(...xs) && x <= Math.max(...xs)))
  assert.deepEqual(scene(), [])
})

test('keys only perform open actions', () => {
  const maze = { actions: ['forward', 'left', 'right', 'kick', 'camp', 'inspect'] }
  assert.deepEqual(keyAction('ArrowUp', maze), { action: 'forward', button: 0 })
  assert.deepEqual(keyAction('F', maze), { action: 'forward', button: 0 })
  assert.deepEqual(keyAction('a', maze), { action: 'left', button: 0 })
  assert.equal(keyAction('y', maze), null)
  assert.deepEqual(keyAction('n', { actions: ['yes', 'no'] }), { action: 'no', button: 0 })
  assert.deepEqual(keyAction('Enter', { actions: ['continue'] }), { action: 'continue', button: 0 })
  assert.deepEqual(keyAction('l', { actions: ['pick-up', 'leave'] }), { action: 'leave', button: 0 })
  assert.equal(keyAction('f', { actions: ['continue'] }), null)
  assert.equal(keyAction('f', null), null)
})

test('at the buttons a letter presses that button and nothing else works', () => {
  const d = { actions: ['button'], buttons: ['A', 'B', 'C'] }
  assert.deepEqual(keyAction('b', d), { action: 'button', button: 2 })
  assert.deepEqual(keyAction('A', d), { action: 'button', button: 1 })
  assert.equal(keyAction('d', d), null)
  assert.equal(keyAction('Enter', d), null)
})

test('every action Go can list has a label', () => {
  for (const a of ['forward', 'left', 'right', 'kick', 'camp', 'inspect', 'continue', 'yes', 'no', 'pick-up', 'leave', 'leave-camp']) {
    assert.ok(actions[a]?.label, a)
  }
})

// inside reports whether (x, y) lies strictly inside a convex polygon.
function inside([x, y], pts) {
  let sign = 0
  for (let i = 0; i < pts.length; i++) {
    const [ax, ay] = pts[i], [bx, by] = pts[(i + 1) % pts.length]
    const c = (bx - ax) * (y - ay) - (by - ay) * (x - ax)
    if (Math.abs(c) < 1e-9) return false
    if (sign && Math.sign(c) !== sign) return false
    sign = Math.sign(c)
  }
  return true
}

// occlusionErrors paints the scene in order over a grid of sample points
// (offset so none lies on an edge) and reports every point where the wall on
// top is not from the nearest square covering it, and any door not drawn
// after its wall or covered by a later one.
function occlusionErrors(shapes) {
  const errs = []
  for (let gx = -40; gx < 40; gx += 1.9) {
    for (let gy = -40; gy < 40; gy += 1.9) {
      const p = [gx + 0.137, gy + 0.291]
      let top = null, nearest = Infinity
      for (const s of shapes) {
        if (s.kind !== 'wall' || !inside(p, s.points)) continue
        top = s
        nearest = Math.min(nearest, s.depth)
      }
      if (top && top.depth !== nearest) errs.push(`at ${p}: square ${top.depth} painted over square ${nearest}`)
    }
  }
  shapes.forEach((s, i) => {
    if (s.kind !== 'door') return
    const mid = s.points.reduce(([x, y], [px, py]) => [x + px / 4, y + py / 4], [0, 0])
    if (!shapes.slice(0, i).some((t) => t.kind === 'wall' && t.depth === s.depth && inside(mid, t.points))) errs.push(`door of square ${s.depth} drawn before its wall`)
    if (shapes.slice(i + 1).some((t) => t.kind === 'wall' && inside(mid, t.points))) errs.push(`door of square ${s.depth} covered by a later wall`)
  })
  return errs
}

// The side choices: a side wall or door, or an opening with what lies ahead
// of it.
const sides = [['wall'], ['door'], ['open', 'open'], ['open', 'wall'], ['open', 'door']]
const square = (l, r, front) => ({ left: l[0], leftAhead: l[1], right: r[0], rightAhead: r[1], front })

test('the reported overlaps: a far panel no longer shows over nearer walls', () => {
  const end = { left: 'wall', right: 'wall', front: 'wall' }
  const cases = [
    // A near right wall, then a right opening with a wall ahead of it.
    [wall, square(['wall'], ['open', 'wall'], 'open'), end],
    // Its mirror image.
    [wall, square(['open', 'wall'], ['wall'], 'open'), end],
    // A right opening with a wall ahead, then another one further on.
    [square(['wall'], ['open', 'wall'], 'open'), square(['wall'], ['open', 'wall'], 'wall')],
    // A side door near, a far panel behind it.
    [square(['door'], ['door'], 'open'), square(['open', 'door'], ['open', 'door'], 'open')],
  ]
  for (const c of cases) assert.deepEqual(occlusionErrors(scene(c)), [], JSON.stringify(c))
})

test('nearer walls cover farther ones in every two-square corridor', () => {
  let n = 0
  for (const l0 of sides) for (const r0 of sides) for (const l1 of sides) for (const r1 of sides) {
    for (const f of ['open', 'wall', 'door']) {
      const c = [square(l0, r0, 'open'), square(l1, r1, f)]
      assert.deepEqual(occlusionErrors(scene(c)), [], JSON.stringify(c))
      n++
    }
  }
  assert.equal(n, 1875)
})

// corridors yields seeded random corridors of up to five squares.
function* corridors(count) {
  let seed = 7
  const rnd = (k) => ((seed = (seed * 1103515245 + 12345) % 2147483648), seed % k)
  for (let i = 0; i < count; i++) {
    const len = 1 + rnd(5)
    yield Array.from({ length: len }, (_, d) => square(sides[rnd(5)], sides[rnd(5)], d < len - 1 ? 'open' : ['open', 'wall', 'door'][rnd(3)]))
  }
}

test('nearer walls cover farther ones in long corridors and every recorded view', () => {
  const views = JSON.parse(readFileSync(new URL('./testdata/dungeon-views.json', import.meta.url)))
  const recorded = views.filter((v) => v.corridor?.length).map((v) => v.corridor)
  assert.ok(recorded.length > 20)
  for (const c of [...corridors(3000), ...recorded]) assert.deepEqual(occlusionErrors(scene(c)), [], JSON.stringify(c))
})

test('the picture is left/right symmetric', () => {
  const mirror = (c) => c.map((s) => ({ ...s, left: s.right, leftAhead: s.rightAhead, right: s.left, rightAhead: s.leftAhead }))
  // Compare each square's shapes as a set: within a square the order of the
  // left and right sides is immaterial (they never overlap).
  const key = (shapes, sign) =>
    shapes.map((s) => `${s.depth}${s.kind}:` + s.points.map(([x, y]) => `${sign * x},${y}`).sort().join(' ')).sort()
  for (const c of corridors(3000)) {
    const a = scene(c), b = scene(mirror(c))
    assert.deepEqual(key(b, 1), key(a, -1))
    assert.deepEqual(b.map((s) => s.depth), a.map((s) => s.depth))
  }
})
