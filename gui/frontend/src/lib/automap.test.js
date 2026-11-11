// The map drawing helpers over real map views recorded from Go
// (gui/map_fixture_test.go TestMapViewFixture writes testdata/map-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { drawing, edge, door, arrow, wrap, feature, mapKey, symbols, INSET, facingTurn, BADGE, CLIP, marker, strandedLabel, MARKER } from './automap.js'

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

test('the squares the party has stood on, and only those, get a floor', () => {
  for (const c of cases) {
    for (const v of [c.full, c.local]) {
      const d = drawing(v)
      assert.deepEqual(d.floors, v.visited.map((q) => ({ x: q.col, y: q.row })), c.case)
      assert.equal(d.floor.match(/M/g)?.length ?? 0, v.visited.length)
      for (const f of d.floors) assert.ok(f.x >= 0 && f.x < v.size && f.y >= 0 && f.y < v.size)
    }
    // The party's own square is always one.
    assert.ok(c.local.visited.some((q) => q.row === 3 && q.col === 3), c.case)
  }
  assert.deepEqual(at('entrance').full.visited, [{ row: 19, col: 0 }])
  assert.ok(cases.some((c) => c.full.visited.length > 1), 'a recording with a walk')
})

test('features sit at the centre of their squares with their own symbols', () => {
  for (const c of cases) {
    for (const f of [...c.full.features, ...c.local.features]) {
      if (c.local.party && f.row === 3 && f.col === 3 && c.local.features.includes(f)) continue // under the party: below
      if (c.full.party && f.row === c.full.party.row && f.col === c.full.party.col) continue
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

// The distance from point (px, py) to the segment a..b.
const toSegment = ([px, py], [ax, ay], [bx, by]) => {
  const dx = bx - ax, dy = by - ay
  const t = Math.max(0, Math.min(1, ((px - ax) * dx + (py - ay) * dy) / (dx * dx + dy * dy)))
  return Math.hypot(px - ax - t * dx, py - ay - t * dy)
}
// The arrow's outline as drawn: its points turned about its centre.
const outline = (a) => {
  const t = a.turn * Math.PI / 180
  return a.points.split(' ').map((p) => p.split(',').map(Number)).map(([x, y]) => [
    a.cx + (x - a.cx) * Math.cos(t) - (y - a.cy) * Math.sin(t), a.cy + (x - a.cx) * Math.sin(t) + (y - a.cy) * Math.cos(t)])
}

test('a feature under the party moves to a badge clear of the arrow, inside its square, whichever way it faces', () => {
  const e = drawing(at('entrance').local)
  assert.equal(e.features.length, 1)
  assert.equal(e.features[0].badge, true)
  assert.equal(e.features[0].kind, 'stairs-up')
  for (const facing of ['N', 'E', 'S', 'W']) {
    for (const v of [at('entrance').local, at('entrance').full]) {
      const d = drawing({ ...v, party: { ...v.party, facing } })
      const f = d.features.find((f) => f.badge)
      assert.ok(f, facing)
      const pts = outline(d.party)
      const clear = Math.min(...pts.map((p, i) => toSegment([f.x, f.y], p, pts[(i + 1) % pts.length])))
      assert.ok(clear >= BADGE.radius, `${facing}: badge ${clear.toFixed(3)} from the arrow`)
      const col = v.party.col, row = v.party.row
      assert.ok(f.x - BADGE.radius >= col && f.x + BADGE.radius <= col + 1 && f.y - BADGE.radius >= row && f.y + BADGE.radius <= row + 1, facing)
      assert.equal(f.path, symbols['stairs-up'].path)
    }
  }
  // Away from the party a feature keeps its full size at the centre.
  const away = drawing({ ...at('entrance').local, party: { ...at('entrance').local.party, row: 2 } })
  assert.deepEqual([away.features[0].x, away.features[0].y, away.features[0].scale, away.features[0].badge], [3.5, 3.5, 1, false])
})

test('a door on the minimap’s outer wall is drawn whole', () => {
  // The door mark is 0.2 wide across its wall's line.
  const half = 0.1
  assert.ok(CLIP >= half)
  let n = 0
  for (const c of cases) {
    for (const w of drawing(c.local).walls.filter((w) => w.door)) {
      for (const v of [w.door.x1, w.door.x2, w.door.y1, w.door.y2]) assert.ok(v >= 0 && v <= c.local.size)
      if ([w.x1, w.y1].some((v) => v === 0) || [w.x1, w.y1].some((v) => v === c.local.size)) n++
    }
  }
  assert.ok(n > 0, 'a recording with a door on the outer wall')
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

// Stranded markers (R-3).

test('one marker a square, with the count of those who lie there, at its centre', () => {
  const full = drawing(at('L1 stranded').full)
  assert.deepEqual(full.stranded.map((m) => [m.place, m.count, m.characters.map((c) => c.name)]), [
    ['L1:R5:C8', 1, ['EVER']],
    ['L1:R18:C1', 3, ['ANNE', 'BRAM', 'CASS']],
    ['L1:R19:C0', 1, ['DUNN']],
  ])
  const far = full.stranded[0]
  assert.deepEqual([far.x, far.y, far.scale, far.badge], [8.5, 5.5, 1, false])
  assert.equal(strandedLabel(full.stranded[1]), 'L1:R18:C1 — 3 stranded')
  assert.deepEqual(full.stranded[1].characters.map((c) => c.condition), ['DEAD', 'PARALYZED', 'ASHES'])
})

test('the LOST and those with no certain square are never marked', () => {
  for (const c of cases) {
    for (const v of [c.full, c.local]) {
      for (const m of v.stranded) {
        assert.ok(!m.characters.some((ch) => ['GONE', 'OLDE'].includes(ch.name)), c.case)
        assert.equal(m.count, m.characters.length)
        assert.ok(m.characters.every((ch) => ch.condition !== 'LOST'))
      }
    }
  }
})

test('a marker shows nothing of the maze around it', () => {
  // Without its markers a map is drawn exactly as before: no floor, wall,
  // feature or way across the edge comes with them.
  for (const c of cases.filter((c) => c.full.stranded.length)) {
    for (const v of [c.full, c.local]) {
      const a = drawing(v), b = drawing({ ...v, stranded: [] })
      assert.deepEqual({ ...a, stranded: [] }, b, c.case)
    }
  }
  // The far marker's square and its neighbours were never explored.
  const v = at('L1 stranded').full
  for (const [r, col] of [[5, 8], [4, 8], [6, 8], [5, 7], [5, 9]]) {
    assert.ok(!v.visited.some((q) => q.row === r && q.col === col), `R${r}:C${col} visited`)
    assert.ok(!v.walls.some((w) => w.row === r && w.col === col), `R${r}:C${col} has walls`)
    assert.ok(!v.features.some((f) => f.row === r && f.col === col), `R${r}:C${col} has a feature`)
  }
})

test('other levels’ markers are only on their own level’s map', () => {
  assert.deepEqual(at('L1 stranded').full.stranded.map((m) => m.place.split(':')[0]), ['L1', 'L1', 'L1'])
  assert.deepEqual(at('L2 stranded').full.stranded.map((m) => m.place), ['L2:R7:C12'])
  assert.ok(at('L1 stranded').full.levels.includes(2), 'level 2 can be chosen')
})

test('a marker under the party moves to a badge clear of the arrow and of a feature’s badge', () => {
  const v = at('L1 stranded')
  for (const facing of ['N', 'E', 'S', 'W']) {
    for (const view of [v.local, v.full]) {
      const d = drawing({ ...view, party: { ...view.party, facing } })
      const m = d.stranded.find((m) => m.badge)
      assert.ok(m, facing)
      assert.equal(m.place, 'L1:R19:C0')
      const pts = outline(d.party)
      const clear = Math.min(...pts.map((p, i) => toSegment([m.x, m.y], p, pts[(i + 1) % pts.length])))
      assert.ok(clear >= MARKER.radius, `${facing}: marker ${clear.toFixed(3)} from the arrow`)
      const { row, col } = view.party
      assert.ok(m.x - MARKER.radius >= col && m.x + MARKER.radius <= col + 1 && m.y - MARKER.radius >= row && m.y + MARKER.radius <= row + 1, facing)
      const f = d.features.find((f) => f.badge)
      assert.ok(f, 'the stairs share the square')
      assert.ok(Math.hypot(f.x - m.x, f.y - m.y) >= BADGE.radius + MARKER.radius, `${facing}: badges overlap`)
    }
  }
})

test('the party is drawn after the markers, so a marker never covers it', () => {
  const src = readFileSync(new URL('./MapSvg.svelte', import.meta.url), 'utf8')
  assert.ok(src.indexOf('d.stranded as m') < src.indexOf('{#if d.party}'))
  // The marker's shapes are declared inside the <svg>: a snippet declared
  // outside it would make them HTML elements, which draw nothing.
  assert.ok(src.indexOf('<svg') < src.indexOf('{#snippet mark'))
})

test('the minimap and the full map agree on the markers they both show', () => {
  let compared = 0
  for (const c of cases) {
    const m = /^L(\d+):R(\d+):C(\d+)$/.exec(c.local.party.place)
    const pr = +m[2], pc = +m[3]
    const inWindow = (r, col) => {
      const dr = ((r - pr) % 20 + 30) % 20 - 10, dc = ((col - pc) % 20 + 30) % 20 - 10
      return Math.abs(dr) <= 3 && Math.abs(dc) <= 3 ? [dr + 3, dc + 3] : null
    }
    const local = new Map(c.local.stranded.map((s) => [s.place, s]))
    for (const s of c.full.stranded) {
      const pos = inWindow(s.row, s.col)
      if (!pos) {
        assert.ok(!local.has(s.place), `${c.case}: ${s.place} should be outside the minimap`)
        continue
      }
      const l = local.get(s.place)
      assert.ok(l, `${c.case}: ${s.place} missing from the minimap`)
      assert.deepEqual([l.row, l.col], pos)
      assert.deepEqual(l.characters, s.characters)
      compared++
    }
    assert.ok(c.local.stranded.every((s) => c.full.stranded.some((f) => f.place === s.place)), c.case)
  }
  assert.ok(compared >= 3, `only ${compared}`)
})

test('marker places a marker given by hand', () => {
  const m = marker({ row: 2, col: 4, place: 'L3:R2:C4', count: 2, characters: [] }, { row: 0, col: 0, facing: 'N' })
  assert.deepEqual([m.x, m.y, m.badge, m.label], [4.5, 2.5, false, 'L3:R2:C4 — 2 stranded'])
})
