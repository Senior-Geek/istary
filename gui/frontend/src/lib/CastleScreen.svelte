<script>
  // The Castle (Castle-0 C0-4): the hub, where every game opens (NAV-1).
  // Its destinations are the Tavern, the Adventurer's Inn, Boltac's Trading
  // Post, the Temple of Cant (the last three closed, with Go's reason) and
  // the Edge of Town; it has no actions of its own, so no Actions section
  // (the character sheet's toggle is in the top bar).
  //
  // Istary always opens here (NAV-1A). With no game open (view.game 0) the
  // Edge of Town is the one way open, and the others say why (Go's
  // reasons): a game is chosen with Select Game there. Its story is then
  // the introductory one ("Welcome to the Castle"), in the same story
  // box, scrolling inside it, closed with its × like any other; once a
  // game is open it is the Castle's own again (App opens the box again
  // when the game changes). Both are Markdown files, gui/content/castle
  // (CONTENT-1A), chosen by Go (story.go) and drawn by the frame.
  //
  // Props:
  //   view    the Go View (view.location is "castle")
  //   busy    a request is running
  //   call    (fn, ...args) => Promise   runs a Session method
  //   onopen  (rosterIndex) => void      open a party member's sheet
  import LocationFrame from './LocationFrame.svelte'

  let { view, busy = false, call, onopen } = $props()
</script>

<!-- The Castle's description is its story (ART-2B), gui/content/castle:
     there is nothing here the player needs to play that the destinations
     do not say. -->
<LocationFrame {view} {busy} {call} {onopen}>
  {#if !view.game}
    <section class="no-game" aria-label="No game selected">
      <p><b>No game is selected.</b> Go to the Edge of Town and choose <b>Select Game</b> to continue a game
        or begin a new one; the Castle's other places open once a game is selected.</p>
    </section>
  {/if}
</LocationFrame>

<style>
  .no-game p { margin: 0; }
</style>
