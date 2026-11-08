<script>
  // The Castle (Castle-0 C0-4): the hub. Its destinations are the Tavern,
  // the Adventurer's Inn, Boltac's Trading Post, the Temple of Cant (the
  // last three closed, with Go's reason) and the Edge of Town; it has no
  // actions of its own.
  //
  // Props:
  //   view    the Go View (view.location is "castle")
  //   busy    a request is running
  //   call    (fn, ...args) => Promise   runs a Session method
  //   onopen  (rosterIndex) => void      open a party member's sheet
  import LocationFrame from './LocationFrame.svelte'

  let { view, busy = false, call, onopen } = $props()
</script>

<LocationFrame {view} {busy} {call} {onopen}>
  <section class="about">
    <p>The party is formed at <b>Gilgamesh's Tavern</b>. Beyond the castle walls, the <b>Edge of Town</b>
      leads to the Training Grounds, where characters are created, and to the maze.</p>
  </section>
</LocationFrame>

<style>
  .about { margin-top: 0; }
  .about p { margin: 0; }
</style>
