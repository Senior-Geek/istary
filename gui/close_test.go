package main

import (
	"errors"
	"maps"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

// Closing the window (close.go): a saved game closes at once, an unsaved
// one only by the player's choice, a save tried while closing repeats
// nothing, and a second close never asks twice.

// asker answers the closing question with answers in turn, recording each
// message it was asked with.
type asker struct {
	answers []string
	asked   []string
}

func (a *asker) ask(message string) (string, error) {
	a.asked = append(a.asked, message)
	if len(a.answers) == 0 {
		return "", errors.New("asked once too often")
	}
	answer := a.answers[0]
	a.answers = a.answers[1:]
	return answer, nil
}

// unsavedPurchase is shopSession's mage after buying a staff while the
// save folder is read-only: in the window, not on disk. The folder stays
// read-only until the test makes it writable.
func unsavedPurchase(t *testing.T) (*Session, string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	s, dir := shopSession(t)
	s.ShopAction(ShopRequest{Action: ShopActEnter, Slot: mageSlot})
	os.Chmod(dir, 0o555)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	if v := s.ShopAction(ShopRequest{Action: ShopActBuy, Item: itemStaff}); v.Unsaved == "" {
		t.Fatal("the purchase was saved")
	}
	return s, dir
}

func TestCloseWhenSaved(t *testing.T) {
	s, dir := shopSession(t)
	a := &asker{}
	before := folder(t, dir)
	if s.beforeClose(a.ask) {
		t.Fatal("a saved game did not close")
	}
	if len(a.asked) != 0 || !maps.Equal(before, folder(t, dir)) {
		t.Fatalf("closing a saved game asked %d times or wrote", len(a.asked))
	}
	// After a failed save that a later save made good, it closes too.
	s2, dir2 := unsavedPurchase(t)
	os.Chmod(dir2, 0o755)
	s2.SaveNow()
	if s2.beforeClose(a.ask) || len(a.asked) != 0 {
		t.Fatal("a game saved again did not close at once")
	}
}

func TestCloseUnsavedKeepPlaying(t *testing.T) {
	s, dir := unsavedPurchase(t)
	mage := s.ctrl.Game.Town.Party.Members[mageSlot]
	before := folder(t, dir)
	a := &asker{answers: []string{closeKeepPlaying}}
	if !s.beforeClose(a.ask) {
		t.Fatal("an unsaved game closed when the player kept playing")
	}
	if len(a.asked) != 1 || !strings.Contains(a.asked[0], "could not save") {
		t.Fatalf("asked %q", a.asked)
	}
	if v := s.State(); v.Unsaved == "" || mage.Gold != 990 || mage.ItemCount != 1 || !maps.Equal(before, folder(t, dir)) {
		t.Fatalf("keeping playing changed the game: unsaved %q, %d GP, %d items", v.Unsaved, mage.Gold, mage.ItemCount)
	}
	// Anything the question does not offer (Escape answers Keep playing)
	// keeps the window, and so does a question that cannot be asked.
	if !s.beforeClose((&asker{answers: []string{""}}).ask) || !s.beforeClose((&asker{}).ask) {
		t.Fatal("an unanswered question closed the window")
	}
}

func TestCloseUnsavedDiscard(t *testing.T) {
	s, dir := unsavedPurchase(t)
	before := folder(t, dir)
	a := &asker{answers: []string{closeDiscard}}
	if s.beforeClose(a.ask) {
		t.Fatal("Quit without saving kept the window")
	}
	if len(a.asked) != 1 || !maps.Equal(before, folder(t, dir)) {
		t.Fatalf("asked %d times, or quitting without saving wrote", len(a.asked))
	}
	os.Chmod(dir, 0o755)
	r := reopened(t, s, dir)
	if m := r.ctrl.Game.Town.Party.Members[mageSlot]; m.Gold != 1000 || m.ItemCount != 0 {
		t.Fatalf("a restart finds %d GP and %d items, want the last save's 1000 and 0", m.Gold, m.ItemCount)
	}
}

func TestCloseUnsavedSaveAgain(t *testing.T) {
	s, dir := unsavedPurchase(t)
	mage := s.ctrl.Game.Town.Party.Members[mageSlot]
	// Still read-only: the save fails, the question comes back, and
	// keeping playing leaves the game as it was.
	before := folder(t, dir)
	a := &asker{answers: []string{closeSaveAgain, closeKeepPlaying}}
	if !s.beforeClose(a.ask) || len(a.asked) != 2 || !maps.Equal(before, folder(t, dir)) {
		t.Fatalf("a failed save while closing: asked %d times", len(a.asked))
	}
	if mage.Gold != 990 || mage.ItemCount != 1 {
		t.Fatalf("in the window: %d GP, %d items", mage.Gold, mage.ItemCount)
	}
	// Writable: the save succeeds and the window closes, the purchase
	// written once and nothing drawn.
	os.Chmod(dir, 0o755)
	a = &asker{answers: []string{closeSaveAgain}}
	rand.Seed(5)
	prevent := s.beforeClose(a.ask)
	next := rand.Int63()
	rand.Seed(5)
	if next != rand.Int63() {
		t.Fatal("saving while closing drew a random number")
	}
	if prevent || len(a.asked) != 1 || s.State().Unsaved != "" {
		t.Fatalf("saving again while closing: prevent %v, asked %d", prevent, len(a.asked))
	}
	if mage.Gold != 990 || mage.ItemCount != 1 {
		t.Fatalf("in the window: %d GP, %d items", mage.Gold, mage.ItemCount)
	}
	mustMatchDisk(t, s, dir, "saved while closing")
	r := reopened(t, s, dir)
	if m := r.ctrl.Game.Town.Party.Members[mageSlot]; m.Gold != 990 || m.ItemCount != 1 {
		t.Fatalf("a restart finds %d GP and %d items, want 990 and 1", m.Gold, m.ItemCount)
	}
}

func TestCloseTwiceAsksOnce(t *testing.T) {
	s, dir := unsavedPurchase(t)
	asking, answer := make(chan struct{}), make(chan string)
	var asked int
	slow := func(string) (string, error) {
		asked++
		asking <- struct{}{}
		return <-answer, nil
	}
	first := make(chan bool)
	go func() { first <- s.beforeClose(slow) }()
	<-asking
	// A second close while the question is open: refused, not asked.
	never := func(string) (string, error) { t.Error("asked a second time"); return closeDiscard, nil }
	for range 3 {
		if !s.beforeClose(never) {
			t.Fatal("a second close closed the window")
		}
	}
	before := folder(t, dir)
	answer <- closeKeepPlaying
	select {
	case prevent := <-first:
		if !prevent {
			t.Fatal("Keep playing closed the window")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the first close did not finish")
	}
	if asked != 1 || !maps.Equal(before, folder(t, dir)) {
		t.Fatalf("asked %d times, or wrote", asked)
	}
	// A later close asks again.
	a := &asker{answers: []string{closeKeepPlaying}}
	if !s.beforeClose(a.ask) || len(a.asked) != 1 {
		t.Fatalf("a later close: asked %d times", len(a.asked))
	}
}

func TestLeaveGameThenClose(t *testing.T) {
	// Leave the game quits only with everything saved, so the close it
	// starts never asks.
	s, _ := shopSession(t)
	quits := make(chan struct{}, 1)
	s.setQuit(func() { quits <- struct{}{} })
	walkTo(t, s, "edge")
	if v := s.LeaveGame(); v.Error != "" {
		t.Fatal(v.Error)
	}
	<-quits
	if s.beforeClose((&asker{}).ask) {
		t.Fatal("the close Leave the game starts was refused")
	}
}
