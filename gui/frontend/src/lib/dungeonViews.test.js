// Contract tests: the dungeon helpers over real views recorded from Go
// (gui/dungeon_fixture_test.go writes testdata/dungeon-views.json from
// seeded practice expeditions).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { scene, actions, keyAction, near } from './dungeon.js'
import { ranks } from './combat.js'

const views = JSON.parse(readFileSync(new URL('./testdata/dungeon-views.json', import.meta.url)))

test('the recording covers Camp, the walk and its prompts', () => {
  const kinds = new Set(views.map((v) => `${v.phase}/${v.prompt}`))
  for (const k of ['camp/', 'maze/', 'maze/stairs', 'maze/messages', 'maze/inspect']) assert.ok(kinds.has(k), k)
})

test('every action Go lists has a label and a key that performs it', () => {
  for (const v of views) {
    for (const a of v.actions ?? []) {
      if (a === 'button') {
        assert.deepEqual(keyAction(v.buttons[0].toLowerCase(), v), { action: 'button', button: 1 })
        continue
      }
      assert.ok(actions[a]?.label, a)
      const k = actions[a].keys.find((key) => keyAction(key, v)?.action === a)
      assert.ok(k, `no key performs ${a} at prompt "${v.prompt}"`)
    }
  }
})

test('keys never perform an action Go did not list', () => {
  const keys = ['f', 'w', 'ArrowUp', 'l', 'a', 'ArrowLeft', 'r', 'd', 'ArrowRight', 'k', 'c', 'i', 'Enter', ' ', 'y', 'n', 'p', 'b', 'q', 'Escape']
  for (const v of views) {
    for (const key of keys) {
      const a = keyAction(key, v)
      if (a) assert.ok((v.actions ?? []).includes(a.action), `${key} -> ${a.action} at "${v.prompt}"`)
    }
  }
})

test('the picture is drawn from the corridor alone, within the near plane', () => {
  for (const v of views) {
    const shapes = scene(v.corridor ?? [])
    if (v.phase !== 'maze' || !v.corridor?.length) {
      assert.equal(shapes.length, 0)
      continue
    }
    const walls = v.corridor.flatMap((c) => [c.left, c.right, c.front]).filter((w) => w !== 'open').length
    assert.ok(shapes.filter((s) => s.kind === 'wall').length >= walls)
    for (const s of shapes) {
      for (const [x, y] of s.points) {
        assert.ok(Number.isFinite(x) && Number.isFinite(y))
        assert.ok(Math.abs(y) <= near(0) && Math.abs(x) <= 3 * near(0))
      }
    }
  }
})

test('the party strip shows everyone in formation order', () => {
  for (const v of views) {
    const r = ranks(v.party)
    assert.deepEqual([...r.front, ...r.back].map((m) => m.slot), v.party.map((m) => m.slot))
  }
})
