import { test } from 'node:test'
import assert from 'node:assert/strict'
import { condition, ranks, step, delay, needsOrder, orderFor, orderLabel, current, highlight, chips, targetKind } from './combat.js'

test('condition treatments come from the displayed status', () => {
  assert.equal(condition({ status: 'OK' }), 'ok')
  assert.equal(condition({ status: 'DEAD' }), 'dead')
  assert.equal(condition({ status: 'ASHED' }), 'dead')
  assert.equal(condition({ status: 'PARALYZE' }), 'paralyzed')
  assert.equal(condition({ status: 'STONED' }), 'paralyzed')
  assert.equal(condition({ status: 'ASLEEP' }), 'asleep')
  assert.equal(condition(undefined), 'ok')
})

test('ranks: slots 1-3 front, 4-6 back, in formation order', () => {
  const party = [1, 2, 3, 4, 5, 6].map((slot) => ({ slot }))
  const r = ranks(party)
  assert.deepEqual(r.front.map((m) => m.slot), [1, 2, 3])
  assert.deepEqual(r.back.map((m) => m.slot), [4, 5, 6])
})

test('only steps without a choice are automatic', () => {
  assert.equal(step({ next: 'advance' }).method, 'CombatAdvance')
  assert.equal(step({ next: 'advance' }).auto, true)
  assert.equal(step({ next: 'end-round' }).method, 'CombatEndRound')
  assert.equal(step({ next: 'chest-result' }).method, 'ChestContinue')
  for (const n of ['orders', 'confirm', 'choice', 'friendly', 'chest', 'leave']) {
    assert.equal(step({ next: n }).auto, false, n)
  }
  assert.equal(step({ next: 'nonsense' }).method, null)
  assert.ok(delay('start') > delay('advance'))
  assert.ok(delay('advance', 'fast') < delay('advance', 'slow'))
})

test('who still needs an order: members who can act without one', () => {
  const combat = {
    party: [{ slot: 1, canAct: true }, { slot: 2, canAct: false }, { slot: 3, canAct: true }],
    orders: [{ slot: 1, action: 'parry' }],
  }
  assert.deepEqual(needsOrder(combat), [3])
  assert.equal(orderFor(combat, 1).action, 'parry')
  assert.equal(orderFor(combat, 3), undefined)
})

test('order labels', () => {
  assert.equal(orderLabel({ action: 'fight', group: 2 }), 'Fight → group 2')
  assert.equal(orderLabel({ action: 'spell', spell: 'DIOS', ally: 'BRANNOC' }), 'DIOS → BRANNOC')
  assert.equal(orderLabel({ action: 'spell', spell: 'HALITO', group: 1 }), 'HALITO → group 1')
  assert.equal(orderLabel({ action: 'spell', spell: 'MILWA' }), 'MILWA')
  assert.equal(orderLabel({ action: 'use', item: 'POTION' }), 'Use POTION')
  assert.equal(orderLabel({ action: 'parry' }), 'Parry')
  assert.equal(orderLabel(undefined), '')
})

test('the event on screen, its highlights and effect chips', () => {
  const ev = {
    kind: 'monster-melee', actor: { side: 'monster', group: 2, member: 1, name: 'KOBOLD' }, target: { side: 'party', slot: 3, name: 'C' },
    lines: ['KOBOLD CLAWS AT', 'C', 'AND HITS FOR 4 DAMAGE!', 'C IS SLAIN!'],
    effects: [{ who: { side: 'party', slot: 3, name: 'C' }, hpChange: -4, statusFrom: 'OK', statusTo: 'DEAD' }],
  }
  const combat = { current: 0, events: [ev] }
  assert.equal(current(combat), ev)
  assert.equal(current({ current: -1, events: [ev] }), null)
  assert.deepEqual(highlight(ev), { actorSlot: 0, actorGroup: 2, targetSlot: 3, targetGroup: 0 })
  assert.deepEqual(highlight({ ...ev, silent: true }), { actorSlot: 0, actorGroup: 0, targetSlot: 0, targetGroup: 0 })
  assert.deepEqual(chips(ev), [{ slot: 3, text: '-4', kind: 'hurt' }, { slot: 3, text: 'DEAD', kind: 'status' }])
  assert.deepEqual(chips({ effects: [{ who: { side: 'monster', group: 1 }, statusFrom: 'OK', statusTo: 'DEAD' }] }), [{ group: 1, text: 'slain', kind: 'monster' }])
})

test('target kinds: Use with nothing usable needs nothing', () => {
  assert.equal(targetKind({ target: 'group' }, {}), 'group')
  assert.equal(targetKind({ target: 'item' }, { items: [] }), 'none')
  assert.equal(targetKind({ target: 'item' }, { items: [{ index: 0, name: 'X' }] }), 'item')
  assert.equal(targetKind(undefined, {}), 'none')
})
