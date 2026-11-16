# ART-3B-3B: Boltac's Trading Post, as integrated

**Branch:** `art-3b-3b-boltac`, on v0.18.0-inn (`bb0f88a`). **Status:**
awaiting review. Design notes and the approved story:
`docs/design/art3/boltac/README.md`. The preview: `../art-3b-3a-boltac/`.

Taken with `wails dev` on a copy of the acceptance library. The copy has
an unidentified helm, a cursed LONG SWORD -1 and 5,000 GP for the first
member, and 300 GP for the second. Headless Chrome read the pages at
exactly 1440×900 or 980×692, at 2× device pixels, with Dark panels,
Parchment story and Full artwork.

| File | What it shows |
|---|---|
| `arrival-story-open-980x692.png` | The first visit: Boltac's story open beside his face and the sign; the first member served, Buy; the Selected Character panel on its glass |
| `buy-1440x900.png` | Buy with a customer, the story closed: the unidentified helm and the cursed sword in the customer's items |
| `ask-sell-worn-980x692.png` | Selling the worn CHAIN MAIL: Boltac's question, the lists and equipment waiting |
| `no-customer-1440x900.png` | After Leave the counter: nobody served; Boltac, the sign and the cat in view |

## Checked (targeted, 66 at each size)

These ran at each size, on a fresh copy:
- **Arrival:** Boltac's own story, titled, with its challenge; the
  picture at `75% 50%`, Full; his face and "TRADING" in view.
- **The Selected Character panel:**
  - its text meets WCAG AA on its glass, which is the Actions panel's own;
  - this also holds in Subdued, Hidden, Slate and Light.
- **Buying and its receipt:** a STAFF bought for 10 GP ("Saved").
- **Equip and Take off:** done from the panel.
- **Selling and its receipt:** the STAFF sold back for 5 GP ("Saved").
- **The worn CHAIN MAIL:**
  - Boltac asks before selling it, and the lists and equipment wait;
  - "No, keep it" leaves it worn, with no gold moved.
- **Identify and Uncurse:** the HELM identified for 50 GP and the LONG
  SWORD -1 uncursed for 500 GP, each "Saved".
- **Pool party gold:** 4,445 + 300 GP to the customer.
- **Leave the counter:** nobody served, then a customer chosen from the
  party bar.
- **Scrolling:** the stock list scrolls, and the customer's items scroll
  with their last button clickable.
- **The story box:**
  - Story reopens it;
  - once closed, it is not reopened on return.
- **Hide panels:** Boltac, the sign and the cat in view.
- **Escape and ← Castle:**
  - Escape brings the panels back;
  - Escape again goes to the Castle;
  - ← Castle goes to the Castle.
- **Layout:** in every state, no control is clipped or covered, and
  there is no overlap between the shop, the Selected Character panel,
  the party bar, the head, the Actions panel and the story box.

In ART-3B-3B, all passed at 1440×900, and at 980×692 two did not. Both
were the Uncurse tab's "Uncurse" buttons, scrolled into view from below
in a list two rows tall, under the list's sticky column heads (finding 2
below). With ART-3B-3C's corrections, all 66 checks pass at each size.
The 66 include the keyboard and story-box checks below.

## The live suite's findings, corrected (ART-3B-3C)

The live suite's `scenes` checks reached Boltac's for the first time here.
They found three things, none of them caused by the picture. ART-3B-3C
corrects them, and the live suite passes them unchanged (319 checks at
each size).

1. **The current member's party-bar slot: secondary text below AA.**
   Since M8-4, `PartyBar.svelte` drew the slot number, class, row and
   the "Lv / HP / AC" line in `--muted` on `--selected`: 3.77:1 in Dark
   and on the game's own colours, and 4.29:1 in Slate. AA asks 4.5:1.
   Both colours are opaque, so the picture played no part. Now, in the
   current slot only, they are `--selected-muted`, a new token in each
   appearance halfway from `--muted` to `--text`:

   | Appearance | Before | Now |
   |---|---|---|
   | Dark (and the game's own) | 3.77:1 | 5.82:1 |
   | Slate | 4.29:1 | 5.58:1 |
   | Light / Parchment | 5.43:1 | 7.72:1 |

   The highlight (`--selected`) and the name are unchanged, and these
   texts are still below the name. The same component draws Boltac's
   customer and Camp's member, so both are corrected.
   (`partyBarContrast.test.js`)
2. **A focused shop button could stop under the sticky column heads.**
   The rows had `scroll-margin-top: 30px` (UX-2), so a row brought into
   view stopped below the heads. A button brought into view on its own,
   by Shift-Tab or by the suite, did not. Now the row's button has the
   same margin. The heads stay sticky, and nothing else in the table
   changes. Checked live: Shift-Tab up and Tab down each list's enabled
   buttons (Buy 27, Sell 5, Identify 1, Uncurse 1), each wholly in view
   below the heads, at both sizes. Without the margin, the same check
   fails on Buy and Sell.
3. **A story box moved or enlarged at one place could cover another's
   controls** (ART-2B: where the player leaves it is the same at every
   place). As a place is entered, or its story opened, the box now finds
   the controls in view there (`StoryBox.svelte`, `lib/storyPlace.js`).
   If the player's place covers none, it is drawn there. Otherwise it is
   drawn at the nearest clear place inside the window, chosen in this
   order:
   - moved up or down, or made shorter, its left and width kept,
     whichever changes it least;
   - only if there is no such place, moved aside;
   - only if there is no clear place at all, where it first opens.

   The player's place is not changed, so other places draw it where the
   player left it. Moving or resizing it starts from where it is drawn
   and becomes the player's place. Closing and opening it work as
   before. No place or coordinate is special. Checked live at both sizes:
   - with the player's place where the Tavern's opens, the Inn, the
     Castle and the Tavern draw it there;
   - Boltac's draws it moved up clear of the tabs (54px at 980×692, 79px
     at 1440×900);
   - Story reopens it clear;
   - a move at Boltac's starts from there and is kept at the Tavern.

   The live suite's own moves and resizes at the Edge and the Castle are
   kept as before. (`storyPlace.test.js`)
