<script>
  // The page only renders the Go-side View and calls Session methods. Every
  // rule (eligibility, point limits, messages) comes from app.Controller.
  import * as S from '../wailsjs/go/main/Session.js'
  import FloatingPanel from './lib/FloatingPanel.svelte'
  import CharacterSheet from './lib/CharacterSheet.svelte'
  import RestorePanel from './lib/RestorePanel.svelte'
  import PartyPanel from './lib/PartyPanel.svelte'

  let view = $state(null)
  let busy = $state(false)
  let newName = $state('')
  let password = $state('')
  let confirm = $state('')
  let editName = $state('')
  let selected = $state(-1)        // roster index shown in the character sheet
  let resetting = $state(false)    // reset confirmation open
  let backups = $state(null)       // backup list while the restore panel is open

  async function call(fn, ...args) {
    busy = true
    try {
      view = await fn(...args)
    } catch (e) {
      view = { ...view, error: String(e) }
    } finally {
      busy = false
    }
  }

  call(S.State)

  const c = $derived(view?.creation)
  const sel = $derived(view?.roster?.find((ch) => ch.index === selected))

  function select(index) {
    selected = selected === index ? -1 : index
  }

  async function renameSelected(name) {
    await call(S.RenameCharacter, selected, name)
  }

  async function deleteSelected(password) {
    await call(S.DeleteCharacter, sel.name, password)
    if (!view.error) selected = -1
  }

  // Party (Gilgamesh's Tavern). Rules and messages come from Go.
  const joinParty = (index) => call(S.JoinParty, index)
  const leaveParty = (slot) => call(S.LeaveParty, slot)
  const submitJoin = (pw) => call(S.SubmitJoinPassword, pw)
  const cancelJoin = () => call(S.CancelJoin)
  // A short form of the Tavern's refusal for the roster's Party column.
  const hint = (msg) => msg.replace(/^\*\* | \*\*$/g, '').replace(/ IS ALREADY IN THE PARTY!$/, ' IN PARTY')

  async function resetRoster() {
    await call(S.ResetRoster)
    resetting = false
    selected = -1
  }

  async function openRestore() {
    resetting = false
    backups = await S.ListBackups()
  }

  async function restore(file) {
    await call(S.RestoreBackup, file)
    if (!view.error) { backups = null; selected = -1 }
  }

  const letter = (i) => String.fromCharCode(65 + i)
</script>

{#snippet classTable(c, editable)}
  <div class="points-left" class:done={c.points === 0}>
    Bonus points left: <span>{c.points}</span>
  </div>
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
  <p class="muted">Class rows show minimums; (−N) is the current shortfall. Hover a value for its race base.</p>
{/snippet}

<main>
  <h1>Istary — Wizardry I Character Creation</h1>
  {#if view}
    <div class="muted">Saves: {view.saveDir}</div>

    {#if view.fatal}
      <div class="banner error">{view.fatal}</div>
    {:else}
      {#if view.error}<div class="banner error">{view.error}</div>{/if}
      {#if view.notice}<div class="banner notice">{view.notice}</div>{/if}

      <!-- Grid areas: wide = roster and rest beside the party; narrow =
           roster, party, rest stacked. -->
      <div class="layout">
      <section class="area-roster">
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
                <tr class="pick" class:selected={ch.index === selected}
                  onclick={() => view.screen === 'roster' && select(ch.index)}>
                  <td>{ch.name}</td><td>{ch.short}</td><td class="narrow-hide">{ch.race}</td>
                  <td class="num">{ch.level}</td><td class="num">{ch.hp}/{ch.maxHp}</td>
                  <td class="num">{ch.ac}</td><td>{ch.status}</td>
                  <td class="num narrow-hide">{ch.gold}</td><td class="num narrow-hide">{ch.age}</td>
                  {#each ch.stats as v}<td class="num">{v}</td>{/each}
                  <td class="partycell">
                    {#if ch.inParty}
                      <span class="inparty">#{ch.partySlot}</span>
                    {:else if !ch.joinRefusal}
                      <button class="small join" disabled={busy || view.screen !== 'roster'}
                        onclick={(e) => { e.stopPropagation(); joinParty(ch.index) }}>Join</button>
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
          <p class="muted">No characters yet.</p>
        {/if}
        {#if view.screen === 'roster'}
          {#if view.roster?.length}<p class="muted">Click a character to see details.</p>{/if}
          <div class="row" style="margin-top:6px">
            <button disabled={busy} onclick={() => call(S.ReloadRoster)}>Reload from disk</button>
            <button disabled={busy} onclick={() => { resetting = true; backups = null }}>Reset roster…</button>
            <button disabled={busy} onclick={openRestore}>Restore roster…</button>
          </div>
          {#if resetting}
            <div class="banner message confirm">
              Back up the current roster to <b>{view.saveDir}/backups</b> and start an empty roster?
              <div class="row" style="margin-top:8px">
                <button class="danger" disabled={busy} onclick={resetRoster}>Back up and reset</button>
                <button disabled={busy} onclick={() => { resetting = false }}>Cancel</button>
              </div>
            </div>
          {/if}
          {#if backups}
            <h2 style="margin-top:14px">Restore roster</h2>
            <RestorePanel {backups} {busy} onrestore={restore} oncancel={() => { backups = null }} />
          {/if}
        {/if}
      </section>

      <div class="area-rest">
      {#if view.screen === 'roster' && sel}
        {#key sel.index}
          <FloatingPanel title="Character — {sel.name}" onclose={() => { selected = -1 }}>
            <CharacterSheet character={sel} statNames={view.statNames} nameMax={view.nameMax}
              {busy} error={view.error} onrename={renameSelected} ondelete={deleteSelected}
              onjoin={() => joinParty(sel.index)} onleave={() => leaveParty(sel.partySlot - 1)} />
          </FloatingPanel>
        {/key}
      {/if}

      {#if view.screen === 'roster'}
        <section>
          <h2>Training Grounds</h2>
          <form class="row" onsubmit={(e) => { e.preventDefault(); call(S.StartCreation, newName) }}>
            <label style="margin:0"><span>Character name</span>
              <input maxlength={view.nameMax} bind:value={newName} autocomplete="off" />
            </label>
            <button class="primary" type="submit" disabled={busy}>Create</button>
          </form>
          <p class="muted">Up to {view.nameMax} characters: letters, digits, space, hyphen and apostrophe.</p>
        </section>
      {/if}

      {#if view.screen === 'confirmCreate'}
        <section>
          <h2>Training Grounds</h2>
          <div class="banner message">{view.message}</div>
          <div class="row" style="margin-top:10px">
            <button class="primary" disabled={busy} onclick={() => { password = ''; confirm = ''; call(S.AnswerCreate, true) }}>Yes</button>
            <button disabled={busy} onclick={() => call(S.AnswerCreate, false)}>No</button>
          </div>
        </section>
      {/if}

      {#if view.screen === 'creation' && c}
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
      {/if}
      </div>
      <aside class="area-party">
        <section>
          <h2>Party</h2>
          <PartyPanel party={view.party ?? []} partyMax={view.partyMax} frontRow={view.frontRow}
            alignment={view.partyAlignment} pending={view.pendingJoin} enabled={view.screen === 'roster'}
            {busy} onopen={(i) => { if (view.screen === 'roster') selected = i }}
            onleave={leaveParty} onpassword={submitJoin} oncancel={cancelJoin} />
        </section>
      </aside>
      </div>
    {/if}
  {:else}
    <p class="muted">Loading…</p>
  {/if}
</main>
