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
  if (!v || v.step === 'confirm') return ''
  return words(v.message)
}

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
    sell: { amount: (it) => it.price, reason: (it) => it.sellRefusal, label: 'Sell' },
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
    note: '',
    label: of.label,
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

// question is Boltac's question about an unusable item, or null.
export function question(shop) {
  const v = shop?.view
  if (!v || v.step !== 'confirm') return null
  const offer = (v.catalog ?? []).find((o) => o.item === v.confirm)
  return { item: offer?.name ?? `item ${v.confirm}`, price: offer?.price ?? 0, class: v.class }
}

// pickLabel is what a click on a party member does here, for the party
// bar: enter at the counter, in place of the customer there. Nobody can
// be chosen while Boltac waits for an answer, and the customer at the
// counter is marked instead.
export function pickLabel(shop, slot) {
  const v = shop?.view
  if (!v || v.step === 'confirm' || slot === v.customer) return null
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
  }[r.action] ?? r.action
  if (!r.saved) return { tone: 'unsaved', did, status: 'NOT saved: it exists only in this window until a save succeeds' }
  if (!r.durable) return { tone: 'notDurable', did, status: 'Saved, but perhaps not durably (see the warning above)' }
  return { tone: 'saved', did, status: 'Saved' }
}
