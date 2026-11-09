<script>
  // The Temple of Cant (T-3), in the location frame. The patients are Go's
  // (view.temple: everyone the Temple can treat, in the party or not, and
  // the LOST, never treatable), with their condition, donation and, when
  // they cannot be treated, Go's reason; lib/temple.js reshapes them and
  // decides nothing. The payer is chosen from the party bar (or the list
  // here) among those Go allows to tithe. Treat… opens the screen's own
  // confirmation (patient, condition, donation, payer, the outcomes and the
  // risk in words, never a chance); it calls nothing, so Cancel and Escape
  // change nothing. Confirming is one call (Session.TempleAction): the
  // donation and the treatment together, one save, Go's receipt.
  //
  // Props:
  //   view    the Go View (view.location is "temple")
  //   busy    a request is running
  //   call    (fn, ...args) => Promise   runs a Session method
  //   onopen  (rosterIndex) => void      open a character's sheet
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'
  import { patients, payers, defaultPayer, pickLabel, confirmation, receipt } from './temple.js'

  let { view, busy = false, call, onopen } = $props()

  let payer = $state(-1) // the party slot chosen to tithe
  let chosen = $state(-1) // the roster index being confirmed, or -1
  let cancelButton = $state(null)

  const temple = $derived(view.temple)
  const rows = $derived(patients(temple))
  const members = $derived(payers(temple))
  // The payer stays the one chosen (at first, the member who may tithe
  // with the most gold) until another is chosen or they may no longer.
  $effect(() => {
    if (!members.some((p) => p.slot === payer && !p.refusal)) payer = defaultPayer(temple)
  })
  const by = $derived(payer)
  const tither = $derived(members.find((p) => p.slot === by))
  const ask = $derived(chosen >= 0 ? confirmation(temple, chosen, by) : null)
  const last = $derived(receipt(temple))
  const partyShown = $derived(view.settings?.partyVisible ?? true)

  const act = (request) => call(S.TempleAction, request)

  function open(index) {
    chosen = index
    queueMicrotask(() => cancelButton?.focus())
  }
  const cancel = () => { chosen = -1 }
  async function confirm() {
    const index = chosen
    chosen = -1
    await act({ action: 'treat', patient: index, payer: by })
  }
  // Escape cancels the confirmation (and does not leave the Temple).
  function onkey(e) {
    if (e.key === 'Escape') {
      e.preventDefault()
      cancel()
    }
  }
</script>

<LocationFrame {view} {busy} {call} {onopen} pick={(slot) => pickLabel(temple, slot, by)} onpick={(slot) => { payer = slot }} current={by}>
  {#snippet actions()}
    {#if tither}
      <div class="tither" aria-label="Who will tithe">
        <div class="who"><span class="muted">Tithing:</span> <b>{tither.name}</b> · {tither.gold} GP</div>
        <div class="row">
          <button class="small" disabled={busy} onclick={() => act({ action: 'pool', payer: by })}
            title="Give {tither.name} every other party member's gold (Pool Gold)">Pool party gold</button>
        </div>
        <p class="muted hint">{partyShown ? 'Click a party member to tithe instead.' : 'Choose who will tithe:'}</p>
      </div>
    {:else}
      <p class="muted">No party member can tithe: only a member who is well may pay.</p>
    {/if}
    {#if !partyShown || members.some((p) => p.refusal)}
      <div class="members" aria-label="Party members">
        {#each members as p (p.slot)}
          {#if !partyShown || p.refusal}
            <button class="small" class:current={p.slot === by} disabled={busy || !pickLabel(temple, p.slot, by)}
              title={p.refusal || `${p.name} tithes`} onclick={() => { payer = p.slot }}>{p.slot + 1}. {p.name} · {p.gold} GP</button>
            {#if p.refusal}<span class="why">{p.refusal}</span>{/if}
          {/if}
        {/each}
      </div>
    {/if}
  {/snippet}

  <section class="temple">
    {#if ask}
      <div class="banner message ask" class:grave={ask.permanent} role="alertdialog" aria-label="Confirm the treatment" tabindex="-1" onkeydown={onkey}>
        <p class="head"><b>{ask.patient}</b> · {ask.condition} · donation <b>{ask.donation} GP</b>
          {#if ask.payer}, tithed by <b>{ask.payer}</b> ({ask.payerGold} GP){/if}</p>
        <p class="risk">{ask.risk}</p>
        <ul class="outcomes">{#each ask.outcomes as o (o)}<li>{o}</li>{/each}</ul>
        {#if ask.permanent}<p class="warn">If this fails, {ask.patient} is lost forever: no donation, spell or rescue can bring them back.</p>{/if}
        <p class="muted">{ask.charged}</p>
        {#if !ask.ready}<p class="why">{ask.why}</p>{/if}
        <div class="row">
          <button class="primary" disabled={busy || !ask.ready} onclick={confirm}>Pay {ask.donation} GP and pray</button>
          <button bind:this={cancelButton} disabled={busy} onclick={cancel}>Cancel</button>
        </div>
      </div>
    {/if}

    {#if last}
      <div class="receipt {last.tone} {last.mood}" aria-label="Last treatment">
        <div class="title"><b>{last.title}</b> <span class="status">{last.status}</span>
          {#if last.tone === 'unsaved'}
            <button class="small" disabled={busy} onclick={() => call(S.SaveNow)}
              title="Writes the game as this window has it; nothing is charged or prayed for again">Try saving again</button>
          {/if}
        </div>
        <ul>{#each last.lines as l, i (i)}<li>{l}</li>{/each}</ul>
      </div>
    {/if}

    {#if rows.length}
      <div class="tablewrap">
        <table class="patients">
          <thead><tr><th>Patient</th><th>Condition</th><th class="num">Donation</th><th></th></tr></thead>
          <tbody>
            {#each rows as r (r.key)}
              <tr class:lost={r.lost} class:refused={!r.treatable} class:chosen={r.index === chosen}>
                <td class="name">
                  <button class="link" onclick={() => onopen?.(r.index)} title="Open {r.name}'s sheet">{r.name}</button>
                  <span class="muted">{r.who}{r.inParty ? ' · in the party' : ''}</span>
                </td>
                <td class="cond {r.kind || 'lost'}">{r.condition}</td>
                <td class="num">{r.donation === null ? '—' : `${r.donation} GP`}</td>
                <td class="act">
                  {#if r.treatable}
                    <button class="small primary" disabled={busy || chosen >= 0} onclick={() => open(r.index)}>Treat…</button>
                    {#if r.note}<span class="note" class:short={r.funding !== 'payer'}>{r.note}</span>{/if}
                  {:else}
                    <span class="why">{r.lost ? 'Beyond mortal help.' : r.reason}</span>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else}
      <p class="muted">No one on the roster needs the Temple's help.</p>
    {/if}
    <p class="notes muted">The priests cure paralysis and stone, and may raise the dead and those in ashes. Only
      characters at the castle can be treated; anyone left in the maze must be brought back first.</p>
  </section>
</LocationFrame>

<style>
  .tither { display: grid; gap: 4px; font-size: 13px; }
  .tither .who b { color: var(--accent); }
  .hint { margin: 0; font-size: 12px; }
  .members { display: flex; flex-wrap: wrap; gap: 4px 6px; margin-top: 6px; align-items: baseline; }
  .members button.current { background: var(--selected); color: var(--text); }
  .members .why { color: var(--muted); font-size: 11px; }
  .temple { margin-top: 0; padding: 10px 12px; display: grid; gap: 8px; }
  .ask { margin: 0; color: var(--text); display: grid; gap: 4px; outline: none; }
  .ask p { margin: 0; }
  .ask .risk { font-weight: 600; }
  .ask.grave { border-left: 3px solid var(--error); }
  .ask .warn { color: var(--error); font-weight: 600; }
  .ask .why { color: var(--gold); }
  .ask .outcomes { margin: 0; padding-left: 18px; }
  .receipt { padding: 6px 10px; border-radius: 4px; font-size: 13px; border-left: 3px solid; }
  .receipt .status { font-weight: 600; font-size: 12px; margin-left: 6px; }
  .receipt ul { margin: 4px 0 0; padding-left: 18px; columns: 2 220px; }
  .receipt.saved { background: #1e2a1e; border-color: var(--accent); }
  .receipt.saved .status { color: var(--accent); }
  .receipt.notDurable { background: #2e2a1a; border-color: var(--gold); }
  .receipt.notDurable .status { color: var(--gold); }
  .receipt.unsaved { background: #3a1d1b; border-color: var(--error); }
  .receipt.unsaved .status { color: var(--error); }
  .receipt.ashes .title b { color: var(--gold); }
  .receipt.lost .title b { color: var(--error); }
  .tablewrap { overflow-x: auto; }
  table.patients td { vertical-align: top; }
  td.name .link { background: none; border: none; padding: 0; margin: 0 6px 0 0; color: var(--text); font-weight: 600; cursor: pointer; }
  td.name .link:hover { text-decoration: underline; }
  td.name .muted { font-size: 11px; display: block; }
  td.cond.raise, td.cond.ashes { color: var(--gold); }
  td.cond.lost { color: var(--error); }
  tr.lost td { color: var(--muted); }
  tr.chosen { background: var(--selected); }
  td.num { white-space: nowrap; }
  td.act { white-space: nowrap; }
  td.act .note { display: block; font-size: 11px; color: var(--muted); white-space: normal; max-width: 260px; }
  td.act .note.short { color: var(--gold); }
  td.act .why { display: block; white-space: normal; color: var(--muted); font-size: 11px; max-width: 260px; }
  .notes { margin: 0; font-size: 12px; }
  button.small { font-size: 12px; padding: 1px 10px; margin: 0; }
</style>
