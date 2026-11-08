<script>
  // The party bar (Castle-0): the six positions of the party being played,
  // the front row and the back row, empty positions included, with each
  // member's class, level, HP and condition. It renders view.partyBar (Go's;
  // the practice copies during practice) and nothing else, and reports a
  // click on a member whose character sheet can be opened.
  //
  // Props:
  //   slots     view.partyBar
  //   frontRow  view.frontRow
  //   onopen    (rosterIndex) => void   open the character sheet (optional)
  import { formation, condition, unwell, hpShare, canOpen } from './party.js'

  let { slots = [], frontRow = 3, onopen } = $props()

  const rows = $derived(formation(slots, frontRow))
</script>

{#snippet member(p)}
  <span class="num">{p.slot}</span>
  {#if p.name}
    <span class="name">{p.name}</span>
    <span class="meta">{p.class} · Lv {p.level}</span>
    <span class="hp" title="HP {p.hp} of {p.maxHp}">
      <span class="value">{p.hp}/{p.maxHp}</span>
      <span class="bar"><span style:width="{hpShare(p) * 100}%"></span></span>
    </span>
    <span class="cond" class:bad={unwell(p)}>{condition(p)}</span>
  {:else}
    <span class="muted empty">empty</span>
  {/if}
{/snippet}

<div class="partybar" aria-label="Party">
  {#each rows as row (row.label)}
    <div class="row" class:front={row.label === 'Front'}>
      <div class="label">{row.label} <span class="muted">· {row.note}</span></div>
      <div class="slots">
        {#each row.slots as p (p.slot)}
          {#if onopen && canOpen(p)}
            <button class="slot" onclick={() => onopen(p.index)} aria-label="{p.name}: open the character sheet">{@render member(p)}</button>
          {:else}
            <div class="slot" class:vacant={!p.name} title={p.name || null}>{@render member(p)}</div>
          {/if}
        {/each}
      </div>
    </div>
  {/each}
</div>

<style>
  .partybar { display: grid; grid-template-columns: 1fr; gap: 8px; }
  @media (min-width: 700px) { .partybar { grid-template-columns: 1fr 1fr; } }
  .row { border: 1px solid var(--line); border-radius: 6px; padding: 4px 8px 6px; min-width: 0; }
  .row.front { border-color: #3d5a3d; background: #18201a; }
  .label { font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--notice); margin: 2px 0 4px; }
  .row.front .label { color: var(--accent); }
  .slots { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 6px; }
  .slot {
    display: grid; grid-template-columns: 14px minmax(0, 1fr); grid-template-areas: "num name" "num meta" "num hp" "num cond";
    gap: 0 6px; align-items: center; text-align: left; min-width: 0; min-height: 72px;
    margin: 0; padding: 4px 6px; border: 1px solid #2c2c2c; border-radius: 4px; background: #202020; font-size: 12px; line-height: 1.3;
  }
  button.slot { cursor: pointer; color: var(--text); }
  button.slot:hover { border-color: var(--accent); }
  .slot.vacant { background: transparent; border-style: dashed; }
  .num { grid-area: num; color: var(--muted); font-variant-numeric: tabular-nums; align-self: start; }
  .name { grid-area: name; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .meta { grid-area: meta; font-size: 11px; color: var(--muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .hp { grid-area: hp; display: flex; gap: 6px; align-items: center; }
  .hp .value { font-variant-numeric: tabular-nums; min-width: 42px; }
  .bar { flex: 1; height: 4px; background: #333; border-radius: 2px; overflow: hidden; }
  .bar span { display: block; height: 100%; background: var(--accent); }
  .cond { grid-area: cond; font-size: 11px; color: var(--muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .cond.bad { color: var(--error); }
  .empty { grid-area: name; }
</style>
