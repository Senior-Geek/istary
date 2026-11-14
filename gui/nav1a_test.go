package main

// NAV-1A: Istary always opens at the Castle. The game last chosen opens
// again, its expedition resumed if one is under way; with none, or one
// that is gone, no game is open and only the Castle, the Edge of Town and
// Settings can be visited, Select Game shows game selection, and nothing
// reaches a game.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/engine"
)

// exitStates lists a view's exits as dest, or dest! when closed.
func exitStates(v View) []string {
	var out []string
	for _, e := range v.Exits {
		if e.Open {
			out = append(out, e.Dest)
		} else {
			out = append(out, e.Dest+"!")
		}
	}
	return out
}

// lastGame is the game the window's settings remember, 0 for none.
func lastGame(t *testing.T, root string) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, settingsFile))
	if os.IsNotExist(err) {
		return 0
	}
	var st Settings
	if err != nil || json.Unmarshal(raw, &st) != nil {
		t.Fatalf("the window's settings: %v %s", err, raw)
	}
	return st.LastGame
}

// noGameCastle fails unless v is the Castle with no game open.
func noGameCastle(t *testing.T, v View) {
	t.Helper()
	if v.Fatal != "" || v.Screen != "roster" || v.Location != "castle" || v.Game != 0 || v.Return != nil || v.Games != nil {
		t.Fatalf("not the Castle with no game: screen %q at %q game %d return %+v", v.Screen, v.Location, v.Game, v.Return)
	}
	if got := exitStates(v); !slices.Equal(got, []string{"tavern!", "inn!", "trading!", "temple!", "edge"}) {
		t.Fatalf("the Castle's ways out with no game: %v", got)
	}
	for _, e := range v.Exits {
		if !e.Open && e.Reason != errChooseGame.Error() {
			t.Fatalf("%s closed for %q", e.Dest, e.Reason)
		}
	}
	if len(v.Roster) != 0 || len(v.Party) != 0 || len(v.PartyBar) != 6 || v.PartyBar[0].Name != "" {
		t.Fatalf("a party with no game: roster %d party %d bar %+v", len(v.Roster), len(v.Party), v.PartyBar)
	}
}

func TestFirstLaunchWithNoGamesOpensTheCastle(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	noGameCastle(t, s.State())
	if v := s.State(); v.Departures.Real != errChooseGame.Error() || v.SaveDir != root {
		t.Fatalf("departures %+v, folder %q", v.Departures, v.SaveDir)
	}
	if exists(filepath.Join(root, gamesDir)) || exists(filepath.Join(root, settingsFile)) {
		t.Fatal("opening wrote a game or the window's settings")
	}

	// Only the Edge of Town is open from the Castle.
	for _, dest := range []string{"tavern", "inn", "trading", "temple", "training", "maze", ""} {
		if v := s.Navigate(dest); v.Error == "" || v.Location != "castle" {
			t.Fatalf("went to %q with no game: %q at %q", dest, v.Error, v.Location)
		}
	}
	if v := s.Navigate("tavern"); v.Error != errChooseGame.Error() {
		t.Fatalf("the Tavern refused with %q", v.Error)
	}
	if v := s.ChangeGame(); v.Error != errLeaveAtEdge.Error() || v.Screen != "roster" {
		t.Fatalf("Select Game at the Castle: %q %q", v.Error, v.Screen)
	}
	if v := s.LeaveGame(); v.Error == "" {
		t.Fatal("left the game from the Castle")
	}

	// The Edge of Town: the Training Grounds and the maze closed, Settings
	// and the way back open.
	v := fine(t, s.Navigate("edge"))
	if got := exitStates(v); v.Location != "edge" || !slices.Equal(got, []string{"training!", "castle", "settings"}) {
		t.Fatalf("the Edge with no game: at %q, %v", v.Location, got)
	}
	if v.Return == nil || !v.Return.Open || v.Departures.Real != errChooseGame.Error() {
		t.Fatalf("the Edge: return %+v departures %+v", v.Return, v.Departures)
	}
	if v = s.Navigate("training"); v.Error != errChooseGame.Error() || v.Location != "edge" {
		t.Fatalf("the Training Grounds with no game: %q at %q", v.Error, v.Location)
	}
	if v = s.RealExpedition(); v.Expedition || v.Location != "edge" || s.ctrl != nil {
		t.Fatalf("entered the maze with no game: %q", v.Screen)
	}
	v = fine(t, s.Navigate("settings"))
	if v.Location != "settings" || !slices.Equal(exitStates(v), []string{"castle"}) || v.Return == nil {
		t.Fatalf("Settings with no game: at %q, %v", v.Location, exitStates(v))
	}
	// The window's settings are its own: kept with no game open.
	if v = s.SetAppearance("light", "slate", "subdued"); v.Settings.Panels != "light" || v.Settings.Note != "" {
		t.Fatalf("appearance with no game: %+v", v.Settings)
	}
	for _, v := range []View{s.ReloadRoster(), s.ResetRoster(), s.RestoreBackup("x.json")} {
		if v.Game != 0 || v.Location != "settings" || exists(filepath.Join(root, gamesDir)) {
			t.Fatalf("a roster tool acted with no game: %q", v.Error)
		}
	}
	// Escape (ReturnToCastle) from Settings and the Edge.
	noGameCastle(t, fine(t, s.ReturnToCastle()))
	fine(t, s.Navigate("edge"))
	noGameCastle(t, fine(t, s.ReturnToCastle()))
	noGameCastle(t, fine(t, s.ReturnToCastle())) // at the Castle, nothing
	if exists(filepath.Join(root, gamesDir)) {
		t.Fatal("a game was written")
	}
}

func TestSelectGameWithNoGameOpen(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	fine(t, s.Navigate("edge"))
	v := fine(t, s.ChangeGame())
	if v.Screen != "games" || v.Games == nil || len(v.Games.Slots) != gameSlots || v.Return == nil || !v.Return.Open {
		t.Fatalf("Select Game: screen %q games %v return %+v", v.Screen, v.Games != nil, v.Return)
	}
	if v = s.Navigate("castle"); v.Error != errChoosing.Error() || v.Screen != "games" {
		t.Fatalf("walked away from game selection: %q %q", v.Error, v.Screen)
	}
	if v = s.LeaveGame(); v.Screen != "games" || v.Notice != "" {
		t.Fatalf("Leave the Game from game selection: %q %q", v.Screen, v.Notice)
	}
	// Escape leaves it for the Castle, a replacement asked about forgotten.
	noGameCastle(t, fine(t, s.ReturnToCastle()))

	// Creating a game opens it at the Castle, and remembers it.
	fine(t, s.Navigate("edge"))
	fine(t, s.ChangeGame())
	v = fine(t, s.NewGame(2))
	if v.Game != 2 || v.Screen != "roster" || v.Location != "castle" || v.Games != nil {
		t.Fatalf("a new game: game %d screen %q at %q", v.Game, v.Screen, v.Location)
	}
	if got := exitStates(v); !slices.Equal(got, []string{"tavern", "inn!", "trading!", "temple!", "edge"}) {
		t.Fatalf("the Castle's ways out in a game: %v", got)
	}
	if lastGame(t, root) != 2 {
		t.Fatalf("remembered game %d", lastGame(t, root))
	}

	// Select Game closes it (it is saved) and shows game selection;
	// Escape from there is the Castle with no game.
	walkTo(t, s, "edge")
	v = fine(t, s.ChangeGame())
	if v.Screen != "games" || v.Game != 0 || s.ctrl != nil || v.Notice != "Game 2 is saved." || v.Return == nil {
		t.Fatalf("Select Game in a game: %q game %d %q", v.Screen, v.Game, v.Notice)
	}
	noGameCastle(t, fine(t, s.ReturnToCastle()))
	if lastGame(t, root) != 2 {
		t.Fatal("leaving game selection forgot the game")
	}

	// Choosing another game opens it at the Castle and remembers it.
	fine(t, s.Navigate("edge"))
	fine(t, s.ChangeGame())
	fine(t, s.NewGame(5))
	walkTo(t, s, "edge")
	fine(t, s.ChangeGame())
	if v = fine(t, s.ContinueGame(2)); v.Game != 2 || v.Location != "castle" || lastGame(t, root) != 2 {
		t.Fatalf("continued game 2: game %d at %q, remembered %d", v.Game, v.Location, lastGame(t, root))
	}
}

func TestTheLastGameOpensAgain(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(3))
	rosterWith(t, s, hero("ALDA", engine.Good, ""))
	join(t, s, 0, "")
	walkTo(t, s, "temple") // wherever the game was left
	save := readFile(t, filepath.Join(s.slotDir(3), saveFile))
	prefs := readFile(t, filepath.Join(root, settingsFile))

	r := relibrary(t, s, root)
	v := r.State()
	if v.Game != 3 || v.Screen != "roster" || v.Location != "castle" || partyNames(v) != "ALDA" || v.Notice != "" {
		t.Fatalf("a restart: game %d screen %q at %q party %q %q", v.Game, v.Screen, v.Location, partyNames(v), v.Notice)
	}
	if !bytes.Equal(readFile(t, filepath.Join(r.slotDir(3), saveFile)), save) || !bytes.Equal(readFile(t, filepath.Join(root, settingsFile)), prefs) {
		t.Fatal("opening the last game wrote its save or the window's settings")
	}

	// Leave the Game, then start again: the same game, at the Castle.
	quit := 0
	r.setQuit(func() { quit++ })
	walkTo(t, r, "edge")
	if v = fine(t, r.LeaveGame()); v.Notice != "Leaving Istary." {
		t.Fatalf("Leave the Game: %q", v.Notice)
	}
	s.shutdown()
	r.shutdown()
	r2 := newLibraryAt(t, root)
	defer r2.shutdown()
	if v = r2.State(); v.Game != 3 || v.Location != "castle" || partyNames(v) != "ALDA" {
		t.Fatalf("after Leave the Game: game %d at %q", v.Game, v.Location)
	}
}

func TestLeaveTheGameWithNoGameOpen(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	quit := make(chan struct{}, 1)
	s.setQuit(func() { quit <- struct{}{} })
	fine(t, s.Navigate("edge"))
	if v := fine(t, s.LeaveGame()); v.Notice != "Leaving Istary." {
		t.Fatalf("Leave the Game: %q", v.Notice)
	}
	<-quit
	if exists(filepath.Join(root, gamesDir)) || exists(filepath.Join(root, settingsFile)) {
		t.Fatal("leaving wrote something")
	}
	r := relibrary(t, s, root)
	defer r.shutdown()
	noGameCastle(t, r.State())
}

// A remembered game that is gone, cannot be opened, or is no slot at all:
// the Castle with no game, saying why, and nothing touched.
func TestAMissingOrInvalidLastGame(t *testing.T) {
	for _, c := range []struct {
		name, settings, says string
		damage               bool
	}{
		{"an empty slot", `{"version":1,"lastGame":4}`, "game 4, the last chosen, is no longer in its slot", false},
		{"no such slot", `{"version":1,"lastGame":9}`, "is not one of the six slots", false},
		{"a negative slot", `{"version":1,"lastGame":-1}`, "is not one of the six slots", false},
		{"a damaged game", `{"version":1,"lastGame":4}`, "game 4, the last chosen, cannot be opened", true},
		{"damaged settings", `{"version":1,"lastGame":`, "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "library")
			if err := os.MkdirAll(filepath.Join(root, gamesDir, "slot-4"), 0o755); err != nil {
				t.Fatal(err)
			}
			if !c.damage {
				os.Remove(filepath.Join(root, gamesDir, "slot-4"))
			} else if err := os.WriteFile(filepath.Join(root, gamesDir, "slot-4", saveFile), []byte("{not json"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, settingsFile), []byte(c.settings), 0o644); err != nil {
				t.Fatal(err)
			}
			s := newLibraryAt(t, root)
			defer s.shutdown()
			v := s.State()
			noGameCastle(t, v)
			if !strings.Contains(v.Notice, c.says) || (c.says == "" && v.Settings.Note == "") {
				t.Fatalf("says %q (settings %q)", v.Notice, v.Settings.Note)
			}
			if got := string(readFile(t, filepath.Join(root, settingsFile))); got != c.settings {
				t.Fatalf("the window's settings were rewritten: %s", got)
			}
			if c.damage && string(readFile(t, filepath.Join(root, gamesDir, "slot-4", saveFile))) != "{not json" {
				t.Fatal("the damaged game was touched")
			}
			// Choosing a game opens and remembers it.
			fine(t, s.Navigate("edge"))
			fine(t, s.ChangeGame())
			if v = fine(t, s.NewGame(1)); v.Game != 1 || v.Location != "castle" || lastGame(t, root) != 1 {
				t.Fatalf("a new game: %d at %q, remembered %d", v.Game, v.Location, lastGame(t, root))
			}
		})
	}
}
