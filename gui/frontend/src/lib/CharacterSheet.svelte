<script>
  // A character sheet. It renders one CharacterView from Go and reports the
  // player's requests through callbacks; it never calls the backend itself.
  // UX-3B: it is embedded in the place's artwork area (LocationFrame), not
  // a window: it fills the area, its header (who the character is) stays
  // in view, and the rest scrolls inside it.
  //
  // Its sections are Overview (vitals and attributes), Equipment (Worn &
  // Wielded and the Pack: one inventory, each item once), Spells, and the
  // place's own (Party at the Tavern, Delete at the Training Grounds). Each
  // is its own snippet, rendered in order, so tabs can later show one at a
  // time without changing them.
  //
  // Props:
  //   character  CharacterView (see gui/view.go)
  //   statNames  attribute names from the View
  //   nameMax    longest allowed name
  //   busy       disable actions while a request is running
  //   error      last error from Go, shown inside the sheet
  //   onrename   (newName) => Promise   rename (optional: where renaming is allowed)
  //   ondelete   (password) => Promise  delete (optional: where deleting is allowed)
  //   onjoin     () => Promise   ask to join the party (optional)
  //   onleave    () => Promise   leave the party (optional)
  //   onequip    (n, equip) => Promise  equip or take off item n (optional: in town;
  //              what may be equipped, and why not, is Go's, on each item)
  import { kindLabel, wornAndPack, removeAction, useAction } from './equip.js'

  let { character: ch, statNames, nameMax, busy = false, error = '', onrename, ondelete, onjoin, onleave, onequip } = $props()

  let renaming = $state(false)
  let newName = $state('')
  let deleting = $state(false)
  let password = $state('')

  const lists = $derived(wornAndPack(ch))
  const note = $derived(onequip ? ch.equipNote || '' : 'not here')

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

{#snippet overview()}
  <section aria-label="Vitals">
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

  <section aria-label="Attributes">
    <h3>Attributes</h3>
    <table class="attrs">
      <tbody>
        <tr>{#each statNames as s}<th>{s}</th>{/each}</tr>
        <tr>{#each ch.stats as v}<td>{v}</td>{/each}</tr>
      </tbody>
    </table>
  </section>
{/snippet}

{#snippet item(it, a)}
  <li>
    <span class="what">{it.name}
      {#if it.kind}{' '}<span class="kind">{kindLabel(it)}</span>{/if}
      {#if !it.identified}{' '}<span class="tag">unidentified</span>{/if}
      {#if it.cursed}{' '}<span class="tag bad">cursed</span>{/if}
    </span>
    {#if a}
      <span class="equip">
        {#if a.reason}<span class="why">{a.reason}</span>{/if}
        <button class="small" class:primary={a.enabled && a.label === 'Use'} disabled={busy || !a.enabled}
          onclick={() => onequip(it.n, a.label === 'Use')}>{a.label}</button>
      </span>
    {/if}
  </li>
{/snippet}

{#snippet equipment()}
  <section class="equipment" aria-label="Equipment">
    <h3>Equipment <span class="count" aria-label="Items carried">{lists.carried}</span></h3>
    <h4>Worn &amp; Wielded</h4>
    {#if lists.worn.length}
      <ul class="items worn">{#each lists.worn as it (it.pos)}{@render item(it, removeAction(it, note))}{/each}</ul>
    {:else}<p class="muted none">Nothing worn or wielded.</p>{/if}
    <h4>Pack</h4>
    {#if lists.pack.length}
      <ul class="items pack">{#each lists.pack as it (it.pos)}{@render item(it, useAction(it, note))}{/each}</ul>
    {:else}<p class="muted none">The pack is empty.</p>{/if}
    {#if onequip && ch.equipNote}<p class="muted">Equipment cannot be changed now: {ch.equipNote}</p>{/if}
    {#if onequip && !ch.equipNote && ch.items?.length}<p class="muted">Use equips an item from the pack and Remove
      returns one to it; the armour class changes at once. An item that is cursed, or of another alignment,
      sticks once worn.</p>{/if}
  </section>
{/snippet}

{#snippet spells()}
  <section aria-label="Spells">
    <h3>Spells</h3>
    <div class="grid">
      <div><span>Mage slots</span>{slots(ch.mageSpells, ch.maxMage)}</div>
      <div><span>Priest slots</span>{slots(ch.priestSpells, ch.maxPriest)}</div>
    </div>
    <p><span class="muted">Known:</span> {ch.spells?.join(', ') || 'none'}</p>
  </section>
{/snippet}

{#snippet here()}
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

  {#if ondelete}
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
  {/if}
{/snippet}

<div class="sheet" aria-label="Character sheet: {ch.name}">
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
          {#if onrename}
            <button class="small" disabled={busy} onclick={() => { newName = ch.name; renaming = true }}>Rename</button>
          {/if}
        </div>
      {/if}
      <div class="line">{ch.race} {ch.class} · {ch.alignment} · Level {ch.level}</div>
      <div class="line muted">
        {ch.status}{#if ch.poison}{' '}(poisoned {ch.poison}){/if}
        {#if ch.inParty}{' '}· in party (slot {ch.partySlot}){:else if ch.stranded}{' '}· stranded in the maze at {ch.stranded}{:else if ch.mazeLevel}{' '}· in the maze, level {ch.mazeLevel}{/if}
        {#if ch.legacy}{' '}· legacy{/if}
      </div>
    </div>
  </header>

  <div class="pages">
    {#if error}<div class="banner error">{error}</div>{/if}
    <!-- Tabs, when they come, choose one of these instead of all in turn. -->
    {@render overview()}
    {@render equipment()}
    {@render spells()}
    {@render here()}
  </div>
</div>

<style>
  /* The sheet fills the area it is given: the header stays, the pages
     scroll inside. */
  .sheet { height: 100%; min-height: 0; display: grid; grid-template-rows: auto minmax(0, 1fr);
    box-sizing: border-box; border: 1px solid var(--line); border-radius: 6px; background: var(--panel); overflow: hidden; }
  .sheet header { display: flex; gap: 12px; align-items: flex-start; padding: 8px 12px; border-bottom: 1px solid var(--line); }
  .pages { overflow-y: auto; min-height: 0; padding: 0 12px 10px; }
  .portrait {
    flex: 0 0 56px; height: 64px;
    display: flex; align-items: center; justify-content: center; text-align: center;
    border: 1px dashed #555; border-radius: 4px;
    color: var(--muted); font-size: 10px;
  }
  .identity { flex: 1; min-width: 0; }
  .name { font-size: 17px; font-weight: 600; display: flex; gap: 10px; align-items: center; }
  .line { margin-top: 2px; font-size: 13px; }
  /* The sheet is one panel: its sections are separated by a rule, not
     boxed again. */
  section { margin: 0; padding: 10px 0 0; background: none; border: 0; border-radius: 0; }
  section + section { margin-top: 10px; border-top: 1px solid var(--line); }
  h3 { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); margin: 0 0 6px; }
  h3 .count { margin-left: 6px; color: var(--text); font-variant-numeric: tabular-nums; letter-spacing: 0; }
  h4 { font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); margin: 8px 0 3px; font-weight: 600; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: 3px 14px; }
  .grid span, p span { display: inline-block; width: 100px; color: var(--muted); }
  .attrs { width: auto; }
  .attrs th { font-size: 11px; color: var(--muted); font-weight: 500; text-align: center; }
  .attrs td { text-align: center; font-weight: 600; font-size: 15px; }
  ul.items { list-style: none; margin: 0; padding: 0; display: grid; gap: 3px; }
  ul.items li { display: flex; flex-wrap: wrap; gap: 2px 10px; align-items: center; justify-content: space-between;
    border-bottom: 1px solid var(--line); padding: 2px 0 3px 6px; }
  ul.worn li { border-left: 3px solid var(--notice); }
  ul.pack li { border-left: 3px dashed var(--line); }
  .none { margin: 0; font-size: 12px; }
  .kind { font-size: 11px; color: var(--muted); }
  .equip { display: flex; gap: 8px; align-items: center; margin-left: auto; }
  .why { font-size: 11px; color: var(--muted); text-align: right; }
  .tag { font-size: 11px; color: var(--muted); border: 1px solid #444; border-radius: 3px; padding: 0 4px; }
  .tag.bad { color: var(--error); border-color: var(--error); }
  button.small { font-size: 12px; padding: 2px 8px; margin: 0; }
  .refusal { color: var(--error); }
  .actions { border-top: 1px solid var(--line); padding-top: 10px; }
  .confirm .row { margin-top: 8px; }
</style>
