# M-ART-1A.3: the larger picture, its look as a setting, and phase transitions

**Branch:** `m-art-1a-maze-perspective`, on M-ART-1A.2 (`835e082`).
**Status:** awaiting final integration and visual review. **Not yet:** the
full release gate (it needs its own approval), and textures (M-ART-2).

## What changed

- **G, the picture's size:** the largest square the view holds, no longer
  capped at 360px.
- **H, its look is a window setting:** Enhanced or Classic, and Close,
  Standard or Wide, are kept in `gui-settings.json`. They are chosen in the
  view's upper left corner and in Settings.
- **I, transitions:** every way into and out of an encounter is checked in
  Go. The M-ART-1A.2 observation is explained and reproduced.
- **The resume notice** (a correction after review): EXPEDITION RESUMED is
  shown over the view for a few seconds, no longer as a banner that took
  the picture's height (see [The resume notice](#the-resume-notice)).

The pipeline is unchanged: `Sight()` → `View.Dungeon.Corridor` →
`mazeScene.js` → `mazeProjection.js` → `MazeView.svelte`. Go's production
changes are the settings field pair and `Session.SetMazeLook`
(`gui/settings.go`). Visibility, maze data, combat, movement, exploration
and the save format are unchanged.

## The picture's size

**The rule:** the view (`.stage`) is a size container. The picture is the
largest square that fits its height, its width less a 184px column on each
side for what sits in its corners, and at most 720px.
- **The corners:** the style and zoom upper left, the minimap upper right.
  Neither ever covers the picture.
- **The commands:** the movement pad and the other commands share one line
  where they fit (66px tall instead of 111px).
- **Padding:** the view's padding is 6px.

Measured in the running app:

| Window | Before | After |
|---|---|---|
| 980×692 | 234px | **345px** |
| 1440×900 | 360px | **553px** |
| 1920×1200 | 360px | **720px** (the cap) |

- **No page scrolling at any size.** The header, the messages, the commands
  and the party bar are where they were. Text is not smaller.
- **The minimap** is unchanged: 168px, north up, upper right, clear of the
  picture at every size (checked live). Its exploration rules are untouched,
  and F10 is as before.
- **The resume notice** takes none of it now (below). The scene shots
  further down were taken before that correction, after the app resumed,
  so they show the old banner and the picture 59px smaller at 980×692
  (286px) and 1440×900 (494px).

## The look: a window setting

**Where it is kept:** `gui-settings.json` gains `mazeStyle` (`enhanced`,
`classic`) and `mazeZoom` (`standard`, `close`, `wide`).
- **Omitted while they are the defaults,** Enhanced and Standard.
- **Earlier files still load:** a missing or unknown value is read as the
  default.
- **Not a game's:** they are the library's, the same in every game, never in
  a game or its save.

**`Session.SetMazeLook(style, zoom)`** sets and saves them.
- **It refuses** a value it does not know.
- **It touches nothing of the game:** no step, turn, light, save or random
  number, in the maze or in combat (tested in Go).

**Two controls, one setting.** Both call `SetMazeLook` and show what
`view.settings` says, so they always agree.
- **The view's upper left corner:** Enhanced / Classic, and Close / Standard
  / Wide.
- **Settings:** a Maze view section, with or without a game open.

**Keyboard focus.** A click lets go of the focus, so the arrow keys move the
party at once. A key press on a control keeps the focus there.

**Why the view's corner, not the header.** At 980×692 the header has 31px
free. The controls would wrap it onto a second line and take about 36px
from the picture at the smallest window. The corner is outside the picture
at every supported size, opposite the minimap. The 1A.2 bottom-left
overlay is gone (`mazeLook.svelte.js` removed).

**Bindings:** `SetMazeLook` and the two `SettingsView` fields only. The
generator's unrelated Story types, from CONTENT-1A, were left out again.

## Phase transitions: the M-ART-1A.2 observation

**When the picture should be there.**
- **The maze phase:** the dungeon screen draws it.
- **Camp:** Camp's own screen.
- **Combat:** its own screen, including the chest, victory and defeat.

**What Go shows (`gui/maze_transitions_test.go`).** Every way into and out
of an encounter, driven through the Session methods the window calls:
- a step onto a fight square, real and practice, and an encounter guarding a
  chest;
- escape, and victory with the chest opened or left;
- defeat, back to the Castle's roster;
- a restart in the middle of an encounter, and after it.

At every step, the dungeon screen appears only in the maze or Camp, and in
the maze its corridor is the engine's `Sight()` at that moment.
- **The exploration:** 240 encounters over 40 seeds, about 25,000 views,
  and never another phase.
- **The test kept:** 4 seeds that reach every outcome, about 16s.

**What happened in M-ART-1A.2.** That run's driver played the fight by
calling `window.go.main.Session` from outside the page. The page learns a
new view only from its own calls, so it stayed on the combat screen while
Go was back in the maze.

**Reproduced here (`fight-a-raw`):**
1. A step by the keyboard enters the fight.
2. The page waits at the orders.
3. The fight is played by calls from outside.
4. Go says `dungeon/maze`, and the page still shows combat with no picture.
5. The page's next call of its own (the stale screen's Party retreat) is
   refused by Go ("no combat order entry is in progress"). The page then
   shows Go's view: the maze, in the look chosen.

A player cannot reach this: every input goes through the page, and the save
folder's lock keeps out a second window. So this is a test-harness effect,
not a defect, and nothing was changed for it.

**A player's fights (`fight-b`, `fight-c`).** Each was walked into by the
keyboard and played through the combat screen's own buttons (Party retreat,
Fight!, Continue). Both came straight back to the maze, the picture exactly
the modules' for Go's view, in the look chosen (Classic at Wide; Enhanced at
Close). A turn afterwards was redrawn.

**Recorded for the frontend.** `testdata/maze-transitions.json`, by Go,
records one real expedition: into an encounter and out by escape; into one
guarding a chest, won and opened; a restart; Camp and back. The look was
Classic at Close throughout. `mazeTransitions.test.js` server-renders the
real `MazeView` for every maze view in it:
- each is exactly the projection of that moment's corridor, in the look kept;
- no picture is stale: different corridors are drawn differently, and the
  same corridor the same, whatever came between;
- the style and zoom change the paint and the frame only.

A deliberately stale `MazeView` (one keeping its first picture) fails two of
these tests.

## The resume notice

**Before:** Go's notice of a resumed expedition (headed EXPEDITION RESUMED,
`View.Title`) was one of the window's banners. Go keeps it in every view
until another notice replaces it, so it stayed for the whole expedition
and took 71px of the window's height (59px of the picture at 980×692).

**Now:** the page shows it over the top of the maze view
(`lib/resumeNotice.js`, `DungeonScreen.svelte`), and never in the banners.
Every other notice is a banner as before. Go is unchanged.
- **Where:** centred at the top of the view, between its corners: clear of
  the style and zoom and of the minimap at every size. At Camp, over the
  top of its screen. It takes no room and no pointer input.
- **Its text:** the whole notice, under its heading, in the banners' blue.
- **How long:** four seconds, then a 250ms fade. With reduced motion, no
  fade: gone at four seconds.
- **Sooner:** the player's next successful action in the dungeon (a step,
  a turn, a kick, Camp: `DungeonAction`, `CampAction`) or a change of the
  maze's look (`SetMazeLook`), and leaving the dungeon screen (an
  encounter, the Castle, another game). A refused action leaves it.
- **Once:** once for each resume; never again on the later views that
  still carry it, and again after the next resume.
- **Only the page's timers:** Go, the Session, the save and the settings
  are not involved. The timer is cleared when it is replaced or the page
  closes, and one from an earlier showing does nothing.

**Measured in the running app** (the `resume` live suite, a real
expedition, Istary closed and started again):

| Window | Picture before the restart | With the notice | After it is gone |
|---|---|---|---|
| 980×692 | 345px | 345px | 345px |
| 1440×900 | 553px | 553px | 553px |

- **Gone by itself** 4.26s (980×692) and 4.23s (1440×900) after it was
  first seen; with reduced motion 3.98s and 4.00s, never fading.
- **A key turning the party** dismissed it 29ms after the key at 1440×900.
- **No page scrolling;** the notice clear of the minimap and the control.

| Files (`resume/`, at 980×692 and 1440×900) | What |
|---|---|
| `resume-shown-enhanced-*.png` | just after the restart, Enhanced |
| `resume-dismissed-auto-enhanced-*.png` | after four seconds, gone by itself |
| `resume-shown-classic-*.png` | just after another restart, Classic |
| `resume-dismissed-by-turn-classic-*.png` | a key turned the party: gone at once |

## Screenshots

Taken from the running app (`wails dev`, headless Chrome, 2× pixels) on
scratch libraries only. Real saves were checksummed before and after.

**Positions.** The ten positions of M-ART-1A.1 and 1A.2: a copy of their
scratch library, with only the save's `expedition.where` changed per
position, and the app restarted at each.

**Styles and zooms.**
- **Every position:** Classic and Enhanced at Standard, at 980×692 and
  1440×900.
- **The straight corridor, the crossing and the door:** also Close and Wide.
- **The straight corridor and the door:** also at 1920×1200.

**Checked at every shot:**
- the polygons and detail lines drawn are exactly the shared modules' output
  for the view Go sent;
- the buttons pressed are the look kept in the file;
- the focus is with the maze;
- the picture is square, centred, clear of the minimap and the control,
  with no scrolling.

**Across restarts.** The look survived every restart: the first session
opened an earlier settings file at the defaults, and each later one opened
with the look chosen before.

**Afterwards, at each position:**
- the clicks made `SetMazeLook` calls alone, and the save was unchanged;
- a key turned the party, and the picture was redrawn.

| Files | What |
|---|---|
| `<scene>-<style>-<zoom>-<size>.png` | the scenes (as 1A.2's table: straight lit/unlit, dead end, T-junction, four-way, door, hidden door unlit/lit, dark ahead, standing in darkness) |
| `settings-*.png`, `maze-from-settings-*.png` | Settings: defaults from an earlier file; Classic and Wide chosen there; after a restart, the maze in that look; Settings after Enhanced and Close were chosen in the maze |
| `fight-*-combat-*.png`, `fight-*-after-*.png` | into an encounter and back (b, c); the reproduction (a) |
| `sheets/before-after-*.png` | the same positions before (1A.2's screenshots, `835e082`) and after |
| `sheets/compare-*.png` | Classic and Enhanced, every scene (crops) |
| `sheets/zoom-*.png` | Close, Standard, Wide (crops) |
| `sheets/screen-sizes.png` | the whole screen at three sizes: header, control, minimap, commands, party |
| `sheets/settings.png`, `sheets/after-combat.png` | Settings and the transitions |

`shots.json` holds, for each scene shot, the corridor Go sent and what was
drawn, with the layout's boxes.

## Tests

- **Go:**
  - `TestMazeLookIsKept`, `TestMazeLookFromOtherFiles`,
    `TestMazeLookTouchesNothingOfTheGame`, `TestMazeLookAcrossGames`;
  - `TestMazePictureAcrossTransitions`, `TestMazeTransitionFixture`;
  - `frame-views.json` gains the two default fields in its 17 views.
- **Frontend:**
  - **the sizing rules:** a size container, the corners, no 360px cap, the
    commands on one line;
  - **the controls:** read from the settings view, one call, focus;
  - **the reader:** `mazeLookOf`'s fallbacks;
  - **transitions:** `mazeTransitions.test.js`;
  - **unchanged:** all of 1A.1's and 1A.2's.
- **Live** (`live.mjs`, nav suite, at both sizes):
  - **the picture's size:** at least 280px or 480px, square, centred, clear
    of the corners, no scrolling;
  - **the controls:** `SetMazeLook` alone, kept, the party where it was,
    the save unchanged, a key still turns the party;
  - **Settings:** it shows, and chooses, the same look.
- **The resume notice:**
  - `resumeNotice.test.js`, on a fake clock: shown on a resume and still
    before four seconds; fading, then gone; reduced motion; dismissed by a
    successful action, not by a refused one; once for each resume; ended
    by leaving the dungeon screen; no stale timer, none left when the page
    closes; no Session call; over the view, no room, no input;
  - the `resume` live suite (both sizes), as measured above.

## Observations and limitations

- **The picture is the screen's focus now.** The masonry, flagstones and
  door planks read clearly at 494px. At 286px they are legible, and fine
  lines at depth 3 are faint.
- **The resume notice** is shown for four seconds. A long one (an
  encounter lost to the closing, stranded adventurers) runs to three or
  four lines over the top of the picture, and may be gone before it is
  read; it is not kept anywhere else in the window.
- **Settings at 980×692:** the Maze view section is below Appearance, inside
  the information panel's own scroll, as Settings already was. The page
  does not scroll.
- **Classic** at Standard is 1A.1's picture, larger.
- **The player's fights** were escapes. Victory, chests and defeat through
  the screen's buttons are covered by Go's transition tests and the live
  combat suite (practice encounters), not by a walked fight here.
- **Size:** this directory is about 22MB of lossless screenshots.
