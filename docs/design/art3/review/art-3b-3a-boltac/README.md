# ART-3B-3A: Boltac's Trading Post, artwork preview

**Baseline:** v0.18.0-inn (`bb0f88a`). **Status:** approved: focus
`75% 50%`, Full, and the Selected Character panel's glass, integrated in
ART-3B-3B (`docs/design/art3/boltac/README.md`,
`docs/design/art3/review/art-3b-3b-boltac/`). The preview itself
integrated nothing and wrote no story.

## The artwork

| | |
|---|---|
| Master | `images/Artwork Samples/Trading_Post/Boltac’s Trading Post_ Emerald-Eyed Cat.png`, 1672×941 RGB PNG, 16:9 to within 0.1%, unchanged (SHA-256 `b1b837c8…f2d89d5b`), outside Git (`/images/Artwork Samples/` in `.git/info/exclude`) |
| Preview WebP | 1920×1080, 350,350 bytes, not in the repository |
| Conversion | `cwebp -q 82 -m 6 -sharp_yuv -metadata none -resize 1920 1080` (ART-2A's, the Inn's) |
| Not used | `Boltac’s Enchanted Trading Post.png`, superseded |

In the picture: Boltac's face about 66% across and 37% down; the sign
53–89% across and 4–25% down; the cat's face about 39% across and 65%
down, its diamond 70% down.

Taken with `wails dev` in a temporary worktree, on a disposable copy of
the acceptance library with an unidentified helm and a cursed LONG
SWORD -1 added to the first member, and a placeholder story the length of
the Inn's. The pages were read by headless Chrome at exactly 1440×900 or
980×692, at 2× device pixels, with Dark panels, Parchment story and Full
artwork. The focus was set by an injected stylesheet.

## How Boltac's differs from the Tavern and the Inn

- **The shop fills the information area** (`fill`; UX-2). The
  merchandise and the Selected Character panel stand under the picture,
  from 40% (1440×900) or 43% (980×692) of the window's height down to the
  party bar. The picture shows in the band above them: 1092×270 beside
  the Actions panel at 1440×900, and 632×208 at 980×692.
- **No vertical crop.** Both windows are narrower than 16:9, so the
  picture fills their height exactly. A focus can only move it sideways.
  Boltac's face is always about 37% down: at the foot of that band.
- **The Selected Character panel has no panel of its own.** The shop's
  section is opaque (`rgb(28,28,28)`), but `aside.selected` is
  transparent. Its text (Gold, AC, Carrying, the items, "Nobody is at the
  counter") is drawn straight on the painting, and its lower half lies on
  Boltac's shirt, the scales and the lantern. The other places' panels are
  glass (`.content-area > section`). That selector does not reach these
  two, which are nested in `.counter-layout`.
- **The party bar is always shown** (`partyAlways`), and there is no
  character sheet (`noSheet`). The shop's question (Unusable or Equipped
  item) and its receipts are drawn in the counter, above the list. A
  waiting question adds a line to the head bar ("Returning answers
  no…"), which moves the panels down by about 28px.

## The crops

What the player sees in the working states (story open or closed, Buy,
Sell, Identify, Uncurse, a receipt, the question). The picture's
position does not change between tabs.

| Focus | 1440×900: Boltac | 1440×900: sign | 980×692: Boltac | 980×692: sign |
|---|---|---|---|---|
| `50% 50%` | Face to the mouth, then the shop's top edge | "TRADING", "POST" under the Actions | Whole face, clear, beside the shop; his hand and chest behind the Selected Character text | "TRADI…" only |
| **`75% 50%`** | Face to the mouth | "TRADING PO…" | **Whole face, clear, just above the shop** | **"TRADING" whole** |
| `100% 50%` | Face to the mouth | "TRADING POS…" | Face; the beard meets the shop's top edge | "TRADING" and a little more |

- **At 1440×900 the crop barely matters for Boltac.** At every crop the
  shop's top edge (358px of 900) crosses his face at the mouth. Only a
  layout change could show more of him in a working state. The crop
  changes only how much of "POST" escapes the Actions panel.
- **At 980×692 the crop matters most.** `75%` keeps his whole face clear
  and the word "TRADING" whole. `50%` gives him more room but loses most
  of the sign. `100%` crowds his beard against the shop.
- **"BOLTAC'S"** is half under the head bar at every crop and size, in
  every working state. A focus cannot change this (no vertical crop).
- **The cat** is under the opaque shop panel in every state with a
  customer, at every crop and size. It is in view when nobody is at the
  counter (whole, at 1440×900; its face but not its diamond at 980×692)
  and with the panels hidden.
- **With the panels hidden**, the whole painting is in view at every
  crop. The restore bar covers part of "BOLTAC'S", as it covers a corner
  at the other places.
- **The story box** (top left) covers the armour, helmets and lantern,
  never Boltac or the sign, at either size.

## Brightness

Full is preferred. Subdued (checked at `75%`, both sizes) dims the
painting under the transparent Selected Character text. That reads
better, but its labels (Gold, AC) are still faint. Subdued is the
player's choice for every place, so it cannot be the fix.

## Interface checks

At each size, at all three crops: arrival (the first member served,
Buy), story open and closed, Sell, Identify (the helm), Uncurse (the long
sword), buying a STAFF and selling it back (both receipts "Saved"),
selling the worn CHAIN MAIL (the question, answered "No, keep it"),
Leave the counter (nobody served) and serving again from the party bar,
Hide panels, then Escape (the panels back) and Escape again (the Castle).

- **No control clipped, covered, off the page or cut off**, in any
  state, at any crop or size (with or without the panel mock below). The
  page never scrolls sideways.
- **Unchanged:** stock, prices, refusals, the question, the receipts,
  the Equip and Take off buttons, `noSheet`, `partyAlways`, the party
  bar's choice of customer, Escape and ← Castle.
- **Hard to read: the Selected Character text over the painting** (see
  above): `crop-50-buy-980x692.png` and `crop-100-buy-980x692.png`.

## Recommendation

- **Focus `75% 50%`, Full brightness.** This is the best balance at
  980×692, where it matters: Boltac's whole face and "TRADING" are clear.
  At 1440×900 the result is the same as the other crops, with a little
  more of the sign than `50%`.
- **Give the Selected Character panel the glass panel the others have**
  (for an illustrated place only; ART-3B-3B, CSS only). The mock in
  `crop-75-buy-glass-mock-980x692.png` is the frame's own `--glass`,
  border, blur and shadow, with the same padding. Its text reads, and no
  control moves off the page or is covered. It changes no control's
  place, and the panel is used only at Boltac's. Without it, the
  painting makes the customer's gold and items hard to read. **APPROVE
  the artwork, with this change.**

## Decisions for review

1. **The Selected Character's glass panel** (above): approve it for
   ART-3B-3B, or choose another remedy.
2. **Boltac's mouth and the cat under the shop at 1440×900** are
   accepted as they are. Showing more would mean changing the shop's
   layout or recomposing the picture, which this phase rules out.
3. **"BOLTAC'S" half under the head bar** in working states, at every
   crop.

## Files

| File | What it shows |
|---|---|
| `crop-50-buy-980x692.png` | `50%`, Buy, story closed: his whole face, the sign cut to "TRADI"; the Selected Character text hard to read over the painting |
| `crop-100-buy-980x692.png` | `100%`, the same state: more of the sign, his beard at the shop's edge |
| `crop-75-buy-glass-mock-980x692.png` | `75%`, the same state, with the Selected Character panel given the glass panel (a mock): the recommendation |
| `crop-75-story-open-1440x900.png` | `75%`, arrival with the story box open: the shop's edge at his mouth |
| `crop-75-no-customer-1440x900.png` | `75%`, nobody at the counter: Boltac, the sign and the cat in view; the transparent Selected Character text |

The rest of the 84 exploratory screenshots are not kept. They are in
`/private/tmp/claude-503/-Users-leopard45/90008b1c-6346-40ca-befd-243334d05121/scratchpad/art3b3a/shots/`
and `…/shots-glass/`, with the harness and measurements.
