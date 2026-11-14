<script>
  // A place's artwork (Castle-0, ART-2A). It shows the place's picture
  // (scenes.js), cropped to cover its area around the picture's focus, never
  // stretched, or, while there is no picture or if it fails to load, a
  // themed panel with the place's emblem and name. It never holds controls
  // or text the player needs.
  //
  // Props:
  //   location  where the player is (view.location)
  //   title     the place's name, shown on the themed panel
  //   mode      how the frame uses it:
  //             panel     the picture fills the display area (a place
  //                       without artwork, or a place's own scene)
  //             backdrop  the picture fills the window, behind the frame
  //                       (an illustrated place); hidden from screen
  //                       readers, which read the view instead
  //             view      the display area of an illustrated place: a clear
  //                       opening onto the backdrop, described for screen
  //                       readers
  //   artwork   how an illustrated place's picture is shown (ART-2C;
  //             backdrop and view): full; subdued, darker and less
  //             saturated, so it draws less of the eye (the panels over it
  //             are unchanged); or hidden, a plain backdrop in the place's
  //             tint, and nothing described
  import { sceneFor } from './scenes.js'

  let { location, title, mode = 'panel', artwork = 'full' } = $props()

  const scene = $derived(sceneFor(location))
  let failed = $state('') // the src that failed to load
  const shown = $derived(!!scene.src && failed !== scene.src && (mode === 'panel' || artwork !== 'hidden'))
</script>

{#snippet fallback(plain = false)}
  <div class="fallback" class:plain data-place={location}>
    {#if !plain}
      <span class="emblem" aria-hidden="true">{scene.emblem}</span>
      <span class="name" aria-hidden="true">{title}</span>
    {/if}
  </div>
{/snippet}

{#if mode === 'backdrop'}
  <div class="backdrop" aria-hidden="true" data-place={location} data-artwork={artwork}>
    {#if shown}
      <img src={scene.src} alt="" style:object-position={scene.focus} onerror={() => { failed = scene.src }} />
    {:else}{@render fallback(artwork === 'hidden')}{/if}
  </div>
{:else if mode === 'view'}
  {#if artwork === 'hidden'}
    <figure class="scene view" aria-hidden="true"></figure>
  {:else}
    <figure class="scene view" role="img" aria-label={scene.alt || title}></figure>
  {/if}
{:else}
  <figure class="scene" aria-label={scene.alt || title}>
    {#if shown}
      <img src={scene.src} alt={scene.alt} style:object-position={scene.focus}
        onerror={() => { failed = scene.src }} />
    {:else}{@render fallback()}{/if}
  </figure>
{/if}

<style>
  .scene { margin: 0; width: 100%; height: 100%; min-height: 0; overflow: hidden; border-radius: 6px; border: 1px solid var(--line); background: var(--panel); }
  /* The opening onto the backdrop: nothing drawn, the picture shows, and
     a pointer reaches it. */
  .scene.view { border-color: transparent; background: transparent; pointer-events: none; }
  /* At the bottom of the shell's stacking context (main.shell.art, which is
     above the page's background): the frame's panels and the banners draw
     over it. */
  .backdrop { position: fixed; inset: 0; z-index: -1; overflow: hidden; background: #161616; }
  img { display: block; width: 100%; height: 100%; object-fit: cover; }
  /* Subdued (ART-2C): the picture darker and less saturated, so it draws
     less of the eye; the panels over it are unchanged. */
  .backdrop[data-artwork='subdued'] img { filter: brightness(0.55) saturate(0.6); }
  /* Hidden: the place's tint alone, and no emblem. */
  .fallback.plain { background: radial-gradient(ellipse at 50% 40%, rgba(var(--tint), 0.14), transparent 75%), #161616; }
  .fallback {
    --tint: 120, 120, 120;
    width: 100%; height: 100%;
    display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px;
    background:
      radial-gradient(ellipse at 50% 40%, rgba(var(--tint), 0.28), transparent 70%),
      repeating-linear-gradient(135deg, rgba(255, 255, 255, 0.025) 0 2px, transparent 2px 12px),
      #161616;
    color: rgba(var(--tint), 0.9);
  }
  .emblem { font-size: clamp(32px, 9vmin, 88px); line-height: 1; font-family: "Apple Symbols", "Segoe UI Symbol", serif; }
  .name { font-size: 12px; letter-spacing: 0.18em; text-transform: uppercase; opacity: 0.8; }
  .fallback[data-place='castle'] { --tint: 214, 178, 96; }
  .fallback[data-place='tavern'] { --tint: 206, 132, 78; }
  .fallback[data-place='inn'] { --tint: 132, 150, 214; }
  .fallback[data-place='trading'] { --tint: 196, 170, 110; }
  .fallback[data-place='temple'] { --tint: 230, 220, 190; }
  .fallback[data-place='edge'] { --tint: 150, 170, 140; }
  .fallback[data-place='training'] { --tint: 126, 190, 112; }
  .fallback[data-place='settings'] { --tint: 160, 160, 160; }
  .fallback[data-place='maze'] { --tint: 120, 110, 150; }
</style>
