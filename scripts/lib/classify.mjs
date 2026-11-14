#!/usr/bin/env node
// classify.mjs decides what a change needs tested (docs/TESTING.md). Given
// the changed files, it finds the Go packages they belong to and every
// package that depends on those, in either module, test imports included
// (go list -test -deps), so a change crossing subsystems is followed by its
// dependencies, not its file name. File names only label the change's areas,
// which choose the live checks and say when race tests and the full gate are
// due. A file it cannot place broadens the plan to everything.
//
//   node scripts/lib/classify.mjs REPO_ROOT [FILE...]    (or the files on stdin)
//
// It prints the plan as JSON.

import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const ROOT_MODULE = 'github.com/Senior-Geek/istary'
const GUI_MODULE = `${ROOT_MODULE}/gui`

// Areas, by path. They label a change; they never choose Go tests.
const AREAS = [
  ['creation', /creation|training|classguide|TrainingScreen/i],
  ['equipment', /equip|shop|stock|trading|curse|item|gold|CharacterSheet|sheet/i],
  ['combat-maze', /combat|dungeon|maze|camp|chest|explore|movement|stairs|automap|map|marker|position|expedition|rescue|stranded|encounter|spell/i],
  ['town', /tavern|inn|temple|edge|castle|navigation|party|location|LocationFrame|Scene/i],
  ['persistence', /save|atomicfile|roster|games|backup|settings|lock|close|legacy|migrat|scenario/i],
]
// Code that runs on more than one goroutine, or guards a file shared between
// processes: a change here wants race tests and the full gate.
const CONCURRENT = [/^gui\/(session|lock|lock_other|close|main|games|backups|settings)\.go$/, /^engine\/(game|save|atomicfile)\.go$/, /^cmd\/istary\/main\.go$/, /^render\/title_anim\.go$/]
// Files that change nothing tested or built.
const INERT = [/\.md$/i, /^docs\//, /^images\//, /^LICENSE$/, /^\.gitignore$/]

export function goPackages(root) {
  // Every package of both modules with its directory and everything its tests
  // depend on: one entry per package (its test variant, when it has tests).
  const list = (dir) => {
    const out = execFileSync('go', ['list', '-test', '-deps=false', '-f', '{{.ImportPath}}|{{.Dir}}|{{join .Deps ","}}', './...'], { cwd: dir, encoding: 'utf8' })
    const pkgs = {}
    for (const line of out.trim().split('\n')) {
      const [ip, d, deps] = line.split('|')
      if (ip.endsWith('.test')) continue
      // "p [p.test]" and an external test package "p_test [p.test]" are p.
      const base = ip.replace(/ \[.*\]$/, '').replace(/_test$/, '')
      const e = (pkgs[base] ??= { importPath: base, dir: d, deps: new Set() })
      for (const x of deps ? deps.split(',') : []) e.deps.add(x)
    }
    return Object.values(pkgs)
  }
  return [...list(root).map((p) => ({ ...p, module: '.' })), ...list(join(root, 'gui')).map((p) => ({ ...p, module: 'gui' }))]
}

export function classify(files, pkgs, root) {
  const plan = {
    files: [], broad: false, broadReasons: [],
    goPackages: [], frontend: false, frontendBuild: false, bindings: false,
    makeBuild: false, wailsBuild: false, gateScripts: false, testScripts: false,
    race: [], live: [], gateRequired: false, gateReasons: [], areas: [],
  }
  const affected = new Set()
  const areas = new Set()
  const rel = (dir) => (dir === root ? '' : dir.slice(root.length + 1))
  const byDir = pkgs.map((p) => ({ ...p, rel: rel(p.dir) })).sort((a, b) => b.rel.length - a.rel.length)
  const broaden = (why) => { plan.broad = true; plan.broadReasons.push(why) }
  const gate = (why) => { plan.gateRequired = true; if (!plan.gateReasons.includes(why)) plan.gateReasons.push(why) }
  const dependents = (ip) => pkgs.filter((p) => p.importPath === ip || p.deps.has(ip)).map((p) => p.importPath)

  for (const f of files) {
    const entry = { path: f, kind: '', areas: [] }
    for (const [name, re] of AREAS) if (re.test(f)) entry.areas.push(name)
    if (CONCURRENT.some((re) => re.test(f))) entry.areas.push('concurrency')
    if (INERT.some((re) => re.test(f))) {
      entry.kind = 'documentation'
      entry.areas = []
    } else if (/^gui\/frontend\/src\/lib\/testdata\//.test(f)) {
      // Recorded by the GUI's Go tests, read by the frontend's.
      entry.kind = 'fixture'
      plan.frontend = true
      affected.add(GUI_MODULE)
    } else if (/^gui\/frontend\/wailsjs\//.test(f)) {
      entry.kind = 'bindings'
      plan.bindings = plan.frontend = plan.frontendBuild = true
      affected.add(GUI_MODULE)
    } else if (/^gui\/frontend\/(src|public)\//.test(f) || f === 'gui/frontend/index.html') {
      entry.kind = 'frontend'
      plan.frontend = plan.frontendBuild = plan.wailsBuild = true
      if (!entry.areas.length) entry.areas.push('ui')
    } else if (/^gui\/frontend\/(package(-lock)?\.json|vite\.config\.\w+|svelte\.config\.\w+|jsconfig\.json)$/.test(f)) {
      entry.kind = 'frontend dependencies'
      broaden(`${f}: the frontend's dependencies or build`)
    } else if (/^gui\/build\//.test(f) || f === 'gui/wails.json') {
      entry.kind = 'packaging'
      plan.wailsBuild = true
      gate(`${f}: the application bundle`)
    } else if (/^(gui\/)?go\.(mod|sum)$/.test(f)) {
      entry.kind = 'go modules'
      broaden(`${f}: Go dependencies`)
      gate(`${f}: Go dependencies`)
    } else if (f === 'Makefile') {
      entry.kind = 'build'
      plan.makeBuild = true
    } else if (/^scripts\/gate(_test)?\.sh$/.test(f)) {
      entry.kind = 'gate'
      plan.gateScripts = true
      gate(`${f}: the regression gate itself`)
    } else if (/^scripts\/(test-[\w-]+\.sh|lib\/|acceptance\/)/.test(f)) {
      entry.kind = 'test scripts'
      plan.testScripts = true
    } else {
      const dir = f.includes('/') ? f.slice(0, f.lastIndexOf('/')) : ''
      // The package whose directory holds the file (or holds the testdata
      // directory it is in): its .go files, embedded data and fixtures.
      const p = byDir.find((x) => x.rel && (dir === x.rel || dir.startsWith(x.rel + '/')))
      if (!p) {
        entry.kind = 'unclassified'
        broaden(`${f}: not part of any package or known file`)
      } else if (/_test\.go$/.test(f) || dir !== p.rel) {
        entry.kind = dir !== p.rel ? 'test data' : 'go test'
        entry.package = p.importPath
        // Test files and testdata are the package's own tests, but a
        // non-test file elsewhere in a package directory could be embedded.
        if (dir !== p.rel && !/\/testdata(\/|$)/.test(dir)) for (const d of dependents(p.importPath)) affected.add(d)
        else affected.add(p.importPath)
      } else {
        entry.kind = /\.go$/.test(f) ? 'go' : 'embedded data'
        entry.package = p.importPath
        for (const d of dependents(p.importPath)) affected.add(d)
      }
    }
    for (const a of entry.areas) areas.add(a)
    plan.files.push(entry)
  }

  if (plan.broad) {
    for (const p of pkgs) affected.add(p.importPath)
    plan.frontend = plan.frontendBuild = plan.makeBuild = plan.wailsBuild = true
    plan.gateScripts = plan.testScripts = true
    for (const a of ['ui', 'creation', 'equipment', 'town', 'combat-maze', 'persistence', 'concurrency']) areas.add(a)
  }
  plan.goPackages = pkgs.filter((p) => affected.has(p.importPath)).map((p) => ({ module: p.module, pkg: p.importPath }))
  if (affected.has(GUI_MODULE)) plan.frontend = plan.wailsBuild = true
  if (plan.goPackages.some((p) => p.module === '.' && /cmd\/istary$/.test(p.pkg))) plan.makeBuild = true

  // Race tests: the packages with code on several goroutines, when the
  // change reaches them and touches concurrency or saving.
  if (areas.has('concurrency') || areas.has('persistence')) {
    for (const ip of [GUI_MODULE, `${ROOT_MODULE}/engine`, `${ROOT_MODULE}/cmd/istary`]) {
      const p = plan.goPackages.find((x) => x.pkg === ip)
      if (p) plan.race.push(p)
    }
    if (areas.has('concurrency')) gate('concurrency: a goroutine, the session lock or the save-folder lock')
    if (areas.has('persistence')) gate('persistence: saving, loading or the save folder')
  }
  if (plan.broad) gate('the change could not be narrowed')

  // Live checks: the town interface, whenever what the window shows could change.
  if (plan.frontend || affected.has(GUI_MODULE)) {
    plan.live.push('layout')
    if (areas.has('creation') || plan.broad) plan.live.push('creation')
    if (areas.has('equipment') || areas.has('ui') || areas.has('town') || plan.broad) plan.live.push('sheet')
    if (areas.has('ui') || areas.has('town') || plan.broad) plan.live.push('scenes')
    if (areas.has('combat-maze') || plan.broad) plan.live.push('combat')
  }
  plan.areas = [...areas].sort()
  return plan
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const root = process.argv[2]
  if (!root) { console.error('usage: classify.mjs REPO_ROOT [FILE...]'); process.exit(2) }
  let files = process.argv.slice(3)
  if (!files.length) files = readFileSync(0, 'utf8').split('\n')
  files = [...new Set(files.map((f) => f.trim()).filter(Boolean))]
  const plan = classify(files, goPackages(root), root)
  console.log(JSON.stringify(plan, null, 1))
}
