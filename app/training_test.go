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
