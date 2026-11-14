# ART-2C — Appearance preferences

Status: implemented on `art-2c-preferences` (2026-11-13), awaiting
approval.

- **Built on ART-2B** (`art-2b-appearance`), itself on ART-2A, neither yet
  merged; the branch holds all three.
- **Presentation only:** no rule of the game, no navigation and no game's
  save changes.

## What the player chooses

In **Settings** (from the Edge of Town), under **Appearance**, three
choices for the places with pictures, the Castle and the Edge of Town:

| Part | Choices (default first) | What changes |
|---|---|---|
| Panels | **Slate**, Dark, Light | the working panels' colours: the head, the actions, the information, the party bar |
| Story box | **Parchment**, Dark, Slate | the story box's colours, its own whatever the panels' |
| Picture | **Full**, Subdued, Hidden | the picture behind them |

- **Each takes effect at once** and holds at every illustrated place.
- **Story and Hide panels stay in the head** of the illustrated places.
  ART-2B's temporary Appearance menu there is gone.
- **Places without artwork are unchanged.** That includes Settings itself,
  the other places in town, the maze, combat and the game selection. Their
  colours and layout are as before, and so is every control in them.

### The appearances

- **Colours only.** Panels and story box take their colours from the same
  sets (`style.css`, `[data-appearance]`). Parchment is the Light set under
  the story box's name.
- **Each set is complete:** every colour the game uses inside a panel. Dark
  repeats the game's own (`:root`). So a Dark story box over Light panels
  takes nothing from them.
  - Before, Dark set only the panel fill and relied on the game's colours
    underneath. That was correct only while the story box and the panels
    always matched.
  - The live check was shown to fail with that version (see Tests).
- **Light buttons keep their drawn edges.** ART-2B drew them with a rule
  for buttons inside Light panels, which also reached a Dark or Slate story
  box inside them. The edge is now a colour each set gives (`--btn-edge`).
  The text field's colours (`--field`, `--field-line`) became colours of
  the sets the same way.
  - Outside the illustrated places these have the values the rules had, so
    nothing there changes.

### The picture

- **Full:** as in ART-2A.
- **Subdued:** the picture is darker and less saturated
  (`brightness(0.55) saturate(0.6)`), so it draws less of the eye.
  - The panels over it are unchanged: the same fills, opacity and text.
  - Contrast was already measured against the picture at its darkest and
    lightest, so subduing it cannot make any text harder to read.
- **Hidden:** no picture. The window shows the place's tint on charcoal,
  with no emblem.
  - The panels, the story box and Story are as with the picture.
  - **Hide panels is not offered**, since there is nothing to look at;
    Escape goes Back as at any place.
  - The display area describes no picture to screen readers.

## Where it is kept

- **In the window's settings file**, `gui-settings.json` in the library
  folder, with "Hide party" and the minimap's toggle (`gui/settings.go`).
  - It holds three optional fields, `panels`, `story` and `artwork`. A
    file without them means the defaults, and an unknown value is read as
    the default.
  - The format's version stays 1. An earlier Istary reads the file and
    ignores the new fields.
- **The same in every game, and in none of them.** No save, checkpoint,
  backup, reset or restore reads or writes it. Changing games leaves it as
  it is.
- **`Session.SetAppearance(panels, story, artwork)`** saves all three at
  once:
  - an unknown value is refused, and nothing changes;
  - it draws no random number and changes nothing of the game;
  - if the file cannot be written, the choice holds until Istary closes,
    and a note says why.

### The story box

Whether the story box is open, and where the player left it, stay as in
ART-2B: the window's, for the session only.

- **Open at first.** Once closed, it stays closed at every place until
  Story is pressed. Once moved or resized, it opens there at both places.
- **Through Settings and back** it keeps its place, open or closed.
- **Not kept across restarts.** Its place depends on the window's size at
  the time. If it should be kept, that is a settings field like the
  others.

## Tests

- **Go (`gui`):**
  - `TestAppearanceIsKept`:
    - the defaults until one is chosen;
    - each choice saved in `gui-settings.json` exactly, and nowhere else in
      the save folder;
    - the view otherwise unchanged, and no random number drawn;
    - unknown values refused with the file untouched;
    - the choice kept across a restart (a new session on the same folder).
  - `TestAppearanceFromOtherFiles`:
    - a settings file from before ART-2C reads as the defaults, with Hide
      party kept;
    - unknown names read as the defaults.
  - `TestSettingsAreTheWindows`, extended: the appearance is the same in a
    new game and back in the first.
- **Frontend (node), `layout.test.js` ART-2C:**
  - the choices and their defaults match Go's lists;
  - every appearance sets every colour, and Dark's are the game's own;
  - no control is styled per appearance;
  - the frame has no appearance menu, and offers Hide panels only with a
    picture shown;
  - Subdued changes only the picture;
  - Settings sends the three parts in one call;
  - the story box is not sent to Go.
- **Live, suite `scenes` (level 2):**
  - the defaults on a fresh library;
  - Settings: each part clicked with the mouse, in effect at once and
    written to `gui-settings.json`;
  - another game and back: the same appearance, the game unchanged;
  - the story box open, where it was left, through Settings and between
    the places;
  - **all 27 combinations at the Castle and at the Edge**, each checked
    for:
    - being shown as chosen, the layout unmoved;
    - 90–95% opacity;
    - WCAG AA contrast for every text, over the picture's darkest and
      lightest;
    - every control in view, uncovered and enabled;
    - the page not scrolling.
  - **The check fails as it should:** with ART-2B's Dark set, which gives
    only the fill, it fails. A Dark story box over Light panels kept
    Light's dark text on a dark fill.
- **Live, by hand:**
  - `scenes` at the MacBook Pro full screens, 1512×982 and 1728×1117;
  - `restarted`, run twice on one library, for a real restart of Istary;
  - `looks`, for the screenshots.

## Remaining limitations

- **The story box can still be dragged over controls**, and covers part
  of the Edge's picture at 980×692 until it is closed or moved (ART-2B).
- **The appearance applies only to the illustrated places.** Settings,
  where it is chosen, is not illustrated, so a choice is seen on going
  back to the Castle or the Edge, not in Settings itself.
- **Banners** (errors and notices) and the rest of the game stay dark.
- **An earlier Istary that saves a setting** (Hide party) writes the file
  without the appearance fields, and this one then shows the defaults.
- **From ART-2A:** the upscaled pictures are soft on a Retina full screen,
  and narrow windows crop their sides.
