<script>
  // One map drawing (M-1), for the minimap and the full map alike: thin
  // green walls, yellow door marks, the stairs and elevator symbols, the
  // party's white arrow (a feature under it moves to a corner badge), a
  // faint floor on the squares the party has stood on, and (full map) the chevrons where a seen way
  // crosses the level's edge, or (minimap) a faint line where the level's
  // edge runs through the window. It draws lib/automap.js's drawing of
  // Go's map view and decides nothing.
  //
  // Where stranded characters lie (R-3) is a red diamond, one a square,
  // with the count inside when more than one lie there; drawn under the
  // party's arrow, and moved to a corner badge on the party's own square.
  // On the full map a marker can be clicked (onselect); the map's list
  // beside it does the same from the keyboard.
  //
  // Props:
  //   view      app.LocalMapView or app.MapView
  //   full      the full map (frame and edge chevrons) rather than the minimap
  //   label     the accessible name
  //   selected  the place (L:R:C) of the marker chosen, if any
  //   onselect  (place) => void   a marker was clicked
  import { drawing, CLIP, BADGE, MARKER, diamond } from './automap.js'

  let { view, full = false, label = 'Map', selected = '', onselect = null } = $props()

  // A marker is drawn larger on the full map, whose squares are smaller,
  // so its count can be read; a badge keeps its size, clear of the arrow.
  const grow = (m) => (full && !m.badge ? m.scale * 1.4 : m.scale)

  const d = $derived(drawing(view))
  const pad = $derived(full ? 0.7 : CLIP)
  const box = $derived(d ? `${-pad} ${-pad} ${d.size + 2 * pad} ${d.size + 2 * pad}` : '0 0 1 1')
</script>

{#if d}
  <svg viewBox={box} class="map" class:full role="img" aria-label={label}>
    <!-- A marker's drawing; declared inside the svg, so it is drawn as SVG. -->
    {#snippet mark(m)}
      <title>{m.label}</title>
      <polygon points={diamond} />
      {#if m.count > 1}<text y="0.02">{m.count}</text>{:else}<circle class="dot" r="0.07" />{/if}
    {/snippet}
    <defs><clipPath id="window-{full ? 'full' : 'local'}"><rect x={-CLIP} y={-CLIP} width={d.size + 2 * CLIP} height={d.size + 2 * CLIP} /></clipPath></defs>
    {#if full}<rect class="frame" x="0" y="0" width={d.size} height={d.size} />{/if}
    <g clip-path={full ? null : `url(#window-local)`}>
      {#if d.floor}<path class="floor" d={d.floor} />{/if}
      {#each d.seams as s, i (i)}<line class="seam" x1={s.x1} y1={s.y1} x2={s.x2} y2={s.y2} />{/each}
      {#each d.walls as w, i (i)}
        <line class="wall" class:oneway={w.oneWay} x1={w.x1} y1={w.y1} x2={w.x2} y2={w.y2} />
        {#if w.door}<line class="door" x1={w.door.x1} y1={w.door.y1} x2={w.door.x2} y2={w.door.y2} />{/if}
      {/each}
      {#each d.features as f, i (i)}
        {#if f.badge}<circle class="badge" cx={f.x} cy={f.y} r={BADGE.radius} />{/if}
        <path class="feature {f.cls}" class:badged={f.badge} transform="translate({f.x} {f.y}) scale({f.scale})" d={f.path}><title>{f.label}</title></path>
      {/each}
      {#each d.stranded as m (m.place)}
        {#if m.badge}<circle class="badge" cx={m.x} cy={m.y} r={MARKER.radius} />{/if}
        {#if onselect}
          <g class="stranded pick" class:selected={m.place === selected} transform="translate({m.x} {m.y}) scale({grow(m)})"
            role="button" tabindex="-1" aria-label={m.label} onclick={() => onselect(m.place)}
            onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') onselect(m.place) }}>
            {@render mark(m)}
          </g>
        {:else}
          <g class="stranded" transform="translate({m.x} {m.y}) scale({grow(m)})">{@render mark(m)}</g>
        {/if}
      {/each}
      {#if d.party}
        <polygon class="party" points={d.party.points} transform="rotate({d.party.turn} {d.party.cx} {d.party.cy})" />
      {/if}
    </g>
    {#each d.wraps as w, i (i)}<polyline class="wrap" class:door={w.kind === 'door'} points={w.points} />{/each}
  </svg>
{/if}

<style>
  .map { display: block; width: 100%; height: 100%; }
  .frame { fill: none; stroke: rgba(79, 209, 106, 0.18); stroke-width: 0.04; stroke-dasharray: 0.2 0.2; }
  .floor { fill: rgba(79, 209, 106, 0.07); }
  .seam { stroke: rgba(79, 209, 106, 0.35); stroke-width: 0.03; stroke-dasharray: 0.12 0.12; }
  .wall { stroke: #4fd16a; stroke-width: 0.07; stroke-linecap: round; }
  .wall.oneway { stroke-width: 0.06; stroke-dasharray: 0.16 0.08; }
  .door { stroke: #f2c14e; stroke-width: 0.2; stroke-linecap: butt; }
  .feature { fill: none; stroke-width: 0.07; stroke-linecap: round; stroke-linejoin: round; }
  .feature.stairs { stroke: #9fd3ff; }
  .feature.elevator { stroke: #d6b3ff; }
  .badge { fill: #000; stroke: rgba(255, 255, 255, 0.25); stroke-width: 0.025; }
  .feature.badged { stroke-width: 0.1; }
  .party { fill: #fff; stroke: #000; stroke-width: 0.03; }
  .wrap { fill: none; stroke: #4fd16a; stroke-width: 0.08; stroke-linecap: round; stroke-linejoin: round; }
  .wrap.door { stroke: #f2c14e; }
  .stranded polygon { fill: #5a1712; stroke: #ff7a6b; stroke-width: 0.06; stroke-linejoin: round; }
  .stranded text { fill: #fff; font: 700 0.38px ui-monospace, Menlo, monospace; text-anchor: middle; dominant-baseline: central; pointer-events: none; }
  .stranded .dot { fill: #ff7a6b; }
  .stranded.pick { cursor: pointer; outline: none; }
  .stranded.selected polygon { stroke: #fff; stroke-width: 0.09; }
</style>
