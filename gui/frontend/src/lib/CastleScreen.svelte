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
  // box, scrolling inside it, closed with its × like
  // any other; once a game is open it is the Castle's own again (App
  // opens the box again when the game changes).
  //
  // Props:
  //   view    the Go View (view.location is "castle")
  //   busy    a request is running
  //   call    (fn, ...args) => Promise   runs a Session method
  //   onopen  (rosterIndex) => void      open a party member's sheet
  import LocationFrame from './LocationFrame.svelte'

  let { view, busy = false, call, onopen } = $props()
</script>

<!-- The Castle's description is its story (ART-2B): there is nothing here
     the player needs to play that the destinations do not say. -->
<LocationFrame {view} {busy} {call} {onopen} storyTitle={view.game ? undefined : 'Welcome to the Castle'}>
  {#snippet story()}
    {#if !view.game}
      <!-- The introductory story (NAV-1A), as approved. -->
      <p>You have arrived at the Castle of Trebor, the Mad Overlord, whose lands lie beneath the shadow of a
        terrible evil.</p>
      <p>Deep below the Castle, the wizard Werdna has hidden himself within a vast and treacherous Maze. There
        he guards a powerful amulet stolen from Trebor himself.</p>
      <p>The Overlord has called upon adventurers from every corner of the realm. Riches and glory await those
        brave enough to descend into the labyrinth, defeat Werdna, and return with the stolen amulet.</p>
      <p>Many have answered the call. Few have returned.</p>
      <p>Now it is your turn.</p>
      <p>But no adventurer should face the Maze alone. You must first gather a company of companions, each
        bringing their own strengths, skills and knowledge of magic.</p>
      <p><b>Your journey begins at the Edge of Town.</b></p>
      <p>There you may establish a new adventure and visit the Training Grounds, where warriors, priests, mages
        and thieves begin their journeys.</p>
      <p>Create your characters, choosing their names, races, professions and abilities. Then return to the
        Castle, where Gilgamesh's Tavern provides a gathering place for your party, Boltac's Trading Post offers
        equipment, and the Adventurer's Inn provides rest.</p>
      <p>When your company is assembled and prepared, the Maze awaits.</p>
      <p class="challenge">Trebor has issued his challenge. Will you answer it?</p>
    {:else}
    <p>The party is formed at <b>Gilgamesh's Tavern</b>. Beyond the castle walls, the <b>Edge of Town</b>
      leads to the Training Grounds, where characters are created, and to the maze.</p>
    {/if}
  {/snippet}

  {#if !view.game}
    <section class="no-game" aria-label="No game selected">
      <p><b>No game is selected.</b> Go to the Edge of Town and choose <b>Select Game</b> to continue a game
        or begin a new one; the Castle's other places open once a game is selected.</p>
    </section>
  {/if}
</LocationFrame>

<style>
  .no-game p { margin: 0; }
  /* The introductory story's last line: the story box's own gold, as its title. */
  .challenge { color: var(--gold); font-weight: 700; }
</style>
