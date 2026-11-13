# ART-2A — Castle entrance and courtyard

Status: implemented on `art-2a-castle` (2026-11-13), awaiting visual
approval. The first illustrated places. A presentation milestone: no rule
of the game, no navigation and nothing saved changes.

## Audit (before)

- **The scene infrastructure (Castle-0):**
  - `lib/scenes.js` named each place's picture; every `src` was `null`.
  - `LocationScene` drew the picture, or a themed panel with the place's
    emblem, in the frame's display area A: the upper left, as high as the
    actions beside it.
- **The frame (UX-3B):** one layout for every town place:
  - the head (title, Back, Hide party);
  - A, the display, with the actions B beside it;
  - C, the information, below both and scrolling on its own;
  - D, the party bar.
- **Width:** the town shell was at most 920 px wide, or 1300 px from a
  1240 px window up, and centred.
- **No way to hide the panels.** "Hide party" (a saved setting, Go's) was
  the only toggle.
- **Window:** 980×720 to start, 760×560 at least; native full screen is
  allowed.

## Artwork

The approved pictures are in `images/Artwork Samples/`. They are kept
unchanged and untracked; they are excluded locally in `.git/info/exclude`,
so a checkout stays clean for the acceptance level and the gate.

| Place | Original | Runtime copy |
|---|---|---|
| The Castle (inside the walls) | `Istary Castle Courtyard Hub.png`, 1671×941 | `gui/frontend/src/assets/scenes/castle.webp`, 1920×1080, 583 KB |
| The Edge of Town (outside the walls) | `Istary Castle_ Gates, Grounds, and Maze.png`, 1672×941 | `gui/frontend/src/assets/scenes/edge.webp`, 1920×1080, 604 KB |

- **Proportions:** both originals are 16:9 to within 0.1%, so nothing is
  cropped to make the copies. Making them 1920×1080 changes the height by
  under one pixel.
- **Upscaled:** the originals are smaller than the target, so each copy is
  about 1.15× larger than its original and holds no more detail. On a
  Retina full screen (3456×2234 device pixels on a 16-inch MacBook Pro)
  the picture is drawn at about twice its pixels.
- **Conversion:** `cwebp -q 82 -m 6 -sharp_yuv -metadata none -resize 1920 1080`.
- **The geography is the manual's:**
  - the courtyard's gate, on the left, leads out to the Training Grounds
    and the maze;
  - outside, the gatehouse is in the middle, the Training Grounds on the
    left and the maze's entrance on the right;
  - through the gate, the courtyard's signs.
- **Not used:** the same folder holds `Bustling Fantasy Castle
  Courtyard.png` (an overview of both scenes), `Inn.png` and `Tavern.png`,
  which are for later milestones.

## Layout

An *illustrated* place is one whose scene has a picture
(`illustrated(location)` in `scenes.js`). Adding a picture to another
place makes it illustrated with no other change. A place that draws its
own scene (the frame's `scene` snippet) is never illustrated.

- **The picture fills the window behind the frame.**
  - `LocationScene mode="backdrop"`: fixed, `object-fit: cover` around the
    scene's focus. It is cropped to the window and never stretched.
  - The shell is its own stacking context (`main.shell.art { isolation:
    isolate }`), so the picture is above the page's background and below
    everything else.
  - The banners (errors, notices) draw over it as before.
- **The areas are the same as UX-3B's**, laid over the picture:
  - **A** is a clear opening (`LocationScene mode="view"`), described for
    screen readers by the scene's text.
  - **The head, the actions and destinations (B), the information (C)
    and the party bar (D)** are separate panels, each near-opaque (90%,
    with a light blur), so their text reads over any part of the picture.
    The picture shows between them.
- **No maximum width.** At an illustrated place the shell takes the whole
  window, and the panels reach its edges.
- **The information is as tall as it needs**, up to 45% of the body, and
  scrolls on its own past that. The display and actions take the rest, so
  at the Edge of Town all the actions fit at 1440×900. Before, they
  scrolled.
- **Hide party** gives its room to the panels: the actions grow, and the
  information moves down.
- **Hide panels** is in the head. It leaves the picture and one control:
  - the place's name with "Show panels", in the upper right;
  - the focus moves to Show panels, and back to Hide panels after;
  - Escape shows the panels too, before it hides a sheet or goes Back;
  - it is the window's own state (App's `PANELS`): nothing is saved or
    sent to Go, and the panels come back at any other place.
- **A character sheet** stands over the picture in A, as before. Hiding
  the panels hides it with the rest.
- **The focus points:**
  - courtyard `45% 50%`: the gate, on the left, is the way out, and the
    right edge is under the actions;
  - outside the walls `50% 50%`.

## What is not done (per the brief)

- **No clickable hotspots.** The signs in the pictures are only part of
  the pictures.
- **The Training Grounds, the Tavern, Boltac's, the Inn and the Temple**
  keep their themed panels until their pictures are approved. The frame
  is ready for them: give a scene a `src`. Two places will need a look
  then:
  - Boltac's lays out C itself (`fill`);
  - the Training Grounds shortens A and B while a character is created
    (`compact`).

## Tests

- **Frontend (node):**
  - the Castle and the Edge have their pictures, 1920×1080 WebP, read
    from the files' headers;
  - every other place has none;
  - the frame shows the picture as a backdrop, offers Hide panels only
    where there is one, and while hidden has one control and no working
    area;
  - Escape shows the panels first;
  - the picture is a cover;
  - nothing about the panels is sent to Go.
- **Live (`scripts/acceptance/live.mjs`, suite `scenes`; see
  `docs/TESTING.md`):** the picture loaded and seen through the display,
  the panels near-opaque, every control uncovered, Hide and Show panels,
  Escape, Hide party, the larger and smaller windows, the ways between the
  places, the sheet over the picture, the game saved exactly as before.
  - The check that the picture is *seen* was written after a first
    version drew it under the page's background. That version passed a
    check of the picture's size and place. The new check fails against
    it.
- **The existing layout and sheet suites** pass unchanged at the
  illustrated places.

## Remaining layout notes

- **At 980×692, the Edge of Town's actions still scroll.** Its Go to list
  is below Character Sheet. The information's two paragraphs take about
  115 px of the height. With the party hidden, all of it fits.
- **Narrower windows crop the picture's sides.** The courtyard's "Castle
  Gate" sign, at its left edge, is partly cut at 980×692 and in any window
  narrower than about 1.6:1. So is the Temple's, at its right edge. The
  pictures are 16:9; full screen on a MacBook Pro is about 1.55:1.
- **Under the panels:** with the panels shown, the actions cover the
  courtyard's Temple and the maze's entrance outside, on the right. Hide
  panels shows them.
- **Sharpness:** the pictures are 1672 px wide. They are visibly soft on a
  large Retina full screen. Masters of 2560×1440 or more would fix that,
  with no change to the code.
