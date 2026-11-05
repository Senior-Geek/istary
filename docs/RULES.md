# Hard-coded game rules

Where the rules live today, as candidates for a future configurable ruleset.
Nothing here is configurable yet. Rules that come from the embedded scenario
JSON (monsters, items, rewards, mazes, experience tables) are already data and
are not listed.

Locations are given as file and symbol so they survive edits.

## Character creation

| Rule | Value | Location |
|---|---|---|
| Race base stats (STR IQ PIE VIT AGI LCK) | Human 8/8/5/8/8/9, Elf 7/10/10/6/9/6, Dwarf 10/7/10/10/5/6, Gnome 7/7/10/8/10/7, Hobbit 5/7/7/6/10/15 | `engine/creation.go` `RaceBaseStats` |
| Bonus points | 7 + rand(4); while < 20, 1 in 11 chance of +10 | `engine/creation.go` `RollBonusPoints` |
| Stat range during allocation | race base to 18 | `engine/creation.go` `AddStatPoint`, `RemoveStatPoint` |
| Class stat minimums | Fighter STR 11, Mage IQ 11, Priest PIE 11, Thief AGI 11, Bishop IQ/PIE 12, Samurai 15/11/10/14/10, Lord 15/12/12/15/14/15, Ninja all 15 | `engine/creation.go` `ClassRequirements` |
| Class alignment limits | Priest/Bishop not Neutral, Thief not Good, Samurai not Evil, Lord Good, Ninja Evil | `engine/creation.go` `ClassAlignRestrictions` (applied in `ClassAvailability` and `CharClassQualifies`) |
| Starting HP | class die (F10 M4 P8 T6 B6 S16 L10 N6) + VIT modifier, two 50% x9/10 passes, min 2 | `engine/creation.go` `ClassHPDie`, `vitMod`, `RollHP` |
| Starting gold | 90 + rand(100) | `engine/creation.go` `RollGold` |
| Starting age | 18*52 + rand(300) weeks | `engine/creation.go` `FinalizeCharacter` |
| Starting spells | Mage/Bishop HALITO+KATINO, 2 level-1 mage slots; Priest DIOS+BADIOS, 2 level-1 priest slots | `engine/creation.go` `FinalizeCharacter` |
| Creation allowed | Wiz 1 only (Wiz 2/3 import characters) | `app/training.go` `submitTrainingName` |
| Name/password length, name characters | 15; A-Z 0-9 space - ' | `app/text.go` `maxNameLen`, `appendNameChar`, `appendPasswordChar` |
| Wiz 3 Rite of Passage | stat loss, class bonus, HP, spells, age 1040, gold cap 500 | `engine/creation.go` `RiteApply`, `RiteAlignOptions` |

## Roster and party

| Rule | Value | Location |
|---|---|---|
| Roster size | 20 | `engine/character.go` `Roster.Add`; also `app/training.go` `submitTrainingName`, `engine/utilities.go` (import/transfer) |
| Party size | 6 | `app/tavern.go` `maxPartySize`; also `cmd/istary/main.go` I)NSPECT pick-up in `handleMazeInput` |
| Party alignment | last non-neutral member's alignment; Good and Evil cannot mix | `app/tavern.go` `PartyAlignment`, `submitAddMember` |
| Who may join | not LOST, not in the maze, Wiz 3 legacy only | `app/tavern.go` `submitAddMember` |
| Items per character | 8 | `engine/character.go` `AddItem`, `TradeItem` |
| Front row (can melee) | first 3 members | `cmd/istary/main.go` `handleCombatChoose` |

## Town

| Rule | Value | Location |
|---|---|---|
| Inn room costs | 0/10/50/200/500 | `engine/town.go` `InnRoomCosts` |
| Inn healing, spell restore, level-up stat changes | | `engine/inn.go` `RestAtInn`, `RestoreSpells`, `InnStatChanges` |
| Temple donation | (Paralyzed 100, Stoned 200, Dead 250, Ashed 500) x level | `engine/town.go` `TempleDonation` |
| Temple outcomes and aging | | `engine/temple.go` `TempleHeal` |
| Shop sell price, identify/uncurse fee | half the item price | `cmd/istary/main.go` `handleShopInput` |
| Class change | level 1, XP 0, age + 252 + 52*rand(3) weeks | `cmd/istary/main.go` `handleClassChange` |
| Disband aging | +25 weeks | `cmd/istary/main.go` `handleCampInput` |

## Advancement

| Rule | Location |
|---|---|
| XP needed per level (from the scenario's experience table) | `engine/combat.go` `XPForNextLevel`, `CheckLevelUp` |
| Level-up HP die: F/L 10, P/S 8, T/B/N 6, M 4 (differs from the creation table, e.g. Samurai 8 vs 16) | `engine/combat.go` `classHPDie`, `RecalcHP` |
| Spell slots by level | `engine/combat.go` `SetSpells` |
| Spell definitions | `engine/magic.go` `SpellTable`, `MageSpellsByLevel`, `PriestSpellsByLevel` |

## Dungeon

| Rule | Value | Location |
|---|---|---|
| Random encounter chance per step | rand(99) == 35 | `cmd/istary/main.go` `handleMazeInput` |
| Regeneration/poison tick | 25% per step, HP += heal - poison | `cmd/istary/main.go` `handleMazeInput` |
| Fight zones per level | 9 random rooms plus encounter squares | `engine/game.go` `InitFightMap` |
| Special squares (stairs, chutes, ouchy, messages, ...) | | `cmd/istary/main.go` `checkSquare`, `hazardDamage`, `ouchyDamage` |
| Level size and wraparound | 20x20 | `engine/game.go` `MoveForward` |

## Combat

| Rule | Location |
|---|---|
| Initiative | `engine/combat.go` `rollPartyInitiative`, `rollMonsterInitiative` |
| Per-round identification, healing/poison (25%) | `engine/combat.go` `ExecuteRound` |
| To-hit, swings, damage, criticals | `engine/combat.go` `executeFight` |
| Monster targeting | `engine/combat.go` `pickPartyTarget`, `pickPartyTargetIndex` |
| Monster XP, rewards, gold | `engine/combat.go` `calcMonsterXP`, `calculateRewards` |
| Chest traps | `engine/combat.go` `generateTrap` |
| Message timing (PAUSE1/PAUSE2) | `cmd/istary/main.go` `combatTimerTick` |
