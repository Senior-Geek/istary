// Display helpers for the Temple of Cant (T-3). They only reshape what Go
// sent in view.temple (gui.TemplePanel: the controller's app.TempleView and
// the last treatment's or pooling's receipt); no rule is decided here. Who
// is a patient, the donation, whether a patient can be treated, who may
// tithe, whether the gold is there, and every refusal are Go's. The risk is
// worded by the kind of treatment, never as a chance: Go does not send one.

const title = (s = '') => s.toLowerCase().replace(/(^|\s)\S/g, (c) => c.toUpperCase())

// conditions are the conditions as the screen names them.
const conditions = { PARALYZE: 'Paralysed', PARALYZED: 'Paralysed', STONED: 'Stoned', DEAD: 'Dead', ASHED: 'Ashes', LOST: 'Lost', OK: 'Well' }
export const condition = (status = '') => conditions[status] ?? title(status)

// risks are the notices for each kind of treatment.
export const risks = {
  cure: 'Treatment is assured when the required donation is available.',
  raise: 'Resurrection is uncertain. If it fails, this character will become ASHES.',
  ashes: 'This is the final chance at restoration. Failure means the character will be permanently LOST.',
}
export const charged = 'The donation is charged whether treatment succeeds or fails.'

// outcomes are what a treatment of a kind may end in.
export const outcomes = {
  cure: ['Well again: hit points as they were.'],
  raise: ['Raised: well again, with a single hit point; a little older.', 'Not raised: reduced to ashes.'],
  ashes: ['Restored: well again, with every hit point; a little older.', 'Not restored: LOST forever, beyond any help.'],
}

// fundingNote words Go's funding for a patient's donation.
export function fundingNote(p) {
  return {
    payer: 'A member can pay the donation.',
    pool: 'Only the party’s gold together covers it: pool it to a payer first.',
    short: 'The party’s gold together does not cover it.',
  }[p.funding] ?? ''
}

// patients are the Temple's patients as the screen lists them: what each
// shows, whether it can be treated (Go's refusal otherwise) and its note.
export function patients(temple) {
  return (temple?.view?.patients ?? []).map((p) => ({
    key: `patient-${p.index}`,
    index: p.index,
    name: p.name,
    who: `${title(p.class)} · level ${p.level}`,
    inParty: p.inParty,
    condition: condition(p.status),
    kind: p.kind,
    lost: p.status === 'LOST',
    donation: p.kind ? p.donation : null,
    treatable: !!p.kind && !p.refusal,
    reason: p.refusal,
    note: p.kind && !p.refusal ? fundingNote(p) : '',
    funding: p.funding,
  }))
}

// payers are the party members, with Go's refusal for those who may not tithe.
export const payers = (temple) => temple?.view?.payers ?? []

// defaultPayer is the member who may tithe with the most gold, or -1.
export function defaultPayer(temple) {
  let best = -1, gold = -1
  for (const p of payers(temple)) {
    if (!p.refusal && p.gold > gold) { best = p.slot; gold = p.gold }
  }
  return best
}

// pickLabel is what a click on a party member does here, for the party
// bar: choose them to tithe. Not the payer already chosen, and not one Go
// says may not tithe.
export function pickLabel(temple, slot, payer) {
  const p = payers(temple).find((p) => p.slot === slot)
  if (!p || p.refusal || slot === payer) return null
  return 'Tithe'
}

// confirmation is what the screen asks before a treatment: the patient,
// the donation, who pays and from what, the outcomes and the warnings. ready
// is false (with why) when the chosen payer's gold, as Go gave it, does not
// cover the donation; Go refuses such a treatment anyway.
export function confirmation(temple, index, payer) {
  const p = (temple?.view?.patients ?? []).find((p) => p.index === index)
  const by = payers(temple).find((x) => x.slot === payer)
  if (!p || !p.kind) return null
  const c = {
    patient: p.name,
    condition: condition(p.status),
    kind: p.kind,
    donation: p.donation,
    payer: by?.name ?? '',
    payerGold: by?.gold ?? 0,
    risk: risks[p.kind],
    outcomes: outcomes[p.kind],
    charged,
    permanent: p.kind === 'ashes',
    ready: true,
    why: '',
  }
  if (p.refusal) return { ...c, ready: false, why: p.refusal }
  if (!by) return { ...c, ready: false, why: 'Choose who will tithe: click a party member.' }
  if (by.refusal) return { ...c, ready: false, why: by.refusal }
  if (by.gold < p.donation) {
    const why = p.funding === 'short'
      ? `${by.name} has ${by.gold} GP, and the party together cannot meet the donation.`
      : `${by.name} has ${by.gold} GP. Pool the party’s gold to ${by.name} first.`
    return { ...c, ready: false, why }
  }
  return c
}

// receipt describes the last treatment or pooling and whether it is saved:
// tone is 'saved', 'notDurable' or 'unsaved'; mood is 'well', 'ashes' or
// 'lost' for a treatment.
export function receipt(temple) {
  const r = temple?.last
  if (!r) return null
  const status = !r.saved
    ? { tone: 'unsaved', status: 'NOT saved: it exists only in this window until a save succeeds' }
    : !r.durable
      ? { tone: 'notDurable', status: 'Saved, but perhaps not durably (see the warning above)' }
      : { tone: 'saved', status: 'Saved' }
  if (r.gold) {
    const m = r.gold
    return {
      ...status, mood: 'gold', title: `Pooled the party's ${m.total} GP to ${m.names[m.to]}`,
      lines: m.names.map((n, i) => `${n}: ${m.before[i]} → ${m.after[i]} GP`),
    }
  }
  const t = r.treat
  if (!t) return null
  const mood = t.healed ? 'well' : t.statusAfter === 'ASHED' ? 'ashes' : 'lost'
  const heading = {
    well: `${t.patient} is well again.`,
    ashes: `The priests could not raise ${t.patient}. ${t.patient} is now ashes.`,
    lost: `${t.patient} could not be restored, and is lost forever.`,
  }[mood]
  const years = (w) => Math.floor(w / 52)
  const lines = [
    `${condition(t.statusBefore)} → ${condition(t.statusAfter)}`,
    `Donation ${t.donation} GP, tithed by ${t.payer}: ${t.payerBefore} → ${t.payerAfter} GP`,
    t.hpAfter !== t.hpBefore ? `Hit points ${t.hpBefore} → ${t.hpAfter} of ${t.maxHp}` : `Hit points ${t.hpAfter} of ${t.maxHp}`,
    t.ageAfter !== t.ageBefore ? `Age ${years(t.ageBefore)} → ${years(t.ageAfter)}` : 'Age unchanged',
  ]
  return { ...status, mood, title: heading, lines, messages: t.messages ?? [] }
}
