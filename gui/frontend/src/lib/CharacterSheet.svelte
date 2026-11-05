<script>
  // A character sheet. It renders one CharacterView from Go and reports the
  // player's requests through callbacks; it never calls the backend itself,
  // so the same component can live in a panel today and in its own native
  // window later.
  //
  // Props:
  //   character  CharacterView (see gui/view.go)
  //   statNames  attribute names from the View
  //   nameMax    longest allowed name
  //   busy       disable actions while a request is running
  //   error      last error from Go, shown inside the sheet
  //   onrename   (newName) => Promise
  //   ondelete   (password) => Promise
  //   onjoin     () => Promise   ask to join the party (optional)
  //   onleave    () => Promise   leave the party (optional)
  let { character: ch, statNames, nameMax, busy = false, error = '', onrename, ondelete, onjoin, onleave } = $props()

  let renaming = $state(false)
  let newName = $state('')
  let deleting = $state(false)
  let password = $state('')

  const equipped = $derived(ch.items?.filter((it) => it.equipped) ?? [])

  function slots(cur, max) {
    return max.map((m, i) => (m ? `L${i + 1} ${cur[i]}/${m}` : '')).filter(Boolean).join('  ') || 'none'
  }

  async function submitRename(e) {
    e.preventDefault()
    await onrename?.(newName)
    if (!error) renaming = false
  }

  async function submitDelete(e) {
    e.preventDefault()
    await ondelete?.(password)
    password = ''
  }
</script>

<div class="sheet">
  <header>
    <div class="portrait" aria-label="Portrait">No portrait</div>
    <div class="identity">
      {#if renaming}
        <form class="row" onsubmit={submitRename}>
          <input maxlength={nameMax} bind:value={newName} aria-label="New name" />
          <button class="primary" type="submit" disabled={busy}>Save name</button>
          <button type="button" disabled={busy} onclick={() => { renaming = false }}>Cancel</button>
        </form>
      {:else}
        <div class="name">{ch.name}
          <button class="small" disabled={busy} onclick={() => { newName = ch.name; renaming = true }}>Rename</button>
        </div>
      {/if}
      <div class="line">{ch.race} {ch.class} · {ch.alignment} · Level {ch.level}</div>
      <div class="line muted">
        {ch.status}{#if ch.poison}{' '}(poisoned {ch.poison}){/if}
        {#if ch.inParty}{' '}· in party (slot {ch.partySlot}){:else if ch.mazeLevel}{' '}· in the maze, level {ch.mazeLevel}{/if}
        {#if ch.legacy}{' '}· legacy{/if}
      </div>
    </div>
  </header>

  {#if error}<div class="banner error">{error}</div>{/if}

  <section>
    <h3>Vitals</h3>
    <div class="grid">
      <div><span>Hit points</span>{ch.hp} / {ch.maxHp}</div>
      <div><span>Armour class</span>{ch.ac}</div>
      <div><span>Experience</span>{ch.xp}{#if ch.nextXp}{' '}(next {ch.nextXp}){/if}</div>
      <div><span>Gold</span>{ch.gold}</div>
      <div><span>Age</span>{ch.age} years</div>
      <div><span>Password</span>{ch.hasPassword ? 'set' : 'none'}</div>
    </div>
  </section>

  <section>
    <h3>Attributes</h3>
    <table class="attrs">
      <tbody>
        <tr>{#each statNames as s}<th>{s}</th>{/each}</tr>
        <tr>{#each ch.stats as v}<td>{v}</td>{/each}</tr>
      </tbody>
    </table>
  </section>

  <section>
    <h3>Equipment</h3>
    {#if equipped.length}
      <ul>{#each equipped as it}<li>{it.name}{#if it.cursed}{' '}<em>(cursed)</em>{/if}</li>{/each}</ul>
    {:else}<p class="muted">Nothing equipped.</p>{/if}
  </section>

  <section>
    <h3>Inventory ({ch.items?.length ?? 0} / 8)</h3>
    {#if ch.items?.length}
      <ul>
        {#each ch.items as it}
          <li>{it.name}
            {#if it.equipped}{' '}<span class="tag">equipped</span>{/if}
            {#if !it.identified}{' '}<span class="tag">unidentified</span>{/if}
            {#if it.cursed}{' '}<span class="tag bad">cursed</span>{/if}
          </li>
        {/each}
      </ul>
    {:else}<p class="muted">Carrying nothing.</p>{/if}
  </section>

  <section>
    <h3>Spells</h3>
    <div class="grid">
      <div><span>Mage slots</span>{slots(ch.mageSpells, ch.maxMage)}</div>
      <div><span>Priest slots</span>{slots(ch.priestSpells, ch.maxPriest)}</div>
    </div>
    <p><span class="muted">Known:</span> {ch.spells?.join(', ') || 'none'}</p>
  </section>

  {#if onjoin || onleave}
    <section class="partyline">
      <h3>Party</h3>
      {#if ch.inParty}
        In party, slot {ch.partySlot}.
        <button class="small" disabled={busy} onclick={() => onleave?.()}>Remove from party</button>
      {:else if !ch.joinRefusal}
        Not in the party.
        <button class="small" disabled={busy} onclick={() => onjoin?.()}>Add to party</button>
      {:else}
        Cannot join now: <span class="refusal">{ch.joinRefusal}</span>
      {/if}
    </section>
  {/if}

  <section class="actions">
    {#if deleting}
      <form class="confirm" onsubmit={submitDelete}>
        Delete <b>{ch.name}</b> permanently? Enter the character's password to confirm.
        <div class="row">
          <input type="password" maxlength={nameMax} bind:value={password} placeholder="password (blank if none)" />
          <button class="danger" type="submit" disabled={busy}>Delete</button>
          <button type="button" disabled={busy} onclick={() => { deleting = false; password = '' }}>Cancel</button>
        </div>
      </form>
    {:else}
      <button class="danger" disabled={busy} onclick={() => { deleting = true }}>Delete…</button>
    {/if}
  </section>
</div>

<style>
  .sheet header { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 8px; }
  .portrait {
    flex: 0 0 96px; height: 112px;
    display: flex; align-items: center; justify-content: center;
    border: 1px dashed #555; border-radius: 4px;
    color: var(--muted); font-size: 11px;
  }
  .identity { flex: 1; min-width: 0; }
  .name { font-size: 18px; font-weight: 600; display: flex; gap: 10px; align-items: center; }
  .line { margin-top: 2px; }
  section { margin-top: 12px; }
  h3 { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); margin: 0 0 6px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 3px 14px; }
  .grid span, p span { display: inline-block; width: 100px; color: var(--muted); }
  .attrs { width: auto; }
  .attrs th { font-size: 11px; color: var(--muted); font-weight: 500; text-align: center; }
  .attrs td { text-align: center; font-weight: 600; font-size: 15px; }
  ul { margin: 0; padding-left: 18px; }
  .tag { font-size: 11px; color: var(--muted); border: 1px solid #444; border-radius: 3px; padding: 0 4px; }
  .tag.bad { color: var(--error); border-color: var(--error); }
  button.small { font-size: 12px; padding: 2px 8px; }
  .refusal { color: var(--error); }
  .actions { border-top: 1px solid var(--line); padding-top: 10px; }
  .confirm .row { margin-top: 8px; }
</style>
