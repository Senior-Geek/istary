package app

import (
	"fmt"
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// The spell descriptions (COMBAT-1): one short, plain account of each
// spell, written from what Istary's engine does with it. Combat, Camp and
// the character sheet all show these, and nothing else describes a spell,
// so the screens cannot disagree.
//
// A description says what the spell is for. Where Istary cannot cast it in
// a place, the reason is given separately (CombatSpellNote, the Camp
// reasons), not folded into the description.

// spellSummaries is each spell's description, by name.
var spellSummaries = map[string]string{
	// Mage, level 1
	"HALITO":  "A small ball of fire: 1–8 damage to one monster.",
	"MOGREF":  "Steels the caster's body: their armour class improves by 2 for the rest of the fight.",
	"KATINO":  "Tries to put a monster group to sleep. Stronger monsters resist, and some cannot be slept at all.",
	"DUMAPIC": "Shows the party's position in the maze and the way it is facing.",
	// Mage, level 2
	"DILTO": "Darkness on one monster group: its armour class worsens by 2, so it is easier to hit.",
	"SOPIC": "Makes the caster hard to see: their armour class improves by 4 for the rest of the fight.",
	// Mage, level 3
	"MAHALITO": "A fireball: 4–24 damage to every monster in one group.",
	"MOLITO":   "A spray of sparks: 3–18 damage to every monster in one group.",
	// Mage, level 4
	"MORLIS":   "Fear on one monster group: its armour class worsens by 3.",
	"DALTO":    "A blizzard: 6–36 damage to every monster in one group.",
	"LAHALITO": "A storm of flame: 6–36 damage to every monster in one group.",
	// Mage, level 5
	"MAMORLIS": "Fear on every monster group: their armour class worsens by 3.",
	"MAKANITO": "A deadly air: kills every monster below level 8, in every group. The undead are unaffected.",
	"MADALTO":  "A frost storm: 8–64 damage to every monster in one group.",
	// Mage, level 6
	"LAKANITO": "Smothers one monster group: each monster may die. Stronger monsters resist.",
	"ZILWAN":   "Destroys one undead monster (10–2000 damage). Others are unaffected.",
	"MASOPIC":  "Protects the whole party: armour class improves by 4 for the rest of the fight.",
	"HAMAN":    "A plea to the gods: costs the caster a level and grants a boon. Fails below level 13.",
	// Mage, level 7
	"MALOR":     "Teleports the party within the maze.",
	"MAHAMAN":   "A greater plea: as HAMAN, with greater boons. Costs a level; fails below level 13.",
	"TILTOWAIT": "A great explosion: 10–150 damage to every monster.",

	// Priest, level 1
	"KALKI":  "A blessing: the whole party's armour class improves by 1 for the rest of the fight.",
	"DIOS":   "Heals one party member 1–8 hit points.",
	"BADIOS": "Wounds one monster: 1–8 damage.",
	"MILWA":  "Light: the party sees farther in the maze for a while.",
	"PORFIC": "Shields the caster: their armour class improves by 4 for the rest of the fight.",
	// Priest, level 2
	"MATU":    "Protects the whole party: armour class improves by 2 for the rest of the fight.",
	"CALFO":   "Divines the trap on a chest; it is right most of the time.",
	"MANIFO":  "Tries to hold one monster group still. Most monsters resist.",
	"MONTINO": "Tries to silence one monster group for a few rounds, so it cannot cast spells.",
	// Priest, level 3
	"LOMILWA":   "Lasting light: the party sees farther in the maze for the rest of the expedition.",
	"DIALKO":    "Wakes or frees one party member who is asleep or paralysed.",
	"LATUMAPIC": "Reveals what the monsters really are.",
	"BAMATU":    "Protects the whole party: armour class improves by 4 for the rest of the fight.",
	// Priest, level 4
	"DIAL":      "Heals one party member 2–16 hit points.",
	"BADIAL":    "Wounds one monster: 2–16 damage.",
	"LATUMOFIS": "Cures one party member's poison.",
	"MAPORFIC":  "A big shield: the whole party's armour class improves by 2 for the rest of the fight.",
	// Priest, level 5
	"DIALMA":   "Heals one party member 3–24 hit points.",
	"BADIALMA": "Wounds one monster: 3–24 damage.",
	"LITOKAN":  "Pillars of flame: 3–24 damage to every monster in one group.",
	"KANDI":    "Finds where a missing character lies in the maze.",
	"DI":       "Tries to raise a dead party member.",
	"BADI":     "Tries to slay one monster outright. Stronger monsters resist.",
	// Priest, level 6
	"LORTO":     "Whirling blades: 6–36 damage to every monster in one group.",
	"MADI":      "Fully heals one party member. In combat it also cures their sleep, paralysis and poison.",
	"MABADI":    "Leaves one monster with only 1–8 hit points.",
	"LOKTOFEIT": "Recalls the party to the castle, at the cost of all its gold and equipment.",
	// Priest, level 7
	"MALIKTO": "Divine wrath: 12–72 damage to every monster.",
	"KADORTO": "Restores a party member to life, even from ashes.",
}

// spellAffects says who or what a spell acts on, where its target type
// alone would mislead.
var spellAffects = map[string]string{
	"DUMAPIC": "the party", "MALOR": "the party", "CALFO": "a chest", "KANDI": "a missing character",
	"MILWA": "the party", "LOMILWA": "the party", "LATUMAPIC": "every monster group", "MAPORFIC": "the whole party",
	"LAKANITO": "one monster group", "ZILWAN": "one monster (choose its group)", "MABADI": "one monster (choose its group)",
	"LOKTOFEIT": "the whole party",
}

// combatNotes are the spells the GUI does not let be cast in combat, and
// why: Istary gives them no effect there, or one known to be wrong, so a
// cast would only spend a point (as Camp's list does, finding #33). The
// engine and the terminal are unchanged.
var combatNotes = map[string]string{
	"DUMAPIC":   "Exploration only: cast it in Camp.",
	"CALFO":     "For chests: offered when a chest is found.",
	"KANDI":     "No effect in combat.",
	"DI":        "No effect in combat.",
	"KADORTO":   "No effect in combat.",
	"LOKTOFEIT": "Not available in combat until its defect (finding #24) is corrected.",
	"MALOR":     "Not available in combat until its defect (finding #24) is corrected.",
}

// SpellSummary is the description of the named spell, or "" if there is
// no such spell.
func SpellSummary(name string) string { return spellSummaries[name] }

// SpellAffects says, in words, what the spell is cast on.
func SpellAffects(sp *engine.Spell) string {
	if a, ok := spellAffects[sp.Name]; ok {
		return a
	}
	switch sp.Target {
	case engine.TargetSelf:
		return "the caster"
	case engine.TargetPartyMember:
		return "one party member"
	case engine.TargetParty:
		return "the whole party"
	case engine.TargetSingleMonster:
		return "one monster (choose its group)"
	case engine.TargetMonsterGroup:
		return "one monster group"
	case engine.TargetAllMonsters:
		return "every monster group"
	}
	return ""
}

// CombatSpellNote is why the GUI will not cast the named spell in combat,
// or "" when it may (if the caster has the point for it).
func CombatSpellNote(name string) string { return combatNotes[name] }

// spellBook is "mage" or "priest".
func spellBook(sp *engine.Spell) string {
	if sp.Type == engine.PriestSpell {
		return "priest"
	}
	return "mage"
}

// castsLeft is ch's points left and most points at sp's level and book.
func castsLeft(ch *engine.Character, sp *engine.Spell) (left, max int) {
	if sp.Type == engine.PriestSpell {
		return ch.PriestSpells[sp.Level-1], ch.MaxPriestSpells[sp.Level-1]
	}
	return ch.MageSpells[sp.Level-1], ch.MaxMageSpells[sp.Level-1]
}

// canUseBook reports whether ch's class casts spells of sp's book.
func canUseBook(ch *engine.Character, sp *engine.Spell) bool {
	switch ch.Class {
	case engine.Mage, engine.Samurai:
		return sp.Type == engine.MageSpell
	case engine.Priest, engine.Lord:
		return sp.Type == engine.PriestSpell
	case engine.Bishop:
		return true
	}
	return false
}

// noCastsLeft is the reason a known spell cannot be cast for want of
// points: none left at its level (since the last Inn stay), or none at
// all yet.
func noCastsLeft(ch *engine.Character, sp *engine.Spell) string {
	left, max := castsLeft(ch, sp)
	if left > 0 {
		return ""
	}
	if max == 0 {
		return fmt.Sprintf("No level %d %s casts yet.", sp.Level, spellBook(sp))
	}
	return fmt.Sprintf("No level %d %s casts left (0 of %d); a stay at the Inn restores them.", sp.Level, spellBook(sp), max)
}

// SpellGuideView is one spell as the spell guide describes it, for the
// character sheet and any spell reference.
type SpellGuideView struct {
	Name    string `json:"name"`
	Level   int    `json:"level"`
	Book    string `json:"book"` // mage or priest
	Summary string `json:"summary"`
	Affects string `json:"affects"`
	Combat  string `json:"combat,omitempty"` // why it is not cast in combat, if it is not
	Camp    bool   `json:"camp"`             // it has an effect in Camp
}

// SpellGuide is every spell, in table order (mage levels 1-7, then
// priest).
func SpellGuide() []SpellGuideView {
	out := make([]SpellGuideView, 0, len(engine.SpellTable))
	for _, sp := range engine.SpellTable {
		out = append(out, SpellGuideView{
			Name: sp.Name, Level: sp.Level, Book: spellBook(sp), Summary: SpellSummary(sp.Name),
			Affects: SpellAffects(sp), Combat: CombatSpellNote(sp.Name), Camp: campSpells[sp.Name] && sp.Name != "MALOR",
		})
	}
	return out
}

// classCannot is the reason ch's class cannot cast sp's book.
func classCannot(ch *engine.Character, sp *engine.Spell) string {
	return fmt.Sprintf("A %s cannot cast %s spells.", strings.ToLower(ch.Class.String()), spellBook(sp))
}
