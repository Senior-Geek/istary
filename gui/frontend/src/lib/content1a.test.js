// CONTENT-1A: the stories are Markdown files (gui/content), parsed by Go
// (story.go) and carried by the View; StoryText draws them with ordinary
// elements, and nothing in a story is ever HTML. Checked by rendering
// StoryText with Svelte's own server renderer, on the stories Go recorded
// from the real files (testdata/frame-views.json) and on made-up ones.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync, readdirSync } from 'node:fs'
import { compile } from 'svelte/compiler'
import { render } from 'svelte/server'

const read = (name) => readFileSync(new URL(name, import.meta.url), 'utf8')
const views = JSON.parse(read('./testdata/frame-views.json'))
const at = (c) => views.find((v) => v.case === c)

// StoryText compiled for the server, its imports of Svelte made absolute
// so it loads from a data: URL.
const StoryText = await (async () => {
  const { js } = compile(read('./StoryText.svelte'), { generate: 'server', filename: 'StoryText.svelte' })
  const code = js.code.replace(/from ['"](svelte[^'"]*)['"]/g, (_, m) => `from ${JSON.stringify(import.meta.resolve(m))}`)
  return (await import('data:text/javascript,' + encodeURIComponent(code))).default
})()
// html is a story drawn: its elements, without Svelte's markers and
// scoped-style classes.
const html = (story) => render(StoryText, { props: { story } }).body
  .replace(/<!--[^]*?-->/g, '').replace(/\s*svelte-\w+/g, '').replace(/ class=""/g, '')
const plain = (runs) => runs.map((r) => r.text).join('')
const decode = (s) => s.replace(/<[^>]*>/g, '').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"').replace(/&#39;/g, "'").replace(/&amp;/g, '&')

test('every kind of block and emphasis is drawn with its element', () => {
  const r = (text, more = {}) => ({ text, ...more })
  const story = { blocks: [
    { kind: 'heading', level: 2, text: [r('A heading')] },
    { kind: 'heading', level: 3, text: [r('A smaller one')] },
    { kind: 'paragraph', text: [r('Plain, '), r('bold', { bold: true }), r(', '), r('italic', { italic: true }), r(' and '), r('both', { bold: true, italic: true })] },
    { kind: 'list', items: [[r('one')], [r('two')]] },
    { kind: 'list', ordered: true, items: [[r('first')]] },
    { kind: 'quote', items: [[r('Quoted.')], [r('Again.')]] },
    { kind: 'challenge', items: [[r('Dare.', { bold: true })]] },
  ] }
  assert.equal(html(story), [
    '<h4>A heading</h4>', '<h5>A smaller one</h5>',
    '<p>Plain, <b>bold</b>, <i>italic</i> and <b><i>both</i></b></p>',
    '<ul><li>one</li><li>two</li></ul>', '<ol><li>first</li></ol>',
    '<blockquote><p>Quoted.</p><p>Again.</p></blockquote>',
    '<p class="challenge"><b>Dare.</b></p>',
  ].join(''))
})

test('a story’s HTML and script are text, never elements', () => {
  const evil = '<script>alert(1)</script><img src=x onerror="alert(1)"> & <b>'
  const out = html({ blocks: [{ kind: 'paragraph', text: [{ text: evil }] }, { kind: 'list', items: [[{ text: evil, bold: true }]] }] })
  assert.ok(!/<script|<img/i.test(out), out)
  assert.equal(out.match(/<b>/g).length, 1, 'only the bold run’s own <b>')
  assert.equal(decode(out), evil + evil, 'shown as the text it is')
  // Nothing in the frontend injects HTML.
  const sources = (dir) => readdirSync(new URL(dir, import.meta.url)).filter((f) => f.endsWith('.svelte')).map((f) => read(dir + f))
  for (const src of [...sources('./'), read('../App.svelte')]) assert.ok(!/\{@html/.test(src), 'no {@html}')
})

test('the six stories, from their files, as Go chose them for each place', () => {
  const intro = at('no game, castle').story, castle = at('castle').story, edge = at('edge').story, tavern = at('tavern').story, inn = at('inn').story, boltac = at('trading').story
  assert.ok(intro && castle && edge && tavern && inn && boltac, 'the Castle with and without a game, the Edge, (ART-3B-1) the Tavern, (ART-3B-2B) the Inn and (ART-3B-3B) Boltac’s have stories')
  assert.notDeepEqual(intro, castle, 'with no game, the introduction; in a game, the Castle’s own')
  assert.deepEqual(at('no game, edge').story, edge, 'the Edge’s, with or without a game')
  for (const c of ['training', 'settings', 'no game, settings', 'real expedition']) assert.equal(at(c).story, undefined, `${c}: no story`)
  // Drawn whole: every block, and its text as Go read it.
  for (const st of [intro, castle, edge, tavern, inn, boltac]) {
    const out = html(st)
    const text = st.blocks.map((b) => (b.text ? plain(b.text) : b.items.map(plain).join(''))).join('')
    assert.equal(decode(out), text)
    assert.equal(out.match(/<b>/g)?.length ?? 0, st.blocks.flatMap((b) => [b.text ?? [], ...(b.items ?? [])].flat()).filter((r) => r.bold).length, 'its bold runs')
  }
  // The introduction: titled, its ten paragraphs, then the challenge.
  assert.ok(intro.title, 'a title, from its first heading')
  assert.equal(castle.title, undefined, 'the Castle’s own story is titled with the place')
  assert.deepEqual(intro.blocks.map((b) => b.kind), [...Array(10).fill('paragraph'), 'challenge'])
  assert.match(html(intro), /<\/p><p class="challenge">[^<]+<\/p>$/, 'the challenge last, as its own paragraph')
  // The Tavern's own (its approved text is checked in Go, story_test.go):
  // titled, eight paragraphs, its challenge; not the Castle's.
  assert.equal(tavern.title, "Gilgamesh's Tavern")
  assert.deepEqual(tavern.blocks.map((b) => b.kind), [...Array(8).fill('paragraph'), 'challenge'])
  assert.match(html(tavern), /<\/p><p class="challenge">Your adventure begins with good company\. Gather your party\.<\/p>$/)
  for (const other of [intro, castle, edge]) assert.notDeepEqual(tavern, other)
  // The Inn's (ART-3B-2B; its approved text is checked in Go): titled,
  // eleven paragraphs, Mistress Elspeth in bold, its challenge.
  assert.equal(inn.title, "The Adventurer's Inn")
  assert.deepEqual(inn.blocks.map((b) => b.kind), [...Array(11).fill('paragraph'), 'challenge'])
  assert.match(html(inn), /I'm <b>Mistress Elspeth<\/b>\./)
  assert.match(html(inn), /<\/p><p class="challenge">Rest your adventurers, restore their spells, and heal their wounds\. Choose your accommodations carefully; comfort has its price\.<\/p>$/)
  for (const other of [intro, castle, edge, tavern]) assert.notDeepEqual(inn, other)
  // Boltac's (ART-3B-3B; its approved text is checked in Go): titled,
  // twelve paragraphs, Boltac in bold, its challenge.
  assert.equal(boltac.title, "Boltac's Trading Post")
  assert.deepEqual(boltac.blocks.map((b) => b.kind), [...Array(12).fill('paragraph'), 'challenge'])
  assert.match(html(boltac), /cries <b>Boltac<\/b>, spreading his arms\./)
  assert.match(html(boltac), /<\/p><p class="challenge">Equip your adventurers, inspect unfamiliar treasures, and spend your gold wisely\. Boltac never forgets a customer — particularly one who still owes him money\.<\/p>$/)
  for (const other of [intro, castle, edge, tavern, inn]) assert.notDeepEqual(boltac, other)
})

test('the challenge is the story box’s gold, bold; an ordinary quote is not', () => {
  const src = read('./StoryText.svelte')
  assert.match(src, /\.challenge \{ color: var\(--gold\); font-weight: 700; \}/)
  assert.match(read('./StoryBox.svelte'), /h3 \{[^}]*color: var\(--gold\)/, 'the colour of the box’s title')
  assert.ok(!/blockquote \{[^}]*(gold|font-weight)/.test(src), 'a quote is not a challenge')
})

test('the frame draws the View’s story; the places hold no story text', () => {
  const frame = read('./LocationFrame.svelte')
  assert.match(frame, /const storyHere = \$derived\(art && !!view\.story && !!panels\)/, 'a story box only where there is a story')
  assert.match(frame, /<StoryBox title=\{view\.story\.title \|\| here\}[\s\S]*?<StoryText story=\{view\.story\} \/><\/StoryBox>/)
  for (const f of ['./CastleScreen.svelte', './EdgeScreen.svelte', './TavernScreen.svelte', './InnScreen.svelte', './TradingScreen.svelte', './SelectedCharacter.svelte']) {
    const src = read(f)
    assert.ok(!/snippet story|storyTitle|class="challenge"/.test(src), `${f}: no story of its own`)
  }
  // The files: the introduction's title is its first heading, and its
  // challenge is marked as one.
  const md = readFileSync(new URL('../../../content/castle/introduction.md', import.meta.url), 'utf8')
  assert.equal(md.split('\n')[0], `# ${at('no game, castle').story.title}`)
  assert.match(md, /\n> \[!challenge\]\n> [^\n]+\n$/)
})
