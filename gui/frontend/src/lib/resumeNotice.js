// The resumed expedition's notice (M-ART-1A.3): Go heads it EXPEDITION
// RESUMED (View.Title) and keeps it in every view until another notice
// replaces it, so the page decides how long it is shown. It is shown over
// the maze view for shownFor, then fades out for fadeFor (at once with
// reduced motion), and goes at once when the player acts in the dungeon.
// Only the page's own timers: nothing of it is Go's, saved or a game action.
//
// A notice is new when the view before had no resume notice, or another:
// so it is shown once for each resume, not again on each later view that
// still carries it, and again after the next resume (another game opened,
// Try Resuming Again).

export const resumedTitle = 'EXPEDITION RESUMED'
export const shownFor = 4000
export const fadeFor = 250

// The view's resume notice, or ''.
export const resumeNoticeOf = (view) => (view && !view.fatal && view.title === resumedTitle && view.notice) || ''

// resumeNotice keeps one notice's showing. onchange(state) is called with
// { text, phase } on each change: phase 'shown', 'fading' or 'hidden'
// (text '' when hidden). timers and reducedMotion are the window's unless
// given (the tests give their own).
export function resumeNotice({ onchange, timers = globalThis, reducedMotion = () => globalThis.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false } = {}) {
  let seen = '' // the notice in the last view, shown or not
  let state = { text: '', phase: 'hidden' }
  let timer = null
  let run = 0 // each showing's own: a callback from an earlier one does nothing
  const set = (text, phase) => { state = { text, phase }; onchange?.(state) }
  const stop = () => { if (timer !== null) timers.clearTimeout(timer); timer = null; run++ }
  const later = (ms, fn) => { const mine = run; timer = timers.setTimeout(() => { timer = null; if (mine === run) fn() }, ms) }
  const hide = () => { stop(); if (state.phase !== 'hidden') set('', 'hidden') }
  const fade = () => {
    if (reducedMotion()) return hide()
    set(state.text, 'fading')
    later(fadeFor, hide)
  }
  return {
    get state() { return state },
    // see takes each view the page is given; acted says it answers the
    // player's action in the dungeon (DungeonAction, CampAction, or the
    // maze's look). A new notice is shown; one already seen goes when the
    // action succeeded (no error) or the dungeon screen is left.
    see(view, acted = false) {
      const now = resumeNoticeOf(view)
      if (now && now !== seen) {
        seen = now
        stop()
        set(now, 'shown')
        later(shownFor, fade)
        return
      }
      seen = now
      if (!now || view.screen !== 'dungeon' || (acted && !view.error)) hide()
    },
    dismiss: hide,
    // The page is closing: no timer is left behind.
    destroy: stop,
  }
}
