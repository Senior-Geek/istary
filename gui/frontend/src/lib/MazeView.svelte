<script>
  // The first-person picture of the maze (M-ART-1A.1, 1A.2). It draws the
  // dungeon view (Go's app.DungeonView) and nothing else: the perspective
  // scene (mazeScene.js) takes what Go saw (the corridor, darkness and
  // light), the projection (mazeProjection.js) places it, and the style
  // (mazeStyle.js) says how each part is painted: Classic, the Wizardry
  // wireframe, or Enhanced, the same surfaces as stone. The zoom is only
  // the picture's frame. It keeps no state and takes no input; each new view
  // redraws it.
  //
  // Props:
  //   dungeon   view.dungeon
  //   style     'classic' or 'enhanced'
  //   zoom      'close', 'standard' or 'wide'
  import { perspective, points } from './mazeProjection.js'
  import { styles, zooms, painted, segmentsPath } from './mazeStyle.js'

  let { dungeon, style = 'classic', zoom = 'standard' } = $props()

  const drawn = $derived(painted(perspective(dungeon), styles[style] ?? styles.classic))
  const frame = $derived(zooms[zoom] ?? zooms.standard)
</script>

<svg viewBox={frame} class="maze-view {styles[style] ? style : 'classic'}" role="img" aria-label="The corridor ahead">
  {#if drawn.some((p) => p.tone)}
    <defs>
      {#each drawn as p (p.id)}
        {#if p.tone}
          <linearGradient id="mz-{p.id}" gradientUnits="userSpaceOnUse"
            x1={p.tone.axis[0]} y1={p.tone.axis[1]} x2={p.tone.axis[2]} y2={p.tone.axis[3]}>
            <stop offset="0" stop-color={p.tone.from} />
            <stop offset="1" stop-color={p.tone.to} />
          </linearGradient>
        {/if}
      {/each}
    </defs>
  {/if}
  {#each drawn as p (p.id)}
    <polygon class={p.paint} points={points(p)} data-surface={p.id} data-depth={p.depth}
      style:fill={p.fill} style:stroke={p.edge ?? (p.tone ? p.fill : undefined)} />
    {#each p.details ?? [] as x (x.kind)}
      <path class="detail {x.kind}" d={segmentsPath(x.segments)} data-detail={p.id} />
    {/each}
  {/each}
</svg>

<style>
  /* The size the dungeon screen gives it (a square: DungeonScreen's
     .picture), whole. */
  .maze-view { display: block; width: 100%; height: 100%; }
  .classic polygon { fill: none; stroke: #d8d2c4; stroke-width: 0.6; vector-effect: non-scaling-stroke; }
  /* Walls are filled, so a nearer one hides what lies behind it. */
  .classic polygon.wall { fill: #1b1915; }
  .classic polygon.door { stroke: #f2c14e; stroke-width: 1.4; }
  /* Where the view ends unseen: darkness, with no edge to suggest a wall. */
  .classic polygon.void { fill: #0d0c0b; stroke: none; }

  /* Enhanced: each surface's colour is its dress (mazeStyle.js). Floors and
     ceilings are outlined in their own colour, so neighbours meet without a
     hairline; walls have a worn edge; a door, a heavy stone frame. */
  .enhanced polygon { stroke-width: 0.8; stroke-linejoin: round; vector-effect: non-scaling-stroke; }
  .enhanced polygon.wall { stroke-width: 0.7; }
  .enhanced polygon.door { stroke-width: 2.2; }
  .enhanced polygon.void { stroke: none; }
  .enhanced .detail { fill: none; stroke-linecap: round; vector-effect: non-scaling-stroke; }
  .enhanced .detail.mortar { stroke: #000; stroke-opacity: 0.38; stroke-width: 0.8; }
  .enhanced .detail.flag { stroke: #000; stroke-opacity: 0.32; stroke-width: 0.7; }
  .enhanced .detail.plank { stroke: #000; stroke-opacity: 0.45; stroke-width: 0.8; }
  .enhanced .detail.band { stroke: #151310; stroke-opacity: 0.85; stroke-width: 1.6; }
</style>
