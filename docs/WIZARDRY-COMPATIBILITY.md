# Wizardry I compatibility audit

Status: **first pass, part of the design baseline** (2026-11-06). Read-only
audit; no code was changed. No discrepancy below has been decided yet.

This document compares the original Apple II *Wizardry: Proving Grounds of
the Mad Overlord* Player's Guide with the Go engine Istary inherited from
`sshoecraft/wizardry`. It records evidence, not conclusions about which side
is "right".

## 1. Sources

**Player's Guide (PG).** `docs/reference/Wizardry-I-Apple-II-Players-Guide.pdf`,
82 PDF pages, a scan with an OCR text layer. The copyright line reads
"1981-1987" (PDF p.2), and the guide mentions later scenarios up to *Heart of
the Maelstrom* (PG p.2). So this is a **late printing**, not the 1981 first
edition.

- **Page numbers** below are the printed page numbers: **PDF page = printed
  page + 6**. For example, PG p.21 is PDF p.27.
- **Faint page:** the second glossary page (PDF p.80, printed p.74) is too
  faint to read and was not used.
- **Pictures:** screen pictures in the guide were read only where the OCR
  made them legible.

**Engine.** The Go code at `gui-party-assembly` (68be3bd). The engine's
comments say it was built from analysis of the original Pascal source and
p-code. For example:

- `app/tavern.go` cites "Pascal ADDPARTY, CASTLE.TEXT lines 148-191";
- `engine/temple.go` cites "SHOPS segment p-code proc 23".

Several comments also cite changes by code, such as `WC030` and `WC014`:

- **`WC030`:** `engine/creation.go` `ClassRequirements`, "Ninja ... reduced
  from 17";
- **`WC014`:** `engine/combat.go`.

Where the `WCnnn` numbering comes from is **not documented in the
repository**, and I have not verified it.

**Consequence:** the guide and the engine may describe **different revisions**
of the program. "Difference" below means only that the engine does not do
what this printing of the guide says. Whether the original program did is a
separate question, listed under follow-up.

**How discrepancies are resolved (decision D1, amended).**

- **Neither source decides on its own.** Istary's Classic rules are a
  documented, historically verified Wizardry I ruleset. Neither this guide
  nor the inherited engine is automatically authoritative.
- **Provisional baseline:** the current engine is the *provisional
  executable baseline*, and existing rules are not changed while a
  discrepancy is open.
- **Open until decided:** every discrepancy stays open in this document until
  evidence supports a decision.
- **Recording a decision:** the decision is recorded in §6, with the
  historical revision, the evidence, the decision and any implementation
  change.

## 2. Status key

| Status | Meaning |
|---|---|
| **Match** | Guide and code agree on the points checked |
| **Difference** | The guide states X; the code clearly does not do X (code read, not yet run against the original) |
| **Presentation** | Same rule, different interface or wording |
| **Missing** | Guide feature with no implementation found |
| **Engine only** | Engine behaviour the guide does not describe (may be original; the guide is not exhaustive) |
| **Unclear** | Evidence is weak: the guide only illustrates, OCR is damaged, or the code path was not fully traced |
| **Not investigated** | Listed so it is not forgotten |

## 3. Most important findings

| # | Finding | Status | Why it matters |
|---|---|---|---|
| 1 | Ninja minimums: guide 17 in all stats (PG p.21); engine 15 (`engine/creation.go` `ClassRequirements`, comment "WC030: reduced from 17") | Difference | A key input to the Classic decision (D1); open |
| 2 | Poison: guide says poisoned characters are cured on return to the Castle (PG p.11, p.60); engine comment says "Pascal does NOT cure poison on return" (`cmd/istary/main.go`, stairs-up handler) | Difference | Survival rule; players following the guide will be surprised |
| 3 | Visiting the Training Grounds disbands a formed party (PG p.24); engine keeps the party (`app.Controller.EnterTrainingGrounds`; Edge of Town `T` in `cmd/istary/main.go`) | Difference | GUI-001 to GUI-003 show Training Grounds and team together. **D2 deferred** pending historical evidence; engine behaviour kept |
| 4 | Gold pooling: guide has P)OOL GOLD in inspect, Boltac's and the Inn, and D)IVVY GOLD in the Tavern (PG p.24, 28, 30, 32, 34); none found in the engine | Missing | The economy is much harder without pooling; Hawkwind's walkthrough depends on it (PG p.53–54, 61). **D3:** candidate missing Classic feature; investigate |
| 5 | Temple: if the payer is short, the whole party's gold is used (PG p.33); engine refuses with "CHEAP APOSTATES! OUT!" (`handleTempleInput`) | Difference | Linked to #4 |
| 6 | Dead or incapacitated characters may join the party after a confirmation (PG p.39); engine allows them with no confirmation (`app/tavern.go` `memberRefusal`) | Difference (prompt only) | Small; the GUI could add the prompt only if the rules decision says so |
| 7 | XP is "divided amongst the survivors" (PG p.6); engine divides by the whole party, dead included, and pays only the living (`engine/combat.go`, comment cites Pascal REWARDS) | Difference | Progression rate |
| 8 | Disarming a chest: you "type in the trap's name. Spelling counts." (PG p.49); no name prompt in the engine (`ChestWhoDisarm` → `CombatState.DisarmChest`) | Difference | Removes a skill element |
| 9 | Utilities "Restart an OUT party" (PG p.38, 39) not found; engine Utilities are Backup, Change Names, Import, Transfer (`render/utilities.go`) | Missing | Recovery from quit/interrupted expeditions. **D3:** candidate missing Classic feature; investigate |
| 10 | Spell points: the guide says they reset "each time you enter the Maze" (PG p.9); the engine restores them at the Inn, any room including stables (`cmd/istary/main.go` inn handler, `engine.RestoreSpells`), not on Maze entry | Difference (wording may be loose) | Expedition pacing |
| 11 | Inn stays: the guide implies weeks pass and age matters (PG p.34, 61); no aging found for Inn stays. Aging was found only for Temple, class change and camp disband | Unclear | Long-term character life |
| 12 | Shop stock changes (PG p.30) are kept in memory only; the save holds only roster and party (`engine/save.go` `SaveState`) | Unclear | Whether the original saved stock is not established |
| 13 | Some magic item prices in `scenarios/wiz1/wiz1_gamedata.json` look like mis-decoded multi-word numbers. Examples: LONG SWORD+1 65536, EVIL SWORD + 3 327680, SHIELD + 3 1638400 | Unclear | Possible data-import defect; the guide gives no prices to check against |
| 15 | Party re-sort and orders: the guide moves the dead (and incapacitated) to the back "at the end of the round" (PG p.48, p.55), so orders are given against the new lines. The engine sorts at the start of `ExecuteRound`, after orders were entered by slot, and does not move the orders with the members (`engine/combat.go:610`, `:199`). A scratch Go probe showed an order given for one member carried out by another. Skipped slots keep the previous round's order; ally targets are slots | **Difference** (timing; order transfer) | Combat outcomes can follow the wrong member. Istary corrections C1–C3 approved in principle (decision log); see `design/m2-exploration/COMBAT-DESIGN-BASELINE.md` §3 |
| 16 | A monster's melee writes one message block per attack, and a miss writes no line, so the terminal shows a blank pause (`engine/combat.go`, monster melee loop; P1 spike: 1,236 in 448 seeded rounds). Asleep and afraid members are offered orders and act (`findNextActor`, `IsAlive`); an asleep caster's spell is "DISRUPTED" | Unclear | The guide does not say whether misses were shown or whether sleeping members could be ordered. Pinned by tests on `spike/p1-combat-events`; see `design/m2-exploration/P1-COMBAT-ENGINE-SPIKE.md` |
| 17 | After a successful R)UN, `ExecuteRound` returns before its round-end step, so the combat stays at "PRESS [RETURN] TO FIGHT, OR GO B)ACK" (`engine/combat.go`, `if cs.Fled { return }`; terminal `CombatConfirm`). Return plays a further round; B)ack reopens order entry, possibly with every monster dead. Found by the M5a differential fuzz, present on `main` before M5a | Engine only (defect). **Corrected** (M5b, approved) | Was: the flight message was not shown and the party fought on. Now a successful escape ends the round normally (`checkRoundEnd`), its message is shown, and combat is left; a failed escape continues as before. See the decision log |
| 18 | U)SE aims an item's spell at a group only for group and all-monster spells; any other spell, including single-monster ones such as SCROLL/HALITO, is aimed at the user with no group (`cmd/istary` before M5a, now `app.SetOrder`) | Unclear | A HALITO scroll may do nothing when used in combat. Kept as found |
| 19 | HAMAN/MAHAMAN (Wiz 2/3): the engine resolves the whole round at once and opens the boon choice while doing so; the terminal then stops playback at the round's first block and accepts 1-3, so the prompt and its options may not be on screen when the choice is asked for, and the chosen boon takes effect after the round's other actions | Engine only (presentation and ordering) | Kept for the terminal. The controller's playback (M5d) runs up to the prompt's block before waiting, and `CombatView` shows the boons only there. Resolving the boon at its point in the round would need a resumable round, out of scope |
| 20 | CALFO's 5% failure: the engine's comment says Pascal re-enters the CALFO prompt; the engine shows "CALFO REVEALS NOTHING...", spends the spell and returns to the chest menu (`CalfoChest`). A wrong inspection report is drawn from every trap name, so it can name the true trap by chance | Unclear | Kept as found (M5e moved the chest to the controller without changing it) |
| 21 | The terminal's U)SE item list shows each item's true name, even for unidentified items (`render/combat.go`); `CombatView` uses the unknown name | Presentation (terminal) | The terminal shows the player something the item's identification should hide. Kept; the GUI must use `CombatView` |
| 22 | The buttons square says "PRESS ONE OR [RET]", but the maze handler returns on any non-letter key before it reaches the buttons prompt, so [RET] does nothing and the party must press a button (`app/maze.go`, `WaitButton`). The handler's [RET] branch is unreachable | Unclear (engine) | Kept as found (M6a moved the handler without changing it); the GUI offers the buttons only |
| 23 | The Classic view draws a hidden door without light one draw in six (`render/dungeon.go`, `shouldDrawDoor`), using the game's random numbers. The terminal redraws after every key, so its drawing draws random numbers the game then uses. `DungeonView` leaves the flicker out and never draws one, so the GUI's random sequence differs from the terminal's for the same keys | Presentation | No rule differs; the GUI shows hidden doors as walls without light. Kept |
| 24 | LOKTOFEIT and MALOR cast in combat set the combat to defeat ("exit combat — party returns to castle", and "exits combat"), but the round's end sets the phase back, so combat goes on: LOKTOFEIT takes every item and all gold and the fight continues; MALOR moves the party and the fight continues (`engine/combat.go` `executeSpell`, `checkRoundEnd`). Present before M5b (same round end in the terminal at `92ff48a^`) | Difference (engine defect) | LOKTOFEIT is the escape spell; casting it costs everything and saves nothing. Kept; a correction needs its own proposal (M6b decision D-C) |
| 25 | Camp MALOR treats the 0-based floor as if 0 were the castle: teleporting within B1F, or up from B2F, lands "IN THE MOAT" and the party is LOST forever; only B1F (0,0) returns to the castle; below B10F leaves the party on a floor that does not exist instead of "IN SOLID ROCK" (`cmd/istary/main.go` `handleMalorInput`) | Difference (engine defect) | Permanent loss of a party from an ordinary teleport. Kept; a correction needs its own proposal (D-C). Camp MALOR stays out of the GUI until then |
| 26 | Camp D)ISBAND leaves the members marked in the maze (`** OUT **` at the Tavern) and ages them 25 weeks, but never records where: `MazeLevel`, `MazeX`, `MazeY` stay 0, so I)NSPECT finds them only on B1F (0,0), the entrance, not where they were left (`handleCampInput` `D`; `app/maze.go` I)NSPECT). The disbanded slots are left as empty entries in the party list until the roster is reloaded | Difference | Rescue does not work as the guide describes (M4). Kept; part of the M6b-4 rescue design (D-D) |
| 27 | P)ICK UP clears each found character's in-maze flag before checking for room, so a character that does not fit is freed back to the castle without being carried; room is counted with the length of the party list, which can include the empty entries #26 leaves (`app/maze.go`) | Difference | A rescue can free characters it did not reach. Kept; part of the M6b-4 design |
| 28 | Outside combat only SOLID ROCK ends the expedition when everyone is dead: a party killed by a pit, a chute, an OUCH square or poison keeps walking, can take the stairs home, and meets encounters it then loses (`app/maze.go`) | Unclear (engine) | A dead party can still move; with saved expeditions that state would persist. Kept; part of the M6b-4 design |
| 29 | A party wiped out in combat (or by SOLID ROCK) is moved to the castle at once, still a party, all dead: "YOUR PARTY HAS PERISHED IN THE MAZE..." (`app.Controller.LeaveCombat`). No bodies stay in the maze, so there is nothing to rescue | Difference (guide p.42: rescue) | **D-D approved:** Istary will leave a defeated party in the maze for a rescue expedition. Designed in M6b, not implemented yet; the engine is unchanged |
| 30 | Camp U)SE answers "** POWERLESS **" for every item, whether it has a spell or not (`handleCampUseItem`) | Missing | Items that cast spells outside combat cannot be used in Camp. Kept |
| 31 | The B4F-B9F elevator squares (data `aux [0 9 4]`) offer buttons A-I starting at B4F, so G goes to B10F and H and I to B11F and B12F, which do not exist in Wizardry I (ten floors). There the party cannot move; only Camp works (`app/maze.go` buttons; found by the M6b-2 save/resume comparison, probe at `fa3a92e`) | Difference (engine or data defect) | A party can be stuck for good. Kept; a saved expedition there is restored as the engine left it. A correction needs its own proposal (D-C) |
| 32 | Loading a roster rebuilds every character's known spells from their spell slots: `Load` clears `SpellKnown` and `MigrateSpellKnown` marks every spell of each level with slots as known (`engine/save.go`, `engine/magic.go`). So any restart (terminal or GUI) gives a character every spell of the levels they can cast, whatever they had learned; a new mage knows HALITO and KATINO, and after a reload all four level-1 mage spells. Found again by the M6b-3A Camp save/resume tests | Difference (engine; a migration that never stops) | Spell knowledge can grow by restarting. Kept; a correction (run the migration once, for old saves only) needs its own proposal (D-C) |
| 33 | Camp gives an effect only to DIOS, DIAL, DIALMA, MADI, LATUMOFIS, MILWA, LOMILWA, DUMAPIC and MALOR (`applyCampSpell`, `app/camp.go`). Any other spell, protection (MAPORFIC) and healing such as DI or KADORTO included, answers "NOT IN CAMP" and is still spent; a targeted one asks for its target first | Not investigated (Classic eligibility unknown) | Kept (decision 2026-11-08). The GUI does not offer these spells, so it never wastes one; the terminal still can. Classic's Camp spell list and invalid-cast behaviour to be investigated separately |
| 34 | Owned items never become cursed in play: only importing a legacy save sets a possession's cursed flag (`engine/dsk.go`). Maze finds and chest rewards arrive uncursed, and equipping a cursed type (LONG SWORD -1, CURSED ROBE, DEADLY RING...) shows no curse and does not stick (`handleEquipSelection`, `app/inspect.go`). So at Boltac's "WE DONT BUY CURSED ITEMS" and U)NCURSE are reachable only with imported items, and cursed loot can be sold or unequipped | Difference (engine defect; guide p.5, p.31) | **Approved for correction in M8** (decision 2026-11-08), with regression tests, before the Trading Post's final GUI acceptance. Pinned as it is by the M8-0 tests |
| 35 | Selling (or dropping) an equipped item does not recompute the character's AC (`DropItem`; Boltac's sell). The terminal recomputes every AC when it starts (`cmd/istary/main.go`); the GUI never does, so it shows the old AC until the next equip | Difference (engine defect) | **Approved for correction in M8** (decision 2026-11-08), with regression tests, before the Trading Post's final GUI acceptance. Pinned as it is by the M8-0 tests |
| 14 | The terminal game has a map overlay (`M` key, README "Map overlay"). The guide expects hand mapping (PG p.43–45) | Engine only | **D9 approved:** automap is a gameplay enhancement, not Classic. The terminal overlay is left as is |

## 4. Detailed matrix

Engine sources are written as file plus symbol. "PG" pages are printed page
numbers.

### 4.1 Character creation

| ID | Guide (PG) | Engine | Status |
|---|---|---|---|
| C1 | Five races, described qualitatively (p.3) | Five races, `engine/creation.go` `RaceBaseStats` | Match |
| C2 | Dwarf base stats 10/7/10/10/5/6 (Hawkwind, p.52) | Dwarf 10/7/10/10/5/6 | Match. Other races are not tabulated in the guide: Unclear |
| C3 | Class minimums: Fighter STR 11; Mage IQ 11; Priest PIE 11; Thief AGI 11; Bishop IQ/PIE 12; Samurai 15/11/10/14/10; Lord 15/12/12/15/14/15; Ninja 17 all (p.21) | Same, except Ninja 15 all (`ClassRequirements`) | Match, except **Ninja: Difference** |
| C4 | Alignment: Priest and Bishop not Neutral; Thief not Good; Samurai not Evil; Lord Good; Ninja Evil (p.7–8, 21) | `ClassAlignRestrictions` | Match |
| C5 | Bonus points are random (p.20); the Hawkwind example has **6** bonus points (p.52) | 7 + rand(4), with a chance of +10 (`RollBonusPoints`); minimum 7 | Unclear (narrative example) |
| C6 | Bonus points are never enough for Lord or Ninja (p.21) | Maximum roll 29; the cheapest Lord needs 33 points (Hobbit); Ninja-15 needs at least 40 | Match (by calculation) |
| C7 | Stats can't go below the starting value or above 18; all points must be spent (p.20) | `AddStatPoint`, `RemoveStatPoint`; `app` `FinishStats` | Match |
| C8 | Optional password (p.19, p.ii) | Creation password step | Match |
| C9 | Starting gold examples 90–155 (p.52–53) | 90 + rand(100) (`RollGold`) | Match (consistent with the examples) |
| C10 | "Each trainee starts ... with 8 hit points" (p.58); screens show 8/8 for every class (p.25, 30, 46) | Rolled by class die plus VIT (`RollHP`) | Unclear (may be illustrative) |
| C11 | Hawkwind's mages cast HALITO and KATINO; his priest has two DIOS (p.56–58) | Mage HALITO+KATINO, priest DIOS+BADIOS, two level-1 slots (`FinalizeCharacter`) | Match (consistent) |
| C12 | Up to 20 characters (glossary, p.73) | Roster 20 (`Roster.Add`) | Match |

### 4.2 Training Grounds

| ID | Guide | Engine | Status |
|---|---|---|---|
| T1 | Menu C)REATE, I)NSPECT, R)OSTER, L)EAVE (p.19) | `NAME >` prompt: an existing name asks for the password, a new name offers creation; `*ROSTER` lists (`app/training.go`) | Presentation |
| T2 | Character menu: Inspect, Delete, Change Class, Alter Password (p.22) | Inspect, Delete, Change Class, **Reroll** (Wiz 1/2), Set Password; Rite of Passage (Wiz 3) (`render/town.go`) | Reroll: Engine only |
| T3 | Delete asks again (p.22) | `ConfirmDeleteCharacter` | Match |
| T4 | Class change: stats to race minimum, XP 0, items unequipped, spells kept, one point per known spell, class bonus points lost (p.23) | `handleClassChange`: race base stats, level 1, XP 0, non-cursed items unequipped, `SetSpells` | Match. Also **Engine only**: ages +252..356 weeks, grants KATINO or DIOS to a new Mage or Priest |
| T5 | Must know the current password to change it (p.23) | The password is needed to open the character | Match |
| T6 | Visiting the Training Grounds disbands a formed party (p.24) | Party kept | **Difference** |
| T7 | Roster shows alive/dead and in castle/out (p.22) | `*ROSTER` | Not investigated |

### 4.3 Gilgamesh's Tavern

| ID | Guide | Engine | Status |
|---|---|---|---|
| V1 | A)DD, I)NSPECT (#), R)EMOVE, D)IVVY GOLD, L)EAVE (p.24) | A)DD, R)EMOVE, #) SEE A MEMBER, L)EAVE (`render/town.go`) | Divvy gold: **Missing** |
| V2 | Add: shows the compatible characters; pick by letter (p.24) | Type a name at `WHO WILL JOIN ? >` | Presentation |
| V3 | Good and Evil never mix; neutrals join either; the first non-neutral sets the alignment (p.21, 24, 39) | `PartyAlignment` (last non-neutral) plus the BAD ALIGNMENT refusal | Match in effect (first and last non-neutral are always equal) |
| V4 | Only characters in the Castle may join (p.39) | `** OUT **` for in-maze characters (`memberRefusal`) | Match |
| V5 | Dead or incapacitated may join after a confirmation (p.39; "even the dead and disabled", p.53) | Allowed with no confirmation | **Difference** (prompt) |
| V6 | No mention of a password when joining | A password is asked on every join (comment cites p-code CASTLE proc 30) | Engine only (guide silent) |
| V7 | Party up to 6 (p.21, 39) | `MaxPartySize` | Match |
| V8 | Lost characters are beyond help (p.11) | Lost characters are skipped (`** WHO? **`) | Match |

### 4.4 Inspecting a character

| ID | Guide | Engine | Status |
|---|---|---|---|
| I1 | Sheet fields (p.25–27) | Character sheet (`render/town.go`; GUI `CharacterSheet.svelte`) | Match / Presentation |
| I2 | Up to 8 items; markers `*` equipped, `#` unusable, `?` unidentified, `-` cursed (p.27) | 8 items (`AddItem`) | Item limit: Match. Markers: not investigated |
| I3 | Commands R)EAD, D)ROP, P)OOL GOLD, I)DENTIFY, E)QUIP, T)RADE, L)EAVE, plus S)PELL and U)SE in camp (p.28–29) | Town: Equip, Drop, Trade, Read, Leave. Camp adds Spell, Use, Identify (`render/town.go`) | Pool Gold: **Missing**. Identify in town: **Difference** (the guide allows it in the Tavern) |
| I4 | From the Training Grounds, only Read and Leave (p.29) | Not traced | Not investigated |
| I5 | Trade gold and items; items must be unequipped first (p.28) | T)RADE; an `** EQUIPPED **` message exists | Unclear (detail not traced) |
| I6 | Some items offer to invoke a power when equipped (p.5) | Not traced | Not investigated |

### 4.5 Boltac's Trading Post

| ID | Guide | Engine | Status |
|---|---|---|---|
| B1 | B)UY, S)ELL, I)DENTIFY, U)NCURSE, P)OOL GOLD, L)EAVE (p.30) | No pool gold | Pool gold: **Missing** |
| B2 | Prices: long sword 25, short sword 15, anointed mace 30, anointed flail 150, staff 10, dagger 5, small shield 20, large shield 40, robes 15, leather 50, chain mail 90 (p.30, 53–54) | Same values in `wiz1_gamedata.json` | Match |
| B3 | Stock is limited and changes as items are sold (p.30) | Stock tracked in memory; not saved (`SaveState`) | Unclear |
| B4 | Buying an unusable item asks for confirmation (p.31) | `UNUSABLE ITEM - CONFIRM BUY` | Match |
| B5 | Boltac pays half (p.54) | Half price | Match |
| B6 | Cursed or unidentified items can't be sold (p.31) | Cursed refused; unidentified sells for 1 GP | **Difference** (unidentified) |
| B7 | Selling an equipped item asks for confirmation (p.31) | No prompt found in the sell handler | Unclear; kept for M8, the original rules to be investigated (decision 2026-11-08) |
| B8 | Identify and uncurse cost a fee (p.31–32) | Fee is half the price (`docs/RULES.md`) | Match (the guide gives no amount) |
| B9 | No prices for magic items | Suspicious values (finding #13) | Unclear |

### 4.6 Temple of Cant

| ID | Guide | Engine | Status |
|---|---|---|---|
| P1 | Shows a list of characters it can help (p.33) | Type a name at `WHO ARE YOU HELPING ? >` | Presentation |
| P2 | If the payer is short, the party's gold is pooled (p.33) | `CHEAP APOSTATES! OUT!` | **Difference** |
| P3 | Paralysis etc. always cured; the dead and ashes may fail; good vitality helps, old age hurts (p.33) | Paralysed and stoned always succeed. Dead: fail if rand(100) > 50+3×VIT. Ashes: > 40+3×VIT. Age is not used (`engine/temple.go`) | Vitality: Match. Age: Unclear (it may act through age-related vitality loss) |
| P4 | Failed dead becomes ashes; failed ashes becomes lost (p.33) | Same | Match |
| P5 | Raised from dead: 1 HP; from ashes: full HP (p.33) | Same | Match |
| P6 | Donation not stated | (100, 200, 250, 500) × level (`TempleDonation`) | Engine only |
| P7 | Poison is cured on return to the Castle (p.11, 60) | Not cured (comment cites Pascal) | **Difference** |
| P8 | Lost characters get a free burial (p.11) | Lost characters stay on the roster with status LOST | Unclear |

### 4.7 Adventurer's Inn

| ID | Guide | Engine | Status |
|---|---|---|---|
| N1 | Stables 0 HP/0 gp; Barracks 1/10; Double Occupancy 3/50; Private Room 7/200; Royal Suite 10/500 per week (p.34) | Same numbers; names STABLES, A COT, ECONOMY, MERCHANT, ROYAL (`engine/inn.go`, `engine/town.go`) | Rules: Match. Names: Presentation |
| N2 | Stay until healed or out of money; SPACE checks out early (p.34) | Heal loop; a key skips the animation | Unclear (early checkout vs skip) |
| N3 | P)OOL GOLD at the Inn (p.34) | None | **Missing** |
| N4 | Weeks pass; age matters (p.34, 61) | No aging found for Inn stays | Unclear |
| N5 | Level check after each stay; stat changes; new spells; "you need N more" (p.35) | `innTransitionToLevelUp`, `CheckLevelUp`, `InnStatChanges` | Match |
| N6 | One level per stay; stay again for more (p.35) | One level per `CheckLevelUp` | Match |
| N7 | Spell points reset on Maze entry (p.9) | Restored at the Inn (`RestoreSpells`) | **Difference** (see finding #10) |
| N8 | Feeble past 50; can die of old age (p.34, 73) | Age-weighted stat loss on level-up; VIT reaching 2 means death (LOST) (`InnStatChanges`) | Match (qualitative) |

### 4.8 Edge of Town and Utilities

| ID | Guide | Engine | Status |
|---|---|---|---|
| E1 | Edge of Town: Training, Maze, Castle, Utilities, Leave Game (p.36) | Maze, Training, Castle, Leave Game; Utilities from the title menu | Presentation |
| E2 | Interrupted expeditions leave characters OUT (p.36) | `InMaze`, `MazeLevel`, `MazeX`, `MazeY` exist per character, but the position is never written (#26), and the save holds no expedition: after a restart the party is in the castle with its losses and loot. Corrected 2026-11-07 (was "Match (not run)"; checked by a probe at `a1eef2a`) | **Difference**. **D-A implemented (M6b-2):** a GUI expedition resumes at its last safe point; the terminal is unchanged |
| E3 | Utilities: M)OVE CHARACTERS, C)HANGE NAME, R)ESTART AN "OUT" PARTY (p.37–38) | B)ACKUP, C)HANGE NAMES, I)MPORT, T)RANSFER | Restart: **Missing**. Move ≈ Import/Transfer: Presentation |
| E4 | Moving to another scenario strips items (p.37) | Import / Rite of Passage (`engine/utilities.go`, `RiteApply`) | Not investigated |

### 4.9 Maze and Camp

| ID | Guide | Engine | Status |
|---|---|---|---|
| M1 | Enter at level 1, south-west corner, facing north, in Camp (p.40, 43) | `PhaseCamp`, level 0, (0,0), North | Match |
| M2 | Forward, left, right, turn around; cursor keys (p.42) | F/L/R, arrows, WASD. F)ORWARD stops at doors ("OUCH!"); K)ICK passes doors and hidden doors (engine comment: Pascal FORWRD and KICK). Now `app.Controller` operations (M6a), unchanged | Partly checked |
| M3 | Keys: C camp, S status, I inspect area, T delay **1–99 in 1/10 s, default 20**, Q **quit expedition**, plus a key to toggle windows (letter lost in OCR) (p.42–43) | C, S, I (with P)ICK UP), T (comment: range 1–5000, default 1500 ms), **Q = quick plot**, K)ICK. No maze quit; camp D)ISBAND leaves members OUT, +25 weeks age | Mixed: **Difference** (quit vs disband, delay scale), K)ICK Engine only |
| M4 | Rescue: inspect the area; must be reachable without doors; room in the party (p.42) | I)NSPECT, P)ICK UP. Bodies are never left in the maze (#29) and disbanded members are found only at the entrance (#26); P)ICK UP frees those it has no room for (#27) | **Difference**. Reachability rule: not investigated. Rescue designed in M6b (D-D) |
| M5 | Quitting forgets MILWA, LOMILWA, LATUMAPIC, MAPORFIC (p.43) | — | Not investigated |
| M6 | 20×20 levels (p.44) | 20×20 with wraparound (`MoveForward`) | Match |
| M7 | DUMAPIC reports position from the stairs (0 E, 0 N, 1 down) and facing (p.44, 64) | — | Not investigated |
| M8 | Items dropped in the Maze are lost (p.28, 45) | — | Not investigated |
| M9 | Hand mapping expected (p.43–45) | Map overlay `M` | Engine only (an enhancement under the charter) |
| K1 | Camp: I)NSPECT, R)EORDER, E)QUIP party, L)EAVE (p.41) | R)EORDER, E)QUIP, D)ISBAND, #) inspect, L)EAVE | Disband: Engine only (see M3) |
| K2 | Reorder by choosing members in order (p.41) | R)EORDER | Match (not traced) |

### 4.10 Combat and chests

| ID | Guide | Engine | Status |
|---|---|---|---|
| F1 | F)IGHT (slots 1–3), P)ARRY, D)ISPELL, S)PELL, U)SE, R)UN, T)AKE BACK (p.47) | F)IGHT S)PELL P)ARRY R)UN U)SE D)ISPELL, GO B)ACK (`render`) | Match |
| F2 | Only slots 1–3 melee and take physical hits; spells reach anyone (p.48, 55) | Fight limited by `app.FrontRowSize`; monster targeting not traced | Partly checked |
| F3 | The dead move to the back at the end of the round (p.48) | `sortPartyByStatus` also moves sleeping or paralysed members back (consistent with "killed or incapacitated", p.55). Until P1c it ran at the start of `ExecuteRound`, after order entry, and orders stayed with slots (finding #15). Since P1c it runs in `BeginNextRound` at the round transition, and orders and ally targets follow the character | Status order: Match. Timing and orders: **Corrected** (Istary C1–C3, integrated into `main` locally 2026-11-06) |
| F4 | Surprise gives a free round; no spells in it (p.48, 56) | 20% / 20% surprise rolls (`engine/combat.go`) | Spell restriction: not investigated |
| F5 | Truce offers from peaceful monsters, with consequences (p.48) | `F)IGHT OR L)EAVE IN PEACE` | Consequences: not investigated |
| F6 | XP divided among survivors (p.6) | Divided by the full party; paid to the living | **Difference** |
| F7 | No XP for dispelled or fleeing monsters (p.47–48) | Counts members with status ≥ dead | Not investigated (how dispel is recorded) |
| F8 | Monsters disguised unless IQ is high enough, or LATUMAPIC (p.47) | Per-round identification | Not investigated |
| H1 | Chest: O)PEN, I)NSPECT, C)ALFO, D)ISARM, L)EAVE, choosing who (p.49) | Same | Match |
| H2 | Disarm requires typing the trap name (p.49) | No prompt | **Difference** |
| H3 | Thieves are much better at inspecting and disarming (p.49) | Chance = level − dungeon level + 50 for thief or ninja | Match (qualitative) |

### 4.11 Spells and items

| ID | Guide | Engine | Status |
|---|---|---|---|
| S1 | 50 spells; mage levels 1–7 have 4/2/2/3/3/4/3, priest 5/4/4/4/6/4/2 (p.63–72) | 21 mage + 29 priest (`engine/magic.go`; README) | Match (count) |
| S2 | At most 9 points per level; at least one per known spell (p.9) | `SetSpells` (cap 9, `minSpCnt`) | Match |
| S3 | Bishop: mage spells at once, priest from level 4. Samurai: mage from 4. Lord: priest from 4 (p.8) | `SetSpells` class offsets | Match (structure; progression not compared) |
| S4 | Knowing one spell of a level eventually teaches all of that level (p.9, 23) | `TryLearn` | Not investigated |
| S5 | Effects and damage ranges per spell (p.64–72) | `engine/combat.go`, `engine/magic.go` | Not investigated individually |
| S6 | HAMAN and MAHAMAN need level 13+ and cost a level (p.66–67) | — | Not investigated |
| X1 | Equipping an item of another alignment impairs and curses it (p.4) | — | Not investigated |
| X2 | Only Boltac's removes curses safely (p.5) | Uncurse at Boltac's | Match |
| X3 | Items casting spells may break; scrolls and potions once, rods and staffs about a dozen times (p.5, 29) | — | Not investigated |

## 5. Follow-up investigations

These are proposed, not scheduled. None needs code changes except where a
test is mentioned, and those tests would only observe the code. No new
comprehensive investigation was carried out for the design baseline.

### 5.1 Priority order (agreed 2026-11-06)

1. **Historical program revision and rules differences.**
   - Find the source of the engine's `WCnnn` change codes and the revision of
     the Pascal source the engine follows.
   - Find which program revision this 1987 guide describes.
   - Compare against other printings, such as the 1981 first edition if
     available.
   - Where the user lawfully owns original disk images, observe the original
     in an Apple II emulator, recording the disk version with every
     observation.

   This underpins findings 1–3, 5–8 and 10, and decisions D1–D3.
2. **Suspicious magic-item prices** (finding #13). Check the decoding of
   multi-word prices in `wiz1_gamedata.json` against the scenario disk
   structure (`engine/dsk.go`).
3. **Experience distribution** (finding #7, row F6).
4. **Poison behaviour** on return to the Castle (finding #2, row P7).
5. **Inn aging** (finding #11, row N4).
6. **Chest trap-name entry** (finding #8, row H2).
7. **Party re-sort timing and order binding** (finding #15). *Proposed addition, not part of the agreed order; the user may move it up.* It is on the critical path to GUI combat. Characterization tests first, then corrections C1–C3 (approved in principle).

For items 3–7, start with read-only characterization tests that pin today's
engine behaviour (roadmap M1). Then seek historical evidence. Shop stock
persistence (finding #12) gets a characterization test at the same time.

### 5.2 Later

- **D3 candidates:** gold pooling, Divvy Gold, Temple pooling and restarting
  an OUT party (findings #4, #5, #9). These are candidate missing Classic
  features, to be investigated under item 1.
- **D2:** Training Grounds disbanding the party (finding #3). It stays
  unresolved until historical evidence is available.
- **"Not investigated" rows above**, by priority:
  - first, spell effects and the targeting of physical attacks;
  - then items, alignment curses, and breaking items.
- **Faint pages:** rescan or re-OCR the faint glossary page (PDF p.80) and
  the screen pictures if needed.

## 6. Decision log

Each resolved discrepancy is recorded here. When a decision is made, the
matching row above is updated to say which behaviour Istary Classic follows.

| Finding / row | Historical revision | Evidence | Decision | Implementation change | Date |
|---|---|---|---|---|---|
| #17 | Engine at `22099c4` (not a historical rules question) | `TestRunEndsTheRoundForPlayback`; M5a differential fuzz | **Approved correction of an engine defect.** A successful R)UN ends the round through the normal round-end step: the flight message is shown, then combat is left. No rule changes: the escape chance, its roll and failed escapes are unchanged | `engine/combat.go` (`ExecuteRound`), M5b. Goldens unchanged (no golden fight escapes) | 2026-11-06 |
| E2 | Not a historical rules question | M6b review (2026-11-07); probe at `a1eef2a` | **D-A approved (Istary behaviour):** after a restart, an expedition resumes at its last safe point instead of the party appearing in the castle. **D-B (interim):** an interrupted combat or chest resumes at that safe point with committed losses and spent resources kept and no unresolved experience, gold or treasure awarded; exact resumption of combat and chests is still needed | **Implemented in M6b-2** for the GUI: optional expedition section (D-E), checkpoints with D-G, D-H, W, L and P approved in principle 2026-11-07; see `design/m6b/EXPEDITION-PERSISTENCE.md` §13. Interim D-B limitation stands: a restart ends an encounter without its rewards; exact resumption still needed | 2026-11-07 |
| #29, M4 | Player's Guide p.42 (rescue) | M6b review | **D-D approved (Istary behaviour change):** a defeated party stays in the maze for a rescue expedition, subject to party size and the recovery rules. To be designed explicitly before implementation | Not implemented | 2026-11-07 |
| #24–#28 | Not decided | M6b review | **D-C:** each dangerous defect may be corrected only after its own proposal and approval, never together with persistence work | None | 2026-11-07 |
| #26 | Not a historical rules question | M6b-3A report | **Disband not exposed in the GUI** until M6b-4 records the position and provides rescue | GUI refuses it (`gui/camp.go`); terminal unchanged | 2026-11-08 |
| #25 | Not decided | M6b-3A report | **MALOR stays unavailable in the GUI** until its floor calculations are corrected separately (D-C) | Refused by `CastSpell` and the GUI; terminal unchanged | 2026-11-08 |
| #33 | Not investigated | M6b-3A report | **Preserve current behaviour.** Investigate Classic's Camp spell eligibility and invalid casts separately; report without implementing | None (the GUI offers only spells with an effect in Camp) | 2026-11-08 |
| #32 | Not a historical rules question | M6b-3A report | **Approved as a defect for correction**, not within M6b-3B. A backward-compatible migration and its regression tests are to be proposed first | None yet | 2026-11-08 |
| #12, B3 | Unclear | M8 review | **Keep resetting stock at start for M8** (it is not saved) | None in M8 | 2026-11-08 |
| B6 | Engine cites the original game's code (1 GP); guide says cannot sell | M8 review | **Keep 1-GP sales of unidentified items** | None | 2026-11-08 |
| B7 | Unclear | M8 review | **Keep selling equipped items without a question initially;** investigate the original rules | None yet | 2026-11-08 |
| #4, B1 | Guide p.24–34 | M8 review | **Pool Gold deferred,** and no disabled control for it in the GUI for now | None | 2026-11-08 |
| #34, #35 | Not historical rules questions (engine defects) | M8 review; pinned by `cmd/istary/shop_test.go` (M8-0) | **Approved for correction during M8,** with regression tests, before the Trading Post's final GUI acceptance | None yet | 2026-11-08 |
| — | — | — | *No other discrepancy decided yet. The current engine remains the provisional executable baseline.* | — | 2026-11-06 |
| #15, F3 | Player's Guide 1987 printing; engine at `bb01fb4` | PG p.48, p.55; `probes/resort-probe_test.go.txt` under `design/m2-exploration/` | **Approved in principle** as an Istary correction of an implementation quirk, consistent with the guide. Orders, ally targets and repeated orders stay with the character (C1). The formation change applies at the round boundary, before order entry (C2; timing to be confirmed in M1). No stale orders (C3). The status order of the sort is unchanged, and recovery does not move members forward | **Implemented** on `spike/p1c-order-binding` and integrated into `main` locally (2026-11-06, not pushed). Characterization tests first (P1a, `a27adb4`); then `engine/combat_rounds.go` (`BeginNextRound`, `bindOrders`), with the re-sort removed from `ExecuteRound` and one terminal call. No golden changed. A 300-seed old-vs-new comparison found 148 differing fights, each first differing only after a member was incapacitated, every time a priest or mage no longer carrying out a dead member's leftover Fight. See `design/m2-exploration/P1-COMBAT-ENGINE-SPIKE.md` §7 | 2026-11-06 |

## 7. Maintenance

- Update a row whenever code touching that rule changes.
- Keep the evidence and citations of superseded rows. Mark them as resolved
  rather than deleting them.
