package main

import (
	"errors"
	"strings"

	"github.com/Senior-Geek/istary/app"
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
	Screen     string          `json:"screen"`  // "roster", "confirmCreate" or "creation"
	Roster     []CharacterView `json:"roster"`
	Creation   *CreationView   `json:"creation"`
	Races      []string        `json:"races"`
	Alignments []string        `json:"alignments"`
	Classes    []string        `json:"classes"`
	StatNames  []string        `json:"statNames"`
	NameMax    int             `json:"nameMax"`
}

type CharacterView struct {
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

	g := s.ctrl.Game
	for _, ch := range g.Town.Roster.Characters {
		if ch == nil {
			continue
		}
		v.Roster = append(v.Roster, CharacterView{
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
		})
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
	case errors.Is(err, app.ErrWrongState):
		return "That action is not available right now."
	}
	return err.Error()
}
