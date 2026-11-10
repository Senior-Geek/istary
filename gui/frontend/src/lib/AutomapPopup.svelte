<script>
  // The full map (M-1): an explored level in its fixed 20x20 frame, over
  // the dungeon screen in the same window. It asks Go for the level
  // (Session.Automap, which changes nothing) and draws it with MapSvg; the
  // level buttons list the levels Go says have been explored. F10 and
  // Escape close it (DungeonScreen handles the keys); while it is open no
  // key reaches the maze.
  //
  // Props:
  //   onclose  () => void
  import * as S from '../../wailsjs/go/main/Session.js'
  import MapSvg from './MapSvg.svelte'
  import { levelLabel } from './automap.js'

  let { onclose } = $props()

  let level = $state(0) // 0: the party's level (Go chooses)
  let map = $state(null)
  let error = $state('')
  let dialog = $state(null)

  $effect(() => {
    const l = level
    let stale = false
    S.Automap(l).then((m) => { if (!stale) { map = m; error = '' } }, (e) => { if (!stale) { map = null; error = String(e) } })
    return () => { stale = true }
  })
  $effect(() => { dialog?.focus() })
</script>

<div class="backdrop" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) onclose() }}>
  <div class="automap" role="dialog" aria-modal="true" aria-label="Map" tabindex="-1" bind:this={dialog}>
    <header>
      <h3>Map{map ? ` · ${levelLabel(map.level)}` : ''}</h3>
      {#if (map?.levels ?? []).length > 1}
        <div class="levels" role="group" aria-label="Level">
          {#each map.levels as l (l)}
            <button class="small" aria-pressed={l === map.level} onclick={() => { level = l }}>{l}</button>
          {/each}
        </div>
      {/if}
      <div class="spacer"></div>
      <button class="small" onclick={onclose}>Close <kbd>F10</kbd></button>
    </header>
    {#if map}
      <div class="sheet"><MapSvg view={map} full label="{levelLabel(map.level)}, explored squares" /></div>
      <p class="legend">
        <span class="k visited"></span>visited <span class="k wall"></span>wall <span class="k door"></span>door <span class="k oneway"></span>one side only
        <span class="k wrap">›</span>way across the edge · only what the party has seen
        {#if !map.party}· the party is not on this level{/if}
      </p>
    {:else}
      <p class="muted">{error || 'Loading…'}</p>
    {/if}
  </div>
</div>

<style>
  .backdrop { position: absolute; inset: 0; z-index: 20; display: grid; place-items: center; background: rgba(0, 0, 0, 0.55); }
  .automap { display: grid; gap: 8px; padding: 12px; border-radius: 8px; border: 1px solid var(--line);
    background: #0d0c0b; width: min(92%, 640px); max-height: 94%; outline: none; grid-template-rows: auto minmax(0, 1fr) auto; }
  header { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  h3 { margin: 0; font-size: 15px; }
  .levels { display: flex; gap: 4px; flex-wrap: wrap; }
  .levels button[aria-pressed='true'] { background: var(--selected); }
  .spacer { flex: 1; }
  .sheet { min-height: 0; aspect-ratio: 1; max-height: 70vh; justify-self: center; width: 100%; max-width: 70vh; }
  .legend { margin: 0; font-size: 11px; color: var(--muted); display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
  .k { display: inline-block; width: 16px; height: 0; border-top: 2px solid #4fd16a; }
  .k.visited { width: 12px; height: 12px; border: none; background: rgba(79, 209, 106, 0.07); outline: 1px solid rgba(79, 209, 106, 0.18); }
  .k.door { border-top: 4px solid #f2c14e; width: 8px; }
  .k.oneway { border-top: 2px dashed #4fd16a; }
  .k.wrap { border: none; width: auto; height: auto; color: #4fd16a; font-weight: 700; }
  .small { font-size: 11px; }
  kbd { font: 600 10px ui-monospace, Menlo, monospace; border: 1px solid var(--line); border-radius: 3px; padding: 0 3px; }
</style>
