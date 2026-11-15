# ART-3B-0: Stage 2 visual review

**Commit reviewed:** `21fa6a3` (branch `art-3b-location-emblems`, on
v0.15.0-markdown-stories). **Status:** approved.

The places' emblems and the frame's layout preparations as a player sees
them, in a copy of Game 1 (Dark panels, Parchment story, full artwork). A
`wails dev` build in a worktree of its own, on that copy, read by headless
Chrome with the page set to exactly 1440×900 or 980×692, at 2× device
pixels.

| File | What it shows |
|---|---|
| `castle-story-open-1440x900.png` | The Castle's picture, the tower beside the title, the story box open; the Castle has no information, so that row is empty |
| `gilgameshs-tavern-1440x900.png` | The tankard in the title and on the themed panel; the roster and party |
| `boltacs-trading-post-1440x900.png` | The balance and coin in the title and on the themed panel; the goods, the selected character |
| `boltacs-trading-post-980x692.png` | The same at the smallest window: the goods scroll in their list, nothing is covered |
| `contact-sheet.png` | The four together |

Measured in each: the emblem 34×34 px, its centre 0.3 px from the title
text's, 10 px before it, in the panel's gold (`#d2b06a`), `aria-hidden`;
no control off the page or covered, no area overflowing sideways.

## The acceptance sizes

The live suites (`scripts/acceptance/live.mjs`) size headless Chrome's
*window* (`--window-size`), and its page is smaller by the window's frame.
At the sizes the acceptance level reports as 1440×900 and 980×692, the
pages were 1440×813 and 980×605. These pictures were taken at the full
page sizes. The checks are stricter, not weaker, for it (at 980×605 the
selected character at Boltac's scrolls; at 980×692 it fits). The
acceptance sizes and checks are unchanged in ART-3B-0.
