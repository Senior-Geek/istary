// MazeView (M-ART-1A.1): the Classic picture of the dungeon view, made of
// the projection alone (mazeProjection.test.js covers what it is drawn from).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

test('MazeView draws the Classic paint of the dungeon view alone, and takes no input', () => {
  const view = readFileSync(new URL('./MazeView.svelte', import.meta.url), 'utf8')
  const imports = [...view.matchAll(/import[^'"]*['"]([^'"]+)['"]/g)].map((m) => m[1]).sort()
  assert.deepEqual(imports, ['./mazeProjection.js', './mazeStyle.js'])
  assert.match(view, /let \{ dungeon \} = \$props\(\)/)
  assert.match(view, /const drawn = \$derived\(painted\(perspective\(dungeon\), classic\)\)/)
  assert.ok(!/wailsjs|Session|minimap|automap|position|on[a-z]+=|tabindex|Math\.random/i.test(view.replace(/\/\/.*$/gm, '')))
  assert.match(view, /role="img"/)
  // The dungeon screen gives it view.dungeon, and draws no picture of its own.
  const screen = readFileSync(new URL('./DungeonScreen.svelte', import.meta.url), 'utf8')
  assert.match(screen, /<MazeView dungeon=\{d\} \/>/)
  assert.ok(!/<polygon/.test(screen))
})
