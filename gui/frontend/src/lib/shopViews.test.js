// Contract tests: the Trading Post helpers over real views recorded from Go
// (gui/shop_test.go TestShopViewFixture writes testdata/shop-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { tabs, rows, words, boltac, customer, stockText, marks, empty, question, pickLabel, receipt } from './shop.js'

const views = JSON.parse(readFileSync(new URL('./testdata/shop-views.json', import.meta.url)))
const at = (name) => {
  const v = views.find((x) => x.case === name)
  assert.ok(v, name)
  return v
}

test('the recording covers every step and every save outcome', () => {
  const steps = new Set(views.map((v) => v.shop.view.step))
  for (const s of ['who', 'main', 'buy', 'confirm', 'sell', 'identify', 'uncurse']) assert.ok(steps.has(s), s)
  const tones = new Set(views.map((v) => receipt(v.shop)?.tone).filter(Boolean))
  assert.deepEqual([...tones].sort(), ['notDurable', 'saved', 'unsaved'])
  for (const v of views) assert.equal(v.location, 'trading', v.case)
})

test('a button is enabled exactly when Go gives no refusal, and shows Go’s reason', () => {
  for (const v of views) {
    const sv = v.shop.view
    for (const t of tabs) {
      for (const r of rows(v.shop, t.key)) {
        if (t.key === 'buy') {
          const o = sv.catalog.find((o) => o.item === r.request.item)
          assert.equal(r.reason, words(o.refusal), `${v.case}: ${r.name}`)
          assert.equal(r.amount, o.price)
        } else {
          const it = sv.items.find((it) => it.n === r.request.n)
          const why = { sell: it.sellRefusal, identify: it.identifyRefusal, uncurse: it.uncurseRefusal }[t.key]
          assert.equal(r.reason, words(why), `${v.case}: ${t.key} ${r.name}`)
          assert.equal(r.amount, t.key === 'sell' ? it.price : it.fee)
        }
      }
    }
  }
})

test('every catalog line and every item is listed once, with the request Go takes', () => {
  for (const v of views) {
    const sv = v.shop.view
    if (sv.customer < 0) {
      for (const t of tabs) assert.deepEqual(rows(v.shop, t.key), [], `${v.case}: ${t.key}`)
      continue
    }
    assert.deepEqual(rows(v.shop, 'buy').map((r) => r.request), sv.catalog.map((o) => ({ action: 'buy', item: o.item })))
    for (const t of ['sell', 'identify', 'uncurse'])
      assert.deepEqual(rows(v.shop, t).map((r) => r.request), (sv.items ?? []).map((it) => ({ action: t, n: it.n })))
  }
})

test('refusals are Boltac’s own words: funds, room, cursed, identified, not cursed', () => {
  const reasons = new Set(views.flatMap((v) => tabs.flatMap((t) => rows(v.shop, t.key).map((r) => r.reason))))
  for (const want of ['YOU CANNOT AFFORD IT', 'YOU CANT CARRY ANYTHING MORE', 'WE DONT BUY CURSED ITEMS',
    'THAT HAS BEEN IDENTIFIED', 'THAT IS NOT A CURSED ITEM', 'YOU CANT AFFORD THE FEE']) assert.ok(reasons.has(want), want)
  const full = at('no room')
  assert.equal(customer(full.shop).room, 0)
  for (const r of rows(full.shop, 'buy')) assert.equal(r.reason, 'YOU CANT CARRY ANYTHING MORE', r.name)
})

test('unusable equipment can still be bought: Buy… stays open, and Boltac’s question is drawn', () => {
  const v = at('mage at the counter')
  const sword = rows(v.shop, 'buy').find((r) => r.name === 'LONG SWORD')
  assert.equal(sword.usable, false)
  assert.equal(sword.reason, '')
  assert.equal(sword.label, 'Buy…')
  assert.match(sword.note, /MAGE cannot use it/)
  const ask = at('Boltac asks: unusable')
  assert.deepEqual(question(ask.shop), { kind: 'buy', item: 'LONG SWORD', price: 25, class: 'MAGE' })
  assert.equal(boltac(ask.shop), '') // the question is drawn, not repeated
  assert.match(ask.return.note, /answers no/)
  assert.equal(question(at('bought anyway').shop), null)
  assert.equal(at('bought anyway').shop.last.item, 'LONG SWORD')
})

test('equipped, cursed and unidentified items are marked from Go’s flags', () => {
  const v = at('mage at the counter')
  const items = v.shop.view.items
  for (const r of rows(v.shop, 'sell')) {
    const it = items.find((it) => it.n === r.n)
    assert.deepEqual(r.marks.map((m) => m.key), marks(it).map((m) => m.key))
    assert.equal(r.marks.some((m) => m.key === 'equipped'), it.equipped)
    assert.equal(r.marks.some((m) => m.key === 'cursed'), it.cursed)
    assert.equal(r.marks.some((m) => m.key === 'unidentified'), !it.identified)
  }
  const keys = rows(v.shop, 'sell').flatMap((r) => r.marks.map((m) => m.key))
  for (const k of ['equipped', 'cursed', 'unidentified']) assert.ok(keys.includes(k), k)
})

test('the customer is chosen from the party bar and marked there', () => {
  const who = at('who')
  assert.equal(customer(who.shop), null)
  for (const p of who.partyBar.filter((p) => p.name)) assert.equal(pickLabel(who.shop, p.slot - 1), 'Enter')
  assert.match(empty(who.shop, 'buy'), /party member/)
  const mage = at('mage at the counter')
  const c = customer(mage.shop)
  assert.deepEqual([c.name, c.class, c.gold, c.room], ['EGUI', 'MAGE', 1000, 4])
  assert.equal(pickLabel(mage.shop, c.slot), null)
  assert.equal(pickLabel(mage.shop, 0), 'Serve instead')
  for (const p of at('Boltac asks: unusable').partyBar) assert.equal(pickLabel(at('Boltac asks: unusable').shop, p.slot - 1), null)
  assert.equal(at('who').partyBar.length, 6)
  assert.equal(at('who').frontRow, 3)
})

test('a customer with nothing says so on the item tabs', () => {
  const thief = at('thief: little gold, nothing to sell')
  assert.deepEqual(rows(thief.shop, 'sell'), [])
  assert.match(empty(thief.shop, 'sell'), /carries nothing/)
  assert.ok(rows(thief.shop, 'buy').some((r) => r.reason === 'YOU CANNOT AFFORD IT'))
})

test('Boltac’s words and stock read as he says them', () => {
  assert.equal(boltac(at('bought, saved').shop), 'JUST WHAT YOU NEEDED')
  assert.equal(boltac(at('refused: cannot afford').shop), 'YOU CANNOT AFFORD IT')
  assert.equal(stockText(-1), 'unlimited')
  assert.equal(stockText(2), '2 left')
  const scroll = rows(at('mage at the counter').shop, 'buy').find((r) => r.name === 'SCROLL/HALITO')
  assert.match(scroll.stock, /^\d+ left$/)
})

test('the receipt says saved, not durable, or NOT saved, as Go reports it', () => {
  const saved = receipt(at('bought, saved').shop)
  assert.deepEqual([saved.tone, saved.did, saved.status], ['saved', 'EGUI bought STAFF for 10 GP', 'Saved'])
  const failed = at('bought, NOT saved')
  assert.equal(receipt(failed.shop).tone, 'unsaved')
  assert.match(receipt(failed.shop).status, /NOT saved/)
  assert.ok(failed.unsaved && /Could not save/.test(failed.error))
  const shaky = at('bought, not durable')
  assert.equal(receipt(shaky.shop).tone, 'notDurable')
  assert.ok(shaky.warning)
  assert.equal(receipt(at('refused: cannot afford').shop), null)
  assert.match(receipt(at('sold').shop).did, /sold LONG SWORD for 12 GP/)
  assert.match(receipt(at('identified').shop).did, /identified EGUI's item for 50 GP: it is HELM/)
  assert.match(receipt(at('uncursed').shop).did, /uncursed EGUI's STAFF/)
})

test('a curse the player has not seen is never shown (#34): not before it is identified or worn', () => {
  let unseen = 0
  for (const v of views) {
    for (const it of v.shop.view.items ?? []) {
      if (it.identified || it.equipped) continue
      assert.equal(it.cursed, false, `${v.case}: ${it.name}`)
      assert.equal(it.sellRefusal === '** WE DONT BUY CURSED ITEMS **', false, `${v.case}: ${it.name}`)
      unseen++
    }
  }
  assert.ok(unseen > 0)
  // The recording's unidentified scroll does carry a curse: trying to sell it would say so.
  const mage = views.find((v) => v.case === 'mage at the counter')
  const scroll = rows(mage.shop, 'sell').find((r) => r.name === 'SCROLL')
  assert.deepEqual([scroll.reason, scroll.marks.map((m) => m.key)], ['', ['unidentified']])
})

test('selling an equipped item asks first (B7): Sell… says so, and the question names the item', () => {
  const mage = views.find((v) => v.case === 'mage at the counter')
  for (const r of rows(mage.shop, 'sell')) {
    const it = mage.shop.view.items.find((it) => it.n === r.n)
    assert.equal(r.label === 'Sell…', it.sellAsks, r.name)
    assert.equal(r.note !== '', it.sellAsks, r.name)
  }
  assert.ok(rows(mage.shop, 'sell').some((r) => r.label === 'Sell…'))
  const ask = views.find((v) => v.case === 'Boltac asks: equipped')
  const q = question(ask.shop)
  assert.equal(q.kind, 'sell')
  assert.equal(q.item, 'LONG SWORD')
  assert.equal(q.name, 'EGUI')
  assert.equal(boltac(ask.shop), '')
  assert.match(ask.return.note, /not sold/)
  for (const p of ask.partyBar) assert.equal(pickLabel(ask.shop, p.slot - 1), null)
  assert.equal(views.find((v) => v.case === 'sold').shop.last.action, 'sell')
  assert.equal(question(views.find((v) => v.case === 'Boltac asks: unusable').shop).kind, 'buy')
})
