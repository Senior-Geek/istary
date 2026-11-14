// NAV-1: the Castle and the Edge of Town, checked in the components'
// source (as layout.test.js is) and in views recorded from Go
// (testdata/frame-views.json). The Castle is the hub, its destinations
// only; the Edge's actions are Enter the Maze, the Training Grounds,
// Select Game, Settings and Leave the Game, with no Go to list and no
// practice; the top bar has Castle (Esc) and the toggles (party, story,
// sheet, panels), and never Settings.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { destinations } from './navigation.js'

const read = (name) => readFileSync(new URL(name, import.meta.url), 'utf8')
const code = (src) => src.replace(/<!--[\s\S]*?-->/g, '').replace(/^\s*\/\/.*$/gm, '')
const frame = read('./LocationFrame.svelte')
const head = frame.slice(frame.indexOf('<header class="head">'), frame.indexOf('</header>'))
const views = JSON.parse(readFileSync(new URL('./testdata/frame-views.json', import.meta.url)))
const at = (c) => views.find((v) => v.case === c)

test('the top bar: Castle (Esc), then the party, story, sheet and panels toggles, and no Settings', () => {
  const order = [
    'onclick={() => call(S.ReturnToCastle)}',
    "{partyShown ? 'Hide party' : 'Show party'}",
    '>Story</button>',
    '{toggleLabel(sheetShown)}',
    '>Hide panels</button>',
  ].map((x) => head.indexOf(x))
  assert.ok(order.every((i) => i > 0), `all five in the head (${order})`)
  assert.deepEqual([...order].sort((a, b) => a - b), order, 'in that order')
  assert.match(head, /← \{way\.label\} <kbd>Esc<\/kbd>/, 'Castle with its key')
  assert.match(head, /\{#if sheetHere\}\s*<button class="toggle" aria-pressed=\{sheetShown\}/, 'the sheet toggle is a toggle, pressed while shown')
  assert.ok(!/Settings|settings/.test(code(head).replace(/view\.settings\?\.partyVisible/g, '')), 'no Settings in the top bar')
})

test('the sheet toggle is the top bar’s only: none in the Actions, and the frame offers one', () => {
  const nav = frame.slice(frame.indexOf('data-area="actions"'), frame.indexOf('data-area="information"'))
  assert.ok(!/toggleLabel|toggleSheet|sheet-toggle/.test(nav), 'nothing of the sheet in the actions area')
  assert.equal(frame.match(/\{toggleLabel\(sheetShown\)\}/g).length, 1, 'one toggle')
  assert.match(nav, /\{#if actions\}\s*<section class="actions"/, 'Actions only when the place has some')
})

test('the Castle: its five destinations, and no Actions', () => {
  const castle = read('./CastleScreen.svelte')
  assert.ok(!/\{#snippet actions/.test(castle), 'no actions of its own')
  assert.deepEqual(destinations(at('castle')).map((d) => d.label),
    ["Gilgamesh's Tavern", "The Adventurer's Inn", "Boltac's Trading Post", 'The Temple of Cant', 'The Edge of Town'])
})

test('the Edge of Town: its five actions in order, its destinations among them, no Go to list', () => {
  const edge = read('./EdgeScreen.svelte')
  assert.match(edge, /<LocationFrame [^>]*\bwaysInActions\b/, 'the frame draws no Go to list here')
  assert.match(frame, /\{#if ways\.length && !waysInActions\}\s*<nav aria-label="Destinations">/)
  const order = [
    '>Enter the Maze</button>',
    "w.dest === 'training'",
    'call(S.ChangeGame)}>Select Game</button>',
    "w.dest === 'settings'",
    'call(S.LeaveGame)}>Leave the Game</button>',
  ].map((x) => edge.indexOf(x))
  assert.ok(order.every((i) => i > 0), `all five (${order})`)
  assert.deepEqual([...order].sort((a, b) => a - b), order, 'in that order')
  // The two destinations are Go's exits, drawn by the frame's one snippet.
  assert.deepEqual(destinations(at('edge')).map((d) => [d.dest, d.label]), [['training', 'The Training Grounds'], ['settings', 'Settings']])
  assert.equal((edge.match(/\{@render destination\(w\)\}/g) ?? []).length, 2)
  assert.ok(!/S\.Navigate, 'training'|S\.Navigate, 'settings'/.test(edge), 'no second way to go there')
  assert.ok(!/Change game/.test(edge), 'renamed')
})

test('the Edge offers no practice; its Session methods stay for the tests', () => {
  const edge = code(read('./EdgeScreen.svelte'))
  assert.ok(!/Practice|practice|checkbox|guards a chest/.test(edge), 'no practice controls')
  const bindings = read('../../wailsjs/go/main/Session.js')
  for (const m of ['PracticeExpedition', 'PracticeEncounter', 'AbandonPractice']) assert.match(bindings, new RegExp(`export function ${m}\\(`), m)
})

// The labels' case (NAV-1 decision 2): navigation destinations and the
// actions that move the player or manage the game in Title Case; the
// interface's toggles in sentence case. A minor word (a, an, the, to,
// into, with, of) after the first stays lower case.
const minor = new Set(['a', 'an', 'the', 'to', 'into', 'with', 'of'])
const titleCase = (s) => s.replace(/^←\s*/, '').split(/\s+/).every((w, i) => (i > 0 && minor.has(w)) || /^[A-Z…]/.test(w))

test('navigation and game management in Title Case, toggles in sentence case', async () => {
  const { choices } = await import('./games.js')
  const games = { legacy: { problem: '' } }
  const labels = [
    ...choices({ slot: 1, state: 'empty' }, games), ...choices({ slot: 1, state: 'game' }, games),
    ...choices({ slot: 1, state: 'damaged' }, games), ...choices({ slot: 1, state: 'game' }, { ...games, replacing: 1 }),
  ].map((c) => c.label)
  const edge = read('./EdgeScreen.svelte'), app = read('../App.svelte')
  const shown = (src, ...names) => names.map((n) => { assert.ok(src.includes(`>${n}</button>`), n); return n })
  labels.push(...shown(edge, 'Enter the Maze', 'Select Game', 'Leave the Game', 'Continue into Maze', 'Return to Town'))
  labels.push(...shown(app, 'Try Resuming Again', 'Return Its Party to the Castle'))
  labels.push(...destinations(at('castle')).map((d) => d.label), ...destinations(at('edge')).map((d) => d.label))
  for (const l of labels) assert.ok(titleCase(l), `Title Case: ${l}`)
  assert.ok(labels.length >= 18, `${labels.length} labels`)
  const { toggleLabel } = await import('./sheet.js')
  for (const t of ['Hide party', 'Show party', toggleLabel(true), toggleLabel(false), 'Hide panels', 'Show panels', 'Story']) {
    assert.ok(/^[A-Z][a-z]*( [a-z]+)*$/.test(t), `sentence case: ${t}`)
  }
  assert.ok(frame.includes("'Hide party' : 'Show party'") && frame.includes('>Hide panels</button>') && frame.includes('>Show panels <kbd>'), 'the frame’s toggles as listed')
})
