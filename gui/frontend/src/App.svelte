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
  const letter = (i) => String.fromCharCode(65 + i)
</script>

{#snippet guideTable(c)}
  <h2 style="margin-top:16px">Class Requirements Guide</h2>
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
        {#each c.stats as v}<td class="num">{v}</td>{/each}
        <td>{c.alignment}</td><td>{c.points} point{c.points === 1 ? '' : 's'} left</td>
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
  <p class="muted">Minimums per statistic; (−N) is the current shortfall.</p>
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
                <tr>
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
          <div class="row" style="margin-top:10px">
            <button disabled={busy} onclick={() => call(S.ReloadRoster)}>Reload from disk</button>
          </div>
        {/if}
      </section>

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
            <p>Bonus points left: <b>{c.points}</b></p>
            <table style="width:auto">
              <tbody>
                {#each view.statNames as s, i}
                  <tr>
                    <td>{s}</td>
                    <td class="num"><b>{c.stats[i]}</b></td>
                    <td class="muted">base {c.baseStats[i]}</td>
                    <td>
                      <button disabled={busy || !c.canRemove[i]} onclick={() => call(S.RemovePoint, i)}>−</button>
                      <button disabled={busy || !c.canAdd[i]} onclick={() => call(S.AddPoint, i)}>+</button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
            <button class="primary" disabled={busy} onclick={() => call(S.FinishStats)}>Done — choose class</button>
            {#if c.guide}{@render guideTable(c)}{/if}

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
            {#if c.guide}{@render guideTable(c)}{/if}

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
