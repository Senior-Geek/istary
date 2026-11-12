<script>
  // Boltac's Trading Post (M8-4), in the location frame. The customer is
  // chosen from the party bar (or, with the party hidden, the list here)
  // and is marked there; the tabs show what Boltac sells and what the
  // customer carries, from Go's view.shop only.
  //
  // UX-2: the frame fills the window without scrolling the page (its fill
  // layout). Under the artwork, the merchandise: the tabs and Boltac's
  // words stay put and only the list scrolls. Beside it, under the
  // Actions, the Selected Character panel (SelectedCharacter): the
  // customer's gold, AC, carrying and items, equipped apart from carried,
  // with the character sheet's Equip and Take off. The Actions keep only
  // the shop's own: Pool party gold and Leave the counter. No sheet opens
  // here; the party bar only chooses the customer.
  //
  // Arriving serves the first party member (Go, Session.serveFirst), so
  // the page opens on the Buy tab with a customer. The party bar is always
  // shown here (the frame's partyAlways) and is the one way to change the
  // customer. After Leave the counter, it waits for a click on the bar;
  // with nobody to serve, it says why and offers the way back. Every price, fee, stock
  // and refusal is Go's (lib/shop.js): a button is enabled exactly when Go
  // gives no refusal, and a disabled one says Go's reason. The tabs and the
  // item lines are the page's own; only the buttons call Go
  // (Session.ShopAction). "← Castle" and Escape leave Boltac's from
  // anywhere, a question waiting answered no (the frame's note says so).
  //
  // Props:
  //   view    the Go View (view.location is "trading")
  //   busy    a request is running
  //   call    (fn, ...args) => Promise   runs a Session method
  //   onopen  (rosterIndex) => void      open a party member's sheet
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'
  import SelectedCharacter from './SelectedCharacter.svelte'
  import { tabs, rows, amountLabel, empty, boltac, customer, question, pickLabel, receipt, selected } from './shop.js'

  let { view, busy = false, call, onopen } = $props()

  let tab = $state('buy')

  const shop = $derived(view.shop)
  const who = $derived(customer(shop))
  const said = $derived(boltac(shop))
  const asks = $derived(question(shop))
  const last = $derived(receipt(shop))
  const lines = $derived(rows(shop, tab))
  const members = $derived((view.partyBar ?? []).filter((p) => p.name))
  const ch = $derived(selected(view))

  const act = (request) => call(S.ShopAction, request)
  const enter = (slot) => act({ action: 'enter', slot })
  const equip = (n, on) => call(S.EquipAction, { index: ch.index, n, equip: on })
</script>

<LocationFrame {view} {busy} {call} {onopen} fill partyAlways pick={(slot) => pickLabel(shop, slot)} onpick={enter} current={who?.slot ?? -1}>
  {#snippet actions()}
    {#if who}
      <div class="row">
        <button class="small" disabled={busy || !!asks} onclick={() => act({ action: 'pool' })}
          title={asks ? 'Answer Boltac first' : `Give ${who.name} every other party member's gold (Pool Gold)`}>Pool party gold</button>
        <button class="small" disabled={busy || !!asks} onclick={() => act({ action: 'leave-customer' })}
          title={asks ? 'Answer Boltac first' : 'Back to WHO WILL ENTER'}>Leave the counter</button>
      </div>
    {:else if members.length}
      <p class="lead">Nobody is at the counter.</p>
      <p class="muted">Click a party member below to serve them.</p>
    {:else}
      <p class="lead">Boltac serves only a party.</p>
      <p class="muted">Nobody in the party can be served. Form a party at Gilgamesh's Tavern first.</p>
      <button class="small" disabled={busy || !view.return?.open} onclick={() => call(S.ReturnToCastle)}>Back to the Castle</button>
    {/if}
  {/snippet}

  <div class="counter-layout">
  <section class="shop">
    <!-- The counter stays in view while the list scrolls: the tabs, what
         Boltac said or asks, and the last transaction. -->
    <div class="counter">
      <div class="tabs" role="tablist" aria-label="Boltac's services">
        {#each tabs as t (t.key)}
          <button role="tab" class="tab" aria-selected={tab === t.key} class:on={tab === t.key}
            onclick={() => { tab = t.key }}>{t.label}</button>
        {/each}
      </div>
      {#if asks?.kind === 'buy'}
        <div class="banner message ask" role="alertdialog" aria-label="Boltac asks">
          <p><b>Unusable item.</b> A {asks.class} cannot use the {asks.item}. Buy it anyway for {asks.price} GP?</p>
          <div class="row">
            <button class="primary" disabled={busy} onclick={() => act({ action: 'confirm', yes: true })}>Buy anyway</button>
            <button disabled={busy} onclick={() => act({ action: 'confirm', yes: false })}>No</button>
          </div>
        </div>
      {:else if asks?.kind === 'sell'}
        <div class="banner message ask" role="alertdialog" aria-label="Boltac asks">
          <p><b>Equipped item.</b> {asks.name} is wearing the {asks.item}. Selling it takes it off and away, for
            {asks.price} GP. Sell it?</p>
          <div class="row">
            <button class="primary" disabled={busy} onclick={() => act({ action: 'confirm-sale', yes: true })}>Sell it</button>
            <button disabled={busy} onclick={() => act({ action: 'confirm-sale', yes: false })}>No, keep it</button>
          </div>
        </div>
      {:else if said}
        <p class="boltac" aria-live="polite"><span class="muted">Boltac:</span> “{said}”</p>
      {/if}

      {#if last}
        <div class="receipt {last.tone}" aria-label="Last transaction">
          <div>{last.did}.</div>
          <div class="status">{last.status}</div>
          {#if last.tone === 'unsaved'}
            <button class="small" disabled={busy} onclick={() => call(S.SaveNow)}
              title="Writes the game as this window has it; nothing is bought, sold or moved again">Try saving again</button>
          {/if}
          {#if last.lines.length}<ul class="moves">{#each last.lines as l, i (i)}<li>{l}</li>{/each}</ul>{/if}
        </div>
      {/if}
    </div>

    {#if lines.length}
      <div class="tablewrap">
        <table class="lines">
          <thead>
            <tr>
              {#if tab !== 'buy'}<th class="num">#</th>{/if}
              <th>Item</th>
              <th class="num">{amountLabel(tab)}</th>
              {#if tab === 'buy'}<th>Stock</th>{/if}
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each lines as l (l.key)}
              <tr class:unusable={tab === 'buy' && !l.usable} class:refused={!!l.reason}>
                {#if tab !== 'buy'}<td class="num muted">{l.n}</td>{/if}
                <td class="name">
                  <span>{l.name}</span>
                  {#each l.marks as m (m.key)}<span class="mark {m.key}">{m.label}</span>{/each}
                  {#if l.note}<span class="note">{l.note}</span>{/if}
                </td>
                <td class="num">{l.amount} GP</td>
                {#if tab === 'buy'}<td class="stock">{l.stock}</td>{/if}
                <td class="act">
                  <button class="small" class:primary={!l.reason} disabled={busy || !!l.reason || !!asks}
                    onclick={() => act(l.request)}>{l.label}</button>
                  {#if l.reason}<span class="why">{l.reason}</span>{/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else}
      <p class="muted">{empty(shop, tab)}</p>
    {/if}
  </section>
  <SelectedCharacter {ch} {busy} locked={asks ? 'Answer Boltac first.' : ''} onequip={equip} />
  </div>
</LocationFrame>

<style>
  .lead { margin: 0 0 4px; }
  .row { display: flex; flex-wrap: wrap; gap: 6px; }
  /* The merchandise and the Selected Character panel share the height
     under the artwork, in the frame's columns: each scrolls inside. */
  .counter-layout { display: grid; grid-template-columns: minmax(0, 1fr) minmax(220px, 300px); gap: 12px 16px;
    height: 100%; min-height: 0; }
  .counter-layout > :global(*) { min-height: 0; }
  /* The counter stays put; only the list under it scrolls. */
  .counter { padding: 0 0 6px; border-bottom: 1px solid var(--line); display: grid; gap: 6px; }
  .ask { margin: 0; color: var(--text); }
  .ask p { margin: 0 0 6px; }
  .boltac { margin: 0; color: var(--gold); }
  .receipt { padding: 6px 10px; border-radius: 4px; font-size: 13px; border-left: 3px solid; }
  .receipt { display: flex; flex-wrap: wrap; gap: 2px 10px; align-items: baseline; }
  .receipt .status { font-weight: 600; font-size: 12px; }
  .receipt ul.moves { flex-basis: 100%; margin: 2px 0 0; padding-left: 18px; columns: 2 160px; }
  .receipt.saved { background: #1e2a1e; border-color: var(--accent); }
  .receipt.saved .status { color: var(--accent); }
  .receipt.notDurable { background: #2e2a1a; border-color: var(--gold); }
  .receipt.notDurable .status { color: var(--gold); }
  .receipt.unsaved { background: #3a1d1b; border-color: var(--error); }
  .receipt.unsaved .status { color: var(--error); }
  .shop { margin: 0; padding: 10px 12px; display: grid; grid-template-rows: auto minmax(0, 1fr); gap: 6px; min-height: 0; height: 100%; box-sizing: border-box; }
  .tabs { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
  .tab { margin: 0; }
  .tab.on { background: var(--selected); color: var(--text); }
  .tablewrap { overflow: auto; min-height: 0; }
  .tablewrap thead th { position: sticky; top: 0; background: var(--panel); z-index: 1; }
  table.lines td { vertical-align: top; }
  /* A line brought into view stops below the column heads. */
  table.lines tbody tr { scroll-margin-top: 30px; }
  td.name { min-width: 150px; }
  td.name .mark { display: inline-block; margin-left: 6px; padding: 0 5px; border-radius: 3px; font-size: 10px;
    text-transform: uppercase; letter-spacing: 0.04em; background: var(--card); color: var(--muted); }
  td.name .mark.equipped { color: var(--notice); }
  td.name .mark.cursed { color: var(--error); }
  td.name .mark.unidentified { color: var(--gold); }
  td.name .note { display: block; font-size: 11px; color: var(--gold); }
  td.stock { color: var(--muted); font-size: 12px; white-space: nowrap; }
  td.num { white-space: nowrap; }
  td.act { white-space: nowrap; }
  td.act .why { display: block; white-space: normal; color: var(--muted); font-size: 11px; max-width: 220px; }
  button.small { font-size: 12px; padding: 1px 10px; margin: 0; }
</style>
