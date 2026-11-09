<script>
  // The Adventurer's Inn (I-4), in the location frame. The guest is chosen
  // from the party bar (or, with the party hidden, the list here) and is
  // marked there. The rooms, their rates, how long a stay lasts, what it
  // costs and the HP it leaves are Go's (view.inn, and Session.InnQuote for
  // a chosen number of weeks; lib/inn.js): a Stay button is enabled exactly
  // when Go gives no refusal, and a disabled one says Go's reason. The mode
  // and the weeks typed are the page's own and only ask Go for quotes. A
  // stay, with its level check, is one call (Session.InnAction) and one
  // save; its receipt is Go's. "← Castle" and Escape leave the Inn.
  //
  // Props:
  //   view    the Go View (view.location is "inn")
  //   busy    a request is running
  //   call    (fn, ...args) => Promise   runs a Session method
  //   onopen  (rosterIndex) => void      open a party member's sheet
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'
  import { guest, rows, maxWeeks, pickLabel, receipt } from './inn.js'

  let { view, busy = false, call, onopen } = $props()

  let mode = $state('healed') // 'healed' (the original's stay) or 'weeks'
  let weeks = $state(1)
  let quotes = $state({}) // room -> Go's quote for `weeks`, in 'weeks' mode

  const inn = $derived(view.inn)
  const who = $derived(guest(inn, view.roster))
  const last = $derived(receipt(inn))
  const most = $derived(maxWeeks(inn))
  const lines = $derived(rows(inn, mode, weeks, quotes))
  const partyShown = $derived(view.settings?.partyVisible ?? true)
  const guests = $derived(inn?.view?.guests ?? [])

  const act = (request) => call(S.InnAction, request)
  const enter = (slot) => act({ action: 'enter', slot })

  // Go's quotes for the chosen weeks, asked again whenever the weeks, the
  // guest or the guest's state (any new view) changes.
  $effect(() => {
    if (mode !== 'weeks' || !who) return
    const n = Number(weeks)
    const ask = (inn?.view?.rooms ?? []).filter((r) => !r.free)
    let stale = false
    quotes = {}
    Promise.all(ask.map((r) => S.InnQuote(r.room, n).then((q) => [r.room, q], (e) => [r.room, { refusal: String(e) }])))
      .then((qs) => { if (!stale) quotes = Object.fromEntries(qs) })
    return () => { stale = true }
  })
</script>

<LocationFrame {view} {busy} {call} {onopen} pick={(slot) => pickLabel(inn, slot)} onpick={enter} current={who?.slot ?? -1}>
  {#snippet actions()}
    {#if who}
      <div class="guest" aria-label="Guest">
        <div class="who"><b>{who.name}</b> <span class="muted">{who.class} · level {who.level}</span></div>
        <dl>
          <dt>Hit points</dt><dd>{who.hp} / {who.maxHp}</dd>
          <dt>Gold</dt><dd>{who.gold} GP</dd>
          <dt>Age</dt><dd>{who.age} years</dd>
          {#if who.mage}<dt>Mage spells</dt><dd>{who.mage}</dd>{/if}
          {#if who.priest}<dt>Priest spells</dt><dd>{who.priest}</dd>{/if}
          <dt>Experience</dt><dd>{who.xp}{who.nextXp ? ` · level ${who.level + 1} at ${who.nextXp}` : ''}</dd>
        </dl>
        <div class="row">
          {#if who.index !== undefined}<button class="small" disabled={busy} onclick={() => onopen?.(who.index)}>Character sheet</button>{/if}
          <button class="small" disabled={busy} onclick={() => act({ action: 'pool' })}
            title="Give {who.name} every other party member's gold (Pool Gold)">Pool party gold</button>
          <button class="small" disabled={busy} onclick={() => act({ action: 'leave-guest' })}
            title="Back to WHO WILL STAY">Check out</button>
        </div>
      </div>
    {:else}
      <p class="lead">Welcome to the Adventurer's Inn. Who will stay?</p>
      <p class="muted">{partyShown ? 'Click a party member below.' : 'Choose a party member:'}</p>
    {/if}
    {#if !partyShown || guests.some((g) => g.refusal)}
      <div class="members" aria-label="Party members">
        {#each guests as g (g.slot)}
          {#if !partyShown || g.refusal}
            <button class="small" class:current={who?.slot === g.slot} disabled={busy || !pickLabel(inn, g.slot)}
              title={g.refusal || `${g.name} stays`} onclick={() => enter(g.slot)}>{g.slot + 1}. {g.name}</button>
            {#if g.refusal}<span class="why">{g.refusal}</span>{/if}
          {/if}
        {/each}
      </div>
    {/if}
  {/snippet}

  <section class="inn">
    <div class="counter">
      <div class="modes" role="radiogroup" aria-label="How long to stay">
        <button role="radio" class="tab" aria-checked={mode === 'healed'} class:on={mode === 'healed'}
          onclick={() => { mode = 'healed' }}>Rest until healed</button>
        <button role="radio" class="tab" aria-checked={mode === 'weeks'} class:on={mode === 'weeks'}
          onclick={() => { mode = 'weeks' }}>Choose weeks</button>
        {#if mode === 'weeks'}
          <label class="weeks">Weeks
            <input type="number" min="1" max={Math.max(1, most)} step="1" bind:value={weeks} disabled={!who} />
          </label>
        {/if}
        {#if who}
          <span class="who-line" aria-label="Guest"><b>{who.name}</b> · HP <b>{who.hp}</b>/{who.maxHp} ·
            <b>{who.gold}</b> GP</span>
        {/if}
      </div>
      <p class="explain muted">
        {#if mode === 'healed'}
          Until fully healed or out of gold; every week is charged in full.
        {:else}
          Only the weeks chosen{most ? ` (up to ${most} here)` : ''}; the guest may leave partly healed.
        {/if}
      </p>

      {#if last}
        <div class="receipt {last.tone}" aria-label="Last stay">
          <div class="title"><b>{last.title}.</b> <span class="status">{last.status}</span>
            {#if last.tone === 'unsaved'}
              <button class="small" disabled={busy} onclick={() => call(S.SaveNow)}
                title="Writes the game as this window has it; nothing is healed, charged or rolled again">Try saving again</button>
            {/if}
          </div>
          <ul>
            {#each last.lines as l, i (i)}<li class={l.kind}>{l.text}</li>{/each}
          </ul>
          {#if last.messages?.length}<p class="said">{last.messages.join(' · ')}</p>{/if}
        </div>
      {/if}
    </div>

    {#if inn?.view?.guestRefusal}
      <p class="refusal" role="status">{inn.view.guestRefusal} Choose another guest.</p>
    {:else if lines.length}
      <div class="tablewrap">
        <table class="rooms">
          <thead><tr><th>Room</th><th>Rate</th><th>{mode === 'healed' ? 'Until healed' : `For ${weeks} week${Number(weeks) === 1 ? '' : 's'}`}</th><th></th></tr></thead>
          <tbody>
            {#each lines as l (l.key)}
              <tr class:free={l.free} class:refused={!!l.reason}>
                <td class="name">{l.name}{#if l.free}<span class="mark">free</span>{/if}</td>
                <td class="rate">{l.rate}</td>
                <td class="quote">{l.text}{#if l.short}<span class="note">{l.short}</span>{/if}</td>
                <td class="act">
                  <button class="small" class:primary={!l.reason} disabled={busy || !!l.reason} onclick={() => act(l.request)}>Stay</button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <p class="notes muted">Every stay restores spell points and ends with a level check. Age matters past 50: it
        weighs on the attribute changes at each new level. Istary does not yet age characters for Inn stays — the
        original's rate is not established.</p>
    {:else}
      <p class="muted">Choose who will stay: click a party member below.</p>
    {/if}
  </section>
</LocationFrame>

<style>
  .lead { margin: 0 0 4px; }
  .guest { display: grid; gap: 4px; }
  .guest .who b { color: var(--accent); font-size: 15px; }
  dl { display: grid; grid-template-columns: auto 1fr; gap: 1px 10px; margin: 0; font-size: 13px; }
  dt { color: var(--muted); }
  dd { margin: 0; font-variant-numeric: tabular-nums; }
  .members { display: flex; flex-wrap: wrap; gap: 4px 6px; margin-top: 6px; align-items: baseline; }
  .members button.current { background: var(--selected); color: var(--text); }
  .members .why { color: var(--muted); font-size: 11px; }
  /* Five rooms fit below the counter, so it does not stick: a long receipt
     never covers them. */
  .counter { padding: 0 0 6px; border-bottom: 1px solid var(--line); display: grid; gap: 6px; }
  .modes { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
  .tab { margin: 0; }
  .tab.on { background: var(--selected); color: var(--text); }
  .weeks { display: inline-flex; gap: 6px; align-items: center; font-size: 13px; margin-left: 6px; }
  .weeks input { width: 64px; margin: 0; }
  .who-line { margin-left: auto; font-size: 12px; color: var(--muted); }
  .who-line b { color: var(--text); font-weight: 600; }
  .explain { margin: 0; font-size: 12px; }
  .notes { margin: 8px 0 0; font-size: 12px; }
  .refusal { margin: 8px 0 0; color: var(--error); font-weight: 600; }
  .receipt { padding: 6px 10px; border-radius: 4px; font-size: 13px; border-left: 3px solid; }
  .receipt .status { font-weight: 600; font-size: 12px; margin-left: 6px; }
  .receipt ul { margin: 4px 0 0; padding-left: 18px; columns: 2 220px; }
  .receipt li.gain, .receipt li.level { color: var(--accent); }
  .receipt li.loss { color: var(--gold); }
  .receipt li.bad { color: var(--error); font-weight: 600; }
  .receipt .said { margin: 4px 0 0; color: var(--gold); font-size: 12px; }
  .receipt.saved { background: #1e2a1e; border-color: var(--accent); }
  .receipt.saved .status { color: var(--accent); }
  .receipt.notDurable { background: #2e2a1a; border-color: var(--gold); }
  .receipt.notDurable .status { color: var(--gold); }
  .receipt.unsaved { background: #3a1d1b; border-color: var(--error); }
  .receipt.unsaved .status { color: var(--error); }
  .inn { margin-top: 0; padding: 10px 12px; }
  .tablewrap { overflow-x: auto; }
  table.rooms td { vertical-align: top; }
  td.name { min-width: 120px; white-space: nowrap; }
  td.name .mark { display: inline-block; margin-left: 6px; padding: 0 5px; border-radius: 3px; font-size: 10px;
    text-transform: uppercase; letter-spacing: 0.04em; background: var(--card); color: var(--accent); }
  td.rate { color: var(--muted); font-size: 12px; }
  tr.refused td.quote { color: var(--muted); }
  td.quote .note { display: block; font-size: 11px; color: var(--gold); }
  td.act { white-space: nowrap; }
  button.small { font-size: 12px; padding: 1px 10px; margin: 0; }
</style>
