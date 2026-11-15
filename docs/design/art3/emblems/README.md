# ART-3B-0: location emblems

**Baseline:** v0.15.0-markdown-stories (`d04942b`). **Branch:**
`art-3b-location-emblems`. **Status:** approved without revisions
(`e8ef4ae`), and in use (Stage 2, below). Amended in Stage 2: the
Training Grounds have the crossed swords, and Boltac's a merchant's
balance with a coin.

One emblem for each place, beside its title, in place of the Unicode
symbols. They are redrawn after the painted signs in the Castle and Edge
pictures (ART-3A): those icons are about 40 px in the 1920 px pictures,
soft and on a board, so they cannot be cut out at Retina sizes.

| Place | File | Design | From |
|---|---|---|---|
| The Castle | `emblem-castle-tower.svg` | Tower, three merlons, door and slit | The Keep's sign |
| The Edge of Town | `emblem-edge-gatehouse.svg` | Two towers, a crenellated wall, an arch with its portcullis | The gatehouse in `edge.webp` |
| Gilgamesh's Tavern | `emblem-tavern-tankard.svg` | Tankard with a head of foam and a handle | The Tavern's sign |
| The Adventurer's Inn | `emblem-inn-bed.svg` | Bed: head- and footboard, pillow, turned-down cover | The Inn's sign |
| Boltac's Trading Post | `emblem-trading-balance-scale.svg` | A merchant's balance on a stand, a coin in one pan | New (Stage 2 amendment) |
| The Temple of Cant | `emblem-temple-star-arch.svg` | The eight-pointed star, filled, inside a Gothic (pointed) arch | The Temple's sign and its window |
| The Training Grounds | `emblem-training-crossed-swords.svg` | Two swords crossed, guards and pommels (the approved drawing, unchanged) | Boltac's sign in the pictures |
| The maze | `emblem-maze-stone-archway.svg` | Rough stone arch of uneven blocks, a dark opening, steps going down | New: the maze's entrance, older than the castle |

`emblem-training-target.svg`, the archery target first approved for the
Training Grounds, is kept as a design reference and is not used.

`emblems-design-sheet.png` shows each file enlarged on an 8-unit grid,
at 40 and 32 px on navy, on the app's dark panel and on a light neutral
background (in the file's gold and in the Light appearance's), and beside
a 20 px title. It was drawn from the files themselves, by Chrome, at 2×
device pixels.

## The system

- `viewBox="0 0 64 64"`, the drawing inside about 4 to 60.
- Outlines stroke 3 (1.5 to 1.9 px at 32 to 40 px), details stroke 2,
  round caps and joins; solid fill only for small marks that would
  vanish as lines (the Temple's star, the balance's finial and coin) and
  the maze's dark opening (`#120f0a`).
- Colour: `currentColor`, set to the app's gold `#d2b06a` on the root
  (`color=`). As an `<img>` the files are that gold; inlined, CSS `color`
  overrides it (the Light appearance's gold is `#7a5a14`).
- Transparent; no text, fonts, images, scripts or links. Each file has a
  `<title>` naming its place and design.
- Places are told apart by shape, not colour: the gatehouse is
  crenellated, symmetrical and barred; the maze is rough, unbattlemented,
  dark and stepped. The Temple's star is set in an arch, unlike the
  banners' bare star. The balance is upright and symmetrical, with
  hanging pans; the swords are two diagonals.

## In Istary (Stage 2)

- **The drawings:** `gui/frontend/src/lib/emblems.js` holds each file's
  shapes as data, drawn inline by `Emblem.svelte` (Svelte elements, no
  `{@html}`), in `currentColor`. `emblems.test.js` checks the data is
  these files, shape for shape, so these files stay the source: change
  one, then `emblems.js` (the test plan runs the frontend tests for them,
  `docs/TESTING.md`).
- **Where:** before the place's title in the frame's head
  (`LocationFrame.svelte`), 34 px, centred on it, and before "The maze"
  in the maze (not at Camp; `DungeonScreen.svelte`). They are
  `aria-hidden`: the heading says only the title. And before each
  destination's name in its button (ART-3B: the Castle's Go to list, the
  Edge's Training Grounds), 22 px, also `aria-hidden`; the button keeps
  its height.
- **Colour:** the appearance's `--gold`: `#d2b06a`, the darker `#7a5a14`
  on the Light panels, `#e0bd6a` on Slate.
- **The themed panels** of the places without a picture show the emblem,
  in the place's tint, in place of the Unicode symbol. Settings keeps its
  symbol (⚙), and so does a place with no entry (✦).

Two layout preparations for the interiors (ART-3A), in the frame:

- **The information keeps its room.** An illustrated place's information
  was capped at 45% of the body. It now has what a place without a
  picture gives it, all but the display's and actions' `--top`, and
  scrolls past that; short information still leaves the picture its
  room. The Tavern's roster, Boltac's goods and character creation keep
  their space when they have pictures.
- **A shown character sheet has priority over the story box.** The box
  is not drawn while a sheet is shown in the display, and comes back
  where it was when the sheet is hidden; Story, with a sheet shown,
  hides the sheet to show the story.
