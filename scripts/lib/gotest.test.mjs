// Tests of scripts/lib/gotest.mjs: how it deals tests into shards, how it
// reads results, and, on a throwaway Go module, that a failing test, a test
// that ends the process early, and a broken build each fail the run.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { execFileSync, spawnSync } from 'node:child_process'
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { shardsFor, results, parseArgs } from './gotest.mjs'

const GOTEST = join(dirname(fileURLToPath(import.meta.url)), 'gotest.mjs')

test('every test is dealt to exactly one shard, the longest first', () => {
  const tests = { A: 80, B: 60, C: 25, D: 14, E: 14, F: 1, G: 1, H: 1 }
  const shards = shardsFor(tests, 40, 8)
  const dealt = shards.flatMap((s) => s.tests).sort()
  assert.deepEqual(dealt, Object.keys(tests).sort())
  // The two long tests are apart; nothing waits behind the longest.
  assert.ok(shards.every((s) => !(s.tests.includes('A') && s.tests.includes('B'))))
  assert.equal(Math.max(...shards.map((s) => s.weight)), 80)
})

test('no more shards than tests or the cap', () => {
  assert.equal(shardsFor({ A: 1 }, 0.1, 8).length, 1)
  assert.equal(shardsFor({ A: 5, B: 5, C: 5 }, 1, 2).length, 2)
  assert.deepEqual(shardsFor({}, 1, 4), [])
})

test('only top-level results count, with their times', () => {
  const out = '=== RUN   TestA\n=== RUN   TestA/sub\n    --- FAIL: TestA/sub (0.00s)\n--- FAIL: TestA (1.50s)\n--- PASS: TestB (0.20s)\n--- SKIP: TestC (0.00s)\nFAIL\n'
  assert.deepEqual(results(out), { TestA: { result: 'FAIL', secs: 1.5 }, TestB: { result: 'PASS', secs: 0.2 }, TestC: { result: 'SKIP', secs: 0 } })
})

test('arguments', () => {
  const o = parseArgs(['--race', '--jobs', '3', '.:./app', 'gui:.'])
  assert.equal(o.race, true)
  assert.equal(o.jobs, 3)
  assert.deepEqual(o.targets, [{ module: '.', pkg: './app' }, { module: 'gui', pkg: '.' }])
  assert.throws(() => parseArgs(['./app']), /MODULE_DIR:PKG/)
  assert.throws(() => parseArgs(['--jobs', '0', '.:x']), /positive/)
})

// A module with one package whose tests are chosen by the files written.
function module(files) {
  const dir = mkdtempSync(join(tmpdir(), 'istary-gotest-test-'))
  writeFileSync(join(dir, 'go.mod'), 'module example.com/m\n\ngo 1.21\n')
  mkdirSync(join(dir, 'p'))
  writeFileSync(join(dir, 'p', 'p.go'), 'package p\n\nfunc One() int { return 1 }\n')
  for (const [name, body] of Object.entries(files)) writeFileSync(join(dir, 'p', name), body)
  return dir
}
const goTest = (body) => `package p\n\nimport (\n\t"os"\n\t"testing"\n)\n\nvar _ = os.Exit\n\n${body}\n`
const passing = Array.from({ length: 12 }, (_, i) => `func TestPass${i}(t *testing.T) { if One() != 1 { t.Fatal("no") } }`).join('\n')
function runIn(dir, extra = []) {
  const r = spawnSync('node', [GOTEST, '--jobs', '4', '--work', join(dir, 'work'), ...extra, `${dir}:./p`], { encoding: 'utf8' })
  return { code: r.status, out: r.stdout + r.stderr }
}
const haveGo = (() => { try { execFileSync('go', ['version']); return true } catch { return false } })()

test('a module whose tests all pass passes, every test run', { skip: !haveGo }, () => {
  const dir = module({ 'p_test.go': goTest(passing) })
  try {
    const r = runIn(dir)
    assert.equal(r.code, 0, r.out)
    assert.match(r.out, /12 tests in 1 packages/)
    assert.match(r.out, /gotest: PASS/)
  } finally { rmSync(dir, { recursive: true, force: true }) }
})

test('one failing test fails the run and is named', { skip: !haveGo }, () => {
  const dir = module({ 'p_test.go': goTest(passing + '\nfunc TestBroken(t *testing.T) { t.Fatal("broken") }') })
  try {
    const r = runIn(dir)
    assert.equal(r.code, 1, r.out)
    assert.match(r.out, /failed: TestBroken/)
  } finally { rmSync(dir, { recursive: true, force: true }) }
})

test('a test that ends the process early leaves its shard-mates without a result, and fails', { skip: !haveGo }, () => {
  const dir = module({ 'p_test.go': goTest(passing + '\nfunc TestExits(t *testing.T) { os.Exit(0) }') })
  try {
    const r = runIn(dir, ['--jobs', '1'])
    assert.equal(r.code, 1, r.out)
    assert.match(r.out, /no result: /)
  } finally { rmSync(dir, { recursive: true, force: true }) }
})

test('a package that does not build fails the run', { skip: !haveGo }, () => {
  const dir = module({ 'p_test.go': goTest('func TestX(t *testing.T) { undefined() }') })
  try {
    const r = runIn(dir)
    assert.equal(r.code, 1, r.out)
    assert.match(r.out, /FAIL build/)
  } finally { rmSync(dir, { recursive: true, force: true }) }
})
