// The perspective projection (mazeProjection.js) and the Classic style
// (mazeStyle.js), over the views recorded from Go (testdata/maze-scenes.json,
// gui/maze_scene_fixture_test.go) and over every corridor shape.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { mazeScene } from './mazeScene.js'
import { project, perspective, far, near } from './mazeProjection.js'
import { classic, painted } from './mazeStyle.js'

const cases = JSON.parse(readFileSync(new URL('./testdata/maze-scenes.json', import.meta.url)))
const byId = Object.fromEntries(cases.map((c) => [c.id, c]))
const of = (id) => perspective(byId[id].dungeon)
const maze = (corridor, dark = false) => ({ phase: 'maze', corridor, dark })

// DrawMaze's values at each depth (render/dungeon.go, Pascal DRAWMAZE), in
// its 82-pixel picture moved to centre on 0: UL is the front wall's left
// and top, WALWIDTH its half-size, DOORWIDT and DOORFRAM the door's.
function drawmaze(depth) {
  let ul = 8, lr = 72, walWidth = 32, doorWidt = 16, doorFram = 8, walHeigh = 64
  for (let d = 0; d < depth; d++) {
    walWidth = Math.trunc(walWidth / 2)
    doorWidt = Math.trunc(walWidth / 2)
    walHeigh = walWidth * 2
    doorFram = Math.trunc(walWidth / 4)
    ul += walWidth
    lr -= walWidth
  }
  const c = (v) => v - 40
  return {
    front: { left: c(ul), right: c(ul + walHeigh), top: c(ul), bottom: c(ul + walHeigh) },
    // DRAWLEFT: the wall from UL - WALWIDTH to UL; the door's jambs at UL -
    // DOORFRAM and UL - DOORFRAM - DOORWIDT, its top from (UL - DOORFRAM, UL)
    // to (UL - DOORFRAM - DOORWIDT, UL - DOORWIDT).
    left: { near: c(ul - walWidth), far: c(ul) },
    leftDoor: [[c(ul - doorFram), c(ul)], [c(ul - doorFram - doorWidt), c(ul - doorWidt)]],
    // DRAWFRNT: jambs at UL + DOORFRAM and UL + WALWIDTH + DOORWIDT + DOORFRAM,
    // from LR up WALWIDTH + DOORWIDT + DOORFRAM.
    frontDoor: { left: c(ul + doorFram), right: c(ul + walWidth + doorWidt + doorFram), top: c(lr - (walWidth + doorWidt + doorFram)), bottom: c(lr) },
  }
}

const xs = (p) => p.points.map(([x]) => x)
const ys = (p) => p.points.map(([, y]) => y)
const box = (p) => ({ left: Math.min(...xs(p)), right: Math.max(...xs(p)), top: Math.min(...ys(p)), bottom: Math.max(...ys(p)) })

test('the geometry is DRAWMAZE’s: walls and doors where it draws them (depths 0–3)', () => {
  for (let d = 0; d < 4; d++) {
    const m = drawmaze(d)
    const corridor = [...Array(d).fill({ left: 'open', right: 'open', front: 'open' }), { left: 'door', right: 'wall', front: 'door' }]
    const p = perspective(maze(corridor))
    const get = (id) => p.find((x) => x.id === id)
    assert.deepEqual(box(get(`d${d}.front`)), m.front, `front ${d}`)
    assert.equal(far(d), m.front.right, `far(${d})`)
    const left = get(`d${d}.left`)
    assert.deepEqual([Math.min(...xs(left)), Math.max(...xs(left))], [m.left.near, m.left.far], `left ${d}`)
    assert.deepEqual(box(get(`d${d}.front.door`)), m.frontDoor, `front door ${d}`)
    // The side door: its far jamb's top and its near jamb's top.
    const door = get(`d${d}.left.door`).points
    assert.deepEqual([door[1], door[2]], m.leftDoor, `left door ${d}`)
  }
})

test('the stepped proportions: each square half the size of the one before', () => {
  assert.deepEqual([far(0), near(0), near(1), far(4)], [32, 64, 32, 2])
})

test('stable ids, painting order and nothing but geometry in each primitive', () => {
  for (const c of cases) {
    const p = perspective(c.dungeon)
    assert.equal(new Set(p.map((x) => x.id)).size, p.length, c.id)
    assert.deepEqual(p.map((x) => x.order), p.map((_, i) => i), c.id)
    for (const x of p) assert.deepEqual(Object.keys(x).sort(), ['depth', 'face', 'id', 'kind', 'lane', 'order', 'points', 'seen'], c.id)
    // Farthest first (the void farthest of all); a door straight after its wall.
    for (let i = 1; i < p.length; i++) assert.ok(p[i].depth <= p[i - 1].depth, `${c.id} ${p[i].id}`)
    p.forEach((x, i) => { if (x.kind === 'door') assert.equal(x.id, p[i - 1].id + '.door', c.id) })
  }
  assert.deepEqual(of('G10').filter((x) => x.kind !== 'floor' && x.kind !== 'ceiling').map((x) => x.id), [
    'd3.left', 'd3.right', 'd3.front', 'd3.front.door',
    'd2.leftAhead', 'd2.leftAhead.door', 'd2.right',
    'd1.left', 'd1.left.door', 'd1.right',
    'd0.left', 'd0.right', 'd0.right.door',
  ])
})

test('the view ends in a void where it ends unseen, and never in a made-up wall', () => {
  for (const c of cases) {
    const p = perspective(c.dungeon), s = mazeScene(c.dungeon)
    const voids = p.filter((x) => x.kind === 'void')
    if (c.ends === 'limit' || c.ends === 'dark') {
      assert.equal(voids.length, 1, c.id)
      assert.equal(p[0].kind, 'void', c.id)
      assert.deepEqual([voids[0].seen, voids[0].depth], [c.ends, s.end.depth], c.id)
      // On the plane where the unseen square begins, across the corridor and both sides.
      const h = near(s.end.depth)
      assert.deepEqual(box(voids[0]), { left: -3 * h, right: 3 * h, top: -h, bottom: h }, c.id)
    } else assert.equal(voids.length, 0, c.id)
    // Walls and doors only where the scene has them.
    const walls = new Set(s.surfaces.filter((x) => x.face === 'side' || x.face === 'front').map((x) => x.id))
    assert.deepEqual(p.filter((x) => x.kind === 'wall').map((x) => x.id).sort(), [...walls].sort(), c.id)
  }
  // Standing in darkness: only the void, over the whole view.
  assert.deepEqual(of('V09').map((x) => x.id), ['d0.void'])
})

test('a hidden door without light is drawn exactly as a wall; across the edge exactly as inside', () => {
  assert.deepEqual(of('V06'), of('V06-wall'))
  assert.deepEqual(of('G12-edge'), of('G12-inside'))
  assert.ok(of('V07').some((x) => x.id === 'd1.front.door'))
})

test('nothing past what Go sees is drawn (V04, V05, V08, V10, V11)', () => {
  for (const id of ['V04', 'V05', 'V08', 'V10', 'V11']) {
    const c = byId[id]
    const p = of(id).filter((x) => x.kind !== 'void')
    assert.ok(p.every((x) => x.depth < c.depths), id)
  }
  assert.ok(!of('V08').some((x) => x.kind === 'door'))
  assert.ok(!of('V10').some((x) => x.kind === 'door'))
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

// occlusionErrors paints the primitives in order over a grid of sample
// points and reports every point where the wall on top is not from the
// nearest square covering it, any floor or ceiling over a nearer wall, and
// any door not drawn after its wall or covered by a later one.
function occlusionErrors(prims) {
  const errs = []
  for (let gx = -40; gx < 40; gx += 1.9) {
    for (let gy = -40; gy < 40; gy += 1.9) {
      const p = [gx + 0.137, gy + 0.291]
      let top = null, nearest = Infinity
      for (const s of prims) {
        if (s.kind === 'door' || s.kind === 'void' || !inside(p, s.points)) continue
        top = s
        if (s.kind === 'wall') nearest = Math.min(nearest, s.depth)
      }
      if (top && nearest < Infinity && top.depth !== nearest) errs.push(`at ${p}: ${top.id} painted over square ${nearest}`)
    }
  }
  prims.forEach((s, i) => {
    if (s.kind !== 'door') return
    const mid = s.points.reduce(([x, y], [px, py]) => [x + px / 4, y + py / 4], [0, 0])
    if (!inside(mid, prims[i - 1].points)) errs.push(`${s.id} outside its wall`)
    if (prims.slice(i + 1).some((t) => t.kind !== 'void' && inside(mid, t.points))) errs.push(`${s.id} covered by a later primitive`)
  })
  return errs
}

const sides = [['wall'], ['door'], ['open', 'open'], ['open', 'wall'], ['open', 'door']]
const square = (l, r, front) => ({ left: l[0], leftAhead: l[1], right: r[0], rightAhead: r[1], front })

// corridors yields seeded random corridors of up to five squares.
function* corridors(count) {
  let seed = 11
  const rnd = (k) => ((seed = (seed * 1103515245 + 12345) % 2147483648), seed % k)
  for (let i = 0; i < count; i++) {
    const len = 1 + rnd(5)
    yield Array.from({ length: len }, (_, d) => square(sides[rnd(5)], sides[rnd(5)], d < len - 1 ? 'open' : ['open', 'wall', 'door'][rnd(3)]))
  }
}

test('nearer surfaces cover farther ones: every two-square corridor, random ones, every recorded case', () => {
  let n = 0
  for (const l0 of sides) for (const r0 of sides) for (const l1 of sides) for (const r1 of sides) {
    for (const f of ['open', 'wall', 'door']) {
      const c = [square(l0, r0, 'open'), square(l1, r1, f)]
      assert.deepEqual(occlusionErrors(perspective(maze(c))), [], JSON.stringify(c))
      n++
    }
  }
  assert.equal(n, 1875)
  for (const c of corridors(1500)) assert.deepEqual(occlusionErrors(perspective(maze(c))), [], JSON.stringify(c))
  for (const c of cases) assert.deepEqual(occlusionErrors(perspective(c.dungeon)), [], c.id)
})

test('the picture is left/right symmetric', () => {
  const mirror = (c) => c.map((s) => ({ ...s, left: s.right, leftAhead: s.rightAhead, right: s.left, rightAhead: s.leftAhead }))
  const flip = (id) => id.replace(/left|right/g, (m) => (m === 'left' ? 'right' : 'left'))
  const key = (prims, sign) => prims.map((p) => `${flip(p.id)}:${p.kind}:` + p.points.map(([x, y]) => `${sign * x},${y}`).sort().join(' ')).sort()
  const own = (prims) => prims.map((p) => `${p.id}:${p.kind}:` + p.points.map(([x, y]) => `${x},${y}`).sort().join(' ')).sort()
  for (const c of [...corridors(800), ...cases.map((k) => k.dungeon.corridor ?? [])]) {
    assert.deepEqual(key(perspective(maze(mirror(c))), -1), own(perspective(maze(c))))
  }
})

test('the projection is deterministic, and the same for the same scene', () => {
  for (const c of cases) {
    const s = mazeScene(c.dungeon)
    assert.deepEqual(project(s), project(structuredClone(s)), c.id)
    assert.equal(JSON.stringify(perspective(c.dungeon)), JSON.stringify(perspective(c.dungeon)), c.id)
  }
})

test('Classic paints only the walls, doors and void the projection gives, in its order', () => {
  assert.deepEqual(classic, { wall: 'wall', door: 'door', void: 'void', floor: null, ceiling: null })
  for (const c of cases) {
    const p = perspective(c.dungeon), drawn = painted(p)
    assert.deepEqual(drawn.map((x) => x.id), p.filter((x) => ['wall', 'door', 'void'].includes(x.kind)).map((x) => x.id), c.id)
    // Unchanged geometry: each drawn primitive is the projection's, plus its paint.
    for (const x of drawn) {
      const { paint, ...geometry } = x
      assert.deepEqual(geometry, p.find((y) => y.id === x.id), c.id)
      assert.equal(paint, x.kind, c.id)
    }
  }
})
