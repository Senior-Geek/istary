// The perspective projection (M-ART-1A.1): the scene's surfaces
// (mazeScene.js) as polygons, in painting order. One projection for every
// style: a style chooses only how each primitive is painted.
//
// The geometry is the Classic view's (Pascal DRAWMAZE, render/dungeon.go's
// DrawMaze), on a square viewBox centred on 0. Square d of the corridor
// runs from its near plane (half-size near(d)) to its far plane (half-size
// far(d)), and each square is half the size of the one before: DrawMaze's
// WALWIDTH (32, then halved at each step) is far(d), its 82-pixel picture
// is -40..40 here, and its front wall at depth 0 (8..72) is -32..32.
//   front wall        the far-plane square, |x|, |y| <= far(d)
//   side wall         the trapezoid between the planes, |x| from near(d) to far(d)
//   beside (lane ±1)  a front wall shifted by ±2·far(d) (DRAWFRNT's LRCENT)
//   floor, ceiling    the trapezoids between the planes, below and above
//   doors             DRAWMAZE's: DOORFRAM = far/4 in from the wall's far
//                     edge and top, DOORWIDT = far/2 wide on a side wall;
//                     on a front wall from 3/4·far left to 3/4·far right
//                     and from the floor to 3/4·far above the centre.
//   void              where the view ends unseen (the sight's limit, or a
//                     dark square): the plane where the unseen square
//                     begins, across the corridor and both sides.
//
// Painting order: the void first, then the farthest square to the nearest,
// so a nearer wall covers whatever lies behind it (what DrawMaze clips).
// Within a square nothing overlaps: floor and ceiling, then the left side,
// the right side and the front, each door straight after its wall.
//
// Each primitive: { id, kind, seen, depth, lane, face, points, order }
//   kind    'wall', 'door' (a door's outline, id '<wall id>.door'),
//           'floor', 'ceiling' or 'void'
//   points  [[x, y], ...], y downwards as in SVG

import { mazeScene } from './mazeScene.js'

export const viewBox = '-40 -40 80 80'
export const far = (d) => 32 / 2 ** d
export const near = (d) => 2 * far(d)

const rect = (cx, h) => [[cx - h, -h], [cx + h, -h], [cx + h, h], [cx - h, h]]

// side is a side wall between the planes of square d (sign -1 left, +1 right).
const side = (d, sign) => {
  const n = near(d), f = far(d)
  return [[sign * n, -n], [sign * f, -f], [sign * f, f], [sign * n, n]]
}

// sideDoor is a door in that wall: from DOORFRAM to DOORFRAM + DOORWIDT in
// from the far edge, its top DOORFRAM below the wall's top, to the floor.
const sideDoor = (d, sign) => {
  const f = far(d), a = f + f / 4, b = f + f / 4 + f / 2
  return [[sign * a, a], [sign * a, -a + f / 4], [sign * b, -b + f / 4], [sign * b, b]]
}

const frontDoor = (d, cx) => {
  const f = far(d), w = (3 * f) / 4
  return [[cx - w, f], [cx - w, -w], [cx + w, -w], [cx + w, f]]
}

// band is a floor (y = +1) or ceiling (y = -1) between the planes of square
// d, in lane 0 or beside it.
const band = (d, lane, y) => {
  const n = near(d), f = far(d)
  const [n0, n1] = lane ? [n, 3 * n].map((v) => lane * v) : [-n, n]
  const [f0, f1] = lane ? [f, 3 * f].map((v) => lane * v) : [-f, f]
  return [[n0, y * n], [n1, y * n], [f1, y * f], [f0, y * f]]
}

// shape gives a surface's polygon.
function shape(s) {
  const f = far(s.depth)
  switch (s.face) {
    case 'side': return side(s.depth, s.lane)
    case 'front': return rect(2 * f * s.lane, f)
    case 'floor': return band(s.depth, s.lane, 1)
    case 'ceiling': return band(s.depth, s.lane, -1)
  }
  throw new Error(`unknown face ${s.face}`)
}

// within orders a square's surfaces: floors, ceilings, the left side (its
// wall, or the wall ahead of the square beside), the right side, the front.
const within = (s) => (s.face === 'floor' ? 0 : s.face === 'ceiling' ? 1 : s.lane < 0 ? 2 : s.lane > 0 ? 3 : 4)

// project returns the scene's primitives in painting order.
export function project(scene) {
  const out = []
  const { end } = scene
  if (end.kind === 'limit' || end.kind === 'dark') {
    const h = near(end.depth)
    out.push({ id: `d${end.depth}.void`, kind: 'void', seen: end.kind, depth: end.depth, lane: 0, face: 'void',
      points: [[-3 * h, -h], [3 * h, -h], [3 * h, h], [-3 * h, h]] })
  }
  const sorted = [...scene.surfaces].sort((a, b) => b.depth - a.depth || within(a) - within(b))
  for (const s of sorted) {
    const kind = s.face === 'floor' || s.face === 'ceiling' ? s.face : 'wall'
    out.push({ id: s.id, kind, seen: s.seen, depth: s.depth, lane: s.lane, face: s.face, points: shape(s) })
    if (s.seen === 'door') {
      const points = s.face === 'side' ? sideDoor(s.depth, s.lane) : frontDoor(s.depth, 2 * far(s.depth) * s.lane)
      out.push({ id: s.id + '.door', kind: 'door', seen: 'door', depth: s.depth, lane: s.lane, face: s.face, points })
    }
  }
  return out.map((p, order) => ({ ...p, order }))
}

// perspective is the dungeon view's primitives: mazeScene, then project.
export const perspective = (dungeon) => project(mazeScene(dungeon))

// points renders a primitive's points for an SVG polygon.
export const points = (p) => p.points.map(([x, y]) => `${x},${y}`).join(' ')
