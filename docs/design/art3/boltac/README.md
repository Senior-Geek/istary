# ART-3B-3B: Boltac's Trading Post

**Baseline:** v0.18.0-inn (`bb0f88a`). **Branch:** `art-3b-3b-boltac`.

Boltac's is illustrated and has its own story, as the Tavern (ART-3B-1)
and the Inn (ART-3B-2B) are and do.

## The artwork

| | |
|---|---|
| Master | `images/Artwork Samples/Trading_Post/Boltac’s Trading Post_ Emerald-Eyed Cat.png`, 1672×941 (16:9 to within 0.1%), SHA-256 `b1b837c8…f2d89d5b` |
| Runtime copy | `gui/frontend/src/assets/scenes/trading.webp`, 1920×1080, 350,350 bytes; byte for byte the preview approved in ART-3B-3A |
| Conversion | `cwebp -q 82 -m 6 -sharp_yuv -metadata none -resize 1920 1080` (ART-2A's) |
| Focus | `75% 50%` |
| Brightness | Full (the appearance's default; nothing is set for this place) |

- **The master is kept outside Git**, unchanged: `/images/Artwork Samples/`
  is already excluded locally in `.git/info/exclude`. Nothing is added to
  `.gitignore`. The earlier `Boltac’s Enchanted Trading Post.png` is
  superseded and not used.
- **The crop is a little right of centre** (ART-3B-3A,
  `docs/design/art3/review/art-3b-3a-boltac/`). At 980×692 this keeps
  Boltac's whole face clear just above the shop and the sign's "TRADING"
  whole. Centred, the Actions panel covered most of the sign. Kept to the
  right edge, his beard met the shop's edge.
- **What the shop covers:** both windows are narrower than 16:9, so the
  picture fills their height and the crop moves it only sideways. The
  shop (UX-2's `fill` layout) stands from about 40% of the window's
  height down. At 1440×900 its top edge crosses Boltac at the mouth.
  With a customer at the counter, the cat on its cushion is under the
  shop. Its upper line, "BOLTAC'S", is half under the head bar. All are
  in view with nobody at the counter (the cat whole at 1440×900) and with
  the panels hidden. Accepted in ART-3B-3A; showing more would mean
  changing the shop's layout.
- **Upscaled** about 1.15×, as the other pictures are.

## The Selected Character panel

At a place without a picture, the Selected Character panel
(`SelectedCharacter.svelte`, used only at Boltac's) was drawn on the
page. Over the picture it was transparent. The frame gives its panels
glass (`.content-area > section`), but its selector does not reach this
panel, which is nested in the shop's `.counter-layout`. The customer's
gold, AC and items were drawn straight on the painting. Now, at an
illustrated place, the panel has the Actions panel's own glass: the
appearance's `--glass` (92% in Dark, with a 6px blur), its gold edge,
radius and shadow. Its size, padding and place are unchanged. Its lists
still scroll, and its Equip and Take off buttons are where they were.
With the artwork Hidden, it is the same as the other panels.

## The story

`gui/content/boltac/story.md`, the Trading Post's (`"trading": "boltac"`
in `storyFolders`, `gui/story.go`). The approved text (2026-11-16):

> **Boltac's Trading Post**
>
> The door opens with the cheerful jingle of a brass bell. Warm light
> spills across polished armour, racks of swords, and shelves crowded
> with bottles of suspiciously brilliant colours. Somewhere beneath the
> counter, a cat opens one eye, judges you unworthy of further attention,
> and goes back to sleep.
>
> Behind a great wooden counter sits a round, white-haired gentleman
> whose smile grows noticeably wider as he notices your purse.
>
> "Welcome, welcome!" cries **Boltac**, spreading his arms. "The finest
> weapons, the stoutest armour, the most remarkable magical treasures in
> all of Trebor's kingdom! And every one of them reasonably priced!"
>
> You glance at a battered shield bearing what looks remarkably like a
> fresh dragon bite.
>
> "Previously owned," Boltac explains smoothly. "By a gentleman who no
> longer requires it. Excellent workmanship."
>
> He gestures toward his shelves.
>
> "Buy, sell, identify, remove a troublesome curse — all under one roof!
> Though I should mention that certain enchantments are rather more
> difficult to remove than they are to acquire."
>
> He leans forward confidentially.
>
> "Now, should you happen to discover anything interesting in Werdna's
> Maze, bring it straight to me. I pay a fair price for everything. Well,
> almost everything."
>
> The cat raises its head at the sound of coins being counted.
>
> Boltac smiles again.
>
> "Remember, my friend: a good sword may save your life. A better sword
> may save it twice. And if neither does, I'll be delighted to purchase
> them back from whoever brings them home."
>
> **Equip your adventurers, inspect unfamiliar treasures, and spend your
> gold wisely. Boltac never forgets a customer — particularly one who
> still owes him money.**

The first line is the story's title (its `#` heading); the last is its
challenge (`> [!challenge]`), in the story box's gold.

The story box behaves as at every place with a story (ART-3B-1): open on
the first visit of the session, closed there once closed, until Story
opens it again; the other places' are left as they were.

## Unchanged

Boltac's rules and the shop: serving the first party member on arrival,
Leave the counter and choosing a customer from the party bar, Buy, Sell,
Identify and Uncurse, stock, prices, fees and refusals, the questions
before buying an unusable item or selling a worn one, the receipts,
Pool party gold, Equip and Take off, the shop's layout, the party bar
always shown and no character sheet here. Also unchanged: navigation,
Escape, the emblems, the other places' pictures and stories, the save
format and the settings.

## Tests

- **Go:** the story files (Boltac's among them), and Boltac's story kept
  to the approved text above, word for word, with its title, the bold
  name and its challenge (`story_test.go`); the frame fixtures, which now
  record Boltac's view (`frame_fixture_test.go`).
- **Frontend:** Boltac's picture (a 1920×1080 WebP, focus 75% 50%) and
  the others' unchanged (`boltac.test.js`); his story in the View and
  drawn, twelve paragraphs and its challenge (`content1a.test.js`); the
  Selected Character panel's glass over a picture only.
- **Live (`scenes`):** Boltac's with the other illustrated places (its
  picture, panels, Hide panels, the larger and smaller windows, all 27
  appearances); its own story on its first visit, closed not opened again
  on return, Story opens it, the others' left as they were. The Temple is
  now the place without artwork the suite checks. Reaching Boltac's for
  the first time, these checks found three things that the picture does
  not cause. ART-3B-3C corrects them: the party bar's current slot, the
  shop's buttons under the column heads, and the story box at a place
  whose controls it would cover. See the review README.

## Visual review

`docs/design/art3/review/art-3b-3a-boltac/` (the preview) and
`docs/design/art3/review/art-3b-3b-boltac/` (as integrated).
