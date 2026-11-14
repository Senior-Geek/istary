// COMBAT-1: the spell picker's, Party Retreat's and the escape's display
// helpers, over hand-made values and the real views Go recorded
// (testdata/combat-views.json).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { firstSpell, spellLevel, spellStatus, runNote, escapeLine } from './combat.js'
import { knownSpells } from './sheet.js'

const views = JSON.parse(readFileSync(new URL('./testdata/combat-views.json', import.meta.url)))
const src = (f) => readFileSync(new URL(f, import.meta.url), 'utf8')

const halito = { name: 'HALITO', level: 1, kind: 'mage', castable: true, left: 2, max: 2, summary: 'A small ball of fire.' }
const spent = { ...halito, castable: false, left: 0, reason: 'No level 1 mage casts left (0 of 2); a stay at the Inn restores them.' }
const dumapic = { name: 'DUMAPIC', level: 1, kind: 'mage', castable: false, reason: 'Exploration only: cast it in Camp.' }

test('the picker describes the first castable spell until one is pointed at', () => {
  assert.equal(firstSpell([dumapic, halito]).name, 'HALITO')
  assert.equal(firstSpell([dumapic, spent]).name, 'DUMAPIC') // none castable: the first known
  assert.equal(firstSpell([]), null)
  assert.equal(spellLevel(halito), 'Mage level 1')
  assert.equal(spellLevel({ kind: 'priest', level: 3 }), 'Priest level 3')
})

test('whether a spell can be cast, and Go\'s reason when not', () => {
  assert.equal(spellStatus('ELENARA', halito), 'ELENARA can cast HALITO now.')
  assert.equal(spellStatus('ELENARA', spent),
    'ELENARA cannot cast HALITO: no level 1 mage casts left (0 of 2); a stay at the Inn restores them.')
  assert.equal(spellStatus('ELENARA', dumapic), 'ELENARA cannot cast DUMAPIC: exploration only: cast it in Camp.')
  assert.equal(spellStatus('ELENARA', null), '')
})

test('a Run order is explained in the review as the party\'s retreat', () => {
  assert.equal(runNote({ orders: [{ action: 'parry', member: 'A' }] }), '')
  const note = runNote({ orders: [{ action: 'run', member: 'ELENARA' }, { action: 'parry', member: 'BORIN' }] })
  assert.match(note, /^The party attempts to retreat: ELENARA will try to lead it away\./)
  assert.match(note, /if it fails, that turn is lost and the fight goes on/)
})

test('an escape attempt says how it went, from Go\'s outcome only', () => {
  assert.equal(escapeLine({ kind: 'party-run', escape: 'escaped', lines: ['THE PARTY FLED!'] }), 'The party escapes!')
  assert.equal(escapeLine({ kind: 'party-run', escape: 'failed', lines: ['A FAILED TO RUN!'] }), 'Escape failed — the fight goes on.')
  assert.equal(escapeLine({ kind: 'party-fight', escape: '' }), '')
  assert.equal(escapeLine({ kind: 'party-run', silent: true, escape: 'failed' }), '')
  assert.equal(escapeLine(null), '')
})

test('the recorded views: every listed spell is described, with points, and a reason when it cannot be cast', () => {
  let listed = 0
  for (const v of views) {
    for (const sp of v.spells ?? []) {
      listed++
      assert.ok(sp.summary && sp.affects, sp.name)
      assert.ok(sp.left >= 0 && sp.max >= sp.left, sp.name)
      assert.equal(!sp.castable, !!sp.reason, sp.name)
      if (!sp.castable) assert.equal(sp.target, 'none', sp.name)
    }
    if (v.phase === 'orders') assert.ok(v.retreat, 'a retreat is offered at order entry')
    else assert.equal(v.retreat, undefined)
  }
  assert.ok(listed > 0)
  assert.ok(views.some((v) => (v.spells ?? []).some((sp) => !sp.castable)), 'a spell that cannot be cast is shown')
})

test('the character sheet takes its spell descriptions from Go\'s guide', () => {
  const guide = [{ name: 'HALITO', level: 1, book: 'mage', summary: 'Fire.' }]
  assert.deepEqual(knownSpells(['HALITO', 'NEWSPELL'], guide), [guide[0], { name: 'NEWSPELL', level: 0, book: '', summary: '' }])
  assert.deepEqual(knownSpells(null, guide), [])
  // No screen writes a description of its own: the summaries shown are the
  // view's or the guide's.
  assert.match(src('./CharacterSheet.svelte'), /knownSpells\(ch\.spells, guide\.spells\)/)
  assert.match(src('./CombatScreen.svelte'), /\{described\.summary\}/)
  assert.match(src('./CampScreen.svelte'), /\{sp\.summary\}/)
  for (const f of ['./CombatScreen.svelte', './CharacterSheet.svelte', './CampScreen.svelte']) {
    assert.doesNotMatch(src(f), /ball of fire|hit points\.|damage to/, f)
  }
})

test('Party Retreat is one call, and spells that cannot be cast are described, not ordered', () => {
  const c = src('./CombatScreen.svelte')
  assert.match(c, /call\(S\.CombatRetreat\)/)
  assert.match(c, /if \(!sp\.castable\) return/)
  assert.match(c, /onmouseenter=\{\(\) => \(pointed = sp\.name\)\} onfocus=\{\(\) => \(pointed = sp\.name\)\}/)
})
