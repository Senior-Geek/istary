// The map drawing helpers over real map views recorded from Go
// (gui/map_fixture_test.go TestMapViewFixture writes testdata/map-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { drawing, edge, door, arrow, wrap, feature, mapKey, symbols, INSET, facingTurn } from './automap.js'

const cases = JSON.parse(readFileSync(new URL('./testdata/map-views.json', import.meta.url)))
const at = (name) => {
  const c = cases.find((x) => x.case === name)
  assert.ok(c, name)
  return c
}
const near = (a, b) => Math.abs(a - b) < 1e-9

test('the recording covers walls, doors, one-way walls, edges, features and every facing', () => {
  const all = (f) => cases.flatMap(f)
  assert.ok(all((c) => c.full.walls).some((w) => w.kind === 'door'))
  assert.ok(all((c) => c.full.walls).some((w) => w.oneWay))
  assert.ok(all((c) => c.full.wraps).length > 0)
  assert.ok(all((c) => c.local.walls).some((w) => w.oneWay))
  for (const k of ['stairs-up', 'stairs-down', 'elevator']) assert.ok(all((c) => c.full.features).some((f) => f.kind === k), k)
  assert.deepEqual([...new Set(cases.map((c) => c.local.party.facing))].sort(), ['E', 'N', 'S', 'W'])
})

test('every wall is drawn on its square’s side; a one-way wall just inside its own square', () => {
  for (const c of cases) {
    for (const v of [c.full, c.local]) {
      const d = drawing(v)
      assert.equal(d.walls.length, v.walls.length, c.case)
      v.walls.forEach((w, i) => {
        const l = d.walls[i]
        const inset = w.oneWay ? INSET : 0
        assert.deepEqual({ x1: l.x1, y1: l.y1, x2: l.x2, y2: l.y2 }, edge(w.row, w.col, w.side, inset))
        // Inside the square (or on its edge), never outside the frame.
        for (const [x, y] of [[l.x1, l.y1], [l.x2, l.y2]]) {
          assert.ok(x >= w.col && x <= w.col + 1 && y >= w.row && y <= w.row + 1, `${c.case} ${JSON.stringify(w)}`)
          assert.ok(x >= 0 && x <= v.size && y >= 0 && y <= v.size)
        }
        if (w.oneWay) {
          // Not on the boundary: the other side is not drawn as blocked.
          const onEdge = { N: l.y1 === w.row, S: l.y1 === w.row + 1, W: l.x1 === w.col, E: l.x1 === w.col + 1 }[w.side]
          assert.equal(onEdge, false)
        }
        assert.equal(!!l.door, w.kind === 'door')
      })
    }
  }
})

test('a door is marked in the middle of its wall', () => {
  const l = edge(3, 4, 'N')
  assert.deepEqual(door(l), { x1: 4.3, y1: 3, x2: 4.7, y2: 3 })
  const v = drawing(at('L1 one-way door').local)
  const d = v.walls.find((w) => w.door && w.oneWay)
  assert.ok(d, 'the one-way door of the recording')
  assert.ok(near((d.door.x1 + d.door.x2) / 2, (d.x1 + d.x2) / 2) && near((d.door.y1 + d.door.y2) / 2, (d.y1 + d.y2) / 2))
})

test('features sit at the centre of their squares with their own symbols', () => {
  for (const c of cases) {
    for (const f of [...c.full.features, ...c.local.features]) {
      const s = feature(f)
      assert.ok(s, f.kind)
      assert.equal(s.x, f.col + 0.5)
      assert.equal(s.y, f.row + 0.5)
      assert.equal(s.path, symbols[f.kind].path)
    }
  }
  assert.equal(feature({ row: 1, col: 1, kind: 'teleporter' }), null, 'not drawn yet')
  const e = at('entrance')
  assert.deepEqual(e.full.features, [{ row: 19, col: 0, kind: 'stairs-up' }])
  assert.deepEqual(e.local.features, [{ row: 3, col: 3, kind: 'stairs-up' }])
})

test('the party’s arrow is at its square’s centre, turned its way; the minimap keeps it centred', () => {
  for (const c of cases) {
    const p = c.local.party
    assert.deepEqual([p.row, p.col], [3, 3], c.case)
    const a = arrow(p)
    assert.deepEqual([a.cx, a.cy, a.turn], [3.5, 3.5, facingTurn[p.facing]])
    if (c.full.party) {
      assert.equal(c.full.party.place, p.place)
      assert.equal(c.full.party.facing, p.facing)
      const m = /^L(\d+):R(\d+):C(\d+)$/.exec(p.place)
      assert.deepEqual([c.full.level, c.full.party.row, c.full.party.col], [+m[1], +m[2], +m[3]])
    }
  }
  assert.equal(at('entrance').full.party.place, 'L1:R19:C0')
  assert.equal(at('entrance').full.party.facing, 'N')
})

test('ways across the edge are chevrons outside the fixed frame, pointing out', () => {
  let n = 0
  for (const c of cases) {
    for (const w of c.full.wraps) {
      const pts = wrap(w).points.split(' ').map((p) => p.split(',').map(Number))
      const tip = pts[1]
      const outside = { N: tip[1] < 0, S: tip[1] > 20, W: tip[0] < 0, E: tip[0] > 20 }[w.side]
      assert.ok(outside, `${c.case} ${JSON.stringify(w)}`)
      // On the edge it crosses.
      assert.ok({ N: w.row === 0, S: w.row === 19, W: w.col === 0, E: w.col === 19 }[w.side])
      n++
    }
  }
  assert.ok(n > 0)
})

test('the minimap shows where the level’s edge runs through it', () => {
  const north = drawing(at('L1 north edge').local)
  assert.deepEqual(north.seams.filter((s) => s.y1 === s.y2).map((s) => s.y1), [3], 'R0 begins at the party’s row')
  const west = drawing(at('L2 west edge').local)
  assert.deepEqual(west.seams.filter((s) => s.x1 === s.x2).map((s) => s.x1), [3])
  const entrance = drawing(at('entrance').local)
  assert.equal(entrance.seams.length, 2, 'the entrance is in the south-west corner')
})

// The minimap's walls, placed back on the level, are the full map's.
test('the minimap and the full map agree on the walls they both show', () => {
  let compared = 0
  for (const c of cases) {
    const m = /^L(\d+):R(\d+):C(\d+)$/.exec(c.local.party.place)
    const pr = +m[2], pc = +m[3]
    const abs = (r, col) => [((pr + r - 3) % 20 + 20) % 20, ((pc + col - 3) % 20 + 20) % 20]
    const key = (r, col, w) => `${r}:${col}:${w.side}:${w.kind}:${!!w.oneWay}`
    const full = new Set(c.full.walls.map((w) => key(w.row, w.col, w)))
    const step = { N: [-1, 0], S: [1, 0], W: [0, -1], E: [0, 1] }
    for (const w of c.local.walls) {
      const [dr, dc] = step[w.side]
      const nr = w.row + dr, nc = w.col + dc
      if (nr < 0 || nr > 6 || nc < 0 || nc > 6) continue // the far side is outside the window
      const [r, col] = abs(w.row, w.col)
      const crosses = { N: r === 0, S: r === 19, W: col === 0, E: col === 19 }[w.side]
      if (crosses) continue // drawn per side at the frame on the full map
      assert.ok(full.has(key(r, col, w)), `${c.case}: ${JSON.stringify(w)} at R${r}:C${col}`)
      compared++
    }
  }
  assert.ok(compared > 50, `only ${compared}`)
})

test('F10 opens and closes the full map, Escape closes it, and no other key gets past it', () => {
  assert.equal(mapKey('F10', false), 'open')
  assert.equal(mapKey('F10', true), 'close')
  assert.equal(mapKey('Escape', true), 'close')
  assert.equal(mapKey('Escape', false), null, 'Escape is the maze’s own when the map is shut')
  for (const k of ['f', 'ArrowUp', 'l', 'r', 'k', 'c', 'i', 'Enter', 'y']) {
    assert.equal(mapKey(k, true), 'block', k)
    assert.equal(mapKey(k, false), null, k)
  }
})
