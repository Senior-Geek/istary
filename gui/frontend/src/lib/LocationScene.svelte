<script>
  // A place's artwork (Castle-0). It shows the place's picture (scenes.js),
  // cropped to fill its area around the picture's focus, or, while there is
  // no picture or if it fails to load, a themed panel with the place's
  // emblem and name. It fills the area the frame gives it and never holds
  // controls or text the player needs: the frame keeps those beside it.
  //
  // Props:
  //   location  where the player is (view.location)
  //   title     the place's name, shown on the themed panel
  import { sceneFor } from './scenes.js'

  let { location, title } = $props()

  const scene = $derived(sceneFor(location))
  let failed = $state('') // the src that failed to load
</script>

<figure class="scene" aria-label={scene.alt || title}>
  {#if scene.src && failed !== scene.src}
    <img src={scene.src} alt={scene.alt} style:object-position={scene.focus}
      onerror={() => { failed = scene.src }} />
  {:else}
    <div class="fallback" data-place={location}>
      <span class="emblem" aria-hidden="true">{scene.emblem}</span>
      <span class="name" aria-hidden="true">{title}</span>
    </div>
  {/if}
</figure>

<style>
  .scene { margin: 0; width: 100%; height: 100%; min-height: 0; overflow: hidden; border-radius: 6px; border: 1px solid var(--line); background: var(--panel); }
  img { display: block; width: 100%; height: 100%; object-fit: cover; }
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
