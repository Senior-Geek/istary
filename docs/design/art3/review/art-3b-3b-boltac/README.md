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

## Checked (targeted, 50 at each size)

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

All passed at 1440×900. At 980×692, two did not. Both are the
Uncurse tab's "Uncurse" buttons, scrolled into view from below in a list
two rows tall, under the list's sticky column heads (see below).

## Open findings

The live suite's `scenes` checks now include Boltac's, which they had
never reached. They found three things, none of them caused by the
picture:

1. **The current customer's party-bar slot: muted text below AA**
   (`PartyBar.svelte`, `.slot.current`, since M8-4). The slot number,
   class and row ("1", "N-FIG", "Front") are `--muted` on `--selected`:
   4.29:1 in Slate and 3.77:1 in Dark (and on the game's own colours),
   against AA's 4.5:1. Light passes, at 5.43:1. Both colours are
   opaque, so the picture plays no part. A current slot is shown only at
   Boltac's and in Camp.
2. **A focused Buy, Sell, Identify or Uncurse button can stop under the
   list's sticky column heads.** Rows have `scroll-margin-top` (UX-2), so
   a row brought into view stops below the heads, and the shop does this
   itself. A button scrolled into view on its own, as the suite does and
   as Shift-Tab would, can be left under them. With the list at its top,
   every control at every window size is in view. The suite's check
   leaves the list scrolled at one size, and the next size's check then
   finds the first Buy under the heads. This is the same in the shop
   without a picture.
3. **A story box the player has moved or enlarged elsewhere can cover
   Boltac's tabs at 980×692** (ART-2B: where the box stands and its size
   are the player's, the same at every place). Boltac's shop starts
   higher than other places' information. Where it first opens, the
   story box covers nothing at Boltac's at either size.

None of these is changed here, and the live suite's checks are left as
they are, failing on them.
