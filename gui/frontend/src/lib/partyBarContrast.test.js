// ART-3B-3C: the party bar's current member (Boltac's customer, Camp's
// member) is drawn on --selected; its secondary text (slot number, class,
// row, level, HP and AC) reads at WCAG AA (4.5:1) on it in every
// appearance, and on the game's own colours. --muted alone did not (3.77:1
// in Dark, 4.29:1 in Slate).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const css = readFileSync(new URL('../style.css', import.meta.url), 'utf8')
const bar = readFileSync(new URL('./PartyBar.svelte', import.meta.url), 'utf8')

// The tokens a block of style.css sets, by its selector.
function tokens(selector) {
  const i = css.indexOf(selector + ' {')
  assert.ok(i >= 0, selector)
  const body = css.slice(i, css.indexOf('\n}', i))
  return Object.fromEntries([...body.matchAll(/--([\w-]+):\s*(#[0-9a-f]{6})\s*;/gi)].map((m) => [m[1], m[2]]))
}

const lin = (c) => { c /= 255; return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4 }
const lum = (hex) => { const n = parseInt(hex.slice(1), 16); return 0.2126 * lin(n >> 16) + 0.7152 * lin((n >> 8) & 255) + 0.0722 * lin(n & 255) }
const ratio = (a, b) => { const [x, y] = [lum(a), lum(b)].sort((p, q) => q - p); return (x + 0.05) / (y + 0.05) }

const looks = {
  'the game’s own': ':root',
  Dark: "[data-appearance='dark']",
  Slate: "[data-appearance='slate']",
  Light: "[data-appearance='light'], [data-appearance='parchment']",
}

test('the current member’s secondary text is at AA on --selected, in every appearance', () => {
  const seen = {}
  for (const [name, selector] of Object.entries(looks)) {
    const t = tokens(selector)
    assert.ok(t.selected && t['selected-muted'] && t.muted && t.text, `${name}: its tokens`)
    const r = ratio(t['selected-muted'], t.selected)
    seen[name] = r.toFixed(2)
    assert.ok(r >= 4.5, `${name}: ${r.toFixed(2)}:1 on --selected`)
    // Still secondary: no brighter (or darker, in Light) than the text.
    assert.ok(ratio(t.text, t.selected) > r, `${name}: below the name`)
  }
  assert.deepEqual(seen, { 'the game’s own': '5.82', Dark: '5.82', Slate: '5.58', Light: '7.72' })
})

test('the party bar draws the current member’s secondary text in it; the highlight is kept', () => {
  assert.match(bar, /\.slot\.current, button\.slot\.current \{ background: var\(--selected\); \}/)
  assert.match(bar, /\.slot\.current :is\(\.num, \.cls, \.row, \.line2\) \{ color: var\(--selected-muted\); \}/)
  // The other slots' secondary text is as it was.
  assert.match(bar, /\n  \.num \{ color: var\(--muted\);/)
  assert.match(bar, /\n  \.line2 \{ font-size: 11px; color: var\(--muted\); \}/)
})
