package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// View is everything the front end renders. It is rebuilt from game state
// after every call; the front end holds no game state of its own.
type View struct {
	SaveDir    string          `json:"saveDir"`
	Fatal      string          `json:"fatal"`   // startup failure; nothing else is usable
	Error      string          `json:"error"`   // validation or refusal from the last action
	Message    string          `json:"message"` // in-game prompt text (e.g. "create it?")
	Notice     string          `json:"notice"`  // confirmation of the last action
	Screen     string          `json:"screen"`  // "roster", "confirmCreate", "creation", "combat" or "dungeon"
	Roster     []CharacterView `json:"roster"`
	Creation   *CreationView   `json:"creation"`
	Races      []string        `json:"races"`
	Alignments []string        `json:"alignments"`
	Classes    []string        `json:"classes"`
	StatNames  []string        `json:"statNames"`
	NameMax    int             `json:"nameMax"`

	// Party (Gilgamesh's Tavern).
	Party          []PartySlotView `json:"party"`
	PartyAlignment string          `json:"partyAlignment"`
	PartyMax       int             `json:"partyMax"`
	FrontRow       int             `json:"frontRow"`    // slots 1..FrontRow are the front row
	PendingJoin    string          `json:"pendingJoin"` // name waiting for its password

	// Combat (Screen "combat"): the controller's CombatView of the
	// encounter under way, and nothing else from the combat engine.
	Combat        *app.CombatView `json:"combat"`
	PracticeChest bool            `json:"practiceChest"` // the encounter was given a chest for practice

	// Dungeon (Screen "dungeon"): the controller's DungeonView of the
	// expedition under way, and nothing else from the maze.
	Dungeon    *app.DungeonView `json:"dungeon"`
	Camp       *CampPanel       `json:"camp"`       // in Camp: its screen and the party's sheets
	Expedition bool             `json:"expedition"` // an expedition (practice or real) is under way (any screen)
	Real       bool             `json:"real"`       // it is the real, saved expedition, not practice

	// Unsaved is the error of the last save that failed; it stays until a
	// save succeeds. Everything since the last successful save exists only
	// in memory until then.
	Unsaved string `json:"unsaved"`
	// Warning is a standing caution: the last save may not be durable, or
	// the save folder could not be locked.
	Warning string `json:"warning"`

	// Recovery is why a saved expedition cannot be resumed (it stays saved);
	// CanReturn says ReturnExpeditionToCastle is offered for it.
	Recovery  string `json:"recovery"`
	CanReturn bool   `json:"canReturn"`
}

// PartySlotView is one party position.
type PartySlotView struct {
	Slot   int    `json:"slot"`  // 1-based
	Index  int    `json:"index"` // roster position of this character
	Name   string `json:"name"`
	Short  string `json:"short"`
	HP     int    `json:"hp"`
	MaxHP  int    `json:"maxHp"`
	AC     int    `json:"ac"`
	Status string `json:"status"`
	Front  bool   `json:"front"`
}

type CharacterView struct {
	Index     int    `json:"index"` // roster position; stable until a delete or reset
	Name      string `json:"name"`
	Race      string `json:"race"`
	Alignment string `json:"alignment"`
	Class     string `json:"class"`
	Short     string `json:"short"` // e.g. "G-FIG"
	Level     int    `json:"level"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"maxHp"`
	AC        int    `json:"ac"`
	Status    string `json:"status"`
	Gold      int    `json:"gold"`
	Age       int    `json:"age"` // years
	Stats     [6]int `json:"stats"`

	// Details for the selected-character panel.
	XP           int        `json:"xp"`
	NextXP       int        `json:"nextXp"` // XP needed for the next level (0 if unknown)
	HasPassword  bool       `json:"hasPassword"`
	MageSpells   [7]int     `json:"mageSpells"`
	MaxMage      [7]int     `json:"maxMage"`
	PriestSpells [7]int     `json:"priestSpells"`
	MaxPriest    [7]int     `json:"maxPriest"`
	Spells       []string   `json:"spells"` // known spells
	Items        []ItemView `json:"items"`
	Poison       int        `json:"poison"`
	InMaze       bool       `json:"inMaze"`
	MazeLevel    int        `json:"mazeLevel"` // 1-based when lost in the maze
	Legacy       bool       `json:"legacy"`

	// Party membership comes from the party list (app.InParty), never from
	// InMaze. JoinRefusal is the Tavern's answer if this character tried to
	// join now ("" = may join, password still required).
	InParty     bool   `json:"inParty"`
	PartySlot   int    `json:"partySlot"` // 1-based, 0 if not in the party
	JoinRefusal string `json:"joinRefusal"`
}

// ItemView is one possession. Unidentified items show their unknown name, as
// the game does.
type ItemView struct {
	Name       string `json:"name"`
	Equipped   bool   `json:"equipped"`
	Cursed     bool   `json:"cursed"`
	Identified bool   `json:"identified"`
}

type CreationView struct {
	Step        string  `json:"step"` // name, password, race, alignment, stats, class, confirm
	Name        string  `json:"name"`
	HasPassword bool    `json:"hasPassword"`
	Race        string  `json:"race"`
	Alignment   string  `json:"alignment"`
	Class       string  `json:"class"`
	Stats       [6]int  `json:"stats"`
	BaseStats   [6]int  `json:"baseStats"`
	Points      int     `json:"points"`
	ClassOK     [8]bool `json:"classOk"`
	CanAdd      [6]bool `json:"canAdd"`
	CanRemove   [6]bool `json:"canRemove"`
	// Guide reports every class against the current stats; set from the
	// bonus-point step on.
	Guide []GuideRow `json:"guide"`
}

// GuideRow is one class in the Class Requirements Guide. Status is one of
// "eligible", "reachable", "move", "points" (not reachable even by moving
// points) or "alignment"; Note explains it in words.
type GuideRow struct {
	Class      string `json:"class"`
	Minimums   [6]int `json:"minimums"`
	Shortfall  [6]int `json:"shortfall"`
	Alignments string `json:"alignments"`
	Status     string `json:"status"`
	Note       string `json:"note"`
}

// guideRows formats app.ClassGuide for display. All decisions come from the
// controller; this only chooses words.
func guideRows(guide []app.ClassStatus, align engine.Alignment, points int) []GuideRow {
	rows := make([]GuideRow, 0, len(guide))
	for _, st := range guide {
		var names []string
		for _, a := range st.Alignments {
			names = append(names, a.String())
		}
		alignText := strings.Join(names, ", ")
		if len(st.Alignments) == 3 {
			alignText = "ANY"
		}
		row := GuideRow{
			Class:      st.Class.String(),
			Minimums:   st.Minimums,
			Shortfall:  st.Shortfall,
			Alignments: alignText,
		}
		switch {
		case st.Eligible:
			row.Status, row.Note = "eligible", "Eligible"
		case !st.AlignmentOK:
			row.Status, row.Note = "alignment", fmt.Sprintf("Not open to %s characters", align)
		case st.ReachableWithRemaining:
			row.Status, row.Note = "reachable", fmt.Sprintf("Needs %s more (%d left)", plural(st.PointsNeeded), points)
		case st.ReachableByMoving:
			row.Status, row.Note = "move", fmt.Sprintf("Needs %s, only %d left: move points from other stats", plural(st.PointsNeeded), points)
		default:
			row.Status, row.Note = "points", fmt.Sprintf("Cannot be reached: needs %s, only %d left", plural(st.PointsNeeded), points)
		}
		rows = append(rows, row)
	}
	return rows
}

func plural(n int) string {
	if n == 1 {
		return "1 point"
	}
	return fmt.Sprintf("%d points", n)
}

// pointMoves reports, for each stat, whether adding or removing a point
// would change anything, by trying the engine's own rule on a copy.
func pointMoves(cs *engine.CreationState) (canAdd, canRemove [6]bool) {
	for i := range canAdd {
		try := *cs
		try.StatCursor = i
		try.AddStatPoint()
		canAdd[i] = try.Stats != cs.Stats
		try = *cs
		try.StatCursor = i
		try.RemoveStatPoint()
		canRemove[i] = try.Stats != cs.Stats
	}
	return canAdd, canRemove
}

func knownSpells(ch *engine.Character) []string {
	var names []string
	for i, known := range ch.SpellKnown {
		if known && i < len(engine.SpellTable) {
			names = append(names, engine.SpellTable[i].Name)
		}
	}
	return names
}

func itemViews(ch *engine.Character, items []data.Item) []ItemView {
	var out []ItemView
	for i := 0; i < ch.ItemCount && i < len(ch.Items); i++ {
		p := ch.Items[i]
		name := fmt.Sprintf("ITEM #%d", p.ItemIndex)
		if p.ItemIndex >= 0 && p.ItemIndex < len(items) {
			name = items[p.ItemIndex].Name
			if !p.Identified && items[p.ItemIndex].NameUnknown != "" {
				name = items[p.ItemIndex].NameUnknown
			}
		}
		out = append(out, ItemView{Name: name, Equipped: p.Equipped, Cursed: p.Cursed && p.Identified, Identified: p.Identified})
	}
	return out
}

// mazeLevel is the 1-based dungeon level to show for a character out in the
// maze, or 0. Display only: joining the party sets InMaze (as in the
// original), so party members are never shown as in the maze; their saved
// InMaze flag is left as the game set it.
func mazeLevel(ch *engine.Character, inParty bool) int {
	if inParty {
		return 0
	}
	if ch.InMaze || ch.MazeLevel != 0 {
		return ch.MazeLevel + 1
	}
	return 0
}

var stepNames = map[engine.CreationStep]string{
	engine.StepName:      "name",
	engine.StepPassword:  "password",
	engine.StepRace:      "race",
	engine.StepAlignment: "alignment",
	engine.StepStats:     "stats",
	engine.StepClass:     "class",
	engine.StepConfirm:   "confirm",
}

// view builds the View; err is the last action's error, if any.
func (s *Session) view(err error) View {
	v := View{
		SaveDir:    s.saveDir,
		Races:      engine.RaceNames[:],
		Alignments: engine.AlignmentNames[:],
		Classes:    engine.ClassNames[:],
		StatNames:  engine.StatNames[:],
		NameMax:    app.MaxNameLength,
		Screen:     "roster",
		Notice:     s.notice,
	}
	if s.startErr != nil {
		v.Fatal = s.startErr.Error()
		return v
	}
	v.Error = errorText(err)
	v.Unsaved = s.unsaved
	v.Warning = strings.TrimSpace(s.uncertain + " " + s.lockNote)
	v.Recovery = s.problem
	v.CanReturn = s.problem != "" && s.ctrl.Game.Expedition != nil
	if c := s.active(); c != nil {
		v.Expedition, v.Real = s.expedition || s.real, s.real
		if cv, cerr := c.CombatView(); cerr == nil {
			v.Screen, v.Combat, v.PracticeChest = "combat", &cv, s.practiceChest && !s.real
		} else if v.Expedition {
			dv := c.DungeonView()
			v.Screen, v.Dungeon = "dungeon", &dv
			if c.Game.Phase == engine.PhaseCamp {
				v.Camp = campPanel(c)
			}
		}
	}

	g := s.ctrl.Game
	for i, ch := range g.Town.Roster.Characters {
		if ch == nil {
			continue
		}
		cv := characterView(g, ch)
		cv.Index, cv.JoinRefusal = i, s.ctrl.JoinRefusal(ch)
		v.Roster = append(v.Roster, cv)
	}

	v.PartyAlignment = app.PartyAlignment(g.Town).String()
	v.PartyMax = app.MaxPartySize
	v.FrontRow = app.FrontRowSize
	for i, m := range g.Town.Party.Members {
		if m == nil {
			continue
		}
		index := -1
		for ri, rc := range g.Town.Roster.Characters {
			if rc == m {
				index = ri
			}
		}
		v.Party = append(v.Party, PartySlotView{
			Slot: i + 1, Index: index, Name: m.Name, Short: engine.AlignClass(m.Alignment, m.Class),
			HP: m.HP, MaxHP: m.MaxHP, AC: m.AC, Status: m.Status.String(), Front: i < app.FrontRowSize,
		})
	}
	if g.Phase == engine.PhaseTown && g.Town.InputMode == engine.InputTavernPassword && g.Town.EditChar != nil {
		v.PendingJoin = g.Town.EditChar.Name
	}

	switch {
	case g.Phase == engine.PhaseTown && g.Town.InputMode == engine.InputConfirmCreate:
		v.Screen = "confirmCreate"
		msg := strings.Join(strings.Fields(g.Town.Message+" "+g.Town.Message2), " ")
		v.Message = strings.Replace(msg, " (Y/N) ?>", "?", 1)
	case s.ctrl.Creation() != nil:
		cs := s.ctrl.Creation()
		v.Screen = "creation"
		cv := &CreationView{
			Step:        stepNames[cs.Step],
			Name:        cs.Name,
			HasPassword: cs.Password != "",
			Stats:       cs.Stats,
			BaseStats:   engine.RaceBaseStats[cs.Race],
			Points:      cs.BonusPoints,
			ClassOK:     cs.ClassAvailability(),
		}
		cv.CanAdd, cv.CanRemove = pointMoves(cs)
		if guide, err := s.ctrl.ClassGuide(); err == nil {
			cv.Guide = guideRows(guide, cs.Alignment, cs.BonusPoints)
		}
		if cs.Step > engine.StepRace {
			cv.Race = cs.Race.String()
		}
		if cs.Step > engine.StepAlignment {
			cv.Alignment = cs.Alignment.String()
		}
		if cs.Step == engine.StepConfirm {
			cv.Class = cs.Class.String()
		}
		v.Creation = cv
	}
	return v
}

// characterView is ch's sheet in game g (the saved game or a practice
// game). Index and JoinRefusal are the roster's to set.
func characterView(g *engine.GameState, ch *engine.Character) CharacterView {
	inParty := app.InParty(g.Town, ch)
	return CharacterView{
		Index:     -1,
		Name:      ch.Name,
		Race:      ch.Race.String(),
		Alignment: ch.Alignment.String(),
		Class:     ch.Class.String(),
		Short:     engine.AlignClass(ch.Alignment, ch.Class),
		Level:     ch.Level,
		HP:        ch.HP,
		MaxHP:     ch.MaxHP,
		AC:        ch.AC,
		Status:    ch.Status.String(),
		Gold:      ch.Gold,
		Age:       ch.Age / 52,
		Stats:     [6]int{ch.Strength, ch.IQ, ch.Piety, ch.Vitality, ch.Agility, ch.Luck},

		XP:           ch.XP,
		NextXP:       engine.XPForNextLevel(ch, g),
		HasPassword:  ch.Password != "",
		MageSpells:   ch.MageSpells,
		MaxMage:      ch.MaxMageSpells,
		PriestSpells: ch.PriestSpells,
		MaxPriest:    ch.MaxPriestSpells,
		Spells:       knownSpells(ch),
		Items:        itemViews(ch, g.Scenario.Items),
		Poison:       ch.PoisonAmt,
		InMaze:       ch.InMaze,
		MazeLevel:    mazeLevel(ch, inParty),
		Legacy:       ch.IsLegacy,
		InParty:      inParty,
		PartySlot:    app.PartySlot(g.Town, ch) + 1,
	}
}

// errorText turns controller errors into messages for the player.
func errorText(err error) string {
	if err == nil {
		return ""
	}
	var refusal *app.Refusal
	switch {
	case errors.As(err, &refusal):
		return refusal.Message
	case errors.Is(err, app.ErrPasswordMismatch):
		return "The passwords do not match. Enter the password again."
	case errors.Is(err, app.ErrPointsRemaining):
		return "Spend all bonus points before choosing a class."
	case errors.Is(err, app.ErrNoClassAvailable):
		return "These statistics do not qualify for any class. Move points to other statistics."
	case errors.Is(err, app.ErrClassUnavailable):
		return "Your statistics and alignment do not qualify for that class."
	case errors.Is(err, app.ErrEmptyName):
		return "Enter a character name."
	case errors.Is(err, app.ErrInvalidName):
		return "Names may use up to 15 letters, digits, spaces, hyphens and apostrophes."
	case errors.Is(err, app.ErrNoSuchCharacter):
		return "That character is no longer on the roster."
	case errors.Is(err, app.ErrPartyEmpty):
		return "The party is empty."
	case errors.Is(err, app.ErrNoSuchPartyMember):
		return "No party member in that slot."
	case errors.Is(err, app.ErrWrongState):
		return "That action is not available right now."
	}
	return err.Error()
}
