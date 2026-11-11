<script>
  // The dungeon screen (M6a; real expeditions M6b-1). It renders view.dungeon (Go's app.DungeonView)
  // and calls Session.DungeonAction with one of the actions Go lists; it
  // keeps no maze state. The picture is a simple wireframe of the corridor
  // Go describes, not the Classic renderer.
  //
  // Inspect (R-2) lists each character found in the party's area, with a
  // Pick up button (keys 1-9) while the party has room, or the reason not.
  // R-3 adds each one's condition (Condition), and the rescue's progress as
  // Go reads it (inspect.js): the party's size and room, who else is here,
  // how many are known to lie elsewhere, and the members who cannot act.
  //
  // The maps (M-1): the minimap (view.minimap, Go's squares around the
  // party) sits in the corner of the view while walking, unless hidden
  // (a saved setting); Map or F10 opens the full map of the explored level
  // (AutomapPopup), which F10 or Escape closes. While it is open no key
  // reaches the maze, so looking at the map never takes a step.
  //
  // Props:
  //   view   the Go View (view.dungeon is the dungeon view)
  //   busy   a request is running
  //   call   (fn, ...args) => Promise   runs a Session method and stores the new view
  import * as S from '../../wailsjs/go/main/Session.js'
  import PartyBar from './PartyBar.svelte'
  import { viewBox, scene, points, actions, keyAction, expeditionTag } from './dungeon.js'
  import CampScreen from './CampScreen.svelte'
  import { pickLabel, placeMark } from './camp.js'
  import Condition from './Condition.svelte'
  import { progress } from './inspect.js'
  import MapSvg from './MapSvg.svelte'
  import AutomapPopup from './AutomapPopup.svelte'
  import { mapKey } from './automap.js'

  let { view, busy = false, call } = $props()

  const d = $derived(view.dungeon)
  const p = $derived(progress(d.inspect))
  const partyShown = $derived(view.settings?.partyVisible ?? true)
  const open = $derived(d.actions ?? [])
  const shapes = $derived(scene(d.corridor ?? []))
  const lines = $derived(d.lines ?? [])
  const moves = ['left', 'forward', 'right']

  const act = (action, button = 0) => call(S.DungeonAction, action, button)
  let campScreen = $state() // the Camp screen, whose pick the party bar uses
  const inCamp = $derived(d.phase === 'camp' && !!view.camp)
  const minimapShown = $derived(view.settings?.minimapVisible ?? true)
  let mapOpen = $state(false)

  function closeMap() {
    mapOpen = false
    // Keys go back to the maze (the window), not to a button left focused.
    queueMicrotask(() => document.activeElement?.blur?.())
  }

  function onkey(e) {
    const m = mapKey(e.key, mapOpen)
    if (m) {
      e.stopImmediatePropagation() // nothing behind the map hears it
      // Tab, Enter and Space still work the map's own buttons.
      if (!(m === 'block' && e.target.closest?.('.automap') && ['Tab', 'Enter', ' '].includes(e.key))) e.preventDefault()
      if (m === 'open') mapOpen = true
      if (m === 'close') closeMap()
      return
    }
    if (busy || e.target.closest?.('input, select, textarea') || e.metaKey || e.ctrlKey) return
    const a = keyAction(e.key, d)
    if (a) {
      e.preventDefault()
      act(a.action, a.button)
    }
  }
</script>

<svelte:window onkeydowncapture={onkey} />

<div class="dungeon">
  <header class="bar">
    <h2>{d.phase === 'camp' ? 'Camp' : 'The maze'}</h2>
    <span class="tag" class:real={view.real}>{expeditionTag(view)}</span>
    {#each d.spells ?? [] as s (s)}<span class="tag spell">{s}</span>{/each}
    <div class="spacer"></div>
    <button class="small" onclick={() => { mapOpen = true }} title="The explored level (F10)">Map <kbd>F10</kbd></button>
    <button class="small" aria-pressed={minimapShown} disabled={busy}
      onclick={() => call(S.SetMinimapVisible, !minimapShown)}>{minimapShown ? 'Hide minimap' : 'Show minimap'}</button>
    <button class="small" aria-pressed={partyShown} disabled={busy}
      onclick={() => call(S.SetPartyVisible, !partyShown)}>{partyShown ? 'Hide party' : 'Show party'}</button>
    {#if !view.real}
      <button class="small" disabled={busy} onclick={() => call(S.AbandonPractice)}>Abandon</button>
    {/if}
  </header>

  <div class="middle" class:maze={!inCamp} class:inspecting={d.prompt === 'inspect'}>
  {#if inCamp}
    <CampScreen bind:this={campScreen} {view} {busy} {call} />
  {:else}
  <section class="stage">
    {#if d.phase === 'maze'}
      <svg {viewBox} class="view" role="img" aria-label="The corridor ahead">
        {#each shapes as s, i (i)}
          <polygon class={s.kind} points={points(s)} />
        {/each}
      </svg>
      {#if d.viewport?.length}
        <div class="viewport">{#each d.viewport as t, i (i)}<p>{t}</p>{/each}</div>
      {/if}
      {#if d.dark && !shapes.length && !d.viewport?.length}
        <div class="viewport muted"><p>darkness</p></div>
      {/if}
      {#if minimapShown && view.minimap}
        <div class="minimap" aria-label="Minimap"><MapSvg view={view.minimap} label="The squares around the party" /></div>
      {/if}
    {:else}
      <div class="camp">CAMP</div>
    {/if}
  </section>

  <section class="messages" aria-live="polite">
    {#each lines as l, i (i)}<p>{l}</p>{/each}
    {#if d.more}<p class="muted">[RET] FOR MORE</p>{/if}
    {#if d.prompt === 'inspect'}
      <p class="muted small progress">{p.party}</p>
      {#if d.inspect?.found?.length}
        <ul class="found" aria-label="Characters found">
          {#each d.inspect.found as f (f.name)}
            <li>
              <span class="who">{f.n}) {f.name} · {f.class} · LEVEL {f.level} <Condition label={f.condition} /></span>
              {#if f.canPick}
                <button disabled={busy} onclick={() => act('pick-up', f.n)}>Pick up{#if f.n <= 9} <kbd>{f.n}</kbd>{/if}</button>
              {:else}
                <span class="muted small">{f.refusal}</span>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
      <p class:muted={!!d.inspect?.found?.length}>{p.here}{#if p.elsewhere}<span class="muted small">{' · ' + p.elsewhere}</span>{/if}</p>
      {#if d.inspect?.carried?.length}
        <div class="carried muted small">
          <span>In the party, unable to act:</span>
          {#each d.inspect.carried as ch (ch.name)}<span class="who">{ch.name} <Condition label={ch.condition} /></span>{/each}
        </div>
      {/if}
    {/if}
  </section>

  <section class="commands">
    {#if open.includes('button')}
      <div class="row">
        {#each d.buttons ?? [] as b, i (b)}
          <button disabled={busy} onclick={() => act('button', i + 1)}>{b}</button>
        {/each}
      </div>
    {:else if d.phase === 'maze' && d.prompt === ''}
      <div class="pad">
        {#each moves as m (m)}
          <button class:primary={m === 'forward'} disabled={busy} onclick={() => act(m)}>{actions[m].label} <kbd>{actions[m].hint}</kbd></button>
        {/each}
      </div>
      <div class="row">
        {#each open.filter((a) => !moves.includes(a)) as a (a)}
          <button disabled={busy} onclick={() => act(a)}>{actions[a]?.label ?? a} <kbd>{actions[a]?.hint}</kbd></button>
        {/each}
      </div>
    {:else}
      <div class="row">
        {#each open.filter((a) => a !== 'pick-up') as a (a)}
          <button class:primary={a === open[0]} disabled={busy} onclick={() => act(a)}>{actions[a]?.label ?? a} <kbd>{actions[a]?.hint}</kbd></button>
        {/each}
      </div>
    {/if}
  </section>
  {/if}
  </div>

  {#if mapOpen}<AutomapPopup onclose={closeMap} />{/if}

  {#if partyShown}
    <div class="partyarea">
      {#if inCamp}
        <PartyBar slots={view.partyBar ?? []} frontRow={view.frontRow} {busy}
          pick={(i) => pickLabel(view.camp, i)} onpick={(i) => campScreen?.pick(i)}
          current={view.camp.view.member} mark={(i) => placeMark(view.camp, i)} />
      {:else}
        <PartyBar slots={view.partyBar ?? []} frontRow={view.frontRow} />
      {/if}
    </div>
  {/if}
</div>

<style>
  /* Window-high: the header and the party bar stay in view; the middle
     (the maze, or Camp) fits between them or scrolls. */
  .dungeon { position: relative; display: grid; gap: 12px; height: 100%; min-height: 0; grid-template-rows: auto minmax(0, 1fr) auto; }
  .middle { min-height: 0; overflow-y: auto; display: grid; gap: 12px; align-content: start; }
  /* The messages and commands take what they need (Inspect's list scrolls
     within its own height); the view gives way, down to 150px. */
  .middle.maze { grid-template-rows: minmax(150px, 1fr) max-content max-content; align-content: stretch; }
  .middle.maze > section { margin-top: 0; }
  .middle.maze.inspecting { grid-template-rows: minmax(110px, 1fr) max-content max-content; }
  .bar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .bar h2 { margin: 0; }
  .spacer { flex: 1; }
  .tag { font-size: 11px; border: 1px solid var(--line); border-radius: 9px; padding: 1px 8px; color: var(--muted); }
  .tag.spell { color: #f2c14e; border-color: #6b5a2c; }
  .tag.real { color: var(--accent); border-color: var(--accent); }
  .stage { position: relative; container-type: inline-size; display: grid; place-items: center; min-height: 0; border-radius: 8px; padding: 10px;
    background: #0d0c0b; border: 1px solid var(--line); }
  .view { height: 100%; max-height: 360px; max-width: 100%; aspect-ratio: 1; }
  .view polygon { fill: none; stroke: #d8d2c4; stroke-width: 0.6; vector-effect: non-scaling-stroke; }
  .view polygon.wall { fill: #1b1915; }
  .view polygon.door { stroke: #f2c14e; stroke-width: 1.4; }
  .viewport { position: absolute; inset: 0; display: grid; place-content: center; text-align: center;
    font: 700 16px ui-monospace, Menlo, monospace; color: #fff; text-shadow: 0 0 6px #000; pointer-events: none; }
  .viewport p { margin: 2px 0; }
  /* A square no taller than the view (less its margins) nor wider than a
     quarter of it, so it never covers the messages below. */
  .minimap { position: absolute; top: 8px; right: 8px; height: min(168px, calc(100% - 16px), 26cqw); aspect-ratio: 1;
    background: rgba(0, 0, 0, 0.55); border: 1px solid rgba(79, 209, 106, 0.3); border-radius: 6px; padding: 4px;
    pointer-events: none; }
  .camp { font: 700 28px ui-monospace, Menlo, monospace; letter-spacing: 6px; color: var(--muted); }
  .messages { min-height: 52px; font-family: ui-monospace, Menlo, monospace; text-align: center; }
  .messages p { margin: 2px 0; color: #f2c14e; }
  .messages p.muted { color: var(--muted); }
  .commands { display: grid; gap: 8px; justify-items: center; }
  .pad, .row { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
  kbd { font: 600 11px ui-monospace, Menlo, monospace; border: 1px solid var(--line); border-radius: 3px; padding: 0 4px; }
  .small { font-size: 11px; }
  /* Inspect's finds: one line each, scrolling if the area holds many. */
  .found { list-style: none; margin: 4px auto; padding: 0; max-height: 168px; overflow-y: auto; max-width: 640px; }
  .found li { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 2px 6px; color: #f2c14e; }
  .found li button { padding: 1px 8px; }
  .who { display: inline-flex; align-items: center; gap: 6px; }
  .progress { margin-bottom: 0; }
  .carried { display: flex; flex-wrap: wrap; justify-content: center; align-items: center; gap: 4px 8px; margin: 4px auto 0; max-width: 640px; }
  .partyarea { border-top: 1px solid var(--line); padding-top: 8px; }
</style>
