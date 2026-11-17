# ART-3D: The Temple of Cant

**Baseline:** v0.19.0-boltac (`3b28701`). **Branch:** `art-3c-3d-castle-artwork`.

The Temple of Cant is illustrated and has its own story, as the Tavern,
the Inn and Boltac's are and do. Its theme: the darkness of doom, but the
hope of light.

## The artwork

| | |
|---|---|
| Master | `images/Artwork Samples/Temple/Elven High Priest in the Gothic Temple.png`, 1672×941 (16:9 to within 0.1%), SHA-256 `2346714a…582c378e` |
| Runtime copy | `gui/frontend/src/assets/scenes/temple.webp`, 1920×1080, 273,392 bytes |
| Conversion | `cwebp -q 82 -m 6 -sharp_yuv -metadata none -resize 1920 1080` (ART-2A's) |
| Focus | `75% 50%` |
| Brightness | Full (the appearance's default; nothing is set for this place) |

- **The master is kept outside Git**, unchanged, as the others are.
- **The crop is a little right of centre.** Father Caelion stands at the
  right. At 1440×900 his face is in view at every crop. At 980×692, 75%
  keeps the top of his staff in view (at 50% it leaves the window; at
  100% it is under the Actions panel), with the stained glass, its
  hooded figure and the figure rising toward the light, the altar and
  the censer.
- **At 980×692 his eyes are under the Actions panel** while it lists the
  members who cannot tithe (at every crop); with a shorter list, as after
  a treatment, his face is in view. The bier is under the party bar
  there. All of it is in view with Hide panels. Showing more would mean
  changing the frame's layout, which this work does not do.
- **Upscaled** about 1.15×, as the other pictures are.

## The story

`gui/content/temple/story.md`, the Temple's (`"temple": "temple"` in
`storyFolders`, `gui/story.go`). The supplied narrative
(`images/Artwork Samples/Temple/Temple_of_Cant`), with one sentence
corrected, as approved (2026-11-16), to what the Temple does: it cures
paralysis and stone and may raise the dead and those in ashes
(`engine/temple.go`); it does not cure poison. "Poison may be cleansed,
paralysis lifted…" is now "Petrification may be reversed, paralysis
lifted…". No other word is changed. It has no challenge.

The story promises nothing the rules do not keep: raising may fail ("Even
the most faithful prayer may fail"), the dead may become ashes and those
in ashes may be lost ("What was once flesh may become ashes, and what has
become ashes may yet be lost forever"), and the Temple asks a donation.

The approved text (2026-11-16):

> **The Temple of Cant**
>
> *Where darkness meets the promise of light*
>
> The great doors close behind you with a deep, hollow echo.
>
> Within the ancient Temple of Cant, towering columns of dark stone rise
> into vaulted ceilings lost in shadow. The air is cool and still, heavy
> with the fragrance of incense.
>
> High above the altar, magnificent stained-glass windows depict the
> eternal passage between life and death. A hooded figure descends into
> darkness, while another rises toward a brilliant light.
>
> A slender ribbon of incense smoke curls upward from a suspended bronze
> censer. As it passes through a shaft of violet light, it shimmers for
> a moment before disappearing into the shadows.
>
> Beside a stone bier stands a tall elven priest, his long silver hair
> falling over robes of charcoal and silver. His angular features and
> high cheekbones speak of an ancient lineage, while his expression
> carries the quiet sorrow of one who has witnessed countless
> departures.
>
> **Father Caelion, Keeper of the Veil.**
>
> He turns toward you, resting one hand upon his ceremonial staff.
>
> "Welcome, Commander, to the Temple of Cant."
>
> His voice is gentle, yet carries effortlessly through the great
> sanctuary.
>
> "Here we tend those whom the darkness has touched. Petrification may
> be reversed, paralysis lifted, and even death itself may sometimes be
> persuaded to release its hold."
>
> He gestures toward the stone bier.
>
> "Many have rested upon that stone. Some have risen again to walk
> beneath the sun. Others have passed beyond our reach."
>
> His eyes turn toward the great stained-glass window.
>
> "Understand this, Commander. We are servants of Cant, not masters of
> life and death. We may petition for mercy, but we cannot command it."
>
> For a moment, the sanctuary falls silent.
>
> "Even the most faithful prayer may fail. And each attempt to recall a
> departed soul carries its own peril. What was once flesh may become
> ashes, and what has become ashes may yet be lost forever."
>
> He looks back at you.
>
> "Bring me your wounded and your fallen. We shall do all that is within
> our power."
>
> A faint, knowing smile crosses his face.
>
> "And as for the offerings required by the Temple..."
>
> He pauses, glancing toward a discreetly placed collection chest.
>
> "Even those who serve eternity must attend to the expenses of the
> present."
>
> The incense continues its slow ascent toward the light.
>
> "Now, Commander. Whom have you brought before Cant?"

The first line is the story's title (its `#` heading); the second, in
italics, is its own paragraph.

The story box behaves as at every place with a story (ART-3B-1).

## Unchanged

The Temple's rules: who may be treated and for what, the donations, the
chances of raising and their risks (the dead to ashes, ashes to lost),
tithing and Pool party gold, the question before each treatment, the
receipts and saving. Also navigation, Escape, the emblems, the save
format and the settings.

## Visual review

`docs/design/art3/review/art-3c-3d-castle/`.
