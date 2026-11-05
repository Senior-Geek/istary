package app

import (
	"errors"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

func atTavern(t *testing.T, num int, roster ...*engine.Character) *Controller {
	t.Helper()
	c := newController(t, num)
	for _, ch := range roster {
		c.Game.Town.Roster.Add(ch)
	}
	if err := c.EnterTavern(); err != nil {
		t.Fatal(err)
	}
	return c
}

func refusal(err error) string {
	var r *Refusal
	if errors.As(err, &r) {
		return r.Message
	}
	return ""
}

func TestTavernAddAndRemoveOperations(t *testing.T) {
	a := member("ARWEN", engine.Good)
	a.Password = "EVENSTAR"
	b := member("BEORN", engine.Neutral)
	c := atTavern(t, 1, a, b)

	needs, err := c.AddMember("arwen")
	if err != nil || !needs {
		t.Fatalf("AddMember: needs=%v err=%v", needs, err)
	}
	if err := c.SubmitTavernPassword("wrong"); refusal(err) != "** THATS NOT IT **" {
		t.Fatalf("wrong password: %v", err)
	}
	c.AddMember("arwen")
	if err := c.SubmitTavernPassword("evenstar"); err != nil {
		t.Fatalf("right password: %v", err)
	}
	c.AddMember("beorn")
	c.SubmitTavernPassword("")
	if c.Game.Town.Party.Size() != 2 || c.Game.Town.Message != "BEORN JOINS THE PARTY!" {
		t.Fatalf("party=%d msg=%q", c.Game.Town.Party.Size(), c.Game.Town.Message)
	}

	if _, err := c.AddMember("arwen"); refusal(err) != "ARWEN IS ALREADY IN THE PARTY!" {
		t.Errorf("duplicate: %v", err)
	}
	if _, err := c.AddMember("nobody"); refusal(err) != "** WHO? **" {
		t.Errorf("unknown: %v", err)
	}
	if _, err := c.AddMember("  "); !errors.Is(err, ErrEmptyName) {
		t.Errorf("blank: %v", err)
	}

	// An invalid slot is rejected without opening the prompt.
	if err := c.RemoveMember(4); !errors.Is(err, ErrNoSuchPartyMember) || c.Game.Town.InputMode != engine.InputNone {
		t.Errorf("bad slot: %v mode=%v", err, c.Game.Town.InputMode)
	}
	if err := c.RemoveMember(0); err != nil || c.Game.Town.Party.Members[0] != b || a.InMaze {
		t.Errorf("remove slot 0: %v", err)
	}
	if err := c.InspectMember(0); err != nil || c.Game.Town.EditChar != b || c.Game.Town.InputMode != engine.InputInspect {
		t.Errorf("inspect: %v", err)
	}
}

func TestTavernPartyFull(t *testing.T) {
	var roster []*engine.Character
	for _, n := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		roster = append(roster, member(n, engine.Neutral))
	}
	c := atTavern(t, 1, roster...)
	for _, ch := range roster[:6] {
		c.AddMember(ch.Name)
		c.SubmitTavernPassword("")
	}
	if _, err := c.AddMember("G"); !errors.Is(err, ErrPartyFull) || c.Game.Town.Message != "** PARTY IS FULL **" {
		t.Errorf("seventh member: %v msg=%q", err, c.Game.Town.Message)
	}
}

func TestPartyAlignment(t *testing.T) {
	town := engine.NewTownState()
	if PartyAlignment(town) != engine.Neutral {
		t.Error("empty party should be neutral")
	}
	town.Party.Members = []*engine.Character{member("G", engine.Good), member("N", engine.Neutral)}
	if PartyAlignment(town) != engine.Good {
		t.Error("good + neutral should be good")
	}
	town.Party.Members = append(town.Party.Members, member("E", engine.Evil))
	if PartyAlignment(town) != engine.Evil {
		t.Error("last non-neutral member should win")
	}
}

func TestTavernPreconditions(t *testing.T) {
	c := newController(t, 1)
	if _, err := c.AddMember("X"); !errors.Is(err, ErrWrongState) {
		t.Errorf("AddMember outside the tavern: %v", err)
	}
	if err := c.SubmitTavernPassword("X"); !errors.Is(err, ErrWrongState) {
		t.Errorf("password without prompt: %v", err)
	}
	if err := c.LeaveTavern(); !errors.Is(err, ErrWrongState) {
		t.Errorf("leave when not there: %v", err)
	}
	c.EnterTavern()
	if err := c.EnterTavern(); !errors.Is(err, ErrWrongState) {
		t.Errorf("enter twice: %v", err)
	}
	if err := c.RemoveMember(0); !errors.Is(err, ErrPartyEmpty) {
		t.Errorf("remove from empty party: %v", err)
	}
	if err := c.LeaveTavern(); err != nil || c.Game.Town.Location != engine.Castle {
		t.Errorf("leave: %v", err)
	}
}

// Keys and operations must reach the same state for the same actions.
func TestTavernKeysMatchOperations(t *testing.T) {
	mk := func() []*engine.Character {
		g := member("GOOD", engine.Good)
		g.Password = "PW"
		return []*engine.Character{g, member("EVIL", engine.Evil), member("NEUTRAL", engine.Neutral)}
	}
	byKey := atTavern(t, 1, mk()...)
	byOp := atTavern(t, 1, mk()...)
	same := func(step string) {
		t.Helper()
		if a, b := snapshot(t, byKey.Game.Town), snapshot(t, byOp.Game.Town); a != b {
			t.Fatalf("after %s:\nkeys %s\n ops %s", step, a, b)
		}
	}

	keys(t, byKey, "agood")
	press(t, byKey, KeyEnter)
	keys(t, byKey, "pw")
	press(t, byKey, KeyEnter)
	byOp.AddMember("good")
	byOp.SubmitTavernPassword("pw")
	same("add GOOD")

	keys(t, byKey, "aevil")
	press(t, byKey, KeyEnter)
	byOp.AddMember("evil")
	same("add EVIL (bad alignment)")

	keys(t, byKey, "aneutral")
	press(t, byKey, KeyEnter)
	press(t, byKey, KeyEnter)
	byOp.AddMember("neutral")
	byOp.SubmitTavernPassword("")
	same("add NEUTRAL")

	keys(t, byKey, "r1")
	byOp.RemoveMember(0)
	same("remove slot 1")

	press(t, byKey, KeyEnter)
	byOp.LeaveTavern()
	same("leave")
}
