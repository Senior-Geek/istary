// The Castle's shared layout (UX-3B), checked in the components' source,
// as UX-2's layout checks are (selectedViews.test.js): every place in town
// is the one location frame, with its four areas; the character sheet is
// embedded in the frame and nowhere else; creation's decisions are all in
// the Actions.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync, existsSync } from 'node:fs'

const read = (name) => readFileSync(new URL(name, import.meta.url), 'utf8')
const places = ['CastleScreen', 'EdgeScreen', 'TrainingScreen', 'TavernScreen', 'TradingScreen', 'InnScreen', 'TempleScreen', 'SettingsScreen']
// The place's own actions snippet, up to the next snippet or the content.
const actionsOf = (src) => {
  const from = src.indexOf('{#snippet actions()}')
  assert.ok(from >= 0, 'an actions snippet')
  let depth = 0
  const re = /\{#snippet |\{\/snippet\}/g
  re.lastIndex = from
  for (let m; (m = re.exec(src));) {
    depth += m[0] === '{/snippet}' ? -1 : 1
    if (depth === 0) return src.slice(from, m.index)
  }
  assert.fail('unclosed actions snippet')
}

test('every place in town is the one location frame', () => {
  for (const p of places) {
    const src = read(`./${p}.svelte`)
    assert.match(src, /import LocationFrame from '\.\/LocationFrame\.svelte'/, p)
    assert.match(src, /<LocationFrame /, p)
    assert.ok(!/class="frame"|grid-template-areas/.test(src), `${p} has no layout of its own`)
  }
})

test('the frame has its four areas: display and actions above, information, then the party', () => {
  const f = read('./LocationFrame.svelte')
  const order = ['data-area="display"', 'data-area="actions"', 'data-area="information"', 'data-area="party"'].map((a) => f.indexOf(a))
  assert.ok(order.every((i) => i > 0), 'all four areas')
  assert.deepEqual([...order].sort((a, b) => a - b), order, 'in order')
  assert.match(f, /grid-template-areas: "scene nav" "content content";/, 'display beside actions, information below')
  assert.match(f, /grid-template-rows: var\(--top\) minmax\(0, 1fr\);/, 'a fixed top row; the information takes the rest')
  assert.match(f, /\.nav-area \{[^}]*overflow-y: auto;/, 'long actions scroll inside')
  assert.match(f, /\.content-area \{[^}]*overflow-y: auto;/, 'the information scrolls on its own')
  assert.match(f, /\n  \.body \{[^}]*\boverflow: hidden;/, 'the display and actions stay in place')
  assert.match(f, /\{#if partyShown\}\s*<div class="party-area"/, 'the party bar can be hidden')
  assert.match(f, /\{partyShown \? 'Hide party' : 'Show party'\}/, 'with its toggle')
  // The Actions come before the destinations: the next decision first.
  assert.ok(f.indexOf('aria-label="Actions here"') < f.indexOf('aria-label="Destinations"'))
})

test('the character sheet is embedded in the frame, with one toggle, and no window', () => {
  assert.ok(!existsSync(new URL('./FloatingPanel.svelte', import.meta.url)), 'no floating panel')
  const f = read('./LocationFrame.svelte')
  assert.match(f, /import CharacterSheet from '\.\/CharacterSheet\.svelte'/)
  assert.match(f, /\{#if sheetShown\}[\s\S]*<CharacterSheet [\s\S]*\{:else if scene\}/, 'the sheet takes the artwork’s place')
  assert.match(f, /\{toggleLabel\(sheetShown\)\}/)
  for (const p of [...places, '../App']) {
    const src = read(`./${p}.svelte`)
    assert.ok(!/CharacterSheet|FloatingPanel/.test(src), `${p} does not show a sheet itself`)
    assert.ok(!/>Character sheet<|>Character Sheet</.test(src), `${p} has no sheet button of its own`)
  }
  const sheet = read('./CharacterSheet.svelte')
  assert.match(sheet, /\.pages \{ overflow-y: auto;/, 'the sheet scrolls inside, its header kept')
  for (const s of ['overview', 'equipment', 'spells']) assert.match(sheet, new RegExp(`\\{@render ${s}\\(\\)\\}`), `the ${s} section`)
  assert.match(sheet, /<h4>Worn &amp; Wielded<\/h4>/)
  assert.match(sheet, /<h4>Pack<\/h4>/)
  assert.ok(!/<h3>Inventory|<h3>Equipment<\/h3>\s*\{#if equipped/.test(sheet), 'no second list of what is equipped')
})

test('at the Training Grounds every creation decision is in the Actions', () => {
  const src = read('./TrainingScreen.svelte')
  const actions = actionsOf(src)
  const rest = src.replace(actions, '')
  for (const m of ['AnswerCreate', 'SetName', 'SetPassword', 'ChooseRace', 'ChooseAlignment', 'S.Back', 'S.Confirm', 'DiscardCreation', 'StartCreation']) {
    assert.ok(actions.includes(m), `${m} in the Actions`)
    assert.ok(!rest.includes(`call(${m.startsWith('S.') ? m : 'S.' + m}`), `${m} not below`)
  }
  for (const label of ['>Yes<', '>No<', '>Keep and save<', '>Discard<', '>Continue<']) {
    assert.ok(actions.includes(label), `${label} in the Actions`)
    assert.ok(!rest.includes(label), `${label} not below`)
  }
  assert.match(actions, /\{next\}/, 'the next decision is said')
  assert.ok(!/FinishStats|Done — choose class/.test(src), 'no separate class-selection step')
  assert.match(rest, /PickClass/, 'the class is chosen from the table')
})

test('ART-2A: an illustrated place lays the same areas over its picture, and its panels can be hidden', () => {
  const f = read('./LocationFrame.svelte')
  assert.match(f, /\{#if art\}<LocationScene [^>]*mode="backdrop" \/>\{\/if\}/, 'the picture behind the frame')
  assert.match(f, /mode=\{art \? 'view' : 'panel'\}/, 'the display opens onto it')
  // Hidden: the picture, and one way back.
  const bare = f.slice(f.indexOf('{#if bare}'), f.indexOf('{:else}', f.indexOf('{#if bare}')))
  assert.equal(bare.match(/<button/g)?.length, 1, 'one control while hidden')
  assert.match(bare, />Show panels <kbd>Esc<\/kbd><\/button>/)
  assert.ok(!/data-area="(display|actions|information|party)"/.test(bare), 'no working area while hidden')
  // Offered only where there is a picture; the frame's areas are the same.
  assert.match(f, /\{#if art\}\s*<button [^>]*onclick=\{hidePanels\}/)
  assert.match(f, /grid-template-areas: "scene nav" "content content";/)
  assert.ok(!/SetSettings|S\.Set\w*Panel/.test(f), 'nothing about the panels is sent to Go')
  // Escape shows hidden panels before anything else.
  const app = read('../App.svelte')
  const key = app.slice(app.indexOf('function onkey'))
  assert.ok(key.indexOf('if (bare)') > 0 && key.indexOf('if (bare)') < key.indexOf('if (sheetShown'), 'Escape: panels first, then the sheet')
  assert.ok(key.indexOf('if (bare)') < key.indexOf('ReturnToCastle'), 'and before Back')
  // The picture is a cover, never stretched.
  assert.match(read('./LocationScene.svelte'), /img \{[^}]*object-fit: cover;/)
})

test('ART-2B: the appearances change colours only, and the story box holds narrative only', () => {
  const css = readFileSync(new URL('../style.css', import.meta.url), 'utf8')
  const blocks = [...css.matchAll(/^(\[data-appearance[^{]*)\{([^}]*)\}/gm)]
  for (const id of ['dark', 'light', 'slate']) {
    const own = blocks.find(([, sel]) => sel.trim() === `[data-appearance='${id}']`)
    assert.ok(own, `${id} is defined`)
    const glass = own[2].match(/--glass: rgba\([^)]*, ([\d.]+)\)/)
    assert.ok(glass && +glass[1] >= 0.9 && +glass[1] <= 0.95, `${id}: panels 90-95% opaque`)
  }
  // Nothing but colours: no size, spacing, type or position.
  for (const [, sel, body] of blocks) {
    for (const decl of body.split(';').map((d) => d.trim()).filter(Boolean)) {
      const prop = decl.split(':')[0].trim()
      assert.ok(/^--|^(color|color-scheme|background|border-color)$/.test(prop), `${sel.trim()}: ${prop} is not a colour`)
    }
  }
  const frame = read('./LocationFrame.svelte')
  assert.match(frame, /data-appearance=\{look\}/)
  assert.match(frame, /\{#each appearances as a/)
  assert.ok(!/S\.\w*(Appearance|Story)/.test(frame), 'nothing about the appearance or the story box is sent to Go')
  // The Castle's description is its story; the Edge keeps saving and
  // practice in its information, outside the story.
  const castle = read('./CastleScreen.svelte'), edge = read('./EdgeScreen.svelte')
  assert.match(castle, /\{#snippet story\(\)\}[\s\S]*The party is formed at[\s\S]*\{\/snippet\}/)
  const story = edge.slice(edge.indexOf('{#snippet story()}'), edge.indexOf('{/snippet}', edge.indexOf('{#snippet story()}')))
  assert.match(story, /The party sets out from here/)
  assert.ok(!/saved|Practice|practice/.test(story), 'no saving or practice in the story')
  const rest = edge.replace(story, '')
  assert.match(rest, /saved after every/)
  assert.match(rest, /<b>Practice<\/b> is Istary's own/)
})
