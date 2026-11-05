<script>
  // The party at Gilgamesh's Tavern. Renders the Go view's party data and
  // reports requests through callbacks; all rules (who may join, the
  // password, the front row) are decided in Go.
  //
  // Props:
  //   party       PartySlotView[] (see gui/view.go)
  //   partyMax    party size limit
  //   frontRow    number of front-row slots
  //   alignment   party alignment
  //   pending     name waiting for its password, or ''
  //   enabled     false while actions are unavailable (e.g. during creation)
  //   busy        a request is running
  //   onopen      (rosterIndex) => void       open the character sheet
  //   onleave     (slot0) => Promise          remove from the party
  //   onpassword  (password) => Promise       answer the pending join
  //   oncancel    () => Promise               abandon the pending join
  let { party = [], partyMax, frontRow, alignment, pending = '', enabled = true, busy = false,
    onopen, onleave, onpassword, oncancel } = $props()

  let password = $state('')

  // Fill empty positions so the six slots and the two rows are always shown.
  const slots = $derived(Array.from({ length: partyMax }, (_, i) => party.find((p) => p.slot === i + 1) ?? { slot: i + 1 }))
  const rows = $derived([
    { label: 'Front row', note: 'can fight in melee', slots: slots.slice(0, frontRow) },
    { label: 'Back row', note: 'cannot melee', slots: slots.slice(frontRow) },
  ])

  async function submit(e) {
    e.preventDefault()
    await onpassword?.(password)
    password = ''
  }
</script>

<div class="party">
  <div class="head">
    <span><b>{party.length}</b> / {partyMax}</span>
    <span class="muted">Alignment</span> <b>{alignment}</b>
  </div>

  {#if pending}
    <form class="banner message pending" onsubmit={submit}>
      <b>{pending}</b> may join. Enter the character's password:
      <div class="row" style="margin-top:6px">
        <input type="password" bind:value={password} placeholder="password (blank if none)" aria-label="Password for {pending}" />
        <button class="primary" type="submit" disabled={busy}>Join</button>
        <button type="button" disabled={busy} onclick={() => { password = ''; oncancel?.() }}>Cancel</button>
      </div>
    </form>
  {/if}

  {#each rows as row}
    <div class="rowgroup" class:front={row.label === 'Front row'}>
      <div class="rowlabel">{row.label} <span class="muted">· {row.note}</span></div>
      {#each row.slots as p}
        <div class="slot" class:empty={!p.name}>
          <span class="num">{p.slot}</span>
          {#if p.name}
            <button class="link" disabled={!enabled} onclick={() => onopen?.(p.index)}>{p.name}</button>
            <span class="muted">{p.short}</span>
            <span class="hp">{p.hp}/{p.maxHp}</span>
            <span class="status" class:bad={p.status !== 'OK'}>{p.status}</span>
            <button class="small" disabled={busy || !enabled} onclick={() => onleave?.(p.slot - 1)}
              aria-label="Remove {p.name} from the party">Remove</button>
          {:else}
            <span class="muted">empty</span>
          {/if}
        </div>
      {/each}
    </div>
  {/each}
  <p class="muted note">Members keep the order they joined. Reordering is done in camp, as in the original game.</p>
</div>

<style>
  .head { display: flex; gap: 8px; align-items: baseline; margin-bottom: 8px; }
  .head span:first-child { margin-right: auto; }
  .rowgroup { border: 1px solid var(--line); border-radius: 6px; padding: 4px 8px 6px; margin-top: 8px; }
  .rowgroup.front { border-color: #3d5a3d; background: #18201a; }
  .rowlabel { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--accent); margin: 2px 0 4px; }
  .rowgroup:not(.front) .rowlabel { color: var(--notice); }
  .slot { display: flex; align-items: center; gap: 8px; white-space: nowrap; padding: 3px 0; border-top: 1px solid #2a2a2a; min-height: 30px; }
  .slot:first-of-type { border-top: none; }
  .slot .num { width: 16px; color: var(--muted); text-align: right; font-variant-numeric: tabular-nums; }
  .slot .hp { margin-left: auto; font-variant-numeric: tabular-nums; }
  .slot .status { width: 64px; font-size: 12px; }
  .slot .status.bad { color: var(--error); }
  .slot.empty { color: var(--muted); }
  button.link { background: none; border: none; padding: 0; margin: 0; color: var(--text); font-weight: 600; cursor: pointer; text-align: left; }
  button.link:hover:not(:disabled) { color: var(--accent); text-decoration: underline; }
  button.small { font-size: 12px; padding: 1px 8px; margin: 0; }
  .pending { margin: 0 0 6px; color: var(--text); }
  .note { margin: 8px 0 0; font-size: 12px; }
</style>
