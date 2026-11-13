#!/usr/bin/env node
// gotest.mjs runs the Go tests of some packages split across parallel
// processes (docs/TESTING.md). Each package's test binary is compiled once;
// its top-level tests are listed with -test.list and dealt into shards, each
// run as its own process (and process group) with -test.run '^(A|B)$'. A
// shard passes only if its process exits 0 and every test dealt to it
// reports --- PASS or --- SKIP, so a test cannot quietly go unrun.
//
// It changes no test: every test of every package named runs, as in
// `go test PKG -count=1`, only in several processes. Tests that share state
// within one process are unaffected (each shard is a whole process, with
// its own TestMain); the full gate still runs each package as one process.
//
//   node scripts/lib/gotest.mjs [options] MODULE_DIR:PKG...
//     e.g. . :./app  gui:.
//   --race            build with -race
//   --jobs N          parallel processes (default: CPUs - 1)
//   --timeout SECS    each shard's -test.timeout (default 1800)
//   --work DIR        where binaries and logs go (default: a temporary directory)
//   --timings FILE    measured times (JSON) used to balance shards, and updated
//   --baseline FILE   times to use when FILE has none (read only)
//
// Exit status: 0 every test passed, 1 a test or a build failed, 2 usage,
// 130/143 interrupted. Only processes it started are ever stopped.

import { spawn } from 'node:child_process'
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, existsSync, createWriteStream } from 'node:fs'
import { cpus, tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'

export function parseArgs(argv) {
  const o = { race: false, jobs: Math.max(1, cpus().length - 1), timeout: 1800, work: '', timings: '', baseline: '', targets: [] }
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i]
    const val = () => { if (i + 1 >= argv.length) throw new Error(`${a} needs a value`); return argv[++i] }
    if (a === '--race') o.race = true
    else if (a === '--jobs') o.jobs = +val()
    else if (a === '--timeout') o.timeout = +val()
    else if (a === '--work') o.work = val()
    else if (a === '--timings') o.timings = val()
    else if (a === '--baseline') o.baseline = val()
    else if (a.startsWith('-')) throw new Error(`unknown option ${a}`)
    else {
      const k = a.indexOf(':')
      if (k < 0) throw new Error(`target ${a} is not MODULE_DIR:PKG`)
      o.targets.push({ module: a.slice(0, k) || '.', pkg: a.slice(k + 1) })
    }
  }
  if (!(o.jobs >= 1) || !(o.timeout >= 1)) throw new Error('--jobs and --timeout must be positive')
  return o
}

// shardsFor deals tests (name -> seconds) into shards, longest first onto
// the lightest shard, aiming at no shard much longer than target seconds.
export function shardsFor(tests, target, maxShards) {
  const names = Object.keys(tests).sort((a, b) => tests[b] - tests[a] || (a < b ? -1 : 1))
  const total = names.reduce((s, n) => s + tests[n], 0)
  const k = Math.max(1, Math.min(maxShards, names.length, Math.ceil(total / Math.max(target, 0.001))))
  const shards = Array.from({ length: k }, () => ({ tests: [], weight: 0 }))
  for (const n of names) {
    const s = shards.reduce((m, x) => (x.weight < m.weight ? x : m))
    s.tests.push(n)
    s.weight += tests[n]
  }
  return shards.filter((s) => s.tests.length)
}

// results reads a shard's -test.v output: each top-level test's result and
// time. Subtests are indented and are not counted here.
export function results(output) {
  const out = {}
  for (const m of output.matchAll(/^--- (PASS|FAIL|SKIP): (\S+) \(([\d.]+)s\)/gm)) out[m[2]] = { result: m[1], secs: +m[3] }
  return out
}

const runRegex = (names) => `^(${names.join('|')})$`

function run(cmd, args, opts) {
  // Its own process group, so a stop reaches what it started and nothing else.
  return new Promise((done) => {
    const p = spawn(cmd, args, { ...opts, detached: true, stdio: ['ignore', 'pipe', 'pipe'] })
    live.add(p)
    let out = ''
    const log = opts.log ? createWriteStream(opts.log) : null
    const take = (d) => { out += d; log?.write(d) }
    p.stdout.on('data', take)
    p.stderr.on('data', take)
    p.on('error', (e) => { take(`cannot run ${cmd}: ${e.message}\n`) })
    p.on('close', (code, sig) => { live.delete(p); log?.end(); done({ code: code ?? 128, sig, out }) })
  })
}

const live = new Set()
function stopAll() {
  for (const p of live) { try { process.kill(-p.pid, 'SIGTERM') } catch {} }
  setTimeout(() => { for (const p of live) { try { process.kill(-p.pid, 'SIGKILL') } catch {} } }, 3000).unref()
}

async function pool(jobs, n, fn) {
  let next = 0
  await Promise.all(Array.from({ length: Math.min(n, jobs.length) }, async () => {
    while (next < jobs.length) await fn(jobs[next++])
  }))
}

const fmt = (s) => `${Math.floor(s / 60)}m${String(Math.round(s % 60)).padStart(2, '0')}s`

async function main() {
  let o
  try { o = parseArgs(process.argv.slice(2)) } catch (e) { console.error(`gotest: ${e.message}`); process.exit(2) }
  if (!o.targets.length) { console.error('gotest: no packages'); process.exit(2) }
  for (const [sig, code] of [['SIGINT', 130], ['SIGTERM', 143], ['SIGHUP', 129]]) {
    process.on(sig, () => { console.log(`gotest: interrupted (${sig}), stopping its own processes`); stopAll(); setTimeout(() => process.exit(code), 3500) })
  }
  const work = resolve(o.work || mkdtempSync(join(tmpdir(), 'istary-gotest-')))
  mkdirSync(join(work, 'bin'), { recursive: true })
  mkdirSync(join(work, 'logs'), { recursive: true })
  let known = {}
  for (const f of [o.timings, o.baseline]) {
    if (f && existsSync(f) && !Object.keys(known).length) { try { known = JSON.parse(readFileSync(f, 'utf8')) } catch {} }
  }
  const kind = o.race ? 'race' : 'test'
  const t0 = Date.now()

  // The packages the targets name (a pattern such as ./... names several).
  const listed = []
  let failed = false
  for (const t of o.targets) {
    const moduleDir = resolve(t.module)
    const r = await run('go', ['list', '-f', '{{.Dir}}|{{.ImportPath}}', t.pkg], { cwd: moduleDir })
    if (r.code !== 0) { failed = true; console.log(`FAIL list ${t.module}:${t.pkg}\n${r.out}`); continue }
    for (const line of r.out.trim().split('\n').filter(Boolean)) {
      const [dir, importPath] = line.split('|')
      if (!listed.some((x) => x.importPath === importPath)) listed.push({ moduleDir, dir, importPath })
    }
  }

  // Compile each package's test binary, and list its tests.
  const pkgs = []
  await pool(listed, o.jobs, async ({ moduleDir, dir, importPath }) => {
    const bin = join(work, 'bin', importPath.replace(/[^A-Za-z0-9]+/g, '_') + (o.race ? '.race' : '') + '.test')
    const b = await run('go', ['test', '-c', '-o', bin, ...(o.race ? ['-race'] : []), importPath], { cwd: moduleDir })
    if (b.code !== 0) { failed = true; console.log(`FAIL build ${importPath}\n${b.out}`); return }
    if (!existsSync(bin)) { console.log(`  ${importPath}: no test files`); return }
    const l = await run(bin, ['-test.list', '.*'], { cwd: dir })
    if (l.code !== 0) { failed = true; console.log(`FAIL list tests ${importPath}\n${l.out}`); return }
    const names = l.out.split('\n').map((s) => s.trim()).filter((s) => /^(Test|Example|Fuzz)\w*$/.test(s))
    pkgs.push({ importPath, dir, bin, names })
  })
  pkgs.sort((a, b) => (a.importPath < b.importPath ? -1 : 1))
  const tBuild = (Date.now() - t0) / 1000
  if (failed) { console.log(`gotest: build or list failed after ${fmt(tBuild)}`); process.exit(1) }

  // Deal the tests into shards: unmeasured tests count as the package's
  // median (or 1 s), and no shard is aimed at much over total/jobs.
  const weigh = (p) => {
    // Race times, until measured, are in proportion to the ordinary ones.
    const m = known[kind]?.[p.importPath] ?? known.test?.[p.importPath] ?? {}
    const seen = Object.values(m).sort((a, b) => a - b)
    const dflt = seen.length ? seen[Math.floor(seen.length / 2)] : 1
    return Object.fromEntries(p.names.map((n) => [n, Math.max(0.05, m[n] ?? dflt)]))
  }
  const weights = pkgs.map(weigh)
  const total = weights.reduce((s, w) => s + Object.values(w).reduce((a, b) => a + b, 0), 0)
  const target = Math.max(total / o.jobs, 1)
  const jobs = []
  pkgs.forEach((p, i) => {
    for (const s of shardsFor(weights[i], target, o.jobs * 2)) jobs.push({ pkg: p, ...s })
  })
  jobs.sort((a, b) => b.weight - a.weight)
  const testCount = pkgs.reduce((s, p) => s + p.names.length, 0)
  console.log(`gotest: ${testCount} tests in ${pkgs.length} packages, ${jobs.length} shards on ${o.jobs} processes${o.race ? ', race detector' : ''} (built in ${fmt(tBuild)})`)

  // Run the shards.
  const measured = {}
  const problems = []
  let n = 0
  await pool(jobs, o.jobs, async (j) => {
    const id = String(++n).padStart(3, '0')
    const log = join(work, 'logs', `${id}-${j.pkg.importPath.replace(/[^A-Za-z0-9]+/g, '_')}.log`)
    const args = ['-test.run', runRegex(j.tests), '-test.count=1', '-test.v', `-test.timeout=${o.timeout}s`]
    const r = await run(j.pkg.bin, args, { cwd: j.pkg.dir, log })
    const got = results(r.out)
    const missing = j.tests.filter((t) => !got[t])
    const bad = j.tests.filter((t) => got[t] && got[t].result === 'FAIL')
    for (const [name, x] of Object.entries(got)) ((measured[j.pkg.importPath] ??= {})[name] = x.secs)
    if (r.code !== 0 || missing.length || bad.length) {
      problems.push({ pkg: j.pkg.importPath, log, code: r.code, sig: r.sig, missing, bad, tail: r.out.split('\n').slice(-30).join('\n') })
    }
  })
  const wall = (Date.now() - t0) / 1000

  if (o.timings) {
    const merged = { ...known, [kind]: { ...(known[kind] ?? {}) } }
    for (const [pkg, m] of Object.entries(measured)) merged[kind][pkg] = { ...(merged[kind][pkg] ?? {}), ...m }
    try { mkdirSync(dirname(o.timings), { recursive: true }); writeFileSync(o.timings, JSON.stringify(merged, null, 1) + '\n') } catch {}
  }

  // The report: per package its test time, and the slowest tests.
  for (const p of pkgs) {
    const m = measured[p.importPath] ?? {}
    const sum = Object.values(m).reduce((a, b) => a + b, 0)
    console.log(`  ${p.importPath}: ${p.names.length} tests, ${fmt(sum)} of test time`)
  }
  const slow = Object.entries(measured).flatMap(([pkg, m]) => Object.entries(m).map(([t, s]) => [s, `${pkg.split('/').pop()}.${t}`]))
    .sort((a, b) => b[0] - a[0]).slice(0, 5)
  if (slow.length) console.log(`  slowest: ${slow.map(([s, t]) => `${t} ${s.toFixed(1)}s`).join(', ')}`)
  for (const pr of problems) {
    console.log(`FAIL ${pr.pkg} (exit ${pr.code}${pr.sig ? ' ' + pr.sig : ''})  log: ${pr.log}`)
    if (pr.bad.length) console.log(`  failed: ${pr.bad.join(' ')}`)
    if (pr.missing.length) console.log(`  no result: ${pr.missing.join(' ')}`)
    console.log(pr.tail.replace(/^/gm, '  | '))
  }
  console.log(`gotest: ${problems.length ? 'FAIL' : 'PASS'} in ${fmt(wall)} (logs: ${work}/logs)`)
  process.exit(problems.length ? 1 : 0)
}

if (import.meta.url === `file://${process.argv[1]}`) main()
