// Istary M2 combat prototype: checks for the difficult-case fixture.
//
// NOT production code and not part of the application. Run with
// `node fixture-check.js`. It checks:
//   1. the fixture's port of sortPartyByStatus gives the same order as the Go
//      engine for the cases a scratch Go probe ran (output recorded below);
//   2. the Go-timing preview reproduces the probe's finding: an order stays
//      with its slot while the re-sort moves the members;
//   3. every outcome the fixture reports for a block is stated by that
//      block's own message lines (no result is shown without its message);
//   4. playback pacing with and without the death animations.
'use strict'
global.clone = (o) => JSON.parse(JSON.stringify(o))
const CS = require('./combat-scene.js')
const { GO_HARD, sortPartyByStatus, ORDERS_H } = require('./combat-fixture-hard.js')

let failed = 0
const check = (ok, what) => { console.log(`${ok ? 'ok  ' : 'FAIL'} ${what}`); if (!ok) failed++ }

// 1. Sort port against Go (go test probe, 2026-11-06, engine/combat.go:585).
const GO_SORT = [
  ['A:OK B:PARALYZE C:OK D:OK E:OK F:OK', 'A C D E F B'],
  ['A:OK B:DEAD C:PARALYZE D:OK E:OK F:OK', 'A D E F C B'],
  ['A:ASLEEP B:OK C:OK D:OK E:OK F:OK', 'B C D E F A'],
  ['A:DEAD B:OK C:OK D:ASHED E:OK F:DEAD', 'B C E A F D'],
  // The fixture's own round-1 outcome (TORVALD dead, YSOLDE paralyzed).
  ['T:DEAD Y:PARALYZE B:OK P:OK I:OK C:OK', 'B P I C Y T'],
]
for (const [input, want] of GO_SORT.slice(0, 4)) {
  const got = sortPartyByStatus(input.split(' ').map((x) => { const [name, st] = x.split(':'); return { name, st } })).map((m) => m.name).join(' ')
  check(got === want, `sort ${input} -> ${got} (Go: ${want})`)
}
{
  const [input, want] = GO_SORT[4]
  const got = sortPartyByStatus(input.split(' ').map((x) => { const [name, st] = x.split(':'); return { name, st } })).map((m) => m.name).join(' ')
  check(got === want, `sort ${input} -> ${got} (same algorithm; not run in Go)`)
}

// 2. Go timing: the order given for slot 3 is carried out by whoever is in
// slot 3 after the re-sort (probe: CORA's Run carried out by DREW).
{
  GO_HARD.reset(); GO_HARD.resortAt = 'go'
  GO_HARD.st.team = ['ALDA', 'BRAM', 'CORA', 'DREW', 'EDDA', 'FINN'].map((name, i) => ({ name, slot: i + 1, st: i === 1 ? 'DEAD' : 'OK' }))
  const orders = [{ act: 'parry' }, null, { act: 'run' }, { act: 'parry' }, { act: 'parry' }, { act: 'parry' }]
  const rows = GO_HARD.goPreview(orders)
  check(rows[2].enteredFor === 'CORA' && rows[2].by === 'DREW' && rows[2].order.act === 'run', `Go timing: slot 3 order given for ${rows[2].enteredFor} is carried out by ${rows[2].by} (probe: DREW)`)
  GO_HARD.resortAt = 'classic'
}

// 3. Play the fixture with its default orders (Classic timing) and check
// that each block's outcomes are stated by its lines.
GO_HARD.reset()
const rounds = []
for (let r = 1; r <= 6 && !GO_HARD.st.result; r++) {
  const moved = GO_HARD.beginRound()
  const team = GO_HARD.st.team.map((m) => m.name)
  const blocks = GO_HARD.commit(ORDERS_H[GO_HARD.st.round].map((o, i) => (o && GO_HARD.canAct(i + 1).ok ? o : null)))
  rounds.push({ r, moved, team, blocks, looks: GO_HARD.st.groups.map((g) => g.look) })
}
check(GO_HARD.st.result === 'victory', `encounter ends in ${GO_HARD.st.result} after ${rounds.length} rounds`)
check(GO_HARD.st.team.find((m) => m.name === 'TORVALD').st === 'DEAD' && GO_HARD.st.team.length === 6, 'the dead member is still one of the six')
check(rounds[1].moved.length === 6 && rounds[1].team.join(' ') === 'BRANNOC PELL ILSA CORVIN YSOLDE TORVALD', `round 2 lines after the round-start re-sort: ${rounds[1].team.join(' ')}`)
check(rounds[2].moved.length === 0, 'round 3: the cured member stays in the back line (no re-sort move)')
let stated = true
for (const { r, blocks } of rounds) for (const b of blocks) {
  if (b.silent) continue
  const text = b.lines.join(' ')
  for (const d of b.deltas || []) {
    const ok = d.k === 'hp' ? /DAMAGE|HEALED/.test(text)
      : d.k === 'cond' ? (d.st === 'DEAD' ? /SLAIN/.test(text) : d.st === 'PARALYZE' ? /PARALYZED/.test(text) : d.st === 'OK' ? /CURED/.test(text) : false)
      : d.k === 'fig' ? (d.to === 'dead' ? /DIES|KILLS ONE|DISPELLED/.test(text) : d.to === 'asleep' ? /ASLEEP/.test(text) : false)
      : false
    if (!ok) { stated = false; console.log(`     round ${r}: "${text}" does not state ${JSON.stringify(d)}`) }
  }
}
check(stated, 'every reported outcome is stated by its own block’s message')

// 4. Pacing: the page's automatic timing (combat-refined.html, advance()).
// An action holds for the reading time, or until the death animation it
// reveals has finished, whichever is longer.
const IMPACT = 0.42, PARTY_DEATH_MS = 1100
function pace(speed) {
  const ms = 950 / speed
  const out = []
  for (const { r, blocks } of rounds) {
    let base = 0, withDeaths = 0, extended = 0, deaths = 0
    const looks = GO_HARD.st.groups.map((g) => g.look)
    for (const b of blocks) {
      if (b.silent) continue
      const dies = (b.deltas || []).flatMap((d) => (d.k === 'fig' && d.to === 'dead' ? [CS.DEATH_MS[looks[d.gi]]] : d.k === 'cond' && d.st === 'DEAD' ? [PARTY_DEATH_MS] : []))
      const read = ms + (900 + 420 * b.lines.length) / speed
      const death = dies.length ? ms * IMPACT + Math.max(...dies) + 120 * (dies.length - 1) + 350 : 0
      base += read; withDeaths += Math.max(read, death); deaths += dies.length
      if (death > read) extended++
    }
    out.push({ r, actions: blocks.filter((b) => !b.silent).length, deaths, base, withDeaths, extended })
  }
  return out
}
for (const speed of [1, 1.8]) {
  console.log(`\nAutomatic playback at speed ${speed} (${speed === 1 ? 'normal' : 'fast'}):`)
  console.log('  round  actions  deaths  without deaths  with deaths  actions lengthened')
  for (const p of pace(speed)) console.log(`  ${String(p.r).padStart(5)}  ${String(p.actions).padStart(7)}  ${String(p.deaths).padStart(6)}  ${(p.base / 1000).toFixed(1).padStart(13)} s  ${(p.withDeaths / 1000).toFixed(1).padStart(9)} s  ${String(p.extended).padStart(18)}`)
}

console.log(failed ? `\n${failed} check(s) failed` : '\nAll checks passed')
process.exitCode = failed ? 1 : 0
