// Istary M2 combat prototype: the difficult-case fixture.
//
// NOT production code and not part of the application. A second stand-in for
// the Go controller with the same interface as GO in combat-fixture.js, used
// by combat-refined.html to validate the presentation model against harder
// situations: four monster groups including a large group, sleeping monsters,
// a front-line member paralyzed, a member killed, the formation change that
// follows, a paralysis cured, and the encounter completed with a dead member
// still in the team. Values and outcomes are scripted for one illustrative
// encounter; nothing here is a rule the GUI would hold.
//
// The one piece of real engine behaviour reproduced exactly is the party
// re-sort, a port of sortPartyByStatus (engine/combat.go:585), because the
// presentation of formation changes depends on its result and its timing.
// fixture-check.js tests the port against the Go output.
'use strict'

// Status order as engine/character.go:61 (OK < ASLEEP < AFRAID < PARALYZE <
// STONED < DEAD < ASHED < LOST). Names as StatusNames.
const STATUS_RANK = { OK: 0, ASLEEP: 1, AFRAID: 2, PARALYZE: 3, STONED: 4, DEAD: 5, ASHED: 6, LOST: 7 }
// Port of sortPartyByStatus: an exchange sort, not stable, that moves any
// worse status (not only death) behind better ones.
function sortPartyByStatus(members) {
  const n = members.length
  for (let i = 0; i < n - 1; i++) {
    for (let j = i + 1; j < n; j++) {
      const si = members[i] ? STATUS_RANK[members[i].st] : 99, sj = members[j] ? STATUS_RANK[members[j].st] : 99
      if (si > sj) { const t = members[i]; members[i] = members[j]; members[j] = t }
    }
  }
  return members
}
// engine IsAlive (engine/character.go:306): OK, ASLEEP or AFRAID. Only these
// members are offered an order (findNextActor, cmd/istary/main.go:4097).
const canActSt = (st) => st === 'OK' || st === 'ASLEEP' || st === 'AFRAID'

const TEAM_H = [
  { name: 'TORVALD', cls: 'G-FIG', lvl: 5, ac: 2, hp: 34, max: 34, st: 'OK', color: '#5f80a6', items: ['LONG SWORD', 'CHAIN MAIL', 'LARGE SHIELD'] },
  { name: 'YSOLDE', cls: 'N-FIG', lvl: 4, ac: 3, hp: 31, max: 31, st: 'OK', color: '#7c6fa6', items: ['BATTLE AXE', 'CHAIN MAIL'] },
  { name: 'BRANNOC', cls: 'G-PRI', lvl: 5, ac: 4, hp: 22, max: 24, st: 'OK', color: '#a8935a', items: ['MACE', 'LEATHER ARMOR', 'SMALL SHIELD'], sp: { 1: 3, 2: 2, 3: 1 } },
  { name: 'PELL', cls: 'N-THI', lvl: 4, ac: 6, hp: 14, max: 15, st: 'OK', color: '#5a9670', items: ['SHORT SWORD', 'LEATHER ARMOR'] },
  { name: 'ILSA', cls: 'G-MAG', lvl: 3, ac: 9, hp: 9, max: 11, st: 'OK', color: '#a66262', items: ['DAGGER', 'ROBES'], sp: { 1: 2 } },
  { name: 'CORVIN', cls: 'G-BIS', lvl: 3, ac: 8, hp: 10, max: 10, st: 'OK', color: '#6f9aa0', items: ['STAFF', 'ROBES', 'ROD OF FLAME (equipped)'], sp: { 1: 1 } },
]
// Keyed by name, not slot: in this fixture members change places.
const SPELLBOOK_H = {
  BRANNOC: [{ name: 'DIOS', lvl: 1, tgt: 'ally' }, { name: 'BADIOS', lvl: 1, tgt: 'group' }, { name: 'MILWA', lvl: 1, tgt: 'none' }, { name: 'DIALKO', lvl: 3, tgt: 'ally' }, { name: 'CALFO', lvl: 2, tgt: 'none', why: 'CALFO is cast on a chest, not in combat' }],
  ILSA: [{ name: 'HALITO', lvl: 1, tgt: 'group' }, { name: 'KATINO', lvl: 1, tgt: 'group' }, { name: 'DUMAPIC', lvl: 1, tgt: 'none', why: 'DUMAPIC has no use in combat' }],
  CORVIN: [{ name: 'KATINO', lvl: 1, tgt: 'group' }],
}
// look: the creature's appearance, which the team can already see; it picks
// the figure and its death style (presentation only, X26).
const GROUPS_H = [
  { unk: ['SMALL HUMANOID', 'SMALL HUMANOIDS'], id: ['BOGLIN', 'BOGLINS'], n: 7, look: 'humanoid', h: 1.15, color: '#6b7a4e', undead: false, resist: [3, 6] },
  { unk: ['SKELETAL FIGURE', 'SKELETAL FIGURES'], id: ['BONE WALKER', 'BONE WALKERS'], n: 2, look: 'skeleton', h: 1.55, color: '#9a927e', undead: true },
  { unk: ['WINGED CREATURE', 'WINGED CREATURES'], id: ['GLOOM WING', 'GLOOM WINGS'], n: 3, look: 'flying', h: 0.75, alt: 1.5, color: '#5b5f86', undead: false },
  { unk: ['HUGE FIGURE', 'HUGE FIGURES'], id: ['CAVE HULK', 'CAVE HULKS'], n: 1, look: 'large', h: 2.45, color: '#7d5f4a', undead: false, hp: 24 },
]
// Monster turns in each round's resolved order, scripted. Party entries are
// slot tokens ('p3'), as Go's initiative slots are per position. A monster
// entry acts only if that figure is awake and alive; `vs` is its pre-assigned
// victim position (0-2), with the engine's fallback to the next living front
// member (engine/combat.go:1130).
const SCRIPT_H = {
  1: ['p5', { g: 2, sub: 0, vs: 0, dmg: 3 }, 'p1', { g: 1, sub: 0, vs: 1, dmg: 4, para: true }, { g: 0, sub: 3, vs: 2, dmg: 0 }, 'p6', 'p2', { g: 3, sub: 0, vs: 0, dmg: 31 }, 'p3', { g: 1, sub: 1, vs: 2, dmg: 2 }, { g: 2, sub: 1, vs: 2, dmg: 0 }, 'p4', { g: 0, sub: 6, vs: 1, dmg: 3 }],
  2: [{ g: 3, sub: 0, vs: 2, dmg: 0 }, 'p1', 'p4', { g: 1, sub: 0, vs: 1, dmg: 3 }, 'p2', { g: 0, sub: 2, vs: 0, dmg: 2 }, 'p3', { g: 1, sub: 1, vs: 0, dmg: 0 }, { g: 2, sub: 0, vs: 0, dmg: 2 }, { g: 2, sub: 1, vs: 1, dmg: 0 }, 'p5', 'p6'],
  3: ['p1', { g: 3, sub: 0, vs: 1, dmg: 5 }, 'p2', 'p3', { g: 1, sub: 0, vs: 0, dmg: 2 }, 'p4', 'p5', { g: 1, sub: 1, vs: 2, dmg: 0 }, 'p6'],
  4: ['p2', { g: 3, sub: 0, vs: 0, dmg: 4 }, 'p1', 'p3', 'p4', 'p5', 'p6'],
}
// Which groups Go's per-round identification roll succeeds for (silent).
const IDENT_H = { 2: [0, 1], 3: [2], 4: [3] }
// The default orders the reviewer jumps use, by position after each re-sort.
const ORDERS_H = {
  1: [{ slot: 1, act: 'fight', tg: 0 }, { slot: 2, act: 'fight', tg: 1 }, { slot: 3, act: 'spell', spell: 'BADIOS', tg: 2 }, { slot: 4, act: 'parry' }, { slot: 5, act: 'spell', spell: 'KATINO', tg: 0 }, { slot: 6, act: 'use', item: 'ROD OF FLAME', tg: 0 }],
  2: [{ slot: 1, act: 'spell', spell: 'DIALKO', ally: 5 }, { slot: 2, act: 'fight', tg: 0 }, { slot: 3, act: 'spell', spell: 'HALITO', tg: 3 }, { slot: 4, act: 'spell', spell: 'KATINO', tg: 2 }, null, null],
  3: [{ slot: 1, act: 'dispel', tg: 1 }, { slot: 2, act: 'fight', tg: 0 }, { slot: 3, act: 'fight', tg: 0 }, { slot: 4, act: 'use', item: 'ROD OF FLAME', tg: 2 }, { slot: 5, act: 'parry' }, null],
  4: [{ slot: 1, act: 'fight', tg: 3 }, { slot: 2, act: 'fight', tg: 3 }, { slot: 3, act: 'parry' }, { slot: 4, act: 'parry' }, { slot: 5, act: 'parry' }, null],
}
const DMG_H = { TORVALD: 11, YSOLDE: 9, BRANNOC: 9, PELL: 7, ILSA: 3, CORVIN: 3 }

const GO_HARD = {
  st: null,
  // 'classic': the re-sort happens at the start of the round, before order
  // entry, as Classic's DSPPARTY in CUTIL (see the comments at
  // engine/combat.go:225, 280, 634). 'go': where the Go engine does it today,
  // at the start of ExecuteRound, after the orders were entered by slot.
  resortAt: 'classic',
  reset() {
    const team = clone(TEAM_H).map((m, i) => Object.assign(m, { slot: i + 1 }))
    const groups = clone(GROUPS_H).map((g) => Object.assign(g, { figs: Array.from({ length: g.n }, () => 'ok'), asleep: 0, known: false }))
    this.st = { round: 1, team, groups, result: null, chest: null, lastOrders: Array(6).fill(null), sortedFor: 1 }
  },
  base(slot) { return this.st.team[slot - 1] },
  name(gi, plural) { const g = this.st.groups[gi]; return (g.known ? g.id : g.unk)[plural ? 1 : 0] },
  alive(gi) { return this.st.groups[gi] && this.st.groups[gi].n > 0 },
  aliveGroups() { return this.st.groups.map((g, i) => i).filter((i) => this.alive(i)) },
  canAct(slot) { const m = this.st.team[slot - 1]; return { ok: canActSt(m.st), why: m.st === 'PARALYZE' ? 'paralyzed' : m.st === 'DEAD' ? 'dead' : m.st.toLowerCase() } },
  menu(slot) {
    const m = this.st.team[slot - 1], back = slot > 3
    const caster = !!SPELLBOOK_H[m.name]
    const canDispel = m.cls.endsWith('PRI')
    const useItem = m.name === 'CORVIN' ? 'ROD OF FLAME' : null
    return [
      { key: 'F', act: 'fight', label: 'Fight', ok: !back, why: back ? 'Back line (4–6) cannot fight' : '' },
      { key: 'P', act: 'parry', label: 'Parry', ok: true },
      { key: 'S', act: 'spell', label: 'Spell', ok: caster, why: caster ? '' : `${m.name} knows no spells` },
      useItem ? { key: 'U', act: 'use', label: `Use ${useItem}`, ok: true, item: useItem } : { key: 'U', act: 'use', label: 'Use', ok: true, note: 'nothing usable; counts as Parry' },
      { key: 'D', act: 'dispel', label: 'Dispel', ok: canDispel, why: canDispel ? '' : m.cls.endsWith('BIS') ? 'Bishops dispel from level 4 (Corvin is level 3)' : `A ${m.cls.slice(2) === 'FIG' ? 'fighter' : m.cls.slice(2) === 'THI' ? 'thief' : 'mage'} cannot dispel` },
      { key: 'R', act: 'run', label: 'Run', ok: true },
    ]
  },
  spells(slot) {
    const m = this.st.team[slot - 1]
    return (SPELLBOOK_H[m.name] || []).map((s) => {
      const left = (m.sp || {})[s.lvl] || 0
      const why = s.why || (left ? '' : `No level ${s.lvl} spell points left`)
      return Object.assign({}, s, { left, ok: !why, why })
    })
  },
  targetKind(o) {
    if (o.act === 'fight' || o.act === 'dispel' || o.act === 'use') return 'group'
    if (o.act === 'spell') return (SPELLBOOK_H[this.st.team[o.slot - 1].name].find((s) => s.name === o.spell) || {}).tgt
    return 'none'
  },

  // Classic timing: called when a new round's order entry begins. Re-sorts
  // and reports who moved (requirement E5). Returns [] in round 1 and when
  // nothing moved.
  beginRound() {
    const st = this.st
    if (this.resortAt !== 'classic' || st.sortedFor === st.round) return []
    st.sortedFor = st.round
    return this._resort()
  },
  _resort() {
    const before = this.st.team.map((m) => m.name)
    sortPartyByStatus(this.st.team)
    this.st.team.forEach((m, i) => { m.slot = i + 1 })
    return this.st.team.map((m, i) => ({ name: m.name, from: before.indexOf(m.name) + 1, to: i + 1, st: m.st })).filter((x) => x.from !== x.to)
  },
  // Go timing, for the reviewer only: what ExecuteRound would do with these
  // orders. Verified part (scratch probe, see COMBAT-DIFFICULT-CASES.md): an
  // order stays with its slot while the re-sort moves the members. Read from
  // the code, not run: slots that were skipped this round keep the previous
  // round's order. What a member does with another member's order (a spell
  // it does not know) is not modelled. Does not change the fixture state.
  goPreview(orders) {
    const team = this.st.team.map((m) => Object.assign({}, m)), before = team.map((m) => m.name)
    sortPartyByStatus(team)
    return team.map((m, i) => {
      const fresh = orders[i], stale = !fresh && this.st.lastOrders[i]
      return { slot: i + 1, by: m.name, st: m.st, enteredFor: before[i], order: fresh || stale || null, stale: !!stale, skipped: !canActSt(m.st) }
    })
  },

  commit(orders) {
    const st = this.st, r = st.round, blocks = [], team = st.team, G = st.groups
    st.lastOrders = orders.map((o) => (o ? clone(o) : null))
    const nm = (gi, pl) => this.name(gi, pl)
    const art = (s) => (/^[AEIOU]/.test(s) ? 'AN ' : 'A ') + s
    const recount = (gi) => { const g = G[gi]; g.n = g.figs.filter((f) => f !== 'dead').length; g.asleep = g.figs.filter((f) => f === 'asleep').length }
    const fig = (gi, idx, to) => { G[gi].figs[idx] = to; recount(gi); return { k: 'fig', gi, idx, to } }
    const firstAlive = () => G.findIndex((g) => g.n > 0)
    const tgFor = (gi) => (this.alive(gi) ? gi : firstAlive())
    const pick = (gi) => { const f = G[gi].figs; const a = f.indexOf('asleep'); return a >= 0 ? a : f.indexOf('ok') }
    const killOne = (gi, dmg) => {
      const g = G[gi]
      if (g.hp != null) { g.hp -= dmg; if (g.hp > 0) return null }
      const idx = pick(gi); return idx >= 0 ? fig(gi, idx, 'dead') : null
    }
    if (IDENT_H[r]) {
      const ids = IDENT_H[r].filter((gi) => !G[gi].known && G[gi].n > 0)
      if (ids.length) { ids.forEach((gi) => { G[gi].known = true }); blocks.push({ silent: true, lines: [], deltas: ids.map((gi) => ({ k: 'name', gi })), note: 'Go’s per-round identification: these group names update (no message).' }) }
    }
    for (const tok of SCRIPT_H[r] || []) {
      if (G.every((g) => g.n === 0)) break
      if (typeof tok === 'string') {
        const slot = +tok[1], m = team[slot - 1], o = orders[slot - 1]
        // Paralyzed or dead members do nothing and show nothing
        // (executePartySlot skips !IsAlive, engine/combat.go:1036).
        if (!o || !canActSt(m.st)) continue
        const A = 'p' + slot
        if (o.act === 'parry' || (o.act === 'use' && !o.item)) { blocks.push({ actor: A, lines: [`${m.name} PARRIED.`], fx: [{ type: 'parry', to: A }] }); continue }
        if (o.act === 'run') { blocks.push({ actor: A, lines: [`${m.name} FAILED TO RUN!`], fx: [] }); continue }
        if (o.act === 'fight') {
          const gi = tgFor(o.tg); if (gi < 0) continue
          const d = DMG_H[m.name], who = nm(gi)
          const lines = [`${m.name} ATTACKS ${art(who)}`, `AND HITS 1 TIMES FOR ${d} DAMAGE`]
          const dl = killOne(gi, d); if (dl) lines.push(`${m.name} KILLS ONE!`)
          blocks.push({ actor: A, tgt: 'g' + gi, lines, fx: [{ type: 'hit', from: A, to: 'g' + gi, text: '−' + d }], deltas: dl ? [dl] : [] })
          continue
        }
        if (o.act === 'dispel') {
          const gi = tgFor(o.tg); if (gi < 0) continue
          const g = G[gi], lines = [`${m.name} TRIES TO DISPEL`], deltas = []
          if (g.undead) g.figs.forEach((f, i) => { if (f !== 'dead') { lines.push(`${art(nm(gi))} IS DISPELLED!`); deltas.push(fig(gi, i, 'dead')) } })
          else lines.push(`${art(nm(gi))} IS UNAFFECTED!`)
          blocks.push({ actor: A, tgt: 'g' + gi, lines, fx: [{ type: 'dispel', from: A, to: 'g' + gi }], deltas })
          continue
        }
        if (o.act === 'use') {
          const gi = tgFor(o.tg); if (gi < 0) continue
          const lines = [`${m.name} USES A ${o.item}`], deltas = []
          for (let i = 0; i < 3; i++) { const dl = killOne(gi, 9); if (dl) { deltas.push(dl); lines.push(`${nm(gi)} DIES!`) } }
          if (!deltas.length) lines.push(`${art(nm(gi))} TAKES 9 DAMAGE`)
          blocks.push({ actor: A, tgt: 'g' + gi, lines, fx: [{ type: 'halito', from: A, to: 'g' + gi, text: '−9' }], deltas })
          continue
        }
        // Spells.
        const sp = SPELLBOOK_H[m.name].find((s) => s.name === o.spell)
        m.sp[sp.lvl]--
        const verb = `${m.name} CASTS ${sp.name}!`
        if (sp.name === 'HALITO' || sp.name === 'BADIOS') {
          const gi = tgFor(o.tg); if (gi < 0) continue
          const d = sp.name === 'HALITO' ? 8 : 7, dl = killOne(gi, d)
          blocks.push({ actor: A, tgt: 'g' + gi, lines: [verb, `${art(nm(gi))} TAKES ${d} DAMAGE`].concat(dl ? [`${nm(gi)} DIES!`] : []), fx: [{ type: 'halito', from: A, to: 'g' + gi, text: '−' + d }], deltas: dl ? [dl] : [] })
        } else if (sp.name === 'KATINO') {
          const gi = tgFor(o.tg); if (gi < 0) continue
          const g = G[gi], deltas = []
          if (!g.undead) g.figs.forEach((f, i) => { if (f === 'ok' && !(g.resist || []).includes(i)) deltas.push(fig(gi, i, 'asleep')) })
          const n = deltas.length
          blocks.push({ actor: A, tgt: 'g' + gi, lines: [verb, n ? `${n} ${nm(gi, n !== 1)} ${n === 1 ? 'IS' : 'ARE'} ASLEEP` : `THE ${nm(gi, true)} ARE NOT AFFECTED`], fx: [{ type: 'sleep', from: A, to: 'g' + gi }], deltas })
        } else if (sp.name === 'DIALKO') {
          // DIALKO cures PARALYZE or ASLEEP only (engine/combat.go:1843).
          const t = team[o.ally - 1], ok = t.st === 'PARALYZE' || t.st === 'ASLEEP'
          if (ok) t.st = 'OK'
          blocks.push({ actor: A, tgt: 'p' + o.ally, lines: [verb, ok ? `${t.name} IS CURED!` : `${t.name} IS NOT HELPED!`], fx: ok ? [{ type: 'heal', from: A, to: 'p' + o.ally }] : [], deltas: ok ? [{ k: 'cond', name: t.name, st: 'OK' }] : [] })
        } else if (sp.name === 'DIOS') {
          const t = team[o.ally - 1], heal = canActSt(t.st) ? Math.min(6, t.max - t.hp) : 0; t.hp += heal
          blocks.push({ actor: A, tgt: 'p' + o.ally, lines: [verb, `${t.name} IS HEALED FOR ${heal}`], fx: [{ type: 'heal', from: A, to: 'p' + o.ally, text: '+' + heal }], deltas: [{ k: 'hp', name: t.name, hp: t.hp }] })
        } else {
          blocks.push({ actor: A, lines: [verb, 'THE LIGHT GROWS BRIGHTER'], fx: [{ type: 'dispel', to: A }] })
        }
      } else {
        const g = G[tok.g]
        if (!g || g.figs[tok.sub] !== 'ok') continue
        let t = team[tok.vs]
        if (!t || !canActSt(t.st)) t = [0, 1, 2].map((i) => team[i]).find((x) => x && canActSt(x.st))
        if (!t) continue
        const slot = t.slot, who = nm(tok.g), verb = g.look === 'skeleton' ? 'CLAWS AT' : g.look === 'flying' ? 'SWOOPS AT' : g.look === 'large' ? 'SMASHES AT' : 'SWINGS AT'
        const lines = [`${art(who)} ${verb} ${t.name}`], deltas = []
        if (tok.dmg) {
          t.hp = Math.max(0, t.hp - tok.dmg); lines.push(`AND HITS FOR ${tok.dmg} DAMAGE!`); deltas.push({ k: 'hp', name: t.name, hp: t.hp })
          if (t.hp === 0) { t.st = 'DEAD'; lines.push(`${t.name} IS SLAIN!`); deltas.push({ k: 'cond', name: t.name, st: 'DEAD' }) }
          else if (tok.para) { t.st = 'PARALYZE'; lines.push(`${t.name} IS PARALYZED`); deltas.push({ k: 'cond', name: t.name, st: 'PARALYZE' }) }
        } else lines.push('AND MISSES')
        blocks.push({ actor: 'g' + tok.g, sub: tok.sub, tgt: 'p' + slot, lines, fx: [{ type: tok.dmg ? 'hit' : 'miss', from: 'g' + tok.g, to: 'p' + slot, text: tok.dmg ? '−' + tok.dmg : '' }], deltas })
      }
    }
    if (G.every((g) => g.n === 0)) {
      st.result = 'victory'
      blocks.push({ actor: null, lines: ['FOR KILLING THE MONSTERS', 'EACH SURVIVOR GETS 210 EXPERIENCE POINTS'], fx: [], end: true })
    } else if (team.every((m) => !canActSt(m.st))) st.result = 'lost'
    // Round end, as CompactGroups (engine/combat.go:92): the dead leave the
    // groups. Figures that remain keep their places.
    G.forEach((g) => { g.figs = g.figs.filter((f) => f !== 'dead') })
    if (!st.result) st.round++
    return blocks
  },
}
if (typeof module !== 'undefined' && module.exports) module.exports = { GO_HARD, sortPartyByStatus, ORDERS_H, TEAM_H, GROUPS_H }
