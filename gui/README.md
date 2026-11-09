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
- **The Castle:** Gilgamesh's Tavern, the Adventurer's Inn and Boltac's
  Trading Post (for a party), the Edge of Town; the Temple of Cant is shown
  closed (not in the window yet).
- **The Adventurer's Inn (I-4):** click a party member to check them in
  (only a character who is OK may stay). Each room shows its rate and, for
  the guest, what a stay there does: Rest until healed (the original's
  stay: until healed or out of gold) or Choose weeks (leave partly healed).
  The stables are a free night that heals nothing. Every stay restores
  spell points and ends with a level check; the receipt shows gold, HP,
  spells, level and attribute changes, and whether it is saved (Try saving
  again if not). Pool party gold gives the guest everyone's gold. Istary
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
  or NOT saved (with Try saving again). Pool Gold is offered at the Inn,
  not yet here.
- **Character sheets** (click a party member or a roster character):
  Equip and Take off for each item, anywhere in town; a greyed button says
  why (the class cannot use it, a cursed item is stuck in its place...).
  The armour class changes at once. Cursed items, and items of another
  alignment, stick once worn: only Boltac can remove them.
- **Gilgamesh's Tavern:** form the party: Join, Remove, and the password;
  Divvy gold shares the party's gold evenly.
- **The Edge of Town:** Enter the maze; practice (Istary only); Leave the
  game; the way to the Training Grounds and to Settings.
- **The Training Grounds:** the roster (click a character for its sheet,
  where it can be renamed or deleted) and character creation.
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

**Practice encounters.**
- **What you fight:** "Practice encounter" at the Edge of Town starts a real
  encounter from the maze entrance, with copies of the party.
- **Nothing is kept:** the results are discarded and the roster is not
  touched.
- **The chest option:** "the encounter guards a chest" adds a chest to try the chest screen.
- **The screen** (`frontend/src/lib/CombatScreen.svelte`):
  - order entry with the actions each member may take and why not;
  - target choice and the order review;
  - playback one action at a time (Space or Enter, or automatic);
  - the chest, victory, defeat and escape;
  - an expandable "details" log.

All rules come from the Go engine through `app.Controller`. The Svelte code
only renders `View` (see `view.go`) and calls `Session` methods
(`session.go`); it holds no game state or rules of its own.

## Saves

The window's own preferences (whether the party bar is shown) are kept in
`gui-settings.json` beside the roster, never in it. A damaged file is left
alone and the defaults are used until a setting is changed.

The GUI never uses the terminal game's `~/.config/wizardry`. It saves to
`~/Library/Application Support/Istary/dev-saves`, or to `$ISTARY_SAVE_DIR`
when set, and refuses to start if pointed at the terminal directory.

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
