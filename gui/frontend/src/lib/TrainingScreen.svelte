<script>
  // The Training Grounds (Castle-0 C0-4): outside the castle walls, where
  // characters are created and the roster is inspected; a character's sheet
  // offers renaming and deleting here. The creation steps are the earlier
  // ones, unchanged; every rule (names, points, classes) is Go's.
  //
  // Props:
  //   view      the Go View (view.location is "training")
  //   busy      a request is running
  //   call      (fn, ...args) => Promise   runs a Session method
  //   selected  roster index whose sheet is open, or -1
  //   onselect  (rosterIndex) => void      open or close a character's sheet
  //   onopen    (rosterIndex) => void      open a party member's sheet
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'

  let { view, busy = false, call, selected = -1, onselect, onopen } = $props()

  let newName = $state('')
  let password = $state('')
  let confirm = $state('')
  let editName = $state('')

  const c = $derived(view.creation)
  const creating = $derived(view.screen === 'confirmCreate' || view.screen === 'creation')
  const letter = (i) => String.fromCharCode(65 + i)
</script>

{#snippet classTable(c, editable)}
  <div class="points-left" class:done={c.points === 0}>
    Bonus points left: <span>{c.points}</span>
  </div>
  <div class="tablewrap">
  <table class="guide">
    <thead>
      <tr>
        <th>Class</th>
        {#each view.statNames as s}<th class="num">{s.slice(0, 3)}</th>{/each}
        <th>Alignment</th><th>Status</th>
      </tr>
    </thead>
    <tbody>
      <tr class="current">
        <td>Current</td>
        {#each c.stats as v, i}
          <td class="num stat" title="Race base {c.baseStats[i]}">
            {#if editable}
              <button class="step up" aria-label="Add a point to {view.statNames[i]}"
                disabled={busy || !c.canAdd[i]} onclick={() => call(S.AddPoint, i)}>▲</button>
              <div class="value">{v}</div>
              <button class="step down" aria-label="Remove a point from {view.statNames[i]}"
                disabled={busy || !c.canRemove[i]} onclick={() => call(S.RemovePoint, i)}>▼</button>
            {:else}
              <div class="value">{v}</div>
            {/if}
          </td>
        {/each}
        <td>{c.alignment}</td>
        <td class="muted">{#if editable}▲ add · ▼ remove{/if}</td>
      </tr>
      {#each c.guide as row}
        <tr>
          <td>{row.class}</td>
          {#each row.minimums as min, i}
            <td class="num" class:short={row.shortfall[i] > 0}>
              {min || '—'}{#if row.shortfall[i] > 0}{' '}(−{row.shortfall[i]}){/if}
            </td>
          {/each}
          <td>{row.alignments}</td>
          <td class="status {row.status}">{row.note}</td>
        </tr>
      {/each}
    </tbody>
  </table>
  </div>
  <p class="muted">Class rows show minimums; (−N) is the current shortfall. Hover a value for its race base.</p>
{/snippet}

<LocationFrame {view} {busy} {call} {onopen}>
  {#snippet actions()}
    {#if creating}
      <p class="note">Creating <b>{c?.name ?? 'a character'}</b>. Keep it at the last step; ← Castle (or Esc)
        discards it.</p>
    {:else}
      <form class="create" onsubmit={(e) => { e.preventDefault(); call(S.StartCreation, newName) }}>
        <label for="newname">Create a character</label>
        <div class="row">
          <input id="newname" maxlength={view.nameMax} bind:value={newName} autocomplete="off" placeholder="Name" />
          <button class="primary" type="submit" disabled={busy}>Create</button>
        </div>
      </form>
      <p class="muted">Up to {view.nameMax} characters: letters, digits, space, hyphen and apostrophe.</p>
      <p class="muted">Click a character in the roster to inspect, rename or delete it.</p>
    {/if}
  {/snippet}

  {#if view.screen === 'confirmCreate'}
    <section>
      <h2>Create a character</h2>
      <div class="banner message">{view.message}</div>
      <div class="row" style="margin-top:10px">
        <button class="primary" disabled={busy} onclick={() => { password = ''; confirm = ''; call(S.AnswerCreate, true) }}>Yes</button>
        <button disabled={busy} onclick={() => call(S.AnswerCreate, false)}>No</button>
      </div>
    </section>
  {:else if view.screen === 'creation' && c}
    <section>
      <h2>Creating {c.name}</h2>
      <div class="summary">
        {#if c.race}Race <b>{c.race}</b>{/if}
        {#if c.alignment} · Alignment <b>{c.alignment}</b>{/if}
        {#if c.class} · Class <b>{c.class}</b>{/if}
      </div>

      {#if c.step === 'name'}
        <form class="row" onsubmit={(e) => { e.preventDefault(); call(S.SetName, editName) }}>
          <label style="margin:0"><span>Name</span>
            <input maxlength={view.nameMax} bind:value={editName} autocomplete="off" />
          </label>
          <button class="primary" type="submit" disabled={busy}>Continue</button>
          <button type="button" disabled={busy} onclick={() => call(S.CancelCreation)}>Cancel creation</button>
        </form>

      {:else if c.step === 'password'}
        <form onsubmit={(e) => { e.preventDefault(); call(S.SetPassword, password, confirm); password = ''; confirm = '' }}>
          <label><span>Password</span><input type="password" maxlength={view.nameMax} bind:value={password} /></label>
          <label><span>Enter it again</span><input type="password" maxlength={view.nameMax} bind:value={confirm} /></label>
          <p class="muted">Leave both empty for no password.</p>
          <button class="primary" type="submit" disabled={busy}>Continue</button>
        </form>

      {:else if c.step === 'race'}
        <p>Choose a race:</p>
        <div class="row">
          {#each view.races as r, i}
            <button disabled={busy} onclick={() => call(S.ChooseRace, i)}>{letter(i)}) {r}</button>
          {/each}
        </div>
        <div class="row" style="margin-top:10px">
          <button disabled={busy} onclick={() => { editName = c.name; call(S.Back) }}>Back to name</button>
        </div>

      {:else if c.step === 'alignment'}
        <p>Choose an alignment:</p>
        <div class="row">
          {#each view.alignments as a, i}
            <button disabled={busy} onclick={() => call(S.ChooseAlignment, i)}>{letter(i)}) {a}</button>
          {/each}
        </div>
        <div class="row" style="margin-top:10px">
          <button disabled={busy} onclick={() => call(S.Back)}>Back to race</button>
        </div>

      {:else if c.step === 'stats'}
        {#if c.guide}{@render classTable(c, true)}{/if}
        <button class="primary" disabled={busy} onclick={() => call(S.FinishStats)}>Done — choose class</button>

      {:else if c.step === 'class'}
        <p>Choose a class (only classes these statistics and alignment allow are enabled):</p>
        <div class="row">
          {#each view.classes as cl, i}
            <button disabled={busy || !c.classOk[i]} onclick={() => call(S.ChooseClass, i)}>{letter(i)}) {cl}</button>
          {/each}
        </div>
        <div class="row" style="margin-top:10px">
          <button disabled={busy} onclick={() => call(S.Back)}>Back to points</button>
        </div>
        {#if c.guide}{@render classTable(c, false)}{/if}

      {:else if c.step === 'confirm'}
        <table style="width:auto; margin-bottom:10px">
          <tbody>
            {#each view.statNames as s, i}
              <tr><td>{s}</td><td class="num">{c.stats[i]}</td></tr>
            {/each}
            <tr><td>Password</td><td>{c.hasPassword ? 'set' : 'none'}</td></tr>
          </tbody>
        </table>
        <p>Keep this character? Hit points, gold and age are rolled when you keep it.</p>
        <button class="primary" disabled={busy} onclick={() => call(S.Confirm, true)}>Keep and save</button>
        <button disabled={busy} onclick={() => call(S.Confirm, false)}>Discard</button>
      {/if}
    </section>
  {:else}
    <section class="roster-section">
      <h2>Roster ({view.roster?.length ?? 0} / 20)</h2>
      {#if view.roster?.length}
        <div class="tablewrap">
        <table class="roster">
          <thead>
            <tr>
              <th>Name</th><th>Class</th><th class="narrow-hide">Race</th><th class="num">Lvl</th>
              <th class="num">HP</th><th class="num">AC</th><th>Status</th>
              <th class="num narrow-hide">Gold</th><th class="num narrow-hide">Age</th>
              {#each view.statNames as s}<th class="num">{s.slice(0, 3)}</th>{/each}
              <th>Party</th>
            </tr>
          </thead>
          <tbody>
            {#each view.roster as ch}
              <tr class="pick" class:selected={ch.index === selected} onclick={() => onselect?.(ch.index)}>
                <td>{ch.name}</td><td>{ch.short}</td><td class="narrow-hide">{ch.race}</td>
                <td class="num">{ch.level}</td><td class="num">{ch.hp}/{ch.maxHp}</td>
                <td class="num">{ch.ac}</td><td>{ch.status}</td>
                <td class="num narrow-hide">{ch.gold}</td><td class="num narrow-hide">{ch.age}</td>
                {#each ch.stats as v}<td class="num">{v}</td>{/each}
                <td>{#if ch.inParty}<span class="inparty">#{ch.partySlot}</span>{:else if ch.mazeLevel}<span class="muted">maze {ch.mazeLevel}</span>{/if}</td>
              </tr>
            {/each}
          </tbody>
        </table>
        </div>
      {:else}
        <p class="muted">No characters yet. Create one with the form beside the picture.</p>
      {/if}
    </section>
  {/if}
</LocationFrame>

<style>
  .create label { display: block; margin: 0 0 6px; color: var(--muted); font-size: 12px; }
  .create .row { flex-wrap: nowrap; }
  .create input { flex: 1; min-width: 0; margin: 0; }
  .create button { margin: 0; }
  .note { margin: 0; }
  section { margin-top: 0; }
  .inparty { color: var(--accent); font-weight: 600; }
</style>
