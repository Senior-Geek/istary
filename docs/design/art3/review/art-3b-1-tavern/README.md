# ART-3B-1: Gilgamesh's Tavern, visual review

**Commit reviewed:** `013515e` (branch `art-3b-1-tavern`, on
v0.16.1-navigation-emblems). **Status:** awaiting review.

The Tavern's picture and story as a player sees them, in a copy of Game 1
(Dark panels, Parchment story). A `wails dev` build in a worktree of its
own, on that copy, read by headless Chrome with the page set to exactly
1440×900 or 980×692, at 2× device pixels. Each file is at both sizes
(`-1440x900.png`, `-980x692.png`).

| File | What it shows |
|---|---|
| `tavern-full-story-open` | Full artwork; the Tavern's own story on its first visit, after the Castle's was closed |
| `tavern-full-story-closed` | Full artwork, the story box closed: Gilgamesh and the sign beside the panels |
| `tavern-subdued-story-open` | Subdued artwork, the story box open |
| `tavern-subdued-story-closed` | Subdued artwork, the story box closed |
| `tavern-sheet` | A character's sheet shown in the display; the story box is not drawn over it |
| `tavern-roster-end` | The roster scrolled to its last row; the six-slot party bar |
| `tavern-full-panels-hidden` | Hide panels: the whole picture |

Checked in the same run (25 checks at each size, all passed): Gilgamesh's
face and body and the whole sign in view, in Full and Subdued, the story
box open or closed; the Tavern's story opened on its first visit after
the Castle's was closed, not reopened on return once closed, reopened by
Story; the sheet over the story box and Story with a sheet shown; Remove
and Join, alignment refused, Divvy Gold offered; the roster scrolling;
Hide panels and Escape; no control off the page or covered.
