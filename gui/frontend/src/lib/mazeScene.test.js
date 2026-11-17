// The perspective scene (mazeScene.js) over views recorded from Go
// (gui/maze_scene_fixture_test.go builds each corridor as a synthetic level
// and checks it against the engine's Sight; testdata/maze-scenes.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { mazeScene } from './mazeScene.js'

const cases = JSON.parse(readFileSync(new URL('./testdata/maze-scenes.json', import.meta.url)))
const byId = Object.fromEntries(cases.map((c) => [c.id, c]))
const sceneOf = (id) => mazeScene(byId[id].dungeon)
const ids = (s) => s.surfaces.map((x) => x.id)
// beside is what is seen in a side lane through an opening (not a side wall).
const beside = (s, lane) => s.surfaces.filter((x) => x.lane === lane && x.face !== 'side').map((x) => x.id)
const walls = (s) => s.surfaces.filter((x) => x.face === 'side' || x.face === 'front')

test('the recording has every geometry and visibility case', () => {
  for (const id of ['G01', 'G02', 'G03', 'G04', 'G05', 'G06', 'G07', 'G08', 'G09', 'G10', 'G11-N', 'G11-E', 'G11-S', 'G11-W',
    'G12-edge', 'G12-inside', 'V01', 'V02', 'V03', 'V04', 'V05', 'V06', 'V06-wall', 'V07', 'V08', 'V09', 'V10', 'V11']) {
    assert.ok(byId[id], id)
  }
})

test('the scene sees as many squares as Go, and ends as Go says', () => {
  for (const c of cases) {
    const s = mazeScene(c.dungeon)
    assert.equal(s.depths, c.depths, c.id)
    assert.equal(s.end.kind, c.ends, c.id)
    assert.equal(s.end.depth, ['wall', 'door'].includes(c.ends) ? c.depths - 1 : c.depths, c.id)
    assert.equal(s.lit, c.light, c.id)
  }
})

test('every wall, door and opening is the corridor’s, and nothing else is', () => {
  for (const c of cases) {
    const s = mazeScene(c.dungeon)
    const want = [], open = []
    ;(c.dungeon.corridor ?? []).forEach((q, d) => {
      for (const side of ['left', 'right']) {
        if (q[side] !== 'open') want.push(`d${d}.${side}:${q[side]}`)
        else {
          open.push(`d${d}.${side}:${q[side + 'Ahead']}`)
          if (q[side + 'Ahead'] !== 'open') want.push(`d${d}.${side}Ahead:${q[side + 'Ahead']}`)
        }
      }
      if (q.front !== 'open') want.push(`d${d}.front:${q.front}`)
    })
    assert.deepEqual(walls(s).map((x) => `${x.id}:${x.seen}`).sort(), want.sort(), c.id)
    assert.deepEqual(s.openings.map((o) => `${o.id}:${o.ahead}`).sort(), open.sort(), c.id)
    // A floor and a ceiling for each square seen, and for each square seen
    // beside it through an opening; no more.
    const bands = s.surfaces.filter((x) => x.face === 'floor' || x.face === 'ceiling')
    assert.equal(bands.length, 2 * (s.depths + s.openings.length), c.id)
    assert.ok(s.surfaces.every((x) => x.depth < s.depths), c.id)
  }
})

test('G01–G02: a straight corridor and a dead end', () => {
  const g1 = sceneOf('G01')
  assert.deepEqual(walls(g1).map((x) => x.id), ['d0.left', 'd0.right', 'd1.left', 'd1.right', 'd2.left', 'd2.right', 'd3.left', 'd3.right', 'd4.left', 'd4.right'])
  assert.deepEqual(g1.end, { kind: 'limit', depth: 5 })
  const g2 = sceneOf('G02')
  assert.deepEqual(g2.end, { kind: 'wall', depth: 2 })
  assert.equal(walls(g2).at(-1).id, 'd2.front')
})

test('G03–G04: an opening on the side Go says, and nothing down the side passage', () => {
  const g3 = sceneOf('G03')
  assert.deepEqual(g3.openings, [{ id: 'd1.left', depth: 1, lane: -1, ahead: 'wall' }])
  assert.deepEqual(beside(g3, -1), ['d1.leftFloor', 'd1.leftCeiling', 'd1.leftAhead'])
  assert.deepEqual(beside(g3, 1), [])
  // G04's side passage goes on ahead of the opening: nothing of it is seen.
  const g4 = sceneOf('G04')
  assert.deepEqual(g4.openings, [{ id: 'd2.right', depth: 2, lane: 1, ahead: 'open' }])
  assert.deepEqual(beside(g4, 1), ['d2.rightFloor', 'd2.rightCeiling'])
  assert.deepEqual(beside(g4, -1), [])
})

test('G05–G07: a corner, a T-junction and a crossing', () => {
  const g5 = sceneOf('G05')
  assert.deepEqual(g5.openings.map((o) => o.id), ['d1.right'])
  assert.deepEqual(g5.end, { kind: 'wall', depth: 1 })
  assert.ok(ids(g5).includes('d1.left') && ids(g5).includes('d1.rightAhead'))
  const g6 = sceneOf('G06')
  assert.deepEqual(g6.openings.map((o) => `${o.id}:${o.ahead}`), ['d1.left:wall', 'd1.right:wall'])
  assert.deepEqual(g6.end, { kind: 'wall', depth: 1 })
  const g7 = sceneOf('G07')
  assert.deepEqual(g7.openings.map((o) => o.id), ['d1.left', 'd1.right'])
  assert.ok(!ids(g7).includes('d1.front'), 'the crossing goes on ahead')
  assert.deepEqual(g7.end, { kind: 'wall', depth: 3 })
})

test('G08–G10: doors where Go sees them, at their depth and side', () => {
  const doors = (id) => sceneOf(id).surfaces.filter((x) => x.seen === 'door').map((x) => `${x.id}:${x.face}:${x.lane}`)
  assert.deepEqual(doors('G08'), ['d2.front:front:0'])
  assert.deepEqual(sceneOf('G08').end, { kind: 'door', depth: 2 })
  assert.deepEqual(doors('G09'), ['d1.left:side:-1'])
  assert.deepEqual(doors('G10'), ['d0.right:side:1', 'd1.left:side:-1', 'd2.leftAhead:front:-1', 'd3.front:front:0'])
})

test('G11: the same square seen facing each way', () => {
  // Its walls: north a wall, east a door, south and west open.
  const walls = ['wall', 'door', 'open', 'open']
  for (const [f, k] of [['N', 0], ['E', 1], ['S', 2], ['W', 3]]) {
    const s = sceneOf('G11-' + f)
    const d0 = Object.fromEntries(s.surfaces.filter((x) => x.depth === 0 && x.face !== 'floor' && x.face !== 'ceiling').map((x) => [x.id, x.seen]))
    const want = {}
    const left = walls[(k + 3) % 4], right = walls[(k + 1) % 4], front = walls[k]
    if (left !== 'open') want['d0.left'] = left
    if (right !== 'open') want['d0.right'] = right
    if (front !== 'open') want['d0.front'] = front
    // The squares around are solid: an open side shows a wall ahead.
    if (left === 'open') want['d0.leftAhead'] = 'wall'
    if (right === 'open') want['d0.rightAhead'] = 'wall'
    assert.deepEqual(d0, want, f)
  }
})

test('G12: a corridor across the level’s edge is the same corridor', () => {
  assert.deepEqual(sceneOf('G12-edge'), sceneOf('G12-inside'))
})

test('V01–V03: two squares without light, five with it, three with QUICK plot', () => {
  assert.equal(sceneOf('V01').depths, 2)
  assert.equal(sceneOf('V02').depths, 5)
  assert.equal(sceneOf('V03').depths, 3)
  for (const id of ['V01', 'V02', 'V03']) assert.equal(sceneOf(id).end.kind, 'limit')
})

test('V04–V05: nothing behind the wall or the door that blocks the view', () => {
  for (const [id, kind] of [['V04', 'wall'], ['V05', 'door']]) {
    const s = sceneOf(id)
    assert.deepEqual(s.end, { kind, depth: 1 })
    assert.ok(s.surfaces.every((x) => x.depth <= 1), id)
    assert.equal(s.surfaces.filter((x) => x.seen === 'door').length, kind === 'door' ? 1 : 0, id)
  }
})

test('V06–V07: a hidden door is a wall without light, a door with it', () => {
  assert.deepEqual(sceneOf('V06'), sceneOf('V06-wall'))
  assert.ok(!sceneOf('V06').surfaces.some((x) => x.seen === 'door'))
  assert.equal(sceneOf('V07').surfaces.find((x) => x.id === 'd1.front').seen, 'door')
})

test('V08–V11: darkness and the sight’s limit end the view; nothing remembered is drawn', () => {
  const v8 = sceneOf('V08')
  assert.deepEqual(v8.end, { kind: 'dark', depth: 2 })
  assert.ok(v8.surfaces.every((x) => x.depth < 2) && !v8.surfaces.some((x) => x.seen === 'door'))
  const v9 = sceneOf('V09')
  assert.deepEqual([v9.depths, v9.surfaces, v9.openings, v9.end], [0, [], [], { kind: 'dark', depth: 0 }])
  // The corridor goes on past the light to a door: the view ends unknown.
  const v10 = sceneOf('V10')
  assert.deepEqual(v10.end, { kind: 'limit', depth: 5 })
  assert.ok(!v10.surfaces.some((x) => x.seen === 'door' || x.id.endsWith('.front')))
  // Looked down with light before (the exploration record has five
  // squares): without light, two, as the corridor says.
  const v11 = sceneOf('V11')
  assert.deepEqual([v11.depths, v11.end], [2, { kind: 'limit', depth: 2 }])
  assert.deepEqual(sceneOf('V11'), sceneOf('V01'))
})

test('off the maze (at Camp, in town) nothing is seen', () => {
  for (const d of [undefined, null, {}, { phase: 'camp', corridor: [{ left: 'wall', right: 'wall', front: 'wall' }] }]) {
    assert.deepEqual(mazeScene(d), { depths: 0, lit: false, end: { kind: 'none', depth: 0 }, surfaces: [], openings: [] })
  }
})

// watched returns a copy of a dungeon view that records every field read.
function watched(value, path, reads) {
  if (value === null || typeof value !== 'object') return value
  return new Proxy(value, {
    get(t, k) {
      if (typeof k === 'string' && !(Array.isArray(t) && (/^\d+$/.test(k) || ['length', 'forEach', 'map', 'includes', 'at', 'constructor'].includes(k)))) {
        reads.add(path + k)
      }
      return watched(t[k], Array.isArray(t) ? path : path + k + '.', reads)
    },
  })
}

test('the scene reads only the corridor, darkness, phase and spells of the view', () => {
  const reads = new Set()
  for (const c of cases) mazeScene(watched(c.dungeon, '', reads))
  const allowed = ['phase', 'corridor', 'dark', 'spells',
    ...['left', 'right', 'front', 'leftAhead', 'rightAhead'].map((k) => 'corridor.' + k)]
  assert.deepEqual([...reads].filter((r) => !allowed.includes(r)), [])
})

test('other parts of the view (maps, position, party) change nothing in the scene', () => {
  for (const c of cases) {
    const more = { ...c.dungeon, minimap: { squares: [1] }, position: { row: 3 }, map: {}, x: 1, y: 2, party: [], lines: ['X'] }
    assert.deepEqual(mazeScene(more), mazeScene(c.dungeon), c.id)
  }
})

const freeze = (v) => { if (v && typeof v === 'object') { Object.values(v).forEach(freeze); Object.freeze(v) } return v }

test('the scene is deterministic and changes nothing it is given', () => {
  for (const c of cases) {
    const before = JSON.stringify(c.dungeon)
    const d = freeze(structuredClone(c.dungeon)) // writing to it would throw
    const a = mazeScene(d), b = mazeScene(d)
    assert.deepEqual(a, b, c.id)
    assert.equal(JSON.stringify(a), JSON.stringify(b), c.id)
    assert.equal(JSON.stringify(c.dungeon), before, c.id)
  }
})

test('the scene and projection modules import nothing of the app and name no other source', () => {
  for (const f of ['./mazeScene.js', './mazeProjection.js', './mazeStyle.js']) {
    const src = readFileSync(new URL(f, import.meta.url), 'utf8')
    const code = src.replace(/\/\/.*$/gm, '')
    const imports = [...code.matchAll(/import[^'"]*['"]([^'"]+)['"]/g)].map((m) => m[1])
    assert.ok(imports.every((i) => i === './mazeScene.js'), `${f} imports ${imports}`)
    assert.ok(!/wailsjs|Session|minimap|automap|\bposition\b|exploration/i.test(code), f)
    assert.ok(!/Math\.random|\bDate\b|fetch\(/.test(code), f)
  }
})
