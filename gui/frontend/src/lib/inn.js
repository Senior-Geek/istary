// Display helpers for the Adventurer's Inn (I-4). They only reshape what Go
// sent in view.inn (gui.InnPanel: the controller's app.InnView and the last
// stay's or pooling's receipt) and the quotes Go gives for chosen weeks
// (Session.InnQuote); no rule is decided here. Rooms, rates, how long a
// stay lasts, what it costs, the hit points it leaves, and every refusal
// are Go's: a Stay button is enabled exactly when Go gives no refusal, and
// a disabled one shows Go's reason. The mode (until healed, or chosen
// weeks) and the weeks typed are the page's own: changing them only asks
// Go for quotes.

export const STAT_NAMES = ['Strength', 'I.Q.', 'Piety', 'Vitality', 'Agility', 'Luck']

const plural = (n, unit) => `${n} ${unit}${n === 1 ? '' : 's'}`
const title = (s = '') => s.toLowerCase().replace(/(^|\s)\S/g, (c) => c.toUpperCase())

// roomName is a room's name as Go gives it ("ECONOMY ROOMS"), for a
// sentence ("Economy Rooms").
export const roomName = (name) => title(name)

// guest is the guest with what the Inn shows of them: Go's InnView names
// the guest; the rest is the guest's roster entry (view.roster), or null at
// "who will stay".
export function guest(inn, roster = []) {
  const v = inn?.view
  if (!v || v.guest < 0) return null
  const ch = roster.find((c) => c.name === v.name)
  if (!ch) return { slot: v.guest, name: v.name }
  const spells = (cur = [], max = []) => max.map((m, i) => (m ? `${cur[i]}/${m}` : null)).filter(Boolean).join(' ')
  return {
    slot: v.guest,
    index: ch.index,
    name: ch.name,
    class: ch.class,
    level: ch.level,
    hp: ch.hp,
    maxHp: ch.maxHp,
    gold: ch.gold,
    age: ch.age,
    xp: ch.xp,
    nextXp: ch.nextXp,
    mage: spells(ch.mageSpells, ch.maxMage),
    priest: spells(ch.priestSpells, ch.maxPriest),
  }
}

// quoteText words what a stay would do, from Go's quote.
export function quoteText(q, room) {
  if (!q) return ''
  if (q.refusal) return q.refusal
  if (room.free) return 'One free night: no healing'
  if (q.weeks === 0) {
    if (q.needed === 0) return 'Not hurt: no weeks, no charge'
    return 'Cannot pay for a week: no healing, no charge'
  }
  let s = `${plural(q.weeks, 'week')} · ${q.cost} GP · HP to ${q.hp}`
  if (q.healed) s += ' (fully healed)'
  return s
}

// shortfall says when the guest's gold pays for some weeks but not full
// recovery, or '' (no week at all is said by quoteText).
export function shortfall(q, name) {
  if (!q || q.refusal || q.needed === 0 || q.weeks === 0 || q.weeks >= q.needed) return ''
  return `${name} can pay for ${plural(q.affordable, 'week')} of the ${plural(q.needed, 'week')} needed`
}

// rows are the rooms as the screen lists them, for a mode: 'healed' uses
// each room's quote until healed from the view; 'weeks' uses the quotes Go
// gave for the chosen weeks (quotes, by room), the stables always one free
// night. Each row says what its Stay button asks Go and why it is disabled
// (Go's refusal: a paid room the guest cannot pay a week of names the free
// stables). A guest who may not stay (Go's guestRefusal) gets no rows.
export function rows(inn, mode = 'healed', weeks = 1, quotes = {}) {
  const v = inn?.view
  if (!v || v.guest < 0) return []
  return (v.rooms ?? []).map((r) => {
    const chosen = mode === 'weeks' && !r.free
    const q = chosen ? quotes[r.room] ?? null : r.untilHealed
    return {
      key: `room-${r.room}`,
      room: r.room,
      name: roomName(r.name),
      free: r.free,
      rate: r.free ? 'Free · heals nothing' : `${r.hpPerWeek} HP a week · ${r.feePerWeek} GP a week`,
      text: quoteText(q, r),
      short: chosen ? '' : shortfall(q, v.name),
      request: { action: 'stay', room: r.room, weeks: chosen ? weeks : 0 },
      reason: q ? q.refusal : 'Working out…',
      maxWeeks: r.maxWeeks,
    }
  })
}

// maxWeeks is the most weeks the guest may choose in any room (0: none).
export function maxWeeks(inn) {
  return Math.max(0, ...(inn?.view?.rooms ?? []).map((r) => r.maxWeeks))
}

// pickLabel is what a click on a party member does here, for the party
// bar: check them in, in place of the guest. The guest is marked instead,
// and a member who may not stay (Go's refusal) cannot be picked.
export function pickLabel(inn, slot) {
  const v = inn?.view
  if (!v || slot === v.guest) return null
  const g = (v.guests ?? []).find((g) => g.slot === slot)
  if (!g || g.refusal) return null
  return v.guest < 0 ? 'Stay' : 'Stay instead'
}

// words turns the Inn's capitals into a sentence: "YOU NEED 1248 MORE".
export const words = (msg = '') => msg.trim()

// statChanges are the attributes a stay changed, from Go's before and after.
export function statChanges(st) {
  const out = []
  for (let i = 0; i < 6; i++) {
    const d = (st.statsAfter?.[i] ?? 0) - (st.statsBefore?.[i] ?? 0)
    if (d) out.push({ name: STAT_NAMES[i], from: st.statsBefore[i], to: st.statsAfter[i], gained: d > 0 })
  }
  return out
}

const spellLine = (before = [], after = []) =>
  after.map((n, i) => (n !== before[i] ? `L${i + 1} ${before[i]}→${n}` : null)).filter(Boolean).join(', ')

// receipt describes the last stay or pooling and whether it is saved: tone
// is 'saved', 'notDurable' or 'unsaved'; lines are what the stay did, each
// with a kind ('bad' for an adverse outcome).
export function receipt(inn) {
  const r = inn?.last
  if (!r) return null
  const status = !r.saved
    ? { tone: 'unsaved', status: 'NOT saved: it exists only in this window until a save succeeds' }
    : !r.durable
      ? { tone: 'notDurable', status: 'Saved, but perhaps not durably (see the warning above)' }
      : { tone: 'saved', status: 'Saved' }
  if (r.gold) {
    const m = r.gold
    const lines = m.names.map((n, i) => ({ kind: 'gold', text: `${n}: ${m.before[i]} → ${m.after[i]} GP` }))
    return { ...status, title: `Pooled the party's ${m.total} GP to ${m.names[m.to]}`, lines }
  }
  const st = r.stay
  if (!st) return null
  const lines = []
  const room = roomName(st.roomName)
  lines.push({ kind: 'stay', text: st.weeks ? `${room}, ${plural(st.weeks, 'week')}: ${st.charged} GP` : `${room}, one night: free` })
  lines.push({ kind: 'hp', text: st.hpAfter > st.hpBefore || st.died ? `HP ${st.hpBefore} → ${st.hpAfter} of ${st.maxHpAfter}` : `HP ${st.hpAfter} of ${st.maxHpAfter}: no healing` })
  if (st.maxHpAfter !== st.maxHpBefore) lines.push({ kind: 'hp', text: `Maximum HP ${st.maxHpBefore} → ${st.maxHpAfter}` })
  const sp = [spellLine(st.mageBefore, st.mageAfter), spellLine(st.priestBefore, st.priestAfter)].filter(Boolean)
  lines.push({ kind: 'spells', text: sp.length ? `Spell points restored: ${sp.join('; ')}` : 'Spell points: nothing to restore' })
  lines.push({ kind: 'age', text: st.ageAfter !== st.ageBefore ? `Age ${Math.floor(st.ageBefore / 52)} → ${Math.floor(st.ageAfter / 52)}` : 'Age unchanged' })
  if (st.levelAfter > st.levelBefore) lines.push({ kind: 'level', text: `Made level ${st.levelAfter}` })
  for (const c of statChanges(st)) lines.push({ kind: c.gained ? 'gain' : 'loss', text: `${c.name} ${c.from} → ${c.to}` })
  if (st.learned?.length) lines.push({ kind: 'level', text: `Learned ${st.learned.join(', ')}` })
  if (st.died) lines.push({ kind: 'bad', text: `${st.guest} died of old age and is LOST` })
  else if (st.xpNeeded > 0) lines.push({ kind: 'xp', text: `${st.xpNeeded} more experience points to make level ${st.levelAfter + 1}` })
  const did = st.weeks ? `${st.guest} stayed ${plural(st.weeks, 'week')}` : `${st.guest} spent a night in the stables`
  return { ...status, title: did, lines, messages: (st.messages ?? []).map(words).filter(Boolean) }
}
