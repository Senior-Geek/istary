<script>
  // A movable, resizable, closable panel inside the main window. It knows
  // nothing about its contents; drag the title bar to move it and the
  // bottom-right corner to resize it. Escape closes it.
  //
  // It opens centered and always stays fully inside the window (see
  // panelGeometry.js), so the title bar and close button remain reachable
  // after any drag, resize, or change in window size.
  import { onMount } from 'svelte'
  import { MIN_WIDTH, MIN_HEIGHT, centered, constrain } from './panelGeometry.js'

  let { title, onclose, width = 560, height = 560, children } = $props()

  const min = { width: MIN_WIDTH, height: MIN_HEIGHT }
  const viewport = () => ({ width: window.innerWidth, height: window.innerHeight })

  let geom = $state(centered({ width, height }, viewport(), min))
  let panel
  let drag = null

  // While dragging, text selection is switched off for the whole page so
  // moving past the panel's limits cannot highlight text; the previous
  // setting is restored when the drag ends (release, cancel, or lost
  // pointer capture). Normal selection works whenever no drag is active.
  let savedUserSelect = null

  function startDrag(e) {
    if (e.button !== 0 || e.target.closest('button')) return
    e.preventDefault() // stop the press from starting a text selection
    drag = { dx: e.clientX - geom.left, dy: e.clientY - geom.top, id: e.pointerId }
    window.getSelection()?.removeAllRanges()
    const root = document.documentElement.style
    savedUserSelect = { user: root.userSelect, webkit: root.webkitUserSelect }
    root.userSelect = 'none'
    root.webkitUserSelect = 'none'
    try {
      e.currentTarget.setPointerCapture(e.pointerId)
    } catch {
      // Synthetic or already-released pointers cannot be captured; dragging
      // still works through the handlers below.
    }
  }
  function moveDrag(e) {
    if (!drag || e.pointerId !== drag.id) return
    e.preventDefault()
    geom = constrain({ ...geom, left: e.clientX - drag.dx, top: e.clientY - drag.dy }, viewport(), min)
  }
  function endDrag(e) {
    if (!drag || (e && e.pointerId !== drag.id)) return
    drag = null
    if (savedUserSelect) {
      const root = document.documentElement.style
      root.userSelect = savedUserSelect.user
      root.webkitUserSelect = savedUserSelect.webkit
      savedUserSelect = null
    }
    try {
      if (e && e.currentTarget?.hasPointerCapture?.(e.pointerId)) e.currentTarget.releasePointerCapture(e.pointerId)
    } catch {
      // Capture already gone.
    }
  }

  function onkeydown(e) {
    if (e.key === 'Escape') onclose?.()
  }
  function onresize() {
    geom = constrain(geom, viewport(), min)
  }

  onMount(() => {
    // A drag must never leave selection disabled, even if the panel closes
    // mid-drag.
    const restore = () => endDrag()
    // The corner handle resizes the element directly; fold its new size back
    // into geom so the panel is re-constrained (moved or limited) as needed.
    const ro = new ResizeObserver(() => {
      const w = panel.offsetWidth
      const h = panel.offsetHeight
      if (w !== geom.width || h !== geom.height) {
        geom = constrain({ ...geom, width: w, height: h }, viewport(), min)
        // The browser wrote the dragged size straight into the element's
        // style. If the constrained size equals what Svelte last applied,
        // Svelte sees no change and would leave the oversized value, so
        // write it back here.
        if (panel.offsetWidth !== geom.width) panel.style.width = `${geom.width}px`
        if (panel.offsetHeight !== geom.height) panel.style.height = `${geom.height}px`
      }
    })
    ro.observe(panel)
    return () => {
      ro.disconnect()
      restore()
    }
  })
</script>

<svelte:window {onkeydown} {onresize} />

<div class="panel" role="dialog" aria-label={title} bind:this={panel}
  style:left="{geom.left}px" style:top="{geom.top}px"
  style:width="{geom.width}px" style:height="{geom.height}px"
  style:min-width="{Math.min(min.width, geom.width)}px" style:min-height="{Math.min(min.height, geom.height)}px">
  <div class="titlebar" role="toolbar" tabindex="-1"
    onpointerdown={startDrag} onpointermove={moveDrag} onpointerup={endDrag} onpointercancel={endDrag}
    onlostpointercapture={endDrag}>
    <span class="title">{title}</span>
    <button class="close" aria-label="Close" onclick={() => onclose?.()}>✕</button>
  </div>
  <div class="body">
    {@render children?.()}
  </div>
</div>

<style>
  .panel {
    position: fixed;
    z-index: 10;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    resize: both;
    overflow: hidden;
    background: var(--panel);
    border: 1px solid #555;
    border-radius: 8px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.6);
  }
  .titlebar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex: none;
    padding: 6px 8px 6px 14px;
    background: #262626;
    border-bottom: 1px solid var(--line);
    cursor: move;
    user-select: none;
    touch-action: none;
  }
  .title { font-weight: 600; color: var(--accent); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .close { margin: 0 0 0 8px; padding: 2px 8px; flex: none; }
  .body { flex: 1; min-height: 0; overflow: auto; padding: 12px 14px; }
</style>
