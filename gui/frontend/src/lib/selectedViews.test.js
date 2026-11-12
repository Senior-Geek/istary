// The Selected Character panel and the Trading Post's layout (UX-2), over
// real views recorded from Go (gui/shop_test.go TestShopViewFixture).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { selected, wornAndCarried, customer, pickLabel } from './shop.js'
import { itemActions } from './equip.js'

const views = JSON.parse(readFileSync(new URL('./testdata/shop-views.json', import.meta.url)))
const at = (name) => {
  const v = views.find((x) => x.case === name)
  assert.ok(v, name)
  return v
}
// A page view as the screen sees it: the customer is somewhere in the roster.
const page = (c) => ({ shop: c.shop, partyBar: c.partyBar, roster: c.customer ? [{ index: 99, name: 'OTHER' }, c.customer] : [] })

test('the panel shows the customer at the counter, found through the party bar', () => {
  for (const c of views) {
    const ch = selected(page(c))
    const who = customer(c.shop)
    if (!who) {
      assert.equal(ch, null, c.case)
      continue
    }
    assert.equal(ch.name, who.name, c.case)
    assert.equal(ch.gold, who.gold, `${c.case}: the panel's gold is Boltac's`)
    assert.equal(ch.partySlot, who.slot + 1, `${c.case}: a party member`)
    assert.equal(8 - (ch.items?.length ?? 0), who.room, `${c.case}: carrying agrees with Boltac's room`)
  }
  assert.equal(selected({ shop: at('who').shop, partyBar: [], roster: [] }), null)
})

test('equipped and carried are told apart, every item once', () => {
  for (const c of views.filter((c) => c.customer)) {
    const { worn, carried } = wornAndCarried(c.customer)
    const items = c.customer.items ?? []
    assert.equal(worn.length + carried.length, items.length, c.case)
    assert.ok(worn.every((it) => it.equipped) && carried.every((it) => !it.equipped), c.case)
    // Keys are unique even when Go leaves the items unnumbered (equipping closed).
    const keys = [...worn, ...carried].map((it) => it.pos)
    assert.equal(new Set(keys).size, items.length, `${c.case}: unique keys`)
    // The panel's items are Boltac's items, equipped alike.
    const owned = c.shop.view.items ?? []
    assert.deepEqual(items.map((it) => [it.name, it.equipped]), owned.map((it) => [it.name, it.equipped]), c.case)
  }
  assert.deepEqual(wornAndCarried(null), { worn: [], carried: [] })
})

test('Equip and Take off are the sheet’s, with Go’s reasons; none while Boltac asks', () => {
  const asks = at('Boltac asks: unusable').customer
  assert.equal(asks.equipNote, "Answer Boltac's question first.")
  assert.ok(asks.items.every((it) => itemActions(it, asks.equipNote) === null))
  const mage = at('mage at the counter').customer
  const sword = mage.items.find((it) => it.name === 'LONG SWORD')
  assert.deepEqual([itemActions(sword, '').label, itemActions(sword, '').enabled], ['Take off', true])
  const helm = mage.items.find((it) => it.name === 'HELM')
  assert.equal(itemActions(helm, '').reason, 'A MAGE cannot use it.')
  const done = at('fighter: equipped at the counter')
  assert.match(done.notice, /AGUI equipped the STAFF\. AC \d+\. Saved\./)
  const staff = done.customer.items.find((it) => it.name === 'STAFF' && it.equipped)
  assert.ok(staff, 'a staff is worn')
  const before = at('bought, not durable').customer // the same fighter, before
  assert.ok(!before.items.some((it) => it.equipped), 'nothing was worn before')
  assert.equal(done.shop.view.items.find((it) => it.n === staff.n).equipped, true, 'Boltac sees it worn too')
  assert.equal(done.shop.view.customer, 0)
})

test('the party bar chooses the customer; nobody while Boltac asks', () => {
  const c = at('mage at the counter')
  assert.equal(pickLabel(c.shop, c.shop.view.customer), null)
  assert.equal(pickLabel(c.shop, 0), 'Serve instead')
  assert.equal(pickLabel(at('Boltac asks: equipped').shop, 0), null)
})

test('the Trading Post: no sheet button, shop actions only, the list alone scrolls', () => {
  const src = readFileSync(new URL('./TradingScreen.svelte', import.meta.url), 'utf8')
  const actions = src.slice(src.indexOf('{#snippet actions()}'), src.indexOf('{/snippet}'))
  assert.ok(!/Character sheet|onopen/.test(actions), 'no Character sheet button in the Actions')
  assert.ok(!/who\.gold|who\.room|who\.items/.test(actions), 'no gold or carrying in the Actions')
  assert.match(actions, /Pool party gold/)
  assert.match(actions, /Leave the counter/)
  assert.ok(!/\{#each members/.test(actions), 'no customer buttons in the Actions: the party bar chooses')
  assert.match(actions, /Back to the Castle/, 'with nobody to serve, the way back')
  assert.match(src, /<LocationFrame [^>]*\bfill\b/)
  assert.match(src, /<LocationFrame [^>]*\bpartyAlways\b/, 'the party bar is always shown at Boltac’s')
  assert.match(src, /let tab = \$state\('buy'\)/, 'a visit opens on the Buy tab')
  assert.match(src, /<SelectedCharacter /)
  assert.match(src, /\.tablewrap \{ overflow: auto;/)
  const frame = readFileSync(new URL('./LocationFrame.svelte', import.meta.url), 'utf8')
  assert.match(frame, /\.frame\.fill \.body \{ overflow: hidden;/)
  const panel = readFileSync(new URL('./SelectedCharacter.svelte', import.meta.url), 'utf8')
  assert.match(panel, /\.inventory \{ min-height: 0; overflow-y: auto;/)
})
