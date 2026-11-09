<script>
  // The party bar (Castle-0): the six positions of the party being played,
  // three across: the front row (1-3) above the back row (4-6), empty
  // positions included. Each card is two lines: name, class and row; then
  // level, HP, AC and condition, with the HP left as a thin bar. It renders
  // view.partyBar (Go's; the practice copies during practice) and nothing
  // else, and reports a click on a member: in Camp, picking the member for
  // what Camp asks, or at Boltac's the customer (pick); elsewhere, opening
  // a character sheet (onopen).
  //
  // Props:
  //   slots     view.partyBar
  //   frontRow  view.frontRow
  //   onopen    (rosterIndex) => void   open the character sheet (optional)
  //   pick      (slot0) => label | null what a click on the member does in
  //             Camp, or null if nothing (optional; Camp's pickLabel)
  //   onpick    (slot0) => void         pick the member (with pick)
  //   current   slot0 of the member Camp is working with, or -1
  //   mark      (slot0) => string       a note on the card, e.g. "place 2"
  //   busy      a request is running
  import { condition, unwell, hpShare, canOpen } from './party.js'

  let { slots = [], frontRow = 3, onopen, pick, onpick, current = -1, mark, busy = false } = $props()
</script>

{#snippet member(p)}
  <span class="line1">
    <span class="num">{p.slot}</span>
    {#if p.name}
      <span class="name">{p.name}</span>
      <span class="cls">{p.class}</span>
    {:else}
      <span class="empty">empty</span>
    {/if}
    {#if mark?.(p.slot - 1)}<span class="mark">{mark(p.slot - 1)}</span>{/if}
    <span class="row">{p.front ? 'Front' : 'Back'}</span>
  </span>
  <span class="line2">
    {#if p.name}
      <span>Lv {p.level}</span>
      <span class="hp">HP <span class="value">{p.hp}/{p.maxHp}</span></span>
      <span>AC {p.ac}</span>
      <span class="cond" class:bad={unwell(p)}>{condition(p)}</span>
    {:else}&nbsp;{/if}
  </span>
  {#if p.name}<span class="bar" aria-hidden="true"><span style:width="{hpShare(p) * 100}%"></span></span>{/if}
{/snippet}

<div class="partybar" aria-label="Party" style:--across={frontRow}>
  {#each slots as p (p.slot)}
    {@const what = p.name && pick ? pick(p.slot - 1) : null}
    {#if what}
      <button class="slot pickable" class:current={current === p.slot - 1} class:back={!p.front} disabled={busy}
        onclick={() => onpick?.(p.slot - 1)} aria-label="{what} {p.name}" title="{what} {p.name}">{@render member(p)}</button>
    {:else if !pick && onopen && canOpen(p)}
      <button class="slot" class:back={!p.front} onclick={() => onopen(p.index)}
        aria-label="{p.name}: open the character sheet" title="{p.name}: open the character sheet">{@render member(p)}</button>
    {:else}
      <div class="slot" class:vacant={!p.name} class:current={current === p.slot - 1} class:back={!p.front}
        title={p.name || null}>{@render member(p)}</div>
    {/if}
  {/each}
</div>

<style>
  /* Three across: positions 1-3 (the front row) above 4-6 (the back row). */
  .partybar { display: grid; grid-template-columns: repeat(var(--across, 3), minmax(0, 1fr)); gap: 6px; }
  .slot {
    position: relative; display: grid; gap: 1px; min-width: 0; margin: 0; padding: 4px 8px 6px;
    text-align: left; font-size: 12px; line-height: 1.35; color: var(--text);
    background: var(--card); border: 1px solid transparent; border-radius: 5px; overflow: hidden;
  }
  .slot.back { background: var(--card-back); }
  .slot.vacant { background: transparent; border: 1px dashed var(--line); color: var(--muted); }
  button.slot { cursor: pointer; background: var(--btn); }
  button.slot:hover:not(:disabled) { background: var(--btn-hover); }
  button.slot:active:not(:disabled) { background: var(--btn-pressed); }
  button.slot.pickable { background: var(--pick); }
  button.slot.pickable:hover:not(:disabled) { background: var(--pick-hover); }
  .slot.current, button.slot.current { background: var(--selected); }
  .line1, .line2 { display: flex; gap: 6px; align-items: baseline; min-width: 0; white-space: nowrap; }
  .num { color: var(--muted); font-variant-numeric: tabular-nums; }
  .name { font-weight: 600; overflow: hidden; text-overflow: ellipsis; min-width: 0; }
  .cls { color: var(--muted); font-size: 11px; }
  .row { margin-left: auto; color: var(--muted); font-size: 10px; text-transform: uppercase; letter-spacing: 0.05em; }
  .mark { color: var(--gold); font-size: 11px; }
  .line2 { font-size: 11px; color: var(--muted); }
  .line2 .value { color: var(--text); font-variant-numeric: tabular-nums; }
  .cond { overflow: hidden; text-overflow: ellipsis; min-width: 0; color: var(--text); }
  .cond.bad { color: var(--error); }
  .bar { position: absolute; left: 0; right: 0; bottom: 0; height: 3px; background: rgba(0, 0, 0, 0.35); }
  .bar span { display: block; height: 100%; background: var(--hp); }
</style>
