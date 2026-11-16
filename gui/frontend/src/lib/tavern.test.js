// ART-3B-1: Gilgamesh's Tavern is illustrated (its picture in scenes.js,
// checked with the others in frameViews.test.js; its story in
// content1a.test.js), and the story box is each place's own: open at a
// place until it is closed there, closed there until Story opens it, and
// every place's open again when the game changes; for the session only.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { scenes, illustrated } from './scenes.js'

const read = (name) => readFileSync(new URL(name, import.meta.url), 'utf8')

test('the Tavern’s picture, centred, described for screen readers', () => {
  assert.ok(illustrated('tavern'))
  assert.match(new URL(scenes.tavern.src).pathname, /\/src\/assets\/scenes\/tavern\.webp$/)
  assert.equal(scenes.tavern.focus, '50% 50%')
  assert.match(scenes.tavern.alt, /^Gilgamesh's Tavern: Gilgamesh, .*behind his bar/)
  // The Castle and the Edge are as they were.
  assert.equal(scenes.castle.focus, '45% 50%')
  assert.equal(scenes.edge.focus, '50% 50%')
})

test('the story box is open or closed at each place, for the session', () => {
  const app = read('../App.svelte')
  // One record of the places whose box is closed; no single state for all.
  assert.match(app, /let storyClosed = \$state\(\{\}\) \/\/ the places whose story box is closed/)
  assert.ok(!/storyOpen/.test(app), 'no story state shared by every place')
  // The frame's story.open is the place's: read and set for where the
  // player stands; a place not yet closed is open.
  assert.match(app, /get open\(\) \{ return !storyClosed\[view\?\.location\] \},\s*set open\(v\) \{ storyClosed\[view\.location\] = !v \},/)
  // Where the box stands is one, at every place.
  assert.match(app, /get rect\(\) \{ return storyRect \},\s*set rect\(r\) \{ storyRect = r \},/)
  // Another game: every place's open again (NAV-1A).
  assert.match(app, /game !== storyGame\) \{\s*storyGame = game\s*storyClosed = \{\}\s*\}/)
  // Nothing of it is saved: it is not sent to Go.
  assert.ok(!/storyClosed[^\n]*S\.|S\.[^\n]*storyClosed/.test(app))
})

test('the frame’s story box is as it was: the sheet first, Story to open it, × to close it', () => {
  const frame = read('./LocationFrame.svelte')
  assert.match(frame, /const storyOpen = \$derived\(storyHere && panels\.story\.open && !sheetShown\)/)
  assert.match(frame, /function closeStory\(\) \{\s*panels\.story\.open = false/)
  assert.match(frame, /function openStory\(\) \{\s*if \(sheetShown\) sheet\.hide\(\)\s*panels\.story\.open = true\s*\}/)
})
