// How a style paints the projection's primitives (mazeProjection.js). A
// style chooses only how each kind of primitive is painted, or null to leave
// it unpainted; it never adds, moves or removes geometry. Both styles paint
// the same primitives at the same coordinates, in the same order.
//
// Classic (M-ART-1A.1): the Wizardry wireframe. Walls are outlined and
// filled with the background, so nearer walls hide what lies behind them;
// doors are outlined in gold; the void where the view ends is the
// background; floors and ceilings are not drawn.
export const classic = { wall: 'wall', door: 'door', void: 'void', floor: null, ceiling: null }

// painted is the primitives a style draws, in painting order, each with its
// paint class (and, for a style that dresses them, its dress: see enhanced).
// A style's dress sees one primitive, and where the view ends unseen (the
// void's depth, if there is one): nothing else of the view.
export function painted(primitives, style = classic) {
  const end = primitives.find((p) => p.kind === 'void')?.depth
  return primitives.filter((p) => style[p.kind]).map((p) => (style.dress ? { ...p, paint: style[p.kind], ...style.dress(p, end) } : { ...p, paint: style[p.kind] }))
}

// Enhanced (M-ART-1A.2): the same primitives as stone. Every one is painted,
// floors and ceilings too, and each is dressed from its own geometry alone:
//   fill     its colour, darkening with distance (fog), or a gradient (tone)
//   tone     { axis: [x1, y1, x2, y2], from, to }: a surface that recedes
//            (a side wall, a floor, a ceiling) runs from its near plane's
//            shade to its far plane's, so squares meet without a step; a
//            door is flat, at its place's shade
//   edge     the outline's colour (a wall's worn edge, a door's frame)
//   details  lines drawn on the surface, after it: [{ kind, segments }]
//            'mortar'  stone courses and joints on walls
//            'flag'    flagstone joints on floors and ceilings
//            'plank'   a door's planks
//            'band'    a door's iron bands
//            Every segment's ends are points of the surface's own polygon
//            (interpolated between its corners), so a detail never leaves
//            its surface: it shows nothing the surface does not.
// A door looks the same whatever it is (Go says only 'door'), and a hidden
// door Go reports as a wall is a wall, dressed as every wall is.

// Fog: how much of a surface's colour is left at plane t (the near plane
// of square d is t = d, its far plane t = d + 1). Where the view ends
// unseen (the void at plane end: the sight's limit, or a dark square), the
// last square seen also fades towards it, to FADE of its colour at the
// void, so the corridor runs into darkness rather than stopping at a black
// panel; nothing seen fades away entirely.
const FADE = 0.45
export const fog = (t, end) => 0.66 ** t * (end === undefined ? 1 : 1 - (1 - FADE) * Math.min(1, Math.max(0, t - (end - 1))))
export const background = [13, 12, 11] // the stage's #0d0c0b, as the void

const base = {
  front: [112, 104, 94], // facing the party: the lightest stone
  left: [97, 91, 82],
  right: [88, 82, 74], // a shade apart, so the two sides read as two planes
  floor: [84, 74, 62], // worn flagstones, browner
  ceiling: [46, 43, 40], // the darkest
  door: [88, 54, 30], // dark wood
  frame: [150, 136, 112], // the door's stone frame
}

const hex = (c) => '#' + c.map((v) => Math.round(v).toString(16).padStart(2, '0')).join('')
export const shade = (rgb, t, end) => hex(rgb.map((v, i) => background[i] + (v - background[i]) * fog(t, end)))
const lighter = (rgb, k) => rgb.map((v) => v + (230 - v) * k)

const lerp = ([ax, ay], [bx, by], k) => [ax + (bx - ax) * k, ay + (by - ay) * k]
// recede maps a fraction s of a square's depth to the fraction of the way
// from its near plane to its far plane on the picture: the far plane is
// twice as far away, so a point s of the way along is at scale 1/(1 + s).
export const recede = (s) => (2 * s) / (1 + s)

// The levels of detail, by depth: far surfaces are a few pixels across.
const COURSES = 4 // stone courses up a wall; a stone is half a square long
const DETAIL = { course: 3, joint: 2, floor: 3, ceiling: 2, door: 3 }

// surfaceAt places (s, v) on a receding surface: s along its depth, v
// across it. Side wall: [nearTop, farTop, farBottom, nearBottom], v from top
// to bottom. Floor or ceiling: [nearA, nearB, farB, farA], v from A to B.
function surfaceAt(p, face) {
  const [a, b, c, d] = p.points
  return face === 'side'
    ? (s, v) => lerp(lerp(a, d, v), lerp(b, c, v), recede(s))
    : (s, v) => lerp(lerp(a, b, v), lerp(d, c, v), recede(s))
}

// flatAt places (u, v) on a front wall [topLeft, topRight, bottomRight,
// bottomLeft], or on a door [bottomA, topA, topB, bottomB] (u from A to B,
// v from top to bottom).
const flatAt = ([a, b, c, d]) => (u, v) => lerp(lerp(a, d, v), lerp(b, c, v), u)
const doorAt = ([a, b, c, d]) => (u, v) => lerp(lerp(b, a, v), lerp(c, d, v), u)

// masonry: the courses and the staggered joints of a wall, at(along, up).
function masonry(at, depth) {
  const out = []
  if (depth > DETAIL.course) return out
  for (let k = 1; k < COURSES; k++) out.push([at(0, k / COURSES), at(1, k / COURSES)])
  if (depth > DETAIL.joint) return out
  for (let k = 0; k < COURSES; k++) {
    for (const j of k % 2 ? [0.25, 0.75] : [0, 0.5]) out.push([at(j, k / COURSES), at(j, (k + 1) / COURSES)])
  }
  return out
}

// flags: a floor's flagstones, half a square each way, staggered row by
// row; a ceiling's slabs, a joint across every half square.
function flags(at, face, depth) {
  const out = []
  if (face === 'ceiling') {
    if (depth <= DETAIL.ceiling) out.push([at(0, 0), at(0, 1)], [at(0.5, 0), at(0.5, 1)])
    return out
  }
  if (depth > DETAIL.floor) return out
  out.push([at(0, 0), at(0, 1)], [at(0.5, 0), at(0.5, 1)])
  out.push([at(0, 0.5), at(0.5, 0.5)])
  for (const u of [0.25, 0.75]) out.push([at(0.5, u), at(1, u)])
  return out
}

// door: its planks, and two iron bands across them.
function door(p) {
  if (p.depth > DETAIL.door) return []
  const at = doorAt(p.points)
  return [
    { kind: 'plank', segments: [0.25, 0.5, 0.75].map((u) => [at(u, 0), at(u, 1)]) },
    { kind: 'band', segments: [0.22, 0.72].map((v) => [at(0, v), at(1, v)]) },
  ]
}

// tone: a receding surface's gradient, from its near plane to its far one.
function tone(p, rgb, end) {
  const [a, b, , d] = p.points
  const [from, to] = [shade(rgb, p.depth, end), shade(rgb, p.depth + 1, end)]
  // A side wall: along it, from its near edge to its far edge.
  if (p.face === 'side') return { axis: [a[0], 0, b[0], 0], from, to }
  // Floors and ceilings: down (or up) the picture, near edge to far edge.
  return { axis: [0, a[1], 0, d[1]], from, to }
}

const nonEmpty = (details) => details.filter((x) => x.segments.length)

function dress(p, end) {
  const t = p.depth + 1 // a front wall, or a door in one, is on the far plane
  switch (p.kind) {
    case 'void':
      return { fill: hex(background), tone: null, edge: null, details: [] }
    case 'wall': {
      if (p.face === 'front') {
        return { fill: shade(base.front, t, end), tone: null, edge: shade(lighter(base.front, 0.3), t, end),
          details: nonEmpty([{ kind: 'mortar', segments: masonry((u, v) => flatAt(p.points)(u, v), p.depth) }]) }
      }
      const rgb = p.lane < 0 ? base.left : base.right
      const g = tone(p, rgb, end)
      return { fill: `url(#mz-${p.id})`, tone: g, edge: shade(lighter(rgb, 0.3), p.depth + 0.5, end),
        details: nonEmpty([{ kind: 'mortar', segments: masonry(surfaceAt(p, 'side'), p.depth) }]) }
    }
    case 'floor':
    case 'ceiling': {
      const g = tone(p, base[p.kind], end)
      return { fill: `url(#mz-${p.id})`, tone: g, edge: null,
        details: nonEmpty([{ kind: 'flag', segments: flags(surfaceAt(p, p.kind), p.kind, p.depth) }]) }
    }
    case 'door': {
      // A door in a side wall stands halfway along its square.
      const at = p.face === 'side' ? p.depth + 0.5 : t
      return { fill: shade(base.door, at, end), tone: null, edge: shade(base.frame, at, end), details: door(p) }
    }
  }
  throw new Error(`unknown kind ${p.kind}`)
}

export const enhanced = { wall: 'wall', door: 'door', void: 'void', floor: 'floor', ceiling: 'ceiling', dress }

export const styles = { classic, enhanced }

// The zoom presets (M-ART-1A.2): only the picture's frame, a square viewBox
// centred where the corridor vanishes. The projection, the scene and what
// is drawn are the same at every zoom. Every frame lies between the
// party's own square's far plane (|x| = far(0) = 32, so everything past it
// stays in view) and its near plane (near(0) = 64, so nothing is shown
// beside or behind the party), and leaves at least 4 units of the party's
// own square's sides in view, so a wall or an opening beside the party is
// seen at every zoom. Close was to be -30 -30 60 60; that hides both sides
// of the party's square, so it is -36 -36 72 72.
export const zooms = {
  close: '-36 -36 72 72',
  standard: '-40 -40 80 80',
  wide: '-52 -52 104 104',
}

// The provisional appearance (until Settings, M-ART-1A.3).
export const defaultLook = { style: 'enhanced', zoom: 'standard' }

// segmentsPath renders detail segments as one SVG path.
const n = (v) => +v.toFixed(3)
export const segmentsPath = (segments) => segments.map(([[x1, y1], [x2, y2]]) => `M${n(x1)} ${n(y1)}L${n(x2)} ${n(y2)}`).join('')
