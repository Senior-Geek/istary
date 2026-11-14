<script>
  // The location frame (Castle-0, UX-3B): one layout for every place in
  // town. It shows where the player is, the way Back, the place's actions
  // and the other destinations (closed ones with Go's reason), its artwork
  // or a character's sheet, its information, and the party bar, each in
  // its own area, so nothing is drawn over a control or the party. Moves
  // go through Session.Navigate and the party toggle through
  // Session.SetPartyVisible; what is open is Go's.
  //
  // Areas (UX-3B, NAV-1): head, the top bar (title, Back to the Castle,
  // then the toggles: party, story, character sheet, panels); A, the
  // display, upper left (the artwork, or the embedded character sheet); B,
  // the actions, upper right (the place's decisions, then the
  // destinations), scrolling inside itself if it is long; C, the
  // information below both, scrolling on its own and taking whatever
  // height the party bar leaves (all of it when the party is hidden); D,
  // the party bar. A and B stay in place while C scrolls. Under 700px the
  // areas stack and the body scrolls as one.
  //
  // An illustrated place (ART-2A; lib/scenes.js): its picture fills the
  // window behind the frame, cropped to cover it, and the areas are the
  // same, laid over it: A is a clear opening onto the picture; the head, the
  // actions, the information and the party bar are near-opaque panels, the
  // information only as tall as it needs, the display and actions taking
  // the rest. "Hide panels"
  // (in the head) leaves the picture alone with one way back, "Show panels"
  // (or Escape); App keeps which (PANELS). Nothing of it is saved or sent
  // to Go.
  //
  // ART-2B: a place's narrative is in a story box over the picture
  // (StoryBox), closed with its ×, opened again with Story in the head, and
  // not reopened by going elsewhere. What the player needs to play stays
  // in C.
  //
  // ART-2C: the illustrated places' appearance is chosen in Settings and
  // kept by Go (App's PANELS look): the panels' (slate, dark or light) and
  // the story box's (parchment, dark or slate), only colours (style.css
  // [data-appearance]), and the picture, shown whole, subdued or hidden.
  // With the picture hidden there is nothing to look at: Hide panels is
  // not offered, and the rest is as with it.
  //
  // The character sheet (lib/sheet.js): App keeps whose it is and whether
  // it is shown (the SHEET context); the frame shows it in A, in place of
  // the artwork, and offers the one toggle for it in the head (NAV-1). What the sheet
  // offers is the place's: renaming and deleting at the Training Grounds,
  // joining and leaving at the Tavern, equipping anywhere in town (Go
  // refuses it when it is not open, with the reason on the sheet).
  //
  // Props:
  //   view      the Go View
  //   busy      a request is running
  //   call      (fn, ...args) => Promise   runs a Session method
  //   onopen    (rosterIndex) => void      open a member's sheet (optional)
  //   scene     snippet drawn instead of the place's artwork (optional)
  //   actions   snippet: the place's actions, first in B (optional); it is
  //             given the destinations (navigation.js) and the snippet
  //             that draws one, for a place that lists them among its
  //             actions
  //   waysInActions  the place's actions list its destinations, and there
  //             is no Go to list (NAV-1, the Edge of Town) (optional)
  //   aside     snippet under the actions, e.g. a map (optional)
  //   children  the place's information, in C (optional)
  //   story     snippet: the place's narrative, in the story box at an
  //             illustrated place (ART-2B) (optional)
  //   storyTitle  the story box's title (optional; the place's name)
  //   fill      the place lays out C itself (UX-2, Boltac's): C does not
  //             scroll, the place's own lists inside it do, and the
  //             artwork is shorter (optional)
  //   partyAlways  the party bar is always shown here, whatever the Hide
  //             party setting, and the toggle is not offered (UX-2,
  //             Boltac's: the bar is how the customer is chosen) (optional)
  //   compact   the display and actions are shorter, for a place whose
  //             information needs the height (the Training Grounds while
  //             a character is created) (optional)
  //   noSheet   no character sheet here (UX-2, Boltac's: the Selected
  //             Character panel is the customer) (optional)
  //   pick, onpick, current
  //             the party bar's picking (PartyBar): what a click on a member
  //             does here, and the member worked with (optional; Boltac's)
  import { getContext } from 'svelte'
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationScene from './LocationScene.svelte'
  import PartyBar from './PartyBar.svelte'
  import CharacterSheet from './CharacterSheet.svelte'
  import { title, back, destinations } from './navigation.js'
  import { SHEET, subject, toggleLabel, available } from './sheet.js'
  import StoryBox from './StoryBox.svelte'
  import { PANELS, illustrated, lookOf } from './scenes.js'

  let { view, busy = false, call, onopen, scene, actions, aside, children, story, storyTitle, pick, onpick, current = -1, fill = false, partyAlways = false, compact = false, noSheet = false, waysInActions = false } = $props()

  const here = $derived(title(view.location))
  const way = $derived(back(view))
  const ways = $derived(destinations(view))
  const partyShown = $derived(partyAlways || (view.settings?.partyVisible ?? true))

  const sheet = getContext(SHEET)
  const sheetHere = $derived(!!sheet && !noSheet && available(view))
  const who = $derived(sheetHere ? subject(view, sheet.index) : null)
  const sheetShown = $derived(sheetHere && sheet.shown && !!who)

  // The place's own scene, if it draws one, is never a backdrop.
  const art = $derived(!scene && illustrated(view.location))
  const panels = getContext(PANELS)
  const look = $derived(art ? (panels?.look ?? lookOf(null)) : null)
  const pictured = $derived(art && look.artwork !== 'hidden')
  const bare = $derived(pictured && !!panels?.hidden)
  let hideButton = $state(), showButton = $state()
  let moved = false // focus goes with the toggle, once it is pressed
  $effect(() => {
    const to = bare ? showButton : hideButton
    if (moved && to) {
      to.focus()
      moved = false
    }
  })
  const hidePanels = () => { moved = true; panels?.hide() }
  const showPanels = () => { moved = true; panels?.show() }

  const storyHere = $derived(art && !!story && !!panels)
  const storyOpen = $derived(storyHere && panels.story.open)
  let display = $state(), storyButton = $state()
  function closeStory() {
    panels.story.open = false
    storyButton?.focus()
  }

  const go = (dest) => call(S.Navigate, dest)
  const toggleSheet = () => (sheetShown ? sheet.hide() : sheet.show(who.index))

  async function rename(name) {
    await call(S.RenameCharacter, who.index, name)
  }
  async function remove(password) {
    const name = who.name
    await call(S.DeleteCharacter, name, password)
    if (!view.error) sheet.forget()
  }
</script>

<!-- One destination: its button, and Go's reason when it is closed. The
     Go to list draws each with it, and a place that puts its destinations
     among its actions (waysInActions) is given it with them. -->
{#snippet destination(w)}
  <button class="way" disabled={busy || !w.open} onclick={() => go(w.dest)}>{w.label}</button>
  {#if !w.open}<span class="why">{w.reason}</span>{/if}
{/snippet}

{#if art}<LocationScene location={view.location} title={here} mode="backdrop" artwork={look.artwork} />{/if}

{#if bare}
  <div class="frame bare" data-location={view.location} data-appearance={look.panels}>
    <div class="restore" data-area="restore">
      <span class="restore-title">{here}</span>
      <button class="primary" bind:this={showButton} onclick={showPanels}
        title="Show the panels again (Esc)">Show panels <kbd>Esc</kbd></button>
    </div>
  </div>
{:else}
<div class="frame" class:fill class:compact class:illustrated={art} data-location={view.location} data-appearance={look?.panels}>
  <header class="head">
    <h2 class="title">{here}</h2>
    {#if way}
      <button class="back" disabled={busy || !way.open} onclick={() => call(S.ReturnToCastle)}
        title="Back to the Castle (Esc)">← {way.label} <kbd>Esc</kbd></button>
    {/if}
    {#if !partyAlways}
      <button class="toggle" aria-pressed={partyShown} disabled={busy}
        onclick={() => call(S.SetPartyVisible, !partyShown)}>{partyShown ? 'Hide party' : 'Show party'}</button>
    {/if}
    {#if storyHere}
      <button class="toggle" bind:this={storyButton} aria-pressed={storyOpen}
        onclick={() => (storyOpen ? closeStory() : (panels.story.open = true))}
        title={storyOpen ? 'Close the story' : 'Open the story'}>Story</button>
    {/if}
    {#if sheetHere}
      <button class="toggle" aria-pressed={sheetShown} disabled={busy || !who} onclick={toggleSheet}
        title={who ? `${sheetShown ? 'Hide' : 'Show'} ${who.name}'s character sheet (click a party member to see theirs)` : 'Nobody to show'}>{toggleLabel(sheetShown)}</button>
    {/if}
    {#if pictured}
      <button class="toggle" bind:this={hideButton} disabled={busy} onclick={hidePanels}
        title="Hide the panels to see the picture (Esc brings them back)">Hide panels</button>
    {/if}
    {#if way?.note}<p class="why head-why">{way.note}</p>{/if}
  </header>

  <div class="body">
  <div class="scene-area" class:sheet={sheetShown} data-area="display" bind:this={display}>
    {#if sheetShown}
      {#key who.index}
        <CharacterSheet character={who} statNames={view.statNames} nameMax={view.nameMax}
          {busy} error={view.error}
          onrename={view.location === 'training' ? rename : undefined}
          ondelete={view.location === 'training' ? remove : undefined}
          onjoin={view.location === 'tavern' ? () => call(S.JoinParty, who.index) : undefined}
          onleave={view.location === 'tavern' ? () => call(S.LeaveParty, who.partySlot - 1) : undefined}
          onequip={(n, equip) => call(S.EquipAction, { index: who.index, n, equip })} />
      {/key}
    {:else if scene}{@render scene()}{:else}<LocationScene location={view.location} title={here} mode={art ? 'view' : 'panel'} artwork={look?.artwork} />{/if}
  </div>

  <div class="nav-area" data-area="actions">
    {#if actions}
      <section class="actions" aria-label="Actions here">
        <h3>Actions</h3>
        {@render actions(ways, destination)}
      </section>
    {/if}
    {#if aside}<div class="aside-area">{@render aside()}</div>{/if}
    {#if ways.length && !waysInActions}
      <nav aria-label="Destinations">
        <h3>Go to</h3>
        <ul>
          {#each ways as w (w.dest)}
            <li>{@render destination(w)}</li>
          {/each}
        </ul>
      </nav>
    {/if}
  </div>

  <div class="content-area" data-area="information">{#if children}{@render children()}{/if}</div>
  </div>

  {#if partyShown}
    <div class="party-area" data-area="party">
      <PartyBar slots={view.partyBar ?? []} frontRow={view.frontRow} {onopen} {pick} {onpick} {current} {busy} />
    </div>
  {/if}

  {#if storyOpen && display}
    <StoryBox title={storyTitle ?? here} anchor={display} look={look.story} rect={panels.story.rect}
      onmove={(r) => (panels.story.rect = r)} onclose={closeStory}>{@render story()}</StoryBox>
  {/if}
</div>
{/if}

<style>
  /* The frame fills the height it is given: head and party bar fixed. In
     the body, wide windows put the display (A) beside the actions (B), a
     fixed height, and the information (C) below takes the rest and
     scrolls on its own. Under 700px they stack and the body scrolls. */
  .frame {
    --top: clamp(150px, calc((100vw - 380px) * 0.5625), 42vh);
    display: grid; height: 100%; min-height: 0;
    grid-template-rows: auto minmax(0, 1fr) auto;
    grid-template-areas: "head" "body" "party";
    gap: 10px;
  }
  .frame.fill, .frame.compact { --top: clamp(130px, calc((100vw - 380px) * 0.5625), 30vh); }
  .head { grid-area: head; display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; border-bottom: 1px solid var(--line); padding-bottom: 8px; }
  .title { margin: 0 auto 0 0; font-size: 20px; color: var(--text); letter-spacing: 0.02em; }
  .head button { margin: 0; }
  .back kbd { font: 600 10px ui-monospace, Menlo, monospace; opacity: 0.7; margin-left: 4px; }
  .head-why { flex-basis: 100%; margin: 0; text-align: right; }
  .body {
    grid-area: body; min-height: 0; overflow: hidden;
    display: grid; gap: 12px 16px;
    grid-template-columns: minmax(0, 1fr) minmax(220px, 300px);
    grid-template-rows: var(--top) minmax(0, 1fr);
    grid-template-areas: "scene nav" "content content";
  }
  /* A and B share a definite height: the artwork or sheet fills it, and
     long actions scroll inside B. */
  .scene-area { grid-area: scene; height: 100%; min-width: 0; min-height: 0; }
  .nav-area { grid-area: nav; height: 100%; min-width: 0; min-height: 0; overflow-y: auto; overscroll-behavior: contain;
    display: grid; gap: 12px; align-content: start; padding-right: 2px; }
  .aside-area { min-width: 0; }
  .content-area { grid-area: content; min-width: 0; min-height: 0; height: 100%; overflow-y: auto; padding-right: 2px; }
  .content-area > :global(:first-child) { margin-top: 0; }
  .frame.fill .content-area { overflow: hidden; }
  .party-area { grid-area: party; border-top: 1px solid var(--line); padding-top: 8px; }
  h3 { font-size: 12px; text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted); margin: 0 0 6px; font-weight: 500; }
  nav ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 4px; }
  nav li { display: grid; }
  .way { width: 100%; text-align: left; margin: 0; }
  .why { color: var(--muted); font-size: 11px; padding: 2px 2px 4px; }
  .actions { margin: 0; padding: 10px 12px; }

  /* An illustrated place: the same areas over the picture. Each working
     panel is near-opaque, so its text reads over any part of the picture;
     the gaps between them, and A, show the picture. */
  /* The panels' colours are the appearance's (style.css [data-appearance]). */
  .frame.illustrated .head,
  .frame.illustrated .actions,
  .frame.illustrated nav,
  .frame.illustrated .content-area > :global(section),
  .frame.illustrated .party-area {
    background: var(--glass); border: 1px solid var(--glass-line); border-radius: 6px;
    -webkit-backdrop-filter: blur(6px); backdrop-filter: blur(6px);
    box-shadow: 0 2px 12px var(--glass-shadow);
  }
  .frame.illustrated .head { padding: 6px 8px 6px 12px; }
  .frame.illustrated nav { padding: 10px 12px; }
  .frame.illustrated .party-area { padding: 6px; }
  /* The information is as tall as it needs, up to 45% of the body, and
     scrolls on its own past that; the display and the actions take the
     rest, so the actions are long enough for all of them more often, and
     hiding the party gives them its room. */
  .frame.illustrated .body { grid-template-rows: minmax(var(--top), 1fr) fit-content(45%); }
  .frame.illustrated .content-area > :global(section) { margin-top: 0; }
  /* The picture alone: one way back, in the corner. */
  .frame.bare { display: flex; justify-content: flex-end; align-items: flex-start; }
  .restore {
    display: flex; align-items: center; gap: 10px; padding: 6px 6px 6px 12px;
    background: var(--glass); color: var(--text); border: 1px solid var(--glass-line); border-radius: 6px;
    -webkit-backdrop-filter: blur(6px); backdrop-filter: blur(6px); box-shadow: 0 2px 12px var(--glass-shadow);
  }
  .restore-title { font-size: 15px; letter-spacing: 0.02em; }
  .restore button { margin: 0; }
  .restore kbd { font: 600 10px ui-monospace, Menlo, monospace; opacity: 0.7; margin-left: 4px; }
  @media (max-height: 640px) { .frame:not(.fill):not(.compact) { --top: clamp(130px, calc((100vw - 380px) * 0.5625), 34vh); } }
  @media (max-width: 699px) {
    .body {
      overflow-y: auto;
      grid-template-columns: minmax(0, 1fr);
      grid-template-rows: none;
      grid-template-areas: "scene" "nav" "content";
    }
    .scene-area { height: auto; aspect-ratio: 21 / 9; max-height: 30vh; }
    .scene-area.sheet { aspect-ratio: auto; height: 60vh; max-height: none; }
    .nav-area, .content-area { height: auto; overflow: visible; }
    .frame.fill .content-area { height: auto; }
    .frame.illustrated .body { grid-template-rows: none; }
  }
</style>
