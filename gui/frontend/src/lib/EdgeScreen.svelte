<script>
  // The Edge of Town (Castle-0 C0-4): outside the castle, where the party
  // sets out for the maze, practice is played (Istary's own, not the
  // original's) and the player leaves the game. Whether each departure is
  // open, and why not, is Go's (view.departures).
  //
  // Props:
  //   view    the Go View (view.location is "edge")
  //   busy    a request is running
  //   call    (fn, ...args) => Promise   runs a Session method
  //   onopen  (rosterIndex) => void      open a party member's sheet
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'

  let { view, busy = false, call, onopen } = $props()

  let practiceChest = $state(false) // practice encounter with a chest
  const real = $derived(view.departures?.real ?? '')
  const practice = $derived(view.departures?.practice ?? '')
</script>

<LocationFrame {view} {busy} {call} {onopen}>
  {#snippet actions()}
    <div class="group">
      <button class="primary wide" disabled={busy || !!real} onclick={() => call(S.RealExpedition)}>Enter the maze</button>
      {#if real}<span class="why">{real}</span>{/if}
    </div>
    <div class="group practice">
      <div class="tag">Practice · Istary only · nothing is kept</div>
      <button class="wide" disabled={busy || !!practice} onclick={() => call(S.PracticeExpedition)}>Practice expedition</button>
      <button class="wide" disabled={busy || !!practice} onclick={() => call(S.PracticeEncounter, practiceChest)}>Practice encounter</button>
      <label class="muted check"><input type="checkbox" bind:checked={practiceChest} /> the encounter guards a chest</label>
      {#if practice}<span class="why">{practice}</span>{/if}
    </div>
    <div class="group">
      <button class="wide" disabled={busy} onclick={() => call(S.LeaveGame)}>Leave the game</button>
      <span class="why">Everything is saved as it happens.</span>
    </div>
  {/snippet}

  <section class="about">
    <p><b>The maze.</b> The party sets out from here. Everything that happens to it is saved after every
      action. If Istary closes during an expedition, it resumes where it was last saved; an encounter it
      was in is over, and its unclaimed rewards are lost.</p>
    <p class="muted"><b>Practice</b> is Istary's own: a practice expedition walks the real maze with copies of the
      party, encounters included; a practice encounter fights one encounter from the maze entrance. Nothing
      either changes is kept.</p>
  </section>
</LocationFrame>

<style>
  .group { display: grid; gap: 4px; margin-bottom: 10px; }
  .group:last-child { margin-bottom: 0; }
  .wide { width: 100%; text-align: left; margin: 0; }
  .practice { border: 1px dashed #444; border-radius: 6px; padding: 6px 8px; }
  .tag { font-size: 11px; color: var(--notice); text-transform: uppercase; letter-spacing: 0.05em; }
  .check { margin: 0; }
  .check input { margin: 0 4px 0 0; }
  .why { color: var(--muted); font-size: 11px; }
  .about { margin-top: 0; }
  .about p { margin: 0 0 8px; }
  .about p:last-child { margin: 0; }
</style>
