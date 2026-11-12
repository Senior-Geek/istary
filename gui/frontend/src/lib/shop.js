// Display helpers for Boltac's Trading Post (M8-4). They only reshape what
// Go sent in view.shop (gui.ShopPanel: the controller's app.ShopView and
// the last transaction's receipt); no rule is decided here. Prices, fees,
// stock, who can use what, and every refusal are Go's: a button is enabled
// exactly when Go gives no refusal for it, and a disabled one shows Go's
// reason. The tabs are the page's own: switching them calls nothing.

export const tabs = [
  { key: 'buy', label: 'Buy' },
  { key: 'sell', label: 'Sell' },
  { key: 'identify', label: 'Identify' },
  { key: 'uncurse', label: 'Uncurse' },
]

// words turns Boltac's capitals into a sentence's: "** YOU CANNOT AFFORD
// IT **" is "You cannot afford it". Go's refusal stays the reason; this
// only drops the stars.
export function words(msg = '') {
  return msg.replace(/^\*\*\s*|\s*\*\*$/g, '').trim()
}

// boltac is what Boltac last said, as he said it (without the stars), or
// ''. His question about an unusable item is drawn as the question itself
// (question), so it is not repeated here.
export function boltac(shop) {
  const v = shop?.view
  if (!v || asking(v)) return ''
  return words(v.message)
}

// asking says Boltac waits for an answer: about an unusable item, or
// before buying an equipped one.
const asking = (v) => v.step === 'confirm' || v.step === 'confirmSell'


// customer is the customer at the counter, or null at "who".
export function customer(shop) {
  const v = shop?.view
  if (!v || v.customer < 0) return null
  return { slot: v.customer, name: v.name, class: v.class, gold: v.gold, room: v.room, items: v.items?.length ?? 0 }
}

// stockText words an offer's stock: -1 is unlimited.
export function stockText(stock) {
  return stock < 0 ? 'unlimited' : `${stock} left`
}

// marks are the flags shown beside one of the customer's items.
export function marks(item) {
  const out = []
  if (item.equipped) out.push({ key: 'equipped', label: 'equipped' })
  if (item.cursed) out.push({ key: 'cursed', label: 'cursed' })
  if (!item.identified) out.push({ key: 'unidentified', label: 'unidentified' })
  return out
}

// rows are the lines of a tab: what each shows, what its button asks Go,
// and why it is disabled (Go's refusal, or '').
export function rows(shop, tab) {
  const v = shop?.view
  if (!v || v.customer < 0) return []
  if (tab === 'buy') {
    return (v.catalog ?? []).map((o) => ({
      key: `buy-${o.item}`,
      name: o.name,
      amount: o.price,
      stock: stockText(o.stock),
      usable: o.usable,
      note: o.usable ? '' : `${v.class} cannot use it; Boltac will ask first`,
      marks: [],
      label: o.usable ? 'Buy' : 'Buy…',
      request: { action: 'buy', item: o.item },
      reason: words(o.refusal),
    }))
  }
  const of = {
    sell: { amount: (it) => it.price, reason: (it) => it.sellRefusal, label: 'Sell', asks: (it) => it.sellAsks },
    identify: { amount: (it) => it.fee, reason: (it) => it.identifyRefusal, label: 'Identify' },
    uncurse: { amount: (it) => it.fee, reason: (it) => it.uncurseRefusal, label: 'Uncurse' },
  }[tab]
  if (!of) return []
  return (v.items ?? []).map((it) => ({
    key: `${tab}-${it.n}`,
    n: it.n,
    name: it.name,
    amount: of.amount(it),
    marks: marks(it),
    note: of.asks?.(it) ? 'equipped: Boltac will ask first' : '',
    label: of.asks?.(it) ? `${of.label}…` : of.label,
    request: { action: tab, n: it.n },
    reason: words(of.reason(it)),
  }))
}

// amountLabel heads a tab's amount column.
export function amountLabel(tab) {
  return { buy: 'Price', sell: 'Boltac pays', identify: 'Fee', uncurse: 'Fee' }[tab] ?? ''
}

// empty says why a tab has no lines.
export function empty(shop, tab) {
  const c = customer(shop)
  if (!c) return 'Choose who will enter: click a party member below.'
  if (tab === 'buy') return 'Boltac has nothing for sale.'
  return `${c.name} carries nothing.`
}

// question is what Boltac asks, or null: kind 'buy' (an item the
// customer's class cannot use) or 'sell' (an item the customer wears).
export function question(shop) {
  const v = shop?.view
  if (!v) return null
  if (v.step === 'confirm') {
    const offer = (v.catalog ?? []).find((o) => o.item === v.confirm)
    return { kind: 'buy', item: offer?.name ?? `item ${v.confirm}`, price: offer?.price ?? 0, class: v.class }
  }
  if (v.step === 'confirmSell') {
    const it = (v.items ?? []).find((it) => it.n === v.sellConfirm)
    return { kind: 'sell', item: it?.name ?? `item ${v.sellConfirm}`, price: it?.price ?? 0, name: v.name }
  }
  return null
}

// pickLabel is what a click on a party member does here, for the party
// bar: enter at the counter, in place of the customer there. Nobody can
// be chosen while Boltac waits for an answer, and the customer at the
// counter is marked instead.
export function pickLabel(shop, slot) {
  const v = shop?.view
  if (!v || asking(v) || slot === v.customer) return null
  return v.customer < 0 ? 'Enter' : 'Serve instead'
}

// receipt words the last transaction and whether it is saved: tone is
// 'saved', 'notDurable' or 'unsaved'.
export function receipt(shop) {
  const r = shop?.last
  if (!r) return null
  const item = r.item
  const gp = Math.abs(r.gold)
  const did = {
    buy: `${r.customer} bought ${item} for ${gp} GP`,
    sell: `${r.customer} sold ${item} for ${gp} GP`,
    uncurse: `Boltac uncursed ${r.customer}'s ${item} and kept it, for ${gp} GP`,
    identify: `Boltac identified ${r.customer}'s item for ${gp} GP: it is ${item}`,
    pool: `Pooled the party's ${r.move?.total ?? 0} GP to ${r.customer}`,
  }[r.action] ?? r.action
  // A pooling lists every member's gold before and after (Go's GoldMove).
  const lines = r.move ? r.move.names.map((n, i) => `${n}: ${r.move.before[i]} → ${r.move.after[i]} GP`) : []
  if (!r.saved) return { tone: 'unsaved', did, lines, status: 'NOT saved: it exists only in this window until a save succeeds' }
  if (!r.durable) return { tone: 'notDurable', did, lines, status: 'Saved, but perhaps not durably (see the warning above)' }
  return { tone: 'saved', did, lines, status: 'Saved' }
}

// selected is the customer's character (Go's CharacterView, from
// view.roster) for the Selected Character panel (UX-2), or null when
// nobody is at the counter. The customer is a party member: the party bar
// gives their roster index.
export function selected(view) {
  const c = customer(view?.shop)
  if (!c) return null
  const index = view.partyBar?.find((p) => p.slot === c.slot + 1)?.index
  return view.roster?.find((ch) => ch.index === index) ?? null
}

// wornAndCarried splits a character's items into the equipped and the
// carried, each in item order. Each keeps its place in the character's
// items as pos (from 1), a key that holds even when equipping is closed
// and Go leaves the items unnumbered (n 0).
export function wornAndCarried(ch) {
  const items = (ch?.items ?? []).map((it, i) => ({ ...it, pos: i + 1 }))
  return { worn: items.filter((it) => it.equipped), carried: items.filter((it) => !it.equipped) }
}
