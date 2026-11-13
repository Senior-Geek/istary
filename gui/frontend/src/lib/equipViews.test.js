// Contract tests: the character sheet's equip helpers over real sheets
// recorded from Go (gui/equip_test.go TestEquipViewFixture writes
// testdata/equip-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { itemActions, kindLabel } from './equip.js'

const sheets = JSON.parse(readFileSync(new URL('./testdata/equip-views.json', import.meta.url)))
const at = (name) => {
  const s = sheets.find((x) => x.case === name)
  assert.ok(s, name)
  return s.sheet
}

test('a button is enabled exactly when Go gives no refusal, and shows Go’s reason', () => {
  for (const { case: name, sheet } of sheets) {
    for (const it of sheet.items) {
      const a = itemActions(it, sheet.equipNote)
      if (sheet.equipNote || !it.kind) {
        assert.equal(a, null, `${name}: ${it.name}`)
        continue
      }
      const why = it.equipped ? it.unequipRefusal : it.equipRefusal
      assert.deepEqual([a.label, a.equip, a.enabled, a.reason], [it.equipped ? 'Take off' : 'Equip', !it.equipped, !why, why], `${name}: ${it.name}`)
    }
  }
})

test('class, special items and stuck items are explained in Go’s words', () => {
  assert.equal(itemActions(at('mage: armour refused, an unknown staff').items[0], '').reason, 'A MAGE cannot use it.')
  assert.equal(itemActions(at('fighter, nothing worn').items[3], ''), null) // a scroll is carried, not equipped
  assert.equal(kindLabel(at('fighter, nothing worn').items[3]), 'carried')
  const stuck = at('fighter: chain mail worn, evil shield stuck')
  assert.match(itemActions(stuck.items[1], '').reason, /cursed shield is stuck/)
  const shield = itemActions(stuck.items[2], '')
  assert.deepEqual([shield.label, shield.enabled], ['Take off', false])
  assert.match(shield.reason, /only Boltac/)
  assert.equal(stuck.items[2].cursed, true)
  assert.equal(stuck.ac, 2)
})

test('equipping is closed during practice, and the sheet says why', () => {
  const closed = at('during practice')
  assert.match(closed.equipNote, /practice/)
  for (const it of closed.items) assert.equal(itemActions(it, closed.equipNote), null)
})

// UX-3B: the sheet's Worn & Wielded and Pack lists.
import { wornAndPack, removeAction, useAction, CAPACITY } from './equip.js'

test('Worn & Wielded and the Pack are one inventory: each item once, counted together', () => {
  for (const { case: name, sheet } of sheets) {
    const { worn, pack, carried } = wornAndPack(sheet)
    assert.equal(worn.length + pack.length, sheet.items.length, name)
    assert.ok(worn.every((it) => it.equipped) && pack.every((it) => !it.equipped), name)
    const places = [...worn, ...pack].map((it) => it.pos).sort((a, b) => a - b)
    assert.deepEqual(places, sheet.items.map((_, i) => i + 1), name)
    assert.equal(carried, `${sheet.items.length} / ${CAPACITY}`, name)
  }
  assert.equal(CAPACITY, 8)
  const f = wornAndPack(at('fighter: chain mail worn, evil shield stuck'))
  assert.deepEqual(f.worn.map((it) => it.name), ['CHAIN MAIL', 'EVIL SHIELD +3'])
  assert.deepEqual(f.pack.map((it) => it.name), ['SMALL SHIELD', 'SCROLL/HALITO'])
  assert.deepEqual(wornAndPack(null), { worn: [], pack: [], carried: `0 / ${CAPACITY}` })
})

test('Remove is Go’s take-off, and a cursed item says why it stays', () => {
  const sheet = at('fighter: chain mail worn, evil shield stuck')
  const { worn } = wornAndPack(sheet)
  const mail = removeAction(worn[0], sheet.equipNote)
  assert.deepEqual(mail, { label: 'Remove', enabled: true, reason: '' })
  const shield = removeAction(worn[1], sheet.equipNote)
  assert.equal(shield.enabled, false)
  assert.match(shield.reason, /cursed/)
  assert.equal(useAction(worn[0], sheet.equipNote), null, 'no Use for what is worn')
})

test('Use equips from the pack; what cannot be equipped has no working Use', () => {
  for (const { case: name, sheet } of sheets) {
    for (const it of wornAndPack(sheet).pack) {
      const a = useAction(it, sheet.equipNote)
      if (sheet.equipNote || !it.kind) {
        assert.equal(a, null, `${name}: ${it.name} has no Use`)
        continue
      }
      assert.equal(a.label, 'Use')
      assert.equal(a.enabled, !it.equipRefusal, `${name}: ${it.name}`)
      assert.equal(a.reason, it.equipRefusal)
      assert.equal(removeAction(it, sheet.equipNote), null, 'no Remove for the pack')
    }
  }
  const mage = wornAndPack(at('mage: armour refused, an unknown staff'))
  const mail = useAction(mage.pack[0], '')
  assert.deepEqual([mail.enabled, mail.reason], [false, 'A MAGE cannot use it.'])
  const scroll = wornAndPack(at('fighter, nothing worn')).pack.find((it) => it.name === 'SCROLL/HALITO')
  assert.equal(useAction(scroll, ''), null, 'a scroll is not used from the sheet')
})

test('while equipment cannot be changed, neither list offers a button', () => {
  const sheet = at('during practice')
  const { worn, pack } = wornAndPack(sheet)
  for (const it of worn) assert.equal(removeAction(it, sheet.equipNote || 'closed'), null)
  for (const it of pack) assert.equal(useAction(it, sheet.equipNote || 'closed'), null)
})
