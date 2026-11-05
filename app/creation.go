package app

import "github.com/Senior-Geek/istary/engine"

// Character creation (Pascal ROLLER). Steps run Password -> Race ->
// Alignment -> Stats -> Class -> Confirm; the Name step is only reached by
// stepping back from Race. Rules (base stats, bonus points, class
// requirements, starting HP/gold/spells) live in engine.CreationState.

// Creation returns the creation in progress, or nil.
func (c *Controller) Creation() *engine.CreationState {
	if c.Game.Phase != engine.PhaseCreation {
		return nil
	}
	return c.Game.Town.Creation
}

func (c *Controller) creationAt(step engine.CreationStep) (*engine.CreationState, error) {
	cs := c.Creation()
	if cs == nil || cs.Step != step {
		return nil, ErrWrongState
	}
	return cs, nil
}

// CancelCreation abandons creation from the name step and returns to town.
func (c *Controller) CancelCreation() error {
	if _, err := c.creationAt(engine.StepName); err != nil {
		return err
	}
	c.Game.Phase = engine.PhaseTown
	c.Game.Town.Creation = nil
	return nil
}

// SetCreationName sets the character's name and moves on to race selection.
func (c *Controller) SetCreationName(name string) error {
	cs, err := c.creationAt(engine.StepName)
	if err != nil {
		return err
	}
	cs.Name = NormalizeName(name)
	return c.submitCreationName(cs)
}

func (c *Controller) submitCreationName(cs *engine.CreationState) error {
	if len(cs.Name) == 0 {
		return ErrEmptyName
	}
	cs.Step = engine.StepRace
	cs.SelectedIndex = 0
	return nil
}

// SetCreationPassword sets the password; confirm must match. An empty
// password is allowed. On mismatch password entry restarts.
func (c *Controller) SetCreationPassword(password, confirm string) error {
	cs, err := c.creationAt(engine.StepPassword)
	if err != nil {
		return err
	}
	cs.PasswordFirst = NormalizePassword(password)
	cs.PasswordStep = 1
	cs.Password = NormalizePassword(confirm)
	return confirmCreationPassword(cs)
}

// confirmCreationPassword checks the second entry against the first.
func confirmCreationPassword(cs *engine.CreationState) error {
	if cs.Password == cs.PasswordFirst {
		cs.Password = cs.PasswordFirst
		cs.Step = engine.StepRace
		return nil
	}
	cs.PasswordStep = 0
	cs.PasswordFirst = ""
	cs.Password = ""
	return ErrPasswordMismatch
}

// ChooseRace picks the race and moves on to alignment.
func (c *Controller) ChooseRace(r engine.Race) error {
	cs, err := c.creationAt(engine.StepRace)
	if err != nil {
		return err
	}
	if r < engine.Human || r > engine.Hobbit {
		return ErrInvalidChoice
	}
	cs.Race = r
	cs.Step = engine.StepAlignment
	cs.SelectedIndex = 0
	return nil
}

// ChooseAlignment picks the alignment, sets race base stats, rolls bonus
// points and moves on to point allocation.
func (c *Controller) ChooseAlignment(a engine.Alignment) error {
	cs, err := c.creationAt(engine.StepAlignment)
	if err != nil {
		return err
	}
	if a < engine.Good || a > engine.Evil {
		return ErrInvalidChoice
	}
	cs.Alignment = a
	cs.InitStats()
	cs.Step = engine.StepStats
	return nil
}

// SelectStat moves the allocation cursor to stat i (0=STR ... 5=LUCK).
func (c *Controller) SelectStat(i int) error {
	cs, err := c.creationAt(engine.StepStats)
	if err != nil {
		return err
	}
	if i < 0 || i > 5 {
		return ErrInvalidChoice
	}
	cs.StatCursor = i
	return nil
}

// NextStat moves the cursor to the next stat, wrapping LUCK -> STRENGTH.
func (c *Controller) NextStat() error {
	cs, err := c.creationAt(engine.StepStats)
	if err != nil {
		return err
	}
	if cs.StatCursor < 5 {
		cs.StatCursor++
	} else {
		cs.StatCursor = 0
	}
	return nil
}

// AddStatPoint spends one bonus point on the selected stat (max 18).
func (c *Controller) AddStatPoint() error {
	cs, err := c.creationAt(engine.StepStats)
	if err != nil {
		return err
	}
	cs.AddStatPoint()
	return nil
}

// RemoveStatPoint refunds one point from the selected stat (min race base).
func (c *Controller) RemoveStatPoint() error {
	cs, err := c.creationAt(engine.StepStats)
	if err != nil {
		return err
	}
	cs.RemoveStatPoint()
	return nil
}

// FinishStats moves on to class selection once every bonus point is spent
// and at least one class qualifies.
func (c *Controller) FinishStats() error {
	cs, err := c.creationAt(engine.StepStats)
	if err != nil {
		return err
	}
	// Pascal line 311: UNTIL (INCHAR = CHR(27)) AND CANCHG AND (PTSLEFT = 0)
	if cs.BonusPoints != 0 {
		return ErrPointsRemaining
	}
	for _, ok := range cs.ClassAvailability() {
		if ok {
			cs.Step = engine.StepClass
			return nil
		}
	}
	return ErrNoClassAvailable
}

// ChooseClass picks a class the stats and alignment qualify for.
func (c *Controller) ChooseClass(class engine.Class) error {
	cs, err := c.creationAt(engine.StepClass)
	if err != nil {
		return err
	}
	if class < engine.Fighter || class > engine.Ninja {
		return ErrInvalidChoice
	}
	if !cs.ClassAvailability()[class] {
		return ErrClassUnavailable
	}
	cs.Class = class
	cs.Step = engine.StepConfirm
	return nil
}

// CreationBack returns to the previous step from race, alignment or class.
func (c *Controller) CreationBack() error {
	cs := c.Creation()
	if cs == nil {
		return ErrWrongState
	}
	switch cs.Step {
	case engine.StepRace:
		cs.Step = engine.StepName
	case engine.StepAlignment:
		cs.Step = engine.StepRace
	case engine.StepClass:
		cs.Step = engine.StepStats
	default:
		return ErrWrongState
	}
	return nil
}

// ConfirmCreation keeps or discards the finished character. Keeping it rolls
// HP, gold and age, adds it to the roster and returns it. A reroll from the
// Training Grounds edit menu returns to that menu with the new character.
func (c *Controller) ConfirmCreation(keep bool) (*engine.Character, error) {
	cs, err := c.creationAt(engine.StepConfirm)
	if err != nil {
		return nil, err
	}
	town := c.Game.Town
	if !keep {
		town.Creation = nil
		c.Game.Phase = engine.PhaseTown
		return nil, nil
	}
	reroll := cs.Reroll
	ch := cs.FinalizeCharacter()
	town.Roster.Add(ch)
	town.Message = ""
	town.Creation = nil
	c.Game.Phase = engine.PhaseTown
	if reroll {
		town.EditChar = ch
		town.InputMode = engine.InputCharEdit
	}
	return ch, nil
}

// creationKey is the terminal key map for the creation screens.
func (c *Controller) creationKey(k Key) {
	cs := c.Game.Town.Creation
	if cs == nil {
		c.Game.Phase = engine.PhaseTown
		return
	}

	switch cs.Step {
	case engine.StepName:
		switch k.Kind {
		case KeyEscape:
			c.CancelCreation()
		case KeyBackspace:
			cs.Name = backspace(cs.Name)
		case KeyEnter:
			c.submitCreationName(cs)
		case KeyRune:
			cs.Name = appendNameChar(cs.Name, k.Rune)
		}

	case engine.StepPassword:
		// From p-code ROLLER proc 18: type, RETURN, type again to confirm.
		switch k.Kind {
		case KeyBackspace:
			if cs.PasswordStep == 0 {
				cs.PasswordFirst = backspace(cs.PasswordFirst)
			} else {
				cs.Password = backspace(cs.Password)
			}
		case KeyEnter:
			if cs.PasswordStep == 0 {
				if len(cs.PasswordFirst) > MaxNameLength {
					cs.PasswordFirst = cs.PasswordFirst[:MaxNameLength]
				}
				cs.PasswordStep = 1
				cs.Password = ""
			} else {
				confirmCreationPassword(cs)
			}
		case KeyRune:
			if cs.PasswordStep == 0 {
				cs.PasswordFirst = appendPasswordChar(cs.PasswordFirst, k.Rune)
			} else {
				cs.Password = appendPasswordChar(cs.Password, k.Rune)
			}
		}

	case engine.StepRace:
		// From p-code ROLLER proc 17: letter A-E selects race
		if k.Kind == KeyEscape {
			c.CreationBack()
		} else if k.Kind == KeyRune {
			if ch := upper(k.Rune); ch >= 'A' && ch <= 'E' {
				c.ChooseRace(engine.Race(ch - 'A'))
			}
		}

	case engine.StepAlignment:
		// From p-code ROLLER proc 15: letter A-C selects alignment
		if k.Kind == KeyEscape {
			c.CreationBack()
		} else if k.Kind == KeyRune {
			if ch := upper(k.Rune); ch >= 'A' && ch <= 'C' {
				c.ChooseAlignment(engine.Alignment(ch - 'A'))
			}
		}

	case engine.StepStats:
		// Pascal GIVEPTS: +/- adjust, RET next stat, ESC done
		switch k.Kind {
		case KeyEscape:
			c.FinishStats()
		case KeyEnter:
			c.NextStat()
		case KeyRune:
			switch k.Rune {
			case '+', ';': // ';' is the Apple II unshifted '+'
				c.AddStatPoint()
			case '-', '=':
				c.RemoveStatPoint()
			}
		}

	case engine.StepClass:
		// From p-code ROLLER proc 14: A-H with fixed indices
		if k.Kind == KeyEscape {
			c.CreationBack()
		} else if k.Kind == KeyRune {
			if idx := int(upper(k.Rune) - 'A'); idx >= 0 && idx < 8 {
				c.ChooseClass(engine.Class(idx))
			}
		}

	case engine.StepConfirm:
		// From p-code ROLLER proc 13: Y/N only
		if k.Kind == KeyRune {
			switch k.Rune {
			case 'y', 'Y':
				c.ConfirmCreation(true)
			case 'n', 'N':
				c.ConfirmCreation(false)
			}
		}
	}
}
