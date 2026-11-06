// Istary M2 design exploration: a throwaway projection renderer for mockups.
//
// This is NOT production code and is not part of the application. It draws
// one fixed dungeon location (a short corridor) from different cameras so the
// storyboard frames share exactly the same geometry. Plain JavaScript, no
// dependencies; it outputs SVG markup strings.
//
// World axes: x = east (right when facing north), y = up, z = north.
// One maze square is 4 units wide; walls are 3 units high. The party stands
// in square (0,0) facing north.

(function (root) {
  'use strict'

  const S = 4 // square size
  const H = 3 // wall height

  // The location. Edges not listed are walls. Only the edges the Classic
  // view rule can reach matter (see visibleFaces).
  //   (0,0) party square        (0,1) door on the left, opening on the right
  //   (1,1) a dead-end niche    (0,2) corridor     (0,3) dead end ahead
  const MAP = {
    '0,0': { N: 'open', S: 'open', W: 'wall', E: 'wall' },
    '0,1': { N: 'open', S: 'open', W: 'door', E: 'open' },
    '1,1': { N: 'wall', S: 'wall', W: 'open', E: 'wall' },
    '0,2': { N: 'open', S: 'open', W: 'wall', E: 'wall' },
    '0,3': { N: 'wall', S: 'open', W: 'wall', E: 'wall' },
  }

  function edge(cx, cz, side) {
    const c = MAP[cx + ',' + cz]
    return c ? c[side] : 'wall'
  }

  // visibleFaces mirrors the shape of the Classic DRAWMAZE walk
  // (render/dungeon.go): for each depth, the left and right edges of the
  // square ahead, or, where a side is open, the front edge of the side
  // square; then the front edge, stopping at the first one that is not open.
  // lightDis is 2 without light, 5 with MILWA/LOMILWA (3 with quick plot).
  // The result is the set of things the presentation may draw, whatever the
  // camera. It depends only on the party's square and facing.
  function visibleFaces(lightDis) {
    const faces = []
    for (let d = 0; d < lightDis; d++) {
      faces.push({ kind: 'floor', cx: 0, cz: d }, { kind: 'ceil', cx: 0, cz: d })
      for (const [side, dx] of [['W', -1], ['E', 1]]) {
        const t = edge(0, d, side)
        if (t !== 'open') faces.push({ kind: 'wall', cx: 0, cz: d, side, type: t })
        else {
          const f = edge(dx, d, 'N')
          if (f !== 'open') {
            faces.push({ kind: 'wall', cx: dx, cz: d, side: 'N', type: f })
            faces.push({ kind: 'floor', cx: dx, cz: d }, { kind: 'ceil', cx: dx, cz: d })
          }
        }
      }
      const front = edge(0, d, 'N')
      if (front !== 'open') {
        faces.push({ kind: 'wall', cx: 0, cz: d, side: 'N', type: front })
        break
      }
    }
    return faces
  }

  // ── geometry ────────────────────────────────────────────────────────────
  const sub = (a, b) => [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
  const dot = (a, b) => a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
  const cross = (a, b) => [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]]
  const norm = (a) => { const l = Math.hypot(a[0], a[1], a[2]) || 1; return [a[0] / l, a[1] / l, a[2] / l] }

  function camera(pos, target, fovDeg, W, Hpx) {
    const f = norm(sub(target, pos))
    const r = norm(cross([0, 1, 0], f))
    const u = cross(f, r)
    const focal = (W / 2) / Math.tan((fovDeg * Math.PI) / 360)
    return {
      pos, W, H: Hpx,
      view: (p) => { const v = sub(p, pos); return [dot(v, r), dot(v, u), dot(v, f)] },
      proj: (c) => [W / 2 + (c[0] / c[2]) * focal, Hpx / 2 - (c[1] / c[2]) * focal],
    }
  }

  const NEAR = 0.05
  // Clip a polygon (view coordinates) against the near plane.
  function clipNear(poly) {
    const out = []
    for (let i = 0; i < poly.length; i++) {
      const a = poly[i], b = poly[(i + 1) % poly.length]
      const ain = a[2] > NEAR, bin = b[2] > NEAR
      if (ain) out.push(a)
      if (ain !== bin) {
        const t = (NEAR - a[2]) / (b[2] - a[2])
        out.push([a[0] + t * (b[0] - a[0]), a[1] + t * (b[1] - a[1]), NEAR])
      }
    }
    return out
  }

  // World quad for a face. Walls are inset slightly so doors sit proud.
  function quad(face) {
    const x0 = face.cx * S - S / 2, x1 = x0 + S
    const z0 = face.cz * S - S / 2, z1 = z0 + S
    if (face.kind === 'floor') return [[x0, 0, z0], [x1, 0, z0], [x1, 0, z1], [x0, 0, z1]]
    if (face.kind === 'ceil') return [[x0, H, z0], [x0, H, z1], [x1, H, z1], [x1, H, z0]]
    switch (face.side) {
      case 'N': return [[x0, 0, z1], [x1, 0, z1], [x1, H, z1], [x0, H, z1]]
      case 'S': return [[x1, 0, z0], [x0, 0, z0], [x0, H, z0], [x1, H, z0]]
      case 'W': return [[x0, 0, z0], [x0, 0, z1], [x0, H, z1], [x0, H, z0]]
      case 'E': return [[x1, 0, z1], [x1, 0, z0], [x1, H, z0], [x1, H, z1]]
    }
  }

  // Door panel on a wall face, in world coordinates.
  function doorQuad(face) {
    const q = quad(face)
    const lerp = (a, b, t) => [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t]
    const bl = lerp(q[0], q[1], 0.3), br = lerp(q[0], q[1], 0.7)
    const up = (p, h) => [p[0], h, p[2]]
    return [bl, br, up(br, 2.3), up(bl, 2.3)]
  }

  // ── styles (visual vocabulary) ──────────────────────────────────────────
  const STYLES = {
    immersive: { bg: '#07080a', wall: [92, 84, 72], floor: [52, 48, 42], ceil: [30, 29, 28], door: [96, 62, 34], line: 'none', mortar: 'rgba(0,0,0,0.35)', lightRange: 15 },
    rich: { bg: '#07080a', wall: [92, 84, 72], floor: [52, 48, 42], ceil: [30, 29, 28], door: [96, 62, 34], line: 'none', mortar: 'rgba(0,0,0,0.35)', lightRange: 15 },
    classic: { bg: '#000', line: '#e8e8e8', classic: true },
  }

  function shade(rgb, b) {
    const k = Math.max(0.05, Math.min(1, b))
    return `rgb(${Math.round(rgb[0] * k)},${Math.round(rgb[1] * k)},${Math.round(rgb[2] * k)})`
  }

  function polyPath(cam, worldPts) {
    const v = clipNear(worldPts.map(cam.view))
    if (v.length < 3) return null
    return { d: 'M' + v.map((p) => cam.proj(p).map((n) => n.toFixed(1)).join(',')).join('L') + 'Z', depth: v.reduce((s, p) => s + p[2], 0) / v.length }
  }

  // ── figures (generic placeholders, not original game art) ───────────────
  // In-square formation: presentation only. The rules know two lines
  // (slots 1-3 front, 4-6 back) and nothing about positions within a square.
  const FORMATION = [[-0.9, 1.0], [0, 1.1], [0.9, 1.0], [-0.9, -0.15], [0, -0.05], [0.9, -0.15]]

  function figureSprite(cam, x, z, opt) {
    const base = cam.view([x, 0, z]), top = cam.view([x, opt.h || 1.45, z])
    if (base[2] < 0.3) return null
    const [bx, by] = cam.proj(base), [, ty] = cam.proj(top)
    const h = by - ty, w = h * (opt.wide || 0.34)
    let g = ''
    if (opt.monster) {
      g += `<ellipse cx="${bx}" cy="${by - h * 0.42}" rx="${w * 0.75}" ry="${h * 0.45}" fill="${opt.color}" stroke="${opt.classic ? '#e8e8e8' : '#000'}" stroke-width="1.2"/>`
      g += `<circle cx="${bx - w * 0.22}" cy="${by - h * 0.62}" r="${Math.max(1, h * 0.035)}" fill="#e9d36b"/><circle cx="${bx + w * 0.22}" cy="${by - h * 0.62}" r="${Math.max(1, h * 0.035)}" fill="#e9d36b"/>`
    } else if (opt.classic) {
      g += `<rect x="${bx - w / 2}" y="${by - h * 0.78}" width="${w}" height="${h * 0.78}" rx="${w * 0.3}" fill="none" stroke="#e8e8e8"/><circle cx="${bx}" cy="${by - h * 0.88}" r="${h * 0.11}" fill="none" stroke="#e8e8e8"/>`
    } else {
      g += `<ellipse cx="${bx}" cy="${by}" rx="${w * 0.8}" ry="${w * 0.22}" fill="rgba(0,0,0,0.45)"/>`
      g += `<rect x="${bx - w / 2}" y="${by - h * 0.78}" width="${w}" height="${h * 0.78}" rx="${w * 0.32}" fill="${opt.color}" stroke="#111" stroke-width="1"/>`
      g += `<circle cx="${bx}" cy="${by - h * 0.88}" r="${h * 0.11}" fill="${opt.color}" stroke="#111" stroke-width="1"/>`
    }
    if (opt.selected) g += `<ellipse cx="${bx}" cy="${by}" rx="${w * 0.95}" ry="${w * 0.3}" fill="none" stroke="#f2c14e" stroke-width="2" stroke-dasharray="4 3"/>`
    if (opt.label) {
      const fs = Math.max(9, Math.min(16, h * 0.13))
      g += `<text x="${bx}" y="${by - h * 0.42}" font-size="${fs}" text-anchor="middle" fill="${opt.classic ? '#e8e8e8' : '#fff'}" font-family="ui-monospace,Menlo,monospace" font-weight="700">${opt.label}</text>`
    }
    if (opt.flash) g += `<circle cx="${bx}" cy="${by - h * 0.5}" r="${h * 0.45}" fill="none" stroke="#ff6b4a" stroke-width="3" opacity="0.85"/><text x="${bx + w}" y="${by - h * 0.95}" font-size="${Math.max(12, h * 0.16)}" fill="#ff6b4a" font-weight="700" font-family="ui-monospace,Menlo,monospace">${opt.flash}</text>`
    return { depth: base[2], svg: g, anchor: [bx, ty] }
  }

  function boxFaces(cx, cz, w, d, h) {
    const x0 = cx - w / 2, x1 = cx + w / 2, z0 = cz - d / 2, z1 = cz + d / 2
    return [
      { pts: [[x0, h, z0], [x0, h, z1], [x1, h, z1], [x1, h, z0]], b: 1 },
      { pts: [[x0, 0, z0], [x1, 0, z0], [x1, h, z0], [x0, h, z0]], b: 0.8 },
      { pts: [[x1, 0, z0], [x1, 0, z1], [x1, h, z1], [x1, h, z0]], b: 0.65 },
      { pts: [[x0, 0, z1], [x0, 0, z0], [x0, h, z0], [x0, h, z1]], b: 0.65 },
    ]
  }

  // ── render ──────────────────────────────────────────────────────────────
  // shot: { pos, target, fov, lightDis, style, party: [{slot,label,color}],
  //         target slot, monsters: [{x,z,color,label,flash}], chest, voidNote }
  function render(shot, W, Hpx) {
    W = W || 960; Hpx = Hpx || 540
    const st = STYLES[shot.style || 'immersive']
    const cam = camera(shot.pos, shot.target, shot.fov || 70, W, Hpx)
    const items = []
    const lightPos = [0, 1.6, 0]

    for (const face of visibleFaces(shot.lightDis || 5)) {
      const q = quad(face)
      const p = polyPath(cam, q)
      if (!p) continue
      const centre = q.reduce((a, b) => [a[0] + b[0] / 4, a[1] + b[1] / 4, a[2] + b[2] / 4], [0, 0, 0])
      const dist = Math.hypot(...sub(centre, lightPos))
      let svg
      if (st.classic) {
        svg = face.kind === 'wall' ? `<path d="${p.d}" fill="#000" stroke="${st.line}" stroke-width="1.6"/>` : ''
      } else {
        const b = 1.05 - dist / st.lightRange
        const base = face.kind === 'floor' ? st.floor : face.kind === 'ceil' ? st.ceil : st.wall
        const sideK = face.kind === 'wall' && face.side !== 'N' ? 0.82 : 1
        svg = `<path d="${p.d}" fill="${shade(base, b * sideK)}" stroke="rgba(0,0,0,0.5)" stroke-width="1"/>`
        if (face.kind === 'wall') {
          for (const yy of [0.75, 1.5, 2.25]) {
            const a = q[0], c = q[1]
            const m = polyPath(cam, [[a[0], yy, a[2]], [c[0], yy, c[2]], [c[0], yy + 0.01, c[2]], [a[0], yy + 0.01, a[2]]])
            if (m) svg += `<path d="${m.d}" fill="none" stroke="${st.mortar}" stroke-width="1"/>`
          }
        }
      }
      if (face.kind === 'wall' && face.type === 'door') {
        const dq = polyPath(cam, doorQuad(face))
        if (dq) svg += st.classic ? `<path d="${dq.d}" fill="none" stroke="${st.line}" stroke-width="1.6"/>` : `<path d="${dq.d}" fill="${shade(st.door, 1.05 - dist / st.lightRange)}" stroke="#1a0f06" stroke-width="2"/>`
      }
      // Floors and ceilings always sort behind walls and figures.
      items.push({ depth: p.depth + (face.kind === 'wall' ? 0 : 1000), svg })
    }

    if (shot.chest) {
      const c = shot.chest
      for (const f of boxFaces(c.x, c.z, 1.0, 0.65, 0.6)) {
        const p = polyPath(cam, f.pts)
        if (!p) continue
        items.push({ depth: p.depth, svg: st.classic ? `<path d="${p.d}" fill="#000" stroke="#e8e8e8"/>` : `<path d="${p.d}" fill="${shade([120, 82, 40], f.b)}" stroke="#2a1806" stroke-width="1.5"/>` })
      }
      const band = polyPath(cam, [[c.x - 0.51, 0.42, c.z - 0.34], [c.x + 0.51, 0.42, c.z - 0.34], [c.x + 0.51, 0.47, c.z - 0.34], [c.x - 0.51, 0.47, c.z - 0.34]])
      if (band) items.push({ depth: band.depth - 0.01, svg: `<path d="${band.d}" fill="${st.classic ? 'none' : '#b8a060'}" stroke="${st.classic ? '#e8e8e8' : '#3a2a10'}"/>` })
    }

    const anchors = {}
    for (const m of shot.party || []) {
      const [x, z] = m.at || FORMATION[m.slot - 1]
      const s = figureSprite(cam, x, z, { color: m.color, label: String(m.slot), selected: m.slot === shot.targetSlot, classic: st.classic, flash: m.flash, h: m.h })
      if (s) { items.push(s); anchors['p' + m.slot] = s.anchor }
    }
    for (const [i, m] of (shot.monsters || []).entries()) {
      const s = figureSprite(cam, m.x, m.z, { monster: true, classic: st.classic, color: st.classic ? '#000' : m.color, h: m.h || 1.35, wide: 0.5, flash: m.flash })
      if (s) { items.push(s); anchors['m' + i] = s.anchor }
    }

    items.sort((a, b) => b.depth - a.depth)
    return {
      svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${W} ${Hpx}" width="100%" preserveAspectRatio="xMidYMid slice" role="img"><rect width="${W}" height="${Hpx}" fill="${st.bg}"/>${items.map((i) => i.svg).join('')}</svg>`,
      anchors,
    }
  }

  const api = { render, visibleFaces, FORMATION, MAP }
  if (typeof module !== 'undefined' && module.exports) module.exports = api
  else root.IstaryScene = api
})(this)
