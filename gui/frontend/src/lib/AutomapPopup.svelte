<script>
  // The full map (M-1): an explored level in its fixed 20x20 frame, over
  // the dungeon screen in the same window. It asks Go for the level
  // (Session.Automap, which changes nothing) and draws it with MapSvg; the
  // level buttons list the levels Go says have been explored. F10 and
  // Escape close it (DungeonScreen handles the keys); while it is open no
  // key reaches the maze.
  //
  // Where stranded characters lie on the level shown (R-3) is listed beside
  // the map, one line a marker ("L1:R11:C5 — 3 stranded"); choosing a line,
  // or clicking its marker, shows who lies there and their condition. Tab,
  // Enter and Space work the list, as they do the map's other buttons.
  // Other levels' markers are shown only when their level is chosen.
  //
  // Props:
  //   onclose  () => void
  import * as S from '../../wailsjs/go/main/Session.js'
  import MapSvg from './MapSvg.svelte'
  import Condition from './Condition.svelte'
  import { levelLabel, strandedLabel } from './automap.js'

  let { onclose } = $props()

  let level = $state(0) // 0: the party's level (Go chooses)
  let map = $state(null)
  let error = $state('')
  let dialog = $state(null)
  let chosen = $state('') // the place of the marker whose details are shown

  const stranded = $derived(map?.stranded ?? [])
  const site = $derived(stranded.find((m) => m.place === chosen) ?? null)
  const choose = (place) => { chosen = chosen === place ? '' : place }

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
            <button class="small" aria-pressed={l === map.level} onclick={() => { level = l; chosen = '' }}>{l}</button>
          {/each}
        </div>
      {/if}
      <div class="spacer"></div>
      <button class="small" onclick={onclose}>Close <kbd>F10</kbd></button>
    </header>
    {#if map}
      <div class="body" class:aside={stranded.length > 0}>
        <div class="sheet"><MapSvg view={map} full label="{levelLabel(map.level)}, explored squares" selected={chosen} onselect={choose} /></div>
        {#if stranded.length}
          <aside aria-label="Stranded on this level">
            <h4>Stranded on {levelLabel(map.level)}</h4>
            <ul class="sites">
              {#each stranded as m (m.place)}
                <li><button class="small" aria-pressed={m.place === chosen} onclick={() => choose(m.place)}>{strandedLabel(m)}</button></li>
              {/each}
            </ul>
            {#if site}
              <div class="site" aria-live="polite">
                <p class="where">{site.place} · {site.count} stranded</p>
                <ul>
                  {#each site.characters as ch (ch.name)}
                    <li><span>{ch.name} · {ch.class} · L{ch.level}</span> <Condition label={ch.condition} /></li>
                  {/each}
                </ul>
              </div>
            {:else}
              <p class="muted hint">Choose a place to see who lies there.</p>
            {/if}
          </aside>
        {/if}
      </div>
      <p class="legend">
        <span class="k visited"></span>visited <span class="k wall"></span>wall <span class="k door"></span>door <span class="k oneway"></span>one side only
        <span class="k wrap">›</span>way across the edge <span class="k stranded"></span>stranded · only what the party has seen
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
  /* With markers, a column beside the map lists them; the map keeps its size. */
  .automap:has(.body.aside) { width: min(96%, 880px); }
  .body { min-height: 0; display: grid; gap: 12px; }
  .body.aside { grid-template-columns: minmax(0, 1fr) 220px; }
  aside { min-height: 0; overflow-y: auto; overflow-x: hidden; font-size: 12px; display: grid; gap: 6px; align-content: start; }
  aside h4 { margin: 0; font-size: 12px; color: var(--muted); font-weight: 600; }
  aside ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 4px; }
  .sites button { width: 100%; text-align: left; font-family: ui-monospace, Menlo, monospace; }
  .sites button[aria-pressed='true'] { background: var(--selected); }
  .site { border-top: 1px solid var(--line); padding-top: 6px; }
  .site .where { margin: 0 0 4px; font: 700 12px ui-monospace, Menlo, monospace; color: #ff7a6b; }
  .site li { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; gap: 2px 6px; }
  .hint { margin: 0; }
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
  .k.stranded { width: 9px; height: 9px; border: 1px solid #ff7a6b; background: #5a1712; transform: rotate(45deg); }
  .k.wrap { border: none; width: auto; height: auto; color: #4fd16a; font-weight: 700; }
  .small { font-size: 11px; }
  kbd { font: 600 10px ui-monospace, Menlo, monospace; border: 1px solid var(--line); border-radius: 3px; padding: 0 3px; }
</style>
