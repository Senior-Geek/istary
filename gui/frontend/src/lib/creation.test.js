// Contract tests: the Training Grounds' creation helpers over real views
// recorded from Go (gui/creation_test.go TestCreationViewFixture writes
// testdata/creation-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { stage, choosing, classRows, instruction } from './creation.js'

const views = JSON.parse(readFileSync(new URL('./testdata/creation-views.json', import.meta.url)))
const at = (name) => {
  const v = views.find((x) => x.case === name)
  assert.ok(v, name)
  return v
}

test('every step is named, and the next decision is always said', () => {
  const want = {
    confirming: 'confirming', password: 'password', race: 'race', alignment: 'alignment',
    'points left': 'points', 'points spent': 'choose', 'class refused': 'choose', 'class step': 'class',
    review: 'confirm', 'evil, points spent': 'choose',
  }
  for (const v of views) {
    assert.equal(stage(v), want[v.case], v.case)
    assert.ok(instruction(v), `${v.case}: an instruction`)
  }
  assert.equal(stage({ screen: 'roster' }), '')
  assert.equal(instruction({ screen: 'roster' }), '')
})

test('while points are left, the whole class table follows them and nothing can be chosen', () => {
  const v = at('points left')
  const rows = classRows(v)
  assert.equal(rows.length, v.classes.length)
  assert.deepEqual(rows.map((r) => r.class), v.classes)
  assert.ok(rows.every((r) => !r.pickable))
  assert.equal(choosing(v), false)
  assert.equal(instruction(v), `Spend the bonus points: ${v.creation.points} left.`)
  // Go's reasons are kept, row by row.
  rows.forEach((r, i) => assert.equal(r.note, v.creation.guide[i].note))
})

test('with no points left, exactly the classes Go allows are highlighted, on the same screen', () => {
  for (const name of ['points spent', 'evil, points spent', 'class step']) {
    const v = at(name)
    const rows = classRows(v)
    assert.equal(choosing(v), true, name)
    assert.equal(instruction(v), 'Choose a highlighted class.', name)
    rows.forEach((r, i) => {
      assert.equal(r.index, i)
      assert.equal(r.pickable, v.creation.classOk[i], `${name}: ${r.class}`)
      assert.equal(r.pickable, r.status === 'eligible', `${name}: ${r.class} status ${r.status}`)
    })
    assert.ok(rows.some((r) => r.pickable) && rows.some((r) => !r.pickable), name)
  }
  assert.equal(at('points spent').creation.step, 'stats', 'no separate class display')
})

test('a refused class leaves the points screen as it was, with Go’s reason', () => {
  const before = at('points spent'), after = at('class refused')
  assert.match(after.error, /qualify/)
  assert.deepEqual(after.creation, before.creation)
})

test('the review has the class and no table to choose from', () => {
  const v = at('review')
  assert.equal(v.creation.class, 'MAGE')
  assert.ok(classRows(v).every((r) => !r.pickable))
  assert.match(instruction(v), /Keep this character/)
})
