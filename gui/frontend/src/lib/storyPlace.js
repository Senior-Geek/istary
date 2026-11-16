// Where the story box stands at a place (ART-3B-3C). The player's place
// for it ({ x, y, w, h }, App's PANELS story.rect) is the same at every
// place, but the places' controls are not where each other's are: a box
// left over the Edge's display can stand over Boltac's tabs. So, as a
// place is entered (or its story opened), clearOf takes the player's
// place and the controls in view there, and gives the place to draw it:
// the player's own if it covers none of them; otherwise the nearest one
// that covers none, inside the window: moved up or down or made shorter,
// its left and width kept, whichever changes it least; only if there is
// no such place, moved aside (which would put it over the place's text);
// or null if there is none at all, and the box opens where it opens for a
// player who never moved it. The player's place itself is not changed:
// the next place starts from it again.

// clearOf returns r, a clear place near it, or null.
//   r          { x, y, w, h }: the player's place, already inside the window
//   controls   [{ x, y, w, h }]: the controls in view, the box's own aside
//   win        { w, h }: the window
//   margin     kept this far inside the window
//   minH       the box's least height
//   gap        kept this far from a control
export function clearOf(r, controls, win, { margin = 8, minH = 90, gap = 6 } = {}) {
  const hits = (p) => controls.some((c) => p.x < c.x + c.w + gap && c.x < p.x + p.w + gap && p.y < c.y + c.h + gap && c.y < p.y + p.h + gap)
  if (!hits(r)) return r
  const inside = (p) => p.h >= minH && p.x >= margin - 0.5 && p.y >= margin - 0.5 && p.x + p.w <= win.w - margin + 0.5 && p.y + p.h <= win.h - margin + 0.5
  const upDown = [], aside = []
  for (const c of controls) {
    upDown.push({ ...r, y: c.y - gap - r.h }) // above it
    upDown.push({ ...r, y: c.y + c.h + gap }) // below it
    if (c.y > r.y) upDown.push({ ...r, h: c.y - gap - r.y }) // shorter, ending above it
    const top = Math.max(margin, c.y - gap - r.h) // as high as it goes, then shorter
    if (c.y > top) upDown.push({ ...r, y: top, h: Math.min(r.h, c.y - gap - top) })
    aside.push({ ...r, x: c.x - gap - r.w }) // left of it
    aside.push({ ...r, x: c.x + c.w + gap }) // right of it
  }
  const cost = (p) => Math.abs(p.x - r.x) + Math.abs(p.y - r.y) + Math.abs(p.h - r.h)
  const nearest = (tries) => tries.filter((p) => inside(p) && !hits(p)).reduce((best, p) => (!best || cost(p) < cost(best) ? p : best), null)
  return nearest(upDown) ?? nearest(aside)
}
