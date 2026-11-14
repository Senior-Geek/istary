// Contract tests: the Inn helpers over real views recorded from Go
// (gui/inn_test.go TestInnViewFixture writes testdata/inn-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { guest, rows, quoteText, shortfall, maxWeeks, pickLabel, receipt, statChanges, roomName } from './inn.js'

const views = JSON.parse(readFileSync(new URL('./testdata/inn-views.json', import.meta.url)))
const at = (name) => {
  const v = views.find((x) => x.case === name)
  assert.ok(v, name)
  return v
}
const quotesOf = (v, weeks) => Object.fromEntries((v.quotes ?? []).filter((q) => q.weeks === weeks).map((q) => [q.room, q.quote]))

test('the recording covers every step and every save outcome', () => {
  const steps = new Set(views.map((v) => v.inn.view.step))
  for (const s of ['who', 'rooms', 'levelUp']) assert.ok(steps.has(s), s)
  const tones = new Set(views.map((v) => receipt(v.inn)?.tone).filter(Boolean))
  assert.deepEqual([...tones].sort(), ['notDurable', 'saved', 'unsaved'])
  for (const v of views) assert.equal(v.location, 'inn', v.case)
})

test('the guest is Go’s, with the roster’s HP, gold, age, spells and experience', () => {
  assert.equal(guest(at('who').inn, []), null)
  const v = at('fighter checks in')
  const g = guest(v.inn, [v.guest])
  assert.equal(g.name, v.inn.view.name)
  assert.equal(g.slot, v.inn.view.guest)
  for (const k of ['hp', 'maxHp', 'gold', 'age', 'level', 'xp', 'nextXp']) assert.equal(g[k], v.guest[k], k)
  const m = guest(at('mage: little gold').inn, [at('mage: little gold').guest])
  assert.match(m.mage, /^L1 \d+\/\d+/)
  const ch = at('mage: little gold').guest
  assert.equal(m.spellsShort, ch.mageSpells.some((n, i) => n < ch.maxMage[i]))
  assert.equal(guest(at('mage: stables').inn, [at('mage: stables').guest]).spellsShort, false, 'after a stay: full')
})

test('until healed: every room as Go quotes it, the stables free, disabled exactly by Go’s refusal', () => {
  for (const v of views.filter((v) => v.inn.view.guest >= 0 && !v.inn.view.guestRefusal)) {
    const rs = rows(v.inn, 'healed')
    assert.equal(rs.length, 5, v.case)
    v.inn.view.rooms.forEach((r, i) => {
      assert.deepEqual(rs[i].request, { action: 'stay', room: r.room, weeks: 0 })
      assert.equal(rs[i].reason, r.untilHealed.refusal)
      assert.equal(rs[i].text, quoteText(r.untilHealed, r))
      assert.equal(rs[i].free, r.free)
    })
    assert.match(rs[0].text, /free night/)
  }
})

test('chosen weeks: the paid rooms ask Go’s quote; a refusal disables with Go’s words', () => {
  const v = at('fighter: two weeks, a level')
  const one = rows(v.inn, 'weeks', 1, quotesOf(v, 1))
  assert.deepEqual(one[1].request, { action: 'stay', room: 1, weeks: 1 })
  assert.equal(one[1].reason, '')
  assert.equal(one[0].reason, '', 'the stables stay one free night in either mode')
  assert.deepEqual(one[0].request, { action: 'stay', room: 0, weeks: 0 })
  assert.match(one[1].text, /^1 week · 10 GP · HP to 10$/)
  const forty = rows(v.inn, 'weeks', 40, quotesOf(v, 40))
  assert.equal(forty[2].reason, 'AGUI needs only 4 weeks to heal fully.')
  assert.equal(forty[2].text, forty[2].reason)
  const waiting = rows(v.inn, 'weeks', 5, {})
  assert.equal(waiting[3].reason, 'Working out…', 'no quote yet: disabled until Go answers')
  assert.equal(maxWeeks(v.inn), Math.max(...v.inn.view.rooms.map((r) => r.maxWeeks)))
})

test('a guest short of gold is told so, from Go’s weeks needed and affordable', () => {
  const v = at('mage: little gold')
  const royal = v.inn.view.rooms[4].untilHealed
  assert.equal(royal.weeks, 0)
  assert.equal(royal.refusal, 'EGUI cannot pay for a week in the royal suites (500 GP a week, 25 GP held). The stables are free.')
  assert.equal(quoteText(royal, v.inn.view.rooms[4]), royal.refusal)
  const r = rows(v.inn, 'healed')
  assert.equal(r[4].reason, royal.refusal, 'a paid room not one week of which is paid for: refused, the stables named')
  assert.equal(r[0].reason, '', 'the stables stay open')
  assert.equal(shortfall(royal, 'EGUI'), '', 'no week at all is said once, by the quote')
  const cot = v.inn.view.rooms[1].untilHealed
  assert.ok(cot.weeks < cot.needed)
  assert.equal(shortfall(cot, 'EGUI'), `EGUI can pay for ${cot.affordable} weeks of the ${cot.needed} weeks needed`)
  assert.equal(shortfall(at('fighter checks in').inn.view.rooms[2].untilHealed, 'AGUI'), '')
})

test('the party bar checks in members Go allows, never the guest or a refusal', () => {
  const who = at('who')
  for (const g of who.inn.view.guests) assert.equal(pickLabel(who.inn, g.slot), g.refusal ? null : 'Stay', g.name)
  const v = at('fighter checks in')
  assert.equal(pickLabel(v.inn, v.inn.view.guest), null)
  assert.equal(pickLabel(v.inn, 4), 'Stay instead')
  assert.equal(pickLabel(v.inn, 3), null, 'the dead priest')
})

test('a stay’s receipt says what Go did: room, weeks, gold, HP, spells, age, level, attributes', () => {
  const r = receipt(at('fighter: two weeks, a level').inn)
  const st = at('fighter: two weeks, a level').inn.last.stay
  assert.equal(r.tone, 'saved')
  assert.equal(r.title, 'AGUI stayed 2 weeks')
  const text = r.lines.map((l) => l.text)
  assert.ok(text.includes('Economy Rooms, 2 weeks: 100 GP'), text)
  assert.ok(text.includes(`HP 3 → 9 of ${st.maxHpAfter}`), text)
  assert.ok(text.includes('Made level 3'))
  assert.ok(text.includes('Age unchanged'))
  for (const c of statChanges(st)) assert.ok(text.includes(`${c.name} ${c.from} → ${c.to}`))
  assert.ok(r.messages.includes('YOU MADE A LEVEL!'))
  const stables = receipt(at('mage: stables').inn)
  assert.ok(stables.lines.some((l) => l.text === 'The Stables, one night: free'))
  assert.ok(stables.lines.some((l) => /^Spell points restored: Mage L1 0→2$/.test(l.text)), stables.lines)
  assert.ok(text.includes('Spell points: none (no spells yet)'), text) // a fighter
  for (const row of rows(at('mage: stables').inn, 'healed')) assert.match(row.rate, /restores spells$/)
  assert.ok(stables.lines.some((l) => l.kind === 'xp'))
})

test('pooling’s receipt lists every member before and after; the total is kept', () => {
  const v = at('fighter: pooled')
  const r = receipt(v.inn)
  const m = v.inn.last.gold
  assert.equal(r.lines.length, m.names.length)
  assert.equal(m.after.reduce((a, b) => a + b), m.before.reduce((a, b) => a + b))
  assert.match(r.title, new RegExp(`^Pooled the party's ${m.total} GP to AGUI$`))
})

test('NOT saved, then saved again: the receipt follows Go', () => {
  const ns = receipt(at('a cot, NOT saved').inn)
  assert.equal(ns.tone, 'unsaved')
  assert.ok(at('a cot, NOT saved').unsaved)
  assert.equal(receipt(at('saved again').inn).tone, 'saved')
  assert.equal(receipt(at('not durable').inn).tone, 'notDurable')
  assert.equal(receipt(at('guest left').inn), null)
})

test('room names read as names', () => {
  assert.equal(roomName('ROYAL SUITES'), 'Royal Suites')
  assert.equal(roomName('THE STABLES'), 'The Stables')
})

test('a guest LOST of old age keeps the receipt, death included, and is offered no room', () => {
  const v = at('old age: LOST')
  const r = receipt(v.inn)
  assert.ok(v.inn.last.stay.died)
  assert.ok(r.lines.some((l) => l.kind === 'bad' && /died of old age and is LOST/.test(l.text)), r.lines)
  assert.ok(r.lines.some((l) => l.kind === 'loss'), 'attribute changes kept')
  assert.match(v.inn.view.guestRefusal, /IS NOT OK: LOST/)
  assert.deepEqual(rows(v.inn, 'healed'), [])
  assert.deepEqual(rows(v.inn, 'weeks', 1, {}), [])
  const again = at('old age: no room again')
  assert.equal(again.error, again.inn.view.guestRefusal)
  assert.ok(receipt(again.inn).lines.some((l) => l.kind === 'bad'), 'a refusal keeps the receipt')
  const who = at('old age: refused at who')
  const g = who.inn.view.guests.find((g) => g.refusal)
  assert.equal(pickLabel(who.inn, g.slot), null)
})
