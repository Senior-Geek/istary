// The maze picture across phase transitions (M-ART-1A.3), over the views
// Go gave on one real expedition (testdata/maze-transitions.json, recorded
// by gui/maze_transitions_test.go, which also checks every way into and out
// of an encounter): into an encounter and out by escaping; into one guarding
// a chest, won, the chest opened; a restart; Camp and back. The window's maze
// look was Classic at Close throughout.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compile } from 'svelte/compiler'
import { render } from 'svelte/server'
import { perspective } from './mazeProjection.js'
import { styles, zooms, painted, mazeLookOf } from './mazeStyle.js'

const read = (f) => readFileSync(new URL(f, import.meta.url), 'utf8')
const steps = JSON.parse(read('./testdata/maze-transitions.json'))

// MazeView compiled for the server (as emblems.test.js loads Emblem).
const MazeView = await (async () => {
  const { js } = compile(read('./MazeView.svelte'), { generate: 'server', filename: 'MazeView.svelte' })
  const code = js.code
    .replace(/from ['"](svelte[^'"]*)['"]/g, (_, m) => `from ${JSON.stringify(import.meta.resolve(m))}`)
    .replace(/from ['"](\.\/[^'"]*)['"]/g, (_, m) => `from ${JSON.stringify(new URL(m, import.meta.url).href)}`)
  return (await import('data:text/javascript,' + encodeURIComponent(code))).default
})()

const dungeonOf = (st) => ({ phase: st.phase, corridor: st.corridor ?? [], dark: !!st.dark, spells: st.spells ?? [] })
const lookOf = (st) => mazeLookOf({ mazeStyle: st.mazeStyle, mazeZoom: st.mazeZoom })
const draw = (st) => {
  const { style, zoom } = lookOf(st)
  return render(MazeView, { props: { dungeon: dungeonOf(st), style, zoom } }).body.replace(/<!--[^]*?-->/g, '')
}
const surfaces = (html) => [...html.matchAll(/<polygon class="([^"]*)"[^>]*data-surface="([^"]*)"/g)].map((m) => `${m[2]}:${m[1].replace(/ ?svelte-\w+/g, '')}`)

test('the recording covers the transitions: escape, a chest, victory, a restart and Camp, back to the maze each time', () => {
  const at = (when) => steps.findIndex((s) => s.when === when)
  const kinds = steps.map((s) => (s.screen === 'combat' ? 'combat/' + s.combat : s.screen === 'dungeon' ? 'dungeon/' + s.phase : s.screen))
  for (const k of ['dungeon/camp', 'dungeon/maze', 'combat/opening', 'combat/orders', 'combat/playback', 'combat/chest', 'combat/victory']) assert.ok(kinds.includes(k), k)
  // After each encounter, and after Camp and a restart, the next view is the maze.
  for (const i of steps.keys()) {
    if (i && steps[i - 1].screen === 'combat' && steps[i].screen !== 'combat') assert.equal(kinds[i], 'dungeon/maze', steps[i].when)
  }
  assert.equal(kinds[at('restarted')], 'dungeon/maze')
  assert.equal(kinds[at('out of Camp')], 'dungeon/maze')
  // Only the maze and Camp are on the dungeon screen (Go checks this for every way out).
  assert.ok(kinds.every((k) => !k.startsWith('dungeon/') || k === 'dungeon/maze' || k === 'dungeon/camp'))
})

test('in the maze, the picture is the corridor of that moment, in the look kept through every transition', () => {
  const drawn = []
  for (const st of steps) {
    // The look is the window's: the same in every view, combat included.
    assert.deepEqual(lookOf(st), { style: 'classic', zoom: 'close' }, st.when)
    if (st.screen !== 'dungeon' || st.phase !== 'maze') continue
    const html = draw(st)
    const want = painted(perspective(dungeonOf(st)), styles.classic).map((p) => `${p.id}:${p.paint}`)
    assert.deepEqual(surfaces(html), want, st.when)
    assert.ok(want.length > 0, st.when)
    assert.match(html, new RegExp(`viewBox="${zooms.close}"`), st.when)
    assert.match(html, /class="maze-view classic/, st.when)
    // Nothing to act on: no handler, no focusable element.
    assert.ok(!/\son[a-z]+=|tabindex|<button|<input/.test(html), st.when)
    drawn.push([st, html])
  }
  // No stale picture: each is its own view's alone. Two moments with
  // different corridors (or facings) are drawn differently, and the same
  // corridor is drawn the same, whatever came between.
  for (const [a, ha] of drawn) {
    for (const [b, hb] of drawn) {
      const same = JSON.stringify(dungeonOf(a)) === JSON.stringify(dungeonOf(b))
      assert.equal(ha === hb, same, `${a.when} / ${b.when}`)
    }
  }
  assert.ok(drawn.length >= 8, `${drawn.length} maze views`)
  assert.ok(new Set(drawn.map(([, h]) => h)).size >= 3)
})

test('Enhanced and every zoom draw the same moments the same way: the look changes the paint and the frame only', () => {
  for (const st of steps.filter((s) => s.screen === 'dungeon' && s.phase === 'maze')) {
    const d = dungeonOf(st)
    const ids = (style) => painted(perspective(d), styles[style]).filter((p) => ['wall', 'door', 'void'].includes(p.kind)).map((p) => p.id)
    assert.deepEqual(ids('enhanced'), ids('classic'), st.when)
    for (const zoom of Object.keys(zooms)) {
      const html = render(MazeView, { props: { dungeon: d, style: 'enhanced', zoom } }).body
      assert.deepEqual(surfaces(html).map((s) => s.split(':')[0]), painted(perspective(d), styles.enhanced).map((p) => p.id), `${st.when} ${zoom}`)
      assert.match(html, new RegExp(`viewBox="${zooms[zoom]}"`))
    }
  }
})

test('which screen shows what: combat its own screen, Camp its own, the maze its picture', () => {
  const app = read('../App.svelte')
  // The combat screen whenever Go says combat; the dungeon screen otherwise in an expedition.
  const combat = app.indexOf("{:else if view.screen === 'combat' && view.combat}"), dungeon = app.indexOf("{:else if view.screen === 'dungeon' && view.dungeon}")
  assert.ok(combat > 0 && dungeon > combat)
  const screen = read('./DungeonScreen.svelte')
  // Camp: its own screen. The maze: the picture, on every maze view.
  assert.match(screen, /\{#if inCamp\}\s*<CampScreen /)
  assert.match(screen, /const inCamp = \$derived\(d\.phase === 'camp' && !!view\.camp\)/)
  assert.match(screen, /\{#if d\.phase === 'maze'\}\s*<div class="picture"><MazeView dungeon=\{d\} style=\{look\.style\} zoom=\{look\.zoom\} \/><\/div>/)
  // The picture and its look come from the view alone: nothing kept from an earlier one.
  assert.match(screen, /const d = \$derived\(view\.dungeon\)/)
  assert.match(screen, /const look = \$derived\(mazeLookOf\(view\.settings\)\)/)
  const view = read('./MazeView.svelte')
  assert.ok(!/\$state|\$effect|onMount|\bstore\b/.test(view.replace(/\/\/.*$/gm, '')))
})
