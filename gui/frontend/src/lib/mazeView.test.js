// MazeView (M-ART-1A.1, 1A.2): the picture of the dungeon view in either
// style, made of the projection alone (mazeProjection.test.js and
// mazeEnhanced.test.js cover what it is drawn from).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

test('MazeView draws a style’s paint of the dungeon view alone, and takes no input', () => {
  const view = readFileSync(new URL('./MazeView.svelte', import.meta.url), 'utf8')
  const imports = [...view.matchAll(/import[^'"]*['"]([^'"]+)['"]/g)].map((m) => m[1]).sort()
  assert.deepEqual(imports, ['./mazeProjection.js', './mazeStyle.js'])
  assert.match(view, /let \{ dungeon, style = 'classic', zoom = 'standard' \} = \$props\(\)/)
  assert.match(view, /const drawn = \$derived\(painted\(perspective\(dungeon\), styles\[style\] \?\? styles\.classic\)\)/)
  // The zoom is the frame alone: it chooses the viewBox and nothing else.
  assert.match(view, /const frame = \$derived\(zooms\[zoom\] \?\? zooms\.standard\)/)
  const code = view.replace(/\/\/.*$/gm, '')
  assert.equal([...code.matchAll(/\bzoom\b/g)].length, 2) // declared, and used for the frame alone
  assert.match(view, /<svg viewBox=\{frame\}/)
  assert.ok(!/\$state|\$effect|\bstore\b/.test(view))
  assert.ok(!/wailsjs|Session|minimap|automap|position|on[a-z]+=|tabindex|Math\.random/i.test(view.replace(/\/\/.*$/gm, '')))
  assert.match(view, /role="img"/)
  // The dungeon screen gives it view.dungeon, and draws no picture of its own.
  const screen = readFileSync(new URL('./DungeonScreen.svelte', import.meta.url), 'utf8')
  assert.match(screen, /<MazeView dungeon=\{d\} style=\{look\.style\} zoom=\{look\.zoom\} \/>/)
  assert.ok(!/<polygon/.test(screen))
})

test('the style and zoom are window settings: read from Go’s view, set by SetMazeLook alone, and the keys go back to the maze', () => {
  const screen = readFileSync(new URL('./DungeonScreen.svelte', import.meta.url), 'utf8')
  // The picture is drawn in the look Go's settings say, each part its default if unknown.
  assert.match(screen, /const look = \$derived\(mazeLookOf\(view\.settings\)\)/)
  assert.match(screen, /<MazeView dungeon=\{d\} style=\{look\.style\} zoom=\{look\.zoom\} \/>/)
  const choose = screen.match(/function choose\(key, value, e\) \{\n([\s\S]*?)\n  \}\n/)[1]
  assert.match(choose, /call\(S\.SetMazeLook, next\.style, next\.zoom\)/)
  // A click lets go of the focus (the keys reach the maze); a key press on
  // the button (detail 0) keeps it there.
  assert.match(choose, /if \(e\.detail\) e\.currentTarget\.blur\(\)/)
  assert.equal([...choose.matchAll(/\bcall\(|\bS\./g)].length, 2) // one call: S.SetMazeLook
  const control = screen.match(/<div class="look"[\s\S]*?<\/div>/)[0]
  assert.match(control, /role="group" aria-label="The maze picture"/)
  assert.match(control, /aria-pressed=\{look\[key\] === o\.id\}/)
  assert.match(control, /onclick=\{\(e\) => choose\(key, o\.id, e\)\}/)
  assert.ok(!/call\(|S\.|on(keydown|keyup)/.test(control.replace(/onclick=\{\(e\) => choose\(key, o\.id, e\)\}/, '')))
  // Nothing of it is kept on the page: no state of its own, no storage.
  assert.ok(!/mazeLook\.svelte|localStorage|look = \$state/.test(screen))
  // Settings offers the same choices, through the same call.
  const settings = readFileSync(new URL('./SettingsScreen.svelte', import.meta.url), 'utf8')
  assert.match(settings, /const maze = \$derived\(mazeLookOf\(view\.settings\)\)/)
  assert.match(settings, /call\(S\.SetMazeLook, next\.style, next\.zoom\)/)
  assert.match(settings, /\{#each mazeLooks\[p\.key\] as o \(o\.id\)\}/)
  for (const k of ['style', 'zoom']) assert.match(settings, new RegExp(`key: '${k}'`))
  // Shown with or without a game (the roster tools need one; this does not).
  const section = settings.indexOf('id="maze-title"'), gameOnly = settings.indexOf('{#if view.game}<section>')
  assert.ok(section > 0 && section < gameOnly)
})

test('the picture takes the view’s space: a square from its height and width, clear of its corners, at most 720px', () => {
  const screen = readFileSync(new URL('./DungeonScreen.svelte', import.meta.url), 'utf8')
  const view = readFileSync(new URL('./MazeView.svelte', import.meta.url), 'utf8')
  const rule = (src, sel) => src.match(new RegExp(`\\n  ${sel.replace(/\./g, '\\.')} \\{([^}]*)\\}`))?.[1] ?? ''
  // The view is a size container: the picture is sized from it, not from a constant.
  assert.match(rule(screen, '.stage'), /--corner: 184px; position: relative; container-type: size/)
  const picture = rule(screen, '.picture')
  assert.match(picture, /width: min\(100cqh, 100cqw - 2 \* var\(--corner\), 720px\)/)
  assert.match(picture, /aspect-ratio: 1/)
  assert.match(screen, /<div class="picture"><MazeView /)
  // MazeView fills it; no cap of its own any more.
  assert.match(rule(view, '.maze-view'), /width: 100%; height: 100%/)
  assert.ok(!/360px/.test(view))
  // The corners: the minimap is never wider than the column kept for it
  // (168px and its 8px margin, within 184px), so it never reaches the picture.
  const minimap = rule(screen, '.minimap')
  assert.match(minimap, /right: 8px/)
  assert.match(minimap, /height: min\(168px, calc\(100% - 16px\), 26cqw\); aspect-ratio: 1/)
  assert.ok(168 + 8 <= 184)
  // And the style and zoom, in the other corner, within its column.
  assert.match(rule(screen, '.look'), /left: 8px; top: 8px;.*max-width: calc\(var\(--corner\) - 8px\)/)
  // The commands share a line where they fit.
  assert.match(rule(screen, '.commands'), /display: flex; flex-wrap: wrap/)
})
