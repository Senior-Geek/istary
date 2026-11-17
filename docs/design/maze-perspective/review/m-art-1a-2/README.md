# M-ART-1A.2: the Enhanced style and three zooms

**Branch:** `m-art-1a-maze-perspective`, on M-ART-1A.1 (`810ebd7`).
**Status:** awaiting review. **Not yet:** Settings, the header's controls and
a larger picture (M-ART-1A.3); light sources and effects (M-ART-2).

## What changed

One maze, one geometry, two styles. The pipeline is M-ART-1A.1's, unchanged:
`Sight()` → `View.Dungeon.Corridor` → `mazeScene.js` → `mazeProjection.js`
→ `MazeView.svelte`. Only the style (`mazeStyle.js`) and the frame are new.

- **Enhanced** paints every primitive the projection gives as stone, floors
  and ceilings included. Classic still paints only walls, doors and the
  void. Both styles paint those at the same coordinates, in the same order.
- **Zoom** is the picture's viewBox, nothing else: Close, Standard, Wide.
- **A review control** in the view's lower left corner chooses the style and
  zoom. Enhanced at Standard is where it starts.

Go's production code, the view schema, gameplay, saves, Settings and maze
data are unchanged.

## Enhanced

Every part is dressed from its own primitive alone (and, for the fade, from
where the view ends: the void's depth). Nothing reads the maze, the map or
the position.

- **Fog:** a surface's colour at plane `t` (the near plane of square `d` is
  `t = d`, its far plane `d + 1`) is `0.66^t` of the way from the background
  (`#0d0c0b`) to its stone. Side walls, floors and ceilings recede, so each
  is a gradient from its near plane's shade to its far plane's, and the next
  square starts where it ends: no steps between squares.
- **The end of sight:** where the view ends unseen (the sight's limit, or a
  dark square), the last square also fades towards the void, to 45% of its
  colour at the void's plane. No wall or door fades away (tested: each stays
  apart from the darkness).
- **Stone:** front walls `#70685e` at full light, the left side a shade
  lighter than the right, so two planes read as two; floors a browner
  `#544a3e`; ceilings darkest `#2e2b28`. Walls have a faint worn edge.
- **Masonry:** four courses up each wall, stones half a square long, joints
  staggered course by course. On a side wall the courses run to the
  vanishing point and the joints are placed in true depth (halfway along a
  square is two thirds of the way across it on the picture). Courses meet
  from square to square. Joints are drawn to depth 2, courses to depth 3,
  nothing beyond.
- **Floors and ceilings:** the projection's own polygons (the square's, and
  the side square's through an opening). Flagstones half a square each way,
  staggered, to depth 3; the ceiling only a joint every half square, to
  depth 2. Floors and ceilings are outlined in their own colour, so no
  hairline shows between neighbours.
- **Doors:** one door for every door: dark wood in a heavy stone frame, three
  plank lines and two iron bands, to depth 3. Go says only "door", so
  nothing suggests locked, trapped, open or secret. A hidden door Go reports
  as a wall is a wall, dressed as every wall is (V06 and V06-wall: the same
  output, byte for byte).
- **Details never leave their surface:** every line's ends are interpolated
  between its own polygon's corners, so it lies inside that polygon, drawn
  straight after it. Nearer surfaces cover it as they cover the surface.

## Zoom

| Preset | viewBox | |
|---|---|---|
| Close | `-36 -36 72 72` | **revised** from `-30 -30 60 60` |
| Standard | `-40 -40 80 80` | Classic's, DRAWMAZE's picture |
| Wide | `-52 -52 104 104` | as planned |

- **Why Close changed:** the party's own square's side walls (or the openings
  beside it) lie at `|x|` 32 to 64. `-30 -30 60 60` shows none of that: a
  wall or an opening right beside the party would not be seen. `-36` keeps
  a 4-unit strip of each side.
- **The rules, tested for every frame:**
  - it frames more than `|x| = 32` (the party's square's far plane), so the
    corridor past it is always whole;
  - it keeps at least 4 units of the party's square's sides;
  - it frames less than `|x| = 64` (that square's near plane), so nothing
    beside or behind the party is shown.
- **What Wide shows that Standard does not:** more of the same surfaces. A
  door in the wall right beside the party (jambs at `|x|` 40 to 56) is
  outside DRAWMAZE's picture and Standard's; at Wide its near part shows.
  Where a side opening leads on with nothing seen, Wide shows more of the
  same darkness.

## Screenshots

**How they were taken** (as for M-ART-1A.1):
- **App and library:** `wails dev` on a scratch library, never the real
  saves (checksums of all 14 real files identical before and after).
- **Positions:** the expedition the app itself started in M-ART-1A.1's
  scratch library (a copy). For each position only the scratch save's
  `expedition.where` was changed (`x`, `y`, `facing`, `light`, messages
  cleared); the app was started again and resumed there. That is why each
  shot shows "Expedition resumed".
- **Style and zoom:** chosen by clicking the control in the running app
  (real mouse events), at 980×692, then captured at 980×692 and 1440×900
  (headless Chrome, 2× device pixels). The positions are M-ART-1A.1's, so
  its Classic shots and these line up.
- **Checked for every shot:** the polygons drawn are exactly
  `painted(perspective(view.dungeon), style)` for the view Go sent, and the
  detail lines exactly its details, computed in the page by the same
  modules; the viewBox is the zoom chosen; the buttons pressed are the
  ones chosen; the focus is back on the window. `shots.json` holds each
  one's corridor, as Go sent it, and what was drawn.

**Files:** `<shot>-<style>-<zoom>-<size>.png`, e.g.
`door-enhanced-wide-1440x900.png`. **Sheets** (`sheets/`): crops of those
screenshots around the picture, side by side, each named under its crop;
nothing is redrawn.
- `compare-980x692.png`, `compare-1440x900.png`: Classic and Enhanced at
  Standard, every position.
- `zoom-980x692.png`, `zoom-1440x900.png`: Enhanced at Close, Standard and
  Wide: the straight corridor, the crossing and the door.

| Shot | Place, facing | Light | Styles and zooms | Expected geometry |
|---|---|---|---|---|
| `straight-lit` | L1:R18:C0, N | yes | Classic, Enhanced; Enhanced Close, Wide | walls both sides at every depth; the corridor runs into darkness past depth 4 |
| `straight-unlit` | L1:R18:C0, N | no | Classic, Enhanced | the same corridor, two squares, then darkness |
| `dead-end` | L1:R15:C18, N | yes | Classic, Enhanced | side walls; a front wall at depth 2 |
| `t-junction` | L1:R12:C3, N | yes | Classic, Enhanced | both sides open at depth 1, each with the wall ahead of the side square; a front wall at depth 1 |
| `four-way` | L1:R9:C8, E | yes | Classic, Enhanced; Enhanced Close, Wide | both sides open at depth 1 onto passages that go on (darkness there); depth 2: left open with a wall ahead, right open; a front wall at depth 2 |
| `door` | L1:R13:C3, S | yes | Classic, Enhanced; Enhanced Close, Wide | side walls; a door in the front wall at depth 2 |
| `dark-ahead` | L1:R9:C9, N | yes | Classic, Enhanced | left opening at depth 0 (wall ahead), left walls at 1–2, right openings at 0–2 (a wall ahead at 2); a dark square at depth 3: nothing past it |
| `standing-in-dark` | L1:R19:C10, N | yes (staged) | Classic, Enhanced | nothing but darkness, and the screen's "darkness" |
| `hidden-door-unlit` | L1:R17:C12, N | no | Classic, Enhanced | a hidden door ahead, drawn as a plain wall |
| `hidden-door-lit` | L1:R17:C12, N | yes | Classic, Enhanced | the same hidden door, drawn as a door |

## Tests

- **`mazeEnhanced.test.js`** (new), over the 30 recorded cases and 600
  seeded random corridors:
  - **one geometry:** both styles draw the same walls, doors and void at
    the same coordinates in the same order; Enhanced draws every primitive
    the projection gives, unmoved, and nothing else;
  - **no extra maze:** every detail point lies on its own surface; nothing
    is drawn at or past where the view ends but the void; past a dark
    square, nothing;
  - **openings stay open:** at points across each opening, only what is
    seen through it (that side's square, the wall ahead of it) is drawn;
  - **occlusion,** floors and ceilings included, over the Wide frame;
  - **hidden doors:** V06 and V06-wall identical in Enhanced; V07 a door;
  - **shading:** darker with distance, continuous between squares, the
    sides a shade apart, the ceiling darker than the floor, the fade
    confined to the last square; no wall or door fades into the darkness;
  - **floors and ceilings:** one for every square seen and every opening,
    sharing their edges with the walls and the next square;
  - **masonry:** in perspective (a joint halfway along a square two thirds
    across it), meeting from square to square, thinning with depth;
  - **doors:** the same door wherever seen, decided by its place alone;
    darker than its wall, with a lighter frame;
  - **no side effects:** frozen inputs, the scene and projection unchanged,
    deterministic;
  - **zoom:** the presets' framing rules; the corridor past the party's
    square whole in every frame; every door past it in view to its middle.
- **`mazeView.test.js`:** MazeView imports only the projection and the
  style, uses the zoom for the viewBox alone, keeps no state and takes no
  input; the control calls no Session method, saves nothing and blurs.
- **Live** (`scripts/acceptance/live.mjs`, the nav suite's practice
  expedition): Enhanced at Standard to start, drawn as the modules give;
  Classic and Wide chosen by clicks with no Session call and no setting
  changed; a key afterwards still turns the party, and the picture follows.
- **The review run** (this directory): 155 checks, 0 failed. Besides those
  above, at every position: the control clear of the picture at both sizes,
  no Session call and nothing saved while choosing, ArrowRight and ArrowLeft
  turning the party with the picture redrawn, a step forward redrawn, and
  no frontend errors (Wails' own "reading 'nodes'" outside its window
  excluded, as `live.mjs` does).
- **After combat** (scratch expedition, walked into fight-zone squares by
  the keyboard, with Classic at Wide chosen): twice the party fought, fled
  and was back in the maze with Classic at Wide kept, the picture exactly
  the modules' for the view Go sent, and a turn redrawn. In a third fight
  the check found the dungeon screen without its picture at that moment
  and recorded nothing more; the screen shows the picture only in the maze
  phase, and this was not seen again in the two later fights.
- **Walking:** 150 keyboard actions on the scratch expedition, the picture
  checked after each: no difference.

## Observations and limitations

- **The size:** 234 px at 980×692 and 360 px (the cap) at 1440×900, as
  before; the dungeon screen's layout is unchanged.
- **The control** sits in the view's lower left corner (x 55–219 at
  980×692), clear of the picture and of the minimap, without moving
  anything. It is temporary: Settings and the header's controls replace it
  in M-ART-1A.3.
- **Depth reads at a glance:** the receding courses and flagstones and the
  fog make distance clear in a straight corridor, and side openings read as
  passages (the floor carries on into darkness).
- **Doors** stand out from stone by colour and frame at every depth to 3; at
  depth 4 a door is its frame and fill alone.
- **The end of sight:** where the view ends unseen it ends in a dark panel
  the stage's colour, with the last square fading into it. Unlit (two
  squares) that panel is large: it reads as darkness ahead rather than a
  wall, but more strongly than in Classic, where everything is that colour.
- **A dark square ahead** looks the same as the end of sight, as in Classic
  and DRAWMAZE: nothing tells them apart.
- **Close** is modest (1.11× Standard), because of the framing rule above.
- **Classic** at Standard is M-ART-1A.1's picture, unchanged.

**Limitations:**
- **Staged darkness:** `standing-in-dark` shows LIGHT, which play cannot
  (stepping onto a dark square puts the light out). The picture is the same
  either way.
- **The banner:** every shot shows "Expedition resumed", because of how
  they were taken.
- **Not reachable here:** QUICK plot cannot be turned on in the GUI (the
  fixtures cover it, V03).
- **No light source, torches, particles or moving shadows:** M-ART-2.
- **Wide** shows part of a door beside the party (outside DRAWMAZE's
  picture) and, past a side opening that leads on, more darkness.

## Design questions for review

1. **The end of sight.** A clear dark panel with the last square fading
   into it (now), a softer vignette over the far squares, or Classic's flat
   background? Unlit, the panel is 40% of the picture's width.
2. **Close.** Keep `-36` (the party's own sides stay in view), or accept
   `-30`/`-32` for a stronger zoom and lose them?
3. **Wide** shows a little more than DRAWMAZE ever did (a door right beside
   the party). Acceptable, or should Wide stop at `-48`?
4. **Light.** The scene knows whether the party has LIGHT; Enhanced does not
   use it. With light the party sees five squares, which already shows. A
   brighter fog with light?
5. **The default:** Enhanced at Standard, as briefed. For M-ART-1A.3: keep
   the corner control in the header, and enlarge the picture?
6. **Palette:** front walls warm grey, sides a shade apart, browner floors,
   dark ceilings, dark wood doors in a light stone frame. Lighter or more
   contrast?
