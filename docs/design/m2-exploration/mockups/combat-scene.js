// Istary M2 combat prototype: a throwaway projection renderer.
//
// NOT production code and not part of the application. It extends the idea of
// scene.js (same corridor, same projection) with what the combat prototype
// needs: a team square that can be approached, the squares behind it, the
// four scenery policies, figure states, monster groups and spell effects.
// scene.js is left unchanged so the original storyboard stays intact.
//
// World axes: x = east (right when facing north), y = up, z = north.
// A square is 4 units; walls are 3 units high. Square (cx, cz) spans
// x in [4cx-2, 4cx+2] and z in [4cz-2, 4cz+2].

(function (root) {
  'use strict'

  const S = 4
  const H = 3

  // The fixture location: the original corridor (squares 0..3 and the niche)
  // plus what lies behind and beside it. Fixture premise: the team arrived by
  // the stairs at (0,-2) facing north and walked to (0,0). It never looked
  // back, so the door at the south end of (0,-2) and the room behind the side
  // door at (0,1) have never been in any view.
  const MAP = {
    '0,-3': { N: 'door', S: 'wall', W: 'wall', E: 'wall' },
    '0,-2': { N: 'open', S: 'door', W: 'wall', E: 'wall', stairs: true },
    '0,-1': { N: 'open', S: 'open', W: 'wall', E: 'wall' },
    '0,0': { N: 'open', S: 'open', W: 'wall', E: 'wall' },
    '0,1': { N: 'open', S: 'open', W: 'door', E: 'open' },
    '1,1': { N: 'wall', S: 'wall', W: 'open', E: 'wall' },
    '-1,1': { N: 'wall', S: 'wall', W: 'wall', E: 'door' },
    '0,2': { N: 'open', S: 'open', W: 'wall', E: 'wall' },
    '0,3': { N: 'wall', S: 'open', W: 'wall', E: 'wall' },
  }
  // Further fixture locations for the difficult-case validation, in the same
  // team-local frame: the team stands at (0,0) facing north and arrived from
  // the south along `path`. Only `corridor` is used by the earlier pages.
  function grid(cells) { const m = {}; for (const [k, v] of Object.entries(cells)) m[k] = Object.assign({ N: 'wall', S: 'wall', W: 'wall', E: 'wall' }, v); return m }
  const BEHIND = { '0,-3': { N: 'door' }, '0,-2': { N: 'open', S: 'door', stairs: true }, '0,-1': { N: 'open', S: 'open' } }
  const ROOM = Object.assign({}, BEHIND)
  for (let x = -1; x <= 1; x++) for (let z = 0; z <= 2; z++) ROOM[x + ',' + z] = { W: x === -1 ? 'wall' : 'open', E: x === 1 ? (z === 2 ? 'door' : 'wall') : 'open', S: z === 0 ? (x === 0 ? 'open' : 'wall') : 'open', N: z === 2 ? 'wall' : 'open' }
  const LOCATIONS = {
    corridor: { name: 'Corridor', map: MAP, path: [-2, -1] },
    room: { name: 'Room (3 × 3, entered from the south)', map: grid(ROOM), path: [-2, -1] },
    junction: { name: 'Junction (crossing one square ahead)', path: [-2, -1], map: grid(Object.assign({}, BEHIND, {
      '0,0': { N: 'open', S: 'open' }, '0,1': { N: 'open', S: 'open', W: 'open', E: 'open' },
      '-1,1': { W: 'open', E: 'open' }, '-2,1': { E: 'open', W: 'open' }, '-3,1': { E: 'open', N: 'door' },
      '1,1': { W: 'open', E: 'open' }, '2,1': { W: 'open', E: 'open' }, '3,1': { W: 'open', S: 'open' },
      '0,2': { N: 'open', S: 'open' }, '0,3': { S: 'open', N: 'open' }, '0,4': { S: 'open' } })) },
    deadend: { name: 'Narrow dead end (one square ahead)', path: [-2, -1], map: grid(Object.assign({}, BEHIND, {
      '0,0': { N: 'open', S: 'open' }, '0,1': { S: 'open', N: 'wall' } })) },
  }
  let curMap = MAP
  const edge = (cx, cz, side) => { const c = curMap[cx + ',' + cz]; return c ? c[side] : 'wall' }
  const useLoc = (loc) => { curMap = (LOCATIONS[loc] || LOCATIONS.corridor).map; return LOCATIONS[loc] || LOCATIONS.corridor }
  const key = (f) => `${f.kind}:${f.cx},${f.cz}:${f.side || ''}`

  // The Classic DRAWMAZE walk (render/dungeon.go), facing north from (0,pz).
  // Same shape as scene.js visibleFaces, with the team's row as a parameter.
  function visibleFrom(pz, lightDis) {
    const faces = []
    for (let d = 0; d < lightDis; d++) {
      const cz = pz + d
      faces.push({ kind: 'floor', cx: 0, cz }, { kind: 'ceil', cx: 0, cz })
      for (const [side, dx] of [['W', -1], ['E', 1]]) {
        const t = edge(0, cz, side)
        if (t !== 'open') faces.push({ kind: 'wall', cx: 0, cz, side, type: t })
        else {
          const f = edge(dx, cz, 'N')
          if (f !== 'open') faces.push({ kind: 'wall', cx: dx, cz, side: 'N', type: f }, { kind: 'floor', cx: dx, cz }, { kind: 'ceil', cx: dx, cz })
        }
      }
      const front = edge(0, cz, 'N')
      if (front !== 'open') { faces.push({ kind: 'wall', cx: 0, cz, side: 'N', type: front }); break }
    }
    return faces
  }

  function union(...lists) {
    const m = new Map()
    for (const l of lists) for (const f of l) if (!m.has(key(f))) m.set(key(f), f)
    return [...m.values()]
  }

  // Every surface of every fixture square: what a renderer holding the real
  // level data could draw. Used only by the "unbounded" hazard demonstration.
  function allFaces() {
    const faces = []
    for (const k of Object.keys(curMap)) {
      const [cx, cz] = k.split(',').map(Number)
      faces.push({ kind: 'floor', cx, cz }, { kind: 'ceil', cx, cz })
      for (const side of ['N', 'S', 'W', 'E']) if (curMap[k][side] !== 'open') faces.push({ kind: 'wall', cx, cz, side, type: curMap[k][side] })
    }
    return faces
  }

  // Scenery policies (COMBAT-EXPERIENCE-REVIEW.md §4):
  //   strict      the Classic visible set from the team's square; nothing else
  //   shell       visible set + a procedural shell (fog, floor fade) that is
  //               generated from the camera alone and reads no level data
  //   remembered  visible set + surfaces already shown to the user on the way
  //               here (an exploration record) + the shell beyond
  //   unbounded   true level geometry around the camera (hazard demo only)
  function faceSet(policy, teamZ, lightDis, loc) {
    const L = useLoc(loc)
    const known = visibleFrom(teamZ, lightDis)
    const remembered = union(...L.path.map((z) => visibleFrom(z, lightDis)), known)
    const permitted = new Set(remembered.map(key))
    if (policy === 'strict' || policy === 'shell') return { faces: known.map((f) => ({ f })), shell: policy === 'shell' }
    if (policy === 'remembered') {
      const now = new Set(known.map(key))
      return { faces: remembered.map((f) => ({ f, memory: !now.has(key(f)) })), shell: true }
    }
    return { faces: allFaces().map((f) => ({ f, leak: !permitted.has(key(f)) })), shell: false }
  }

  // Presentation placement of monster groups (difficult-case validation).
  // Uses only what the team can see from its square: the floor of its own
  // column ahead, up to the first visible wall. It never reads the level
  // beyond the visible set, so where figures stand cannot reveal anything.
  // Group 1 is nearest, as the group order is. Large creatures take two
  // places in a row. Rows compress when the visible floor is short.
  function layout(groups, loc, lightDis) {
    useLoc(loc)
    const vis = visibleFrom(0, lightDis || 5)
    const ahead = vis.filter((f) => f.kind === 'floor' && f.cx === 0 && f.cz >= 1).map((f) => f.cz)
    const zmax = ahead.length ? Math.max(...ahead) * S + S / 2 - 0.55 : 2
    const z0 = 4.0, depthCap = 7.6
    const rows = groups.map((g) => { const per = g.look === 'large' ? 2 : 4; return Math.ceil(g.n / per) })
    const need = rows.reduce((a, r) => a + r, 0) + 0.6 * (groups.length - 1)
    const step = Math.max(0.3, Math.min(1.0, Math.min(depthCap, zmax - z0) / Math.max(1, need)))
    // Short floor (a dead end): start nearer the team rather than overrun
    // the wall.
    let z = Math.max(2.9, Math.min(z0, zmax - (need - 0.6) * step))
    return groups.map((g, gi) => {
      const per = g.look === 'large' ? 2 : 4, out = [], sp = g.look === 'large' ? 1.7 : 0.95
      // Narrow groups after the first stand alternately west and east of
      // the centre line, so a group behind is not hidden by the one in front.
      const width = (Math.min(per, g.n) - 1) * sp + (g.look === 'large' ? 1.2 : 0.7)
      const shift = gi ? (gi % 2 ? -1 : 1) * Math.max(0, Math.min(0.85, (3.5 - width) / 2)) : 0
      for (let r = 0; r < rows[gi]; r++) {
        const inRow = Math.min(per, g.n - r * per)
        for (let i = 0; i < inRow; i++) out.push([+((i - (inRow - 1) / 2) * sp + (r % 2 ? 0.22 : 0) + shift).toFixed(2), +(z + r * step).toFixed(2)])
      }
      z += rows[gi] * step + 0.6 * step
      return out
    })
  }

  // ── geometry ────────────────────────────────────────────────────────────
  const sub = (a, b) => [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
  const dot = (a, b) => a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
  const cross = (a, b) => [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]]
  const norm = (a) => { const l = Math.hypot(a[0], a[1], a[2]) || 1; return [a[0] / l, a[1] / l, a[2] / l] }
  const lerp3 = (a, b, t) => [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t]

  function camera(pos, target, fovDeg, W, Hpx) {
    const f = norm(sub(target, pos))
    const r = norm(cross([0, 1, 0], f))
    const u = cross(f, r)
    const focal = (W / 2) / Math.tan((fovDeg * Math.PI) / 360)
    return {
      view: (p) => { const v = sub(p, pos); return [dot(v, r), dot(v, u), dot(v, f)] },
      proj: (c) => [W / 2 + (c[0] / c[2]) * focal, Hpx / 2 - (c[1] / c[2]) * focal],
    }
  }

  const NEAR = 0.05
  function clipNear(poly) {
    const out = []
    for (let i = 0; i < poly.length; i++) {
      const a = poly[i], b = poly[(i + 1) % poly.length]
      const ain = a[2] > NEAR, bin = b[2] > NEAR
      if (ain) out.push(a)
      if (ain !== bin) { const t = (NEAR - a[2]) / (b[2] - a[2]); out.push([a[0] + t * (b[0] - a[0]), a[1] + t * (b[1] - a[1]), NEAR]) }
    }
    return out
  }
  function polyPath(cam, pts) {
    const v = clipNear(pts.map(cam.view))
    if (v.length < 3) return null
    return { d: 'M' + v.map((p) => cam.proj(p).map((n) => n.toFixed(1)).join(',')).join('L') + 'Z', depth: v.reduce((s, p) => s + p[2], 0) / v.length }
  }

  function quad(face) {
    const x0 = face.cx * S - S / 2, x1 = x0 + S, z0 = face.cz * S - S / 2, z1 = z0 + S
    if (face.kind === 'floor') return [[x0, 0, z0], [x1, 0, z0], [x1, 0, z1], [x0, 0, z1]]
    if (face.kind === 'ceil') return [[x0, H, z0], [x0, H, z1], [x1, H, z1], [x1, H, z0]]
    switch (face.side) {
      case 'N': return [[x0, 0, z1], [x1, 0, z1], [x1, H, z1], [x0, H, z1]]
      case 'S': return [[x1, 0, z0], [x0, 0, z0], [x0, H, z0], [x1, H, z0]]
      case 'W': return [[x0, 0, z0], [x0, 0, z1], [x0, H, z1], [x0, H, z0]]
      case 'E': return [[x1, 0, z1], [x1, 0, z0], [x1, H, z0], [x1, H, z1]]
    }
  }
  function doorQuad(face) {
    const q = quad(face)
    const bl = lerp3(q[0], q[1], 0.3), br = lerp3(q[0], q[1], 0.7)
    return [bl, br, [br[0], 2.3, br[2]], [bl[0], 2.3, bl[2]]]
  }

  // ── palette (placeholder look; not an art direction, X16) ───────────────
  const PAL = { bg: '#050607', wall: [104, 94, 80], floor: [58, 53, 46], ceil: [30, 29, 28], door: [110, 70, 38], range: 16 }
  const shade = (rgb, b) => { const k = Math.max(0.04, Math.min(1.1, b)); return `rgb(${Math.min(255, Math.round(rgb[0] * k))},${Math.min(255, Math.round(rgb[1] * k))},${Math.min(255, Math.round(rgb[2] * k))})` }
  const mix = (rgb, to, t) => rgb.map((c, i) => c + (to[i] - c) * t)

  // ── figures ─────────────────────────────────────────────────────────────
  // Presentation-only formation inside the team's square: front line nearer
  // the opponents. The rules know two lines and nothing about positions.
  const FORMATION = [[-1.0, 1.05], [0, 1.15], [1.0, 1.05], [-1.0, -0.2], [0, -0.1], [1.0, -0.2]]
  const GROUP_COLOR = ['#e0a54a', '#c77dd8', '#6cc4d8', '#d86c6c']

  function sprite(cam, x, z, h) {
    const base = cam.view([x, 0, z]), top = cam.view([x, h, z])
    if (base[2] < 0.3 || top[2] < 0.3) return null
    const [bx, by] = cam.proj(base), [, ty] = cam.proj(top)
    if (by - ty <= 0) return null
    return { bx, by, ty, hh: by - ty, depth: base[2] }
  }

  // Party figure states. 'ok', 'poison' and 'dead' are the original ones.
  // The difficult-case validation adds (all provisional, X26 addendum):
  //   paralyzed  solid and recognisable, flat and rigid, a pale rim
  //   asleep     slumped low, with a z; never confused with death
  //   dying      m.k 0..1: stagger, collapse, then the body gives way to...
  //   ghost      a translucent figure: the dead member is still with the team
  //   ashes      a low grey mound with a few faint motes
  // A sprite standing at height `alt` (a flier). `lift` is the pixel gap
  // between its base and its shadow on the floor.
  function spriteAt(cam, x, z, h, alt) {
    const floor = cam.view([x, 0, z]), base = cam.view([x, alt, z]), top = cam.view([x, alt + h, z])
    if (base[2] < 0.3 || top[2] < 0.3) return null
    const [bx, by] = cam.proj(base), [, ty] = cam.proj(top), [, fy] = cam.proj(floor)
    return { bx, by, ty, hh: by - ty, depth: base[2], lift: fy - by }
  }

  const clamp01 = (k) => Math.max(0, Math.min(1, k))
  const smooth = (k) => { k = clamp01(k); return k * k * (3 - 2 * k) }
  function personBody(bx, by, h, w, col, sel, flat) {
    const st = sel ? '#f2c14e' : '#0b0b0b', sw = sel ? 2 : 1
    const bodyY = by - h * 0.8
    let g = `<rect x="${bx - w / 2}" y="${bodyY}" width="${w}" height="${h * 0.8}" rx="${w * 0.34}" fill="${col}" stroke="${st}" stroke-width="${sw}"/>`
    if (!flat) g += `<rect x="${bx - w / 2}" y="${bodyY}" width="${w}" height="${h * 0.8}" rx="${w * 0.34}" fill="url(#cs-shade)"/>`
    g += `<circle cx="${bx}" cy="${by - h * 0.9}" r="${h * 0.11}" fill="${col}" stroke="${st}" stroke-width="${sw}"/>`
    return g
  }
  const numText = (x, y, h, n, op) => `<text x="${x}" y="${y}" font-size="${Math.max(9, Math.min(17, h * 0.15))}" text-anchor="middle" fill="#fff" ${op ? `fill-opacity="${op}"` : ''} font-family="ui-monospace,Menlo,monospace" font-weight="700" style="paint-order:stroke" stroke="rgba(0,0,0,0.6)" stroke-width="2.5">${n}</text>`
  const desat = (hex, t, to) => { const c = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)); const d = to || [150, 160, 172]; return `rgb(${c.map((v, i) => Math.round(v + (d[i] - v) * t)).join(',')})` }
  function ghostSvg(bx, by, h, w, slot, op) {
    const bodyY = by - h * 0.8
    return `<g opacity="${op}"><rect x="${bx - w / 2}" y="${bodyY}" width="${w}" height="${h * 0.8}" rx="${w * 0.34}" fill="rgba(205,216,230,0.30)" stroke="rgba(230,238,248,0.75)" stroke-width="1.3" stroke-dasharray="4 3"/><circle cx="${bx}" cy="${by - h * 0.9}" r="${h * 0.11}" fill="rgba(205,216,230,0.30)" stroke="rgba(230,238,248,0.75)" stroke-width="1.3" stroke-dasharray="4 3"/>${numText(bx, by - h * 0.42, h, slot, 0.7)}</g>`
  }
  function personSvg(s, m) {
    const h = s.hh, w = h * 0.34, { bx, by } = s
    const st = m.state || 'ok'
    const col = st === 'dead' ? '#3a3a3a' : m.color
    const shadow = (k) => `<ellipse cx="${bx}" cy="${by}" rx="${w * 0.85}" ry="${w * 0.24}" fill="rgba(0,0,0,${0.5 * (k == null ? 1 : k)})"/>`
    let g = st === 'ghost' || st === 'ashes' ? '' : shadow()
    if (m.selected) g += `<ellipse class="selring" cx="${bx}" cy="${by}" rx="${w * 1.05}" ry="${w * 0.33}" fill="none" stroke="#f2c14e" stroke-width="2.5"/>`
    else if (m.candidate) g += `<ellipse class="candring" cx="${bx}" cy="${by}" rx="${w * 1.05}" ry="${w * 0.33}" fill="none" stroke="#8fd18f" stroke-width="2" stroke-dasharray="5 3"/>`
    // The refined prototype's target marker: the same white ring for a member
    // and (via groupRing 'mark') for a monster group.
    if (m.target) g += `<ellipse cx="${bx}" cy="${by}" rx="${w * 1.2}" ry="${w * 0.38}" fill="rgba(255,255,255,0.10)" stroke="#ffffff" stroke-width="2.5"/>`
    if (st === 'paralyzed') {
      // Rigid: no shading, a cold tint and a thin pale rim. Still upright.
      g += personBody(bx, by, h, w, desat(m.color, 0.45), m.selected, true)
      const bodyY = by - h * 0.8
      g += `<rect x="${bx - w / 2 - 2}" y="${bodyY - 2}" width="${w + 4}" height="${h * 0.8 + 4}" rx="${w * 0.36}" fill="none" stroke="#cfe3ff" stroke-width="1.4" opacity="0.85"/>`
      g += numText(bx, by - h * 0.42, h, m.slot)
    } else if (st === 'asleep') {
      // Slumped: half height, head dropped to one side.
      const hh = h * 0.5, bodyY = by - hh
      g += `<rect x="${bx - w * 0.6}" y="${bodyY}" width="${w * 1.2}" height="${hh}" rx="${w * 0.45}" fill="${desat(m.color, 0.2, [60, 62, 70])}" stroke="#0b0b0b"/>`
      g += `<circle cx="${bx + w * 0.5}" cy="${bodyY + h * 0.04}" r="${h * 0.1}" fill="${desat(m.color, 0.2, [60, 62, 70])}" stroke="#0b0b0b"/>`
      g += `<text x="${bx + w * 0.9}" y="${bodyY - h * 0.08}" font-size="${Math.max(10, h * 0.16)}" fill="#c9c2ff" font-weight="700" font-family="ui-monospace,Menlo,monospace">z</text>`
      g += numText(bx - w * 0.1, by - hh * 0.3, h * 0.8, m.slot)
    } else if (st === 'dying') {
      // Brief and theatrical: a stagger back (0-0.25), a fall (0.25-0.65),
      // then the fallen body fades as the translucent figure appears.
      const k = clamp01(m.k == null ? 1 : m.k)
      const lean = k < 0.25 ? -14 * smooth(k / 0.25) : -14 - 72 * smooth((k - 0.25) / 0.4)
      const bodyOp = k < 0.65 ? 1 : 1 - smooth((k - 0.65) / 0.35)
      const sink = k < 0.25 ? 0 : h * 0.06 * smooth((k - 0.25) / 0.4)
      g = shadow(bodyOp) + g.replace(shadow(), '')
      if (bodyOp > 0.01) g += `<g opacity="${bodyOp.toFixed(3)}" transform="translate(0 ${sink.toFixed(1)}) rotate(${lean.toFixed(1)} ${bx} ${by})">${personBody(bx, by, h, w, col, false)}</g>`
      if (k > 0.6) g += ghostSvg(bx, by, h, w, m.slot, smooth((k - 0.6) / 0.4))
    } else if (st === 'ghost') {
      g += ghostSvg(bx, by, h, w, m.slot, 1)
    } else if (st === 'ashes') {
      g += `<ellipse cx="${bx}" cy="${by - h * 0.03}" rx="${w * 0.75}" ry="${h * 0.07}" fill="#57524c" stroke="#2a2724"/>`
      g += `<ellipse cx="${bx}" cy="${by - h * 0.07}" rx="${w * 0.45}" ry="${h * 0.05}" fill="#6c665e"/>`
      for (const [dx, dy, r] of [[-0.3, 0.22, 1.6], [0.15, 0.34, 1.2], [0.35, 0.18, 1.4], [-0.05, 0.46, 1]]) g += `<circle cx="${bx + dx * w}" cy="${by - dy * h}" r="${r}" fill="#d8cfc0" opacity="0.45"/>`
      g += numText(bx, by - h * 0.18, h * 0.8, m.slot, 0.6)
    } else {
      g += personBody(bx, by, h, w, col, m.selected)
      g += numText(bx, by - h * 0.42, h, m.slot)
      if (st === 'poison') g += `<circle cx="${bx + w * 0.55}" cy="${by - h * 1.02}" r="${Math.max(3, h * 0.05)}" fill="#7bd36a" stroke="#000"/>`
    }
    if (m.poison && st !== 'poison' && st !== 'ghost' && st !== 'ashes' && st !== 'dying') g += `<circle cx="${bx + w * 0.55}" cy="${by - h * 1.02}" r="${Math.max(3, h * 0.05)}" fill="#7bd36a" stroke="#000"/>`
    if (m.dim) g = `<g opacity="${typeof m.dim === 'number' ? m.dim : 0.45}">${g}</g>`
    return g
  }

  // Monster appearance ("look") is presentation: the shape the team can
  // already see, the same for an identified or unidentified group. Death
  // styles are chosen from the look and from the revealed result only.
  //   blob (original), humanoid, skeleton, slime, flying, large
  function monsterBody(look, bx, by, h, w, col, asleep, eyes) {
    const eye = (y, sp) => eyes ? `<circle cx="${bx - w * sp}" cy="${y}" r="${Math.max(1.2, h * 0.035)}" fill="#f0dc72"/><circle cx="${bx + w * sp}" cy="${y}" r="${Math.max(1.2, h * 0.035)}" fill="#f0dc72"/>` : ''
    switch (look) {
      case 'humanoid': case 'large': {
        const bw = look === 'large' ? w * 0.95 : w * 0.62
        return `<rect x="${bx - bw / 2}" y="${by - h * 0.78}" width="${bw}" height="${h * 0.78}" rx="${bw * 0.38}" fill="${col}" stroke="#0b0b0b" stroke-width="1.2"/><rect x="${bx - bw / 2}" y="${by - h * 0.78}" width="${bw}" height="${h * 0.78}" rx="${bw * 0.38}" fill="url(#cs-shade)"/><circle cx="${bx}" cy="${by - h * 0.86}" r="${h * (look === 'large' ? 0.12 : 0.14)}" fill="${col}" stroke="#0b0b0b" stroke-width="1.2"/>${eye(by - h * 0.87, 0.12)}`
      }
      case 'skeleton': {
        const bone = asleep ? '#7d7a72' : '#d9d2bf'
        let g = `<rect x="${bx - w * 0.06}" y="${by - h * 0.78}" width="${w * 0.12}" height="${h * 0.5}" fill="${bone}" stroke="#2b2925"/>`
        for (const y of [0.7, 0.62, 0.54]) g += `<path d="M${bx - w * 0.3},${by - h * y}Q${bx},${by - h * (y + 0.05)} ${bx + w * 0.3},${by - h * y}" fill="none" stroke="${bone}" stroke-width="${Math.max(1.5, h * 0.025)}"/>`
        g += `<path d="M${bx - w * 0.22},${by}L${bx - w * 0.08},${by - h * 0.3}L${bx + w * 0.08},${by - h * 0.3}L${bx + w * 0.22},${by}" fill="none" stroke="${bone}" stroke-width="${Math.max(1.5, h * 0.03)}"/>`
        g += `<circle cx="${bx}" cy="${by - h * 0.87}" r="${h * 0.1}" fill="${bone}" stroke="#2b2925"/>`
        if (eyes) g += `<circle cx="${bx - h * 0.035}" cy="${by - h * 0.88}" r="${h * 0.022}" fill="#200"/><circle cx="${bx + h * 0.035}" cy="${by - h * 0.88}" r="${h * 0.022}" fill="#200"/>`
        return g
      }
      case 'slime':
        return `<path d="M${bx - w * 0.8},${by}Q${bx - w * 0.75},${by - h * 0.95} ${bx},${by - h * 0.95}Q${bx + w * 0.75},${by - h * 0.95} ${bx + w * 0.8},${by}Z" fill="${col}" fill-opacity="0.9" stroke="#0b0b0b"/><ellipse cx="${bx - w * 0.25}" cy="${by - h * 0.62}" rx="${w * 0.14}" ry="${h * 0.1}" fill="#fff" opacity="0.25"/>${eye(by - h * 0.55, 0.2)}`
      case 'flying': {
        const wing = asleep ? `M${bx - w * 0.15},${by - h * 0.5}L${bx - w * 0.45},${by - h * 0.2}L${bx - w * 0.1},${by - h * 0.25}ZM${bx + w * 0.15},${by - h * 0.5}L${bx + w * 0.45},${by - h * 0.2}L${bx + w * 0.1},${by - h * 0.25}Z`
                            : `M${bx - w * 0.2},${by - h * 0.55}L${bx - w * 1.15},${by - h * 0.95}L${bx - w * 0.75},${by - h * 0.35}ZM${bx + w * 0.2},${by - h * 0.55}L${bx + w * 1.15},${by - h * 0.95}L${bx + w * 0.75},${by - h * 0.35}Z`
        return `<path d="${wing}" fill="${col}" fill-opacity="0.85" stroke="#0b0b0b"/><ellipse cx="${bx}" cy="${by - h * 0.45}" rx="${w * 0.32}" ry="${h * 0.38}" fill="${col}" stroke="#0b0b0b"/>${eye(by - h * 0.6, 0.1)}`
      }
      default:
        return `<ellipse cx="${bx}" cy="${by - h * 0.42}" rx="${w * 0.72}" ry="${h * 0.44}" fill="${col}" stroke="#0b0b0b" stroke-width="1.2"/><ellipse cx="${bx}" cy="${by - h * 0.42}" rx="${w * 0.72}" ry="${h * 0.44}" fill="url(#cs-shade)"/>${eye(by - h * 0.62, 0.22)}`
    }
  }
  // Death styles, m.k 0..1 over the style's own duration (DEATH_MS). The last
  // frame is the remains, which stay until the round's display update.
  const DEATH_MS = { blob: 800, humanoid: 900, skeleton: 1000, slime: 1100, flying: 900, large: 1200 }
  function monsterDeath(look, s, m, col) {
    const h = s.hh, w = h * 0.5, { bx, by } = s
    const k = clamp01(m.k == null ? 1 : m.k)
    const dark = desat(col.length === 7 ? col : '#777777', 0.55, [40, 38, 36])
    switch (look) {
      case 'skeleton': {
        // Falls apart: skull, ribs and legs drop and scatter.
        const f = smooth(k), bone = '#cfc7b2'
        const part = (dx, dy, rot, inner) => `<g transform="translate(${(dx * f).toFixed(1)} ${(dy * f).toFixed(1)}) rotate(${(rot * f).toFixed(0)} ${bx} ${by - h * 0.5})">${inner}</g>`
        let g = part(-w * 0.5, h * 0.8, -70, `<circle cx="${bx}" cy="${by - h * 0.87}" r="${h * 0.1}" fill="${bone}" stroke="#2b2925"/>`)
        g += part(w * 0.35, h * 0.55, 80, `<rect x="${bx - w * 0.06}" y="${by - h * 0.78}" width="${w * 0.12}" height="${h * 0.5}" fill="${bone}" stroke="#2b2925"/>`)
        for (const [i, y] of [0.7, 0.62, 0.54].entries()) g += part((i - 1) * w * 0.45, h * (y - 0.04), (i - 1) * 50, `<path d="M${bx - w * 0.3},${by - h * y}Q${bx},${by - h * (y + 0.05)} ${bx + w * 0.3},${by - h * y}" fill="none" stroke="${bone}" stroke-width="${Math.max(1.5, h * 0.025)}"/>`)
        g += part(w * 0.1, h * 0.1, 30, `<path d="M${bx - w * 0.22},${by}L${bx - w * 0.08},${by - h * 0.3}L${bx + w * 0.08},${by - h * 0.3}L${bx + w * 0.22},${by}" fill="none" stroke="${bone}" stroke-width="${Math.max(1.5, h * 0.03)}"/>`)
        return g
      }
      case 'slime': {
        // Dissolves: flattens, spreads and darkens into a puddle.
        const f = smooth(k), sy = 1 - 0.88 * f, sx = 1 + 0.6 * f
        return `<g transform="translate(${bx} ${by}) scale(${sx.toFixed(3)} ${sy.toFixed(3)}) translate(${-bx} ${-by})" opacity="${(1 - 0.35 * f).toFixed(2)}">${monsterBody('slime', bx, by, h, w, k > 0.5 ? dark : col, false, k < 0.3)}</g>${k > 0.4 ? `<ellipse cx="${bx}" cy="${by}" rx="${w * 1.2 * f}" ry="${w * 0.2 * f}" fill="${dark}" opacity="0.6"/>` : ''}`
      }
      case 'flying': {
        // Tumbles: drops from its height, turning over, to the floor.
        const f = k * k, drop = (m.alt || 0) * f, spin = 400 * f
        const ss = { bx, by: by + drop, hh: h }
        return `<g transform="rotate(${spin.toFixed(0)} ${bx} ${ss.by - h * 0.45})">${monsterBody('flying', bx, ss.by, h, w, k > 0.8 ? dark : col, k > 0.8, false)}</g>`
      }
      case 'large': {
        // A more substantial collapse: sway, drop to the knees, fall forward,
        // and a puff of dust.
        const sway = k < 0.3 ? Math.sin((k / 0.3) * Math.PI * 2) * 6 : 0
        const kneel = k < 0.3 ? 1 : k < 0.55 ? 1 - 0.3 * smooth((k - 0.3) / 0.25) : 0.7
        const fall = k < 0.55 ? 0 : 82 * smooth((k - 0.55) / 0.35)
        let g = `<g transform="rotate(${(sway + fall).toFixed(1)} ${bx} ${by}) translate(${bx} ${by}) scale(1 ${kneel.toFixed(3)}) translate(${-bx} ${-by})">${monsterBody('large', bx, by, h, w, k > 0.7 ? dark : col, false, k < 0.5)}</g>`
        if (k > 0.75) { const d = smooth((k - 0.75) / 0.25); g += `<ellipse cx="${bx + w * 0.9}" cy="${by}" rx="${w * (0.6 + 1.2 * d)}" ry="${w * (0.15 + 0.25 * d)}" fill="#8a7f6e" opacity="${(0.5 * (1 - d) + 0.1).toFixed(2)}"/>` }
        return g
      }
      case 'humanoid': {
        // Staggers back, then collapses.
        const lean = k < 0.3 ? 10 * smooth(k / 0.3) : 10 - 95 * smooth((k - 0.3) / 0.5)
        return `<g transform="rotate(${lean.toFixed(1)} ${bx} ${by})">${monsterBody('humanoid', bx, by, h, w, k > 0.6 ? dark : col, false, k < 0.4)}</g>`
      }
      default:
        return `<g opacity="${(1 - 0.7 * k).toFixed(2)}">${monsterBody('blob', bx, by, h, w, col, false, false)}</g>`
    }
  }
  function monsterSvg(s, m, gi) {
    const h = s.hh, w = h * 0.5, { bx, by } = s
    const look = m.look || 'blob'
    const col = m.asleep ? '#4c4f58' : m.color
    // A flier is drawn at its height with its shadow on the floor; asleep,
    // it rests on the floor.
    const lift = m.liftPx || 0
    let g = `<ellipse cx="${bx}" cy="${by + lift}" rx="${w * 0.8}" ry="${w * 0.2}" fill="rgba(0,0,0,${m.state === 'dead' ? 0.25 : 0.55})"/>`
    if (m.state === 'dying') g += monsterDeath(look, s, m, m.color)
    else if (m.state === 'dead') g += `<g opacity="0.6">${monsterDeath(look, s, Object.assign({}, m, { k: 1 }), m.color)}</g>`
    else {
      g += monsterBody(look, bx, by, h, w, col, m.asleep, !m.asleep)
      if (m.asleep) g += `<text x="${bx + w * 0.4}" y="${by - h * 0.9}" font-size="${Math.max(10, h * 0.2)}" fill="#b9b2ff" font-weight="700" font-family="ui-monospace,Menlo,monospace">z</text>`
    }
    if (m.fading) g = `<g opacity="0.3">${g}</g>`
    if (typeof m.dim === 'number') g = `<g opacity="${m.dim}">${g}</g>`
    return g
  }

  // Ring on the floor around a monster group: identifies the group number
  // (rules: groups are numbered in order) without implying range or position.
  function groupRing(cam, members, gi, state) {
    if (!members.length) return null
    const cx = members.reduce((a, m) => a + m.x, 0) / members.length
    const cz = members.reduce((a, m) => a + m.z, 0) / members.length
    const rx = Math.max(0.9, (Math.max(...members.map((m) => m.x)) - Math.min(...members.map((m) => m.x))) / 2 + 0.6)
    const pts = []
    for (let i = 0; i < 28; i++) { const a = (i / 28) * Math.PI * 2; pts.push([cx + Math.cos(a) * rx, 0.02, cz + Math.sin(a) * 0.75]) }
    const p = polyPath(cam, pts)
    if (!p) return null
    const col = GROUP_COLOR[gi % 4]
    // Refined prototype: 'actor' (gold, the group acting now) and 'mark' (white,
    // the group targeted now) match the member rings, so one highlight
    // language covers both sides.
    if (state === 'actor' || state === 'mark') {
      const c2 = state === 'actor' ? '#f2c14e' : '#ffffff'
      return { depth: p.depth + 0.5, svg: `<path class="grp ${state}" d="${p.d}" fill="${state === 'mark' ? 'rgba(255,255,255,0.10)' : col + '22'}" stroke="${c2}" stroke-width="2.5"/>` }
    }
    const cls = state === 'target' ? 'grp target' : state === 'chosen' ? 'grp chosen' : 'grp'
    return { depth: p.depth + 0.5, svg: `<path class="${cls}" d="${p.d}" fill="${state ? col + '22' : 'none'}" stroke="${col}" stroke-width="${state ? 2.5 : 1.3}" ${state ? '' : 'stroke-dasharray="6 5" opacity="0.7"'}/>` }
  }

  // ── effects (screen space, drawn over the scene) ────────────────────────
  function effectSvg(e, A) {
    const a = A[e.from], b = A[e.to]
    if (!b) return ''
    const [tx, ty] = b
    switch (e.type) {
      case 'hit': return `<g class="fx"><path d="M${tx - 20},${ty + 6}L${tx + 20},${ty + 34}M${tx + 18},${ty + 2}L${tx - 14},${ty + 30}" stroke="#ff5a3c" stroke-width="4" stroke-linecap="round"/><text x="${tx + 22}" y="${ty - 4}" font-size="22" fill="#ff7a5c" font-weight="800" font-family="ui-monospace,Menlo,monospace" style="paint-order:stroke" stroke="#000" stroke-width="3">${e.text || ''}</text></g>`
      case 'miss': return `<g class="fx"><text x="${tx + 16}" y="${ty - 2}" font-size="17" fill="#cfcfcf" font-weight="700" font-family="ui-monospace,Menlo,monospace" style="paint-order:stroke" stroke="#000" stroke-width="3">MISS</text></g>`
      case 'halito': return `<g class="fx">${a ? `<path d="M${a[0]},${a[1]}Q${(a[0] + tx) / 2},${Math.min(a[1], ty) - 70} ${tx},${ty + 20}" fill="none" stroke="#ffb347" stroke-width="3" stroke-dasharray="2 6" stroke-linecap="round"/>` : ''}<circle cx="${tx}" cy="${ty + 22}" r="34" fill="url(#cs-fire)"/><text x="${tx + 30}" y="${ty - 6}" font-size="22" fill="#ffb347" font-weight="800" font-family="ui-monospace,Menlo,monospace" style="paint-order:stroke" stroke="#000" stroke-width="3">${e.text || ''}</text></g>`
      case 'sleep': return `<g class="fx"><ellipse cx="${tx}" cy="${ty + 24}" rx="120" ry="44" fill="url(#cs-mist)"/><text x="${tx + 40}" y="${ty - 10}" font-size="22" fill="#c9c2ff" font-weight="800" font-family="ui-monospace,Menlo,monospace">z z</text></g>`
      case 'dispel': return `<g class="fx">${a ? `<path d="M${a[0]},${a[1]}L${tx},${ty + 20}" stroke="#f4f4ff" stroke-width="2" opacity="0.8"/>` : ''}<circle cx="${tx}" cy="${ty + 22}" r="38" fill="url(#cs-holy)"/></g>`
      case 'heal': return `<g class="fx"><circle cx="${tx}" cy="${ty + 30}" r="30" fill="url(#cs-heal)"/><text x="${tx + 18}" y="${ty - 4}" font-size="20" fill="#8fe38f" font-weight="800" font-family="ui-monospace,Menlo,monospace" style="paint-order:stroke" stroke="#000" stroke-width="3">${e.text || ''}</text></g>`
      case 'parry': return `<g class="fx"><path d="M${tx - 22},${ty + 50}Q${tx},${ty - 10} ${tx + 22},${ty + 50}" fill="none" stroke="#7fb8ff" stroke-width="4" stroke-linecap="round"/></g>`
      case 'cast': return `<g class="fx"><circle cx="${tx}" cy="${ty + 34}" r="30" fill="url(#cs-holy)" opacity="0.55"/><circle cx="${tx}" cy="${ty + 34}" r="24" fill="none" stroke="#ffe7a8" stroke-width="1.5" stroke-dasharray="2 4"/></g>`
      case 'focus': return `<g class="fx"><circle cx="${tx}" cy="${ty + 30}" r="44" fill="none" stroke="#f2c14e" stroke-width="1.5" stroke-dasharray="3 5"/></g>`
      default: return ''
    }
  }

  // ── render ──────────────────────────────────────────────────────────────
  // shot: { pos, target, fov, policy, teamZ, lightDis, showLeaks,
  //         party: [{slot,color,state,selected,candidate,dim,at}],
  //         groups: [{members:[{x,z,h,color,asleep,fading}], ring}],
  //         chest: {x,z}, effects: [{type,from,to,text}] }
  function render(shot, W, Hpx) {
    W = W || 960; Hpx = Hpx || 540
    const cam = camera(shot.pos, shot.target, shot.fov || 70, W, Hpx)
    const teamZ = shot.teamZ || 0
    const light = [0, 1.7, teamZ * S + 0.6]
    const items = []
    const set = faceSet(shot.policy || 'strict', teamZ, shot.lightDis || 5, shot.loc)
    let leaks = 0

    if (set.shell) {
      // Procedural shell: generated from nothing but the camera and the team's
      // square, so it cannot carry level data. A floor that fades into fog,
      // and a fog curtain wherever the team square has no visible edge.
      const z0 = teamZ * S - S / 2
      const fl = polyPath(cam, [[-2, 0, z0 - 9], [2, 0, z0 - 9], [2, 0, z0], [-2, 0, z0]])
      if (fl) items.push({ depth: fl.depth + 1000, svg: `<path d="${fl.d}" fill="url(#cs-fogfloor)"/>` })
      const cur = polyPath(cam, [[2, 0, z0 - 0.02], [-2, 0, z0 - 0.02], [-2, H, z0 - 0.02], [2, H, z0 - 0.02]])
      if (cur) items.push({ depth: cur.depth + 0.2, svg: `<path d="${cur.d}" fill="url(#cs-fogwall)"/>` })
    }

    for (const { f: face, memory, leak } of set.faces) {
      const q = quad(face)
      const p = polyPath(cam, q)
      if (!p) continue
      const c = q.reduce((a, b) => [a[0] + b[0] / 4, a[1] + b[1] / 4, a[2] + b[2] / 4], [0, 0, 0])
      const dist = Math.hypot(...sub(c, light))
      let bright = 1.12 - dist / PAL.range
      let base = face.kind === 'floor' ? PAL.floor : face.kind === 'ceil' ? PAL.ceil : PAL.wall
      if (memory) { base = mix(base, [70, 78, 92], 0.45); bright *= 0.75 }
      const sideK = face.kind === 'wall' && (face.side === 'W' || face.side === 'E') ? 0.8 : 1
      let svg = `<path d="${p.d}" fill="${shade(base, bright * sideK)}" stroke="rgba(0,0,0,0.55)" stroke-width="1"/>`
      if (face.kind === 'floor') {
        // Square boundary: a depth cue only.
        svg = `<path d="${p.d}" fill="${shade(base, bright)}" stroke="rgba(255,240,210,0.09)" stroke-width="1.2"/>`
      }
      if (face.kind === 'wall') {
        for (const yy of [0.75, 1.5, 2.25]) {
          const m = polyPath(cam, [[q[0][0], yy, q[0][2]], [q[1][0], yy, q[1][2]], [q[1][0], yy + 0.01, q[1][2]], [q[0][0], yy + 0.01, q[0][2]]])
          if (m) svg += `<path d="${m.d}" fill="none" stroke="rgba(0,0,0,0.32)" stroke-width="1"/>`
        }
        if (face.type === 'door') {
          const dq = polyPath(cam, doorQuad(face))
          if (dq) svg += `<path d="${dq.d}" fill="${shade(memory ? mix(PAL.door, [70, 78, 92], 0.45) : PAL.door, bright)}" stroke="#1a0f06" stroke-width="2"/>`
        }
      }
      if (leak && shot.showLeaks) { svg += `<path d="${p.d}" fill="rgba(255,60,140,0.18)" stroke="#ff3c8c" stroke-width="2.5" stroke-dasharray="7 4"/>`; leaks++ }
      else if (leak) leaks++
      // Architecture is drawn before every figure: figures stand in the
      // team's column and every camera is inside it, so no wall can be in
      // front of a figure. (A painter's sort on large wall polygons could
      // otherwise draw a near wall over a far figure.)
      items.push({ depth: p.depth + (face.kind === 'wall' ? 500 : 1000), svg })
    }

    // Refined prototype: a faint floor line between the front line (1-3) and
    // the back line (4-6). Presentation only; the rules know two lines and
    // nothing about distance.
    if (shot.rowLine) {
      const zl = teamZ * S + 0.45
      const p = polyPath(cam, [[-1.7, 0.015, zl - 0.03], [1.7, 0.015, zl - 0.03], [1.7, 0.015, zl + 0.03], [-1.7, 0.015, zl + 0.03]])
      if (p) items.push({ depth: p.depth + 998, svg: `<path d="${p.d}" fill="rgba(242,193,78,${shot.rowLine})"/>` })
    }

    // Light pool under the team: atmosphere only.
    {
      const pts = []
      for (let i = 0; i < 32; i++) { const a = (i / 32) * Math.PI * 2; pts.push([Math.cos(a) * 1.9, 0.01, teamZ * S + 0.5 + Math.sin(a) * 1.9]) }
      const p = polyPath(cam, pts)
      if (p) items.push({ depth: p.depth + 999, svg: `<path d="${p.d}" fill="url(#cs-pool)"/>` })
    }

    if (shot.chest) {
      const c = shot.chest
      const faces = [
        { pts: [[c.x - 0.5, 0.6, c.z - 0.33], [c.x - 0.5, 0.6, c.z + 0.33], [c.x + 0.5, 0.6, c.z + 0.33], [c.x + 0.5, 0.6, c.z - 0.33]], b: 1 },
        { pts: [[c.x - 0.5, 0, c.z - 0.33], [c.x + 0.5, 0, c.z - 0.33], [c.x + 0.5, 0.6, c.z - 0.33], [c.x - 0.5, 0.6, c.z - 0.33]], b: 0.8 },
        { pts: [[c.x + 0.5, 0, c.z - 0.33], [c.x + 0.5, 0, c.z + 0.33], [c.x + 0.5, 0.6, c.z + 0.33], [c.x + 0.5, 0.6, c.z - 0.33]], b: 0.62 },
        { pts: [[c.x - 0.5, 0, c.z + 0.33], [c.x - 0.5, 0, c.z - 0.33], [c.x - 0.5, 0.6, c.z - 0.33], [c.x - 0.5, 0.6, c.z + 0.33]], b: 0.62 },
      ]
      for (const f of faces) { const p = polyPath(cam, f.pts); if (p) items.push({ depth: p.depth, svg: `<path d="${p.d}" fill="${shade([128, 88, 44], f.b)}" stroke="${shot.chest.selected ? '#f2c14e' : '#2a1806'}" stroke-width="${shot.chest.selected ? 2.5 : 1.5}"/>` }) }
      const band = polyPath(cam, [[c.x - 0.51, 0.42, c.z - 0.34], [c.x + 0.51, 0.42, c.z - 0.34], [c.x + 0.51, 0.47, c.z - 0.34], [c.x - 0.51, 0.47, c.z - 0.34]])
      if (band) items.push({ depth: band.depth - 0.01, svg: `<path d="${band.d}" fill="#b8a060" stroke="#3a2a10"/>` })
    }

    const anchors = {}
    const toPct = (x, y) => [x, y, (100 * x) / W, (100 * y) / Hpx]
    for (const m of shot.party || []) {
      const [x, z0] = m.at || FORMATION[m.slot - 1]
      const s = sprite(cam, x, z0 + teamZ * S, m.h || 1.38)
      if (!s) continue
      items.push({ depth: s.depth, svg: personSvg(s, m) })
      anchors['p' + m.slot] = toPct(s.bx, s.ty)
      anchors['pf' + m.slot] = toPct(s.bx, s.by)
    }
    for (const [gi, g] of (shot.groups || []).entries()) {
      const ms = g.members || []
      const ring = groupRing(cam, ms, gi, g.ring)
      if (ring) items.push(ring)
      let top = null, sx = 0, n = 0
      for (const m of ms) {
        // Fliers hover at m.alt; asleep or dead they are on the floor. A dying
        // flier starts at its height and its death style drops it `lift` px.
        const dying = m.state === 'dying', grounded = m.asleep || m.state === 'dead'
        const alt = m.alt && !grounded ? m.alt : 0
        const s = alt ? spriteAt(cam, m.x, m.z, m.h || 1.35, alt) : sprite(cam, m.x, m.z, m.h || 1.35)
        if (!s) continue
        const mm = alt ? Object.assign({}, m, { liftPx: s.lift, alt: dying ? s.lift : 0 }) : Object.assign({}, m, { alt: 0 })
        items.push({ depth: s.depth, svg: monsterSvg(s, mm, gi) })
        // Remains do not hold the group label up, unless nothing else is left.
        if (m.state !== 'dead' || !ms.some((o) => o.state !== 'dead')) { top = top === null ? s.ty : Math.min(top, s.ty); sx += s.bx; n++ }
      }
      if (n) anchors['g' + gi] = toPct(sx / n, top)
    }
    if (shot.chest) { const s = sprite(cam, shot.chest.x, shot.chest.z, 0.6); if (s) anchors.chest = toPct(s.bx, s.ty) }

    items.sort((a, b) => b.depth - a.depth)
    const fx = (shot.effects || []).map((e) => effectSvg(e, anchors)).join('')
    const defs = `<defs>
      <linearGradient id="cs-shade" x1="0" x2="1"><stop offset="0" stop-color="#fff" stop-opacity="0.16"/><stop offset="0.55" stop-color="#000" stop-opacity="0"/><stop offset="1" stop-color="#000" stop-opacity="0.42"/></linearGradient>
      <radialGradient id="cs-pool"><stop offset="0" stop-color="#ffcf8a" stop-opacity="0.16"/><stop offset="1" stop-color="#ffcf8a" stop-opacity="0"/></radialGradient>
      <radialGradient id="cs-fire"><stop offset="0" stop-color="#fff2b0" stop-opacity="0.95"/><stop offset="0.45" stop-color="#ff8a2a" stop-opacity="0.75"/><stop offset="1" stop-color="#ff3a00" stop-opacity="0"/></radialGradient>
      <radialGradient id="cs-mist"><stop offset="0" stop-color="#b9a8ff" stop-opacity="0.55"/><stop offset="1" stop-color="#6f5cff" stop-opacity="0"/></radialGradient>
      <radialGradient id="cs-holy"><stop offset="0" stop-color="#ffffff" stop-opacity="0.9"/><stop offset="1" stop-color="#cfe0ff" stop-opacity="0"/></radialGradient>
      <radialGradient id="cs-heal"><stop offset="0" stop-color="#b8ffb0" stop-opacity="0.75"/><stop offset="1" stop-color="#3ad13a" stop-opacity="0"/></radialGradient>
      <linearGradient id="cs-fogfloor" x1="0" y1="1" x2="0" y2="0"><stop offset="0" stop-color="#3a3630" stop-opacity="0.85"/><stop offset="1" stop-color="#050607" stop-opacity="0"/></linearGradient>
      <linearGradient id="cs-fogwall" x1="0" y1="1" x2="0" y2="0"><stop offset="0" stop-color="#1c1b19" stop-opacity="0.9"/><stop offset="1" stop-color="#050607" stop-opacity="0.35"/></linearGradient>
    </defs>`
    return {
      svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${W} ${Hpx}" preserveAspectRatio="xMidYMid slice" role="img" aria-label="Dungeon scene">${defs}<rect width="${W}" height="${Hpx}" fill="${PAL.bg}"/>${items.map((i) => i.svg).join('')}${fx}</svg>`,
      anchors,
      leaks,
    }
  }

  const api = { render, visibleFrom, faceSet, FORMATION, GROUP_COLOR, MAP, LOCATIONS, DEATH_MS, layout, lerp3 }
  if (typeof module !== 'undefined' && module.exports) module.exports = api
  else root.CombatScene = api
})(this)
