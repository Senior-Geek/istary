// Tests of scripts/lib/classify.mjs on a package graph shaped like Istary's
// (the real one is read with go list; these fix it so the rules are clear).
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { classify } from './classify.mjs'

const M = 'github.com/Senior-Geek/istary'
const R = '/repo'
const pkg = (path, module, deps) => ({ importPath: `${M}/${path}`, dir: `${R}/${path}`, module, deps: new Set(deps.map((d) => `${M}/${d}`)) })
const PKGS = [
  pkg('data', '.', []),
  pkg('engine', '.', ['data']),
  pkg('app', '.', ['data', 'engine']),
  pkg('render', '.', ['data', 'engine']),
  pkg('scenarios/wiz1', '.', ['data']),
  pkg('cmd/istary', '.', ['app', 'data', 'engine', 'render', 'scenarios/wiz1']),
  pkg('gui', 'gui', ['app', 'data', 'engine', 'scenarios/wiz1']),
]
const plan = (...files) => classify(files, PKGS, R)
const pkgs = (p) => p.goPackages.map((x) => x.pkg.slice(M.length + 1)).sort()

test('a frontend change tests the frontend and builds it, and no Go', () => {
  const p = plan('gui/frontend/src/lib/LocationFrame.svelte')
  assert.deepEqual(pkgs(p), [])
  assert.equal(p.frontend, true)
  assert.equal(p.frontendBuild, true)
  assert.equal(p.broad, false)
  assert.deepEqual(p.live, ['layout', 'sheet', 'scenes'])
})

test('creation in the frontend adds the creation live checks', () => {
  const p = plan('gui/frontend/src/lib/TrainingScreen.svelte')
  assert.ok(p.areas.includes('creation'))
  assert.ok(p.live.includes('creation'))
})

test('a change to engine reaches every package that depends on it, in both modules', () => {
  const p = plan('engine/item.go')
  assert.deepEqual(pkgs(p), ['app', 'cmd/istary', 'engine', 'gui', 'render'])
  assert.equal(p.frontend, true, 'the GUI module is reached, so its recorded fixtures are read by the frontend tests')
  assert.equal(p.makeBuild, true)
})

test('a change to app does not test engine', () => {
  assert.deepEqual(pkgs(plan('app/shop.go')), ['app', 'cmd/istary', 'gui'])
})

test('a test file tests its own package only', () => {
  assert.deepEqual(pkgs(plan('app/shop_test.go')), ['app'])
  assert.deepEqual(pkgs(plan('cmd/istary/testdata/screen.golden')), ['cmd/istary'])
})

test('embedded scenario data reaches what embeds it and what depends on that', () => {
  assert.deepEqual(pkgs(plan('scenarios/wiz1/wiz1_gamedata.json')), ['cmd/istary', 'gui', 'scenarios/wiz1'])
})

test('recorded GUI fixtures run the GUI tests and the frontend tests', () => {
  const p = plan('gui/frontend/src/lib/testdata/creation-views.json')
  assert.deepEqual(pkgs(p), ['gui'])
  assert.equal(p.frontend, true)
})

test('persistence and concurrency want race tests and the full gate', () => {
  const p = plan('gui/session.go')
  assert.deepEqual(p.race.map((x) => x.pkg.slice(M.length + 1)), ['gui'])
  assert.equal(p.gateRequired, true)
  const s = plan('engine/save.go')
  assert.deepEqual(s.race.map((x) => x.pkg.slice(M.length + 1)), ['gui', 'engine', 'cmd/istary'])
})

test('documentation alone tests nothing', () => {
  const p = plan('docs/RULES.md', 'README.md', 'images/term_maze.png')
  assert.deepEqual(pkgs(p), [])
  assert.equal(p.frontend, false)
  assert.equal(p.broad, false)
  assert.equal(p.gateRequired, false)
})

test('an unknown file, the Go modules or the frontend dependencies broaden to everything', () => {
  for (const f of ['tools/new.sh', 'go.mod', 'gui/go.sum', 'gui/frontend/package-lock.json', 'newdir/x.go']) {
    const p = plan(f)
    assert.equal(p.broad, true, f)
    assert.equal(p.goPackages.length, PKGS.length, f)
    assert.equal(p.frontend && p.wailsBuild && p.makeBuild, true, f)
    assert.ok(p.race.length === 3 && p.gateRequired, f)
    assert.deepEqual(p.live, ['layout', 'creation', 'sheet', 'scenes'], f)
  }
})

test('the gate and the test scripts run their own tests', () => {
  const g = plan('scripts/gate.sh')
  assert.equal(g.gateScripts, true)
  assert.equal(g.gateRequired, true)
  assert.equal(plan('scripts/lib/gotest.mjs').testScripts, true)
  assert.equal(plan('scripts/test-fast.sh').testScripts, true)
})

test('packaging builds the app', () => {
  const p = plan('gui/build/darwin/Info.plist')
  assert.equal(p.wailsBuild, true)
  assert.equal(p.broad, false)
})
