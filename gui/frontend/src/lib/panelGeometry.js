// Geometry rules for FloatingPanel, kept free of the DOM so they can be
// tested with `node --test`.
//
// The panel always stays entirely inside the viewport (less a margin), which
// keeps its title bar and close button reachable. Sizes are at least the
// minimum but never larger than the viewport allows.

export const MARGIN = 8
export const MIN_WIDTH = 360
export const MIN_HEIGHT = 240

// fitSize limits a requested size to [minimum, viewport - margins]. When the
// viewport is smaller than the minimum, the viewport wins.
export function fitSize(size, viewport, min = { width: MIN_WIDTH, height: MIN_HEIGHT }, margin = MARGIN) {
  const maxW = Math.max(0, viewport.width - 2 * margin)
  const maxH = Math.max(0, viewport.height - 2 * margin)
  return {
    width: Math.min(Math.max(size.width, min.width), maxW),
    height: Math.min(Math.max(size.height, min.height), maxH),
  }
}

// clampPosition keeps a panel of the given size fully inside the viewport.
export function clampPosition(pos, size, viewport, margin = MARGIN) {
  const maxLeft = Math.max(margin, viewport.width - size.width - margin)
  const maxTop = Math.max(margin, viewport.height - size.height - margin)
  return {
    left: Math.round(Math.min(Math.max(pos.left, margin), maxLeft)),
    top: Math.round(Math.min(Math.max(pos.top, margin), maxTop)),
  }
}

// centered returns a fitted size and a position that centers it.
export function centered(size, viewport, min, margin = MARGIN) {
  const fitted = fitSize(size, viewport, min, margin)
  const pos = clampPosition(
    { left: (viewport.width - fitted.width) / 2, top: (viewport.height - fitted.height) / 2 },
    fitted, viewport, margin,
  )
  return { ...fitted, ...pos }
}

// constrain re-applies the rules to a panel's current geometry, e.g. after a
// drag, a user resize, or a change in viewport size.
export function constrain(geom, viewport, min, margin = MARGIN) {
  const fitted = fitSize(geom, viewport, min, margin)
  return { ...fitted, ...clampPosition(geom, fitted, viewport, margin) }
}
