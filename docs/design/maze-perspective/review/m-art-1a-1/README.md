# M-ART-1A.1: the geometry contract and the Classic renderer

**Branch:** `m-art-1a-maze-perspective`, on v0.20.0-castle-artwork
(`3667441`). **Status:** awaiting review. Classic is the only style;
Enhanced, zoom and Settings controls come in M-ART-1A.2 and 1A.3.

## What changed

The dungeon screen's picture is now `MazeView.svelte`. It is drawn from
`view.dungeon` in three pure steps:

1. **The scene** (`lib/mazeScene.js`):
   - **What it reads:** only `phase`, `corridor`, `dark` and `spells`.
   - **What it gives:** the surfaces seen, relative to the party, plus the
     openings and what ends the view.
   - **What it never reads:** the maps, the position or the exploration
     record.
2. **The projection** (`lib/mazeProjection.js`): the surfaces as polygons
   in DRAWMAZE's stepped geometry, in painting order. It is the same for
   every style.
3. **The style** (`lib/mazeStyle.js`): which kinds are painted, and how.
   Classic paints walls, doors and the void, and leaves floors and ceilings
   unpainted.

How far the party sees, darkness and hidden doors remain the engine's
`Sight()` (`engine/sight.go`). Go's production code, the view schema,
gameplay, saves and maze data are unchanged.

## Screenshots

**How they were taken:**
- **App and library:** `wails dev` on a scratch library written by
  `TestWriteAcceptanceLibrary`.
- **The expedition:** the app itself started a real expedition through its
  Session methods (Enter the Maze, leave Camp, decline the stairs) and saved
  it.
- **Moving between positions:** for each position, only the scratch save's
  `expedition.where` was changed (`x`, `y`, `facing`, `light`, with the
  message fields cleared). The app was then started again, resumed the
  expedition there, and was photographed. That is why each shot shows
  "Expedition resumed".
- **Capture:** headless Chrome, at exactly the size named, at 2× device
  pixels.
- **Where the positions come from:** Wiz 1's own floor 1, read-only. Real
  saves were never touched; their checksums were the same before and after.

**Checked for every shot:** the polygons drawn (`data-surface`) were exactly
the Classic paint of the projection of the corridor the app sent.
`shots.json` holds each one's corridor, as Go sent it, and the surfaces
drawn.

| Shot (`-980x692.png`, `-1440x900.png`) | Place, facing | Light | Squares seen | Expected geometry |
|---|---|---|---|---|
| `straight-lit` | L1:R18:C0, N | yes | 5 | walls on both sides at every depth; the view ends unseen (the void) past depth 4 |
| `straight-unlit` | L1:R18:C0, N | no | 2 | the same corridor, two squares, then the void |
| `dead-end` | L1:R15:C18, N | yes | 3 | side walls; a front wall at depth 2 |
| `t-junction` | L1:R12:C3, N | yes | 2 | both sides open at depth 1, each showing the wall ahead of the side square; a front wall at depth 1 |
| `four-way` | L1:R9:C8, E | yes | 3 | both sides open at depth 1 with passages going on (nothing drawn there); depth 2: left open with a wall ahead, right open; a front wall at depth 2 |
| `door` | L1:R13:C3, S | yes | 3 | side walls; a door in the front wall at depth 2, in DRAWMAZE's proportions |
| `dark-ahead` | L1:R9:C9, N | yes | 3 | an opening on the left at depth 0 (wall ahead), right openings at depths 0–2 (a wall ahead at 2), left walls at 1–2; a dark square at depth 3: nothing past it |
| `standing-in-dark` | L1:R19:C10, N | yes (staged) | 0 | nothing but the void, and the screen's "darkness" |
| `hidden-door-unlit` | L1:R17:C12, N | no | 1 | a hidden door ahead, drawn as a plain wall |
| `hidden-door-lit` | L1:R17:C12, N | yes | 1 | the same hidden door, drawn as a door |

**Observations:**
- **The picture's size:** 234 px square at 980×692 and 360 px (the cap) at
  1440×900, the same as before.
- **Unchanged:** the minimap, the header, the commands and the party bar.
- **A dark square ahead** ends the view in a void, which looks the same as
  the background. Classic gives no other sign of it, as in DRAWMAZE.
  "darkness" is written only when nothing at all is seen.

**Staged, not reachable in play:** `standing-in-dark` shows LIGHT. In
play, stepping onto a dark square puts the light out and writes "IT'S VERY
DARK HERE". Here the party was placed there with its light still on, and
with the message cleared. The picture is the same either way, because
standing in darkness nothing is seen with or without light.

## The fixtures

`gui/maze_scene_fixture_test.go` builds each case as a synthetic 20×20
level, in memory, in a copy of the practice game's scenario. It records the
dungeon view the GUI receives to
`gui/frontend/src/lib/testdata/maze-scenes.json`.

**Checked in Go for every case:**
- the corridor is `Sight()`, square for square;
- the number of squares seen, and what ends them;
- that taking the view changes nothing;
- that recording twice gives the same bytes.

| ID | Case | Squares | Ends |
|---|---|---|---|
| G01 | straight corridor | 5 | limit |
| G02 | dead end | 3 | wall |
| G03 | left opening (side square's walls solid) | 5 | limit |
| G04 | right opening onto a passage that goes on | 5 | limit |
| G05 | corner turning right | 2 | wall |
| G06 | T-junction | 2 | wall |
| G07 | four-way crossing | 4 | wall |
| G08 | door ahead | 3 | door |
| G09 | door on the left | 4 | wall |
| G10 | doors right (d0), left (d1), ahead of a left opening (d2), front (d3) | 4 | door |
| G11-N/E/S/W | one square seen facing each way | 1–2 | wall/door |
| G12-edge / -inside | a corridor across the east edge, and the same inside | 5 | wall |
| V01 / V02 / V03 | no light / light / QUICK plot with light | 2 / 5 / 3 | limit |
| V04 / V05 | a wall / a door ahead, with a corridor and a door behind it | 2 | wall / door |
| V06 / V06-wall | a hidden door without light / a plain wall | 2 | wall (identical views) |
| V07 | a hidden door with light | 2 | door |
| V08 | a dark square two ahead, a door past it | 2 | dark |
| V09 | standing in a dark square | 0 | dark |
| V10 | the corridor goes on past the light, to a door | 5 | limit |
| V11 | looked down with light (the exploration record holds 5 squares), now without | 2 | limit |

## Tests

Run with `node --test`:

- **`mazeScene.test.js`:**
  - every recorded case's depth, end, walls, doors and openings are the
    corridor's;
  - the cases named above, one by one;
  - the scene reads nothing but the four fields (proven by watching every
    field read);
  - other parts of the view change nothing;
  - the scene is deterministic and doesn't change its input (the input is
    frozen);
  - the modules import nothing of the app.
- **`mazeProjection.test.js`:**
  - **DRAWMAZE's numbers:** the front wall, the side wall and both doors at
    depths 0–3, from its own integer recurrence;
  - **identifiers and order:** stable ids, painting order, and nothing but
    geometry in each primitive;
  - **the void and occlusion:** the void exactly where the view ends unseen;
    nearer surfaces cover farther ones in all 1,875 two-square corridors,
    1,500 random corridors and every case;
  - **symmetry, determinism and Classic:** left/right symmetry,
    determinism, and Classic painting only the projection's walls, doors and
    void, unchanged.
- **`mazeView.test.js`:** `MazeView` draws only that. It takes only
  `dungeon`, has no input or handlers, and imports nothing of the app. The
  dungeon screen gives it `view.dungeon` and draws no polygons of its own.
- **Unchanged and passing:** `dungeon.test.js` and `dungeonViews.test.js`
  now run against the new geometry through `dungeon.js`'s `scene` wrapper.

## Differences from DRAWMAZE

- **No hidden-door flicker.** DRAWMAZE draws a hidden door one time in six
  without light. Nothing that only looks may draw a random number, so this
  is omitted (`engine/sight.go`, WIZARDRY-COMPATIBILITY #23).
- **Depth 4 doors are not rounded.** DRAWMAZE's integer halving makes
  DOORFRAM 0 there. The projection uses exact quarters, so it matches
  DRAWMAZE exactly at depths 0–3.
- **Filled walls.** Walls are filled, in a near-black slightly lighter
  than the background (as the old placeholder was), rather than clipped as DRAWMAZE clips them (XLOWER/XUPPER). The result is
  the same (tested), and the fill keeps the existing look.
- **The picture is smooth vector,** not 82×79 pixels.
