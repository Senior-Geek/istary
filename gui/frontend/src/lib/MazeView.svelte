<script>
  // The first-person picture of the maze (M-ART-1A.1), in the Classic
  // style: the Wizardry wireframe. It draws the dungeon view (Go's
  // app.DungeonView) and nothing else: the perspective scene (mazeScene.js)
  // takes what Go saw (the corridor, darkness and light), the projection
  // (mazeProjection.js) places it, and the style (mazeStyle.js) says how
  // each part is painted. It keeps no state and takes no input; each new
  // view redraws it.
  //
  // Props:
  //   dungeon   view.dungeon
  import { viewBox, perspective, points } from './mazeProjection.js'
  import { classic, painted } from './mazeStyle.js'

  let { dungeon } = $props()

  const drawn = $derived(painted(perspective(dungeon), classic))
</script>

<svg {viewBox} class="maze-view classic" role="img" aria-label="The corridor ahead">
  {#each drawn as p (p.id)}
    <polygon class={p.paint} points={points(p)} data-surface={p.id} data-depth={p.depth} />
  {/each}
</svg>

<style>
  /* The size the dungeon screen gave its picture: square, at most 360px. */
  .maze-view { height: 100%; max-height: 360px; max-width: 100%; aspect-ratio: 1; }
  .classic polygon { fill: none; stroke: #d8d2c4; stroke-width: 0.6; vector-effect: non-scaling-stroke; }
  /* Walls are filled, so a nearer one hides what lies behind it. */
  .classic polygon.wall { fill: #1b1915; }
  .classic polygon.door { stroke: #f2c14e; stroke-width: 1.4; }
  /* Where the view ends unseen: darkness, with no edge to suggest a wall. */
  .classic polygon.void { fill: #0d0c0b; stroke: none; }
</style>
