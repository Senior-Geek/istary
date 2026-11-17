// Display helpers for the dungeon screen. They only reshape what Go sent in
// view.dungeon (app.DungeonView); no rule is decided here. What is seen, what
// is legal and what a step does are all Go's.

// The first-person picture is drawn by MazeView (M-ART-1A.1) from the
// perspective scene (mazeScene.js) and its projection (mazeProjection.js),
// in the proportions of the Classic view. scene and points are kept here for
// the screen and the tests that use them: scene(corridor) is the walls and
// doors of a corridor, in painting order (farthest square first), each
// { kind: 'wall' | 'door', points, depth }.
import { perspective, points, viewBox, far, near } from './mazeProjection.js'

export { points, viewBox, far, near }

export const scene = (corridor = []) =>
  perspective({ phase: 'maze', corridor })
    .filter((p) => p.kind === 'wall' || p.kind === 'door')
    .map(({ kind, points, depth }) => ({ kind, points, depth }))

// The dungeon actions: label, key hint, and the keys that perform them. The
// keys follow the terminal (F/L/R/K/C/I, A/W/D) plus the arrow keys.
export const actions = {
  forward: { label: 'Forward', hint: 'F ↑', keys: ['f', 'w', 'ArrowUp'] },
  left: { label: 'Left', hint: 'L ←', keys: ['l', 'a', 'ArrowLeft'] },
  right: { label: 'Right', hint: 'R →', keys: ['r', 'd', 'ArrowRight'] },
  kick: { label: 'Kick', hint: 'K', keys: ['k'] },
  camp: { label: 'Camp', hint: 'C', keys: ['c'] },
  inspect: { label: 'Inspect / Rescue', hint: 'I', keys: ['i'] }, // UX-1: Inspect is where stranded characters are picked up
  continue: { label: 'Continue', hint: '↵', keys: ['Enter', ' '] },
  yes: { label: 'Yes', hint: 'Y', keys: ['y'] },
  no: { label: 'No', hint: 'N', keys: ['n'] },
  'pick-up': { label: 'Pick up', hint: '1-9', keys: [] }, // one character at a time: a digit, or its button
  leave: { label: 'Leave', hint: 'L', keys: ['l'] },
  'leave-camp': { label: 'Leave camp', hint: 'L', keys: ['l'] },
}

// suggested is the answer the screen highlights: the stairs' own (Go's
// DungeonView.Default, by how the party came to them), otherwise the first
// open action. Only highlighted; no key takes it.
export const suggested = (dungeon) => dungeon?.default || (dungeon?.actions ?? [])[0]

// keyAction maps a key press to one of the view's open actions, or null.
// At the buttons, a letter presses that button; at Inspect, a digit picks
// up that character (Inspect.found[].n), if the party has room.
export function keyAction(key, dungeon) {
  const open = dungeon?.actions ?? []
  if (open.includes('button')) {
    const i = (dungeon.buttons ?? []).indexOf(key.toUpperCase())
    return i >= 0 ? { action: 'button', button: i + 1 } : null
  }
  if (open.includes('pick-up') && /^[1-9]$/.test(key)) {
    const n = Number(key)
    const f = dungeon.inspect?.found?.find((x) => x.n === n)
    return f?.canPick ? { action: 'pick-up', button: n } : null
  }
  const k = key.length === 1 ? key.toLowerCase() : key
  const action = open.find((a) => actions[a]?.keys.includes(k))
  return action ? { action, button: 0 } : null
}

// expeditionTag names what is under way, for the dungeon and combat
// screens: the real expedition, which is saved, or practice, which is not.
export function expeditionTag(view) {
  if (view?.real) return 'Real expedition · saved after every action'
  const what = view?.expedition ? 'Practice expedition' : 'Practice encounter'
  return what + (view?.practiceChest ? ' · with a chest' : '') + ' · results are not kept'
}
