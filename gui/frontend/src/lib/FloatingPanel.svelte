<script>
  // A movable, resizable, closable panel inside the main window. It knows
  // nothing about its contents; drag the title bar to move it and the
  // bottom-right corner to resize it. Escape closes it.
  let { title, onclose, x = 120, y = 60, width = 560, height = 560, children } = $props()

  let left = $state(x)
  let top = $state(y)
  let drag = null

  function startDrag(e) {
    if (e.button !== 0 || e.target.closest('button')) return
    drag = { dx: e.clientX - left, dy: e.clientY - top }
    e.currentTarget.setPointerCapture(e.pointerId)
  }
  function moveDrag(e) {
    if (!drag) return
    // Keep the title bar reachable inside the window.
    left = Math.min(Math.max(e.clientX - drag.dx, -width + 80), window.innerWidth - 80)
    top = Math.min(Math.max(e.clientY - drag.dy, 0), window.innerHeight - 40)
  }
  function endDrag() { drag = null }

  function onkeydown(e) {
    if (e.key === 'Escape') onclose?.()
  }
</script>

<svelte:window {onkeydown} />

<!-- Position and size are separate style directives so moving the panel
     does not undo a size the user set with the resize handle. -->
<div class="panel" role="dialog" aria-label={title}
  style:left="{left}px" style:top="{top}px" style:width="{width}px" style:height="{height}px">
  <div class="titlebar" role="toolbar" tabindex="-1"
    onpointerdown={startDrag} onpointermove={moveDrag} onpointerup={endDrag} onpointercancel={endDrag}>
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
    display: flex;
    flex-direction: column;
    min-width: 360px;
    min-height: 240px;
    max-width: calc(100vw - 16px);
    max-height: calc(100vh - 16px);
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
    padding: 6px 8px 6px 14px;
    background: #262626;
    border-bottom: 1px solid var(--line);
    cursor: move;
    user-select: none;
    touch-action: none;
  }
  .title { font-weight: 600; color: var(--accent); }
  .close { margin: 0; padding: 2px 8px; }
  .body { flex: 1; overflow: auto; padding: 12px 14px; }
</style>
