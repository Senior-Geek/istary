// Contract tests: the Camp helpers over real panels recorded from Go
// (gui/camp_test.go TestCampPanelFixture writes testdata/camp-views.json
// from seeded practice expeditions).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { labels, operation, partyPick, back, buttons, menuOrder, inspectOrder, spellBook, whyNot } from './camp.js'

const panels = JSON.parse(readFileSync(new URL('./testdata/camp-views.json', import.meta.url)))

test('the recording covers every Camp screen the GUI reaches', () => {
  const screens = new Set(panels.map((p) => p.view.screen))
  for (const s of ['menu', 'reorder', 'inspect', 'equip', 'drop', 'trade', 'tradeGold', 'tradeItem',
    'spellBooks', 'spellList', 'spellTarget', 'dumapic']) assert.ok(screens.has(s), s)
})

test('every action Go lists has a label, and each screen says what it is doing', () => {
  for (const p of panels) {
    for (const a of p.view.actions) assert.ok(labels[a], a)
    assert.ok(operation(p).length > 0, p.view.screen)
  }
})

test('Disband is never offered: it is shown disabled with Go’s reason', () => {
  for (const p of panels) {
    assert.ok(!p.view.actions.includes('disband'))
    if (p.view.screen !== 'menu') continue
    const d = buttons(p, menuOrder).find((b) => b.action === 'disband')
    assert.equal(d.enabled, false)
    assert.match(d.reason, /#26/)
  }
})

test('a party of one sees why it cannot reorder', () => {
  const one = panels.find((p) => p.view.screen === 'menu' && p.party.length === 1)
  assert.ok(one)
  const r = buttons(one, menuOrder).find((b) => b.action === 'reorder')
  assert.equal(r.enabled, false)
  assert.ok(r.reason)
})

test('buttons enable exactly the actions Go lists', () => {
  for (const p of panels) {
    const order = p.view.screen === 'menu' ? menuOrder : p.view.screen === 'inspect' ? inspectOrder : []
    for (const b of buttons(p, order)) assert.equal(b.enabled, p.view.actions.includes(b.action), b.action)
    for (const a of p.view.actions.filter((a) => order.includes(a)))
      assert.ok(buttons(p, order).some((b) => b.action === a && b.enabled), a)
  }
})

test('clicking a party member performs only a listed action, never on the members Go refuses', () => {
  let picked = 0
  for (const p of panels) {
    p.party.forEach((m, slot) => {
      const pick = partyPick(p, slot)
      if (!pick) return
      picked++
      assert.ok(p.view.actions.includes(pick.action), pick.action)
      if (pick.action === 'trade-with') assert.notEqual(slot, p.view.member)
      if (pick.action === 'reorder-pick') assert.ok(!p.view.placed?.includes(slot))
    })
  }
  assert.ok(picked > 0)
})

test('Escape leaves a screen only by a listed action', () => {
  for (const p of panels) {
    const b = back(p)
    if (b) assert.ok(p.view.actions.includes(b))
  }
})

test('MALOR is never castable, and every spell that is not says why', () => {
  for (const p of panels) {
    for (const sp of p.view.spells ?? []) {
      if (sp.name === 'MALOR') assert.equal(sp.castable, false)
      assert.equal(sp.castable, !sp.reason, sp.name)
    }
  }
})

test('the spell list shows the open book’s known spells, by level', () => {
  const list = panels.find((p) => p.view.screen === 'spellList')
  const book = spellBook(list.view.spells, list.view.spellBook)
  const shown = book.flatMap((l) => l.spells)
  assert.deepEqual(shown, list.view.spells.filter((sp) => sp.book === list.view.spellBook))
  for (let i = 1; i < book.length; i++) assert.ok(book[i].level > book[i - 1].level)
})

test('equip choices carry the names the sheet shows (unknown names until identified)', () => {
  for (const p of panels.filter((p) => p.view.choices?.length)) {
    const items = p.party[p.view.member].items
    for (const c of p.view.choices) assert.equal(c.name, items[c.item - 1].name)
  }
})

test('DUMAPIC’s result is only on its own screen', () => {
  for (const p of panels) assert.equal(!!p.view.dumapic, p.view.screen === 'dumapic')
})

test('every spell that cannot be cast is listed once, under its reason', () => {
  for (const p of panels) {
    const groups = whyNot(p.view.spells)
    const names = groups.flatMap((g) => g.names)
    assert.deepEqual(names.sort(), (p.view.spells ?? []).filter((sp) => !sp.castable).map((sp) => sp.name).sort())
    for (const g of groups) for (const n of g.names) assert.equal(p.view.spells.find((sp) => sp.name === n).reason, g.reason)
  }
})
