<script>
  // The page only renders the Go-side View and calls Session methods. Every
  // rule (eligibility, point limits, messages) comes from app.Controller.
  import * as S from '../wailsjs/go/main/Session.js'

  let view = $state(null)
  let busy = $state(false)
  let newName = $state('')
  let password = $state('')
  let confirm = $state('')
  let editName = $state('')
  let selected = $state('')        // name of the roster row being viewed
  let deleting = $state(false)     // delete confirmation open
  let deletePassword = $state('')
  let resetting = $state(false)    // reset confirmation open

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
  const sel = $derived(view?.roster?.find((ch) => ch.name === selected))

  function select(name) {
    selected = selected === name ? '' : name
    deleting = false
    deletePassword = ''
  }

  async function deleteSelected() {
    await call(S.DeleteCharacter, selected, deletePassword)
    deletePassword = ''
    if (!view.error) { deleting = false; selected = '' }
  }

  async function resetRoster() {
    await call(S.ResetRoster)
    resetting = false
    selected = ''
  }

  const spellSlots = (cur, max) =>
    max.map((m, i) => (m ? `L${i + 1} ${cur[i]}/${m}` : '')).filter(Boolean).join('  ') || 'none'
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

      <section>
        <h2>Roster ({view.roster?.length ?? 0} / 20)</h2>
        {#if view.roster?.length}
          <table>
            <thead>
              <tr>
                <th>Name</th><th>Class</th><th>Race</th><th class="num">Lvl</th>
                <th class="num">HP</th><th class="num">AC</th><th>Status</th>
                <th class="num">Gold</th><th class="num">Age</th>
                {#each view.statNames as s}<th class="num">{s.slice(0, 3)}</th>{/each}
              </tr>
            </thead>
            <tbody>
              {#each view.roster as ch}
                <tr class="pick" class:selected={ch.name === selected}
                  onclick={() => view.screen === 'roster' && select(ch.name)}>
                  <td>{ch.name}</td><td>{ch.short}</td><td>{ch.race}</td>
                  <td class="num">{ch.level}</td><td class="num">{ch.hp}/{ch.maxHp}</td>
                  <td class="num">{ch.ac}</td><td>{ch.status}</td>
                  <td class="num">{ch.gold}</td><td class="num">{ch.age}</td>
                  {#each ch.stats as v}<td class="num">{v}</td>{/each}
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="muted">No characters yet.</p>
        {/if}
        {#if view.screen === 'roster'}
          {#if view.roster?.length}<p class="muted">Click a character to see details.</p>{/if}
          <div class="row" style="margin-top:6px">
            <button disabled={busy} onclick={() => call(S.ReloadRoster)}>Reload from disk</button>
            <button disabled={busy} onclick={() => { resetting = true }}>Reset roster…</button>
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
        {/if}
      </section>

      {#if view.screen === 'roster' && sel}
        <section>
          <h2>{sel.name}</h2>
          <div class="details">
            <div><span>Race</span>{sel.race}</div>
            <div><span>Class</span>{sel.class}</div>
            <div><span>Alignment</span>{sel.alignment}</div>
            <div><span>Level</span>{sel.level}</div>
            <div><span>Experience</span>{sel.xp}{#if sel.nextXp}{' '}(next level {sel.nextXp}){/if}</div>
            <div><span>Hit points</span>{sel.hp} / {sel.maxHp}</div>
            <div><span>Armour class</span>{sel.ac}</div>
            <div><span>Status</span>{sel.status}{#if sel.poison} (poisoned {sel.poison}){/if}</div>
            <div><span>Gold</span>{sel.gold}</div>
            <div><span>Age</span>{sel.age} years</div>
            <div><span>Password</span>{sel.hasPassword ? 'set' : 'none'}</div>
            {#if sel.mazeLevel}<div><span>Location</span>In the maze, level {sel.mazeLevel}</div>{/if}
            {#if sel.legacy}<div><span>Legacy</span>yes</div>{/if}
          </div>
          <table style="width:auto; margin-top:10px">
            <tbody>
              <tr>{#each view.statNames as s}<th class="num">{s}</th>{/each}</tr>
              <tr>{#each sel.stats as v}<td class="num"><b>{v}</b></td>{/each}</tr>
            </tbody>
          </table>
          <div class="details" style="margin-top:10px">
            <div><span>Mage spell slots</span>{spellSlots(sel.mageSpells, sel.maxMage)}</div>
            <div><span>Priest spell slots</span>{spellSlots(sel.priestSpells, sel.maxPriest)}</div>
            <div class="wide"><span>Known spells</span>{sel.spells?.join(', ') || 'none'}</div>
            <div class="wide"><span>Items</span>
              {#if sel.items?.length}
                {#each sel.items as it, i}{i ? ', ' : ''}{it.equipped ? '*' : ''}{it.name}{it.cursed ? ' (cursed)' : ''}{/each}
                <span class="muted key">* = equipped</span>
              {:else}none{/if}
            </div>
          </div>

          <div class="row" style="margin-top:12px">
            {#if !deleting}
              <button class="danger" disabled={busy} onclick={() => { deleting = true }}>Delete…</button>
              <button disabled={busy} onclick={() => select(sel.name)}>Close</button>
            {/if}
          </div>
          {#if deleting}
            <form class="banner message confirm" onsubmit={(e) => { e.preventDefault(); deleteSelected() }}>
              Delete <b>{sel.name}</b> permanently? Enter the character's password to confirm.
              <div class="row" style="margin-top:8px">
                <input type="password" maxlength={view.nameMax} bind:value={deletePassword} placeholder="password (blank if none)" />
                <button class="danger" type="submit" disabled={busy}>Delete</button>
                <button type="button" disabled={busy} onclick={() => { deleting = false; deletePassword = '' }}>Cancel</button>
              </div>
            </form>
          {/if}
        </section>
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
    {/if}
  {:else}
    <p class="muted">Loading…</p>
  {/if}
</main>
