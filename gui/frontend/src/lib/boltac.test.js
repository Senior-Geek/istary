// ART-3B-3B: Boltac's Trading Post is illustrated (its picture in
// scenes.js, checked with the others in frameViews.test.js; its story in
// content1a.test.js). Its picture is a little right of centre, and over it
// the Selected Character panel has the frame's glass, as its others do.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { scenes, illustrated } from './scenes.js'

test('Boltac’s picture, a little right of centre, described for screen readers', () => {
  assert.ok(illustrated('trading'))
  assert.match(new URL(scenes.trading.src).pathname, /\/src\/assets\/scenes\/trading\.webp$/)
  assert.equal(scenes.trading.focus, '75% 50%')
  assert.match(scenes.trading.alt, /^Boltac's Trading Post: Boltac, .*counter.*sign.*black cat with emerald eyes and a diamond on its collar/)
  // The other pictures are as they were.
  assert.equal(scenes.castle.focus, '45% 50%')
  assert.equal(scenes.edge.focus, '50% 50%')
  assert.equal(scenes.tavern.focus, '50% 50%')
  assert.equal(scenes.inn.focus, '100% 50%')
  // (ART-3C) the Training Grounds': castleArt.test.js.
  assert.equal(scenes.temple.src, null, 'temple: no picture yet')
})

test('over a picture, the Selected Character panel is glass, as the Actions panel is; nothing moves', () => {
  const panel = readFileSync(new URL('./SelectedCharacter.svelte', import.meta.url), 'utf8')
  const frame = readFileSync(new URL('./LocationFrame.svelte', import.meta.url), 'utf8')
  const glass = panel.match(/:global\(\.frame\.illustrated\) \.selected \{([^}]*)\}/)
  assert.ok(glass, 'only at an illustrated place')
  for (const rule of ['background: var(--glass);', 'border: 1px solid var(--glass-line);', 'border-radius: 6px;', 'backdrop-filter: blur(6px);', 'box-shadow: 0 2px 12px var(--glass-shadow);']) {
    assert.ok(glass[1].includes(rule), rule)
    assert.ok(frame.includes(rule), `the frame's panels have ${rule}`)
  }
  assert.ok(!/(margin|padding|width|height|position|grid|display)\s*:/.test(glass[1]), 'no size or place of its own')
  assert.match(panel, /\n  \.selected \{ margin: 0; padding: 10px 12px;/, 'its padding as it was')
})

test('a shop button brought into view (Tab, Shift-Tab) stops below the sticky column heads, as its line does', () => {
  const src = readFileSync(new URL('./TradingScreen.svelte', import.meta.url), 'utf8')
  assert.match(src, /\.tablewrap thead th \{ position: sticky; top: 0;/, 'the heads stay')
  assert.match(src, /table\.lines tbody tr, table\.lines td\.act button \{ scroll-margin-top: 30px; \}/)
})
