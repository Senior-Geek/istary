// Istary M2 combat prototype: exposure check for camera shots.
//
// NOT production code and not part of the application. Counts the fixture
// surfaces the team has never seen that a camera shot would actually expose:
// facing the camera, inside the frame, and not hidden behind a wall or door.
// Walls are full height, so occlusion is tested in plan. The SVG mockup
// renderer cannot answer this itself: it has no occlusion, only a painter's
// sort, so drawing the true level ("unbounded") overstates what is exposed.
//
// Usage: node leak-check.js                (the shots the prototypes use)
//        node leak-check.js '{"name":{"pos":[x,y,z],"target":[x,y,z],"fov":n}}'

'use strict'
const CS = require('./combat-scene.js')

const S = 4, H = 3
const sub = (a, b) => [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
const dot = (a, b) => a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
const cross = (a, b) => [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]]
const norm = (a) => { const l = Math.hypot(a[0], a[1], a[2]); return [a[0] / l, a[1] / l, a[2] / l] }

// Same face geometry as combat-scene.js.
function quad(face) {
  const x0 = face.cx * S - S / 2, x1 = x0 + S, z0 = face.cz * S - S / 2, z1 = z0 + S
  if (face.kind === 'floor') return [[x0, 0, z0], [x1, 0, z0], [x1, 0, z1], [x0, 0, z1]]
  if (face.kind === 'ceil') return [[x0, H, z0], [x0, H, z1], [x1, H, z1], [x1, H, z0]]
  return {
    N: [[x0, 0, z1], [x1, 0, z1], [x1, H, z1], [x0, H, z1]],
    S: [[x1, 0, z0], [x0, 0, z0], [x0, H, z0], [x1, H, z0]],
    W: [[x0, 0, z0], [x0, 0, z1], [x0, H, z1], [x0, H, z0]],
    E: [[x1, 0, z1], [x1, 0, z0], [x1, H, z0], [x1, H, z1]],
  }[face.side]
}
// A surface faces into its own square; seen from behind it shows nothing.
const INWARD = { N: [0, 0, -1], S: [0, 0, 1], W: [1, 0, 0], E: [-1, 0, 0] }
function facesCamera(face, pos) {
  if (face.kind === 'floor') return pos[1] > 0
  if (face.kind === 'ceil') return pos[1] < H
  return dot(sub(pos, quad(face)[0]), INWARD[face.side]) > 0
}

const all = CS.faceSet('unbounded', 0, 5).faces
const walls = all.filter((x) => x.f.kind === 'wall').map((x) => { const q = quad(x.f); return { f: x.f, a: [q[0][0], q[0][2]], b: [q[1][0], q[1][2]] } })

// Does the plan segment p→q cross the wall a→b strictly before reaching q?
function blocks(p, q, a, b) {
  const d = (q[0] - p[0]) * (b[1] - a[1]) - (q[1] - p[1]) * (b[0] - a[0])
  if (Math.abs(d) < 1e-9) return false
  const t = ((a[0] - p[0]) * (b[1] - a[1]) - (a[1] - p[1]) * (b[0] - a[0])) / d
  const u = ((a[0] - p[0]) * (q[1] - p[1]) - (a[1] - p[1]) * (q[0] - p[0])) / d
  return t > 1e-4 && t < 1 - 1e-3 && u > 1e-4 && u < 1 - 1e-4
}

function exposed(shot, face) {
  if (!facesCamera(face, shot.pos)) return false
  const W = 960, Hp = 540
  const f = norm(sub(shot.target, shot.pos)), r = norm(cross([0, 1, 0], f)), u = cross(f, r)
  const focal = (W / 2) / Math.tan((shot.fov * Math.PI) / 360)
  const q = quad(face)
  for (let i = 1; i < 8; i++) {
    for (let j = 1; j < 8; j++) {
      const a = i / 8, b = j / 8
      const p = [0, 1, 2].map((k) => q[0][k] * (1 - a) * (1 - b) + q[1][k] * a * (1 - b) + q[2][k] * a * b + q[3][k] * (1 - a) * b)
      const v = sub(p, shot.pos), c = [dot(v, r), dot(v, u), dot(v, f)]
      if (c[2] < 0.05) continue
      const sx = W / 2 + (c[0] / c[2]) * focal, sy = Hp / 2 - (c[1] / c[2]) * focal
      if (sx < 0 || sx > W || sy < 0 || sy > Hp) continue
      const from = [shot.pos[0], shot.pos[2]], to = [p[0], p[2]]
      if (!walls.some((w) => w.f !== face && blocks(from, to, w.a, w.b))) return true
    }
  }
  return false
}

const DEFAULT = {
  'refined: battlefield (stable)': { pos: [1.35, 2.88, -1.95], target: [-0.3, -0.35, 4.6], fov: 92 },
  'refined: push-in, Torvald': { pos: [0.8, 2.88, -1.6], target: [-0.6, -0.35, 4.6], fov: 92 },
  'refined: push-in, Brannoc': { pos: [1.35, 2.88, -1.6], target: [0.2, -0.35, 4.6], fov: 92 },
  'chest close-up': { pos: [1.5, 2.1, 1.7], target: [-0.15, 0.25, 4.4], fov: 58 },
  'first person': { pos: [0, 1.55, -1.9], target: [0, 1.45, 10], fov: 72 },
  'combat.html: reverse shot': { pos: [1.3, 2.85, 9.6], target: [-0.15, 0.6, 0.6], fov: 66 },
}
const shots = process.argv[2] ? JSON.parse(process.argv[2]) : DEFAULT
for (const [name, shot] of Object.entries(shots)) {
  const e = all.filter((x) => x.leak && exposed(shot, x.f)).map((x) => `${x.f.kind}(${x.f.cx},${x.f.cz})${x.f.side || ''}`)
  console.log(`${name.padEnd(30)} never-seen surfaces exposed: ${e.length}${e.length ? '  ' + e.join(' ') : ''}`)
}
