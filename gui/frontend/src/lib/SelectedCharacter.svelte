<script>
  // The Selected Character panel at Boltac's (UX-2): the customer at the
  // counter, as Go's CharacterView gives them (view.roster): name, class,
  // level, condition, gold, AC, carrying, and their items, equipped apart
  // from carried. Equip and Take off are the character sheet's own
  // (lib/equip.js itemActions, Session.EquipAction): what may be worn, and
  // why not (class, a curse, an item already worn), is Go's. Nothing here
  // decides a rule. The item lists scroll on their own.
  //
  // Props:
  //   ch       the customer's CharacterView, or null (nobody at the counter)
  //   busy     a request is running
  //   locked   why equipment cannot be changed now ('' if it can), e.g.
  //            while Boltac waits for an answer
  //   onequip  (n, equip) => Promise   equip or take off item n
  import { itemActions, kindLabel } from './equip.js'
  import { wornAndCarried } from './shop.js'

  let { ch, busy = false, locked = '', onequip } = $props()

  const parts = $derived(wornAndCarried(ch))
  const note = $derived(locked || ch?.equipNote || '')
</script>

<aside class="selected" aria-label="Selected character">
  <h3>Selected character</h3>
  {#if ch}
    <div class="who">
      <b class="name">{ch.name}</b>
      <span class="muted">{ch.alignment} {ch.class} · Level {ch.level}</span>
      <span class="cond" class:bad={ch.status !== 'OK'}>{ch.status}</span>
    </div>
    <dl class="stats">
      <dt>Gold</dt><dd>{ch.gold} GP</dd>
      <dt>AC</dt><dd>{ch.ac}</dd>
      <dt>Carrying</dt><dd>{ch.items?.length ?? 0} of 8 · room for {8 - (ch.items?.length ?? 0)}</dd>
    </dl>
    <div class="inventory">
      {#snippet line(it)}
        {@const a = note ? null : itemActions(it, '')}
        <li>
          <span class="what">{it.name}
            {#if it.kind || !note}<span class="kind">{kindLabel(it)}</span>{/if}
            {#if !it.identified}<span class="tag">unidentified</span>{/if}
            {#if it.cursed}<span class="tag bad">cursed</span>{/if}
          </span>
          {#if a}
            <span class="act">
              <button class="small" class:primary={a.enabled && a.equip} disabled={busy || !a.enabled}
                onclick={() => onequip(it.n, a.equip)}>{a.label}</button>
              {#if a.reason && !(a.equip && it.equipped)}<span class="why">{a.reason}</span>{/if}
            </span>
          {/if}
        </li>
      {/snippet}
      <h4>Equipped <span class="muted">({parts.worn.length})</span></h4>
      {#if parts.worn.length}
        <ul class="items worn">{#each parts.worn as it (it.pos)}{@render line(it)}{/each}</ul>
      {:else}<p class="muted none">Nothing equipped.</p>{/if}
      <h4>Carried, not equipped <span class="muted">({parts.carried.length})</span></h4>
      {#if parts.carried.length}
        <ul class="items carried">{#each parts.carried as it (it.pos)}{@render line(it)}{/each}</ul>
      {:else}<p class="muted none">Nothing else carried.</p>{/if}
      {#if note}<p class="muted note">Equipment cannot be changed now: {note}</p>{/if}
    </div>
  {:else}
    <p class="muted">Nobody is at the counter. Click a party member below to serve them.</p>
  {/if}
</aside>

<style>
  .selected { margin: 0; padding: 10px 12px; display: grid; grid-template-rows: auto auto auto minmax(0, 1fr); gap: 6px; min-height: 0; height: 100%; box-sizing: border-box; }
  .selected:has(> p) { grid-template-rows: auto auto; }
  h3 { font-size: 12px; text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted); margin: 0; font-weight: 500; }
  h4 { font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); margin: 4px 0 2px; font-weight: 600; }
  h4:first-child { margin-top: 0; }
  .who { display: flex; flex-wrap: wrap; align-items: baseline; gap: 2px 8px; }
  .name { color: var(--accent); font-size: 15px; }
  .cond { font-size: 12px; font-weight: 600; }
  .cond.bad { color: var(--error); }
  dl { display: grid; grid-template-columns: auto 1fr; gap: 1px 10px; margin: 0; font-size: 13px; }
  dt { color: var(--muted); }
  dd { margin: 0; font-variant-numeric: tabular-nums; }
  .inventory { min-height: 0; overflow-y: auto; border-top: 1px solid var(--line); padding-top: 6px; }
  ul.items { list-style: none; margin: 0; padding: 0; display: grid; gap: 3px; }
  ul.items li { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; gap: 2px 8px; font-size: 13px; }
  ul.worn li { border-left: 3px solid var(--notice); padding-left: 6px; }
  ul.carried li { border-left: 3px dashed var(--line); padding-left: 6px; }
  .what { min-width: 0; }
  .kind, .tag { display: inline-block; margin-left: 4px; padding: 0 4px; border-radius: 3px; font-size: 10px; text-transform: uppercase;
    letter-spacing: 0.04em; background: var(--card); color: var(--muted); }
  .tag { color: var(--gold); }
  .tag.bad { color: var(--error); }
  .act { display: grid; justify-items: end; }
  .why { color: var(--muted); font-size: 11px; max-width: 200px; text-align: right; }
  .none, .note { margin: 0; font-size: 12px; }
  button.small { font-size: 12px; padding: 1px 10px; margin: 0; }
</style>
