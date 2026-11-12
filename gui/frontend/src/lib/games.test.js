// Contract tests: game selection's helpers over real views recorded from Go
// (gui/games_fixture_test.go TestGamesViewFixture writes
// testdata/games-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { summary, choices, heading, state } from './games.js'

const views = JSON.parse(readFileSync(new URL('./testdata/games-views.json', import.meta.url)))
const at = (name) => views.find((v) => v.case === name)
const acts = (slot, games) => choices(slot, games).map((c) => c.act)

test('the recording covers every kind of slot and earlier save', () => {
  const states = new Set(views.flatMap((v) => v.games.slots.map((s) => s.state)))
  for (const s of ['empty', 'game', 'damaged']) assert.ok(states.has(s), s)
  assert.ok(views.some((v) => v.games.legacy && !v.games.legacy.problem), 'an earlier save to import')
  assert.ok(views.some((v) => v.games.legacy?.problem), 'a damaged earlier save')
  assert.ok(views.some((v) => v.games.replacing), 'a replacement awaiting confirmation')
  assert.ok(views.every((v) => v.screen === 'games' && v.games.slots.length === 6))
})

test('an empty slot begins a new game, and takes the earlier save only when it can be imported', () => {
  const none = at('no games')
  for (const s of none.games.slots) assert.deepEqual(acts(s, none.games), ['new'])
  const earlier = at('an earlier save')
  for (const s of earlier.games.slots) assert.deepEqual(acts(s, earlier.games), ['new', 'import'])
  const damaged = at('games of every kind')
  assert.deepEqual(acts(damaged.games.slots[3], damaged.games), ['new'], 'a damaged earlier save is not offered')
})

test('a saved game continues first; replacing it is a second, separate choice', () => {
  const v = at('games of every kind')
  for (const s of v.games.slots.filter((s) => s.state === 'game')) {
    const c = choices(s, v.games)
    assert.deepEqual(c.map((x) => x.act), ['continue', 'replace'])
    assert.equal(c[0].kind, 'primary')
    assert.notEqual(c[1].kind, 'danger', 'asking to replace is not itself destructive')
  }
})

test('a damaged game cannot be continued, only replaced, and says why', () => {
  const v = at('games of every kind')
  const s = v.games.slots.find((s) => s.state === 'damaged')
  assert.deepEqual(acts(s, v.games), ['replace'])
  assert.ok(s.problem)
  assert.equal(state(s), 'Cannot be opened')
  assert.match(at('continuing a damaged game').error, /cannot be continued/)
})

test('while a replacement awaits confirmation, that slot offers only confirming or keeping; the rest are as before', () => {
  const v = at('replacing game 3')
  for (const s of v.games.slots) {
    if (s.slot === 3) {
      const c = choices(s, v.games)
      assert.deepEqual(c.map((x) => x.act), ['confirmReplace', 'cancelReplace'])
      assert.equal(c[0].kind, 'danger')
    } else {
      assert.deepEqual(acts(s, v.games), acts(s, at('games of every kind').games))
    }
  }
})

test('a slot is summed up from what Go read in its save', () => {
  const v = at('games of every kind')
  assert.deepEqual(summary(v.games.slots[0]), ['3 characters', 'Out in the maze: 2 on B2F', '1 stranded in the maze'])
  assert.deepEqual(summary(v.games.slots[1]), ['3 characters', 'Party: ALDA, BRAN'])
  assert.deepEqual(summary(v.games.slots[2]), ['No characters yet'])
  assert.equal(heading(v.games.slots[1]), 'Game 2 · imported')
  assert.equal(heading(v.games.slots[0]), 'Game 1')
  assert.deepEqual(summary(at('an earlier save').games.legacy), ['3 characters', 'Party: ALDA, BRAN'])
})
