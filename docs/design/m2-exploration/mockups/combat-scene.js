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
  const edge = (cx, cz, side) => { const c = MAP[cx + ',' + cz]; return c ? c[side] : 'wall' }
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
    for (const k of Object.keys(MAP)) {
      const [cx, cz] = k.split(',').map(Number)
      faces.push({ kind: 'floor', cx, cz }, { kind: 'ceil', cx, cz })
      for (const side of ['N', 'S', 'W', 'E']) if (MAP[k][side] !== 'open') faces.push({ kind: 'wall', cx, cz, side, type: MAP[k][side] })
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
  function faceSet(policy, teamZ, lightDis) {
    const known = visibleFrom(teamZ, lightDis)
    const remembered = union(visibleFrom(-2, lightDis), visibleFrom(-1, lightDis), known)
    const permitted = new Set(remembered.map(key))
    if (policy === 'strict' || policy === 'shell') return { faces: known.map((f) => ({ f })), shell: policy === 'shell' }
    if (policy === 'remembered') {
      const now = new Set(known.map(key))
      return { faces: remembered.map((f) => ({ f, memory: !now.has(key(f)) })), shell: true }
    }
    return { faces: allFaces().map((f) => ({ f, leak: !permitted.has(key(f)) })), shell: false }
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

  function personSvg(s, m) {
    const h = s.hh, w = h * 0.34, { bx, by } = s
    const col = m.state === 'dead' ? '#3a3a3a' : m.color
    let g = `<ellipse cx="${bx}" cy="${by}" rx="${w * 0.85}" ry="${w * 0.24}" fill="rgba(0,0,0,0.5)"/>`
    if (m.selected) g += `<ellipse class="selring" cx="${bx}" cy="${by}" rx="${w * 1.05}" ry="${w * 0.33}" fill="none" stroke="#f2c14e" stroke-width="2.5"/>`
    else if (m.candidate) g += `<ellipse class="candring" cx="${bx}" cy="${by}" rx="${w * 1.05}" ry="${w * 0.33}" fill="none" stroke="#8fd18f" stroke-width="2" stroke-dasharray="5 3"/>`
    const bodyY = by - h * 0.8
    g += `<rect x="${bx - w / 2}" y="${bodyY}" width="${w}" height="${h * 0.8}" rx="${w * 0.34}" fill="${col}" stroke="${m.selected ? '#f2c14e' : '#0b0b0b'}" stroke-width="${m.selected ? 2 : 1}"/>`
    g += `<rect x="${bx - w / 2}" y="${bodyY}" width="${w}" height="${h * 0.8}" rx="${w * 0.34}" fill="url(#cs-shade)"/>`
    g += `<circle cx="${bx}" cy="${by - h * 0.9}" r="${h * 0.11}" fill="${col}" stroke="${m.selected ? '#f2c14e' : '#0b0b0b'}" stroke-width="${m.selected ? 2 : 1}"/>`
    const fs = Math.max(9, Math.min(17, h * 0.15))
    g += `<text x="${bx}" y="${by - h * 0.42}" font-size="${fs}" text-anchor="middle" fill="#fff" font-family="ui-monospace,Menlo,monospace" font-weight="700" style="paint-order:stroke" stroke="rgba(0,0,0,0.6)" stroke-width="2.5">${m.slot}</text>`
    if (m.state === 'poison') g += `<circle cx="${bx + w * 0.55}" cy="${by - h * 1.02}" r="${Math.max(3, h * 0.05)}" fill="#7bd36a" stroke="#000"/>`
    if (m.dim) g = `<g opacity="0.45">${g}</g>`
    return g
  }

  function monsterSvg(s, m, gi) {
    const h = s.hh, w = h * 0.5, { bx, by } = s
    const col = m.asleep ? '#4c4f58' : m.color
    let g = `<ellipse cx="${bx}" cy="${by}" rx="${w * 0.8}" ry="${w * 0.2}" fill="rgba(0,0,0,0.55)"/>`
    g += `<ellipse cx="${bx}" cy="${by - h * 0.42}" rx="${w * 0.72}" ry="${h * 0.44}" fill="${col}" stroke="#0b0b0b" stroke-width="1.2"/>`
    g += `<ellipse cx="${bx}" cy="${by - h * 0.42}" rx="${w * 0.72}" ry="${h * 0.44}" fill="url(#cs-shade)"/>`
    if (!m.asleep) {
      const r = Math.max(1.2, h * 0.035)
      g += `<circle cx="${bx - w * 0.22}" cy="${by - h * 0.62}" r="${r}" fill="#f0dc72"/><circle cx="${bx + w * 0.22}" cy="${by - h * 0.62}" r="${r}" fill="#f0dc72"/>`
    } else {
      g += `<text x="${bx + w * 0.4}" y="${by - h * 0.9}" font-size="${Math.max(10, h * 0.2)}" fill="#b9b2ff" font-weight="700" font-family="ui-monospace,Menlo,monospace">z</text>`
    }
    if (m.fading) g = `<g opacity="0.3">${g}</g>`
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
    const set = faceSet(shot.policy || 'strict', teamZ, shot.lightDis || 5)
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
      items.push({ depth: p.depth + (face.kind === 'wall' ? 0 : 1000), svg })
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
        const s = sprite(cam, m.x, m.z, m.h || 1.35)
        if (!s) continue
        items.push({ depth: s.depth, svg: monsterSvg(s, m, gi) })
        top = top === null ? s.ty : Math.min(top, s.ty); sx += s.bx; n++
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

  const api = { render, visibleFrom, faceSet, FORMATION, GROUP_COLOR, MAP, lerp3 }
  if (typeof module !== 'undefined' && module.exports) module.exports = api
  else root.CombatScene = api
})(this)
