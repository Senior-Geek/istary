<script>
  // The Training Grounds (Castle-0 C0-4, UX-3B): outside the castle walls,
  // where characters are created and the roster is inspected; a
  // character's sheet (the frame's) offers renaming and deleting here.
  // Every rule (names, points, classes) is Go's; lib/creation.js only
  // words it.
  //
  // UX-3B: every decision of creation is in the Actions (the next one
  // always said first): Yes / No, the name and password, the race and
  // alignment, Back, Keep and save, Discard. The information below holds
  // the character so far, the attributes and the class table, or the
  // roster. The bonus points and the class are one screen: the class table
  // follows the points as they are spent, and once none are left the
  // classes Go allows are highlighted and chosen there (Session.PickClass),
  // straight to the review. Points can be moved until a class is chosen.
  //
  // Props:
  //   view      the Go View (view.location is "training")
  //   busy      a request is running
  //   call      (fn, ...args) => Promise   runs a Session method
  //   selected  roster index whose sheet is shown, or -1
  //   onselect  (rosterIndex) => void      show or hide a character's sheet
  //   onopen    (rosterIndex) => void      show a party member's sheet
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'
  import { stage, choosing, classRows, instruction } from './creation.js'

  let { view, busy = false, call, selected = -1, onselect, onopen } = $props()

  let newName = $state('')
  let password = $state('')
  let confirm = $state('')
  let editName = $state('')

  const c = $derived(view.creation)
  const at = $derived(stage(view))
  const creating = $derived(at !== '')
  const rows = $derived(classRows(view))
  const picking = $derived(choosing(view))
  const next = $derived(instruction(view))
  const letter = (i) => String.fromCharCode(65 + i)

  // A class from the table: from the points (PickClass), or at the older
  // class step (ChooseClass).
  const pick = (row) => call(at === 'class' ? S.ChooseClass : S.PickClass, row.index)
</script>

{#snippet so_far()}
  <div class="summary">
    Creating <b>{c.name}</b>
    {#if c.race}{' '}· Race <b>{c.race}</b>{/if}
    {#if c.alignment}{' '}· Alignment <b>{c.alignment}</b>{/if}
    {#if c.class}{' '}· Class <b>{c.class}</b>{/if}
  </div>
{/snippet}

{#snippet classTable()}
  <div class="tablewrap">
  <table class="guide" class:picking>
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
            {#if at === 'points' || at === 'choose'}
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
        <td class="muted">{#if at === 'points' || at === 'choose'}▲ add · ▼ remove{/if}</td>
      </tr>
      {#each rows as row (row.index)}
        <tr class="class-row" class:pickable={row.pickable} class:closed={picking && !row.pickable}
          onclick={() => { if (row.pickable && !busy) pick(row) }}>
          <td>
            {#if row.pickable}
              <button class="pick-class" disabled={busy} onclick={(e) => { e.stopPropagation(); pick(row) }}
                aria-label="Choose {row.class}">{row.class}</button>
            {:else}
              <span aria-disabled={picking ? 'true' : undefined}>{row.class}</span>
            {/if}
          </td>
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
  <p class="muted">Class rows show minimums; (−N) is the current shortfall. Hover a value for its race base.
    {#if picking}Highlighted classes can be chosen; points can still be moved until one is.{/if}</p>
{/snippet}

<LocationFrame {view} {busy} {call} {onopen}>
  {#snippet actions()}
    {#if creating}
      <div class="decision" aria-live="polite">
        <p class="next">{next}</p>

        {#if at === 'confirming'}
          <p class="message">{view.message}</p>
          <div class="row">
            <button class="primary" disabled={busy} onclick={() => { password = ''; confirm = ''; call(S.AnswerCreate, true) }}>Yes</button>
            <button disabled={busy} onclick={() => call(S.AnswerCreate, false)}>No</button>
          </div>

        {:else if at === 'name'}
          <form class="stack" onsubmit={(e) => { e.preventDefault(); call(S.SetName, editName) }}>
            <label><span>Name</span>
              <input maxlength={view.nameMax} bind:value={editName} autocomplete="off" />
            </label>
            <button class="primary" type="submit" disabled={busy}>Continue</button>
          </form>

        {:else if at === 'password'}
          <form class="stack" onsubmit={(e) => { e.preventDefault(); call(S.SetPassword, password, confirm); password = ''; confirm = '' }}>
            <label><span>Password</span><input type="password" maxlength={view.nameMax} bind:value={password} /></label>
            <label><span>Enter it again</span><input type="password" maxlength={view.nameMax} bind:value={confirm} /></label>
            <button class="primary" type="submit" disabled={busy}>Continue</button>
          </form>

        {:else if at === 'race'}
          <div class="choices">
            {#each view.races as r, i}
              <button disabled={busy} onclick={() => call(S.ChooseRace, i)}>{letter(i)}) {r}</button>
            {/each}
          </div>
          <button disabled={busy} onclick={() => { editName = c.name; call(S.Back) }}>Back to name</button>

        {:else if at === 'alignment'}
          <div class="choices">
            {#each view.alignments as a, i}
              <button disabled={busy} onclick={() => call(S.ChooseAlignment, i)}>{letter(i)}) {a}</button>
            {/each}
          </div>
          <button disabled={busy} onclick={() => call(S.Back)}>Back to race</button>

        {:else if at === 'points' || at === 'choose' || at === 'class'}
          <div class="points-left" class:done={c.points === 0}>
            Bonus points left: <span>{c.points}</span>
          </div>
          {#if at === 'class'}
            <button disabled={busy} onclick={() => call(S.Back)}>Back to points</button>
          {/if}

        {:else if at === 'confirm'}
          <div class="row">
            <button class="primary" disabled={busy} onclick={() => call(S.Confirm, true)}>Keep and save</button>
            <button disabled={busy} onclick={() => call(S.Confirm, false)}>Discard</button>
          </div>
        {/if}

        {#if at !== 'confirming' && at !== 'confirm'}
          <button class="discard" disabled={busy} onclick={() => call(S.DiscardCreation)}
            title="Discard this character and stay here">Discard</button>
        {/if}
        <p class="muted hint">← Castle (or Esc) also discards it.</p>
      </div>
    {:else}
      <form class="create" onsubmit={(e) => { e.preventDefault(); call(S.StartCreation, newName) }}>
        <label for="newname">Create a character</label>
        <div class="row">
          <input id="newname" maxlength={view.nameMax} bind:value={newName} autocomplete="off" placeholder="Name" />
          <button class="primary" type="submit" disabled={busy}>Create</button>
        </div>
      </form>
      <p class="muted">Up to {view.nameMax} characters: letters, digits, space, hyphen and apostrophe.</p>
      <p class="muted">Click a character in the roster to show, rename or delete it.</p>
    {/if}
  {/snippet}

  {#if creating && at !== 'confirming'}
    <section class="creation">
      {@render so_far()}
      {#if (at === 'points' || at === 'choose' || at === 'class') && c.guide}
        {@render classTable()}
      {:else if at === 'confirm'}
        <h3>Review</h3>
        <table class="review">
          <tbody>
            <tr><td>Name</td><td>{c.name}</td></tr>
            <tr><td>Race</td><td>{c.race}</td></tr>
            <tr><td>Alignment</td><td>{c.alignment}</td></tr>
            <tr><td>Class</td><td>{c.class}</td></tr>
            {#each view.statNames as s, i}
              <tr><td>{s}</td><td class="num">{c.stats[i]}</td></tr>
            {/each}
            <tr><td>Password</td><td>{c.hasPassword ? 'set' : 'none'}</td></tr>
          </tbody>
        </table>
        <p class="muted">Hit points, gold and age are rolled when the character is kept.</p>
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
              <tr class="pick" class:selected={ch.index === selected} onclick={() => { if (!creating) onselect?.(ch.index) }}>
                <td>{ch.name}</td><td>{ch.short}</td><td class="narrow-hide">{ch.race}</td>
                <td class="num">{ch.level}</td><td class="num">{ch.hp}/{ch.maxHp}</td>
                <td class="num">{ch.ac}</td><td>{ch.status}</td>
                <td class="num narrow-hide">{ch.gold}</td><td class="num narrow-hide">{ch.age}</td>
                {#each ch.stats as v}<td class="num">{v}</td>{/each}
                <td>{#if ch.inParty}<span class="inparty">#{ch.partySlot}</span>{:else if ch.stranded}<span class="muted" title="Stranded in the maze by a defeated party">stranded {ch.stranded}</span>{:else if ch.mazeLevel}<span class="muted">maze {ch.mazeLevel}</span>{/if}</td>
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
  .decision { display: grid; gap: 6px; }
  .decision p { margin: 0; }
  .decision button { margin: 0; }
  .next { font-weight: 600; color: var(--accent); }
  .message { font-size: 13px; }
  .stack { display: grid; gap: 6px; }
  .stack label { display: grid; gap: 2px; margin: 0; font-size: 12px; color: var(--muted); }
  .stack input { margin: 0; width: 100%; box-sizing: border-box; }
  .choices { display: grid; gap: 4px; }
  .choices button { text-align: left; }
  .discard { justify-self: start; }
  .hint { font-size: 11px; }
  section { margin-top: 0; }
  .summary { margin: 0 0 8px; }
  .review { width: auto; }
  .review td:first-child { color: var(--muted); padding-right: 16px; }
  .inparty { color: var(--accent); font-weight: 600; }
  /* Choosing a class: the classes Go allows are a quiet green and can be
     clicked; the others stay, dimmed, with their reasons. */
  table.guide.picking tr.pickable td { background: rgba(110, 190, 110, 0.13); cursor: pointer; }
  table.guide.picking tr.pickable:hover td { background: rgba(110, 190, 110, 0.22); }
  table.guide.picking tr.closed td { opacity: 0.55; }
  table.guide.picking tr.closed td.status { opacity: 0.85; }
  .pick-class { margin: 0; padding: 1px 8px; font-size: 12px; font-weight: 600; background: rgba(110, 190, 110, 0.25);
    border-color: rgba(110, 190, 110, 0.6); }
</style>
