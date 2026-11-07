import { test } from 'node:test'
import assert from 'node:assert/strict'
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
  assert.deepEqual(s.map((x) => x.kind), ['wall', 'wall', 'wall', 'wall', 'wall', 'door'])
  // The front door is drawn on the far plane of square 1.
  const door = s.at(-1)
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
