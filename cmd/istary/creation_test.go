package main

import (
	"encoding/json"
	"testing"

	"github.com/Senior-Geek/istary/engine"
	"github.com/gdamore/tcell/v2"
)

// toTrainingName walks Castle -> Edge of Town -> Training Grounds and types a
// name at the "NAME >" prompt.
func toTrainingName(d *driver, name string) *driver {
	d.keys("ET")
	if d.game.Town.Location != engine.Training {
		d.t.Fatalf("location = %v, want Training Grounds", d.game.Town.Location)
	}
	return d.keys(name)
}

// spendPoints presses '+' on each stat in turn, starting at the current
// cursor, until the bonus pool is empty, moving on with RETURN when a stat
// hits 18.
func spendPoints(d *driver) {
	cs := d.game.Town.Creation
	for i := 0; i < 6 && cs.BonusPoints > 0; i++ {
		for cs.BonusPoints > 0 {
			before := cs.Stats[cs.StatCursor]
			d.keys("+")
			if cs.Stats[cs.StatCursor] == before {
				break
			}
		}
		d.enter()
	}
}

func TestCreateCharacterThroughTrainingGrounds(t *testing.T) {
	g := newGame(t, 1)
	d := newDriver(t, g)
	seed(42)

	toTrainingName(d, "gandalf").enter()
	if g.Town.InputMode != engine.InputConfirmCreate || g.Town.PendingCreateName != "GANDALF" {
		t.Fatalf("unknown name should prompt to create: mode=%v pending=%q", g.Town.InputMode, g.Town.PendingCreateName)
	}

	d.keys("Y")
	cs := g.Town.Creation
	if g.Phase != engine.PhaseCreation || cs == nil || cs.Step != engine.StepPassword || cs.Name != "GANDALF" {
		t.Fatalf("Y should start creation at the password step: phase=%v creation=%+v", g.Phase, cs)
	}

	d.keys("pw").enter().keys("pw").enter()
	if cs.Step != engine.StepRace || cs.Password != "PW" {
		t.Fatalf("matching passwords should advance to race: step=%v password=%q", cs.Step, cs.Password)
	}

	d.keys("b") // ELF
	if cs.Race != engine.Elf || cs.Step != engine.StepAlignment {
		t.Fatalf("race = %v step = %v", cs.Race, cs.Step)
	}

	d.keys("a") // GOOD
	if cs.Alignment != engine.Good || cs.Step != engine.StepStats || cs.Stats != engine.RaceBaseStats[engine.Elf] {
		t.Fatalf("alignment = %v step = %v stats = %v", cs.Alignment, cs.Step, cs.Stats)
	}
	if cs.BonusPoints < 7 {
		t.Fatalf("bonus points = %d, want >= 7", cs.BonusPoints)
	}

	// ESC is ignored while bonus points remain.
	d.esc()
	if cs.Step != engine.StepStats {
		t.Fatal("ESC with unspent points should stay on the stats step")
	}

	d.enter() // start on I.Q. so the elf qualifies as a Mage
	spendPoints(d)
	d.esc()
	if cs.Step != engine.StepClass {
		t.Fatalf("ESC with 0 points should go to class selection, step = %v", cs.Step)
	}

	// A class the stats don't qualify for is ignored; Mage (B) is available.
	d.keys("h")
	if cs.Step != engine.StepClass {
		t.Fatal("Ninja should not be selectable for a good elf")
	}
	d.keys("b")
	if cs.Class != engine.Mage || cs.Step != engine.StepConfirm {
		t.Fatalf("class = %v step = %v", cs.Class, cs.Step)
	}

	d.keys("y")
	if g.Phase != engine.PhaseTown || g.Town.Creation != nil {
		t.Fatalf("after confirm: phase=%v creation=%v", g.Phase, g.Town.Creation)
	}
	if len(g.Town.Roster.Characters) != 1 {
		t.Fatalf("roster has %d characters, want 1", len(g.Town.Roster.Characters))
	}

	// Golden: the full character created by this seeded key sequence.
	raw, _ := json.MarshalIndent(g.Town.Roster.Characters[0], "", "  ")
	checkGolden(t, "created_gandalf.golden.json", append(raw, '\n'))
}

func TestCreationNameInputFiltering(t *testing.T) {
	g := newGame(t, 1)
	d := newDriver(t, g)
	toTrainingName(d, "a-b'c 9!@").enter()
	if g.Town.PendingCreateName != "A-B'C 9" {
		t.Errorf("pending name = %q, want %q (letters uppercased, ! and @ dropped)", g.Town.PendingCreateName, "A-B'C 9")
	}
	d.keys("n")
	if g.Town.InputMode != engine.InputTrainingName || g.Town.Creation != nil {
		t.Errorf("N should return to the name prompt: mode=%v", g.Town.InputMode)
	}

	// Names are limited to 15 characters.
	d.keys("ABCDEFGHIJKLMNOPQRS")
	if got := g.Town.InputBuf; got != "ABCDEFGHIJKLMNO" {
		t.Errorf("InputBuf = %q, want first 15 characters", got)
	}
}

func TestCreationPasswordMismatchRestarts(t *testing.T) {
	g := newGame(t, 1)
	d := newDriver(t, g)
	toTrainingName(d, "X").enter().keys("Y")
	cs := g.Town.Creation
	d.keys("ONE").enter().keys("TWO").enter()
	if cs.Step != engine.StepPassword || cs.PasswordStep != 0 || cs.PasswordFirst != "" {
		t.Errorf("mismatch should restart password entry: step=%v pwstep=%d first=%q", cs.Step, cs.PasswordStep, cs.PasswordFirst)
	}
	// Empty password is allowed.
	d.enter().enter()
	if cs.Step != engine.StepRace || cs.Password != "" {
		t.Errorf("empty password should be accepted: step=%v password=%q", cs.Step, cs.Password)
	}
}

func TestCreationDeclineDiscardsCharacter(t *testing.T) {
	g := newGame(t, 1)
	d := newDriver(t, g)
	seed(5)
	toTrainingName(d, "TEMP").enter().keys("Y").enter().enter().keys("A").keys("B") // human neutral
	spendPoints(d)
	d.esc().keys("A").keys("N") // fighter, then decline
	if g.Phase != engine.PhaseTown || g.Town.Creation != nil || len(g.Town.Roster.Characters) != 0 {
		t.Errorf("N at confirm should discard: phase=%v roster=%d", g.Phase, len(g.Town.Roster.Characters))
	}
}

func TestCreationEscapeStepsBack(t *testing.T) {
	g := newGame(t, 1)
	d := newDriver(t, g)
	toTrainingName(d, "BACK").enter().keys("Y").enter().enter()
	cs := g.Town.Creation
	d.keys("C") // GNOME
	if cs.Step != engine.StepAlignment {
		t.Fatalf("step = %v, want alignment after picking race", cs.Step)
	}
	d.esc()
	if cs.Step != engine.StepRace {
		t.Errorf("ESC at alignment should return to race, step = %v", cs.Step)
	}
	d.esc()
	if cs.Step != engine.StepName {
		t.Errorf("ESC at race should return to name, step = %v", cs.Step)
	}
	d.esc()
	if g.Phase != engine.PhaseTown || g.Town.Creation != nil {
		t.Errorf("ESC at name should cancel creation")
	}
}

// Wiz 2 and 3 have no character creation; unknown names silently re-prompt.
func TestNoCreationInLaterScenarios(t *testing.T) {
	for _, num := range []int{2, 3} {
		g := newGame(t, num)
		d := newDriver(t, g)
		toTrainingName(d, "NEWBIE").enter()
		if g.Town.InputMode != engine.InputTrainingName || g.Town.PendingCreateName != "" {
			t.Errorf("wiz%d: unknown name should re-prompt, mode=%v", num, g.Town.InputMode)
		}
	}
}

func TestRosterFullBlocksCreation(t *testing.T) {
	g := newGame(t, 1)
	for i := 0; i < 20; i++ {
		g.Town.Roster.Add(character(string(rune('A'+i)), engine.Human, engine.Fighter, engine.Good))
	}
	d := newDriver(t, g)
	toTrainingName(d, "ONEMORE").enter()
	if g.Town.Message != "THERE IS NO ROOM LEFT - TRY DELETING" {
		t.Errorf("message = %q", g.Town.Message)
	}
}

func TestExistingNameAsksForPassword(t *testing.T) {
	g := newGame(t, 1)
	c := character("BOROMIR", engine.Human, engine.Fighter, engine.Good)
	c.Password = "HORN"
	g.Town.Roster.Add(c)
	d := newDriver(t, g)

	toTrainingName(d, "boromir").enter()
	if g.Town.InputMode != engine.InputPassword || g.Town.EditChar != c {
		t.Fatalf("existing name should ask for password: mode=%v", g.Town.InputMode)
	}
	d.keys("HORN").enter()
	if g.Town.InputMode != engine.InputCharEdit {
		t.Errorf("correct password should open the character edit menu, mode=%v", g.Town.InputMode)
	}
}

func TestCreationIgnoresNonRuneKeysAtRace(t *testing.T) {
	g := newGame(t, 1)
	d := newDriver(t, g)
	toTrainingName(d, "K").enter().keys("Y").enter().enter()
	d.key(tcell.KeyUp).keys("Z")
	if g.Town.Creation.Step != engine.StepRace {
		t.Errorf("arrow keys and letters past E should not pick a race")
	}
}
