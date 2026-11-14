# Istary desktop GUI (prototype)

A Wails v2 + Svelte 5 front end. It covers Wizardry I character creation,
roster management, the party (Gilgamesh's Tavern), combat (M7a) and the
dungeon (M6a): real expeditions (M6b-1), practice expeditions and practice
encounters played through `app.Controller`, `app.DungeonView` and
`app.CombatView`.

**Getting around (Castle-0).** The window opens at the Castle, the hub, as
the original does. Each place has its own screen: its name and "← Castle"
at the top (or press Esc; it abandons only what is unfinished, such as a
character being created), its destinations ("Go to") and its actions apart, its
picture (a themed panel until artwork exists), and the party bar at the
bottom, shown or hidden with Hide party.
- **The Castle:** Gilgamesh's Tavern, the Adventurer's Inn, Boltac's
  Trading Post and the Temple of Cant (for a party), the Edge of Town.
- **The Temple of Cant (T-3):** every character the priests can help,
  in the party or not, with their condition and the donation; the LOST
  are listed, beyond help. Choose who tithes from the party bar (only a
  member who is well), then Treat…: if the payer is short, the screen
  offers the whole party's gold (as the guide says) and shows each
  member's share before anything is paid; the
  screen says what may happen and asks first (no chance is shown). A
  raise from the dead may fail into ashes; from ashes, failure is LOST
  for good; the donation is charged either way. Only characters at the
  castle can be treated. The receipt says what happened and whether it is
  saved.
- **The Adventurer's Inn (I-4):** click a party member to check them in
  (only a character who is OK may stay). Each room shows its rate and, for
  the guest, what a stay there does: Rest until healed (the original's
  stay: until healed or out of gold) or Choose weeks (leave partly healed).
  The stables are a free night that heals nothing. Every stay restores
  spell points and ends with a level check; the receipt shows gold, HP,
  spells, level and attribute changes, and whether it is saved (Try saving
  again if not). A room the guest cannot pay a week of is refused (the
  stables are free), and a guest who dies of old age cannot stay again.
  Pool party gold gives the guest everyone's gold. Istary
  does not yet age characters for Inn stays (the original's rate is not
  established).
- **Boltac's Trading Post (M8-4):** click a party member in the party bar
  to bring them to the counter (another click on someone else switches);
  Buy, Sell, Identify and Uncurse list Boltac's catalog and the customer's
  items with prices, fees, stock, who can use what and equipped, cursed and
  unidentified marks. A greyed button says Boltac's reason. Buying
  equipment the customer cannot use asks first, and so does selling
  something they wear. Each purchase, sale, identification or uncursing is
  saved at once, and the receipt says Saved, saved but perhaps not durably,
  or NOT saved (with Try saving again). Pool party gold gives the customer
  everyone's gold.
- **Character sheets** (click a party member or a roster character, or
  Character Sheet in the Actions): shown in place of the place's picture;
  Hide Character Sheet (or Escape) brings the picture back. Worn &
  Wielded and the Pack are one inventory of up to 8: Remove takes an item
  off into the pack, Use equips one from it, anywhere in town; a greyed
  button says why (the class cannot use it, a cursed item is stuck in its
  place...). The armour class changes at once. Cursed items, and items of
  another alignment, stick once worn: only Boltac can remove them. Not at
  Boltac's, where the Selected Character panel is the customer.
- **Gilgamesh's Tavern:** form the party: Join, Remove, and the password;
  Divvy gold shares the party's gold evenly.
- **Game selection** (UX-3A): Istary opens on six game slots, each its own
  adventure. New game in an empty slot, Continue a saved one, or Replace
  it (asked twice; the old game is moved aside, not deleted). A game saved
  by an earlier Istary is offered for import into an empty slot; the
  original is left as it was.
- **The Edge of Town:** Enter the maze; practice (Istary only); Change game;
  Leave the game; the way to the Training Grounds and to Settings.
- **The Training Grounds:** the roster (click a character for its sheet,
  where it can be renamed or deleted) and character creation: each
  decision in the Actions; the bonus points and the class on one screen,
  the classes the points allow highlighted once every point is spent;
  Discard at any step.
- **Settings:** the save folder, Reload, Reset roster and Restore roster.

Each action is accepted only at its own place; nothing moves the player
for an action.

**Real expeditions.** "Enter the maze" at the Edge of Town takes the party
itself into the maze, to Camp at the entrance. It plays like a practice
expedition (below), but everything is kept: the game is saved after every
action. It ends in the castle, by the stairs or with the party's defeat
(the engine returns the dead party there); it cannot be abandoned. If
Istary closes or crashes during an expedition, it resumes where it was last
saved; an encounter it was in is over and its unclaimed rewards are lost.
While the party is out, the roster, Tavern and Training Grounds wait. A
save that fails is shown until one succeeds; what failed to be saved stays in
the window, and Try saving again writes it without doing anything twice. Only one Istary window can use
a save folder at a time.

**Practice expeditions.** "Practice expedition" at the Edge of Town takes
copies of the party into the real maze, to Camp at the entrance.
- **Moving:** the buttons or the terminal's keys (F/W/↑ forward, L/A/←
  left, R/D/→ right, K kick, C camp, I inspect, Enter, Y/N, button letters).
  Doors need a kick, as in the engine.
- **Everything happens:** square effects, healing and poison, and
  encounters, exactly as in the terminal. An encounter opens the combat
  screen; after a victory the walk goes on.
- **Nothing is kept:** taking the stairs up, perishing or Abandon ends the
  expedition and discards it; the roster is not touched.
- **The screen** (`frontend/src/lib/DungeonScreen.svelte`) draws a simple
  wireframe of the corridor, not the Classic picture, with the party bar
  at the bottom; in Camp a click on a member there picks them.
- **Maps (M-1):** a minimap in the corner of the view shows the explored
  squares around the party (north up; a faint floor where the party has
  been, green walls, yellow doors, stairs and elevator symbols, the party's
  arrow); Hide minimap keeps the choice. Map
  or F10 opens the full map of an explored level; F10 or Escape closes it,
  and no key reaches the maze while it is open. Only what the party has
  seen is shown, each wall from the side seen; a real expedition saves the
  map with every action, practice discards it.

**Practice encounters.**
- **What you fight:** "Practice encounter" at the Edge of Town starts a real
  encounter from the maze entrance, with copies of the party.
- **Nothing is kept:** the results are discarded and the roster is not
  touched.
- **The chest option:** "the encounter guards a chest" adds a chest to try the chest screen.
- **The screen** (`frontend/src/lib/CombatScreen.svelte`):
  - order entry with the actions each member may take and why not;
  - the spell picker: every known spell, with a description area that
    follows the spell pointed at or focused, its casts left, and why it
    cannot be cast when it cannot (COMBAT-1);
  - Party retreat: one order for the whole party, reviewed before the
    round (COMBAT-1);
  - target choice and the order review;
  - playback one action at a time (Space or Enter, or automatic);
  - the chest, victory, defeat and escape;
  - an expandable "details" log.

All rules come from the Go engine through `app.Controller`. The Svelte code
only renders `View` (see `view.go`) and calls `Session` methods
(`session.go`); it holds no game state or rules of its own.

## Saves

The window's own preferences (whether the party bar and the minimap are
shown, and the illustrated places' appearance, chosen in Settings) are kept
in `gui-settings.json` in the save folder, never in a game. A damaged file is left
alone and the defaults are used until a setting is changed.

The GUI never uses the terminal game's `~/.config/wizardry`. It saves to
`~/Library/Application Support/Istary/dev-saves`, or to `$ISTARY_SAVE_DIR`
when set, and refuses to start if pointed at the terminal directory. Each
game is in `games/slot-N/` there (`docs/design/ux3/UX-3A.md`); the window's
preferences and lock stay in the folder itself.

## Build and run (macOS)

Requires Go, Node 20.19+/22.12+, Xcode command-line tools, and the Wails CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0`).

```bash
cd gui
wails build -platform darwin/arm64   # or `make gui` from the repo root
open build/bin/Istary.app

wails dev                            # live-reload development
```

This directory is a separate Go module (`github.com/Senior-Geek/istary/gui`)
with a `replace` back to the main module, so `go test ./...` at the repo root
and the terminal build do not depend on Wails. Run its tests with
`cd gui && go test ./...`.
