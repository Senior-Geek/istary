// The Enhanced style and the zoom presets (mazeStyle.js, M-ART-1A.2), over
// the views recorded from Go (testdata/maze-scenes.json) and over every
// corridor shape: one geometry, two styles, three frames.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { mazeScene } from './mazeScene.js'
import { perspective, far, near, viewBox } from './mazeProjection.js'
import { classic, enhanced, styles, painted, fog, shade, background, recede, zooms, defaultLook, segmentsPath } from './mazeStyle.js'

const cases = JSON.parse(readFileSync(new URL('./testdata/maze-scenes.json', import.meta.url)))
const byId = Object.fromEntries(cases.map((c) => [c.id, c]))
const maze = (corridor, dark = false) => ({ phase: 'maze', corridor, dark })
const geometry = ({ id, kind, seen, depth, lane, face, points, order }) => ({ id, kind, seen, depth, lane, face, points, order })

const sides = [['wall'], ['door'], ['open', 'open'], ['open', 'wall'], ['open', 'door']]
const square = (l, r, front) => ({ left: l[0], leftAhead: l[1], right: r[0], rightAhead: r[1], front })
function* corridors(count) {
  let seed = 23
  const rnd = (k) => ((seed = (seed * 1103515245 + 12345) % 2147483648), seed % k)
  for (let i = 0; i < count; i++) {
    const len = 1 + rnd(5)
    yield Array.from({ length: len }, (_, d) => square(sides[rnd(5)], sides[rnd(5)], d < len - 1 ? 'open' : ['open', 'wall', 'door'][rnd(3)]))
  }
}
const allViews = () => [...cases.map((c) => [c.id, c.dungeon]), ...[...corridors(600)].map((c, i) => [`random ${i}`, maze(c)])]

// within reports whether (x, y) lies inside or on a convex polygon.
function within([x, y], pts, eps = 1e-6) {
  let sign = 0
  for (let i = 0; i < pts.length; i++) {
    const [ax, ay] = pts[i], [bx, by] = pts[(i + 1) % pts.length]
    const len = Math.hypot(bx - ax, by - ay)
    const c = ((bx - ax) * (y - ay) - (by - ay) * (x - ax)) / len
    if (Math.abs(c) < eps) continue
    if (sign && Math.sign(c) !== sign) return false
    sign = Math.sign(c)
  }
  return true
}
const strictly = ([x, y], pts) => {
  let sign = 0
  for (let i = 0; i < pts.length; i++) {
    const [ax, ay] = pts[i], [bx, by] = pts[(i + 1) % pts.length]
    const c = (bx - ax) * (y - ay) - (by - ay) * (x - ax)
    if (Math.abs(c) < 1e-9 || (sign && Math.sign(c) !== sign)) return false
    sign = Math.sign(c)
  }
  return true
}

test('Classic and Enhanced draw the same walls, doors and void, at the same coordinates, in the same order', () => {
  for (const [id, dungeon] of allViews()) {
    const p = perspective(dungeon)
    const c = painted(p, classic), e = painted(p, enhanced)
    const solid = (list) => list.filter((x) => ['wall', 'door', 'void'].includes(x.kind)).map(geometry)
    assert.deepEqual(solid(e), solid(c), id)
    // Enhanced paints every primitive the projection gives, and nothing else:
    // floors and ceilings are the projection's, unmoved.
    assert.deepEqual(e.map(geometry), p, id)
    for (const x of e) assert.equal(x.paint, x.kind, id)
  }
})

test('Enhanced adds no maze: its details stay on their own surface, and nothing passes the view’s end', () => {
  for (const [id, dungeon] of allViews()) {
    const s = mazeScene(dungeon)
    const e = painted(perspective(dungeon), enhanced)
    for (const x of e) {
      for (const dt of x.details) {
        assert.ok(['mortar', 'flag', 'plank', 'band'].includes(dt.kind), `${id} ${x.id} ${dt.kind}`)
        assert.ok(dt.segments.length > 0, `${id} ${x.id}`)
        // Masonry, flagstones, planks and bands: clipped to the polygon they dress.
        for (const seg of dt.segments) for (const pt of seg) assert.ok(within(pt, x.points), `${id} ${x.id} ${dt.kind} ${pt}`)
      }
      // Nothing at or past where the view ends unseen, but the void.
      if (x.kind !== 'void') assert.ok(x.depth < s.depths, `${id} ${x.id}`)
      if (x.kind === 'void') assert.deepEqual([x.details, x.tone], [[], null], id)
    }
  }
  // Standing in darkness: the void alone; past a dark square, nothing.
  assert.deepEqual(painted(perspective(byId.V09.dungeon), enhanced).map((x) => x.id), ['d0.void'])
  for (const id of ['V04', 'V05', 'V08', 'V10', 'V11']) {
    const e = painted(perspective(byId[id].dungeon), enhanced)
    assert.ok(e.every((x) => x.kind === 'void' || x.depth < byId[id].depths), id)
  }
  assert.ok(!painted(perspective(byId.V08.dungeon), enhanced).some((x) => x.kind === 'door'))
})

test('every opening stays open: nothing of the corridor’s own is drawn across it', () => {
  let n = 0
  for (const [id, dungeon] of allViews()) {
    const s = mazeScene(dungeon)
    const e = painted(perspective(dungeon), enhanced)
    for (const o of s.openings) {
      // Points across where that side's wall would stand, between the planes.
      const f = far(o.depth), nr = near(o.depth)
      for (const k of [0.15, 0.5, 0.85]) for (const v of [-0.8, -0.3, 0, 0.3, 0.8]) {
        const x = f + (nr - f) * k, pt = [o.lane * x, v * x]
        for (const q of e) {
          if (q.kind === 'void' || q.depth > o.depth) continue
          // Only what is seen through it (the square beside, the wall
          // ahead of it); details stay on their surface (tested above).
          if (strictly(pt, q.points)) assert.equal(q.lane, o.lane, `${id}: ${q.id} across the opening ${o.id}`)
        }
        n++
      }
    }
  }
  assert.ok(n > 1000, `${n} points`)
})

// topAt paints the drawn list over a point and gives what is on top
// (details are on their own surface, so their surface stands for them).
const topAt = (drawn, pt) => drawn.reduce((top, x) => (strictly(pt, x.points) ? x : top), null)

test('in Enhanced too, nearer surfaces cover farther ones, over the widest frame', () => {
  const [x0, , w] = zooms.wide.split(' ').map(Number)
  const views = [...cases.map((c) => [c.id, c.dungeon]), ...[...corridors(300)].map((c, i) => [`random ${i}`, maze(c)])]
  for (const [id, dungeon] of views) {
    const e = painted(perspective(dungeon), enhanced)
    for (let gx = x0; gx < x0 + w; gx += 2.3) {
      for (let gy = x0; gy < x0 + w; gy += 2.3) {
        const pt = [gx + 0.137, gy + 0.291]
        const top = topAt(e, pt)
        if (!top || top.kind === 'void') continue
        // The nearest surface of any kind covering the point is on top.
        const nearest = Math.min(...e.filter((x) => x.kind !== 'void' && x.kind !== 'door' && strictly(pt, x.points)).map((x) => x.depth))
        assert.equal(top.depth, nearest, `${id} at ${pt}: ${top.id}`)
      }
    }
  }
})

test('a hidden door without light is a wall in Enhanced too; across the edge as inside', () => {
  const of = (id) => painted(perspective(byId[id].dungeon), enhanced)
  assert.deepEqual(of('V06'), of('V06-wall'))
  assert.deepEqual(of('G12-edge'), of('G12-inside'))
  assert.ok(of('V07').some((x) => x.id === 'd1.front.door' && x.paint === 'door'))
  assert.ok(!of('V06').some((x) => x.kind === 'door'))
})

test('depth shading: darker with distance, continuous from square to square, two sides apart', () => {
  assert.equal(fog(0), 1)
  for (let t = 0; t < 7; t++) assert.ok(fog(t + 1) < fog(t))
  const lum = (h) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16)).reduce((a, b) => a + b)
  for (const [id, dungeon] of allViews()) {
    const e = painted(perspective(dungeon), enhanced)
    const get = (sid) => e.find((x) => x.id === sid)
    for (const x of e) {
      if (x.tone) {
        // From its near plane to its far plane, darker.
        assert.ok(lum(x.tone.to) < lum(x.tone.from), `${id} ${x.id}`)
        assert.equal(x.fill, `url(#mz-${x.id})`, id)
      }
      // A receding surface's far shade is the next square's near shade.
      const next = get(`d${x.depth + 1}.${x.id.split('.')[1]}`)
      if (x.tone && next?.tone && x.kind === next.kind) assert.equal(x.tone.to, next.tone.from, `${id} ${x.id}`)
    }
    // Front walls at the far plane of their square; the farther, the darker.
    const fronts = e.filter((x) => x.kind === 'wall' && x.face === 'front')
    for (const a of fronts) for (const b of fronts) if (a.depth < b.depth) assert.ok(lum(a.fill) > lum(b.fill), `${id} ${a.id} ${b.id}`)
    // The two sides, at the same depth, a shade apart; the ceiling darker than the floor.
    for (let d = 0; d < 6; d++) {
      const [l, r] = [get(`d${d}.left`), get(`d${d}.right`)]
      if (l?.kind === 'wall' && r?.kind === 'wall') assert.notEqual(l.tone.from, r.tone.from, id)
      const [fl, ce] = [get(`d${d}.floor`), get(`d${d}.ceiling`)]
      if (fl) assert.ok(lum(ce.tone.from) < lum(fl.tone.from), id)
    }
  }
  // Where the view ends unseen, its last square fades into the void, and
  // only that square; a view ending in a wall does not fade.
  const sq = { left: 'wall', right: 'wall', front: 'open' }
  const ending = (front) => painted(perspective(maze([sq, { ...sq, front }])), enhanced)
  const [unseen, walled] = [ending('open'), ending('wall')]
  const get = (list, id) => list.find((x) => x.id === id)
  for (const id of ['d0.left', 'd0.right', 'd0.floor', 'd0.ceiling']) assert.deepEqual(get(unseen, id).tone, get(walled, id).tone, id)
  for (const id of ['d1.left', 'd1.floor']) {
    assert.equal(get(unseen, id).tone.from, get(walled, id).tone.from, id)
    assert.ok(lum(get(unseen, id).tone.to) < lum(get(walled, id).tone.to), id)
  }
  // No wall or door seen fades away: each stays apart from the darkness.
  // (Floors and ceilings tell nothing, and the ceiling is dark by design.)
  for (const [id, dungeon] of allViews()) {
    for (const x of painted(perspective(dungeon), enhanced)) {
      if (x.kind !== 'wall' && x.kind !== 'door') continue
      // (A door is seen by its frame.)
      const darkest = x.kind === 'door' ? x.edge : x.tone ? x.tone.to : x.fill
      assert.ok(lum(darkest) - lum('#0d0c0b') >= 10, `${id} ${x.id} ${darkest}`)
    }
  }
  assert.ok(Math.abs(fog(2, 2) - 0.66 ** 2 * 0.45) < 1e-12)
  assert.equal(fog(1, 2), 0.66)
  // The void is the stage's background: the view ends in plain darkness.
  assert.equal(painted(perspective(byId.V01.dungeon), enhanced)[0].fill, '#0d0c0b')
  assert.deepEqual(background, [13, 12, 11])
  assert.equal(shade([200, 200, 200], 0), '#c8c8c8')
})

test('floors and ceilings: the projection’s own, across the corridor and through each opening, meeting the walls', () => {
  for (const [id, dungeon] of allViews()) {
    const s = mazeScene(dungeon), e = painted(perspective(dungeon), enhanced)
    const ids = new Set(e.filter((x) => x.kind === 'floor' || x.kind === 'ceiling').map((x) => x.id))
    for (let d = 0; d < s.depths; d++) {
      assert.ok(ids.has(`d${d}.floor`) && ids.has(`d${d}.ceiling`), `${id} d${d}`)
    }
    for (const o of s.openings) {
      const side = o.lane < 0 ? 'left' : 'right'
      assert.ok(ids.has(`d${o.depth}.${side}Floor`) && ids.has(`d${o.depth}.${side}Ceiling`), `${id} ${o.id}`)
    }
    assert.equal(ids.size, s.surfaces.filter((x) => x.face === 'floor' || x.face === 'ceiling').length, id)
    // The floor of each square shares its edges with the side walls and the
    // next square's floor: no gap between them.
    const get = (sid) => e.find((x) => x.id === sid)
    for (let d = 0; d < s.depths; d++) {
      const fl = get(`d${d}.floor`), l = get(`d${d}.left`)
      if (l?.face === 'side') assert.deepEqual([fl.points[0], fl.points[3]], [l.points[3], l.points[2]], `${id} d${d}`)
      const next = get(`d${d + 1}.floor`)
      if (next) assert.deepEqual([fl.points[3], fl.points[2]], [next.points[0], next.points[1]], `${id} d${d}`)
    }
  }
})

test('masonry and flagstones follow the perspective, and thin out with distance', () => {
  assert.deepEqual([recede(0), recede(1), recede(0.5)], [0, 1, 2 / 3])
  const e = painted(perspective(maze([{ left: 'wall', right: 'wall', front: 'open' }, { left: 'wall', right: 'wall', front: 'open' },
    { left: 'wall', right: 'wall', front: 'open' }, { left: 'wall', right: 'wall', front: 'open' }, { left: 'wall', right: 'wall', front: 'wall' }])), enhanced)
  const get = (sid) => e.find((x) => x.id === sid)
  const count = (sid, kind) => get(sid).details.find((x) => x.kind === kind)?.segments.length ?? 0
  // Courses and joints near, courses alone farther, none at depth 4.
  assert.deepEqual([0, 1, 2, 3, 4].map((d) => count(`d${d}.left`, 'mortar')), [3 + 8, 3 + 8, 3 + 8, 3, 0])
  assert.deepEqual([0, 1, 2, 3, 4].map((d) => count(`d${d}.floor`, 'flag')), [5, 5, 5, 5, 0])
  assert.equal(count('d4.front', 'mortar'), 0)
  // A course on the left wall at depth 1 runs to the vanishing point: from
  // a quarter down the near edge to a quarter down the far edge.
  const course = get('d1.left').details[0].segments[0]
  assert.deepEqual(course, [[-32, -16], [-16, -8]])
  // The joint halfway along the square is two-thirds of the way across it
  // on the picture (the far plane is twice as far away).
  const joint = get('d1.left').details[0].segments[4]
  assert.deepEqual(joint[0].map((v) => +v.toFixed(6)), [-32 + 16 * (2 / 3), -(32 - 16 * (2 / 3))].map((v) => +v.toFixed(6)))
  // Courses meet from square to square.
  const at = (sid, k) => get(sid).details[0].segments[k]
  for (let d = 0; d < 3; d++) for (let k = 0; k < 3; k++) assert.deepEqual(at(`d${d}.left`, k)[1], at(`d${d + 1}.left`, k)[0], `d${d} course ${k}`)
  assert.equal(segmentsPath([[[0, 1], [2.0004, -3]]]), 'M0 1L2 -3')
})

const hexMid = (a, b) => '#' + [1, 3, 5].map((i) => Math.round((parseInt(a.slice(i, i + 2), 16) + parseInt(b.slice(i, i + 2), 16)) / 2).toString(16).padStart(2, '0')).join('')

test('every door the same: wood in a stone frame, with planks and bands, from Go’s ‘door’ alone', () => {
  const lum = (h) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16)).reduce((a, b) => a + b)
  const recipe = (x) => JSON.stringify({ ...x, id: '', order: 0 })
  const seen = new Map()
  for (const [id, dungeon] of allViews()) {
    const e = painted(perspective(dungeon), enhanced)
    const end = e.find((x) => x.kind === 'void')?.depth
    e.forEach((x, i) => {
      if (x.kind !== 'door') return
      const wall = e[i - 1]
      assert.equal(wall.id + '.door', x.id, id)
      // Distinct from its wall: a darker, warmer fill and a lighter frame.
      // (a side wall at its middle, where its door stands.)
      const mid = (t) => hexMid(t.from, t.to)
      const fill = x.fill, wallFill = wall.tone ? mid(wall.tone) : wall.fill
      assert.ok(!x.tone && fill.startsWith('#'), `${id} ${x.id}`)
      assert.ok(lum(wallFill) - lum(fill) >= 25 * fog(x.depth + 1), `${id} ${x.id}`)
      assert.ok(lum(x.edge) > lum(wallFill), `${id} ${x.id}`)
      if (x.depth <= 3) assert.deepEqual(x.details.map((d) => [d.kind, d.segments.length]), [['plank', 3], ['band', 2]], id)
      // The same door in every view: nothing but its place (and where the
      // view ends, for the fade) decides how it looks.
      const key = `${x.depth}:${x.lane}:${x.face}:${end}`
      if (seen.has(key)) assert.equal(recipe(x), seen.get(key), `${id} ${x.id}`)
      else seen.set(key, recipe(x))
    })
  }
  assert.ok(seen.size >= 8, `${seen.size} door places`)
})

test('styles never change the scene, the projection or the view they are given', () => {
  const deepFreeze = (o) => { if (o && typeof o === 'object') { Object.freeze(o); Object.values(o).forEach(deepFreeze) } return o }
  for (const c of cases) {
    const dungeon = deepFreeze(structuredClone(c.dungeon))
    const scene = JSON.stringify(mazeScene(dungeon))
    const p = deepFreeze(perspective(dungeon))
    const before = JSON.stringify(p)
    const a = painted(p, enhanced), b = painted(p, classic), again = painted(p, enhanced)
    assert.equal(JSON.stringify(p), before, c.id)
    assert.equal(JSON.stringify(mazeScene(dungeon)), scene, c.id)
    assert.equal(JSON.stringify(a), JSON.stringify(again), c.id)
    assert.ok(b.length <= a.length, c.id)
    assert.deepEqual(dungeon, c.dungeon, c.id)
  }
  assert.deepEqual(styles, { classic, enhanced })
  assert.deepEqual(defaultLook, { style: 'enhanced', zoom: 'standard' })
})

test('the zoom presets: square frames about the vanishing point, between the party’s own square’s planes', () => {
  assert.deepEqual(Object.keys(zooms), ['close', 'standard', 'wide'])
  assert.equal(zooms.standard, viewBox)
  const halves = Object.values(zooms).map((z) => {
    const [x, y, w, h] = z.split(' ').map(Number)
    assert.deepEqual([x, y, w], [-w / 2, -h / 2, h], z)
    return w / 2
  })
  assert.deepEqual(halves, [36, 40, 52])
  for (const h of halves) {
    // Everything past the party's own square is always in view, whole...
    assert.ok(h > far(0), `${h}`)
    // ...at least 4 units of that square's sides (a wall, a door's jamb or
    // an opening beside the party) are in view...
    assert.ok(h - far(0) >= 4, `${h}`)
    // ...and nothing beside or behind the party is ever framed.
    assert.ok(h < near(0), `${h}`)
  }
})

test('every zoom frames the same drawing: the nearest walls, openings and doors in every frame', () => {
  const frames = Object.values(zooms).map((z) => z.split(' ').map(Number)[2] / 2)
  for (const [id, dungeon] of allViews()) {
    const e = painted(perspective(dungeon), enhanced)
    for (const x of e) {
      if (x.kind === 'void') continue
      for (const h of frames) {
        const pts = x.points
        const inFrame = (p) => Math.abs(p[0]) <= h + 1e-9 && Math.abs(p[1]) <= h + 1e-9
        if (x.depth >= 1 && x.lane === 0) {
          // The corridor past the party's square, whole: its walls, doors,
          // floors and ceilings.
          assert.ok(pts.every(inFrame), `${id} ${x.id} at ${h}`)
        }
        if (x.depth >= 1 && x.kind === 'door') {
          // A door beside the corridor (in a side wall, or ahead of a side
          // square) is in view at least to its middle; at depth 1 one ahead
          // of a side square reaches past every frame, as in Classic.
          const mid = pts.reduce(([a, b], [px, py]) => [a + px / 4, b + py / 4], [0, 0])
          assert.ok(inFrame(mid), `${id} ${x.id} at ${h}`)
        }
        if (x.depth === 0 && x.face === 'side' && x.kind === 'wall') {
          // A wall beside the party: a strip of it at every zoom.
          const strip = [x.lane * (far(0) + 2), 0]
          assert.ok(Math.abs(strip[0]) <= h && strictly(strip, pts), `${id} ${x.id} at ${h}`)
        }
      }
    }
  }
})
