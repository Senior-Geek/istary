# ART-3B-1: Gilgamesh's Tavern

**Baseline:** v0.16.1-navigation-emblems (`9cb83be`). **Branch:**
`art-3b-1-tavern`.

The Tavern is illustrated, has its own story, and the story box is each
place's own.

## The artwork

| | |
|---|---|
| Master | `images/Tavern/a_highly_detailed_warm_cinematic_fantasy_tavern.png`, 1672×941, 16:9 to within 0.1% |
| Runtime copy | `gui/frontend/src/assets/scenes/tavern.webp`, 1920×1080, 439 KB |
| Conversion | `cwebp -q 82 -m 6 -sharp_yuv -metadata none -resize 1920 1080` (ART-2A's) |
| Focus | `50% 50%` |

- **The master is kept outside Git**, unchanged, as ART-2A's are:
  `/images/Tavern/` is excluded locally in `.git/info/exclude`, so a
  checkout stays clean for the acceptance level and the gate. Nothing is
  added to `.gitignore`.
- **Recomposed for the panels** (ART-3B-1A, approved in ART-3B-1B): the
  first Tavern picture (`images/Artwork Samples/Tavern.png`) had
  Gilgamesh and the sign under the Actions panel at both window sizes.
  In this one they are in the middle of the picture, where the display
  shows them at 980×692 and 1440×900 with the story box open or closed:
  Gilgamesh's face about 45% across and 31% down, the sign 40–57% across.
- **Not under the panels:** the hearth, the near table and the left of
  the gallery are behind an open story box, and the lute player is
  cropped at 980×692; all are seen with the story box closed or the
  panels hidden. The bar's right end and the barrels are under the
  actions.
- **Upscaled** about 1.15×, like the Castle's and the Edge's: somewhat
  soft on a large Retina full screen. A larger master would need no
  change to the code.

## The story

`gui/content/tavern/story.md`, the Tavern's (`storyFolders` in
`gui/story.go`). The approved text (2026-11-15), Claude's first draft
with two revisions (the alignment paragraph and the challenge):

> **Gilgamesh's Tavern**
>
> The door swings shut behind you, and the chill of the courtyard is
> gone. Firelight plays across old timbers. Somewhere a lute is being
> played, not quite in tune, and nobody seems to mind.
>
> Every table is taken. Fighters compare scars while a young mage
> pretends not to listen. In the gallery above, a priest blesses a bowl
> of stew, and the thief beside him quietly counts the spoons.
>
> Behind the bar stands a broad, gray-bearded man with his arms folded
> and a smile that has outlasted a great many customers.
>
> "Welcome, welcome," says **Gilgamesh**. "Another one answering
> Trebor's call, I see. Sit anywhere you like. Not that table, though.
> It's kept for a party that went down to settle matters with Werdna."
> He polishes a tankard thoughtfully. "Some while ago now."
>
> He nods toward the crowded room.
>
> "Everyone here means to go down into the Maze, and the wise ones know
> better than to go alone. Choose your companions with care. A friend
> who will carry you home is worth more than gold. A friend who offers to
> carry your gold is not quite the same thing."
>
> You notice that good and evil seldom share a table for long. Gilgamesh
> assures you they fare even worse sharing a corridor beneath the castle.
>
> "Well, then," says Gilgamesh, setting down the tankard. "Find yourself
> some good company. The Maze will still be there in the morning. It
> always is."
>
> **Your adventure begins with good company. Gather your party.**

The first line is the story's title (its `#` heading); the last is its
challenge (`> [!challenge]`), in the story box's gold.

## Each place's story box

Before, the story box was open or closed everywhere at once: closing the
Castle's left the Edge's closed, and now would have hidden the Tavern's
on its first visit. Now it is each place's own (`App.svelte`):

- a place's story box is open when the player first comes there;
- closed with its ×, it stays closed there, coming back included, until
  Story opens it again;
- closing one place's leaves every other place's as it was;
- where the box stands and its size are still one, the player's, at every
  place;
- a game opened or closed opens them all again (NAV-1A), so a game's
  Castle shows its own story.

**For the session only.** Which places' boxes are closed is the window's
state, like the hidden panels; nothing is written to the game or to
`gui-settings.json`, and Istary opens with every story box open, as
before. Keeping it would add to the settings file's schema, for a
setting the player cannot see; it is left until there is a need.

## Unchanged

The rules: the Tavern's joining, removing and alignment, Divvy Gold, the
roster and the six-slot party bar; the Castle's and the Edge's pictures
and stories; the appearances (Full, Subdued, Hidden), Hide panels, the
sheet's priority over the story box (ART-3B-0), Escape, navigation and
the emblems.

## Tests

- **Go:** the story files (the Tavern's among them), and the Tavern's
  story kept to the approved text above, word for word, with its title,
  bold and challenge (`story_test.go`); the frame fixtures.
- **Frontend:** the Tavern's picture (a 1920×1080 WebP, focus 50% 50%);
  the Tavern's story in the View and drawn; the story box's state kept
  for each place and reset with the game.
- **Live (`scenes`):** the Tavern with the Castle and the Edge as an
  illustrated place (its picture, panels, Hide panels, Hide party, the
  larger and smaller windows, all 27 appearances); the Tavern's own story
  on its first visit after the Castle's was closed; closed, not opened
  again on return; Story opens it; the Castle's and the Edge's kept
  apart. The Inn is now the place without artwork the suite checks.
  Three checks written for the Castle and the Edge now allow for an
  illustrated place with a long story and a long roster: the picture is
  seen through the display at three or more of its centre and corners,
  beside the story box; Hide party's room goes to the actions or to the
  information (the Tavern's roster at 980×692, as ART-3B-0 intends); and
  the themed panel is checked only at the places without a picture.
