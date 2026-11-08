<script>
  // The dungeon screen (M6a; real expeditions M6b-1). It renders view.dungeon (Go's app.DungeonView)
  // and calls Session.DungeonAction with one of the actions Go lists; it
  // keeps no maze state. The picture is a simple wireframe of the corridor
  // Go describes, not the Classic renderer.
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

  let { view, busy = false, call } = $props()

  const d = $derived(view.dungeon)
  const partyShown = $derived(view.settings?.partyVisible ?? true)
  const open = $derived(d.actions ?? [])
  const shapes = $derived(scene(d.corridor ?? []))
  const lines = $derived(d.lines ?? [])
  const moves = ['left', 'forward', 'right']

  const act = (action, button = 0) => call(S.DungeonAction, action, button)
  let campScreen = $state() // the Camp screen, whose pick the party bar uses
  const inCamp = $derived(d.phase === 'camp' && !!view.camp)

  function onkey(e) {
    if (busy || e.target.closest?.('input, select, textarea') || e.metaKey || e.ctrlKey) return
    const a = keyAction(e.key, d)
    if (a) {
      e.preventDefault()
      act(a.action, a.button)
    }
  }
</script>

<svelte:window onkeydown={onkey} />

<div class="dungeon">
  <header class="bar">
    <h2>{d.phase === 'camp' ? 'Camp' : 'The maze'}</h2>
    <span class="tag" class:real={view.real}>{expeditionTag(view)}</span>
    {#each d.spells ?? [] as s (s)}<span class="tag spell">{s}</span>{/each}
    <div class="spacer"></div>
    <button class="small" aria-pressed={partyShown} disabled={busy}
      onclick={() => call(S.SetPartyVisible, !partyShown)}>{partyShown ? 'Hide party' : 'Show party'}</button>
    {#if !view.real}
      <button class="small" disabled={busy} onclick={() => call(S.AbandonPractice)}>Abandon</button>
    {/if}
  </header>

  <div class="middle" class:maze={!inCamp}>
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
    {:else}
      <div class="camp">CAMP</div>
    {/if}
  </section>

  <section class="messages" aria-live="polite">
    {#each lines as l, i (i)}<p>{l}</p>{/each}
    {#if d.more}<p class="muted">[RET] FOR MORE</p>{/if}
    {#if d.prompt === 'inspect'}
      <p>{d.found?.length ? 'FOUND: ' + d.found.join(', ') : 'LOOKING... NO ONE IS HERE.'}</p>
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
        {#each open as a (a)}
          <button class:primary={a === open[0]} disabled={busy} onclick={() => act(a)}>{actions[a]?.label ?? a} <kbd>{actions[a]?.hint}</kbd></button>
        {/each}
      </div>
    {/if}
  </section>
  {/if}
  </div>

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
  .dungeon { display: grid; gap: 12px; height: 100%; min-height: 0; grid-template-rows: auto minmax(0, 1fr) auto; }
  .middle { min-height: 0; overflow-y: auto; display: grid; gap: 12px; align-content: start; }
  .middle.maze { grid-template-rows: minmax(150px, 1fr) auto auto; align-content: stretch; }
  .bar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .bar h2 { margin: 0; }
  .spacer { flex: 1; }
  .tag { font-size: 11px; border: 1px solid var(--line); border-radius: 9px; padding: 1px 8px; color: var(--muted); }
  .tag.spell { color: #f2c14e; border-color: #6b5a2c; }
  .tag.real { color: var(--accent); border-color: var(--accent); }
  .stage { position: relative; display: grid; place-items: center; min-height: 0; border-radius: 8px; padding: 10px;
    background: #0d0c0b; border: 1px solid var(--line); }
  .view { height: 100%; max-height: 360px; max-width: 100%; aspect-ratio: 1; }
  .view polygon { fill: none; stroke: #d8d2c4; stroke-width: 0.6; vector-effect: non-scaling-stroke; }
  .view polygon.wall { fill: #1b1915; }
  .view polygon.door { stroke: #f2c14e; stroke-width: 1.4; }
  .viewport { position: absolute; inset: 0; display: grid; place-content: center; text-align: center;
    font: 700 16px ui-monospace, Menlo, monospace; color: #fff; text-shadow: 0 0 6px #000; pointer-events: none; }
  .viewport p { margin: 2px 0; }
  .camp { font: 700 28px ui-monospace, Menlo, monospace; letter-spacing: 6px; color: var(--muted); }
  .messages { min-height: 52px; font-family: ui-monospace, Menlo, monospace; text-align: center; }
  .messages p { margin: 2px 0; color: #f2c14e; }
  .messages p.muted { color: var(--muted); }
  .commands { display: grid; gap: 8px; justify-items: center; }
  .pad, .row { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
  kbd { font: 600 11px ui-monospace, Menlo, monospace; border: 1px solid var(--line); border-radius: 3px; padding: 0 4px; }
  .small { font-size: 11px; }
  .partyarea { border-top: 1px solid var(--line); padding-top: 8px; }
</style>
