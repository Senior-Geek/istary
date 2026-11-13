// The embedded character sheet (UX-3B). The sheet is shown in the place's
// artwork area, not in a window of its own; App keeps whose sheet it is
// and whether it is shown, and gives the frame both through a context
// (SHEET). These helpers only choose and word; the sheet's content is
// Go's CharacterView.

export const SHEET = Symbol('sheet')

// subject is the roster character the sheet is about: the one chosen
// (index), or, while nobody is, the first party member who has a sheet,
// then the roster's first. null when there is nobody.
export function subject(view, index) {
  const roster = view?.roster ?? []
  const byIndex = (i) => roster.find((ch) => ch.index === i)
  if (index >= 0 && byIndex(index)) return byIndex(index)
  for (const p of view?.partyBar ?? []) {
    if (p.name && p.index >= 0 && byIndex(p.index)) return byIndex(p.index)
  }
  return roster[0] ?? null
}

// toggleLabel is the Actions button's label.
export function toggleLabel(shown) {
  return shown ? 'Hide Character Sheet' : 'Character Sheet'
}

// available says whether the sheet can be shown here: in town, at a place,
// and not while a character is being created.
export function available(view) {
  return !!view && view.screen === 'roster' && !!view.location && view.location !== 'maze'
}
