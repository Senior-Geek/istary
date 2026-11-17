// ART-3E: over a picture, the Actions collapse to their heading and come
// back (LocationFrame), one choice for every illustrated place for the
// session (App's PANELS), not while a character is being created. The live
// scenes suite drives it; here, what it is made of.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const read = (f) => readFileSync(new URL(f, import.meta.url), 'utf8')
const frame = read('./LocationFrame.svelte')
const app = read('../App.svelte')

test('the choice is the session’s: App keeps it in PANELS, expanded at the start, and saves nothing', () => {
  assert.match(app, /let actionsCollapsed = \$state\(false\)/)
  assert.match(app, /actions: \{\s*get collapsed\(\) \{ return actionsCollapsed \},\s*set collapsed\(v\) \{ actionsCollapsed = v \},\s*\}/)
  assert.equal(app.match(/actionsCollapsed/g).length, 3, 'set only through PANELS')
  assert.ok(!/actionsCollapsed[\s\S]{0,80}S\.\w+\(/.test(app), 'no Session call')
})

test('one control in the Actions’ heading, at every illustrated place with Actions, except while creating', () => {
  assert.match(frame, /const collapsible = \$derived\(art && !!actions && !waysInActions && !!panels\?\.actions && !compact\)/)
  assert.match(frame, /const collapsed = \$derived\(collapsible && panels\.actions\.collapsed\)/)
  assert.equal(frame.match(/collapseActions\((true|false)\)/g).length, 2, 'one way to collapse, one to expand')
  // Named, with its state and what it controls; focus follows the toggle.
  assert.match(frame, /aria-expanded="true" aria-controls="actions-body"\s+aria-label="Collapse Actions" title="Collapse Actions, to see the picture"[^\n]*\}>Collapse<\/button>/)
  assert.match(frame, /aria-expanded="false" aria-controls="actions-body"\s+title="Show the Actions"[^\n]*\}>Show Actions<\/button>/)
  assert.match(frame, /const to = collapsed \? expandButton : collapseButton/)
  // The body is hidden, not removed: what a place's Actions hold is kept.
  assert.match(frame, /<div id="actions-body" class="actions-body" hidden=\{collapsed\}>\{@render actions\(ways, destination\)\}<\/div>/)
  // Escape is App's, as before: the frame adds no key of its own.
  assert.ok(!/onkeydown|Escape/.test(frame.slice(frame.indexOf('<div class="nav-area"'), frame.indexOf('data-area="information"'))))
})

test('collapsed, the heading alone at the column’s top right; the rest lets clicks through; nothing else moves', () => {
  assert.match(frame, /\.nav-area\.collapsed \{ pointer-events: none; overflow: hidden; \}/)
  assert.match(frame, /\.nav-area\.collapsed > :not\(\.actions\) \{ display: none; \}/)
  assert.match(frame, /\.nav-area\.collapsed \.actions \{ pointer-events: auto; justify-self: end; width: max-content;/)
  // No size, place or crop of the grid, the display or the picture changes.
  const css = frame.slice(frame.indexOf('<style>'))
  assert.ok(!/collapsed[^{]*\{[^}]*(grid-template|object-position|height:|--top)/.test(css))
})

test('Settings says what its appearance is for, as it is now', () => {
  const settings = read('./SettingsScreen.svelte')
  assert.match(settings, /Appearance options control the illustrations shown in supported Castle locations and\s+the Edge of Town\./)
  assert.ok(!/the Castle and the Edge of Town\. Kept/.test(settings))
})
