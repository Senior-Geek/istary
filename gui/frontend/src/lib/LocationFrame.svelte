<script>
  // The location frame (Castle-0): one layout for every place. It shows
  // where the player is, the way Back, the other destinations (closed ones
  // with Go's reason), the place's own actions and content, its artwork,
  // and the party bar, each in its own area, so the artwork never covers a
  // control or the party. Moves go through Session.Navigate and the party
  // toggle through Session.SetPartyVisible; what is open is Go's.
  //
  // Areas: head (title, Back, party toggle), scene (the artwork), nav (the
  // destinations, then the place's actions), content (anything wider, below
  // both), party (the party bar). A future graphical maze can draw into the
  // scene area (the scene snippet) and an automap into an aside beside the
  // nav; neither changes how places are reached.
  //
  // Props:
  //   view      the Go View
  //   busy      a request is running
  //   call      (fn, ...args) => Promise   runs a Session method
  //   onopen    (rosterIndex) => void      open a member's sheet (optional)
  //   scene     snippet drawn instead of the place's artwork (optional)
  //   actions   snippet: the place's actions, under the destinations (optional)
  //   aside     snippet beside the nav, e.g. a map (optional)
  //   children  the place's wider content (optional)
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationScene from './LocationScene.svelte'
  import PartyBar from './PartyBar.svelte'
  import { title, back, destinations } from './navigation.js'

  let { view, busy = false, call, onopen, scene, actions, aside, children } = $props()

  const here = $derived(title(view.location))
  const way = $derived(back(view))
  const ways = $derived(destinations(view))
  const partyShown = $derived(view.settings?.partyVisible ?? true)

  const go = (dest) => call(S.Navigate, dest)
</script>

<div class="frame" class:with-aside={!!aside} data-location={view.location}>
  <header class="head">
    <h2 class="title">{here}</h2>
    {#if way}
      <button class="back" disabled={busy || !way.open} onclick={() => go(way.dest)}>← {way.label}</button>
    {/if}
    <button class="toggle" aria-pressed={partyShown} disabled={busy}
      onclick={() => call(S.SetPartyVisible, !partyShown)}>{partyShown ? 'Hide party' : 'Show party'}</button>
    {#if way && !way.open}<p class="why head-why">{way.reason}</p>{/if}
  </header>

  <div class="scene-area">
    {#if scene}{@render scene()}{:else}<LocationScene location={view.location} title={here} />{/if}
  </div>

  <div class="nav-area">
    {#if ways.length}
      <nav aria-label="Destinations">
        <h3>Go to</h3>
        <ul>
          {#each ways as w (w.dest)}
            <li>
              <button class:primary={w.open} disabled={busy || !w.open} onclick={() => go(w.dest)}>{w.label}</button>
              {#if !w.open}<span class="why">{w.reason}</span>{/if}
            </li>
          {/each}
        </ul>
      </nav>
    {/if}
    {#if actions}
      <section class="actions" aria-label="Actions here">{@render actions()}</section>
    {/if}
  </div>

  {#if aside}<div class="aside-area">{@render aside()}</div>{/if}

  {#if children}<div class="content-area">{@render children()}</div>{/if}

  {#if partyShown}
    <div class="party-area">
      <PartyBar slots={view.partyBar ?? []} frontRow={view.frontRow} {onopen} />
    </div>
  {/if}
</div>

<style>
  /* Wide and normal windows: the artwork beside the controls, the content
     below, the party bar kept in view at the bottom. Narrow: stacked, the
     artwork as a banner. */
  .frame {
    display: grid;
    gap: 12px 16px;
    grid-template-columns: minmax(0, 1fr) minmax(220px, 300px);
    grid-template-areas: "head head" "scene nav" "content content" "party party";
    align-items: start;
  }
  .frame.with-aside { grid-template-areas: "head head" "scene nav" "scene aside" "content content" "party party"; }
  .head { grid-area: head; display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; border-bottom: 1px solid var(--line); padding-bottom: 8px; }
  .title { margin: 0 auto 0 0; font-size: 20px; color: var(--text); letter-spacing: 0.02em; }
  .head button { margin: 0; }
  .head-why { flex-basis: 100%; margin: 0; text-align: right; }
  .scene-area { grid-area: scene; aspect-ratio: 16 / 9; max-height: 46vh; min-height: 120px; min-width: 0; }
  .nav-area { grid-area: nav; display: grid; gap: 12px; min-width: 0; }
  .aside-area { grid-area: aside; min-width: 0; }
  .content-area { grid-area: content; min-width: 0; }
  .content-area > :global(:first-child) { margin-top: 0; }
  .party-area {
    grid-area: party; position: sticky; bottom: 0; z-index: 1;
    background: var(--bg); padding: 8px 0 4px; border-top: 1px solid var(--line);
  }
  nav h3 { font-size: 12px; text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted); margin: 0 0 6px; font-weight: 500; }
  nav ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 4px; }
  nav li { display: grid; }
  nav button { width: 100%; text-align: left; margin: 0; }
  .why { color: var(--muted); font-size: 11px; padding: 2px 2px 4px; }
  .actions { margin: 0; padding: 10px 12px; }
  @media (max-height: 640px) { .scene-area { max-height: 36vh; } }
  @media (max-width: 699px) {
    .frame, .frame.with-aside {
      grid-template-columns: minmax(0, 1fr);
      grid-template-areas: "head" "scene" "nav" "aside" "content" "party";
    }
    .scene-area { aspect-ratio: 21 / 9; max-height: 30vh; }
  }
</style>
