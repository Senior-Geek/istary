# ART-3C: The Training Grounds

**Baseline:** v0.19.0-boltac (`3b28701`). **Branch:** `art-3c-3d-castle-artwork`.

The Training Grounds are illustrated and have their own story, as the
Tavern, the Inn and Boltac's are and do.

## The artwork

| | |
|---|---|
| Master | `images/Artwork Samples/Training_Ground/Dwarven Drillmaster at Castle Camp.png`, 1672×941 (16:9 to within 0.1%), SHA-256 `0c34814b…c5be0989` |
| Runtime copy | `gui/frontend/src/assets/scenes/training.webp`, 1920×1080, 336,664 bytes |
| Conversion | `cwebp -q 82 -m 6 -sharp_yuv -metadata none -resize 1920 1080` (ART-2A's) |
| Focus | `100% 50%` |
| Brightness | Full (the appearance's default; nothing is set for this place) |

- **The master is kept outside Git**, unchanged: `/images/Artwork Samples/`
  is already excluded locally in `.git/info/exclude`. Nothing is added to
  `.gitignore`.
- **The crop keeps the right edge.** Sergeant Major Brannock Ironheel
  stands at the right, his face about 81% across the picture, where the
  Actions panel stands. Both test windows are narrower than 16:9, so the
  picture fills their height and the crop only moves it sideways (by at
  most 160 px at 1440×900 and 250 px at 980×692). Kept to its right edge,
  the most of him is in view: at 1440×900 his eyes, beard, medals and the
  hand on his axe, with the Actions panel's edge at his cheek; the castle
  gate, the banner, the young mage's flame and the recruits at their
  practice are in view. Crops at 50% and 75% put more of him under the
  panel.
- **At 980×692 his face is under the Actions panel** at every crop (the
  face would need a focus past 100%). His shoulders, the axe's haft and
  his medals are in view above the information; all of him is in view
  with Hide panels, or in a larger window. Showing him there would mean
  changing the frame's layout, which this work does not do.
- **Upscaled** about 1.15×, as the other pictures are.

## The story

`gui/content/training-grounds/story.md`, the Training Grounds'
(`"training": "training-grounds"` in `storyFolders`, `gui/story.go`).
The supplied narrative (`images/Artwork Samples/Training_Ground/Training_grounds`),
with two quotation marks it lacked: one opening Brannock's first speech
("So, you're the new commander…") and one closing his last ("…I hate
training replacements."). No word is changed. It has no challenge.

The approved text (2026-11-16):

> **The Training Grounds**
>
> *Beyond the walls of Trebor's castle*
>
> The sound of clashing steel carries across the dusty training grounds.
> Beneath the castle's towering battlements, recruits of every race
> practice their trades. Swords strike wooden targets, arrows whistle
> through the air, and a young mage struggles to control a flame dancing
> between her fingers.
>
> Overseeing this confusion stands a broad-shouldered dwarf with an
> iron-grey beard, an immaculate uniform, and a magnificent
> double-headed battle axe resting casually across one shoulder.
>
> **Sergeant Major Brannock Ironheel.**
>
> Veteran of more campaigns than anyone cares to remember, Brannock has
> trained generations of adventurers for the service of Trebor. His
> medals are polished, his boots immaculate, and his patience legendary
> for its absence.
>
> "So, you're the new commander, eh? Well, I've got fighters, priests,
> thieves, and mages for you to look over. Some promising specimens
> among 'em. Some I'd hesitate to trust with a spoon."
>
> "You choose 'em, Commander. I'll see they're properly prepared. But
> remember, a fine-looking recruit doesn't necessarily make a fine
> adventurer."
>
> He shifts his enormous double-headed axe.
>
> "Once you've selected your people, take 'em to Gilgamesh. He'll help
> you assemble your party. After that, what happens underground is your
> responsibility."
>
> He pauses, then adds:
>
> "And Commander? Try to bring 'em back. I hate training replacements."

The first line is the story's title (its `#` heading); the second, in
italics, is its own paragraph.

The story box behaves as at every place with a story (ART-3B-1): open on
the first visit of the session, closed there once closed, until Story
opens it again. While a character is being created, the View has no
story (`view.go`: only the roster screen has one), so the story box and
its Story button are not shown: creation's decisions, the attributes and
the class table are never under it.

## Unchanged

Character creation (the name and password, race, alignment, the bonus
points, the class table and its guidance, Keep and Discard), the roster
and its character sheets (renaming and deleting), navigation, Escape, the
emblems, the save format and the settings.

## Visual review

`docs/design/art3/review/art-3c-3d-castle/`.
