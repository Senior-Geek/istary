// Contract tests: the location frame's and party bar's helpers over real
// views recorded from Go (gui/frame_fixture_test.go TestFrameViewFixture
// writes testdata/frame-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { places, title, back, destinations } from './navigation.js'
import { formation, condition, unwell, hpShare, canOpen } from './party.js'
import { scenes, sceneFor, fallback, illustrated } from './scenes.js'

const views = JSON.parse(readFileSync(new URL('./testdata/frame-views.json', import.meta.url)))
const at = (name) => views.find((v) => v.case === name)

test('the recording covers every place the window reaches, and every kind of party', () => {
  const locations = new Set(views.map((v) => v.location))
  for (const l of ['castle', 'tavern', 'edge', 'training', 'settings', 'maze']) assert.ok(locations.has(l), l)
  assert.ok(views.some((v) => v.partyBar.every((p) => !p.name)), 'no party')
  assert.ok(views.some((v) => v.partyBar.some((p) => p.poisoned)), 'poison')
  assert.ok(views.some((v) => v.partyBar.some((p) => p.name && p.index < 0)), 'practice copies')
  assert.ok(views.some((v) => !v.settings.partyVisible), 'hidden party')
  assert.ok(views.some((v) => v.exits?.some((e) => !e.open)), 'closed exits')
})

test('every place Go reports, and every exit, has a title and a scene', () => {
  for (const v of views) {
    assert.ok(places[v.location], v.location)
    assert.ok(scenes[v.location], v.location)
    for (const e of v.exits ?? []) {
      assert.ok(places[e.dest], e.dest)
      assert.ok(scenes[e.dest], e.dest)
    }
  }
})

test('Back is the open way to the Castle from everywhere in town, and nowhere else', () => {
  for (const v of views) {
    const b = back(v)
    if (v.location === 'castle' || v.location === 'maze') {
      assert.equal(b, null, v.case)
      continue
    }
    assert.equal(b.dest, 'castle', v.case)
    assert.equal(b.label, 'Castle')
    assert.equal(b.open, true, v.case)
  }
})

test('the destinations are the other exits, in Go’s order, closed ones with Go’s reason', () => {
  for (const v of views) {
    const d = destinations(v)
    const home = v.exits?.some((e) => e.dest === 'castle') && v.location !== 'castle' ? ['castle'] : []
    assert.deepEqual([...home, ...d.map((x) => x.dest)].sort(), (v.exits ?? []).map((e) => e.dest).sort(), v.case)
    for (const x of d) {
      const e = v.exits.find((e) => e.dest === x.dest)
      assert.equal(x.open, e.open)
      assert.equal(x.reason, e.reason)
      if (!x.open) assert.ok(x.reason, `${v.case}: ${x.dest} closed without a reason`)
    }
  }
  assert.deepEqual(destinations(at('castle')).map((x) => x.dest), ['tavern', 'inn', 'trading', 'temple', 'edge'])
  assert.deepEqual(destinations(at('edge')).map((x) => x.dest), ['training', 'settings'])
  assert.match(destinations(at('castle, no party')).find((x) => x.dest === 'inn').reason, /NO PARTY/)
  assert.equal(destinations(at('castle')).find((x) => x.dest === 'temple').open, true)
})

test('while a character is being created, Back says it discards the character', () => {
  const b = back(at('creating'))
  assert.equal(b.open, true)
  assert.match(b.note, /discards the character being created/)
  assert.equal(back(at('training')).note, '')
})

test('the party bar shows six positions, the front three and the back three, empty ones included', () => {
  for (const v of views) {
    const [front, rear] = formation(v.partyBar, v.frontRow)
    assert.equal(front.slots.length, 3, v.case)
    assert.equal(rear.slots.length, 3, v.case)
    assert.deepEqual([...front.slots, ...rear.slots].map((p) => p.slot), [1, 2, 3, 4, 5, 6])
    for (const p of front.slots) assert.equal(p.front, true)
    for (const p of rear.slots) assert.equal(p.front, false)
  }
  const empty = at('castle').partyBar.filter((p) => !p.name)
  assert.equal(empty.length, 4)
  for (const p of empty) assert.equal(canOpen(p), false)
})

test('a member’s condition and HP are worded from Go’s data', () => {
  const second = at('castle').partyBar[1]
  assert.equal(condition(second), 'OK · POISONED')
  assert.equal(unwell(second), true)
  assert.equal(hpShare(second), second.hp / second.maxHp)
  const lead = at('castle').partyBar[0]
  assert.equal(condition(lead), 'OK')
  assert.equal(unwell(lead), false)
  assert.equal(hpShare({ hp: 5, maxHp: 0 }), 0)
  assert.equal(hpShare({ hp: -2, maxHp: 8 }), 0)
})

test('practice copies are shown with their own HP, and their sheets are not offered', () => {
  const practice = at('practice expedition')
  assert.equal(practice.partyBar[0].hp, 1)
  for (const p of practice.partyBar) assert.equal(canOpen(p), false)
  for (const p of at('real expedition').partyBar) assert.equal(canOpen(p), true)
})

// A WebP's width and height, from its header (simple lossy, lossless or
// extended).
function webpSize(buf) {
  assert.equal(buf.toString('ascii', 0, 4), 'RIFF')
  assert.equal(buf.toString('ascii', 8, 12), 'WEBP')
  const kind = buf.toString('ascii', 12, 16)
  if (kind === 'VP8 ') return [buf.readUInt16LE(26) & 0x3fff, buf.readUInt16LE(28) & 0x3fff]
  if (kind === 'VP8L') { const b = buf.readUInt32LE(21); return [(b & 0x3fff) + 1, ((b >> 14) & 0x3fff) + 1] }
  assert.equal(kind, 'VP8X')
  return [buf.readUIntLE(24, 3) + 1, buf.readUIntLE(27, 3) + 1]
}

test('ART-2A: the Castle and the Edge of Town have their pictures; every other place its themed panel', () => {
  const art = ['castle', 'edge']
  for (const [place, s] of Object.entries(scenes)) {
    assert.ok(s.emblem && s.alt, place)
    assert.match(s.focus, /^\d+% \d+%$/, place)
    assert.equal(illustrated(place), art.includes(place), place)
    if (!art.includes(place)) {
      assert.equal(s.src, null, place)
      continue
    }
    const file = new URL(s.src)
    assert.equal(file.protocol, 'file:', place)
    assert.match(file.pathname, new RegExp(`/src/assets/scenes/${place}\\.webp$`), place)
    assert.deepEqual(webpSize(readFileSync(file)), [1920, 1080], `${place}: 1920x1080`)
  }
  assert.equal(illustrated('nowhere'), false)
  assert.equal(illustrated(undefined), false)
  assert.equal(sceneFor('nowhere'), fallback)
  assert.equal(sceneFor(undefined), fallback)
})

test('the Temple opens for a party, and gives the castle’s answer without one', () => {
  const closed = destinations(at('castle, no party')).find((x) => x.dest === 'temple')
  assert.deepEqual([closed.open, closed.reason], [false, 'YOU HAVE NO PARTY!'])
  assert.equal(destinations(at('castle')).find((x) => x.dest === 'temple').open, true)
  for (const v of views) for (const d of destinations(v)) assert.doesNotMatch(d.reason ?? '', /not open in Istary/)
})

test('the Inn opens for a party, and gives the castle’s answer without one', () => {
  const closed = destinations(at('castle, no party')).find((x) => x.dest === 'inn')
  assert.deepEqual([closed.open, closed.reason], [false, 'YOU HAVE NO PARTY!'])
  assert.equal(destinations(at('castle')).find((x) => x.dest === 'inn').open, true)
})

test('Boltac’s opens for a party, and gives the castle’s answer without one', () => {
  const closed = destinations(at('castle, no party')).find((x) => x.dest === 'trading')
  assert.deepEqual([closed.open, closed.reason], [false, 'YOU HAVE NO PARTY!'])
  assert.equal(destinations(at('castle')).find((x) => x.dest === 'trading').open, true)
})

test('the Edge of Town’s departures carry Go’s reasons', () => {
  assert.match(at('castle, no party').departures.real, /Form a party/)
  assert.match(at('castle, no party').departures.practice, /Form a party/)
  assert.deepEqual(at('edge').departures, { real: '', practice: '' })
  assert.ok(at('real expedition').departures.real, 'a second expedition while one is out')
})
