package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// beginCreation puts the controller at the password step for name, the
// state the Training Grounds "create it? Y" answer produces.
func beginCreation(c *Controller, name string) {
	cs := engine.NewCreationState()
	cs.Name = name
	cs.Step = engine.StepPassword
	c.Game.Town.Creation = cs
	c.Game.Phase = engine.PhaseCreation
}

func snapshot(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The terminal key map and the semantic operations must produce identical
// state at every step, including the random draws.
func TestCreationOperationsMatchKeyPath(t *testing.T) {
	for _, tc := range []struct {
		race  engine.Race
		align engine.Alignment
		class engine.Class
		stat  int // stat to pour points into first
	}{
		{engine.Human, engine.Good, engine.Fighter, 0},
		{engine.Elf, engine.Neutral, engine.Mage, 1},
		{engine.Dwarf, engine.Evil, engine.Priest, 2},
		{engine.Hobbit, engine.Neutral, engine.Thief, 4},
		{engine.Gnome, engine.Good, engine.Priest, 2},
	} {
		t.Run(fmt.Sprintf("%v-%v-%v", tc.race, tc.align, tc.class), func(t *testing.T) {
			byKey := newController(t, 1)
			byOp := newController(t, 1)
			beginCreation(byKey, "HERO")
			beginCreation(byOp, "HERO")
			same := func(step string) {
				t.Helper()
				if a, b := snapshot(t, byKey.Game.Town), snapshot(t, byOp.Game.Town); a != b {
					t.Fatalf("after %s:\nkeys %s\n ops %s", step, a, b)
				}
			}

			keys(t, byKey, "sesame")
			press(t, byKey, KeyEnter)
			keys(t, byKey, "sesame")
			press(t, byKey, KeyEnter)
			if err := byOp.SetCreationPassword("sesame", "sesame"); err != nil {
				t.Fatal(err)
			}
			same("password")

			keys(t, byKey, string(rune('a'+tc.race)))
			if err := byOp.ChooseRace(tc.race); err != nil {
				t.Fatal(err)
			}
			same("race")

			seed(int64(tc.class) + 500)
			keys(t, byKey, string(rune('a'+tc.align)))
			seed(int64(tc.class) + 500)
			if err := byOp.ChooseAlignment(tc.align); err != nil {
				t.Fatal(err)
			}
			same("alignment")

			// Spend points starting at tc.stat, moving on when a stat hits 18.
			for i := 0; i < tc.stat; i++ {
				press(t, byKey, KeyEnter)
				byOp.NextStat()
			}
			for n := 0; n < 6 && byOp.Creation().BonusPoints > 0; n++ {
				for byOp.Creation().BonusPoints > 0 {
					before := byOp.Creation().Stats
					keys(t, byKey, "+")
					byOp.AddStatPoint()
					if byOp.Creation().Stats == before {
						break
					}
				}
				press(t, byKey, KeyEnter)
				byOp.NextStat()
			}
			same("stats")

			press(t, byKey, KeyEscape)
			if err := byOp.FinishStats(); err != nil {
				t.Fatal(err)
			}
			same("finish stats")

			keys(t, byKey, string(rune('a'+tc.class)))
			if err := byOp.ChooseClass(tc.class); err != nil {
				t.Fatalf("%v: %v (avail %v)", tc.class, err, byOp.Creation().ClassAvailability())
			}
			same("class")

			seed(int64(tc.race) + 900)
			keys(t, byKey, "y")
			seed(int64(tc.race) + 900)
			ch, err := byOp.ConfirmCreation(true)
			if err != nil || ch == nil {
				t.Fatalf("ConfirmCreation: %v", err)
			}
			same("confirm")
			if byOp.Game.Phase != engine.PhaseTown || len(byOp.Game.Town.Roster.Characters) != 1 {
				t.Fatal("confirmed character should be in the roster, back in town")
			}
		})
	}
}

func TestCreationOperationPreconditions(t *testing.T) {
	c := newController(t, 1)
	if err := c.ChooseRace(engine.Elf); !errors.Is(err, ErrWrongState) {
		t.Errorf("ChooseRace with no creation: %v", err)
	}
	beginCreation(c, "X")
	if err := c.ChooseRace(engine.Elf); !errors.Is(err, ErrWrongState) {
		t.Errorf("ChooseRace at password step: %v", err)
	}
	if err := c.SetCreationPassword("ONE", "TWO"); !errors.Is(err, ErrPasswordMismatch) {
		t.Errorf("mismatched passwords: %v", err)
	}
	cs := c.Creation()
	if cs.Step != engine.StepPassword || cs.PasswordStep != 0 || cs.PasswordFirst != "" {
		t.Errorf("mismatch should restart password entry: %+v", cs)
	}
	if err := c.SetCreationPassword("", ""); err != nil || cs.Step != engine.StepRace {
		t.Errorf("empty password: err=%v step=%v", err, cs.Step)
	}
	if err := c.ChooseRace(engine.Race(9)); !errors.Is(err, ErrInvalidChoice) {
		t.Errorf("out-of-range race: %v", err)
	}
	c.ChooseRace(engine.Human)
	seed(1)
	c.ChooseAlignment(engine.Good)
	if err := c.FinishStats(); !errors.Is(err, ErrPointsRemaining) || cs.Step != engine.StepStats {
		t.Errorf("FinishStats with points left: err=%v step=%v", err, cs.Step)
	}
	if err := c.SelectStat(6); !errors.Is(err, ErrInvalidChoice) {
		t.Errorf("SelectStat(6): %v", err)
	}
	c.SelectStat(0)
	for cs.BonusPoints > 0 && cs.Stats[0] < 18 {
		c.AddStatPoint()
	}
	c.SelectStat(3)
	for cs.BonusPoints > 0 {
		c.AddStatPoint()
	}
	if err := c.FinishStats(); err != nil {
		t.Fatal(err)
	}
	if err := c.ChooseClass(engine.Ninja); !errors.Is(err, ErrClassUnavailable) || cs.Step != engine.StepClass {
		t.Errorf("unavailable class: err=%v step=%v", err, cs.Step)
	}
	if err := c.CreationBack(); err != nil || cs.Step != engine.StepStats {
		t.Errorf("back from class: err=%v step=%v", err, cs.Step)
	}
}

func TestCreationNameStepAndCancel(t *testing.T) {
	c := newController(t, 1)
	beginCreation(c, "X")
	c.SetCreationPassword("", "")
	if err := c.CreationBack(); err != nil || c.Creation().Step != engine.StepName {
		t.Fatalf("back from race: %v", err)
	}
	if err := c.SetCreationName("!!"); !errors.Is(err, ErrEmptyName) {
		t.Errorf("name of only invalid characters: %v", err)
	}
	if err := c.SetCreationName("frodo baggins of the shire"); err != nil || c.Creation().Name != "FRODO BAGGINS O" {
		t.Errorf("name: err=%v name=%q", err, c.Creation().Name)
	}
	c.CreationBack()
	if err := c.CancelCreation(); err != nil || c.Game.Phase != engine.PhaseTown || c.Game.Town.Creation != nil {
		t.Errorf("cancel: err=%v phase=%v", err, c.Game.Phase)
	}
}

func TestDeclineCreation(t *testing.T) {
	c := newController(t, 1)
	beginCreation(c, "X")
	cs := c.Creation()
	cs.Step = engine.StepConfirm
	ch, err := c.ConfirmCreation(false)
	if err != nil || ch != nil || c.Game.Town.Creation != nil || len(c.Game.Town.Roster.Characters) != 0 {
		t.Errorf("decline: ch=%v err=%v", ch, err)
	}
}

func TestNormalize(t *testing.T) {
	if got := NormalizeName("o'brien-2 !x"); got != "O'BRIEN-2 X" {
		t.Errorf("NormalizeName = %q", got)
	}
	if got := NormalizePassword("pa$$word-is-too-long"); got != "PA$$WORD-IS-TOO" {
		t.Errorf("NormalizePassword = %q", got)
	}
}

// Abandoning creation at any step leaves the town exactly as it was before
// creation began, as Cancel and Discard do, and draws no random number.
func TestAbandonCreationAtEveryStep(t *testing.T) {
	steps := []func(c *Controller) error{
		func(c *Controller) error { return c.SetCreationPassword("", "") },
		func(c *Controller) error { return c.ChooseRace(engine.Human) },
		func(c *Controller) error { seed(9); return c.ChooseAlignment(engine.Good) },
		func(c *Controller) error {
			for c.Creation().BonusPoints > 0 {
				c.SelectStat(0)
				if err := c.AddStatPoint(); err != nil {
					return err
				}
				if c.Creation().Stats[0] >= 18 {
					c.SelectStat(3)
				}
			}
			return c.FinishStats()
		},
		func(c *Controller) error { return c.ChooseClass(engine.Fighter) },
	}
	for k := 0; k <= len(steps); k++ {
		c := newController(t, 1)
		c.EnterTrainingGrounds()
		before := snapshot(t, c.Game.Town)
		beginCreation(c, "HERO")
		for _, step := range steps[:k] {
			if err := step(c); err != nil {
				t.Fatalf("after %d steps: %v", k, err)
			}
		}
		at := c.Creation().Step
		seed(21)
		if err := c.AbandonCreation(); err != nil {
			t.Fatalf("abandon at %v: %v", at, err)
		}
		next := rand.Int63()
		seed(21)
		if next != rand.Int63() {
			t.Errorf("abandon at %v drew a random number", at)
		}
		if c.Game.Phase != engine.PhaseTown || snapshot(t, c.Game.Town) != before {
			t.Errorf("abandon at %v: phase %v, town\n%s\nwas\n%s", at, c.Game.Phase, snapshot(t, c.Game.Town), before)
		}
	}

	c := newController(t, 1)
	if err := c.AbandonCreation(); !errors.Is(err, ErrWrongState) {
		t.Errorf("not creating: %v", err)
	}
	beginCreation(c, "HERO")
	c.Creation().Reroll = true
	if err := c.AbandonCreation(); !errors.Is(err, ErrWrongState) || c.Game.Phase != engine.PhaseCreation {
		t.Errorf("a reroll: %v", err)
	}
}
