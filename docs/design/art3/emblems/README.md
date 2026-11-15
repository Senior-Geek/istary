# ART-3B-0: location emblems (candidates for review)

**Baseline:** v0.15.0-markdown-stories (`d04942b`). **Branch:**
`art-3b-location-emblems`. **Status:** candidates, awaiting visual
approval. Nothing here is used by Istary yet: `scenes.js` still names the
Unicode emblems, and no Svelte, CSS or story file changed.

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
| Boltac's Trading Post | `emblem-trading-crossed-swords.svg` | Two swords crossed, guards and pommels | Boltac's sign |
| The Temple of Cant | `emblem-temple-star-arch.svg` | The eight-pointed star, filled, inside a Gothic (pointed) arch | The Temple's sign and its window |
| The Training Grounds | `emblem-training-target.svg` | Archery target on a stand, an arrow in the gold | The targets in `edge.webp` |
| The maze | `emblem-maze-stone-archway.svg` | Rough stone arch of uneven blocks, a dark opening, steps going down | New: the maze's entrance, older than the castle |

`emblems-design-sheet.png` shows each file enlarged on an 8-unit grid,
at 40 and 32 px on navy, on the app's dark panel and on a light neutral
background (in the file's gold and in the Light appearance's), and beside
a 20 px title. It was drawn from the files themselves, by Chrome, at 2×
device pixels.

## The system

- `viewBox="0 0 64 64"`, the drawing inside about 4 to 60.
- Outlines stroke 3 (1.5 to 1.9 px at 32 to 40 px), details stroke 2,
  round caps and joins; solid fill only for small marks that would
  vanish as lines (the Temple's star, the target's gold, the arrow's
  fletching) and the maze's dark opening (`#120f0a`).
- Colour: `currentColor`, set to the app's gold `#d2b06a` on the root
  (`color=`). As an `<img>` the files are that gold; inlined, CSS `color`
  overrides it (the Light appearance's gold is `#7a5a14`).
- Transparent; no text, fonts, images, scripts or links. Each file has a
  `<title>` naming its place and design.
- Places are told apart by shape, not colour: the gatehouse is
  crenellated, symmetrical and barred; the maze is rough, unbattlemented,
  dark and stepped. The Temple's star is set in an arch, unlike the
  banners' bare star. The target is round; the swords are diagonal.
