<script>
  // The Edge of Town (Castle-0 C0-4, NAV-1): outside the castle, where the
  // party sets out for the maze and the game is managed. Its actions, in
  // order: Enter the Maze, the Training Grounds, Select Game, Settings and
  // Leave the Game. The Training Grounds and Settings are its destinations
  // (Go's exits, drawn by the frame's own snippet), so there is no Go to
  // list; the Castle is the top bar's. Whether each departure is open, and
  // why not, is Go's (view.departures, view.exits).
  //
  // Practice (Istary's own) is not offered here: its Session methods stay,
  // for the automated tests.
  //
  // Props:
  //   view    the Go View (view.location is "edge")
  //   busy    a request is running
  //   call    (fn, ...args) => Promise   runs a Session method
  //   onopen  (rosterIndex) => void      open a party member's sheet
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'

  let { view, busy = false, call, onopen } = $props()

  const real = $derived(view.departures?.real ?? '')
  const caution = $derived(view.readiness?.line ?? '')
  let checking = $state(false) // the equipment check is shown; nothing is sent yet

  function enter() {
    if (caution && !checking) checking = true
    else {
      checking = false
      call(S.RealExpedition)
    }
  }
  function backToTown() {
    checking = false
    call(S.Navigate, 'castle')
  }
</script>

<LocationFrame {view} {busy} {call} {onopen} waysInActions>
  {#snippet actions(ways, destination)}
    <div class="group">
      <button class="primary wide" disabled={busy || !!real || checking} onclick={enter}>Enter the Maze</button>
      {#if real}<span class="why">{real}</span>{/if}
      {#if checking && caution && !real}
        <div class="check-equipment" role="alertdialog" aria-label="Equipment check">
          <p>{caution}</p>
          <p class="why">Items carried but not equipped do not count. A member's sheet (the party list) equips what they carry; Boltac's sells the rest.</p>
          <div class="choices">
            <button class="primary" disabled={busy} onclick={enter}>Continue into maze</button>
            <button disabled={busy} onclick={backToTown}>Return to town</button>
          </div>
        </div>
      {/if}
    </div>
    {#each ways.filter((w) => w.dest === 'training') as w (w.dest)}
      <div class="group">{@render destination(w)}</div>
    {/each}
    <div class="group">
      {#if view.game}
        <button class="wide" disabled={busy} onclick={() => call(S.ChangeGame)}>Select Game</button>
      {/if}
      {#each ways.filter((w) => w.dest === 'settings') as w (w.dest)}{@render destination(w)}{/each}
      <button class="wide" disabled={busy} onclick={() => call(S.LeaveGame)}>Leave the Game</button>
      <span class="why">Everything is saved as it happens{#if view.game}; this is game {view.game}{/if}.</span>
    </div>
  {/snippet}

  <!-- The story box has the place's narrative (ART-2B); what the player
       needs to know to play, saving, stays here, whether or not it is
       open. -->
  {#snippet story()}
    <p><b>The maze.</b> The party sets out from here.</p>
  {/snippet}

  <section class="about">
    <p><b>Saving.</b> Everything that happens to the party in the maze is saved after every
      action. If Istary closes during an expedition, it resumes where it was last saved; an encounter it
      was in is over, and its unclaimed rewards are lost.</p>
  </section>
</LocationFrame>

<style>
  .group { display: grid; gap: 4px; margin-bottom: 10px; }
  .group:last-child { margin-bottom: 0; }
  .wide { width: 100%; text-align: left; margin: 0; }
  .check-equipment { border: 1px solid var(--notice); border-radius: 6px; padding: 6px 8px; display: grid; gap: 4px; }
  .check-equipment p { margin: 0; }
  .choices { display: flex; gap: 6px; flex-wrap: wrap; }
  .choices button { margin: 0; }
  .why { color: var(--muted); font-size: 11px; }
  .about { margin-top: 0; }
  .about p { margin: 0 0 8px; }
  .about p:last-child { margin: 0; }
</style>
