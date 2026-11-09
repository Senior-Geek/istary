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
