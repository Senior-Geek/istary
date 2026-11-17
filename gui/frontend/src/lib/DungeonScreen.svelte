<script>
  // The dungeon screen (M6a; real expeditions M6b-1). It renders view.dungeon (Go's app.DungeonView)
  // and calls Session.DungeonAction with one of the actions Go lists; it
  // keeps no maze state. The picture is MazeView's (M-ART-1A.1): the
  // corridor Go describes, and nothing else, as the Classic wireframe or in
  // Enhanced stone (M-ART-1A.2), at one of three zooms. Which is a window
  // setting (M-ART-1A.3: view.settings, Session.SetMazeLook, also in
  // Settings), chosen in the view's top left corner: presentation alone,
  // never a game action. A button clicked lets go of the focus, so the keys
  // reach the maze at once; from the keyboard the focus stays on it. The
  // picture is the largest square the view holds (see .picture).
  //
  // Inspect (R-2) lists each character found in the party's area, with a
  // Pick up button (keys 1-9) while the party has room, or the reason not.
  // R-3 adds each one's condition (Condition), and the rescue's progress as
  // Go reads it (inspect.js): the party's size and room, who else is here,
  // how many are known to lie elsewhere, and the members who cannot act.
  //
  // The party's coordinates and facing (UX-1, view.position: Go's L:R:C,
  // the maps' own) stand in the header, in the maze and, larger, at Camp.
  // When characters lie on the party's square (its map marker's count), a
  // line under the view says so and points to Inspect / Rescue.
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
  import { scene, actions, keyAction, expeditionTag, suggested } from './dungeon.js'
  import MazeView from './MazeView.svelte'
  import { mazeLooks, mazeLookOf } from './mazeStyle.js'
  import CampScreen from './CampScreen.svelte'
  import { pickLabel, placeMark } from './camp.js'
  import Condition from './Condition.svelte'
  import { progress } from './inspect.js'
  import MapSvg from './MapSvg.svelte'
  import AutomapPopup from './AutomapPopup.svelte'
  import { mapKey } from './automap.js'
  import Emblem from './Emblem.svelte'
  import { where, whereTitle, strandedHint } from './position.js'

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

  const look = $derived(mazeLookOf(view.settings))
  const lookNames = { style: 'Maze style', zoom: 'Maze zoom' }
  function choose(key, value, e) {
    const next = { ...look, [key]: value }
    if (e.detail) e.currentTarget.blur() // clicked: the keys go back to the maze
    call(S.SetMazeLook, next.style, next.zoom)
  }

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
    <h2>{#if d.phase !== 'camp'}<Emblem location="maze" size="34px" />{/if}{d.phase === 'camp' ? 'Camp' : 'The maze'}</h2>
    {#if view.position}
      <span class="where" class:prominent={d.phase === 'camp'} title={whereTitle(view.position)}
        aria-label="Position: {whereTitle(view.position)}">{where(view.position)}</span>
    {:else}
      <span class="where unknown" title={whereTitle(null)}>position unknown</span>
    {/if}
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
      <div class="picture"><MazeView dungeon={d} style={look.style} zoom={look.zoom} /></div>
      {#if d.viewport?.length}
        <div class="viewport">{#each d.viewport as t, i (i)}<p>{t}</p>{/each}</div>
      {/if}
      {#if d.dark && !shapes.length && !d.viewport?.length}
        <div class="viewport muted"><p>darkness</p></div>
      {/if}
      <div class="look" role="group" aria-label="The maze picture">
        {#each Object.entries(mazeLooks) as [key, options] (key)}
          <span class="set" role="group" aria-label={lookNames[key]} title="{lookNames[key]} (also in Settings)">{#each options as o (o.id)}<button class="small"
            aria-pressed={look[key] === o.id} disabled={busy} onclick={(e) => choose(key, o.id, e)}>{o.label}</button>{/each}</span>
        {/each}
      </div>
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
    {#if d.phase === 'maze' && d.prompt === '' && strandedHint(view.position)}
      <p class="hint">{strandedHint(view.position)} — Inspect / Rescue <kbd>I</kbd></p>
    {/if}
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
          <button class:primary={a === suggested(d)} disabled={busy} onclick={() => act(a)}>{actions[a]?.label ?? a} <kbd>{actions[a]?.hint}</kbd></button>
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
  .bar h2 { margin: 0; display: flex; align-items: center; gap: 10px; }
  /* The maze's emblem (ART-3B-0), in the gold of the town's titles. */
  .bar h2 :global(.emblem) { color: var(--gold); margin: -4px 0; }
  .where { font: 600 13px ui-monospace, Menlo, monospace; color: var(--accent); border: 1px solid var(--line); border-radius: 4px; padding: 1px 6px; white-space: nowrap; }
  .where.prominent { font-size: 17px; padding: 2px 10px; border-color: var(--accent); }
  .where.unknown { color: var(--muted); font-weight: 400; }
  .spacer { flex: 1; }
  .tag { font-size: 11px; border: 1px solid var(--line); border-radius: 9px; padding: 1px 8px; color: var(--muted); }
  .tag.spell { color: #f2c14e; border-color: #6b5a2c; }
  .tag.real { color: var(--accent); border-color: var(--accent); }
  /* The view (M-ART-1A.3): its own size is the grid's, so the picture can
     be sized from it (container units), in width and height. */
  .stage { --corner: 184px; position: relative; container-type: size; display: grid; place-items: center; min-height: 0; border-radius: 8px; padding: 6px;
    background: #0d0c0b; border: 1px solid var(--line); }
  .viewport { position: absolute; inset: 0; display: grid; place-content: center; text-align: center;
    font: 700 16px ui-monospace, Menlo, monospace; color: #fff; text-shadow: 0 0 6px #000; pointer-events: none; }
  .viewport p { margin: 2px 0; }
  /* A square no taller than the view (less its margins) nor wider than a
     quarter of it, so it never covers the messages below. */
  .minimap { position: absolute; top: 8px; right: 8px; height: min(168px, calc(100% - 16px), 26cqw); aspect-ratio: 1;
    background: rgba(0, 0, 0, 0.55); border: 1px solid rgba(79, 209, 106, 0.3); border-radius: 6px; padding: 4px;
    pointer-events: none; }
  /* The picture's style and zoom (M-ART-1A.3): the view's upper left
     corner, within the column kept for it (--corner), as the minimap is
     in the upper right. */
  .look { position: absolute; left: 8px; top: 8px; display: flex; flex-direction: column; align-items: flex-start; gap: 4px; max-width: calc(var(--corner) - 8px); }
  .look .set { display: inline-flex; }
  .look button { padding: 2px 8px; border-radius: 0; margin: 0; }
  .look button + button { margin-left: -1px; }
  .look button:first-child { border-radius: 4px 0 0 4px; }
  .look button:last-child { border-radius: 0 4px 4px 0; }
  .look button[aria-pressed='true'] { color: var(--accent); border-color: var(--accent); position: relative; }
  .camp { font: 700 28px ui-monospace, Menlo, monospace; letter-spacing: 6px; color: var(--muted); }
  .messages { min-height: 52px; font-family: ui-monospace, Menlo, monospace; text-align: center; }
  .messages p { margin: 2px 0; color: #f2c14e; }
  .messages p.muted { color: var(--muted); }
  .messages p.hint { color: #ff8a7a; font-size: 13px; }
  /* The picture: the largest square the view holds, less a column on each
     side for what sits in its corners (the style and zoom on the left, the
     minimap on the right: neither ever covers it), and at most 720px. */
  .picture { width: min(100cqh, 100cqw - 2 * var(--corner), 720px); aspect-ratio: 1; }
  /* The movement pad and the other commands share a line where they fit
     (wrapping where they do not), so the view keeps the height. */
  .commands { display: flex; flex-wrap: wrap; justify-content: center; align-items: center; gap: 8px 24px; }
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
