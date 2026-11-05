package app

import (
	"strings"

	"github.com/Senior-Geek/istary/engine"
)

// Training Grounds name prompt (Pascal ROLLER). Typing a name opens an
// existing character (after its password) or offers to create a new one.
// The password check and character edit menu are still terminal-only.

// NameLookup is the outcome of entering a name at the Training Grounds.
type NameLookup int

const (
	NameLeft          NameLookup = iota // empty name: back to the castle
	NameRoster                          // "*ROSTER": roster listing shown
	NameNeedsPassword                   // existing character: password prompt
	NameOfferCreate                     // unknown name: "create it (Y/N)?"
	NameNotFound                        // unknown name where creation is not allowed (Wiz 2/3)
	NameRosterFull                      // unknown name, roster has 20 characters
)

// EnterTrainingGrounds moves from town to the Training Grounds. In the
// terminal this is reached from the Edge of Town menu.
func (c *Controller) EnterTrainingGrounds() error {
	if c.Game.Phase != engine.PhaseTown || c.Game.Town.InputMode != engine.InputNone {
		return ErrWrongState
	}
	c.Game.Town.Location = engine.Training
	return nil
}

// LeaveTrainingGrounds returns to the castle.
func (c *Controller) LeaveTrainingGrounds() error {
	if !c.inTown(engine.Training) {
		return ErrWrongState
	}
	GoToCastle(c.Game.Town)
	return nil
}

// TrainingEnterName submits a name at the Training Grounds "NAME >" prompt.
func (c *Controller) TrainingEnterName(name string) (NameLookup, error) {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.Location != engine.Training ||
		(town.InputMode != engine.InputNone && town.InputMode != engine.InputTrainingName) {
		return 0, ErrWrongState
	}
	town.InputMode = engine.InputTrainingName
	town.InputBuf = typedAtTraining(name)
	r := c.submitTrainingName()
	if r == NameRosterFull {
		return r, &Refusal{Message: town.Message}
	}
	return r, nil
}

// typedAtTraining returns name as typed from the Training Grounds menu: the
// first key opens the prompt and is kept as-is (uppercased), which is how
// "*ROSTER" gets its asterisk; later keys follow the name prompt's filter.
func typedAtTraining(name string) string {
	buf := ""
	for i, ch := range name {
		if i == 0 {
			buf = string(upper(ch))
			continue
		}
		buf = appendNameChar(buf, ch)
	}
	return buf
}

// submitTrainingName handles RETURN at the name prompt (Pascal ROLLER proc 10).
func (c *Controller) submitTrainingName() NameLookup {
	town := c.Game.Town
	name := strings.TrimSpace(town.InputBuf)
	if name == "" {
		GoToCastle(town)
		town.InputMode = engine.InputNone
		town.InputBuf = ""
		return NameLeft
	}
	if strings.EqualFold(name, "*ROSTER") {
		town.InputMode = engine.InputRoster
		town.InputBuf = ""
		return NameRoster
	}
	for _, ch := range town.Roster.Characters {
		if ch != nil && strings.EqualFold(ch.Name, name) {
			town.EditChar = ch
			// From p-code ROLLER proc 2 (IC 4826-4874):
			// Always prompt "PASSWORD >" — even for empty passwords.
			town.InputMode = engine.InputPassword
			town.InputBuf = ""
			town.Message = ""
			return NameNeedsPassword
		}
	}
	if c.Game.Scenario.ScenarioNum >= 2 {
		// Wiz 2/3: no character creation — silently re-prompt
		town.InputMode = engine.InputTrainingName
		town.InputBuf = ""
		return NameNotFound
	}
	if len(town.Roster.Characters) >= 20 {
		town.Message = "THERE IS NO ROOM LEFT - TRY DELETING"
		town.InputMode = engine.InputNone
		town.InputBuf = ""
		return NameRosterFull
	}
	// From p-code ROLLER proc 10 (IC 3164):
	// "THAT CHARACTER DOES NOT EXIST."
	// "DO YOU WANT TO CREATE IT? Y/N ? >"
	town.Message = "THAT CHARACTER DOES NOT EXIST. DO YOU"
	town.Message2 = "WANT TO CREATE IT (Y/N) ?> "
	town.PendingCreateName = strings.ToUpper(name)
	town.InputMode = engine.InputConfirmCreate
	town.InputBuf = ""
	return NameOfferCreate
}

// ConfirmCreate answers "DO YOU WANT TO CREATE IT?". Yes starts character
// creation at the password step; no returns to the name prompt.
func (c *Controller) ConfirmCreate(yes bool) error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputConfirmCreate {
		return ErrWrongState
	}
	if yes {
		town.Creation = engine.NewCreationState()
		town.Creation.Name = town.PendingCreateName
		town.Creation.Step = engine.StepPassword
		c.Game.Phase = engine.PhaseCreation
		town.Message = ""
		town.Message2 = ""
		town.PendingCreateName = ""
		town.InputMode = engine.InputNone
		return nil
	}
	town.Message = ""
	town.Message2 = ""
	town.PendingCreateName = ""
	town.InputMode = engine.InputTrainingName
	town.InputBuf = ""
	return nil
}

// CancelPasswordCheck backs out of the existing-character "PASSWORD >"
// prompt to the name prompt, as ESC does.
func (c *Controller) CancelPasswordCheck() error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputPassword {
		return ErrWrongState
	}
	town.InputMode = engine.InputTrainingName
	town.InputBuf = ""
	town.EditChar = nil
	town.Message = ""
	return nil
}

// SubmitCharacterPassword answers an existing character's "PASSWORD >"
// prompt (ROLLER proc 2). The right password opens the character edit menu;
// a wrong one returns to the name prompt with "** THATS NOT IT **".
func (c *Controller) SubmitCharacterPassword(password string) error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputPassword {
		return ErrWrongState
	}
	town.InputBuf = NormalizePassword(password)
	if !c.submitCharacterPassword() {
		return &Refusal{Message: town.Message}
	}
	return nil
}

func (c *Controller) submitCharacterPassword() bool {
	town := c.Game.Town
	if town.EditChar != nil && town.InputBuf == town.EditChar.Password {
		town.InputMode = engine.InputCharEdit
		town.InputBuf = ""
		town.Message = ""
		return true
	}
	// From p-code: CSP EXIT returns to training grounds
	town.InputMode = engine.InputTrainingName
	town.InputBuf = ""
	town.EditChar = nil
	town.Message = "** THATS NOT IT **"
	return false
}

// EditCharacter is the character open in the Training Grounds edit menu.
func (c *Controller) EditCharacter() *engine.Character {
	if c.Game.Phase != engine.PhaseTown || c.Game.Town.InputMode != engine.InputCharEdit {
		return nil
	}
	return c.Game.Town.EditChar
}

// LeaveCharacterEdit closes the edit menu and returns to the name prompt.
func (c *Controller) LeaveCharacterEdit() error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputCharEdit {
		return ErrWrongState
	}
	town.InputMode = engine.InputTrainingName
	town.InputBuf = ""
	town.EditChar = nil
	town.Message = ""
	return nil
}

// BeginDeleteCharacter asks "ARE YOU SURE YOU WANT TO DELETE (Y/N) ?" for
// the character open in the edit menu (ROLLER proc 5).
func (c *Controller) BeginDeleteCharacter() error {
	if c.EditCharacter() == nil {
		return ErrWrongState
	}
	c.Game.Town.InputMode = engine.InputConfirmDelete
	c.Game.Town.Message = ""
	return nil
}

// ConfirmDeleteCharacter answers the delete confirmation. Yes removes the
// character from the roster and party; no returns to the edit menu.
func (c *Controller) ConfirmDeleteCharacter(yes bool) error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputConfirmDelete {
		return ErrWrongState
	}
	if !yes {
		town.InputMode = engine.InputCharEdit
		town.Message = ""
		return nil
	}
	ch := town.EditChar
	for i, rc := range town.Roster.Characters {
		if rc == ch {
			town.Roster.Characters = append(town.Roster.Characters[:i], town.Roster.Characters[i+1:]...)
			break
		}
	}
	for i, m := range town.Party.Members {
		if m == ch {
			town.Party.Members = append(town.Party.Members[:i], town.Party.Members[i+1:]...)
			break
		}
	}
	town.Message = ch.Name + " DELETED"
	town.InputMode = engine.InputTrainingName
	town.InputBuf = ""
	town.EditChar = nil
	return nil
}

// CloseNamePrompt closes the Training Grounds "NAME >" prompt without
// entering a name, as ESC does.
func (c *Controller) CloseNamePrompt() error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputTrainingName {
		return ErrWrongState
	}
	town.InputMode = engine.InputNone
	town.InputBuf = ""
	town.Message = ""
	return nil
}

// CloseRoster leaves the *ROSTER listing and returns to the name prompt.
func (c *Controller) CloseRoster() error {
	town := c.Game.Town
	if c.Game.Phase != engine.PhaseTown || town.InputMode != engine.InputRoster {
		return ErrWrongState
	}
	town.InputMode = engine.InputTrainingName
	town.InputBuf = ""
	return nil
}

// trainingMenuKey handles keys at the Training Grounds with no prompt open:
// RETURN/ESC leave, any letter starts typing a name.
func (c *Controller) trainingMenuKey(k Key) {
	town := c.Game.Town
	switch k.Kind {
	case KeyEnter, KeyEscape:
		GoToCastle(town)
	case KeyRune:
		town.Message = ""
		town.InputMode = engine.InputTrainingName
		town.InputBuf = string(upper(k.Rune))
	}
}

// trainingPromptKey handles the name prompt, roster listing, create
// confirmation, existing-character password and delete confirmation.
func (c *Controller) trainingPromptKey(k Key) {
	town := c.Game.Town
	switch town.InputMode {
	case engine.InputRoster:
		// Only L exits (from p-code: key == 76 'L')
		if k.Kind == KeyRune && (k.Rune == 'l' || k.Rune == 'L') {
			c.CloseRoster()
		}

	case engine.InputConfirmCreate:
		switch {
		case k.Kind == KeyRune && (k.Rune == 'y' || k.Rune == 'Y'):
			c.ConfirmCreate(true)
		case k.Kind == KeyRune && (k.Rune == 'n' || k.Rune == 'N'), k.Kind == KeyEscape:
			c.ConfirmCreate(false)
		}

	case engine.InputPassword:
		switch k.Kind {
		case KeyBackspace:
			town.InputBuf = backspace(town.InputBuf)
		case KeyEnter:
			c.submitCharacterPassword()
		case KeyEscape:
			c.CancelPasswordCheck()
		case KeyRune:
			town.InputBuf = appendPasswordChar(town.InputBuf, k.Rune)
		}

	case engine.InputConfirmDelete:
		if k.Kind == KeyRune {
			switch upper(k.Rune) {
			case 'Y':
				c.ConfirmDeleteCharacter(true)
			case 'N':
				c.ConfirmDeleteCharacter(false)
			}
		}

	case engine.InputTrainingName:
		switch k.Kind {
		case KeyEscape:
			c.CloseNamePrompt()
		case KeyBackspace:
			town.InputBuf = backspace(town.InputBuf)
		case KeyEnter:
			c.submitTrainingName()
		case KeyRune:
			town.InputBuf = appendNameChar(town.InputBuf, k.Rune)
		}
	}
}
