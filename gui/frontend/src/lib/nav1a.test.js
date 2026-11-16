// NAV-1A: Istary opens at the Castle, with or without a game. With none
// (view.game 0) the Castle's only open destination is the Edge of Town,
// the others closed with Go's reason; the Edge offers Select Game,
// Settings and Leave the Game, the maze and the Training Grounds closed;
// Settings has only the appearance; the Castle's story is the
// introductory one (chosen by Go, CONTENT-1A); and game selection has a
// way back to the Castle.
// Checked in views recorded from Go (testdata/frame-views.json) and in
// the components' source, as nav1.test.js is.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { destinations, back } from './navigation.js'

const read = (name) => readFileSync(new URL(name, import.meta.url), 'utf8')
const views = JSON.parse(readFileSync(new URL('./testdata/frame-views.json', import.meta.url)))
const at = (c) => views.find((v) => v.case === c)
const REASON = 'Select a game first: Select Game is at the Edge of Town.'

test('with no game, the Castle opens only the Edge of Town, the others saying why', () => {
  const v = at('no game, castle')
  assert.equal(v.location, 'castle')
  assert.equal(back(v), null, 'no way back from the Castle')
  assert.deepEqual(destinations(v).map((d) => [d.label, d.open, d.reason]), [
    ["Gilgamesh's Tavern", false, REASON],
    ["The Adventurer's Inn", false, REASON],
    ["Boltac's Trading Post", false, REASON],
    ['The Temple of Cant', false, REASON],
    ['The Edge of Town', true, ''],
  ])
  assert.ok(v.partyBar.length === 6 && v.partyBar.every((p) => !p.name), 'an empty party bar')
})

test('with no game, the Edge closes the maze and the Training Grounds, and Settings is open', () => {
  const v = at('no game, edge')
  assert.equal(back(v)?.open, true, 'Castle (Esc)')
  assert.deepEqual(destinations(v).map((d) => [d.dest, d.open, d.reason]), [['training', false, REASON], ['settings', true, '']])
  assert.equal(v.departures.real, REASON, 'Enter the Maze closed, with the reason')
  const edge = read('./EdgeScreen.svelte')
  assert.match(edge, /disabled=\{busy \|\| !!real \|\| checking\} onclick=\{enter\}><Emblem location="maze" size="22px" \/>Enter the Maze/, 'Enter the Maze follows Go')
  assert.match(edge, /\{#if real\}<span class="why">\{real\}<\/span>/, 'and shows its reason')
  assert.match(edge, /<button class="wide" disabled=\{busy\} onclick=\{\(\) => call\(S\.ChangeGame\)\}>Select Game<\/button>/, 'Select Game whether or not a game is open')
  assert.ok(!/\{#if view\.game\}\s*<button[^>]*ChangeGame/.test(edge), 'not only in a game')
  assert.match(edge, /call\(S\.LeaveGame\)\}>Leave the Game/, 'Leave the Game')
  const s = at('no game, settings')
  assert.deepEqual(destinations(s), [], 'Settings: only the way back')
  assert.equal(back(s)?.open, true)
})

test('Settings with no game: the appearance, not the roster tools', () => {
  const src = read('./SettingsScreen.svelte')
  const actions = src.slice(src.indexOf('{#snippet actions()}'), src.indexOf('{/snippet}'))
  assert.match(actions, /\{#if view\.game\}[\s\S]*ReloadRoster[\s\S]*Reset roster[\s\S]*Restore roster[\s\S]*\{:else\}/, 'the roster tools only in a game')
  assert.match(src, /\{#if view\.game\}<section>\s*<h2>Save folder<\/h2>/, 'the game’s save folder only in a game')
  assert.ok(src.indexOf('class="appearance"') > 0 && !/\{#if view\.game\}\s*<section class="appearance"/.test(src), 'the appearance always')
})

test('the Castle’s story: the introductory one with no game, its own in one', () => {
  // Go chooses it (story.go, CONTENT-1A): its text is its file's, checked
  // against the approved text in Go (story_test.go) and drawn in
  // content1a.test.js.
  const intro = at('no game, castle').story, own = views.find((v) => v.case === 'castle').story
  assert.ok(intro?.title && intro.blocks.at(-1).kind === 'challenge', 'titled, ending in its challenge')
  assert.ok(own && !own.title && own.blocks.every((b) => b.kind !== 'challenge'), 'the Castle’s own in a game, titled with the place')
  const src = read('./CastleScreen.svelte')
  assert.match(src, /\{#if !view\.game\}\s*<section class="no-game"/, 'and the information says no game is selected')
  // App opens every place's story box again when the game changes, so the
  // Castle's own story is shown once a game is chosen (each place's own
  // since ART-3B-1).
  const app = read('../App.svelte')
  assert.match(app, /if \(view && !view\.fatal && game !== storyGame\) \{\s*storyGame = game\s*storyClosed = \{\}/)
})

test('game selection has ← Castle (Esc), which App’s Escape also is', () => {
  const src = read('./GamesScreen.svelte')
  assert.match(src, /\{#if view\.return\?\.open\}\s*<button class="back" disabled=\{busy\} onclick=\{\(\) => call\(S\.ReturnToCastle\)\}>← Castle <kbd>Esc<\/kbd><\/button>/)
  const app = read('../App.svelte')
  const onkey = app.slice(app.indexOf('function onkey'), app.indexOf('</script>'))
  assert.ok(!/screen === 'games'/.test(onkey), 'Escape is not kept from game selection')
  assert.match(onkey, /if \(!view\?\.return\?\.open\) return[\s\S]*call\(S\.ReturnToCastle\)/)
})
