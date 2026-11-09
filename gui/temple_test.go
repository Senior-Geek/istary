package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// The Temple of Cant in the GUI (T-3): a treatment, with its donation and
// roll, is one action and one save; pooling is one; a refusal and looking
// write nothing; a failed save keeps the treatment and Try saving again
// writes it without charging or rolling again.

// templeSession is partySession's party (AGUI paying, DGUI made dead) and,
// outside it, patients of every condition, at the Temple.
func templeSession(t *testing.T) (*Session, string) {
	t.Helper()
	rand.Seed(13)
	s, dir := partySession(t, false)
	g := s.ctrl.Game
	m := g.Town.Party.Members
	m[0].Gold, m[1].Gold, m[2].Gold = 5000, 100, 40
	m[3].Status, m[3].HP, m[3].Gold = engine.Dead, 0, 3000
	for _, p := range []struct {
		name  string
		st    engine.Status
		level int
		out   bool
	}{
		{"PARA", engine.Paralyzed, 3, false}, {"STONE", engine.Stoned, 2, false},
		{"DEAD", engine.Dead, 2, false}, {"ASHY", engine.Ashed, 4, false},
		{"LOSTY", engine.Lost, 1, false}, {"BELOW", engine.Dead, 1, true},
	} {
		ch := engine.NewCharacter(p.name, engine.Human, engine.Fighter, engine.Neutral)
		ch.Status, ch.Level, ch.Vitality, ch.MaxHP, ch.HP, ch.Age = p.st, p.level, 10, 30, 12, 30*52
		if p.st >= engine.Dead {
			ch.HP = 0
		}
		if p.out {
			ch.InMaze, ch.MazeLevel = true, 3
		}
		g.Town.Roster.Add(ch)
	}
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	walkTo(t, s, "temple")
	return s, dir
}

// templeIndex is name's place on the roster.
func templeIndex(s *Session, name string) int {
	for i, ch := range s.ctrl.Game.Town.Roster.Characters {
		if ch.Name == name {
			return i
		}
	}
	return -1
}

// rollSeed is a seed whose first roll (rand.Intn(100)) satisfies want.
func rollSeed(want func(int) bool) int64 {
	for s := int64(1); ; s++ {
		rand.Seed(s)
		if want(rand.Intn(100)) {
			return s
		}
	}
}

func templeAct(t *testing.T, s *Session, dir string, r TempleRequest) (View, bool) {
	t.Helper()
	before := folder(t, dir)
	v := s.TempleAction(r)
	return v, !maps.Equal(before, folder(t, dir))
}

func TestTempleTreatSavedOnce(t *testing.T) {
	s, dir := templeSession(t)
	v := s.State()
	if v.Location != "temple" || v.Temple == nil || len(v.Temple.View.Patients) != 7 {
		t.Fatalf("at the Temple: %+v", v.Temple)
	}
	// Refusals write nothing and keep no receipt.
	for _, tc := range []struct {
		r    TempleRequest
		want string
	}{
		{TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, "LOSTY"), Payer: 0}, "LOSTY IS LOST"},
		{TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, "BELOW"), Payer: 0}, "BELOW is OUT in the maze"},
		{TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, "PARA"), Payer: 3}, "DGUI IS NOT OK and cannot tithe."},
		{TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, "PARA"), Payer: 2}, "CHEAP APOSTATES! OUT!"},
		{TempleRequest{Action: "bless"}, errNoTempleAction.Error()},
	} {
		if v, wrote := templeAct(t, s, dir, tc.r); wrote || !strings.HasPrefix(v.Error, tc.want) || v.Temple.Last != nil {
			t.Errorf("%+v: %q, wrote %v", tc.r, v.Error, wrote)
		}
	}
	// A cure: one save, the receipt.
	rand.Seed(1)
	v, wrote := templeAct(t, s, dir, TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, "PARA"), Payer: 0})
	tr := v.Temple.Last.Treat
	if !wrote || v.Error != "" || tr == nil || !tr.Healed || tr.Donation != 300 || tr.PayerAfter != 4700 || !v.Temple.Last.Saved ||
		!strings.Contains(v.Notice, "PARA is well again") {
		t.Fatalf("a cure: wrote %v %q %+v", wrote, v.Error, v.Temple.Last)
	}
	mustMatchDisk(t, s, dir, "after a cure")
	// A refusal now keeps the receipt.
	if v, _ := templeAct(t, s, dir, TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, "LOSTY")}); v.Temple.Last == nil || v.Temple.Last.Treat.Patient != "PARA" {
		t.Errorf("a refusal dropped the receipt: %+v", v.Temple.Last)
	}
	// A failed raise: ashes, the donation kept.
	rand.Seed(rollSeed(func(r int) bool { return r > 80 }))
	v, _ = templeAct(t, s, dir, TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, "DEAD"), Payer: 0})
	if tr = v.Temple.Last.Treat; tr.Healed || tr.StatusAfter != "ASHED" || tr.PayerAfter != 4200 || !strings.Contains(v.Notice, "now ashes") {
		t.Errorf("a failed raise: %+v %q", tr, v.Notice)
	}
	// Pooling to a payer: one save.
	total := 0
	for _, m := range s.ctrl.Game.Town.Party.Members {
		total += m.Gold
	}
	v, wrote = templeAct(t, s, dir, TempleRequest{Action: TempleActPool, Payer: 1})
	if !wrote || v.Temple.Last.Gold == nil || s.ctrl.Game.Town.Party.Members[1].Gold != total {
		t.Errorf("pooled: %+v", v.Temple.Last)
	}
	mustMatchDisk(t, s, dir, "after pooling")
	// The Tavern and the party bar show the new conditions at once.
	if bar := s.State().PartyBar; !strings.Contains(bar[3].Status, "DEAD") {
		t.Errorf("party bar: %+v", bar[3])
	}
	before := folder(t, dir)
	if v := s.ReturnToCastle(); v.Location != "castle" || v.Temple != nil || !maps.Equal(before, folder(t, dir)) {
		t.Errorf("back to the castle: %s", v.Location)
	}
	if v := s.TempleAction(TempleRequest{Action: TempleActTreat}); !strings.Contains(v.Error, "Temple of Cant") {
		t.Errorf("away from the Temple: %q", v.Error)
	}
	if v := walkTo(t, s, "temple"); v.Temple.Last != nil {
		t.Errorf("another visit kept the receipt")
	}
}

// A treatment whose save fails stays in the window, rolled once and
// charged once; Try saving again writes it as it is; a restart finds it.
func TestTempleFailedSaveRetried(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	s, dir := templeSession(t)
	g := s.ctrl.Game
	os.Chmod(dir, 0o555)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	rand.Seed(rollSeed(func(r int) bool { return r <= 70 })) // ashes raised (40+30)
	v := s.TempleAction(TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, "ASHY"), Payer: 0})
	ashy := g.Town.Roster.Characters[templeIndex(s, "ASHY")]
	if v.Unsaved == "" || v.Temple.Last.Saved || !strings.Contains(v.Notice, "NOT saved") || ashy.Status != engine.OK || g.Town.Party.Members[0].Gold != 3000 {
		t.Fatalf("a failed save: %q %+v", v.Unsaved, v.Temple.Last)
	}
	snap, _ := json.Marshal(g.Town.Roster.Characters)
	os.Chmod(dir, 0o755)
	rand.Seed(9)
	v = s.SaveNow()
	next := rand.Int63()
	rand.Seed(9)
	now, _ := json.Marshal(g.Town.Roster.Characters)
	if next != rand.Int63() || v.Unsaved != "" || !v.Temple.Last.Saved || !bytes.Equal(snap, now) {
		t.Fatalf("saved again: %q %+v", v.Unsaved, v.Temple.Last)
	}
	mustMatchDisk(t, s, dir, "saved again")
	r := reopened(t, s, dir)
	got := r.ctrl.Game.Town.Roster.Characters[templeIndex(s, "ASHY")]
	if got.Status != engine.OK || got.HP != 30 || got.Age != ashy.Age || r.ctrl.Game.Town.Party.Members[0].Gold != 3000 {
		t.Errorf("a restart finds %v HP %d age %d, gold %d", got.Status, got.HP, got.Age, r.ctrl.Game.Town.Party.Members[0].Gold)
	}
}

type templeCase struct {
	Case     string          `json:"case"`
	Location string          `json:"location"`
	Temple   *TemplePanel    `json:"temple"`
	Roster   []CharacterView `json:"roster"`
	PartyBar []PartyBarSlot  `json:"partyBar"`
	Error    string          `json:"error"`
	Notice   string          `json:"notice"`
	Unsaved  string          `json:"unsaved"`
	Warning  string          `json:"warning"`
}

// TestTempleViewFixture records real views at the Temple, one for each
// state and outcome the Temple screen draws, for the page's tests
// (frontend/src/lib/templeViews.test.js). Run with ISTARY_UPDATE_FIXTURES=1
// to rewrite it.
func TestTempleViewFixture(t *testing.T) {
	var cases []json.RawMessage
	s, dir := templeSession(t)
	add := func(name string, v View) {
		raw, _ := json.Marshal(templeCase{name, v.Location, v.Temple, v.Roster, v.PartyBar, v.Error, v.Notice, v.Unsaved, v.Warning})
		cases = append(cases, raw)
	}
	treat := func(name string, payer int) View {
		return s.TempleAction(TempleRequest{Action: TempleActTreat, Patient: templeIndex(s, name), Payer: payer})
	}
	add("patients", s.State())
	add("refused: out in the maze", treat("BELOW", 0))
	add("refused: payer not OK", treat("PARA", 3))
	add("refused: short", treat("PARA", 2))
	rand.Seed(1)
	add("cured", treat("PARA", 0))
	rand.Seed(rollSeed(func(r int) bool { return r > 80 }))
	add("raise failed: ashes", treat("DEAD", 0))
	rand.Seed(rollSeed(func(r int) bool { return r <= 70 }))
	add("ashes raised", treat("DEAD", 0))
	rand.Seed(rollSeed(func(r int) bool { return r > 70 }))
	add("ashes lost", treat("ASHY", 0))
	add("pooled", s.TempleAction(TempleRequest{Action: TempleActPool, Payer: 1}))
	if os.Geteuid() != 0 {
		os.Chmod(dir, 0o555)
		rand.Seed(1)
		v := treat("STONE", 1)
		os.Chmod(dir, 0o755)
		tidy := func(msg string) string {
			msg = strings.ReplaceAll(msg, "/private"+dir, "<saves>")
			msg = strings.ReplaceAll(msg, dir, "<saves>")
			return regexp.MustCompile(`tmp-[0-9a-f]+`).ReplaceAllString(msg, "tmp-<random>")
		}
		v.Error, v.Unsaved = tidy(v.Error), tidy(v.Unsaved)
		add("cured, NOT saved", v)
		add("saved again", s.SaveNow())
	}
	saveGame = func(c *app.Controller) error {
		if err := c.Save(); err != nil {
			return err
		}
		return &engine.NotDurableError{Err: errors.New("no sync")}
	}
	rand.Seed(rollSeed(func(r int) bool { return r <= 80 }))
	add("raised, not durable", treat("DGUI", 1))
	saveGame = (*app.Controller).Save
	s.shutdown()

	got, _ := json.MarshalIndent(cases, "", " ")
	got = append(got, '\n')
	path := filepath.Join("frontend", "src", "lib", "testdata", "temple-views.json")
	if os.Getenv("ISTARY_UPDATE_FIXTURES") != "" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with ISTARY_UPDATE_FIXTURES=1)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("temple-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
