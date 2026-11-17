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

test('the style and zoom are chosen without the backend, are never saved, and give the keys back to the maze', () => {
  const screen = readFileSync(new URL('./DungeonScreen.svelte', import.meta.url), 'utf8')
  const choose = screen.match(/function choose\(key, value, e\) \{([^}]*)\}/)[1]
  assert.match(choose, /look\[key\] = value/)
  assert.match(choose, /e\.currentTarget\.blur\(\)/)
  assert.ok(!/call|S\.|Session/.test(choose))
  const control = screen.match(/<div class="look"[\s\S]*?<\/div>/)[0]
  assert.ok(!/call\(|S\./.test(control))
  assert.match(control, /aria-pressed=\{look\[key\] === value\}/)
  const state = readFileSync(new URL('./mazeLook.svelte.js', import.meta.url), 'utf8')
  assert.ok(!/wailsjs|Session|localStorage|settings/i.test(state.replace(/\/\/.*$/gm, '')))
  assert.match(state, /export const look = \$state\(\{ \.\.\.defaultLook \}\)/)
})
