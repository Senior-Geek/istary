// Contract tests: the Temple helpers over real views recorded from Go
// (gui/temple_test.go TestTempleViewFixture writes testdata/temple-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { patients, payers, defaultPayer, pickLabel, confirmation, receipt, risks, charged, condition } from './temple.js'

const views = JSON.parse(readFileSync(new URL('./testdata/temple-views.json', import.meta.url)))
const at = (name) => {
  const v = views.find((x) => x.case === name)
  assert.ok(v, name)
  return v
}
const idx = (v, name) => v.temple.view.patients.find((p) => p.name === name)?.index

test('the recording covers every outcome and every save outcome', () => {
  const moods = new Set(views.map((v) => receipt(v.temple)?.mood).filter(Boolean))
  for (const m of ['well', 'ashes', 'lost', 'gold']) assert.ok(moods.has(m), m)
  const tones = new Set(views.map((v) => receipt(v.temple)?.tone).filter(Boolean))
  assert.deepEqual([...tones].sort(), ['notDurable', 'saved', 'unsaved'])
  for (const v of views) assert.equal(v.location, 'temple', v.case)
})

test('patients: Go’s list, treatable exactly when Go gives no refusal; the LOST never', () => {
  for (const v of views) {
    const rows = patients(v.temple)
    assert.equal(rows.length, v.temple.view.patients.length, v.case)
    rows.forEach((r, i) => {
      const p = v.temple.view.patients[i]
      assert.equal(r.index, p.index)
      assert.equal(r.treatable, !!p.kind && !p.refusal, `${v.case}: ${p.name}`)
      assert.equal(r.reason, p.refusal)
      if (r.lost) assert.equal(r.treatable, false)
      if (r.treatable) assert.equal(r.donation, p.donation)
    })
  }
  const first = patients(at('patients').temple)
  const below = first.find((r) => r.name === 'BELOW')
  assert.equal(below.treatable, false)
  assert.match(below.reason, /OUT in the maze/)
  assert.ok(first.some((r) => r.name === 'DEAD' && !r.inParty && r.treatable), 'a patient outside the party')
  const lost = patients(at('ashes lost').temple).find((r) => r.name === 'ASHY')
  assert.deepEqual([lost.lost, lost.treatable, lost.condition, lost.donation], [true, false, 'Lost', null])
})

// quote is Go's recorded quote for name in case v (paid by the case's payer).
const quote = (v, name) => v.quotes[v.temple.view.patients.findIndex((p) => p.name === name)]

test('the risk is worded by kind, never as a chance', () => {
  const v = at('patients')
  for (const [name, kind] of [['PARA', 'cure'], ['DEAD', 'raise'], ['ASHY', 'ashes']]) {
    const c = confirmation(v.temple, idx(v, name), 0, quote(v, name))
    assert.equal(c.kind, kind)
    assert.equal(c.risk, risks[kind])
    assert.equal(c.charged, charged)
    assert.equal(c.permanent, kind === 'ashes')
    for (const s of [c.risk, ...c.outcomes]) assert.doesNotMatch(s, /\d+ ?%|chance of|odds/i)
  }
  assert.doesNotMatch(JSON.stringify(v.temple.view) + JSON.stringify(v.quotes), /chance|percent|threshold/i, 'Go sends no chance')
})

test('the confirmation follows Go’s quote: the payer alone, the party’s gold, or Go’s refusal', () => {
  const v = at('patients')
  assert.equal(payers(v.temple)[defaultPayer(v.temple)].name, 'AGUI')
  const c = confirmation(v.temple, idx(v, 'PARA'), 0, quote(v, 'PARA'))
  assert.deepEqual([c.patient, c.condition, c.donation, c.payer, c.ready, c.party], ['PARA', 'Paralysed', 300, 'AGUI', true, false])
  // CGUI (40 GP) tithing: the party's gold meets it; each member's share is Go's.
  const w = at('quotes: CGUI tithing')
  const q = quote(w, 'PARA')
  assert.equal(q.mode, 'party')
  const p = confirmation(w.temple, idx(w, 'PARA'), 2, q)
  assert.deepEqual([p.ready, p.party, p.payer, p.donation, p.partyGold], [true, true, 'CGUI', 300, q.partyGold])
  assert.deepEqual(p.shares, q.shares.map((s) => ({ name: s.name, gold: s.gold })))
  assert.match(p.why, /entire party’s gold goes into the fund/)
  // Go's refusals, and no quote yet: not ready.
  assert.equal(confirmation(v.temple, idx(v, 'PARA'), 0, null).ready, false)
  const refused = confirmation(v.temple, idx(v, 'PARA'), 0, { refusal: 'CHEAP APOSTATES! OUT!' })
  assert.deepEqual([refused.ready, refused.why], [false, 'CHEAP APOSTATES! OUT!'])
  const out = confirmation(v.temple, idx(v, 'BELOW'), 0, quote(v, 'BELOW'))
  assert.equal(out.ready, false)
  assert.match(out.why, /OUT in the maze/)
  const gone = at('ashes lost')
  assert.equal(confirmation(gone.temple, idx(gone, 'ASHY'), 0, quote(gone, 'ASHY')), null, 'the LOST: nothing to confirm')
})

test('the party bar chooses a payer Go allows', () => {
  const v = at('patients')
  for (const p of payers(v.temple)) assert.equal(pickLabel(v.temple, p.slot, -1), p.refusal ? null : 'Tithe', p.name)
  assert.equal(pickLabel(v.temple, 0, 0), null, 'the payer already chosen')
})

test('receipts: restrained words for each outcome; the donation, HP and age as Go reports', () => {
  const cured = receipt(at('cured').temple)
  assert.equal(cured.title, 'PARA is well again.')
  assert.ok(cured.lines.includes('Paralysed → Well'))
  assert.ok(cured.lines.some((l) => /^Donation 300 GP, tithed by AGUI: 5000 → 4700 GP$/.test(l)))
  const ash = receipt(at('raise failed: ashes').temple)
  assert.equal(ash.mood, 'ashes')
  assert.equal(ash.title, 'The priests could not raise DEAD. DEAD is now ashes.')
  assert.ok(ash.lines.includes('Age unchanged'))
  const raised = receipt(at('ashes raised').temple)
  const t = at('ashes raised').temple.last.treat
  assert.ok(raised.lines.includes(`Hit points 0 → ${t.maxHp} of ${t.maxHp}`), 'from ashes: every hit point')
  const lost = receipt(at('ashes lost').temple)
  assert.equal(lost.title, 'ASHY could not be restored, and is lost forever.')
  assert.ok(lost.lines.includes('Ashes → Lost'))
  const party = receipt(at('party: raise failed').temple)
  const t2 = at('party: raise failed').temple.last.treat
  assert.equal(party.mood, 'ashes')
  assert.ok(party.lines.some((l) => l.startsWith('The party’s gold went into the fund: ') && t2.shares.every((s) => l.includes(`${s.name} ${s.gold} GP`)) &&
    l.includes(`CGUI’s own ${t2.payerBefore} GP`) && l.endsWith(`${t2.party.total} GP in all`)), party.lines)
  assert.ok(party.lines.includes(`Donation ${t2.donation} GP from the fund; CGUI keeps ${t2.payerAfter} GP`))
  const pooled = receipt(at('pooled').temple)
  assert.equal(pooled.lines.length, at('pooled').temple.last.gold.names.length)
})

test('NOT saved, then saved again, then not durable: the receipt follows Go', () => {
  assert.equal(receipt(at('cured, NOT saved').temple).tone, 'unsaved')
  assert.ok(at('cured, NOT saved').unsaved)
  assert.equal(receipt(at('saved again').temple).tone, 'saved')
  assert.equal(receipt(at('raised, not durable').temple).tone, 'notDurable')
  assert.equal(receipt(at('refused: short').temple), null, 'a refusal before any treatment: no receipt')
  assert.equal(condition('PARALYZE'), 'Paralysed')
})
