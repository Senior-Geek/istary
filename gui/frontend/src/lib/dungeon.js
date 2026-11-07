// Display helpers for the dungeon screen. They only reshape what Go sent in
// view.dungeon (app.DungeonView); no rule is decided here. What is seen, what
// is legal and what a step does are all Go's.

// The first-person picture is a simple wireframe in the proportions of the
// Classic view, on a square viewBox centred on 0 (-40..40). Square d of the
// corridor runs from its near plane (half-size near(d)) to its far plane
// (half-size far(d)); each square is half the size of the one before.
export const viewBox = '-40 -40 80 80'
export const far = (d) => 32 / 2 ** d
export const near = (d) => 2 * far(d)

// A wall seen from the corridor is "wall" or "door"; "open" draws nothing.
const solid = (w) => w === 'wall' || w === 'door'

// side returns the trapezoid of a side wall (sign -1 left, +1 right) and, for
// a door, its outline within the wall.
function side(d, sign, kind) {
  const n = near(d), f = far(d)
  const shapes = [{ kind: 'wall', points: [[sign * n, -n], [sign * f, -f], [sign * f, f], [sign * n, n]] }]
  if (kind === 'door') {
    // From a quarter to three quarters along the wall, two thirds high.
    const at = (t) => n - t * (n - f)
    const a = at(0.25), b = at(0.75)
    shapes.push({ kind: 'door', points: [[sign * a, a], [sign * a, -a / 3], [sign * b, -b / 3], [sign * b, b]] })
  }
  return shapes
}

// front returns a wall facing the party, centred at x = cx.
function front(d, cx, kind) {
  const f = far(d)
  const shapes = [{ kind: 'wall', points: [[cx - f, -f], [cx + f, -f], [cx + f, f], [cx - f, f]] }]
  if (kind === 'door') {
    const w = f / 2
    shapes.push({ kind: 'door', points: [[cx - w, f], [cx - w, -f / 3], [cx + w, -f / 3], [cx + w, f]] })
  }
  return shapes
}

// scene turns the corridor into shapes to draw, in painting order: farthest
// square first, so each square's filled walls cover whatever lies behind them
// (a wall ahead of a far side opening reaches out behind the nearer side
// walls). Within a square the shapes never overlap; a door follows its wall.
export function scene(corridor = []) {
  const out = []
  corridor.forEach((sq, d) => {
    const f = far(d), shapes = []
    if (solid(sq.left)) shapes.push(...side(d, -1, sq.left))
    else if (solid(sq.leftAhead)) shapes.push(...front(d, -2 * f, sq.leftAhead))
    if (solid(sq.right)) shapes.push(...side(d, 1, sq.right))
    else if (solid(sq.rightAhead)) shapes.push(...front(d, 2 * f, sq.rightAhead))
    if (solid(sq.front)) shapes.push(...front(d, 0, sq.front))
    out.unshift(...shapes.map((s) => ({ ...s, depth: d })))
  })
  return out
}

// points renders a shape's points for an SVG polygon.
export const points = (shape) => shape.points.map(([x, y]) => `${x},${y}`).join(' ')

// The dungeon actions: label, key hint, and the keys that perform them. The
// keys follow the terminal (F/L/R/K/C/I, A/W/D) plus the arrow keys.
export const actions = {
  forward: { label: 'Forward', hint: 'F ↑', keys: ['f', 'w', 'ArrowUp'] },
  left: { label: 'Left', hint: 'L ←', keys: ['l', 'a', 'ArrowLeft'] },
  right: { label: 'Right', hint: 'R →', keys: ['r', 'd', 'ArrowRight'] },
  kick: { label: 'Kick', hint: 'K', keys: ['k'] },
  camp: { label: 'Camp', hint: 'C', keys: ['c'] },
  inspect: { label: 'Inspect', hint: 'I', keys: ['i'] },
  continue: { label: 'Continue', hint: '↵', keys: ['Enter', ' '] },
  yes: { label: 'Yes', hint: 'Y', keys: ['y'] },
  no: { label: 'No', hint: 'N', keys: ['n'] },
  'pick-up': { label: 'Pick up', hint: 'P', keys: ['p'] },
  leave: { label: 'Leave', hint: 'L', keys: ['l'] },
  'leave-camp': { label: 'Leave camp', hint: 'L', keys: ['l'] },
}

// keyAction maps a key press to one of the view's open actions, or null.
// At the buttons, a letter presses that button.
export function keyAction(key, dungeon) {
  const open = dungeon?.actions ?? []
  if (open.includes('button')) {
    const i = (dungeon.buttons ?? []).indexOf(key.toUpperCase())
    return i >= 0 ? { action: 'button', button: i + 1 } : null
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
