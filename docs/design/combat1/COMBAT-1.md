# COMBAT-1 — Spellcasting, spell descriptions and Party Retreat

Status: implemented on `combat-1-spells-retreat` (2026-11-14), from
v0.12.0-illustrated-castle. Approved 2026-11-14, with Option A for existing
player data; awaiting the full regression gate and release approval.

## Stage A — investigation

### Elenara: four known spells, 0/2 casts

Reproduced with controlled fixtures (`app/combat1_test.go`,
`gui/combat1_test.go`) and read from a copy of the player's game (never the
game itself).

**Creation is correct.** A new mage or bishop has two level 1 mage casts
and knows HALITO and KATINO. A new priest has two level 1 priest casts and
knows DIOS and BADIOS. Each starts with every cast it has (2/2), as the
original does (ROLLER KEEPCHYN; `engine.FinalizeCharacter`). Samurai and
lords have no spells until level 4, and the other classes none.

**The four known spells come from loading** (finding #32, known since
M6b-3A). On every load, `engine.Load` clears each character's known spells,
and `MigrateSpellKnown` marks every spell of each level with casts as known.
So a first-level mage knows HALITO and KATINO when made, and after the
first restart all four level 1 spells: HALITO, MOGREF, KATINO and DUMAPIC.
That is Elenara's list. It gives spells, never takes casts away.

**The 0/2 was not caused by creation, loading, or the combat list.**

- The game's history shows it:
  - `roster1.json`, saved before the game was moved into the game library
    (12 Nov, 19:21), already has the party at 807 experience each, its
    level 1 map partly explored, Thorgrim at 6/8 and Thalion at 8/16 hit
    points.
  - In that save, Elenara and Boddin (the bishop), the two casters who had
    been out, both have 0 of 2 casts. Nimble, a bishop who had never left
    town, has 2 of 2.
  - So the casts were spent on an earlier trip into the maze. The party then
    set out again without an Inn stay, as the unhealed hit points also show.
- **Casts come back only at the Inn,** in any room including the stables
  (finding #10). The Players Guide says they reset "each time you enter the
  Maze" (p.9); Istary follows the original program's code, where SETSPELS
  runs at the Inn.
- **A cast is spent only when the spell is cast,** at the caster's turn in
  the round, after the checks for sleep, paralysis and silence. It is not
  spent when the spell is ordered, taken back, disrupted or refused.
  A spell that fizzles in an anti-magic area is spent, as in the original
  (COMBAT4 line 776).
- **The combat list filtered correctly,** but left out every spell without
  casts and gave no reason. A caster with none saw an empty list and "No
  spell can be cast now." That defect is what the player met; it is
  corrected.

**Other classes:** the same holds for priests, bishops, samurai and lords;
the list and the casts are per book and level.

**Existing saves need no correction.** No character holds a wrong
allowance: 0/2 is the true state of a caster who has cast twice since the
Inn, and a stay restores it. Known spells grow with each load (#32). That
is a rules question, not a save to correct (decided in Stage C).

### The existing escape

- **Run is a member's order** (R in the terminal, Run in the window). It is
  tried at that member's place in the round's initiative.
- **The chance is (50 + 2 × agility)%**, for that member: 74% at
  agility 12, 86% at 18 (`engine/combat.go`, ActionRun).
- **Success ends the fight for the whole party at once.** It says "THE
  PARTY FLED!" and the party is back in the maze. Members who have not yet
  acted, and monsters, do not act. No experience or gold is given.
- **Failure costs that member's turn** ("X FAILED TO RUN!"). Everyone else,
  monsters included, acts as ordered, and the next round follows.
- **Ordering several members to Run** gives several tries in one round.
- **No encounter refuses Run.** Istary has no inescapable encounter, so
  none can be bypassed.
- **Who can run:** only members who can be given an order. That means OK,
  asleep and afraid; asleep members act (#16). Dead, paralysed and stoned
  members get no order, and on a success they go with the party. A runner
  slain or paralysed before their turn does not run.
- **The guide differs:** "If you don't [succeed], then the monsters get a
  free round of attacks" (p.47). The engine's chance has no Pascal source
  cited, unlike the rules around it (finding #42).

## Stage B — implementation

### The spell list in combat

Cast opens a list of **every spell the caster knows**, in level order. Each
entry shows its level and casts left (L1 mage · 2/2). A description area
beside the list describes the spell pointed at or focused (the first
castable one to begin with), at once:

- the spell's name and its book and level;
- what it does, in a sentence;
- what it is cast on ("one monster (choose its group)", "the caster", …);
- the casts left at its level ("2 of 2");
- whether the caster can cast it now, and if not why. For example, "ELENARA
  cannot cast HALITO: no level 1 mage casts left (0 of 2); a stay at the
  Inn restores them."

A spell that cannot be cast is listed, dimmed and described; clicking it
orders nothing. The Spell order notes "no spell can be cast now: see why"
when none can. The reasons, in order:

1. the class does not cast that book ("A mage cannot cast priest spells.");
2. no effect in combat (below);
3. no casts at its level, either left or yet ("No level 3 priest casts
   yet.").

**Spells with no effect in combat** are listed and not offered. In Istary a
cast would spend a point for nothing, as Camp already treats such spells
(#33). Each one's reason:

| Spell | Reason |
|---|---|
| DUMAPIC | Exploration only: cast it in Camp. |
| CALFO | For chests: offered when a chest is found. |
| KANDI, DI, KADORTO | No effect in combat. ("SPELL DISRUPTED" in the engine; DI and KADORTO have no effect in Camp either, #33) |
| LOKTOFEIT, MALOR | Not available in combat until their defect (#24) is corrected. |

The engine and the terminal are unchanged: there these can still be cast.

### One description per spell

- **The text lives in Go** (`app/spells.go`): all 50 spells, one sentence
  each, written from what Istary's engine does. A test reads every damage
  and healing range against the engine's dice, and every armour-class
  change against its modifier.
- **Every screen reads it from there:**
  - combat, on each spell in the view;
  - Camp, beside the spells that can be cast;
  - the character sheet, through `Session.SpellGuide`, loaded once when the
    window opens. Each known spell has its book, level and description.
  No screen writes a description of its own.
- **Descriptions say what a spell is for.** Where Istary differs, the
  reason fields say so. (MAPORFIC's and LATUMOFIS's combat defects, #40
  and #41, were corrected in Stage C, and LATUMOFIS is offered in combat
  again.)

### Party Retreat

**Party retreat** sits beside Review orders during order entry. One click
orders Run for the most agile member who can act (the earlier in the
formation on a tie) and Parry for everyone else who can act. It replaces
any orders already given, then shows the review. The button's note names
the runner and says escape is not certain. The review says:

> The party attempts to retreat: AGUI will try to lead it away. If that
> succeeds the fight is over at once; if it fails, that turn is lost and the
> fight goes on.

**Change orders takes it back.** In the round, the attempt is said ("The
party attempts to retreat!"), then the engine's line, then the outcome:
"The party escapes!" or "Escape failed — the fight goes on." The outcome
comes from the engine's own event (`escape` on the event view).

**It is only orders a player could give by hand:**

- the same chance, at the same point in the round;
- the same cost of failure;
- the same rule for who can run.

A test plays 80 seeded rounds both ways and finds the same lines and the
same outcome every time. **Choosing the most agile member** is what a
player choosing the runner by hand would do, since agility sets the chance.

### Not changed

- No rule of the game.
- No save field. `Session.SpellGuide` reads no game state.
- Practice writes nothing.
- No animation or sound.
- Combat's other screens are unchanged.

## Stage C — decisions applied (2026-11-14)

### #32: loading no longer teaches spells

- **`engine.Load` keeps each character's saved known spells,** when they
  fit the character's spell points: some are known, and each is of a book
  and level at which the character has points (`SpellKnownFits`). The game
  keeps that true (SETSPELS gives at least one point per spell known at a
  level), so any save Istary has written passes.
- **Only what cannot be trusted is rebuilt** from the points, as every
  load did before:
  - a character with no known spells saved (the oldest saves, imports);
  - known spells that do not fit, as spells saved in the table's order
    before it was reordered to Pascal SPELLSKN[1..50] would generally not.
- **Save format and compatibility:** unchanged. No field was added. An
  earlier Istary reading a new save rebuilds known spells as it always did.
- **A new mage knows HALITO and KATINO** after any number of restarts. A
  learned level 2 spell, or a spell kept through a class change, stays.

### Existing player data: migration options (not applied)

Saves written by earlier Istary already hold the widened spells. Every
load wrote them back, so the save cannot tell a learned spell from one the
load gave. The fix keeps them exactly as saved; **no player's data was
changed.** In the game played on 13 November, three characters are
affected:

| Character | Class, level | Known | Made with |
|---|---|---|---|
| ELENARA | Mage 1 | HALITO, MOGREF, KATINO, DUMAPIC | HALITO, KATINO |
| BODDIN | Bishop 1 | the same | HALITO, KATINO |
| NIMBLE | Bishop 1 | the same | HALITO, KATINO |

This matters beyond the list: SETSPELS gives at least one point per known
spell, so at level 2 a mage knowing four level 1 spells has 4 level 1
casts where one knowing two has 2.

Options:

- **A. Leave them as saved** (current). Nothing is changed. The guide says
  a character who knows one spell of a level eventually learns them all
  (p.9), so the difference is in when, and in the extra casts until then.
- **B. A one-time, per-character correction, for level 1 characters
  only.** A level 1 character cannot yet have learned a spell (TryLearn
  runs at a level-up), so their legitimate spells are their class's
  creation set. Exceptions are class changes and the Rite of Passage, which
  a save does not always record. The correction would:
  - be a separate tool or prompt, run only with approval;
  - back up the game first (as restores do);
  - list every change before applying it.

  For this game it would remove MOGREF and DUMAPIC from the three above.
- **C. Leave higher-level characters alone in any case.** Their learned
  spells cannot be told from the given ones. A reset there would remove
  spells legitimately learned, so it is not proposed.

**Decided 2026-11-14: Option A.** Existing player data is left unchanged
and no correction or migration is made. The extra spells of ELENARA,
BODDIN and NIMBLE are accepted as legacy data.

### #10: casts come back at the Inn only

- **Unchanged rule:** any room, the free Stables included, restores every
  point (`RestoreSpells`); entering the maze does not.
- **The Inn now says so:**
  - every room's rate ends "restores spells" (the Stables: "Free · heals
    nothing · restores spells");
  - the note under the rooms says every stay, the Stables included,
    restores all spell points, and that they come back only at the Inn and
    not on entering the maze;
  - a guest with points used has "Spell points used: any stay, the free
    Stables too, restores them all." under their points, now given by
    level ("L1 0/2").
- **The receipt reports by book:** "Spell points restored: Mage L1 0→2",
  "Spell points: already full", or "Spell points: none (no spells yet)".

### #42: Run unchanged

Run and Party Retreat are as in Stage B: no free monster round, no change
of chance.

### #40: MAPORFIC

- **Before:** it added 2 to each member's combat armour-class modifier,
  which the monsters' to-hit (20 − AC − level + modifier) reads as 2
  worse.
- **Now:** it subtracts 2, the party's armour class 2 better, applied as
  MOGREF's and MATU's are. That is the original's ACMOD2 := 2 "BIG SHIELD"
  (guide p.70).
- **Still for the fight only.** The guide has it last the expedition; that
  is not changed here.

### #41: LATUMOFIS

**In combat it cures the target member's poison,** as in Camp (guide p.70:
"removes the effects of poison", any time, one person). The message is
unchanged ("POISON CURED!"). It is offered in the GUI's combat list again,
for a party member.

## Tests

- **Go (`app/combat1_test.go`):**
  - new characters' casts and known spells, every class;
  - the combat list: castable spells and their targets, DUMAPIC, an
    exhausted caster (every reason, the Spell order's note, an order still
    refused), a bishop without priest casts, a mage knowing a priest spell,
    a fighter who once was a mage;
  - a cast spends one point when cast and not when ordered, taken back or
    disrupted (40 seeds, both seen);
  - the stables restore every cast, and the spell is castable in the next
    fight;
  - every description against the engine's dice and armour classes, the
    guide in table order, every combat note naming a spell;
  - Party Retreat's orders: the runner, Parry for the rest, nothing for the
    dead and paralysed, replacing earlier orders, at confirmation, taken
    back, refused with nobody to run;
  - Party Retreat played against the same orders by hand, 80 seeds, both
    outcomes seen, with the escape on the event.
- **Go (`gui/combat1_test.go`):**
  - a mage made at the Training Grounds: 2/2 through a restart and on the
    first expedition, HALITO, MOGREF and KATINO castable and described,
    DUMAPIC refused with its reason and no point spent;
  - Party Retreat through the bridge, refused in town and at confirmation,
    and a practice retreat writing nothing;
  - a save from before COMBAT-1 opening, and nothing rewritten.
- **Frontend (`combat1.test.js`):**
  - the picker's helpers;
  - the review's and the outcome's words;
  - the recorded views: every spell described, a reason exactly when it
    cannot be cast, retreat offered at order entry;
  - the sheet's descriptions from the guide, and no screen writing its own.
- **Live, suite `combat`** (level 2, whenever a change reaches combat), at
  980×692 and 1440×900, and by hand at 1512×982 and 1728×1117:
  - the mage's list;
  - the description panel, changing when a spell is pointed at and when it
    is focused;
  - the picker in the window without scrolling, nothing overlapping, cut
    off or covered;
  - the priest's CALFO with its reason, ordering nothing;
  - the mage after spending both casts (0 of 2, every reason, the Spell
    order's note);
  - Party retreat in the window, its review and its outcome;
  - the sheet's descriptions;
  - nothing saved.
- **Existing tests changed:**
  - `TestCombatViewSpellsAndItems` now expects the spell without casts,
    with its reason.
  - The GUI drivers give only castable spells, as the window does.
  - `TestRealExpeditionsThroughTheBridge` plays seeds until a win is seen,
    at least 12 and at most 60. Without CALFO in combat, the first twelve no
    longer had one. It checks everything it did before.
  - The camp, combat and dungeon fixtures were regenerated. They have the
    new fields, and in combat the orders that follow from CALFO not being
    offered. With CALFO allowed, only the new fields differ.

## Tests added in Stage C

- **`engine/combat1_test.go`:**
  - known spells through three reloads: a new mage and priest, a bishop
    with a learned level 2 spell, a fighter keeping HALITO;
  - the rebuild only for none saved or ones that do not fit;
  - MAPORFIC 2 better for every member, beside MOGREF;
  - LATUMOFIS curing its target only.
  Each fails with its correction undone.
- **Golden:** `save_roster1.resaved.golden.json` now keeps ALDRIC's DIOS
  and BADIOS. Before, the resave widened them to every level 1 and 2
  priest spell. Nothing else in it changed.
- **GUI and front end:**
  - a reload keeps a new mage's two spells (`TestCreateSaveAndReload`, and
    the rename test now compares the spells too);
  - the first expedition with HALITO and KATINO only;
  - the Inn's rates, guest note and receipt (`innViews.test.js`).
- **Live `combat`:** the library's mage knows HALITO and KATINO only.

## Remaining limitations

- **The combat screen is taller than the window** at both 980×692 and
  1440×900 (1056 px, as before COMBAT-1; in the roadmap's backlog). The
  picker, the description area, Review orders and Party retreat are in
  view without scrolling at both sizes; the party strip below the commands
  needs a scroll.
- **Widened known spells in existing saves stay** (Option A), as legacy
  data.
- **MAPORFIC lasts one fight,** not the expedition: in the roadmap's
  backlog, to investigate against the original.
- **The runner is chosen by agility.** A player who wants another member to
  run gives the orders by hand, as before.
- **The descriptions are Istary's,** checked against its engine, not the
  original's manual.
