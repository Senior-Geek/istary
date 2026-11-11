// Conditions and the rescue's progress on Inspect (R-3), over Go's
// recorded dungeon views (testdata/dungeon-views.json) and by hand.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { condition } from './condition.js'
import { progress } from './inspect.js'

test('each incapacity has its own word and mark; the word is always shown', () => {
  const want = ['PARALYZED', 'STONED', 'DEAD', 'ASHES']
  const shown = want.map(condition)
  assert.deepEqual(shown.map((c) => c.label), want)
  assert.equal(new Set(shown.map((c) => c.mark)).size, 4, 'four different marks')
  assert.equal(new Set(shown.map((c) => c.cls)).size, 4, 'four different looks')
  assert.ok(shown.every((c) => c.mark !== ''))
  assert.deepEqual(condition('SOMETHING NEW'), { label: 'SOMETHING NEW', mark: '', cls: 'other' })
})

test('the Svelte condition shows the word itself, not only a colour', () => {
  const src = readFileSync(new URL('./Condition.svelte', import.meta.url), 'utf8')
  assert.match(src, /\{c\.label\}/)
})

test('Inspect’s progress lines', () => {
  const base = { found: [], room: 2, size: 4, picked: false, elsewhere: 0, carried: [] }
  assert.deepEqual(progress(base), { party: 'PARTY 4 OF 6 · ROOM FOR 2', here: 'LOOKING... NO ONE IS HERE.', elsewhere: '' })
  assert.equal(progress({ ...base, picked: true }).here, 'NO ONE ELSE IS HERE.')
  assert.equal(progress({ ...base, found: [{}, {}] }).here, '2 CHARACTERS ARE HERE.')
  assert.equal(progress({ ...base, found: [{}], picked: true }).here, '1 MORE IS HERE.')
  assert.equal(progress({ ...base, elsewhere: 3 }).elsewhere, '3 MORE LIE ELSEWHERE (MAP, F10).')
  assert.equal(progress({ ...base, elsewhere: 1 }).elsewhere, '1 MORE LIES ELSEWHERE (MAP, F10).')
  assert.deepEqual(progress(undefined), { party: '', here: '', elsewhere: '' })
})

test('every recorded Inspect view names each found character’s condition', () => {
  const views = JSON.parse(readFileSync(new URL('./testdata/dungeon-views.json', import.meta.url)))
  let n = 0
  for (const c of views) {
    const ins = c.view?.inspect ?? c.inspect
    if (!ins) continue
    n++
    assert.equal(ins.size + ins.room, 6)
    for (const f of ins.found) assert.ok(f.condition, JSON.stringify(f))
    assert.ok(Array.isArray(ins.carried))
  }
  assert.ok(n > 0)
})
