package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// gamesView is what game selection reads from a View.
type gamesView struct {
	Case    string     `json:"case"`
	Screen  string     `json:"screen"`
	Error   string     `json:"error"`
	Games   *GamesView `json:"games"`
	SaveDir string     `json:"saveDir"`
}

// TestGamesViewFixture records game selection's part of real views, with
// every kind of slot and earlier save, for the page's tests
// (frontend/src/lib/games.test.js). Dates and folders are fixed, so the
// recording does not depend on when or where it is made. Run with
// ISTARY_UPDATE_FIXTURES=1 to rewrite it.
func TestGamesViewFixture(t *testing.T) {
	var views []gamesView
	record := func(name string, v View) {
		if v.Games != nil {
			for i := range v.Games.Slots {
				sl := &v.Games.Slots[i]
				if sl.Saved != "" {
					sl.Saved = "2026-11-12 10:00"
				}
				if sl.Created != "" {
					sl.Created = "2026-11-10 09:00"
				}
			}
			if v.Games.Legacy != nil {
				v.Games.Legacy.Saved = "2026-11-01 12:00"
			}
		}
		views = append(views, gamesView{name, v.Screen, v.Error, v.Games, "/saves"})
	}
	root := filepath.Join(t.TempDir(), "library")
	put := func(rel, text string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := newLibraryAt(t, root)
	record("no games", s.State())
	s.shutdown()

	put(saveFile, `{"roster":[{"Name":"ALDA"},{"Name":"BRAN"},{"Name":"CORA"}],"party":["ALDA","BRAN"]}`)
	s = newLibraryAt(t, root)
	record("an earlier save", s.State())
	fine(t, s.ImportEarlierGame(2))
	walkTo(t, s, "edge")
	fine(t, s.ChangeGame())
	s.shutdown()

	newRecord := `{"version":1,"created":"2026-11-10T09:00:00Z","origin":"new"}`
	put("games/slot-1/game.json", newRecord)
	put("games/slot-1/roster1.json", `{"roster":[{"Name":"DORN"},{"Name":"EVA"},{"Name":"FARO","stranded":{"level":2,"row":3,"col":4}}],"party":[],`+
		`"expedition":{"version":1,"party":["DORN","EVA"],"where":{"phase":"maze","level":1,"x":3,"y":4,"facing":0}}}`)
	put("games/slot-3/game.json", newRecord)
	put("games/slot-3/roster1.json", `{"roster":[],"party":[]}`)
	put("games/slot-5/roster1.json", "{not json")
	put(saveFile, "not a save")
	s = newLibraryAt(t, root)
	defer s.shutdown()
	record("games of every kind", s.State())
	record("replacing game 3", s.ReplaceGame(3))
	s.CancelReplaceGame()
	record("continuing a damaged game", s.ContinueGame(5))

	got, _ := json.MarshalIndent(views, "", " ")
	got = append(got, '\n')
	path := filepath.Join("frontend", "src", "lib", "testdata", "games-views.json")
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
		t.Fatal("games-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
