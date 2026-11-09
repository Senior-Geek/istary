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

// The Adventurer's Inn in the GUI (I-4): a stay, with its level check, is
// one action and one save; so is pooling; choosing a guest, a refusal and
// looking write nothing; a failed save keeps the stay and Try saving again
// writes it without doing anything again.

const (
	innFighter = 0 // 3/14 HP, 1000 GP, experience for level 3
	innDead    = 3 // the priest, dead
	innMage    = 4 // 2/14 HP, 25 GP, spells spent
	innOld     = 5 // made old in the fixture, to die of old age
)

// innSession is partySession's party, wounded and paid, at the Inn; the
// save holds what the game holds.
func innSession(t *testing.T) (*Session, string) {
	t.Helper()
	rand.Seed(11)
	s, dir := partySession(t, false)
	m := s.ctrl.Game.Town.Party.Members
	for i, ch := range m {
		ch.Age = (20 + 3*i) * 52
	}
	m[innFighter].HP, m[innFighter].Gold, m[innFighter].XP = 3, 1000, 1724
	m[innDead].Status, m[innDead].HP = engine.Dead, 0
	m[innMage].HP, m[innMage].Gold, m[innMage].MageSpells[0] = 2, 25, 0
	m[2].Gold = 140
	if err := s.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	walkTo(t, s, "inn")
	return s, dir
}

// innAct runs r and reports whether it wrote to the save folder.
func innAct(t *testing.T, s *Session, dir string, r InnRequest) (View, bool) {
	t.Helper()
	before := folder(t, dir)
	v := s.InnAction(r)
	return v, !maps.Equal(before, folder(t, dir))
}

func TestInnStaySavedOnce(t *testing.T) {
	s, dir := innSession(t)
	fighter := s.ctrl.Game.Town.Party.Members[innFighter]
	v := s.State()
	if v.Location != "inn" || v.Inn == nil || v.Inn.View.Step != "who" || v.Inn.View.Guests[innDead].Refusal != "DGUI IS NOT OK" {
		t.Fatalf("at the Inn: %+v", v.Inn)
	}
	if v, wrote := innAct(t, s, dir, InnRequest{Action: InnActEnter, Slot: innDead}); wrote || v.Error != "DGUI IS NOT OK" {
		t.Errorf("the dead priest: %q, wrote %v", v.Error, wrote)
	}
	if v, wrote := innAct(t, s, dir, InnRequest{Action: InnActStay}); wrote || v.Error != errNoGuest.Error() {
		t.Errorf("a stay with no guest: %q", v.Error)
	}
	v, wrote := innAct(t, s, dir, InnRequest{Action: InnActEnter, Slot: innFighter})
	if wrote || v.Error != "" || v.Inn.View.Guest != innFighter || len(v.Inn.View.Rooms) != 5 {
		t.Fatalf("the fighter checks in: %q %+v", v.Error, v.Inn.View)
	}
	// Quoting changes nothing and draws nothing.
	before := folder(t, dir)
	rand.Seed(5)
	q, err := s.InnQuote(int(engine.Economy), 2)
	next := rand.Int63()
	rand.Seed(5)
	if err != nil || q.Weeks != 2 || q.Cost != 100 || q.HP != 9 || next != rand.Int63() || !maps.Equal(before, folder(t, dir)) {
		t.Errorf("a quote: %+v %v", q, err)
	}
	if q, _ := s.InnQuote(int(engine.Economy), 9); q.Refusal != "AGUI needs only 4 weeks to heal fully." {
		t.Errorf("too many weeks: %+v", q)
	}
	if _, err := s.InnQuote(7, 0); err != errNoRoom {
		t.Errorf("no such room: %v", err)
	}
	// A refused stay writes nothing.
	if v, wrote := innAct(t, s, dir, InnRequest{Action: InnActStay, Room: int(engine.Economy), Weeks: 9}); wrote || v.Error == "" || v.Inn.Last != nil {
		t.Errorf("refused: %q, wrote %v", v.Error, wrote)
	}
	// Two weeks: one save, the stay and its level check together.
	v, wrote = innAct(t, s, dir, InnRequest{Action: InnActStay, Room: int(engine.Economy), Weeks: 2})
	st := v.Inn.Last.Stay
	if !wrote || v.Error != "" || st == nil || st.Weeks != 2 || st.Charged != 100 || st.HPAfter != 9 || st.LevelAfter != 3 ||
		!v.Inn.Last.Saved || v.Inn.View.Step != "levelUp" || !strings.Contains(v.Notice, "stayed 2 weeks") {
		t.Fatalf("two weeks: wrote %v, %q %+v", wrote, v.Error, v.Inn.Last)
	}
	mustMatchDisk(t, s, dir, "after a stay")
	// Until healed, from the level check's result.
	v, wrote = innAct(t, s, dir, InnRequest{Action: InnActStay, Room: int(engine.Economy)})
	if !wrote || v.Inn.Last.Stay.Weeks != 4 || v.Inn.Last.Stay.Charged != 200 || fighter.HP != fighter.MaxHP {
		t.Errorf("until healed: %+v", v.Inn.Last.Stay)
	}
	// Pool Gold to the guest: one save.
	total := 0
	for _, m := range s.ctrl.Game.Town.Party.Members {
		total += m.Gold
	}
	v, wrote = innAct(t, s, dir, InnRequest{Action: InnActPool})
	if !wrote || v.Inn.Last.Gold == nil || fighter.Gold != total || v.Inn.Last.Gold.Total != total {
		t.Errorf("pooled: %+v, fighter %d of %d", v.Inn.Last, fighter.Gold, total)
	}
	mustMatchDisk(t, s, dir, "after pooling")
	// The guest leaves, another checks in, and the Inn is left: no writes.
	for _, r := range []InnRequest{{Action: InnActLeaveGuest}, {Action: InnActEnter, Slot: innMage}, {Action: InnActEnter, Slot: innFighter}} {
		if v, wrote := innAct(t, s, dir, r); wrote || v.Error != "" {
			t.Errorf("%+v: %q, wrote %v", r, v.Error, wrote)
		}
	}
	before = folder(t, dir)
	if v := s.ReturnToCastle(); v.Location != "castle" || v.Inn != nil || !maps.Equal(before, folder(t, dir)) {
		t.Errorf("back to the castle: %s", v.Location)
	}
	if v := s.InnAction(InnRequest{Action: InnActStay}); !strings.Contains(v.Error, "Adventurer's Inn") {
		t.Errorf("away from the Inn: %q", v.Error)
	}
}

// A stay whose save fails stays in the window, its level check rolled
// once; Try saving again writes it as it is, and a restart finds it.
func TestInnFailedSaveRetried(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	s, dir := innSession(t)
	fighter := s.ctrl.Game.Town.Party.Members[innFighter]
	s.InnAction(InnRequest{Action: InnActEnter, Slot: innFighter})
	os.Chmod(dir, 0o555)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	v := s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Royal)})
	if v.Unsaved == "" || v.Inn.Last.Saved || !strings.Contains(v.Notice, "NOT saved") || fighter.Level != 3 {
		t.Fatalf("a failed save: %q %+v level %d", v.Unsaved, v.Inn.Last, fighter.Level)
	}
	after := *fighter
	snap, _ := json.Marshal(fighter)
	os.Chmod(dir, 0o755)
	rand.Seed(9)
	v = s.SaveNow()
	next := rand.Int63()
	rand.Seed(9)
	if next != rand.Int63() {
		t.Fatal("saving again drew a random number")
	}
	now, _ := json.Marshal(fighter)
	if v.Unsaved != "" || !v.Inn.Last.Saved || !v.Inn.Last.Durable || !bytes.Equal(now, snap) {
		t.Fatalf("saved again: %q %+v", v.Unsaved, v.Inn.Last)
	}
	mustMatchDisk(t, s, dir, "saved again")
	r := reopened(t, s, dir)
	got := r.ctrl.Game.Town.Party.Members[innFighter]
	if got.Level != 3 || got.HP != after.HP || got.MaxHP != after.MaxHP || got.Gold != after.Gold ||
		[6]int{got.Strength, got.IQ, got.Piety, got.Vitality, got.Agility, got.Luck} != [6]int{after.Strength, after.IQ, after.Piety, after.Vitality, after.Agility, after.Luck} {
		t.Fatalf("a restart finds %+v, want %+v", *got, after)
	}
}

// A guest who dies of old age keeps the stay's receipt, death included, and
// is offered no room again: not now, not after checking out, not on another
// visit. Refused stays write nothing.
func TestInnLostGuest(t *testing.T) {
	s, dir := innSession(t)
	old := s.ctrl.Game.Town.Party.Members[innFighter]
	old.Age, old.Vitality, old.XP = 200*52, 3, 1<<30
	s.ctrl.Save()
	s.InnAction(InnRequest{Action: InnActEnter, Slot: innFighter})
	var v View
	for i := 0; old.Status == engine.OK; i++ {
		if i > 20 {
			t.Fatal("no death of old age in 20 stays")
		}
		v = s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Stables)})
	}
	st := v.Inn.Last.Stay
	if !st.Died || st.Status != "LOST" || !v.Inn.Last.Saved || st.StatsAfter == st.StatsBefore || v.Inn.View.GuestRefusal == "" || len(v.Inn.View.Rooms) != 0 {
		t.Fatalf("died: %+v view %+v", st, v.Inn.View)
	}
	mustMatchDisk(t, s, dir, "after the death")
	for room := 0; room < 5; room++ {
		v, wrote := innAct(t, s, dir, InnRequest{Action: InnActStay, Room: room})
		if wrote || v.Error != "AGUI IS NOT OK: LOST and cannot stay at the Inn." || !v.Inn.Last.Stay.Died {
			t.Errorf("room %d after death: %q, wrote %v, receipt %+v", room, v.Error, wrote, v.Inn.Last)
		}
		if q, _ := s.InnQuote(room, 0); q.Refusal != v.Error {
			t.Errorf("quote after death: %+v", q)
		}
	}
	v, _ = innAct(t, s, dir, InnRequest{Action: InnActEnter, Slot: innMage})
	if v.Inn.View.Guests[innFighter].Refusal != "AGUI IS NOT OK" {
		t.Errorf("switching guests: %+v", v.Inn.View.Guests)
	}
	if v, wrote := innAct(t, s, dir, InnRequest{Action: InnActEnter, Slot: innFighter}); wrote || v.Error != "AGUI IS NOT OK" {
		t.Errorf("checking the LOST back in: %q", v.Error)
	}
	walkTo(t, s, "castle")
	if v := walkTo(t, s, "inn"); v.Inn.View.Guests[innFighter].Refusal != "AGUI IS NOT OK" || v.Inn.Last != nil {
		t.Errorf("another visit: %+v", v.Inn)
	}
}

func TestDivvyGoldAtTheTavern(t *testing.T) {
	s, dir := innSession(t)
	walkTo(t, s, "castle")
	before := folder(t, dir)
	if v := s.GoldAction(GoldRequest{Action: GoldActDivvy}); !strings.Contains(v.Error, "Tavern") || !maps.Equal(before, folder(t, dir)) {
		t.Errorf("divvy at the castle: %q", v.Error)
	}
	walkTo(t, s, "tavern")
	total := 0
	for _, m := range s.ctrl.Game.Town.Party.Members {
		total += m.Gold
	}
	v := s.GoldAction(GoldRequest{Action: GoldActDivvy})
	if v.Error != "" || v.Gold == nil || !v.Gold.Saved || v.Gold.Move.Total != total || maps.Equal(before, folder(t, dir)) {
		t.Fatalf("divvy: %q %+v", v.Error, v.Gold)
	}
	for i, m := range s.ctrl.Game.Town.Party.Members {
		want := total / 6
		if i < total%6 {
			want++
		}
		if m.Gold != want {
			t.Errorf("%s: %d GP, want %d", m.Name, m.Gold, want)
		}
	}
	mustMatchDisk(t, s, dir, "after sharing")
	// Sharing again changes nothing and writes nothing.
	before = folder(t, dir)
	if v := s.GoldAction(GoldRequest{Action: GoldActDivvy}); v.Error != "" || !maps.Equal(before, folder(t, dir)) {
		t.Errorf("sharing an even split: %q", v.Error)
	}
	if v := s.GoldAction(GoldRequest{Action: "steal"}); v.Error == "" {
		t.Errorf("an unknown action: %q", v.Error)
	}
}

type innCase struct {
	Case     string         `json:"case"`
	Location string         `json:"location"`
	Inn      *InnPanel      `json:"inn"`
	Guest    *CharacterView `json:"guest"`
	Quotes   []innQuoteCase `json:"quotes"`
	PartyBar []PartyBarSlot `json:"partyBar"`
	Error    string         `json:"error"`
	Notice   string         `json:"notice"`
	Unsaved  string         `json:"unsaved"`
	Warning  string         `json:"warning"`
}

type innQuoteCase struct {
	Room  int          `json:"room"`
	Weeks int          `json:"weeks"`
	Quote app.InnQuote `json:"quote"`
}

// TestInnViewFixture records real views at the Inn, one for each state the
// Inn screen draws, with the guest's roster entry and quotes for chosen
// weeks, for the page's tests (frontend/src/lib/innViews.test.js). Run
// with ISTARY_UPDATE_FIXTURES=1 to rewrite it.
func TestInnViewFixture(t *testing.T) {
	var cases []json.RawMessage // each case as it was when recorded
	s, dir := innSession(t)
	add := func(name string, v View) {
		c := innCase{Case: name, Location: v.Location, Inn: v.Inn, PartyBar: v.PartyBar,
			Error: v.Error, Notice: v.Notice, Unsaved: v.Unsaved, Warning: v.Warning}
		if v.Inn != nil && v.Inn.View.Guest >= 0 {
			for i := range v.Roster {
				if v.Roster[i].Name == v.Inn.View.Name {
					c.Guest = &v.Roster[i]
				}
			}
			for _, rq := range [][2]int{{0, 1}, {1, 1}, {1, 3}, {2, 2}, {2, 40}, {4, 1}} {
				q, _ := s.InnQuote(rq[0], rq[1])
				c.Quotes = append(c.Quotes, innQuoteCase{rq[0], rq[1], q})
			}
		}
		raw, _ := json.Marshal(c)
		cases = append(cases, raw)
	}
	rand.Seed(21)
	add("who", s.State())
	add("refused: not OK", s.InnAction(InnRequest{Action: InnActEnter, Slot: innDead}))
	add("mage: little gold", s.InnAction(InnRequest{Action: InnActEnter, Slot: innMage}))
	add("mage: stables", s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Stables)}))
	add("mage: a cot until out of gold", s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Cot)}))
	add("fighter checks in", s.InnAction(InnRequest{Action: InnActEnter, Slot: innFighter}))
	add("fighter: two weeks, a level", s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Economy), Weeks: 2}))
	add("fighter: refused weeks", s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Economy), Weeks: 9}))
	add("fighter: pooled", s.InnAction(InnRequest{Action: InnActPool}))
	add("fighter: until healed", s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Royal)}))
	add("fighter: stables, nothing to save", s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Stables)}))
	if os.Geteuid() != 0 {
		s.ctrl.Game.Town.Party.Members[innFighter].HP = 1
		os.Chmod(dir, 0o555)
		v := s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Cot), Weeks: 2})
		os.Chmod(dir, 0o755)
		tidy := func(msg string) string {
			msg = strings.ReplaceAll(msg, "/private"+dir, "<saves>")
			msg = strings.ReplaceAll(msg, dir, "<saves>")
			return regexp.MustCompile(`tmp-[0-9a-f]+`).ReplaceAllString(msg, "tmp-<random>")
		}
		v.Error, v.Unsaved = tidy(v.Error), tidy(v.Unsaved)
		add("a cot, NOT saved", v)
		add("saved again", s.SaveNow())
	}
	saveGame = func(c *app.Controller) error {
		if err := c.Save(); err != nil {
			return err
		}
		return &engine.NotDurableError{Err: errors.New("no sync")}
	}
	s.ctrl.Game.Town.Party.Members[innFighter].HP = 1
	add("not durable", s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Cot), Weeks: 1}))
	saveGame = (*app.Controller).Save
	add("guest left", s.InnAction(InnRequest{Action: InnActLeaveGuest}))
	// An old member dies of old age at a level check: the receipt says so,
	// and he is offered no room again.
	old := s.ctrl.Game.Town.Party.Members[innOld]
	old.Age, old.Vitality, old.XP, old.Status = 200*52, 3, 1<<30, engine.OK
	s.ctrl.Save()
	s.InnAction(InnRequest{Action: InnActEnter, Slot: innOld})
	for i := 0; old.Status == engine.OK; i++ {
		if i > 20 {
			t.Fatal("no death of old age in 20 stays")
		}
		v := s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Stables)})
		if old.Status != engine.OK {
			add("old age: LOST", v)
		}
	}
	add("old age: no room again", s.InnAction(InnRequest{Action: InnActStay, Room: int(engine.Stables)}))
	add("old age: refused at who", s.InnAction(InnRequest{Action: InnActLeaveGuest}))
	s.shutdown()

	got, _ := json.MarshalIndent(cases, "", " ")
	got = append(got, '\n')
	path := filepath.Join("frontend", "src", "lib", "testdata", "inn-views.json")
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
		t.Fatal("inn-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
