// The resumed expedition's notice (M-ART-1A.3): shown for a few seconds
// over the maze view, then gone; at once when the player acts. On a fake
// clock: no real waiting.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resumeNotice, resumeNoticeOf, resumedTitle, shownFor, fadeFor } from './resumeNotice.js'

const read = (f) => readFileSync(new URL(f, import.meta.url), 'utf8')

// A clock the test moves by hand.
function clock() {
  let now = 0, next = 1
  const due = new Map()
  return {
    setTimeout: (fn, ms) => { const id = next++; due.set(id, { at: now + ms, fn }); return id },
    clearTimeout: (id) => { due.delete(id) },
    tick(ms) {
      const until = now + ms
      for (;;) {
        const [id, t] = [...due].filter(([, t]) => t.at <= until).sort((a, b) => a[1].at - b[1].at)[0] ?? []
        if (!t) break
        due.delete(id); now = t.at; t.fn()
      }
      now = until
    },
    get pending() { return due.size },
  }
}
const setup = ({ reduced = false } = {}) => {
  const c = clock(), changes = []
  const n = resumeNotice({ timers: c, reducedMotion: () => reduced, onchange: (s) => changes.push(s) })
  return { c, n, changes }
}

const text = 'Your party has returned to the dungeon. Stranded adventurers remain at their recorded locations.'
const maze = (over = {}) => ({ screen: 'dungeon', location: 'maze', dungeon: { phase: 'maze' }, notice: text, title: resumedTitle, ...over })
const resumedView = maze()
const walked = maze() // Go keeps the notice in later views
const refused = maze({ error: 'You cannot do that.' })
const town = { screen: 'roster', location: 'castle', notice: '', title: '' }

test('the resume notice is the one Go heads EXPEDITION RESUMED, alone', () => {
  assert.equal(resumeNoticeOf(resumedView), text)
  assert.equal(resumeNoticeOf({ ...resumedView, title: '' }), '')
  assert.equal(resumeNoticeOf({ notice: 'Saved.', title: '' }), '')
  assert.equal(resumeNoticeOf({ ...resumedView, fatal: 'x' }), '')
  assert.equal(resumeNoticeOf(null), '')
})

test('a resumed expedition: shown with its whole text, still shown before the four seconds are up', () => {
  const { c, n } = setup()
  n.see(resumedView)
  assert.deepEqual(n.state, { text, phase: 'shown' })
  c.tick(shownFor - 1)
  assert.deepEqual(n.state, { text, phase: 'shown' })
})

test('after four seconds it fades, then is gone; nothing is left to run', () => {
  const { c, n, changes } = setup()
  n.see(resumedView)
  c.tick(shownFor)
  assert.deepEqual(n.state, { text, phase: 'fading' })
  c.tick(fadeFor - 1)
  assert.equal(n.state.phase, 'fading')
  c.tick(1)
  assert.deepEqual(n.state, { text: '', phase: 'hidden' })
  assert.deepEqual(changes.map((s) => s.phase), ['shown', 'fading', 'hidden'])
  assert.equal(c.pending, 0)
  assert.ok(fadeFor >= 200 && fadeFor <= 300 && shownFor === 4000)
})

test('reduced motion: no fade, gone at once when the time is up', () => {
  const { c, n, changes } = setup({ reduced: true })
  n.see(resumedView)
  c.tick(shownFor)
  assert.deepEqual(n.state, { text: '', phase: 'hidden' })
  assert.deepEqual(changes.map((s) => s.phase), ['shown', 'hidden'])
  assert.equal(c.pending, 0)
  // And the style has no transition then.
  assert.match(read('./DungeonScreen.svelte'), /@media \(prefers-reduced-motion: reduce\) \{ \.resumed \{ transition: none; \} \}/)
})

test('the player acting in the dungeon dismisses it at once; its timer goes with it', () => {
  for (const after of [walked, maze({ dungeon: { phase: 'camp' } })]) {
    const { c, n } = setup()
    n.see(resumedView)
    c.tick(1000)
    n.see(after, true)
    assert.deepEqual(n.state, { text: '', phase: 'hidden' })
    assert.equal(c.pending, 0)
    c.tick(10000)
    assert.equal(n.state.phase, 'hidden')
  }
  // While it fades, too.
  const { c, n } = setup()
  n.see(resumedView); c.tick(shownFor + 10)
  n.see(walked, true)
  assert.equal(n.state.phase, 'hidden')
  assert.equal(c.pending, 0)
})

test('a refused action does not dismiss it, nor does a view the player did not act for', () => {
  const { c, n } = setup()
  n.see(resumedView)
  n.see(refused, true) // Go refused: the view's error
  assert.equal(n.state.phase, 'shown')
  n.see(walked, false) // e.g. Hide minimap, or the page's State
  assert.equal(n.state.phase, 'shown')
  c.tick(shownFor + fadeFor)
  assert.equal(n.state.phase, 'hidden') // its own time still ends it
})

test('shown once for each resume, not again on the views after it, and again after the next resume', () => {
  const { c, n, changes } = setup()
  n.see(resumedView)
  n.see(walked, true)
  for (let i = 0; i < 20; i++) { n.see(walked, true); n.see(walked, false); n.see(refused, true) }
  c.tick(60000)
  assert.deepEqual(changes.map((s) => s.phase), ['shown', 'hidden'])
  // Timed out, then more of the same expedition: still not again.
  const b = setup()
  b.n.see(resumedView); b.c.tick(shownFor + fadeFor)
  for (let i = 0; i < 5; i++) b.n.see(walked, true)
  assert.deepEqual(b.changes.map((s) => s.phase), ['shown', 'fading', 'hidden'])
  // The dungeon left (Go's notice gone), then another game's expedition resumed.
  n.see(town, true)
  n.see(resumedView)
  assert.equal(n.state.phase, 'shown')
  // Another notice of resuming (an imported game's): shown afresh.
  n.see(maze({ notice: 'Game 4 was imported into slot 4. ' + text }))
  assert.equal(n.state.text, 'Game 4 was imported into slot 4. ' + text)
  assert.equal(c.pending, 1)
})

test('leaving the dungeon screen (combat, the Castle, another game) ends it', () => {
  for (const away of [{ ...walked, screen: 'combat' }, town, { screen: 'games', notice: '', title: '' }]) {
    const { c, n } = setup()
    n.see(resumedView)
    n.see(away, false)
    assert.equal(n.state.phase, 'hidden', away.screen)
    assert.equal(c.pending, 0, away.screen)
  }
})

test('no stale timer: one from an earlier resume never ends a later one, and none outlives the page', () => {
  const { c, n } = setup()
  n.see(resumedView)
  c.tick(3000)
  n.see(town)
  n.see(resumedView) // a second resume, 3s into the first
  c.tick(1500) // the first's deadline passes
  assert.deepEqual(n.state, { text, phase: 'shown' })
  c.tick(2500)
  assert.equal(n.state.phase, 'fading')
  // A timer that fires after it was replaced does nothing (as if clearTimeout came too late).
  const late = []
  const stubborn = { setTimeout: (fn) => { late.push(fn); return late.length }, clearTimeout: () => {} }
  const m = resumeNotice({ timers: stubborn, reducedMotion: () => false })
  m.see(resumedView); m.see(town); m.see(resumedView)
  late[0]() // the first showing's
  assert.equal(m.state.phase, 'shown')
  // The page closing clears what is pending, and nothing is called after.
  const d = setup()
  d.n.see(resumedView)
  d.n.destroy()
  assert.equal(d.c.pending, 0)
  d.c.tick(60000)
  assert.deepEqual(d.changes.map((s) => s.phase), ['shown'])
})

test('nothing of the game: the notice calls no Session method, and the page passes it the views alone', () => {
  const src = read('./resumeNotice.js')
  assert.ok(!/\bimport\b/.test(src.replace(/\/\/.*$/gm, '')), 'imports nothing')
  assert.ok(!/Session|window\.go/.test(src.replace(/\/\/.*$/gm, '')))
  const app = read('../App.svelte')
  // Every view the page is given, once, after the call: acted for the dungeon's own actions only.
  assert.match(app, /const dungeonActs = new Set\(\[S\.DungeonAction, S\.CampAction, S\.SetMazeLook\]\)/)
  assert.match(app, /finally \{\s*busy = false\s*\}\s*resumeShown\.see\(view, dungeonActs\.has\(fn\)\)\s*\}/)
  assert.match(app, /\$effect\(\(\) => \(\) => resumeShown\.destroy\(\)\)/)
  // Not in the banners any more; every other notice is, as before.
  assert.match(app, /\{#if view\.notice && !view\.shop\?\.last && !view\.inn\?\.last && !view\.gold && !view\.temple\?\.last && !resumeNoticeOf\(view\)\}<div class="banner notice">/)
  assert.match(app, /<DungeonScreen \{view\} \{busy\} \{call\} \{resumed\} \/>/)
})

test('over the view, taking no room and no input, between the corners', () => {
  const screen = read('./DungeonScreen.svelte')
  const rule = screen.match(/\n\s*\.resumed \{([^}]*)\}/)[1]
  for (const want of ['position: absolute', 'pointer-events: none', 'top: 8px', 'left: calc(var(--corner, 8px) + 8px)', 'right: calc(var(--corner, 8px) + 8px)', 'transition: opacity 250ms']) assert.ok(rule.includes(want), want)
  assert.match(screen, /\.resumed\.fading \{ opacity: 0; \}/)
  // In the view (the maze) and over Camp's screen, never in the grid's flow.
  assert.match(screen, /<div class="camp">CAMP<\/div>\s*\{\/if\}\s*\{@render resumeNote\(\)\}\s*<\/section>/)
  assert.match(screen, /<CampScreen [^>]*\/>\s*\{@render resumeNote\(\)\}/)
  assert.match(screen, /<div class="resumed" class:fading=\{resumed\.phase === 'fading'\} role="status"><strong class="title">\{resumedTitle\}<\/strong>\{resumed\.text\}<\/div>/)
  // Nothing in it to press or focus.
  const note = screen.match(/\{#snippet resumeNote\(\)\}([^]*?)\{\/snippet\}/)[1]
  assert.ok(!/<button|tabindex|on[a-z]+=/.test(note))
})
