<script>
  // The story box (ART-2B): a place's narrative, in a small window of its own
  // over the picture. It opens in the lower left of the display area (A),
  // where there are no controls; it can be dragged by its title and resized
  // by its corner (or, with the title focused, moved with the arrow keys and
  // resized with Shift and the arrow keys), always kept inside the window;
  // and closed with ×, to be opened again with the frame's Story button.
  // Where it is, once moved or resized, and whether it is open are the
  // window's, for the session (App, PANELS): nothing is saved or sent to Go.
  //
  // It holds narrative only. What the player needs to know to play (saving,
  // practice, warnings) stays in the frame's information, whether or not
  // the story box is open.
  //
  // Props:
  //   title     the story's title
  //   anchor    the element it opens over (the display area)
  //   look      its appearance (ART-2C): parchment, dark or slate, its own
  //             whatever the panels' (style.css [data-appearance])
  //   rect      { x, y, w, h } where the player last left it, or null: the
  //             lower left of the anchor, as wide as suits it, as tall as
  //             its text
  //   onmove    (rect) => void   it was moved or resized
  //   onclose   () => void       × was pressed
  //   children  the story
  import { onMount } from 'svelte'

  let { title, anchor, look = 'parchment', rect = null, onmove, onclose, children } = $props()

  const M = 8 // kept this far inside the window
  const MIN_W = 220, MIN_H = 90
  let box = $state()
  let win = $state({ w: innerWidth, h: innerHeight })
  let auto = $state(null) // where it opens while the player has not moved it
  const at = $derived(fit(rect ?? auto, win))

  // fit keeps r inside the window, no smaller than the minimum.
  function fit(r, { w: W, h: H }) {
    if (!r) return null
    const w = Math.max(Math.min(r.w, W - 2 * M), Math.min(MIN_W, W - 2 * M))
    const h = r.h == null ? null : Math.max(Math.min(r.h, H - 2 * M), Math.min(MIN_H, H - 2 * M))
    const tall = h ?? box?.offsetHeight ?? MIN_H
    return { ...r, w, h, x: Math.min(Math.max(r.x, M), W - w - M), y: Math.min(Math.max(r.y, M), H - tall - M) }
  }

  // The lower left of the anchor: as wide as a short line of reading, as
  // tall as the text (measured once drawn).
  function place() {
    const a = anchor?.getBoundingClientRect() ?? { x: M, y: M, width: innerWidth - 2 * M, bottom: innerHeight - M }
    const w = Math.round(Math.min(380, Math.max(MIN_W, a.width * 0.45)))
    auto = { x: a.x + 12, y: a.bottom - 12, w, h: null, measuring: true }
    requestAnimationFrame(() => {
      if (!box || rect) return
      const h = Math.min(box.offsetHeight, Math.max(MIN_H, a.bottom - a.y - 24))
      auto = { x: a.x + 12, y: a.bottom - 12 - h, w, h: box.offsetHeight > h ? h : null }
    })
  }
  onMount(place)

  function moved(r) {
    const { x, y, w, h } = fit({ ...r, h: r.h ?? box.offsetHeight }, win)
    onmove?.({ x, y, w, h })
  }

  // Dragging the title moves it; dragging the corner resizes it.
  function drag(e, how) {
    if (e.button !== 0 || e.target.closest('button')) return
    e.preventDefault()
    const start = { ...at, h: at.h ?? box.offsetHeight }
    const [px, py] = [e.clientX, e.clientY]
    const el = e.currentTarget
    el.setPointerCapture(e.pointerId)
    const step = (ev) => {
      const dx = ev.clientX - px, dy = ev.clientY - py
      moved(how === 'move' ? { ...start, x: start.x + dx, y: start.y + dy } : { ...start, w: start.w + dx, h: start.h + dy })
    }
    const end = () => { el.removeEventListener('pointermove', step); el.removeEventListener('pointerup', end); el.removeEventListener('pointercancel', end) }
    el.addEventListener('pointermove', step)
    el.addEventListener('pointerup', end)
    el.addEventListener('pointercancel', end)
  }

  function key(e) {
    const d = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] }[e.key]
    if (!d) return
    e.preventDefault()
    const n = e.altKey ? 1 : 16
    const r = { ...at, h: at.h ?? box.offsetHeight }
    moved(e.shiftKey ? { ...r, w: r.w + d[0] * n, h: r.h + d[1] * n } : { ...r, x: r.x + d[0] * n, y: r.y + d[1] * n })
  }
</script>

<svelte:window onresize={() => { win = { w: innerWidth, h: innerHeight }; if (!rect) place() }} />

{#if at}
  <aside class="story" bind:this={box} aria-label="Story: {title}" data-appearance={look}
    style:left="{at.x}px" style:top="{at.y}px" style:width="{at.w}px" style:height={at.h == null ? null : `${at.h}px`}
    style:visibility={at.measuring ? 'hidden' : null}>
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <header class="bar" tabindex="0" onpointerdown={(e) => drag(e, 'move')} onkeydown={key}
      title="Drag to move; with the keyboard, the arrow keys move it and Shift with them resizes it">
      <h3>{title}</h3>
      <button class="close" aria-label="Close the story" title="Close the story (the Story button opens it again)" onclick={onclose}>×</button>
    </header>
    <div class="text">{@render children?.()}</div>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="grip" aria-hidden="true" title="Drag to resize" onpointerdown={(e) => drag(e, 'size')}></div>
  </aside>
{/if}

<style>
  .story {
    position: fixed; z-index: 2; display: flex; flex-direction: column; min-height: 0;
    background: var(--glass); color: var(--text); border: 1px solid var(--glass-line); border-radius: 8px;
    -webkit-backdrop-filter: blur(6px); backdrop-filter: blur(6px); box-shadow: 0 6px 24px var(--glass-shadow);
  }
  .bar {
    display: flex; align-items: center; gap: 8px; padding: 6px 6px 4px 12px; cursor: move;
    touch-action: none; user-select: none; -webkit-user-select: none; border-radius: 8px 8px 0 0;
  }
  .bar:focus-visible { outline: 2px solid var(--focus); outline-offset: -2px; }
  h3 { flex: 1; margin: 0; font-size: 12px; text-transform: uppercase; letter-spacing: 0.08em; color: var(--gold); font-weight: 600; }
  .close { margin: 0; padding: 0 8px; line-height: 22px; font-size: 16px; }
  .text { flex: 1 1 auto; min-height: 0; overflow-y: auto; padding: 2px 14px 14px; font-size: 14px; line-height: 1.55; }
  .text :global(p) { margin: 0 0 8px; }
  .text :global(p:last-child) { margin: 0; }
  .grip {
    position: absolute; right: 0; bottom: 0; width: 16px; height: 16px;
    background: linear-gradient(135deg, transparent 50%, var(--glass-line) 50%, var(--glass-line) 60%, transparent 60%, transparent 70%, var(--glass-line) 70%, var(--glass-line) 80%, transparent 80%);
    cursor: nwse-resize; touch-action: none; border-radius: 0 0 8px 0;
  }
</style>
