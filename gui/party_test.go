package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// rosterWith adds characters straight to the roster and saves, for party
// tests that need particular alignments or states.
func rosterWith(t *testing.T, s *Session, chars ...*engine.Character) {
	t.Helper()
	for _, ch := range chars {
		if ch.HP == 0 {
			ch.HP, ch.MaxHP = 8, 8
		}
		s.ctrl.Game.Town.Roster.Add(ch)
	}
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
}

func hero(name string, align engine.Alignment, password string) *engine.Character {
	ch := engine.NewCharacter(name, engine.Human, engine.Fighter, align)
	ch.Password = password
	return ch
}

// savedParty reads the party names from roster1.json.
func savedParty(t *testing.T, dir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Party []string `json:"party"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f.Party
}

func partyNames(v View) string {
	var names []string
	for _, p := range v.Party {
		names = append(names, p.Name)
	}
	return strings.Join(names, ",")
}

// join runs the two-step join and fails the test on any error.
func join(t *testing.T, s *Session, index int, password string) View {
	t.Helper()
	v := s.JoinParty(index)
	if v.Error != "" || v.PendingJoin == "" {
		t.Fatalf("join %d: error=%q pending=%q", index, v.Error, v.PendingJoin)
	}
	v = s.SubmitJoinPassword(password)
	if v.Error != "" {
		t.Fatalf("password for %d: %q", index, v.Error)
	}
	return v
}

func TestJoinAndLeaveParty(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("ARWEN", engine.Good, "STAR"), hero("BEORN", engine.Neutral, ""))

	v := s.JoinParty(0)
	if v.Error != "" || v.PendingJoin != "ARWEN" || v.Screen != "roster" {
		t.Fatalf("ask to join: error=%q pending=%q screen=%s", v.Error, v.PendingJoin, v.Screen)
	}
	v = s.SubmitJoinPassword("wrong")
	if v.Error != "** THATS NOT IT **" || v.PendingJoin != "" || len(v.Party) != 0 {
		t.Fatalf("wrong password: error=%q pending=%q party=%v", v.Error, v.PendingJoin, partyNames(v))
	}
	// The password is asked for again on the next attempt.
	v = join(t, s, 0, "star")
	if partyNames(v) != "ARWEN" || v.Notice != "ARWEN joined the party." {
		t.Fatalf("join: party=%q notice=%q", partyNames(v), v.Notice)
	}
	a, b := v.Roster[0], v.Roster[1]
	if !a.InParty || a.PartySlot != 1 || a.JoinRefusal != "ARWEN IS ALREADY IN THE PARTY!" || b.InParty || b.JoinRefusal != "" {
		t.Errorf("roster flags: a=%+v b=%+v", a, b)
	}
	if v.PartyAlignment != "GOOD" {
		t.Errorf("party alignment = %q", v.PartyAlignment)
	}
	join(t, s, 1, "")
	if got := strings.Join(savedParty(t, dir), ","); got != "ARWEN,BEORN" {
		t.Errorf("saved party = %q", got)
	}
	if v := NewSession(dir).State(); partyNames(v) != "ARWEN,BEORN" || !v.Roster[0].InParty {
		t.Errorf("party after restart = %q", partyNames(v))
	}

	v = s.LeaveParty(0)
	if v.Error != "" || partyNames(v) != "BEORN" || v.Notice != "ARWEN left the party (still on the roster)." {
		t.Fatalf("leave: error=%q party=%q notice=%q", v.Error, partyNames(v), v.Notice)
	}
	if len(v.Roster) != 2 || v.Roster[0].InParty || v.Roster[0].InMaze {
		t.Errorf("after leaving: %+v", v.Roster[0])
	}
	if v := NewSession(dir).State(); partyNames(v) != "BEORN" {
		t.Errorf("party after leave + restart = %q", partyNames(v))
	}
	if v := s.LeaveParty(5); v.Error != "No party member in that slot." {
		t.Errorf("leave empty slot: %q", v.Error)
	}
}

// Every hint shown on the roster equals what actually happens on joining.
func TestJoinHintsMatchJoinOutcome(t *testing.T) {
	s, _ := newTestSession(t)
	out := hero("WANDERER", engine.Neutral, "")
	out.InMaze = true // out in the dungeon, not in the party
	lost := hero("GONE", engine.Neutral, "")
	lost.Status = engine.Lost
	rosterWith(t, s, hero("PALADIN", engine.Good, ""), hero("ROGUE", engine.Evil, ""), out, lost,
		hero("MONK", engine.Neutral, ""))
	join(t, s, 0, "") // party is GOOD now

	v := s.State()
	want := map[string]string{
		"PALADIN":  "PALADIN IS ALREADY IN THE PARTY!",
		"ROGUE":    "** BAD ALIGNMENT **",
		"WANDERER": "** OUT **",
		"GONE":     "** WHO? **",
		"MONK":     "",
	}
	for _, row := range v.Roster {
		if row.JoinRefusal != want[row.Name] {
			t.Errorf("%s hint = %q, want %q", row.Name, row.JoinRefusal, want[row.Name])
		}
	}
	// Try each one: the outcome matches the hint, and nothing is left open.
	for _, row := range v.Roster {
		got := s.JoinParty(row.Index)
		outcome := got.Error
		if got.PendingJoin != "" {
			outcome = ""
			s.CancelJoin()
		}
		if outcome != row.JoinRefusal {
			t.Errorf("%s: joining gave %q, hint said %q", row.Name, outcome, row.JoinRefusal)
		}
	}
	if v := s.State(); v.PendingJoin != "" || partyNames(v) != "PALADIN" {
		t.Errorf("after trying everyone: pending=%q party=%q", v.PendingJoin, partyNames(v))
	}
}

func TestPartyFullAndRows(t *testing.T) {
	s, _ := newTestSession(t)
	for i := 0; i < 7; i++ {
		rosterWith(t, s, hero(fmt.Sprintf("P%d", i+1), engine.Neutral, ""))
	}
	for i := 0; i < 6; i++ {
		join(t, s, i, "")
	}
	v := s.State()
	if v.PartyMax != 6 || v.FrontRow != 3 || len(v.Party) != 6 {
		t.Fatalf("party max/front/size = %d/%d/%d", v.PartyMax, v.FrontRow, len(v.Party))
	}
	for _, p := range v.Party {
		if p.Front != (p.Slot <= 3) {
			t.Errorf("slot %d front = %v", p.Slot, p.Front)
		}
	}
	if v.Roster[6].JoinRefusal != "** PARTY IS FULL **" {
		t.Errorf("seventh hint = %q", v.Roster[6].JoinRefusal)
	}
	if v := s.JoinParty(6); v.Error != "** PARTY IS FULL **" || v.PendingJoin != "" {
		t.Errorf("seventh join: error=%q pending=%q", v.Error, v.PendingJoin)
	}
	// Leaving from the front row moves everyone up, as in the Tavern.
	v = s.LeaveParty(0)
	if partyNames(v) != "P2,P3,P4,P5,P6" || !v.Party[2].Front || v.Party[3].Front {
		t.Errorf("after removing slot 1: %q", partyNames(v))
	}
}

func TestPendingJoinIsCancelledByOtherActions(t *testing.T) {
	s, _ := newTestSession(t)
	rosterWith(t, s, hero("ONE", engine.Neutral, "PW"))
	if v := s.JoinParty(0); v.PendingJoin != "ONE" {
		t.Fatal("expected a pending join")
	}
	if v := s.CancelJoin(); v.PendingJoin != "" || v.Error != "" {
		t.Errorf("cancel: pending=%q error=%q", v.PendingJoin, v.Error)
	}
	s.JoinParty(0)
	// Starting creation goes to the Training Grounds and closes the prompt.
	v := mustOK(t, s.StartCreation("newbie"), "confirmCreate")
	if v.PendingJoin != "" {
		t.Error("pending join should be cancelled when moving to the Training Grounds")
	}
	if v := s.SubmitJoinPassword("PW"); v.Error != "That action is not available right now." {
		t.Errorf("password with no pending join: %q", v.Error)
	}
}

// Moving between Training Grounds and Tavern in any order works.
func TestLocationTransitions(t *testing.T) {
	s, dir := newTestSession(t)
	createMage(t, s, "alpha") // ends at the Training Grounds
	join(t, s, 0, "secret")   // Training Grounds -> castle -> Tavern
	createMage(t, s, "beta")  // Tavern -> castle -> Training Grounds
	join(t, s, 1, "secret")
	if v := s.DeleteCharacter("alpha", "secret"); v.Error != "" || partyNames(v) != "BETA" {
		t.Fatalf("delete from Tavern: error=%q party=%q", v.Error, partyNames(v))
	}
	if got := strings.Join(savedParty(t, dir), ","); got != "BETA" {
		t.Errorf("saved party after delete = %q", got)
	}
	// Party actions are refused while a character is being created.
	mustOK(t, s.StartCreation("gamma"), "confirmCreate")
	mustOK(t, s.AnswerCreate(true), "creation")
	if v := s.LeaveParty(0); v.Error != "That action is not available right now." {
		t.Errorf("leave during creation: %q", v.Error)
	}
	if v := s.JoinParty(0); v.Error != "That action is not available right now." {
		t.Errorf("join during creation: %q", v.Error)
	}
}

func TestPartyWithRosterManagement(t *testing.T) {
	s, dir := newTestSession(t)
	rosterWith(t, s, hero("FRODO", engine.Good, ""), hero("SAM", engine.Good, ""))
	join(t, s, 0, "")
	join(t, s, 1, "")

	// Rename keeps the slot; the save stores the new name.
	if v := s.RenameCharacter(1, "samwise"); v.Error != "" || partyNames(v) != "FRODO,SAMWISE" {
		t.Fatalf("rename: error=%q party=%q", v.Error, partyNames(v))
	}
	if got := strings.Join(savedParty(t, dir), ","); got != "FRODO,SAMWISE" {
		t.Errorf("saved party after rename = %q", got)
	}
	// Reset backs up the roster with its party and clears both.
	if v := s.ResetRoster(); v.Error != "" || len(v.Party) != 0 {
		t.Fatalf("reset: error=%q party=%q", v.Error, partyNames(v))
	}
	// Restore brings the party back.
	v := s.RestoreBackup(s.ListBackups()[0].File)
	if v.Error != "" || partyNames(v) != "FRODO,SAMWISE" || !v.Roster[0].InParty {
		t.Fatalf("restore: error=%q party=%q", v.Error, partyNames(v))
	}
}

func TestPartyLeavesTerminalSavesAlone(t *testing.T) {
	s, _ := newTestSession(t)
	rosterWith(t, s, hero("PIP", engine.Neutral, ""))
	join(t, s, 0, "")
	s.LeaveParty(0)
	terminal := filepath.Join(os.Getenv("HOME"), ".config", "wizardry")
	if _, err := os.Stat(terminal); !os.IsNotExist(err) {
		t.Errorf("GUI must not create or write %s (stat err %v)", terminal, err)
	}
}
