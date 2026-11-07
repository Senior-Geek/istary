// Contract tests: the display helpers over real views recorded from Go
// (gui/combat_fixture_test.go writes testdata/combat-views.json from one
// seeded practice encounter with a chest).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { condition, ranks, step, needsOrder, current, highlight, chips, targetKind } from './combat.js'

const views = JSON.parse(readFileSync(new URL('./testdata/combat-views.json', import.meta.url)))

test('the recording covers a whole encounter', () => {
  const phases = new Set(views.map((v) => v.phase))
  for (const p of ['opening', 'orders', 'confirm', 'playback', 'chest', 'victory']) assert.ok(phases.has(p), p)
})

test('every next step is one the screen knows how to handle', () => {
  for (const v of views) {
    const st = step(v)
    const handledByChoice = ['orders', 'confirm', 'friendly', 'choice', 'chest'].includes(v.next)
    assert.ok(st.method || handledByChoice, v.next)
    if (handledByChoice) assert.equal(st.auto, false, v.next)
  }
})

test('the strip shows the six in formation order, split into ranks', () => {
  for (const v of views) {
    const r = ranks(v.party)
    assert.deepEqual([...r.front, ...r.back].map((m) => m.slot), v.party.map((m) => m.slot))
    for (const m of v.party) assert.ok(['ok', 'dead', 'paralyzed', 'asleep', 'afraid'].includes(condition(m)), m.status)
  }
})

test('order entry: an actor who can act, and orders only for members who can', () => {
  for (const v of views.filter((v) => v.next === 'orders')) {
    const actor = v.party.find((m) => m.slot === v.actor)
    assert.ok(actor?.canAct, 'actor can act')
    for (const opt of v.menu) {
      assert.ok(['group', 'member', 'spell', 'item', 'none'].includes(targetKind(opt, v)))
      if (!opt.available) assert.ok(opt.reason, `${opt.action} has a reason`)
    }
    for (const slot of needsOrder(v)) assert.ok(v.party.find((m) => m.slot === slot).canAct)
  }
})

test('playback marks only who is on screen, and only revealed events exist', () => {
  for (const v of views.filter((v) => v.phase === 'playback')) {
    assert.ok(v.frozen, 'round-start values during playback')
    const ev = current(v)
    if (v.next === 'end-round') assert.equal(ev, null)
    else assert.equal(v.current, v.events.length - 1, 'the newest released event is on screen')
    const h = highlight(ev)
    if (h.actorSlot) assert.ok(v.party.some((m) => m.slot === h.actorSlot))
    if (h.targetSlot) assert.ok(v.party.some((m) => m.slot === h.targetSlot))
    for (const c of chips(ev)) {
      if (c.slot) assert.ok(v.party.some((m) => m.slot === c.slot))
    }
  }
})

test('no engine-only field reaches the front end', () => {
  const raw = JSON.stringify(views)
  for (const key of ['monsterId', 'initiative', 'victim', 'trapType', 'totalGold', 'hpBefore', 'hpAfter']) {
    assert.ok(!raw.includes(`"${key}"`), key)
  }
})

test('lists may arrive as null (Go nil slices); the helpers cope', () => {
  const v = views.find((v) => v.phase === 'chest' && v.groups === null)
  assert.ok(v, 'the recording has a view with no groups')
  assert.doesNotThrow(() => { ranks(v.party); needsOrder(v); highlight(current(v)); chips(current(v)) })
})
