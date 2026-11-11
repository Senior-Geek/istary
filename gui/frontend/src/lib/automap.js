// Drawing helpers for the maps (M-1): the minimap beside the dungeon view
// (view.minimap, Go's app.LocalMapView) and the full map of an explored
// level (Session.Automap, app.MapView). They only turn what Go sent into
// lines and marks; which walls exist, which are one way, which ways cross
// the level's edge, where the party is and which way it faces are all Go's.
//
// Both maps are drawn in square units: square (row, col) spans x col..col+1
// and y row..row+1, north up (row 0 at the top).

// INSET is how far inside its square a one-way wall is drawn, so it reads
// as that side's wall only.
export const INSET = 0.16

// edge is the line along side ('N', 'E', 'S' or 'W') of square (row, col),
// moved inset into the square.
export function edge(row, col, side, inset = 0) {
  switch (side) {
    case 'N': return { x1: col, y1: row + inset, x2: col + 1, y2: row + inset }
    case 'S': return { x1: col, y1: row + 1 - inset, x2: col + 1, y2: row + 1 - inset }
    case 'W': return { x1: col + inset, y1: row, x2: col + inset, y2: row + 1 }
    default: return { x1: col + 1 - inset, y1: row, x2: col + 1 - inset, y2: row + 1 }
  }
}

// door is the yellow mark of a door: the middle of its wall's line.
export function door(line) {
  const at = (t) => [line.x1 + t * (line.x2 - line.x1), line.y1 + t * (line.y2 - line.y1)]
  const [x1, y1] = at(0.3), [x2, y2] = at(0.7)
  return { x1, y1, x2, y2 }
}

// facingTurn is the arrow's rotation for a facing, in degrees.
export const facingTurn = { N: 0, E: 90, S: 180, W: 270 }

// arrow is the party's arrow at the centre of its square, pointing its way.
export function arrow(party) {
  const cx = party.col + 0.5, cy = party.row + 0.5
  return { cx, cy, turn: facingTurn[party.facing] ?? 0, points: `${cx},${cy - 0.36} ${cx + 0.27},${cy + 0.3} ${cx},${cy + 0.14} ${cx - 0.27},${cy + 0.3}` }
}

// symbols are the features the maps draw: each a small icon at the centre
// of its square, in paths on a -0.5..0.5 box.
export const symbols = {
  'stairs-up': { label: 'Stairs up', path: 'M-0.32,0.3 h0.21 v-0.17 h0.21 v-0.17 h0.21 v-0.17 M-0.3,-0.06 l0.14,-0.2 l0.14,0.2', cls: 'stairs' },
  'stairs-down': { label: 'Stairs down', path: 'M-0.32,-0.21 h0.21 v0.17 h0.21 v0.17 h0.21 v0.17 M-0.3,0.06 l0.14,0.2 l0.14,-0.2', cls: 'stairs' },
  elevator: { label: 'Elevator', path: 'M-0.26,-0.32 h0.52 v0.64 h-0.52 z M-0.12,-0.04 l0.12,-0.16 l0.12,0.16 z M-0.12,0.06 l0.12,0.16 l0.12,-0.16 z', cls: 'elevator' },
}

// feature places a feature's symbol, or null for a kind the maps do not
// draw yet (teleporters, chutes, special squares). Where stranded
// characters lie is a marker of its own (marker), not a feature.
export function feature(f) {
  const s = symbols[f.kind]
  return s ? { ...s, kind: f.kind, x: f.col + 0.5, y: f.row + 0.5, scale: 1, badge: false } : null
}

// BADGE is where a feature's symbol goes when the party stands on its
// square: shrunk on a dark disc in the square's corner ahead and to the
// party's right, clear of the arrow whichever way it faces.
export const BADGE = { offset: 0.285, scale: 0.6, radius: 0.21 }

// underParty moves a feature's symbol out from under the party's arrow.
export function underParty(f, party) {
  const t = (facingTurn[party.facing] ?? 0) * Math.PI / 180
  const [ax, ay] = [BADGE.offset, -BADGE.offset] // ahead-right when facing north
  const dx = ax * Math.cos(t) - ay * Math.sin(t), dy = ax * Math.sin(t) + ay * Math.cos(t)
  return { ...f, x: f.x + dx, y: f.y + dy, scale: BADGE.scale, badge: true }
}

// MARKER is the stranded marker's size: a diamond of this half-width at
// the centre of its square, and when the party stands there, shrunk on a
// disc in the corner ahead and to the party's left: the arrow's tail fills
// the corners behind it, and a feature's badge takes the one ahead-right,
// so neither covers the arrow nor the other.
export const MARKER = { half: 0.3, scale: 0.62, radius: 0.21 }

// marker places a square's stranded marker (Go's app.MapStranded): one
// whatever the count, the count shown when more than one lie there.
export function marker(m, party) {
  const out = {
    place: m.place, count: m.count, characters: m.characters ?? [],
    x: m.col + 0.5, y: m.row + 0.5, scale: 1, badge: false,
    label: strandedLabel(m),
  }
  if (party && party.row === m.row && party.col === m.col) {
    const t = (facingTurn[party.facing] ?? 0) * Math.PI / 180
    const [ax, ay] = [-BADGE.offset, -BADGE.offset] // ahead-left when facing north
    out.x += ax * Math.cos(t) - ay * Math.sin(t)
    out.y += ax * Math.sin(t) + ay * Math.cos(t)
    out.scale = MARKER.scale
    out.badge = true
  }
  return out
}

// diamond is the marker's outline, centred on (0, 0) before its scale.
export const diamond = `0,${-MARKER.half} ${MARKER.half},0 0,${MARKER.half} ${-MARKER.half},0`

// strandedLabel is a marker's line: "L1:R11:C5 — 3 stranded".
export const strandedLabel = (m) => `${m.place} — ${m.count} stranded`

// wrap marks a way across the level's edge, at the square it leaves from:
// a chevron just outside the frame, pointing out.
export function wrap(w) {
  const mid = { N: [w.col + 0.5, w.row], S: [w.col + 0.5, w.row + 1], W: [w.col, w.row + 0.5], E: [w.col + 1, w.row + 0.5] }[w.side]
  const out = { N: [0, -1], S: [0, 1], W: [-1, 0], E: [1, 0] }[w.side]
  const [mx, my] = mid, [ox, oy] = out
  const tip = [mx + ox * 0.42, my + oy * 0.42], base = [mx + ox * 0.14, my + oy * 0.14]
  const px = -oy * 0.22, py = ox * 0.22
  return {
    kind: w.kind, oneWay: !!w.oneWay,
    points: `${base[0] + px},${base[1] + py} ${tip[0]},${tip[1]} ${base[0] - px},${base[1] - py}`,
  }
}

// CLIP is how far past the minimap's window its drawing may run, so the
// marks on the window's outer walls (a door's, the wall's own line) are
// drawn whole.
export const CLIP = 0.14

// drawing turns a map view (local or full) into what the SVG draws.
export function drawing(view) {
  if (!view) return null
  const walls = (view.walls ?? []).map((w) => {
    const line = edge(w.row, w.col, w.side, w.oneWay ? INSET : 0)
    return { ...line, kind: w.kind, oneWay: !!w.oneWay, door: w.kind === 'door' ? door(line) : null }
  })
  const seams = []
  if (view.northEdge >= 0) seams.push({ x1: 0, y1: view.northEdge, x2: view.size, y2: view.northEdge })
  if (view.westEdge >= 0) seams.push({ x1: view.westEdge, y1: 0, x2: view.westEdge, y2: view.size })
  const p = view.party
  const features = (view.features ?? []).map((f) => {
    const s = feature(f)
    return s && p && f.row === p.row && f.col === p.col ? underParty(s, p) : s
  }).filter(Boolean)
  return {
    size: view.size,
    floors: (view.visited ?? []).map((q) => ({ x: q.col, y: q.row })),
    // The same squares as one shape, so neighbours meet without a seam.
    floor: (view.visited ?? []).map((q) => `M${q.col},${q.row}h1v1h-1z`).join(''),
    walls,
    features,
    wraps: (view.wraps ?? []).map(wrap),
    seams,
    stranded: (view.stranded ?? []).map((m) => marker(m, p)),
    party: view.party ? arrow(view.party) : null,
  }
}

// mapKey decides what a key does to the full map: 'open' or 'close' it
// (F10 toggles; Escape closes), 'block' a key while it is open (so no key
// reaches the maze behind it), or null to let the key through.
export function mapKey(key, open) {
  if (key === 'F10') return open ? 'close' : 'open'
  if (!open) return null
  return key === 'Escape' ? 'close' : 'block'
}

// levelLabel names a level for the map's level choice.
export const levelLabel = (l) => `Level ${l}`
