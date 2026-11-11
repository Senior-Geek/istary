package app

// Saved expeditions (M6b-2; docs/design/m6b/EXPEDITION-PERSISTENCE.md).
//
// Checkpoint records the game at a safe point: in the maze (with or without
// an open maze prompt), in Camp (with any Camp screen open, M6b-3A), or
// during an encounter. Each step has
// finished by then: the move, the square's effect, healing and poison and
// the encounter check all happen within one key press, and an encounter's
// fight room is already cleared when combat begins. Resume puts the game
// back at that point without replaying anything and without drawing a
// random number.
//
// During an encounter the checkpoint is the game as it stands without the
// encounter, marked interrupted: committed changes outside the characters
// (a chest's teleporter or alarm, combat MALOR's move) are kept, and the
// encounter itself, if the app stops before it ends, is treated as over
// with nothing more awarded (interim decision D-B, with D-G). Characters
// are not in the checkpoint: their one record is the roster entry, saved
// with it.

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// Interrupted encounter kinds in a saved expedition.
const (
	InterruptedCombat = "combat"
	InterruptedChest  = "chest"
)

// ErrNotOnExpedition is returned when there is no expedition to save or
// resume.
var ErrNotOnExpedition = errors.New("no expedition is under way")

// ExpeditionProblem is a saved expedition that cannot be resumed as it is.
// It is never discarded; the front end says why and offers to return the
// party to the castle.
type ExpeditionProblem struct{ Problem string }

func (p *ExpeditionProblem) Error() string {
	return "the saved expedition cannot be resumed: " + p.Problem
}

// Checkpoint returns the expedition as it stands, to be saved with the
// roster. pristine is the scenario as loaded, to find changed special
// squares. The party must be in the maze, in Camp or in an encounter.
func (c *Controller) Checkpoint(pristine *data.Scenario) (*engine.Expedition, error) {
	g := c.Game
	var where engine.Checkpoint
	interrupted := ""
	switch g.Phase {
	case engine.PhaseMaze:
		where.Phase = "maze"
	case engine.PhaseCamp:
		where.Phase = "camp"
		where.CampMessage = g.Town.Message
		screen, err := c.campScreen()
		if err != nil {
			return nil, err
		}
		where.Camp = screen
	case engine.PhaseCombat:
		if g.Combat == nil {
			return nil, ErrNotOnExpedition
		}
		where.Phase = "maze"
		interrupted = InterruptedCombat
		if g.Combat.Phase == engine.CombatChest || g.Combat.Phase == engine.CombatChestResult {
			interrupted = InterruptedChest
		}
	default:
		return nil, ErrNotOnExpedition
	}
	var party []string
	for _, m := range g.Town.Party.Members {
		if m != nil {
			party = append(party, m.Name)
		}
	}
	if len(party) == 0 {
		return nil, ErrPartyEmpty
	}

	where.Level, where.X, where.Y, where.Facing = g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing
	where.Light, where.Protect, where.Fizzles = g.LightLevel, g.ProtectLevel, g.Fizzles
	where.QuickPlot, where.ChestAlarm = g.QuickPlot, g.ChestAlarm
	for x := 0; x < 20; x++ {
		var b strings.Builder
		for y := 0; y < 20; y++ {
			if g.FightMap[x][y] {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		where.FightMap = append(where.FightMap, b.String())
	}
	if interrupted == "" {
		where.Message, where.Message2 = g.MazeMessage, g.MazeMessage2
		where.Viewport, where.Viewport2 = g.ViewportMsg, g.ViewportMsg2
		where.Messages = append([]string(nil), g.MazeMessages...)
		where.MsgScroll, where.MsgWait = g.MazeMsgScroll, g.MazeMsgWait
		where.SearchYN, where.SearchMonster = g.MazeSearchYN, g.SearchMonster
		if g.SearchCell != nil {
			if g.SearchCell != g.CurrentCell() {
				return nil, fmt.Errorf("the pending search is not for the party's square")
			}
			where.SearchHere = true
		}
		where.Back2Shop, where.TryGet, where.TryGetItem = g.MazePendingBack2Shop, g.MazePendingTryGet, g.MazeTryGetItem
		where.WaitButton, where.ButtonCount = g.WaitButton, g.ButtonCount
		where.Inspecting = g.MazeInspecting
		for _, f := range g.MazeInspectFound {
			where.InspectFound = append(where.InspectFound, f.Name)
		}
	}
	return &engine.Expedition{
		Version:     engine.ExpeditionVersion,
		Party:       party,
		Where:       where,
		Interrupted: interrupted,
		Squares:     engine.ChangedSquares(pristine, g.Scenario),
	}, nil
}

// Resume puts the saved expedition (Game.Expedition) back: the party from
// the roster, the special squares, and the maze or Camp at the checkpoint;
// or, when no one in the party can go on, leaves it stranded there, with
// the game in the castle.
// An interrupted encounter is over: the party stands where the checkpoint
// left it, with the encounter's messages cleared as leaving combat does.
// Nothing is replayed and no random number is drawn. The game must be in
// town with no party formed; on a problem nothing is changed.
func (c *Controller) Resume(pristine *data.Scenario) error {
	g := c.Game
	e := g.Expedition
	if e == nil {
		return ErrNotOnExpedition
	}
	if g.Phase != engine.PhaseTown || g.Town.Party.Size() != 0 {
		return &ExpeditionProblem{"a party is already formed in the castle"}
	}
	members, found, err := c.checkExpedition(e)
	if err != nil {
		return err
	}
	if _, onSquare := engine.PlaceOf(len(g.Scenario.Mazes.Levels), e.Where.Level, e.Where.X, e.Where.Y); !onSquare && !engine.PartyCanContinue(members) {
		return &ExpeditionProblem{"no one in the party can go on, and it is not on a square of the maze to be left on"}
	}
	if err := engine.RestoreSquares(pristine, g.Scenario, e.Squares); err != nil {
		return &ExpeditionProblem{err.Error()}
	}

	w := e.Where
	g.Town.Party.Members = members
	g.Combat = nil
	g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = w.Level, w.X, w.Y, w.Facing
	g.LightLevel, g.ProtectLevel, g.Fizzles = w.Light, w.Protect, w.Fizzles
	g.QuickPlot, g.ChestAlarm = w.QuickPlot, w.ChestAlarm
	for x := 0; x < 20; x++ {
		for y := 0; y < 20; y++ {
			g.FightMap[x][y] = w.FightMap[x][y] == '1'
		}
	}
	g.MazeMessage, g.MazeMessage2 = w.Message, w.Message2
	g.ViewportMsg, g.ViewportMsg2 = w.Viewport, w.Viewport2
	g.MazeMessages = append([]string(nil), w.Messages...)
	g.MazeMsgScroll, g.MazeMsgWait = w.MsgScroll, w.MsgWait
	g.MazeSearchYN, g.SearchMonster, g.SearchCell = w.SearchYN, w.SearchMonster, nil
	if w.SearchHere {
		g.SearchCell = g.CurrentCell()
	}
	g.MazePendingBack2Shop, g.MazePendingTryGet, g.MazeTryGetItem = w.Back2Shop, w.TryGet, w.TryGetItem
	g.WaitButton, g.ButtonCount = w.WaitButton, w.ButtonCount
	g.MazeInspecting, g.MazeInspectFound = w.Inspecting, found
	g.ShowMap, g.MazeDelayInput, g.MazeDelayBuf = false, false, ""

	town := g.Town
	town.InputMode, town.EditChar, town.InputBuf, town.TradeTarget = engine.InputNone, nil, "", nil
	town.EquipCategory, town.EquipChoices, town.EquipPartyMode, town.EquipPartyIdx = 0, nil, false, 0
	town.ReorderPos, town.ReorderResult, town.PendingSpell = 0, nil, nil
	town.MalorDeltaEW, town.MalorDeltaNS, town.MalorDeltaUD = 0, 0, 0
	town.Message = ""
	if w.Phase == "camp" {
		g.Phase = engine.PhaseCamp
		town.Message = w.CampMessage
		if w.Camp != nil {
			c.restoreCampScreen(w.Camp)
		}
	} else {
		g.Phase = engine.PhaseMaze
	}
	if e.Interrupted != "" {
		// As LeaveCombat after a victory: the encounter's messages go.
		g.MazeMessage, g.MazeMessage2 = "", ""
	}
	// R-1 (decision D-R): a party that cannot go on (an encounter saved
	// after its defeat was decided, or an older save's dead party left
	// walking, #28) is stranded at the checkpoint's square before anything
	// else can happen, as LeaveCombat would have done. The checkpoint holds
	// no random state, so doing this again after a restart gives the same
	// result.
	if !engine.PartyCanContinue(members) {
		if err := c.strand(); err != nil {
			return err
		}
		g.Expedition = nil // over: its members are stranded, not out on it
	}
	return nil
}

// checkExpedition checks a saved expedition against the roster and the
// maze and returns its members and the Inspect screen's finds.
func (c *Controller) checkExpedition(e *engine.Expedition) (members, found []*engine.Character, err error) {
	g := c.Game
	byName := func(name string) (*engine.Character, error) {
		var match *engine.Character
		for _, ch := range g.Town.Roster.Characters {
			if ch != nil && ch.Name == name {
				if match != nil {
					return nil, &ExpeditionProblem{fmt.Sprintf("two roster characters are named %s", name)}
				}
				match = ch
			}
		}
		if match == nil {
			return nil, &ExpeditionProblem{fmt.Sprintf("%s is no longer on the roster", name)}
		}
		return match, nil
	}
	if e.Version != engine.ExpeditionVersion {
		return nil, nil, &ExpeditionProblem{fmt.Sprintf("it is version %d", e.Version)}
	}
	if len(e.Party) == 0 || len(e.Party) > MaxPartySize {
		return nil, nil, &ExpeditionProblem{fmt.Sprintf("it has %d members", len(e.Party))}
	}
	seen := map[string]bool{}
	for _, name := range e.Party {
		if seen[name] {
			return nil, nil, &ExpeditionProblem{fmt.Sprintf("%s is listed twice", name)}
		}
		seen[name] = true
		ch, err := byName(name)
		if err != nil {
			return nil, nil, err
		}
		members = append(members, ch)
	}
	w := e.Where
	if w.Phase != "maze" && w.Phase != "camp" {
		return nil, nil, &ExpeditionProblem{fmt.Sprintf("unknown place %q", w.Phase)}
	}
	// A floor past the last one is accepted: the engine can put the party
	// there (the B4F-B9F elevator's H and I buttons, finding #31), and the
	// game is restored as the engine left it.
	if w.Level < 0 || w.X < 0 || w.X >= 20 || w.Y < 0 || w.Y >= 20 {
		return nil, nil, &ExpeditionProblem{fmt.Sprintf("the position L%d (%d,%d) is outside the maze", w.Level+1, w.X, w.Y)}
	}
	if w.Facing < engine.North || w.Facing > engine.West {
		return nil, nil, &ExpeditionProblem{fmt.Sprintf("unknown facing %d", w.Facing)}
	}
	if len(w.FightMap) != 20 {
		return nil, nil, &ExpeditionProblem{"the fight zones are incomplete"}
	}
	for _, row := range w.FightMap {
		if len(row) != 20 || strings.Trim(row, "01") != "" {
			return nil, nil, &ExpeditionProblem{"the fight zones are unreadable"}
		}
	}
	if e.Interrupted != "" && e.Interrupted != InterruptedCombat && e.Interrupted != InterruptedChest {
		return nil, nil, &ExpeditionProblem{fmt.Sprintf("unknown interruption %q", e.Interrupted)}
	}
	for _, name := range w.InspectFound {
		ch, err := byName(name)
		if err != nil {
			return nil, nil, err
		}
		found = append(found, ch)
	}
	if cs := w.Camp; cs != nil {
		if w.Phase != "camp" {
			return nil, nil, &ExpeditionProblem{"a Camp screen is open outside Camp"}
		}
		if _, ok := campModes[cs.Screen]; !ok {
			return nil, nil, &ExpeditionProblem{fmt.Sprintf("unknown Camp screen %q", cs.Screen)}
		}
		inParty := func(name string) bool { return name == "" || slices.Contains(e.Party, name) }
		if !inParty(cs.Member) || !inParty(cs.TradeWith) {
			return nil, nil, &ExpeditionProblem{"a Camp screen names someone outside the party"}
		}
		for _, name := range cs.Reordered {
			if !inParty(name) {
				return nil, nil, &ExpeditionProblem{"the reorder names someone outside the party"}
			}
		}
		if cs.Spell != "" && engine.LookupSpell(cs.Spell) == nil {
			return nil, nil, &ExpeditionProblem{fmt.Sprintf("unknown spell %q", cs.Spell)}
		}
		if cs.Member == "" && campNeedsMember[cs.Screen] {
			return nil, nil, &ExpeditionProblem{fmt.Sprintf("the Camp screen %q has no character", cs.Screen)}
		}
		for _, pos := range cs.EquipChoices {
			if pos < 0 || pos >= 8 {
				return nil, nil, &ExpeditionProblem{"the equip choices are unreadable"}
			}
		}
	}
	return members, found, nil
}

// campModes maps Camp screen names (as CampView names them) to modes.
var campModes = func() map[string]engine.InputMode {
	m := map[string]engine.InputMode{"menu": engine.InputNone}
	for mode, name := range campScreens {
		m[name] = mode
	}
	return m
}()

// campNeedsMember are the screens that work on one character.
var campNeedsMember = map[string]bool{
	"inspect": true, "equip": true, "drop": true, "trade": true, "tradeGold": true, "tradeItem": true,
	"spellBooks": true, "spellList": true, "castSpell": true, "spellTarget": true, "useItem": true,
	"dumapic": true, "malor": true,
}

// campScreen records the open Camp screen ("menu" at the menu) and every
// Camp field, so that even what a screen leaves behind comes back.
func (c *Controller) campScreen() (*engine.CampScreen, error) {
	town := c.Game.Town
	name, ok := campScreens[town.InputMode]
	if c.campAtMenu() {
		name, ok = "menu", true
	}
	if !ok {
		return nil, fmt.Errorf("the Camp screen (mode %d) cannot be recorded", town.InputMode)
	}
	nameOf := func(ch *engine.Character) string {
		if ch == nil {
			return ""
		}
		return ch.Name
	}
	cs := &engine.CampScreen{
		Screen: name, Member: nameOf(town.EditChar), TradeWith: nameOf(town.TradeTarget), Typed: town.InputBuf,
		EquipCategory: town.EquipCategory, EquipChoices: append([]int(nil), town.EquipChoices...),
		EquipParty: town.EquipPartyMode, EquipPartyIdx: town.EquipPartyIdx,
		ReorderPos: town.ReorderPos, PriestBook: town.InspectSpells == engine.PriestSpell,
		MalorEW: town.MalorDeltaEW, MalorNS: town.MalorDeltaNS, MalorUD: town.MalorDeltaUD,
	}
	for _, m := range town.ReorderResult {
		cs.Reordered = append(cs.Reordered, nameOf(m))
	}
	if town.PendingSpell != nil {
		cs.Spell = town.PendingSpell.Name
	}
	return cs, nil
}

// restoreCampScreen opens a recorded Camp screen with its progress. The
// expedition was checked first (checkExpedition), so the names are party
// members.
func (c *Controller) restoreCampScreen(cs *engine.CampScreen) {
	town := c.Game.Town
	member := func(name string) *engine.Character {
		for _, m := range town.Party.Members {
			if m != nil && name != "" && m.Name == name {
				return m
			}
		}
		return nil
	}
	town.InputMode = campModes[cs.Screen]
	town.EditChar, town.TradeTarget, town.InputBuf = member(cs.Member), member(cs.TradeWith), cs.Typed
	town.EquipCategory, town.EquipChoices = cs.EquipCategory, append([]int(nil), cs.EquipChoices...)
	town.EquipPartyMode, town.EquipPartyIdx = cs.EquipParty, cs.EquipPartyIdx
	town.ReorderPos = cs.ReorderPos
	for _, name := range cs.Reordered {
		town.ReorderResult = append(town.ReorderResult, member(name))
	}
	town.InspectSpells = engine.MageSpell
	if cs.PriestBook {
		town.InspectSpells = engine.PriestSpell
	}
	if cs.Spell != "" {
		town.PendingSpell = engine.LookupSpell(cs.Spell)
	}
	town.MalorDeltaEW, town.MalorDeltaNS, town.MalorDeltaUD = cs.MalorEW, cs.MalorNS, cs.MalorUD
}

// ReturnExpeditionToCastle ends a saved expedition that cannot be resumed:
// its members who are on the roster join the castle party, as a restart
// left them before expeditions were saved, and the section is cleared. If
// they do not all fit beside a party already in the castle it refuses and
// changes nothing, so no member is left out. Use only after Resume
// reported an ExpeditionProblem.
func (c *Controller) ReturnExpeditionToCastle() error {
	g := c.Game
	names := g.ExpeditionMembers()
	if g.Expedition == nil && g.OtherExpedition == nil {
		return ErrNotOnExpedition
	}
	if g.Phase != engine.PhaseTown {
		return ErrWrongState
	}
	var back []*engine.Character
	for _, name := range names {
		for _, ch := range g.Town.Roster.Characters {
			if ch != nil && ch.Name == name && !InParty(g.Town, ch) && !slices.Contains(back, ch) {
				back = append(back, ch)
				break
			}
		}
	}
	if g.Town.Party.Size()+len(back) > MaxPartySize {
		return &Refusal{Message: "** PARTY IS FULL **"}
	}
	g.Town.Party.Members = append(g.Town.Party.Members, back...)
	g.Expedition, g.OtherExpedition = nil, nil
	return nil
}
