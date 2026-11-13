# ART-2B — Panel appearances and the story box

Status: implemented on `art-2b-appearance` (2026-11-13), for comparison
before a default is chosen.

- **Built on ART-2A** (`art-2a-castle`), which is not yet merged; the
  branch holds both.
- **Presentation only:** no rule of the game, no navigation, and nothing
  saved changes.

## Appearances

Three appearances for the illustrated places' panels, chosen with the
temporary **Appearance** menu in the head:

| Appearance | Panels | Text | Accents |
|---|---|---|---|
| Dark | charcoal, 92% opaque | pale grey | gold edges; olive main action |
| Light | warm parchment, 94% | dark brown | gold-brown edges and titles; pale olive main action; drawn button edges |
| Slate | blue-grey, 92% | pale blue-white | the heraldry's gold for edges and titles; olive main action |

- **Only colours change.** Each appearance is a set of colour variables on
  the frame (`style.css`, `[data-appearance]`): the panels' own (`--glass`,
  `--glass-line`, `--glass-shadow`) and the game's ordinary ones as used
  inside the panels.
  - The layout, the controls, the type sizes and the pictures are the same
    in all three.
  - A test reads the CSS and refuses anything but a colour there. The live
    check finds every element in the same place in each appearance.
- **Contrast:** all text reads at WCAG AA or better in each appearance:
  4.5:1, or 3:1 for large text.
  - It is measured live, with every fill and opacity between the text and
    the picture composited over black and over white, the picture's
    extremes.
  - The check was shown to fail with one colour made faint.
- **For the session only:** the choice holds from place to place. It is
  not saved; the default is Dark. Places without artwork are unchanged.
- **Reusable:** any illustrated place takes the appearance. The Tavern,
  Inn, Trading Post, Training Grounds and Temple will once their pictures
  are set (`scenes.js`).

### Comparison

- **Dark** stays closest to the rest of the game (the maze, combat, places
  without artwork). It recedes most, and leaves the picture the most
  colourful thing in the window. Its edges against the picture are the
  softest.
- **Light** reads most like a book: the parchment and brown suit the
  courtyard's timber and signboards, and the story box looks like a page.
  - It is the brightest, and takes the most attention from the picture,
    most at 980×692, where the panels cover much of it.
  - Its buttons need their drawn edges to stand apart from the panel.
  - It differs most from the unillustrated places, which are dark.
- **Slate** matches the banners' blue and gold, and sits well over the
  stone and sky of both pictures.
  - It is as readable as Dark, and slightly cooler than the warm pictures.
  - Its blue tint is clearest in the large full-screen windows.

## The story box

A place's narrative is in a small window over the picture (`StoryBox`), in
place of the descriptive panel:

- **Where it opens:** in the lower left of the display area, where there
  are no controls. It is as tall as its text and about a short line wide.
- **Moving and resizing:** drag its title to move it and its corner to
  resize it. With the title focused, the arrow keys move it by 16 px, and
  Shift with them resizes it; Alt makes the step 1 px. It is always kept
  inside the window, at least 220×90, and its text scrolls if it does not
  fit.
- **Closing:** × closes it, and **Story** in the head opens it again. The
  focus goes to Story when it closes.
- **For the session:**
  - It is open at first.
  - Once closed, it stays closed at every place until Story is pressed.
  - Once moved or resized, it stays where the player left it.
  - None of it is saved.
- **Hide panels hides it** with the rest.

### What is in it

Only the description the places already had, divided between story and
what the player needs to play:

- **The Castle:** its paragraph ("The party is formed at Gilgamesh's
  Tavern…") is description; the destinations say the same. It is the
  story, and the Castle's information area is now empty, which shows more
  of the picture.
- **The Edge of Town:** only "**The maze.** The party sets out from here."
  is narrative.
  - Saving and resuming, and the practice explanation, stay in the
    information area, story box open or closed. So do the practice label,
    the equipment check and the reasons for closed departures, which were
    always in the actions.
  - The saving paragraph is headed **Saving.** and says "the party in the
    maze" where it said "it". These are the only changes to its wording.

No new story or lore was written: the boxes are short until there is
narrative to put in them.

## Tests

- **Frontend (node):**
  - the appearances are colour-only, 90–95% opaque, and offered from the
    frame;
  - nothing about them or the story box is sent to Go;
  - the Castle's description is its story;
  - the Edge's story has no saving or practice, and those are still in its
    information.
- **Live, suite `scenes`** (part of level 2), at both sizes:
  - the story box's placement, dragging and resizing (mouse and keys), its
    limits, closing, staying closed across places, reopening where it was
    left, and Hide panels;
  - for each appearance at both places: the layout unchanged, the opacity,
    the contrast, every control uncovered;
  - the appearance holding across places, and a place without artwork
    unchanged.
- **Live, suite `looks`:** the comparison screenshots (`docs/TESTING.md`).

## Remaining limitations

- **Short stories:** the Edge's story box is one sentence, and the
  Castle's restates the destinations. Both are short until narrative is
  written; this milestone does not invent any.
- **Over controls:** the story box can be dragged over the panels and
  their controls; it stands above them. It opens clear of them, and closes
  with one click.
- **At 980×692:** the Edge's actions still scroll (ART-2A). The story box
  covers part of the picture's lower left, the Training Grounds' sign.
  Closing it shows the sign.
- **The rest of the game is dark.** The Light and Slate appearances apply
  only to the illustrated places. Banners (errors, notices), the
  appearance menu's open list (drawn by the system) and places without
  artwork stay as they were.
- **The Appearance menu is temporary.** Once a default is chosen it should
  go, or move to Settings as a saved preference. Saving it would be a
  settings change, not a save-file one.
