# NAV-1A: Castle-first startup

**Baseline:** v0.14.0-nav (`85e2229`). **Branch:** `nav-1a-castle-startup`.

Istary always opens at the Castle, with or without a game. Game selection
is no longer what the window opens on; it is reached from the Edge of
Town's Select Game. No rule, save format or saved game changes.

## What was there (the inspection)

- **Startup:** a library window (`NewLibrary`, the app's window) opened
  on game selection with no game open (`Session.open`), and Continue or
  New Game opened a game at the Castle (or its expedition, resumed).
- **No game open** was `Session.ctrl == nil`: every game action found the
  game closed (`Session.closed`) and did nothing, and the view was game
  selection's.
- **The last game** was not remembered anywhere.
- **Select Game** (`ChangeGame`) closed the open game (it is saved as
  every action is) and showed game selection, with no way back.

## What changed

- **Startup** (`gui/startup.go`): the window opens at the Castle. The
  game last chosen opens again if its slot still holds a game that opens;
  a saved expedition resumes in the maze, as on Continue. Otherwise no
  game is open: none was ever chosen, its slot is empty or damaged, the
  slot number is not one of the six, or the window's settings cannot be
  read. The Castle then says why, and nothing is written: neither the
  game nor the window's settings.
- **The last game** is the window's setting `lastGame` in
  `gui-settings.json` (with the appearance and the party toggle), written
  when a game is chosen: Continue, New Game, Import Earlier Game Here or
  Replace This Game. It is never in a game's save. Older Istarys ignore it.
- **No game open** (the lobby): the Castle, the Edge of Town and Settings,
  the window's own places here, not the game's.
  - **The Castle:** only the Edge of Town is open; the Tavern, the Inn,
    Boltac's and the Temple are shown closed: "Select a game first:
    Select Game is at the Edge of Town." Its information says no game is
    selected, and its story is the introductory one (below).
  - **The Edge of Town:** Enter the Maze and the Training Grounds closed,
    with the same reason; Select Game, Settings and Leave the Game open.
  - **Settings:** the appearance only, the window's; the roster tools are
    a game's and are not shown.
  - **Escape** and "← Castle" return to the Castle from the Edge,
    Settings and game selection.
  - Every game action still finds no game open and does nothing, so
    nothing here reaches the engine (`TestNoGameOpenActsOnNothing` calls
    every Session method).
- **Select Game:** shows game selection, closing the open game first if
  there is one (as before). Game selection now has "← Castle" (Escape),
  which leaves it for the Castle with no game open. Choosing a game opens
  it at the Castle, or its expedition resumed.
- **Leave the Game** closes the window with or without a game.

## The introductory story

With no game open the Castle's story box (ART-2B) tells the introductory
story instead of the Castle's own, over the Castle's picture, titled
"Welcome to the Castle". It is the same story box, with its × and the
top bar's Story toggle: non-modal, closed and reopened like any story,
and blocking nothing. It opens in the picture's area, no taller than it,
and the text scrolls inside it (at 980×692 and 1440×900 it is longer
than the box), clear of the destinations and the party. Its last line is
emphasised in the story box's own gold, the colour of its title.

When a game opens (or closes) the story box opens again (App), so a
game's Castle shows the Castle's own story. A window that opens with its
last game goes straight to the Castle's own story; one whose last game
is gone shows the introductory story again. Nothing of it is saved, in a
game or in the window's settings.

The approved text (2026-11-14):

> **Welcome to the Castle**
>
> You have arrived at the Castle of Trebor, the Mad Overlord, whose lands
> lie beneath the shadow of a terrible evil.
>
> Deep below the Castle, the wizard Werdna has hidden himself within a
> vast and treacherous Maze. There he guards a powerful amulet stolen
> from Trebor himself.
>
> The Overlord has called upon adventurers from every corner of the
> realm. Riches and glory await those brave enough to descend into the
> labyrinth, defeat Werdna, and return with the stolen amulet.
>
> Many have answered the call. Few have returned.
>
> Now it is your turn.
>
> But no adventurer should face the Maze alone. You must first gather a
> company of companions, each bringing their own strengths, skills and
> knowledge of magic.
>
> **Your journey begins at the Edge of Town.**
>
> There you may establish a new adventure and visit the Training Grounds,
> where warriors, priests, mages and thieves begin their journeys.
>
> Create your characters, choosing their names, races, professions and
> abilities. Then return to the Castle, where Gilgamesh's Tavern provides
> a gathering place for your party, Boltac's Trading Post offers
> equipment, and the Adventurer's Inn provides rest.
>
> When your company is assembled and prepared, the Maze awaits.
>
> **Trebor has issued his challenge. Will you answer it?**

## Decisions (approved 2026-11-14)

1. **Startup:** Istary always opens at the Castle, unless a valid
   expedition must resume in the maze; the last valid game opens by
   itself; with none, the Castle with the introductory story, only the
   Edge of Town open.
2. **Select Game** closes the open game at once, as before; the game
   stays the one remembered for the next start. Choosing or creating a
   game opens it at the Castle.
3. **Settings** with no game shows only the window's appearance; the
   roster tools wait for a game.
4. **Escape** and the maze's and combat's restrictions are unchanged.

## Tests

- **Go** (`gui/nav1a_test.go`):
  - first launch with no games: the Castle with no game, its ways out
    and their reason, the Edge's and Settings', the maze, the Training
    Grounds and the roster tools refused, Escape, nothing written;
  - Select Game with no game open and in a game, ← Castle out of it,
    creating and continuing games, each remembered;
  - the last game opening again at the Castle, without writing its save
    or the settings, and after Leave the Game;
  - Leave the Game with no game;
  - a missing, damaged or impossible remembered game, or damaged
    settings: the Castle with no game, saying why, nothing rewritten.
  - The expedition test (`TestAnExpeditionContinuesWhereItWasSaved`)
    now resumes the expedition on starting, by itself.
- **Updated tests:** the library helpers open at the Castle; tests that
  read game selection reach it by Select Game (`selectGame`), as the
  player does; tests that start again find the last game open
  (`openedLast`), or start with none remembered (`relibraryNoGame`) where
  they test game selection after a restart. The game selection fixture
  is unchanged.
- **Frontend** (`nav1a.test.js`): the no-game views recorded from Go
  (the Castle, the Edge, Settings), the Edge's Select Game, Settings
  without the roster tools, game selection's way back, and the
  introductory story: the approved text's eleven paragraphs, its title
  and its emphasised challenge, and the Castle's own story in a game.
- **Live** (`--suites startup`, with `nav` for town and UI changes): the
  no-game Castle, Edge and Settings; the introductory story, its title,
  its scrolling inside the box to the last line, the box clear of the
  controls, the challenge's emphasis, and its ×; Escape; and Select Game
  and Continue back to the Castle's own story, at 980×692 and 1440×900.

## Notes

- **Relaunch** is tested in Go (a new session on the same folder), not
  live: the live harness keeps one window.
- **Leaving game selection** returns to the Castle with no game, even
  when Select Game closed one; that game is still the one remembered, so
  the next start opens it.
