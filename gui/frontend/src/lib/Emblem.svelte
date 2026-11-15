<script>
  // A place's emblem (ART-3B-0; emblems.js), drawn inline in the colour
  // around it (currentColor): the appearance's gold in a title, the
  // place's tint on its themed panel. It is decoration: the title beside
  // it names the place, so screen readers skip it. Nothing where the place
  // has no emblem.
  //
  // Props:
  //   location  the place (view.location)
  //   size      its width and height, a CSS length (default 1em)
  import { emblemFor } from './emblems.js'

  let { location, size = '1em' } = $props()

  const emblem = $derived(emblemFor(location))
</script>

{#if emblem}
  <svg class="emblem" data-emblem={location} viewBox="0 0 64 64" style:width={size} style:height={size}
    fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"
    aria-hidden="true" focusable="false">
    {#each emblem.shapes as { tag, ...attrs }}
      {#if tag === 'circle'}<circle {...attrs} />{:else}<path {...attrs} />{/if}
    {/each}
  </svg>
{/if}

<style>
  svg { display: block; flex: none; overflow: visible; }
</style>
