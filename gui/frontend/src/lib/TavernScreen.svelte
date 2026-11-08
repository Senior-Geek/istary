<script>
  // Gilgamesh's Tavern (Castle-0 C0-4): where the party is formed. The
  // roster says who may join and, for those who may not, the Tavern's
  // answer (Go's JoinRefusal); members can be removed. The party itself is
  // shown by the party bar below, as everywhere. Who may join, the
  // password and the order are Go's.
  //
  // Props:
  //   view      the Go View (view.location is "tavern")
  //   busy      a request is running
  //   call      (fn, ...args) => Promise   runs a Session method
  //   selected  roster index whose sheet is open, or -1
  //   onselect  (rosterIndex) => void      open or close a character's sheet
  //   onopen    (rosterIndex) => void      open a party member's sheet
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'

  let { view, busy = false, call, selected = -1, onselect, onopen } = $props()

  let password = $state('')

  const members = $derived(view.party?.length ?? 0)
  // A short form of the Tavern's refusal for the roster's Party column.
  const hint = (msg) => msg.replace(/^\*\* | \*\*$/g, '').replace(/ IS ALREADY IN THE PARTY!$/, ' IN PARTY')

  async function submit(e) {
    e.preventDefault()
    await call(S.SubmitJoinPassword, password)
    password = ''
  }
</script>

<LocationFrame {view} {busy} {call} {onopen}>
  {#snippet actions()}
    <p class="summary">Party <b>{members}</b> / {view.partyMax} · alignment <b>{view.partyAlignment}</b></p>
    {#if view.pendingJoin}
      <form class="banner message pending" onsubmit={submit}>
        <b>{view.pendingJoin}</b> may join. Enter the character's password:
        <input type="password" bind:value={password} placeholder="password (blank if none)"
          aria-label="Password for {view.pendingJoin}" />
        <div class="row">
          <button class="primary" type="submit" disabled={busy}>Join</button>
          <button type="button" disabled={busy} onclick={() => { password = ''; call(S.CancelJoin) }}>Cancel</button>
        </div>
      </form>
    {:else}
      <p class="muted">Choose Join beside a character to add them; Remove takes a member out. Members keep the
        order they joined; the order is changed in Camp, as in the original game.</p>
    {/if}
  {/snippet}

  <section>
    <h2>Who will join?</h2>
    {#if view.roster?.length}
      <div class="tablewrap">
      <table class="roster">
        <thead>
          <tr><th>Name</th><th>Class</th><th class="num">Lvl</th><th class="num">HP</th><th>Status</th><th>Party</th></tr>
        </thead>
        <tbody>
          {#each view.roster as ch}
            <tr class="pick" class:selected={ch.index === selected} onclick={() => onselect?.(ch.index)}>
              <td>{ch.name}</td><td>{ch.short}</td><td class="num">{ch.level}</td>
              <td class="num">{ch.hp}/{ch.maxHp}</td><td>{ch.status}</td>
              <td class="partycell">
                {#if ch.inParty}
                  <span class="inparty">#{ch.partySlot}</span>
                  <button class="small" disabled={busy} onclick={(e) => { e.stopPropagation(); call(S.LeaveParty, ch.partySlot - 1) }}
                    aria-label="Remove {ch.name} from the party">Remove</button>
                {:else if !ch.joinRefusal}
                  <button class="small join" disabled={busy}
                    onclick={(e) => { e.stopPropagation(); call(S.JoinParty, ch.index) }}>Join</button>
                {:else}
                  <span class="muted refusal" title={ch.joinRefusal}>{hint(ch.joinRefusal)}</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
      </div>
    {:else}
      <p class="muted">No characters yet. They are created at the Training Grounds, beyond the Edge of Town.</p>
    {/if}
  </section>
</LocationFrame>

<style>
  section { margin-top: 0; }
  .summary { margin: 0 0 8px; }
  .pending { margin: 0; color: var(--text); display: grid; gap: 6px; }
  .pending input { width: 100%; margin: 0; }
  .pending .row button { margin: 0; }
  .inparty { color: var(--accent); font-weight: 600; margin-right: 6px; }
  button.small { font-size: 12px; padding: 1px 8px; margin: 0; }
</style>
