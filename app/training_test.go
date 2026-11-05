package app

import (
	"errors"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

func atTraining(t *testing.T, num int) *Controller {
	t.Helper()
	c := newController(t, num)
	if err := c.EnterTrainingGrounds(); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTrainingNameLookups(t *testing.T) {
	c := atTraining(t, 1)
	existing := member("BILBO", engine.Neutral)
	c.Game.Town.Roster.Add(existing)

	if r, err := c.TrainingEnterName("*roster"); err != nil || r != NameRoster {
		t.Fatalf("*ROSTER: %v %v", r, err)
	}
	if err := c.CloseRoster(); err != nil || c.Game.Town.InputMode != engine.InputTrainingName {
		t.Fatalf("CloseRoster: %v mode=%v", err, c.Game.Town.InputMode)
	}

	if r, err := c.TrainingEnterName("bilbo"); err != nil || r != NameNeedsPassword || c.Game.Town.EditChar != existing {
		t.Fatalf("existing name: %v %v", r, err)
	}
	if err := c.CancelPasswordCheck(); err != nil || c.Game.Town.InputMode != engine.InputTrainingName || c.Game.Town.EditChar != nil {
		t.Fatalf("CancelPasswordCheck: %v mode=%v", err, c.Game.Town.InputMode)
	}

	if r, err := c.TrainingEnterName("new one"); err != nil || r != NameOfferCreate || c.Game.Town.PendingCreateName != "NEW ONE" {
		t.Fatalf("unknown name: %v %v pending=%q", r, err, c.Game.Town.PendingCreateName)
	}
	if err := c.ConfirmCreate(false); err != nil || c.Game.Town.InputMode != engine.InputTrainingName {
		t.Fatalf("decline create: %v", err)
	}
	c.TrainingEnterName("new one")
	if err := c.ConfirmCreate(true); err != nil {
		t.Fatal(err)
	}
	cs := c.Creation()
	if cs == nil || cs.Name != "NEW ONE" || cs.Step != engine.StepPassword {
		t.Fatalf("creation not started: %+v", cs)
	}
}

func TestTrainingEmptyNameLeaves(t *testing.T) {
	c := atTraining(t, 1)
	if r, err := c.TrainingEnterName("   "); err != nil || r != NameLeft || c.Game.Town.Location != engine.Castle {
		t.Errorf("blank name: %v %v location=%v", r, err, c.Game.Town.Location)
	}
}

func TestTrainingLaterScenariosCannotCreate(t *testing.T) {
	for _, num := range []int{2, 3} {
		c := atTraining(t, num)
		if r, err := c.TrainingEnterName("NEWBIE"); err != nil || r != NameNotFound {
			t.Errorf("wiz%d: %v %v", num, r, err)
		}
	}
}

func TestTrainingRosterFull(t *testing.T) {
	c := atTraining(t, 1)
	for i := 0; i < 20; i++ {
		c.Game.Town.Roster.Add(member(string(rune('A'+i)), engine.Good))
	}
	r, err := c.TrainingEnterName("ONEMORE")
	var refusal *Refusal
	if r != NameRosterFull || !errors.As(err, &refusal) || refusal.Message != "THERE IS NO ROOM LEFT - TRY DELETING" {
		t.Errorf("roster full: %v %v", r, err)
	}
}

func TestTrainingPreconditions(t *testing.T) {
	c := newController(t, 1) // at the castle
	if _, err := c.TrainingEnterName("X"); !errors.Is(err, ErrWrongState) {
		t.Errorf("name outside Training Grounds: %v", err)
	}
	if err := c.ConfirmCreate(true); !errors.Is(err, ErrWrongState) {
		t.Errorf("ConfirmCreate without prompt: %v", err)
	}
	if err := c.LeaveTrainingGrounds(); !errors.Is(err, ErrWrongState) {
		t.Errorf("leave when not there: %v", err)
	}
}

// Typing at the prompt and calling TrainingEnterName reach the same state.
func TestTrainingKeysMatchOperations(t *testing.T) {
	byKey := atTraining(t, 1)
	byOp := atTraining(t, 1)
	for _, name := range []string{"*roster", "!odd name"} {
		keys(t, byKey, name)
		press(t, byKey, KeyEnter)
		byOp.TrainingEnterName(name)
		if a, b := snapshot(t, byKey.Game.Town), snapshot(t, byOp.Game.Town); a != b {
			t.Fatalf("%q: keys and operations differ:\nkeys %s\n ops %s", name, a, b)
		}
		if name == "*roster" {
			keys(t, byKey, "l")
			byOp.CloseRoster()
		} else {
			keys(t, byKey, "n")
			byOp.ConfirmCreate(false)
		}
		press(t, byKey, KeyEscape) // close the name prompt
		byOp.Game.Town.InputMode, byOp.Game.Town.InputBuf, byOp.Game.Town.Message = engine.InputNone, "", ""
	}
	keys(t, byKey, "gimli")
	press(t, byKey, KeyEnter)
	keys(t, byKey, "y")
	byOp.TrainingEnterName("gimli")
	byOp.ConfirmCreate(true)
	if a, b := snapshot(t, byKey.Game.Town), snapshot(t, byOp.Game.Town); a != b {
		t.Errorf("keys and operations differ:\nkeys %s\n ops %s", a, b)
	}
	if byKey.Game.Phase != engine.PhaseCreation || byOp.Game.Phase != engine.PhaseCreation {
		t.Error("both paths should be in creation")
	}
}

// openForEdit looks up name and answers its password prompt.
func openForEdit(t *testing.T, c *Controller, name, password string) error {
	t.Helper()
	if r, err := c.TrainingEnterName(name); err != nil || r != NameNeedsPassword {
		t.Fatalf("lookup %s: %v %v", name, r, err)
	}
	return c.SubmitCharacterPassword(password)
}

func TestCharacterPasswordAndDelete(t *testing.T) {
	c := atTraining(t, 1)
	keep := member("KEEP", engine.Good)
	doomed := member("DOOMED", engine.Good)
	doomed.Password = "BYE"
	c.Game.Town.Roster.Add(keep)
	c.Game.Town.Roster.Add(doomed)
	c.Game.Town.Party.Members = []*engine.Character{doomed}

	// Wrong password refuses and returns to the name prompt.
	err := openForEdit(t, c, "doomed", "nope")
	var r *Refusal
	if !errors.As(err, &r) || r.Message != "** THATS NOT IT **" || c.Game.Town.InputMode != engine.InputTrainingName {
		t.Fatalf("wrong password: %v mode=%v", err, c.Game.Town.InputMode)
	}
	if err := c.BeginDeleteCharacter(); !errors.Is(err, ErrWrongState) {
		t.Errorf("delete without opening the character: %v", err)
	}

	// Right password opens the edit menu; "no" keeps the character.
	if err := openForEdit(t, c, "doomed", "bye"); err != nil || c.EditCharacter() != doomed {
		t.Fatalf("right password: %v", err)
	}
	c.BeginDeleteCharacter()
	if err := c.ConfirmDeleteCharacter(false); err != nil || c.EditCharacter() != doomed || len(c.Game.Town.Roster.Characters) != 2 {
		t.Fatalf("decline delete: %v", err)
	}

	// "Yes" removes it from roster and party.
	c.BeginDeleteCharacter()
	if err := c.ConfirmDeleteCharacter(true); err != nil {
		t.Fatal(err)
	}
	if len(c.Game.Town.Roster.Characters) != 1 || c.Game.Town.Roster.Characters[0] != keep ||
		c.Game.Town.Party.Size() != 0 || c.Game.Town.Message != "DOOMED DELETED" ||
		c.Game.Town.InputMode != engine.InputTrainingName {
		t.Errorf("after delete: roster=%d party=%d msg=%q mode=%v", len(c.Game.Town.Roster.Characters),
			c.Game.Town.Party.Size(), c.Game.Town.Message, c.Game.Town.InputMode)
	}

	// Leaving the edit menu.
	openForEdit(t, c, "keep", "")
	if err := c.LeaveCharacterEdit(); err != nil || c.Game.Town.InputMode != engine.InputTrainingName || c.Game.Town.EditChar != nil {
		t.Errorf("leave edit: %v", err)
	}
	if err := c.LeaveCharacterEdit(); !errors.Is(err, ErrWrongState) {
		t.Errorf("leave edit twice: %v", err)
	}
}

// Keys and operations reach the same state for password and delete.
func TestDeleteKeysMatchOperations(t *testing.T) {
	mk := func() *Controller {
		c := atTraining(t, 1)
		d := member("GONE", engine.Neutral)
		d.Password = "PW"
		c.Game.Town.Roster.Add(d)
		c.Game.Town.Roster.Add(member("STAY", engine.Neutral))
		return c
	}
	byKey, byOp := mk(), mk()
	keys(t, byKey, "gone")
	press(t, byKey, KeyEnter)
	keys(t, byKey, "pw")
	press(t, byKey, KeyEnter)
	byOp.TrainingEnterName("gone")
	byOp.SubmitCharacterPassword("pw")
	if a, b := snapshot(t, byKey.Game.Town), snapshot(t, byOp.Game.Town); a != b {
		t.Fatalf("after password:\nkeys %s\n ops %s", a, b)
	}
	// D is still handled by the terminal edit menu, so drive the op directly
	// on both and compare the Y/N key against ConfirmDeleteCharacter.
	byKey.BeginDeleteCharacter()
	byOp.BeginDeleteCharacter()
	keys(t, byKey, "y")
	byOp.ConfirmDeleteCharacter(true)
	if a, b := snapshot(t, byKey.Game.Town), snapshot(t, byOp.Game.Town); a != b {
		t.Fatalf("after delete:\nkeys %s\n ops %s", a, b)
	}
}
