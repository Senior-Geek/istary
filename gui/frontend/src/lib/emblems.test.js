// ART-3B-0: the places' emblems, the approved SVGs (docs/design/art3/
// emblems) drawn inline in the appearance's gold beside each place's title
// and on its themed panel; and the frame's two layout preparations: the
// information keeps the room a place without a picture would give it, and
// a shown character sheet has priority over the story box.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { compile } from 'svelte/compiler'
import { render } from 'svelte/server'
import { emblems, emblemFor } from './emblems.js'
import { places } from './navigation.js'

const read = (name) => readFileSync(new URL(name, import.meta.url), 'utf8')
const approved = (file) => readFileSync(new URL(`../../../../docs/design/art3/emblems/emblem-${file}.svg`, import.meta.url), 'utf8')
const attrs = (s) => Object.fromEntries([...s.matchAll(/([\w-]+)="([^"]*)"/g)].map(([, k, v]) => [k, v]))

// Emblem compiled for the server, its imports made absolute so it loads
// from a data: URL (as content1a.test.js does for StoryText).
const Emblem = await (async () => {
  const { js } = compile(read('./Emblem.svelte'), { generate: 'server', filename: 'Emblem.svelte' })
  const code = js.code
    .replace(/from ['"](svelte[^'"]*)['"]/g, (_, m) => `from ${JSON.stringify(import.meta.resolve(m))}`)
    .replace(/from ['"](\.\/[^'"]*)['"]/g, (_, m) => `from ${JSON.stringify(new URL(m, import.meta.url).href)}`)
  return (await import('data:text/javascript,' + encodeURIComponent(code))).default
})()
const html = (props) => render(Emblem, { props }).body.replace(/<!--[^]*?-->/g, '')

test('each place has its approved emblem, drawn exactly as its SVG file', () => {
  const want = ['castle', 'edge', 'tavern', 'inn', 'trading', 'temple', 'training', 'maze']
  assert.deepEqual(Object.keys(emblems), want)
  for (const place of Object.keys(places)) assert.equal(!!emblemFor(place), want.includes(place), place)
  assert.equal(emblemFor('nowhere'), null)
  const root = attrs(read('./Emblem.svelte').match(/<svg [^>]*>/)[0])
  for (const [place, { file, shapes }] of Object.entries(emblems)) {
    const svg = approved(file)
    // The file's defaults are the component's: viewBox, colour, strokes.
    const own = attrs(svg.match(/<svg [^>]*>/)[0])
    for (const k of ['viewBox', 'fill', 'stroke', 'stroke-width', 'stroke-linecap', 'stroke-linejoin']) assert.equal(root[k], own[k], `${place}: ${k}`)
    assert.equal(own.color, '#d2b06a', `${file}: the app's gold`)
    // Its shapes, in order, attribute for attribute.
    const drawn = [...svg.matchAll(/<(path|circle)\s+([^>]*?)\/>/g)].map(([, tag, a]) => ({ tag, ...attrs(a) }))
    assert.deepEqual(shapes, drawn, `${place}: the drawing of emblem-${file}.svg`)
    assert.ok(!/<text|<image|href|<script|url\(/.test(svg), `${file}: no text, images, links or scripts`)
  }
  // As amended: the swords are the Training Grounds', Boltac's a balance;
  // the archery target is kept beside them as a reference, unused.
  assert.equal(emblems.training.file, 'training-crossed-swords')
  assert.equal(emblems.trading.file, 'trading-balance-scale')
  assert.match(approved('training-target'), /<title>Not used/)
  assert.ok(!Object.values(emblems).some((e) => e.file === 'training-target'))
})

test('an emblem is decoration, in the colour around it, and nothing for a place without one', () => {
  for (const place of Object.keys(emblems)) {
    const out = html({ location: place, size: '34px' })
    const root = attrs(out.match(/^<svg [^>]*>/)[0])
    assert.ok(root.class.split(' ').includes('emblem') && root['data-emblem'] === place, place)
    assert.equal(root.viewBox, '0 0 64 64', place)
    assert.equal(root.style, 'width: 34px; height: 34px;', `${place}: its size`)
    assert.equal(root.stroke, 'currentColor', `${place}: the colour around it`)
    assert.match(out, /aria-hidden="true" focusable="false"/, `${place}: hidden from screen readers`)
    assert.equal(out.match(/<(path|circle) /g).length, emblems[place].shapes.length, `${place}: every shape`)
    assert.equal(out.replace(/<[^>]*>/g, '').trim(), '', `${place}: no text`)
    // Gold only where the file says so (currentColor); the maze's opening
    // is its one dark fill.
    const colours = [...out.matchAll(/(?:fill|stroke)="(#[0-9a-f]+)"/g)].map((m) => m[1])
    assert.deepEqual(colours, place === 'maze' ? ['#120f0a'] : [], place)
  }
  assert.equal(html({ location: 'settings' }).trim(), '')
  assert.equal(html({ location: undefined }).trim(), '')
})

test('the emblem stands beside the title, in the appearance’s gold, and on the themed panel', () => {
  const frame = read('./LocationFrame.svelte')
  assert.match(frame, /<h2 class="title"><Emblem location=\{view\.location\} size="34px" \/>\{here\}<\/h2>/, 'before the title, inside its heading: the title alone is spoken')
  assert.match(frame, /\.title \{[^}]*display: flex; align-items: center; gap: 10px;/, 'centred on the title')
  assert.match(frame, /\.title :global\(\.emblem\) \{ color: var\(--gold\);/)
  // Every appearance has a gold the emblem takes: the Light panels' darker.
  const css = readFileSync(new URL('../style.css', import.meta.url), 'utf8')
  const gold = (sel) => css.match(new RegExp(`${sel.replace(/[[\]]/g, '\\$&')}[^{]*\\{[^}]*--gold: (#[0-9a-f]+)`))?.[1]
  assert.equal(gold(":root"), '#d2b06a')
  assert.equal(gold("[data-appearance='light']"), '#7a5a14', 'darker on the light panels')
  for (const id of ['dark', 'slate']) assert.ok(gold(`[data-appearance='${id}']`), id)
  // The maze, as in town; not at Camp.
  const dungeon = read('./DungeonScreen.svelte')
  assert.match(dungeon, /<h2>\{#if d\.phase !== 'camp'\}<Emblem location="maze" size="34px" \/>\{\/if\}/)
  assert.match(dungeon, /\.bar h2 :global\(\.emblem\) \{ color: var\(--gold\);/)
  // The themed panel: the emblem, in the place's tint, or a symbol where
  // there is none.
  const scene = read('./LocationScene.svelte')
  assert.match(scene, /\{#if emblemFor\(location\)\}<Emblem \{location\} size="clamp\(32px, 9vmin, 88px\)" \/>\{:else\}<span class="emblem" aria-hidden="true">\{scene\.emblem\}<\/span>\{\/if\}/)
})

test('an illustrated place gives its information the room it would have without the picture', () => {
  const frame = read('./LocationFrame.svelte')
  assert.match(frame, /\.frame\.illustrated \.body \{ grid-template-rows: minmax\(var\(--top\), 1fr\) fit-content\(calc\(100% - var\(--top\) - 12px\)\); \}/)
  assert.ok(!/fit-content\(45%\)/.test(frame), 'no cap of 45%')
  // The display and actions keep --top, as at a place without a picture,
  // whose information takes the rest; the gap is the body's.
  assert.match(frame, /grid-template-rows: var\(--top\) minmax\(0, 1fr\);/)
  assert.match(frame, /\.body \{[^}]*gap: 12px 16px;/)
})

test('a shown character sheet has priority over the story box', () => {
  const frame = read('./LocationFrame.svelte')
  assert.match(frame, /const storyOpen = \$derived\(storyHere && panels\.story\.open && !sheetShown\)/, 'not drawn over the sheet')
  assert.match(frame, /\{#if storyOpen && display\}\s*<StoryBox /)
  assert.match(frame, /function openStory\(\) \{\s*if \(sheetShown\) sheet\.hide\(\)\s*panels\.story\.open = true\s*\}/, 'Story shows the story, hiding the sheet')
  assert.match(frame, /onclick=\{\(\) => \(storyOpen \? closeStory\(\) : openStory\(\)\)\}/)
  // Showing the sheet leaves the story box open, for when it is hidden.
  assert.ok(!/sheet\.show[^\n]*story\.open = false|story\.open = false[^\n]*sheet\.show/.test(frame))
})
