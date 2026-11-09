<script>
  // Boltac's Trading Post (M8-4), in the location frame. The customer is
  // chosen from the party bar (or, with the party hidden, the list here)
  // and is marked there; the tabs show what Boltac sells and what the
  // customer carries, from Go's view.shop only. Every price, fee, stock
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
  import { tabs, rows, amountLabel, empty, boltac, customer, question, pickLabel, receipt } from './shop.js'

  let { view, busy = false, call, onopen } = $props()

  let tab = $state('buy')

  const shop = $derived(view.shop)
  const who = $derived(customer(shop))
  const said = $derived(boltac(shop))
  const asks = $derived(question(shop))
  const last = $derived(receipt(shop))
  const lines = $derived(rows(shop, tab))
  const partyShown = $derived(view.settings?.partyVisible ?? true)
  const members = $derived((view.partyBar ?? []).filter((p) => p.name))
  const sheet = $derived(who ? view.partyBar?.[who.slot]?.index ?? -1 : -1)

  const act = (request) => call(S.ShopAction, request)
  const enter = (slot) => act({ action: 'enter', slot })
</script>

<LocationFrame {view} {busy} {call} {onopen} pick={(slot) => pickLabel(shop, slot)} onpick={enter} current={who?.slot ?? -1}>
  {#snippet actions()}
    {#if who}
      <div class="customer" aria-label="Customer">
        <div class="who"><b>{who.name}</b> <span class="muted">{who.class}</span></div>
        <dl>
          <dt>Gold</dt><dd>{who.gold} GP</dd>
          <dt>Carrying</dt><dd>{who.items} of 8 · room for {who.room}</dd>
        </dl>
        <div class="row">
          {#if sheet >= 0}<button class="small" disabled={busy} onclick={() => onopen?.(sheet)}>Character sheet</button>{/if}
          <button class="small" disabled={busy || !!asks} onclick={() => act({ action: 'leave-customer' })}
            title={asks ? 'Answer Boltac first' : 'Back to WHO WILL ENTER'}>Leave the counter</button>
        </div>
      </div>
    {:else}
      <p class="lead">Welcome to the Trading Post. Who will enter?</p>
      <p class="muted">{partyShown ? 'Click a party member below.' : 'Choose a party member:'}</p>
    {/if}
    {#if !partyShown && !asks}
      <div class="members">
        {#each members as p (p.slot)}
          <button class="small" class:current={who?.slot === p.slot - 1} disabled={busy || !pickLabel(shop, p.slot - 1)}
            onclick={() => enter(p.slot - 1)}>{p.slot}. {p.name}</button>
        {/each}
      </div>
    {/if}
  {/snippet}

  <section class="shop">
    <!-- The counter stays in view while the list scrolls: the tabs, what
         Boltac said or asks, and the last transaction. -->
    <div class="counter">
      <div class="tabs" role="tablist" aria-label="Boltac's services">
        {#each tabs as t (t.key)}
          <button role="tab" class="tab" aria-selected={tab === t.key} class:on={tab === t.key}
            onclick={() => { tab = t.key }}>{t.label}</button>
        {/each}
        {#if who}
          <span class="who-line" aria-label="Customer at the counter"><b>{who.name}</b> {who.class} ·
            <b>{who.gold}</b> GP · room for <b>{who.room}</b></span>
        {/if}
      </div>
      {#if asks}
        <div class="banner message ask" role="alertdialog" aria-label="Boltac asks">
          <p><b>Unusable item.</b> A {asks.class} cannot use the {asks.item}. Buy it anyway for {asks.price} GP?</p>
          <div class="row">
            <button class="primary" disabled={busy} onclick={() => act({ action: 'confirm', yes: true })}>Buy anyway</button>
            <button disabled={busy} onclick={() => act({ action: 'confirm', yes: false })}>No</button>
          </div>
        </div>
      {:else if said}
        <p class="boltac" aria-live="polite"><span class="muted">Boltac:</span> “{said}”</p>
      {/if}

      {#if last}
        <div class="receipt {last.tone}" aria-label="Last transaction">
          <div>{last.did}.</div>
          <div class="status">{last.status}</div>
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
</LocationFrame>

<style>
  .lead { margin: 0 0 4px; }
  .customer { display: grid; gap: 4px; }
  .customer .who b { color: var(--accent); font-size: 15px; }
  dl { display: grid; grid-template-columns: auto 1fr; gap: 1px 10px; margin: 0; font-size: 13px; }
  dt { color: var(--muted); }
  dd { margin: 0; font-variant-numeric: tabular-nums; }
  .members { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 6px; }
  .members button.current { background: var(--selected); color: var(--text); }
  .counter { position: sticky; top: 0; z-index: 1; background: var(--panel); padding: 10px 0 6px; margin-top: -10px;
    border-bottom: 1px solid var(--line); display: grid; gap: 6px; }
  .ask { margin: 0; color: var(--text); }
  .ask p { margin: 0 0 6px; }
  .boltac { margin: 0; color: var(--gold); }
  .receipt { padding: 6px 10px; border-radius: 4px; font-size: 13px; border-left: 3px solid; }
  .receipt { display: flex; flex-wrap: wrap; gap: 2px 10px; align-items: baseline; }
  .receipt .status { font-weight: 600; font-size: 12px; }
  .receipt.saved { background: #1e2a1e; border-color: var(--accent); }
  .receipt.saved .status { color: var(--accent); }
  .receipt.notDurable { background: #2e2a1a; border-color: var(--gold); }
  .receipt.notDurable .status { color: var(--gold); }
  .receipt.unsaved { background: #3a1d1b; border-color: var(--error); }
  .receipt.unsaved .status { color: var(--error); }
  .shop { margin-top: 0; padding: 10px 12px; }
  .tabs { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
  .who-line { margin-left: auto; font-size: 12px; color: var(--muted); }
  .who-line b { color: var(--text); font-weight: 600; }
  .tab { margin: 0; }
  .tab.on { background: var(--selected); color: var(--text); }
  .tablewrap { overflow-x: auto; }
  table.lines td { vertical-align: top; }
  /* A line brought into view stops below the counter, not under it. */
  table.lines tbody tr { scroll-margin-top: 150px; }
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
