# ART-3B-2B: the Adventurer's Inn

**Baseline:** v0.17.0-tavern (`bbea3aa`). **Branch:** `art-3b-2b-inn`.

The Inn is illustrated and has its own story, as the Tavern (ART-3B-1).

## The artwork

| | |
|---|---|
| Master | `images/Inn/Adventurer’s Inn Reception Hall.png`, 1670×942 (16:9 to within 0.3%), SHA-256 `445c9018…e528` |
| Runtime copy | `gui/frontend/src/assets/scenes/inn.webp`, 1920×1080, 452,818 bytes; byte for byte the preview approved in ART-3B-2A |
| Conversion | `cwebp -q 82 -m 6 -sharp_yuv -metadata none -resize 1920 1080` (ART-2A's) |
| Focus | `100% 50%` |

- **The master is kept outside Git**, unchanged: `/images/Inn/` is
  excluded locally in `.git/info/exclude`. Nothing is added to
  `.gitignore`.
- **The crop keeps the picture's right edge** (ART-3B-2A,
  `docs/design/art3/review/art-3b-2a-inn/`). Centred, at 980×692 the
  actions covered the sign from "Adven…" on and, with a guest chosen,
  half of Mistress Elspeth's face. Kept right, she, the sign and the desk
  are clear of the panels at both window sizes, with the story box open
  or closed and with or without a guest.
- **What the crop gives up:** the picture's left 10% (1440×900) or 20%
  (980×692), the fireplace's left side and the reader in his armchair,
  mostly behind an open story box at either crop. The desk's front and
  the bell are under the information, as the Tavern's bar is.
- **Upscaled** about 1.15×, as the other pictures are.

## The story

`gui/content/inn/story.md`, the Inn's (`storyFolders` in
`gui/story.go`). The approved text (2026-11-16):

> **The Adventurer's Inn**
>
> The heavy oak door closes behind you, shutting out the noise of the
> castle courtyard. Warmth, candlelight, and the comforting smell of
> fresh bread greet you. Across the hall, a fire crackles, and the
> distant sound of laughter drifts down a corridor.
>
> Behind a polished wooden desk stands a smiling woman in a
> blue-and-gold gown. An enormous ledger lies open before her, beside a
> brass bell and a quill and inkwell.
>
> "Welcome to the Adventurer's Inn," she says. "I'm **Mistress
> Elspeth**. You look as though you've come from the Maze. Or are about
> to. It's sometimes difficult to tell."
>
> She glances at your companions with a practiced eye.
>
> "Stables are free, of course. A little straw, a roof overhead, and
> time enough to recover your spells. But if you've acquired a few
> injuries, I recommend a proper bed. We have accommodations for every
> purse, from a modest cot to the Royal Suite."
>
> She pauses, turning a page in her ledger.
>
> "Naturally, the better the room, the quicker the recovery. Gold has a
> remarkable effect on one's health, I've found."
>
> Her smile softens.
>
> "And should one of your companions be in a condition that sleep cannot
> remedy, I'm afraid you'll need the Temple of Cant. We provide beds,
> not miracles."
>
> She reaches for her quill.
>
> "Now then, who's staying, and how long?"
>
> **Rest your adventurers, restore their spells, and heal their wounds.
> Choose your accommodations carefully; comfort has its price.**

The first line is the story's title (its `#` heading); the last is its
challenge (`> [!challenge]`), in the story box's gold.

The story box behaves as at every place with a story (ART-3B-1): open on
the first visit of the session, closed there once closed, until Story
opens it again.

## Unchanged

The Inn's rules (the rooms and their rates, healing, spells, payment, the
level check, the refusal of a guest who is not OK), the party bar, the
sheet's priority over the story box, navigation, Escape, the emblems,
the other places' pictures and stories, the save format and the settings.

## Tests

- **Go:** the story files (the Inn's among them), and the Inn's story
  kept to the approved text above, word for word, with its title, the
  bold name and its challenge (`story_test.go`); the frame fixtures, which
  now record the Inn's view (`frame_fixture_test.go`).
- **Frontend:** the Inn's picture (a 1920×1080 WebP, focus 100% 50%) and
  the others' unchanged (`inn.test.js`); the Inn's story in the View and
  drawn, eleven paragraphs and its challenge (`content1a.test.js`).
- **Live (`scenes`):** the Inn with the other illustrated places (its
  picture, panels, Hide panels, Hide party, the larger and smaller
  windows, all 27 appearances); its own story on its first visit, closed
  not opened again on return, Story opens it, the Tavern's left as it
  was. Boltac's is now the place without artwork the suite checks. No
  check was loosened for the Inn.
