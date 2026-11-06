// Istary M2 combat prototype: the shared illustrative fixture.
//
// NOT production code and not part of the application. Moved out of
// combat.html so that combat.html and combat-refined.html play exactly the same
// encounter. The only addition is `sub` on monster blocks (which member of the
// group acts), used by the refined page's animation. Classic script (no
// module) so both pages open from file://.
'use strict'
// ════════════════════════════════════════════════════════════════════════
// FIXTURE CONTROLLER. Stands in for the Go controller. In the product, all
// of this (order menus and their reasons, targets, the resolved round, the
// chest results) comes from Go; nothing here is a rule the GUI would hold.
// Values are hard-coded for one illustrative encounter.
// ════════════════════════════════════════════════════════════════════════
const TEAM0 = [
  { slot: 1, name: 'TORVALD', cls: 'G-FIG', lvl: 5, ac: 2, hp: 34, max: 34, st: 'OK', color: '#5f80a6', items: ['LONG SWORD', 'CHAIN MAIL', 'LARGE SHIELD'] },
  { slot: 2, name: 'YSOLDE', cls: 'N-FIG', lvl: 4, ac: 3, hp: 31, max: 31, st: 'OK', color: '#7c6fa6', items: ['BATTLE AXE', 'CHAIN MAIL'] },
  { slot: 3, name: 'BRANNOC', cls: 'G-PRI', lvl: 4, ac: 4, hp: 18, max: 22, st: 'OK', color: '#a8935a', items: ['MACE', 'LEATHER ARMOR', 'SMALL SHIELD'], sp: { 1: 3, 2: 1 } },
  { slot: 4, name: 'PELL', cls: 'N-THI', lvl: 4, ac: 6, hp: 14, max: 15, st: 'OK', color: '#5a9670', items: ['SHORT SWORD', 'LEATHER ARMOR'] },
  { slot: 5, name: 'ILSA', cls: 'G-MAG', lvl: 3, ac: 9, hp: 9, max: 11, st: 'OK', color: '#a66262', items: ['DAGGER', 'ROBES'], sp: { 1: 2 } },
  { slot: 6, name: 'CORVIN', cls: 'G-BIS', lvl: 3, ac: 8, hp: 10, max: 10, st: 'OK', color: '#6f9aa0', items: ['STAFF', 'ROBES', 'SILVER ROD (equipped)'], sp: { 1: 1 } },
]
const SPELLBOOK = {
  3: [{ name: 'DIOS', lvl: 1, tgt: 'ally' }, { name: 'BADIOS', lvl: 1, tgt: 'group' }, { name: 'MILWA', lvl: 1, tgt: 'none' }, { name: 'PORFIC', lvl: 1, tgt: 'none' }, { name: 'CALFO', lvl: 2, tgt: 'none', why: 'CALFO is cast on a chest, not in combat' }],
  5: [{ name: 'HALITO', lvl: 1, tgt: 'group' }, { name: 'KATINO', lvl: 1, tgt: 'group' }, { name: 'DUMAPIC', lvl: 1, tgt: 'none', why: 'DUMAPIC has no use in combat' }],
  6: [{ name: 'KATINO', lvl: 1, tgt: 'group' }],
}
const GROUPS0 = [
  { unk: ['CREATURE', 'CREATURES'], id: ['BOGLIN', 'BOGLINS'], n: 4, asleep: 0, undead: false, known: false,
    at: [[-1.3, 4.6, 1.2], [-0.45, 4.9, 1.2], [0.45, 4.6, 1.2], [1.3, 4.9, 1.2]], color: '#6b7a4e' },
  { unk: ['FIGURE', 'FIGURES'], id: ['BONE WALKER', 'BONE WALKERS'], n: 2, asleep: 0, undead: true, known: false,
    at: [[-0.75, 6.9, 1.6], [0.75, 6.9, 1.6]], color: '#7a5468' },
]
const clone = (o) => JSON.parse(JSON.stringify(o))

const GO = {
  st: null, // authoritative fixture state ("engine")
  reset() {
    this.st = { round: 1, team: clone(TEAM0), groups: clone(GROUPS0), atk: 0, runTried: false, result: null, poisoned: false,
      chest: { trap: 'POISON NEEDLE', inspected: {}, disarmed: false, open: false, calfo: false } }
  },
  name(gi, plural) { const g = this.st.groups[gi]; return (g.known ? g.id : g.unk)[plural ? 1 : 0] },
  alive(gi) { return this.st.groups[gi] && this.st.groups[gi].n > 0 },
  aliveGroups() { return this.st.groups.map((g, i) => i).filter((i) => this.alive(i)) },
  // Order menu for one member: what Go would send. Reasons are part of it.
  menu(slot) {
    const m = this.st.team[slot - 1], back = slot > 3
    const caster = !!SPELLBOOK[slot]
    const canDispel = m.cls.endsWith('PRI') || (m.cls.endsWith('BIS') && m.lvl > 3)
    const useItem = slot === 6 ? 'SILVER ROD' : null
    return [
      { key: 'F', act: 'fight', label: 'Fight', ok: !back, why: back ? 'Back line (4–6) cannot fight' : '' },
      { key: 'P', act: 'parry', label: 'Parry', ok: true },
      { key: 'S', act: 'spell', label: 'Spell', ok: caster, why: caster ? '' : `${m.name} knows no spells` },
      useItem ? { key: 'U', act: 'use', label: `Use ${useItem}`, ok: true, item: useItem }
              : { key: 'U', act: 'use', label: 'Use', ok: true, note: 'nothing usable; counts as Parry' },
      { key: 'D', act: 'dispel', label: 'Dispel', ok: canDispel, why: canDispel ? '' : m.cls.endsWith('BIS') ? 'Bishops dispel from level 4 (Corvin is level 3)' : `A ${m.cls.slice(2) === 'FIG' ? 'fighter' : m.cls.slice(2) === 'THI' ? 'thief' : 'mage'} cannot dispel` },
      { key: 'R', act: 'run', label: 'Run', ok: true },
    ]
  },
  spells(slot) {
    const m = this.st.team[slot - 1]
    return (SPELLBOOK[slot] || []).map((s) => {
      const left = (m.sp || {})[s.lvl] || 0
      const why = s.why || (left ? '' : `No level ${s.lvl} spell points left`)
      return Object.assign({}, s, { left, ok: !why, why })
    })
  },
  targetKind(o) {
    if (o.act === 'fight' || o.act === 'dispel' || o.act === 'use') return 'group'
    if (o.act === 'spell') return (SPELLBOOK[o.slot].find((s) => s.name === o.spell) || {}).tgt
    return 'none'
  },

  // Resolve a whole round at once, as engine.ExecuteRound does. Returns the
  // round as display blocks; the authoritative state is final on return.
  commit(orders) {
    const st = this.st, r = st.round, blocks = []
    const team = st.team, G = st.groups
    const parry = new Set(orders.filter((o) => o.act === 'parry' || (o.act === 'use' && !o.item)).map((o) => o.slot))
    const cnt = (gi) => ({ k: 'count', gi, n: G[gi].n, asleep: G[gi].asleep })
    if (r >= 2 && G.some((g) => !g.known)) {
      G.forEach((g) => { g.known = true })
      blocks.push({ silent: true, lines: [], deltas: G.map((g, gi) => ({ k: 'name', gi })), note: 'Go’s per-round identification succeeded: group names update (no message).' })
    }
    const kill = (gi) => { const g = G[gi]; if (g.asleep > 0) g.asleep--; g.n--; if (g.asleep > g.n) g.asleep = g.n }
    const firstAlive = () => G.findIndex((g) => g.n > 0)
    const order = r === 1 ? ['p5', 'g0.0', 'p1', 'g1.0', 'g0.1', 'p3', 'p6', 'g0.2', 'p2', 'g1.1', 'g0.3', 'p4']
                          : ['p1', 'g0.0', 'p3', 'p5', 'g1.0', 'g0.1', 'p2', 'p6', 'g1.1', 'p4', 'g0.2', 'g0.3']
    const DMG = [5, 0, 4, 3, 0, 6, 2, 0]
    for (const tok of order) {
      if (G.every((g) => g.n === 0) || st.result) break
      if (tok[0] === 'p') {
        const slot = +tok[1], m = team[slot - 1], o = orders[slot - 1]
        if (!o || m.hp <= 0) continue
        const tgFor = (gi) => (this.alive(gi) ? gi : firstAlive())
        if (o.act === 'fight') {
          const gi = tgFor(o.tg); if (gi < 0) continue
          const asleep = G[gi].asleep > 0
          const kills = slot === 1 || asleep || (slot === 2 && r % 2 === 0) || (slot === 3 && r >= 2)
          const dmg = slot === 1 ? 11 : slot === 2 ? (kills ? 9 : 4) : 3 + r
          const nm = this.name(gi)
          const lines = [`${m.name} ATTACKS A${/^[AEIOU]/.test(nm) ? 'N' : ''} ${nm}`, `AND HITS ${slot === 1 ? 2 : 1} TIME${slot === 1 ? 'S' : ''} FOR ${dmg} DAMAGE`]
          if (kills) { kill(gi); lines.push(`${nm} IS KILLED!`) }
          blocks.push({ actor: 'p' + slot, tgt: 'g' + gi, lines, fx: [{ type: 'hit', from: 'p' + slot, to: 'g' + gi, text: '−' + dmg }], deltas: [cnt(gi)] })
        } else if (o.act === 'spell' || o.act === 'use') {
          const spell = o.act === 'use' ? 'HALITO' : o.spell
          if (o.act === 'spell') m.sp[SPELLBOOK[slot].find((s) => s.name === spell).lvl]--
          const verb = o.act === 'use' ? `${m.name} USES A ${o.item || 'NOTHING'}` : `${m.name} CASTS ${spell}`
          if (o.act === 'use' && !o.item) continue
          if (spell === 'HALITO' || spell === 'BADIOS') {
            const gi = tgFor(o.tg); if (gi < 0) { blocks.push({ actor: 'p' + slot, lines: [verb, 'BUT NOTHING IS LEFT TO HIT'], fx: [] }); continue }
            const nm = this.name(gi), dmg = spell === 'HALITO' ? 8 : 7
            kill(gi)
            blocks.push({ actor: 'p' + slot, tgt: 'g' + gi, lines: [verb, `A${/^[AEIOU]/.test(nm) ? 'N' : ''} ${nm} TAKES ${dmg} DAMAGE`, `${nm} IS KILLED!`], fx: [{ type: 'halito', from: 'p' + slot, to: 'g' + gi, text: '−' + dmg }], deltas: [cnt(gi)] })
          } else if (spell === 'KATINO') {
            const gi = tgFor(o.tg); if (gi < 0) continue
            const g = G[gi]
            if (g.undead) { blocks.push({ actor: 'p' + slot, tgt: 'g' + gi, lines: [verb, `THE ${this.name(gi, true)} ARE NOT AFFECTED`], fx: [{ type: 'sleep', from: 'p' + slot, to: 'g' + gi }] }); continue }
            const before = g.asleep; g.asleep = g.n
            blocks.push({ actor: 'p' + slot, tgt: 'g' + gi, lines: [verb, `${g.asleep - before} ${this.name(gi, g.asleep - before !== 1)} FALL${g.asleep - before === 1 ? 'S' : ''} ASLEEP`], fx: [{ type: 'sleep', from: 'p' + slot, to: 'g' + gi }], deltas: [cnt(gi)] })
          } else if (spell === 'DIOS') {
            const t = team[o.ally - 1], heal = Math.min(6, t.max - t.hp); t.hp += heal
            blocks.push({ actor: 'p' + slot, tgt: 'p' + o.ally, lines: [verb, `${t.name} IS HEALED FOR ${heal}`], fx: [{ type: 'heal', from: 'p' + slot, to: 'p' + o.ally, text: '+' + heal }], deltas: [{ k: 'hp', slot: o.ally, hp: t.hp }] })
          } else {
            blocks.push({ actor: 'p' + slot, lines: [verb, spell === 'MILWA' ? 'THE LIGHT GROWS BRIGHTER' : `A SHIELD SURROUNDS ${m.name}`], fx: [{ type: spell === 'MILWA' ? 'dispel' : 'parry', to: 'p' + slot }] })
          }
        } else if (o.act === 'dispel') {
          const gi = tgFor(o.tg); if (gi < 0) continue
          const nm = this.name(gi)
          if (G[gi].undead) { kill(gi); blocks.push({ actor: 'p' + slot, tgt: 'g' + gi, lines: [`${m.name} TRIES TO DISPEL`, `A${/^[AEIOU]/.test(nm) ? 'N' : ''} ${nm} IS DISPELLED!`], fx: [{ type: 'dispel', from: 'p' + slot, to: 'g' + gi }], deltas: [cnt(gi)] }) }
          else blocks.push({ actor: 'p' + slot, tgt: 'g' + gi, lines: [`${m.name} TRIES TO DISPEL`, 'BUT NOTHING HAPPENS'], fx: [{ type: 'dispel', from: 'p' + slot, to: 'g' + gi }] })
        } else if (o.act === 'run') {
          if (!st.runTried) { st.runTried = true; blocks.push({ actor: 'p' + slot, lines: [`${m.name} TRIES TO RUN`, 'BUT THE PARTY CANNOT ESCAPE!'], fx: [] }) }
          else { st.result = 'fled'; blocks.push({ actor: 'p' + slot, lines: [`${m.name} TRIES TO RUN`, 'THE PARTY FLEES!'], fx: [] }) }
        }
      } else {
        const [gi, k] = tok.slice(1).split('.').map(Number), g = G[gi]
        if (!g || k >= g.n - g.asleep) continue
        const front = [1, 2, 3].filter((s) => team[s - 1].hp > 0)
        const slot = front[st.atk % front.length], t = team[slot - 1]
        let d = DMG[st.atk % DMG.length]; st.atk++
        if (parry.has(slot) && st.atk % 2 === 0) d = 0
        if (t.hp - d < 4) d = 0
        const nm = this.name(gi)
        const lines = [`A${/^[AEIOU]/.test(nm) ? 'N' : ''} ${nm} ${g.undead ? 'CLAWS AT' : 'SWINGS AT'} ${t.name}`, d ? `AND HITS FOR ${d} DAMAGE` : 'AND MISSES']
        const deltas = []
        if (d) { t.hp -= d; deltas.push({ k: 'hp', slot, hp: t.hp }) }
        if (d && g.undead && !st.poisoned) { st.poisoned = true; t.st = 'POISON'; lines.push(`${t.name} IS POISONED`); deltas.push({ k: 'st', slot, st: 'POISON' }) }
        blocks.push({ actor: 'g' + gi, sub: k, tgt: 'p' + slot, lines, fx: [{ type: d ? 'hit' : 'miss', from: 'g' + gi, to: 'p' + slot, text: d ? '−' + d : '' }], deltas })
      }
    }
    if (!st.result && G.every((g) => g.n === 0)) {
      st.result = 'victory'
      blocks.push({ actor: null, lines: ['FOR KILLING THE MONSTERS', 'EACH SURVIVOR GETS 135 EXPERIENCE POINTS'], fx: [], end: true })
    }
    if (!st.result) st.round++
    return blocks
  },

  // Chest. The real trap stays here; only reports are returned.
  chestWho(act) {
    const t = this.st.team, c = this.st.chest
    if (act === 'inspect') return t.filter((m) => !c.inspected[m.slot]).map((m) => m.slot)
    if (act === 'calfo') return t.filter((m) => m.slot === 3 && m.sp[2] > 0).map((m) => m.slot)
    return t.filter((m) => m.hp > 0).map((m) => m.slot)
  },
  chest(act, slot, trapName) {
    const c = this.st.chest, m = this.st.team[slot - 1]
    if (act === 'inspect') {
      c.inspected[slot] = true
      const said = { 4: 'POISON NEEDLE', 2: 'GAS BOMB', 1: 'NO TRAP', 3: 'NO TRAP', 5: 'STUNNER', 6: 'POISON NEEDLE' }[slot]
      return { lines: [`${m.name} INSPECTS THE CHEST`, said === 'NO TRAP' ? 'AND FINDS NO TRAP' : `IT LOOKS LIKE A ${said}`], report: { by: m.name, how: 'Inspect', said } }
    }
    if (act === 'calfo') { m.sp[2]--; return { lines: ['BRANNOC CASTS CALFO', 'THE TRAP IS A POISON NEEDLE'], report: { by: m.name, how: 'CALFO', said: 'POISON NEEDLE' } } }
    if (act === 'disarm') {
      if (trapName === c.trap) { c.disarmed = true; return { lines: [`${m.name} DISARMS THE ${trapName}!`] } }
      return { lines: [`${m.name} TRIES TO DISARM A ${trapName}`, `OOPS! A ${c.trap}!`, `${m.name} IS POISONED`], deltas: [{ k: 'st', slot, st: 'POISON' }], sprung: true, then: () => { m.st = 'POISON'; c.disarmed = true } }
    }
    if (act === 'open') {
      c.open = true
      const lines = c.disarmed ? [`${m.name} OPENS THE CHEST`] : [`${m.name} OPENS THE CHEST`, `OOPS! A ${c.trap}!`, `${m.name} IS POISONED`]
      if (!c.disarmed) m.st = 'POISON'
      return { lines: lines.concat(['THE PARTY FINDS 160 GOLD']), deltas: c.disarmed ? [] : [{ k: 'st', slot, st: 'POISON' }], end: true }
    }
    return { lines: ['THE PARTY LEAVES THE CHEST'], end: true }
  },
}
