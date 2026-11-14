# COMBAT-1 — Spellcasting, spell descriptions and Party Retreat

Status: implemented on `combat-1-spells-retreat` (2026-11-14), from
v0.12.0-illustrated-castle; awaiting approval. Three rules questions are
left for decision (below); nothing in them was changed.

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
is a rules question, not a save to correct (decision 1).

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
| LATUMOFIS | No effect in combat in Istary: cast it in Camp (finding #41) |
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
  reason fields say so. MAPORFIC is described as protecting the party,
  although Istary applies its armour class the wrong way (finding #40).

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

## Decisions requested

1. **Known spells after loading (#32).** Today every restart teaches all
   the spells of each level with casts. Options:
   - **a.** Keep it (current).
   - **b.** Run the migration once, for saves from before known spells
     were stored. Saves made since then keep exactly what was learned.
     Characters who have already gained spells this way (Elenara's MOGREF
     and DUMAPIC) would keep them unless a reset is also wanted. The guide
     says knowing one spell of a level eventually teaches all of it (p.9),
     so this mostly changes when, not whether.
2. **When casts come back (#10).** Options:
   - **a.** At the Inn only (current; the original's code).
   - **b.** Also on entering the maze from the castle (the guide's
     wording). That would have given Elenara her casts on the second
     trip, and would make the game easier.
3. **Run (#42).** Party Retreat keeps Run as it is. Options:
   - **a.** Keep it (current).
   - **b.** Follow the guide: one attempt before the round, and on failure
     the monsters get a free round. The chance needs a source from the
     original before it is changed.
   - **c.** Keep it, but count a single attempt per round however many
     members are ordered to Run. Today six runners get six tries.

Also found, and **not corrected** (each a combat-balance change, so each
needs approval):

- **#40:** MAPORFIC worsens the party's armour class by 2 instead of
  improving it.
- **#41:** LATUMOFIS in combat says "POISON CURED!" and cures nothing.

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

## Remaining limitations

- **The combat screen is taller than the window** at both 980×692 and
  1440×900 (1056 px, as before COMBAT-1). The picker, the description area,
  Review orders and Party retreat are in view without scrolling at both
  sizes; the party strip below the commands needs a scroll.
- **The three decisions above** are not implemented. Known spells still
  grow on loading, casts return only at the Inn, and Run is unchanged.
- **The runner is chosen by agility.** A player who wants another member to
  run gives the orders by hand, as before.
- **The descriptions are Istary's,** checked against its engine, not the
  original's manual. Where the two differ (#40, #41), the description gives
  the spell's purpose.
