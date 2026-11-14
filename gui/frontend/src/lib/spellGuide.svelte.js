// The spell guide (COMBAT-1): Go's one description of every spell
// (app.SpellGuide), loaded once when the window opens. The character sheet
// reads it here, so it still never calls the backend itself, and no screen
// writes a spell description of its own. The lookup is sheet.js knownSpells.

export const guide = $state({ spells: [] })

// loadGuide fetches the guide with fn (Session.SpellGuide).
export async function loadGuide(fn) {
  try {
    guide.spells = (await fn()) ?? []
  } catch {
    guide.spells = [] // the sheet then lists the names alone
  }
}
