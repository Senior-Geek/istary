# NAV-1: Castle and Edge of Town navigation

**Baseline:** v0.13.0-combat (`5c081d5`). **Branch:** `nav-1-town-navigation`.

A navigation and layout refinement, not a gameplay change: no rule, save
format or saved game changes, and the Escape logic is unchanged.

## What was there (the inspection)

- **Where a game opens:** already the Castle. The town location is not
  saved; a loaded game's town starts at the Castle (`engine.NewTownState`),
  and only an expedition under way resumes elsewhere, in the maze. Nothing
  needed changing; NAV-1 adds tests for it.
- **The character sheet's toggle** was in the Actions at every place in
  town (UX-3B), not only at the Castle, where it was the Actions' only
  entry. It is one control in the shared frame (`LocationFrame.svelte`).
- **The Edge of Town:** Enter the maze, the practice group (Practice
  expedition, Practice encounter, the chest checkbox), Change game and
  Leave the game in its Actions; the Training Grounds and Settings in a Go
  to list. Both are Go's exits (`View.Exits`), and Settings is the window's
  own place at the Edge (`gui/navigation.go`).
- **Practice:** no development mode exists. The Go tests call the practice
  Session methods directly, and the live checks already started practice
  through the bindings, not the buttons.

None of this conflicted with the brief.

## What changed

- **The top bar** (the frame's head): the place's title, "← Castle"
  (Escape) where there is a way back, then the toggles: Hide party, Story
  and Hide panels at the illustrated places, and the character sheet's
  toggle, moved from the Actions. Its label is "Show sheet" / "Hide sheet",
  worded as the party toggle is, and it is pressed while a sheet is shown.
  Settings is not in it.
- **The Castle:** no Actions section (it had only the sheet toggle), so its
  panel is its five destinations.
- **The Edge of Town:** its Actions are Enter the Maze, The Training
  Grounds, Select Game, Settings and Leave the Game, in that order, and it
  has no Go to list. The two places are its exits from Go, drawn by the
  frame's one destination snippet, which the frame hands to a place that
  lists its destinations among its actions (`waysInActions`). Change game
  is now Select Game, and it opens the existing game selection
  (`Session.ChangeGame`, unchanged). Its two refusals say Select Game.
- **Practice:** the practice controls and the paragraph about them are
  gone from the Edge. `PracticeExpedition`, `PracticeEncounter` and
  `AbandonPractice` are unchanged, for the tests.
- **Bindings:** `models.ts` regenerated. Since COMBAT-1 it had lacked
  `CampSpellView.summary` (the build always regenerated it, so the app
  was right).

## Behaviour

- **Startup:** the game slots, as before; Continue (or a new game) opens
  the Castle. An expedition under way resumes in the maze.
- **Escape:** from the Tavern, the Inn, Boltac's, the Temple, the Training
  Grounds, Settings and the Edge, back to the Castle. Hidden panels and a
  shown sheet take Escape first, as before. In the maze, Camp and combat,
  Escape is theirs, and the Castle is refused during an expedition or
  practice.
- **Boltac's:** no sheet or party toggle, as before (UX-2: the customer
  panel and the party bar are how the customer is chosen).

## Tests

- **Go** (`gui/nav1_test.go`):
  - a game opens at the Castle when new, continued after Select Game
    (left from Settings or the Training Grounds) and continued after a
    restart, without its save being written;
  - the Castle's and the Edge's exits;
  - Select Game refused away from the Edge, by its new name.
- **Frontend** (`nav1.test.js`):
  - the top bar's order, with no Settings;
  - the one sheet toggle, in the top bar;
  - the Castle's five destinations and no Actions;
  - the Edge's five actions in order, drawn by the frame's snippet, with
    no Go to list and no practice;
  - the practice bindings still there.

  These fail on v0.13.0's components, except the Castle's, which already
  had no actions of its own.
- **Live** (`--suites nav`, added to the acceptance level for town and UI
  changes):
  - Select Game and Continue;
  - every Castle destination and Edge action, and Escape from each place;
  - the top bar on one row at each place;
  - the toggles;
  - a real expedition entered with Enter the Maze, where Escape leaves
    nothing and the Castle is refused;
  - practice and combat through the Session methods, saving nothing.
- **Updated:**
  - `sheet.test.js` and `layout.test.js` for the toggle's place and label;
  - the live sheet and combat suites, which find the toggle in the top bar;
  - the scenes suite's panel check, which now names each place's panels:
    the Castle has three (top bar, destinations and party), the Edge four.

## Limitations and notes

- **Capitalisation:** the Edge's labels follow the brief ("Enter the
  Maze", "Select Game", "Leave the Game"), while the window's other
  buttons are in sentence case ("Hide party", "Continue into maze").
- **No way back at the Castle:** it has no "← Castle", as before; it is
  the place it would lead to.
- **No practice for players:** they can no longer start practice from the
  window.
