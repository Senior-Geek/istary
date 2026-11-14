package main

// Game slots (UX-3A, games.go): each slot one independent adventure, the
// earlier single save imported without being touched, a game replaced only
// once the player confirms, and nothing of one game reaching another.

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// newLibraryAt opens a library window on root, at game selection.
func newLibraryAt(t *testing.T, root string) *Session {
	t.Helper()
	s := NewLibrary(root)
	if v := s.State(); v.Fatal != "" || v.Screen != "games" || v.Game != 0 {
		t.Fatalf("library: fatal %q screen %q game %d", v.Fatal, v.Screen, v.Game)
	}
	return s
}

func newLibrary(t *testing.T) (*Session, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "library")
	return newLibraryAt(t, root), root
}

// relibrary is Istary started again on root while s is open, as reopened
// is for one save folder.
func relibrary(t *testing.T, s *Session, root string) *Session {
	t.Helper()
	s.lock.release()
	r := NewLibrary(root)
	r.lock.release()
	lock, err := lockSaveFolder(root)
	if err != nil {
		t.Fatalf("take the folder back: %v", err)
	}
	s.lock = lock
	return r
}

// fine fails the test on an error in v.
func fine(t *testing.T, v View) View {
	t.Helper()
	if v.Error != "" || v.Fatal != "" {
		t.Fatalf("error %q fatal %q", v.Error, v.Fatal)
	}
	return v
}

// changeGame goes to the Edge of Town and back to game selection.
func changeGame(t *testing.T, s *Session) View {
	t.Helper()
	walkTo(t, s, "edge")
	v := fine(t, s.ChangeGame())
	if v.Screen != "games" || v.Game != 0 || s.ctrl != nil {
		t.Fatalf("still in a game: %q %d", v.Screen, v.Game)
	}
	return v
}

func rosterNames(v View) []string {
	var out []string
	for _, c := range v.Roster {
		out = append(out, c.Name)
	}
	return out
}

func slotStates(v View) []string {
	var out []string
	for _, sl := range v.Games.Slots {
		out = append(out, sl.State)
	}
	return out
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// (1, 11) Two games in two slots, each with its own roster and party; the
// slots list them, and switching between them carries nothing across. Each
// game's actions are saved in its own slot, never in the earlier save's
// place or another slot.
func TestTwoGamesAreIndependent(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	v := s.State()
	if !slices.Equal(slotStates(v), []string{"empty", "empty", "empty", "empty", "empty", "empty"}) || v.Games.Legacy != nil {
		t.Fatalf("a new library: %v %+v", slotStates(v), v.Games.Legacy)
	}

	v = fine(t, s.NewGame(1))
	if v.Game != 1 || v.Location != "castle" || len(v.Roster) != 0 || len(v.Party) != 0 || s.ctrl.Game.Exploration != nil {
		t.Fatalf("new game 1: game %d at %q roster %v party %v", v.Game, v.Location, rosterNames(v), v.Party)
	}
	rosterWith(t, s, hero("ALDA", engine.Good, ""), hero("BRAN", engine.Good, ""))
	join(t, s, 0, "")
	if got := savedParty(t, s.slotDir(1)); !slices.Equal(got, []string{"ALDA"}) {
		t.Fatalf("slot 1's save: party %v", got)
	}
	v = changeGame(t, s)
	if sl := v.Games.Slots[0]; sl.State != "game" || sl.Characters != 2 || !slices.Equal(sl.Party, []string{"ALDA"}) || sl.Saved == "" || sl.Created == "" {
		t.Fatalf("slot 1 listed as %+v", sl)
	}

	v = fine(t, s.NewGame(2))
	if v.Game != 2 || len(v.Roster) != 0 || len(v.Party) != 0 || s.ctrl.Game.Exploration != nil || len(s.ctrl.Game.StrandedSites()) != 0 {
		t.Fatalf("new game 2 inherited: roster %v party %v", rosterNames(v), v.Party)
	}
	slot1 := readFile(t, filepath.Join(s.slotDir(1), saveFile))
	rosterWith(t, s, hero("CORA", engine.Evil, ""))
	join(t, s, 0, "")
	if !bytes.Equal(readFile(t, filepath.Join(s.slotDir(1), saveFile)), slot1) {
		t.Fatal("game 2's actions changed slot 1")
	}
	changeGame(t, s)

	v = fine(t, s.ContinueGame(1))
	if v.Game != 1 || !slices.Equal(rosterNames(v), []string{"ALDA", "BRAN"}) || partyNames(v) != "ALDA" {
		t.Fatalf("game 1 continued: %v %q", rosterNames(v), partyNames(v))
	}
	changeGame(t, s)
	v = fine(t, s.ContinueGame(2))
	if !slices.Equal(rosterNames(v), []string{"CORA"}) || partyNames(v) != "CORA" {
		t.Fatalf("game 2 continued: %v %q", rosterNames(v), partyNames(v))
	}
	if exists(filepath.Join(root, saveFile)) {
		t.Fatal("a game was saved in the earlier save's place")
	}
}

// (2) A game's characters come back as they were: equipment, gold, level,
// experience, status and poison, after changing games and after a restart.
func TestAGameKeepsEquipmentGoldAndStatus(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(3))
	ch := hero("DORN", engine.Neutral, "SECRET")
	ch.Gold, ch.Level, ch.XP, ch.Status, ch.PoisonAmt = 4321, 3, 1234, engine.Afraid, 2
	ch.Items[0] = engine.Possession{ItemIndex: 1, Equipped: true, Identified: true}
	ch.Items[1] = engine.Possession{ItemIndex: 2, Identified: true}
	ch.ItemCount = 2
	rosterWith(t, s, ch, hero("EVA", engine.Neutral, ""))
	before := s.State().Roster
	saved := *s.ctrl.Game.Town.Roster.Characters[0]

	changeGame(t, s)
	if v := fine(t, s.ContinueGame(3)); !reflect.DeepEqual(v.Roster, before) {
		t.Fatalf("after changing games:\n%+v\nwas\n%+v", v.Roster, before)
	}
	r := relibrary(t, s, root)
	defer r.shutdown()
	v := fine(t, r.ContinueGame(3))
	if !reflect.DeepEqual(v.Roster, before) {
		t.Fatalf("after a restart:\n%+v\nwas\n%+v", v.Roster, before)
	}
	got := r.ctrl.Game.Town.Roster.Characters[0]
	if got.Gold != saved.Gold || got.Items != saved.Items || got.ItemCount != 2 || got.Status != engine.Afraid ||
		got.PoisonAmt != 2 || got.XP != 1234 || got.Password != "SECRET" || got.AC != saved.AC {
		t.Fatalf("DORN came back as %+v", got)
	}
}

// (3, 4, 6) Out on an expedition the game cannot be changed. Restarted,
// Istary opens on game selection, the slot says where its party is, and
// Continue puts the party back on its square, facing the same way, with
// its map, under EXPEDITION RESUMED.
func TestAnExpeditionContinuesWhereItWasSaved(t *testing.T) {
	rand.Seed(7)
	s, root := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(1))
	fillParty(t, s, false, engine.Neutral)
	fine(t, s.RealExpedition())
	s.DungeonAction(app.ActLeaveCamp, 0)
	s.DungeonAction(app.ActNo, 0)
	g := s.ctrl.Game
	for i := 0; i < 4 && g.WallAhead() != data.WallOpen; i++ {
		s.DungeonAction(app.ActRight, 0)
	}
	g.FightMap = [20][20]bool{}
	if v := s.DungeonAction(app.ActForward, 0); v.Screen != "dungeon" || v.Dungeon.Prompt != "" {
		t.Fatalf("stepping: %q %+v", v.Screen, v.Dungeon)
	}
	v := fine(t, s.DungeonAction(app.ActLeft, 0))
	where := [4]int{g.MazeLevel, g.PlayerX, g.PlayerY, int(g.Facing)}
	if where == [4]int{0, 0, 0, int(engine.North)} {
		t.Fatal("the party never left the entrance")
	}
	explored, position := g.Exploration.Clone(), *v.Position
	if v := s.ChangeGame(); v.Error == "" || v.Screen != "dungeon" || s.ctrl == nil {
		t.Fatalf("changed games in the maze: %q %q", v.Error, v.Screen)
	}

	r := relibrary(t, s, root)
	defer r.shutdown()
	rv := r.State()
	if rv.Screen != "games" || r.ctrl != nil {
		t.Fatalf("a restart opened %q", rv.Screen)
	}
	if sl := rv.Games.Slots[0]; len(sl.Expedition) != 6 || sl.Depth != 1 || len(sl.Party) != 0 {
		t.Fatalf("slot 1 listed as %+v", sl)
	}
	rv = fine(t, r.ContinueGame(1))
	rg := r.ctrl.Game
	if !rv.Real || rv.Screen != "dungeon" || rv.Title != resumedTitle || rg.Town.Party.Size() != 6 {
		t.Fatalf("continued: real %v screen %q title %q", rv.Real, rv.Screen, rv.Title)
	}
	if got := [4]int{rg.MazeLevel, rg.PlayerX, rg.PlayerY, int(rg.Facing)}; got != where || *rv.Position != position {
		t.Fatalf("continued at %v %+v, saved at %v %+v", got, *rv.Position, where, position)
	}
	if !reflect.DeepEqual(rg.Exploration, explored) {
		t.Fatal("the map changed")
	}
}

// (6) A game saved during an encounter continues under the existing rule
// (M6b): the party is back in the maze where it fought, the encounter
// over and said to be, not silently dropped or moved to the castle.
func TestAnInterruptedEncounterContinuesUnderTheExistingRule(t *testing.T) {
	rand.Seed(33)
	s, root := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(2))
	fillParty(t, s, false, engine.Neutral)
	realEncounter(t, s)
	g := s.ctrl.Game
	here, _ := g.Place()
	if err := errorsJoin(s.checkpoint(), s.save()); err != nil {
		t.Fatal(err)
	}
	if g.Expedition == nil || g.Expedition.Interrupted != app.InterruptedCombat {
		t.Fatal("no interrupted encounter saved")
	}
	r := relibrary(t, s, root)
	defer r.shutdown()
	rv := fine(t, r.ContinueGame(2))
	there, _ := r.ctrl.Game.Place()
	if !rv.Real || rv.Screen != "dungeon" || r.ctrl.Game.Phase != engine.PhaseMaze || there != here || rv.Title != resumedTitle ||
		!strings.Contains(rv.Notice, "Istary closed during an encounter: it is over") {
		t.Fatalf("continued: real %v screen %q at %v (was %v): %q", rv.Real, rv.Screen, there, here, rv.Notice)
	}
}

// (7) Clearing the roster (Settings, Reset roster) empties the roster and
// the party and backs them up in the game's own slot; the game's map stays,
// in memory, in the save and after continuing again.
func TestClearingTheRosterKeepsTheMap(t *testing.T) {
	rand.Seed(5)
	s, root := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(1))
	fillParty(t, s, false, engine.Neutral)
	fine(t, s.RealExpedition())
	s.DungeonAction(app.ActLeaveCamp, 0)
	if v := fine(t, s.DungeonAction(app.ActYes, 0)); v.Real || v.Location != "castle" {
		t.Fatalf("not home: %v %q", v.Real, v.Location)
	}
	explored := s.ctrl.Game.Exploration.Clone()
	if explored == nil || len(explored.LevelsSeen()) == 0 {
		t.Fatal("nothing explored")
	}
	walkTo(t, s, "settings")
	v := fine(t, s.ResetRoster())
	if len(v.Roster) != 0 || len(v.Party) != 0 || !reflect.DeepEqual(s.ctrl.Game.Exploration, explored) {
		t.Fatalf("after reset: roster %v party %v", rosterNames(v), v.Party)
	}
	st := onDisk(t, s.slotDir(1))
	if len(st.Roster) != 0 || len(st.PartyNames) != 0 || len(st.Exploration) == 0 {
		t.Fatalf("saved: roster %d party %v exploration %d bytes", len(st.Roster), st.PartyNames, len(st.Exploration))
	}
	if b, _ := os.ReadDir(filepath.Join(s.slotDir(1), "backups")); len(b) != 1 || exists(filepath.Join(root, "backups")) {
		t.Fatalf("backup not in the slot: %v", b)
	}
	changeGame(t, s)
	fine(t, s.ContinueGame(1))
	if !reflect.DeepEqual(s.ctrl.Game.Exploration, explored) {
		t.Fatal("the map was lost on continuing")
	}
}

// (8) A game is replaced only after Replace and its confirmation, for that
// slot: New Game on it, a confirmation unasked, cancelled or for another
// slot change nothing. The replaced game is moved aside, not deleted.
func TestAGameIsReplacedOnlyOnceConfirmed(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(1))
	rosterWith(t, s, hero("ELDER", engine.Good, ""))
	changeGame(t, s)
	save := filepath.Join(s.slotDir(1), saveFile)
	was := readFile(t, save)
	unchanged := func(what string, v View) {
		t.Helper()
		if v.Game != 0 || s.ctrl != nil || !bytes.Equal(readFile(t, save), was) || exists(filepath.Join(root, replacedDir)) {
			t.Fatalf("%s changed game 1 (game %d open)", what, v.Game)
		}
	}

	v := s.NewGame(1)
	if v.Error != errSlotTaken.Error() {
		t.Fatalf("new game on game 1: %q", v.Error)
	}
	unchanged("New Game", v)
	if v = s.ConfirmReplaceGame(1); v.Error != errNotReplace.Error() {
		t.Fatalf("unasked confirmation: %q", v.Error)
	}
	unchanged("an unasked confirmation", v)
	if v = fine(t, s.ReplaceGame(1)); v.Games.Replacing != 1 {
		t.Fatalf("replacing %d", v.Games.Replacing)
	}
	unchanged("Replace", v)
	if v = s.CancelReplaceGame(); v.Games.Replacing != 0 {
		t.Fatalf("cancelled, still replacing %d", v.Games.Replacing)
	}
	unchanged("Cancel", v)
	s.ReplaceGame(1)
	if v = s.ConfirmReplaceGame(2); v.Error != errNotReplace.Error() || v.Games.Replacing != 0 {
		t.Fatalf("confirmed for slot 2: %q %d", v.Error, v.Games.Replacing)
	}
	unchanged("a confirmation for another slot", v)
	if v = s.ReplaceGame(3); v.Error != errSlotEmpty.Error() {
		t.Fatalf("replace an empty slot: %q", v.Error)
	}

	s.ReplaceGame(1)
	v = fine(t, s.ConfirmReplaceGame(1))
	if v.Game != 1 || len(v.Roster) != 0 || !strings.Contains(v.Notice, "kept in") {
		t.Fatalf("replaced: game %d roster %v %q", v.Game, rosterNames(v), v.Notice)
	}
	aside, _ := os.ReadDir(filepath.Join(root, replacedDir))
	if len(aside) != 1 || !bytes.Equal(readFile(t, filepath.Join(root, replacedDir, aside[0].Name(), saveFile)), was) {
		t.Fatalf("the replaced game was not kept: %v", aside)
	}
}

// legacyLibrary makes a library whose folder holds the save of an earlier
// Istary from dir, with its backups, as the folder of an earlier version
// held it.
func legacyLibrary(t *testing.T, dir string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "library")
	if err := os.MkdirAll(filepath.Join(root, "backups"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, filepath.Join(dir, saveFile))
	if err := os.WriteFile(filepath.Join(root, saveFile), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backups", "roster1-20261101-120000.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Date(2026, 11, 1, 12, 0, 0, 0, time.Local)
	os.Chtimes(filepath.Join(root, saveFile), old, old)
	return root
}

// (5, 9, 10) The earlier single save, here a rescue under way with others
// stranded, is offered for import with what it holds. Imported into a
// slot, it resumes as it was saved: the stranded where they lie, the
// rescuers in the maze. The original and its date are untouched, its
// backups copied. Restarted, Istary does not offer it again and nothing is
// imported twice.
func TestTheEarlierSaveIsImportedOnce(t *testing.T) {
	rand.Seed(5)
	src, dir, fallen, pit := strandedAndRescuers(t, 2)
	sites, stranded := sitesOf(src.ctrl.Game), strandedOnDiskAt(t, dir, pit)
	src.shutdown()
	root := legacyLibrary(t, dir)
	legacy := filepath.Join(root, saveFile)
	raw := readFile(t, legacy)
	fi, _ := os.Stat(legacy)

	s := newLibraryAt(t, root)
	defer s.shutdown()
	v := s.State()
	l := v.Games.Legacy
	if l == nil || l.Problem != "" || l.Characters != len(fallen)+2 || l.Stranded != len(fallen) || len(l.Expedition) != 2 || l.Depth != 1 || l.Saved != "2026-11-01 12:00" {
		t.Fatalf("earlier save offered as %+v", l)
	}
	v = fine(t, s.ImportEarlierGame(4))
	if v.Game != 4 || !v.Real || v.Title != resumedTitle || !strings.Contains(v.Notice, "imported into slot 4") {
		t.Fatalf("imported: game %d real %v %q %q", v.Game, v.Real, v.Title, v.Notice)
	}
	if got := sitesOf(s.ctrl.Game); !slices.Equal(got, sites) || len(sites) == 0 {
		t.Fatalf("stranded %v, were %v", got, sites)
	}
	if got := strandedOnDiskAt(t, s.slotDir(4), pit); !slices.Equal(got, stranded) {
		t.Fatalf("stranded in the slot's save %v, were %v", got, stranded)
	}
	if !bytes.Equal(readFile(t, legacy), raw) {
		t.Fatal("the earlier save was changed")
	}
	if now, _ := os.Stat(legacy); !now.ModTime().Equal(fi.ModTime()) {
		t.Fatal("the earlier save was touched")
	}
	if !exists(filepath.Join(s.slotDir(4), "backups", "roster1-20261101-120000.json")) {
		t.Fatal("the backups were not copied")
	}

	r := relibrary(t, s, root)
	defer r.shutdown()
	rv := r.State()
	if rv.Games.Legacy != nil || !slices.Equal(slotStates(rv), []string{"empty", "empty", "empty", "game", "empty", "empty"}) || !rv.Games.Slots[3].Imported {
		t.Fatalf("after a restart: legacy %+v slots %v", rv.Games.Legacy, slotStates(rv))
	}
	if rv = r.ImportEarlierGame(1); rv.Error != errNoLegacy.Error() || exists(r.slotDir(1)) {
		t.Fatalf("imported again: %q", rv.Error)
	}
	if rv = fine(t, r.ContinueGame(4)); !rv.Real || !slices.Equal(sitesOf(r.ctrl.Game), sites) {
		t.Fatalf("continued: real %v stranded %v", rv.Real, sitesOf(r.ctrl.Game))
	}
}

// An imported save is exactly the original, byte for byte, until the
// game's first action; and once the slot holding it is replaced, the
// original (kept as it was) is offered again, so it is never lost.
func TestAnImportIsTheOriginalUntilPlayed(t *testing.T) {
	src, dir := newTestSession(t)
	rosterWith(t, src, hero("FARO", engine.Good, ""), hero("GWEN", engine.Evil, ""))
	join(t, src, 0, "")
	src.shutdown()
	root := legacyLibrary(t, dir)
	raw := readFile(t, filepath.Join(root, saveFile))

	s := newLibraryAt(t, root)
	defer s.shutdown()
	v := fine(t, s.ImportEarlierGame(1))
	if v.Real || v.Title != "" || !slices.Equal(rosterNames(v), []string{"FARO", "GWEN"}) || partyNames(v) != "FARO" {
		t.Fatalf("imported: %v %q", rosterNames(v), partyNames(v))
	}
	if !bytes.Equal(readFile(t, filepath.Join(s.slotDir(1), saveFile)), raw) {
		t.Fatal("the import differs from the original")
	}
	v = changeGame(t, s)
	if v.Games.Legacy != nil {
		t.Fatal("offered again while slot 1 holds it")
	}
	s.ReplaceGame(1)
	fine(t, s.ConfirmReplaceGame(1))
	v = changeGame(t, s)
	if v.Games.Legacy == nil {
		t.Fatal("not offered once no slot holds it")
	}
	if v = fine(t, s.ImportEarlierGame(2)); !slices.Equal(rosterNames(v), []string{"FARO", "GWEN"}) {
		t.Fatalf("imported again: %v", rosterNames(v))
	}
}

// (11) Changing games loads Wizardry I afresh and clears what the window
// held for the last game: a special square used in one game's maze, its
// map and its messages are not another's.
func TestChangingGamesCarriesNothingAcross(t *testing.T) {
	rand.Seed(5)
	s, _ := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(1))
	fillParty(t, s, false, engine.Neutral)
	fine(t, s.RealExpedition())
	s.DungeonAction(app.ActLeaveCamp, 0)
	fine(t, s.DungeonAction(app.ActYes, 0))
	if s.ctrl.Game.Exploration == nil {
		t.Fatal("nothing explored")
	}
	s.ctrl.Game.Scenario.Mazes.Levels[0].Cells[3][4].Count++ // as a used special square
	used := s.scenario
	walkTo(t, s, "edge")
	s.notice = "game 1's message"
	changeGame(t, s)

	v := fine(t, s.NewGame(2))
	if s.scenario == used || len(engine.ChangedSquares(s.pristine, s.scenario)) != 0 {
		t.Fatal("game 2 has game 1's maze")
	}
	if s.ctrl.Game.Exploration != nil || v.Minimap != nil || v.Notice != "" || v.Real || s.practice != nil || s.receipt != nil {
		t.Fatalf("game 2 inherited: map %v notice %q", s.ctrl.Game.Exploration != nil, v.Notice)
	}
}

// (12) Damaged slots are listed as damaged, cannot be continued and are
// left exactly as they are; one may be replaced (and is kept aside). A
// save that parses but cannot be loaded opens nothing. A damaged earlier
// save is not imported and not touched. Slots out of range are refused.
func TestDamagedSavesAreLeftAsTheyAre(t *testing.T) {
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
	record := `{"version":1,"created":"2026-11-12T10:00:00Z","origin":"new"}`
	put("games/slot-1/roster1.json", "{not json")
	put("games/slot-2/game.json", record)
	put("games/slot-3/game.json", `{"version":99}`)
	put("games/slot-3/roster1.json", `{"roster":[],"party":[]}`)
	put("games/slot-4/game.json", "garbage")
	put("games/slot-4/roster1.json", `{"roster":[],"party":[]}`)
	put("games/slot-5/game.json", record)
	put("games/slot-5/roster1.json", `{"roster":[],"party":[],"expedition":"bad"}`)
	put(saveFile, "not a save")
	files := map[string][]byte{}
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files[p], _ = os.ReadFile(p)
		}
		return nil
	})

	s := newLibraryAt(t, root)
	defer s.shutdown()
	v := s.State()
	if want := []string{"damaged", "damaged", "damaged", "damaged", "game", "empty"}; !slices.Equal(slotStates(v), want) {
		t.Fatalf("slots %v, want %v", slotStates(v), want)
	}
	for _, sl := range v.Games.Slots[:4] {
		if sl.Problem == "" {
			t.Fatalf("slot %d damaged without a reason", sl.Slot)
		}
	}
	if v.Games.Legacy == nil || v.Games.Legacy.Problem == "" {
		t.Fatalf("damaged earlier save offered as %+v", v.Games.Legacy)
	}
	for slot := 1; slot <= 5; slot++ {
		if v = s.ContinueGame(slot); v.Error == "" || v.Game != 0 || s.ctrl != nil || v.Screen != "games" {
			t.Fatalf("continued damaged slot %d: %q", slot, v.Error)
		}
	}
	if v = s.ImportEarlierGame(6); v.Error == "" || exists(s.slotDir(6)) {
		t.Fatalf("imported a damaged save: %q", v.Error)
	}
	for _, slot := range []int{0, 7, -1} {
		if v = s.ContinueGame(slot); v.Error != errNoSuchSlot.Error() {
			t.Fatalf("slot %d: %q", slot, v.Error)
		}
	}
	for path, was := range files {
		if now, err := os.ReadFile(path); err != nil || !bytes.Equal(now, was) {
			t.Fatalf("%s was changed", path)
		}
	}

	s.ReplaceGame(1)
	if v = fine(t, s.ConfirmReplaceGame(1)); v.Game != 1 {
		t.Fatalf("replacing a damaged game: %d", v.Game)
	}
	aside, _ := os.ReadDir(filepath.Join(root, replacedDir))
	if len(aside) != 1 || string(readFile(t, filepath.Join(root, replacedDir, aside[0].Name(), saveFile))) != "{not json" {
		t.Fatal("the damaged game was not kept aside")
	}
}

// A slot folder with no game in it, as a new game interrupted before its
// first save leaves it, is damaged, not empty: replacing it keeps what it
// holds and begins a new game.
func TestAnEmptySlotFolderIsReplaced(t *testing.T) {
	root := filepath.Join(t.TempDir(), "library")
	left := filepath.Join(root, gamesDir, "slot-2", "backups", "roster1-x.json")
	if err := os.MkdirAll(filepath.Dir(left), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(left, []byte("{}"), 0o644)
	s := newLibraryAt(t, root)
	defer s.shutdown()
	if sl := s.State().Games.Slots[1]; sl.State != "damaged" || sl.Problem == "" {
		t.Fatalf("slot 2 listed as %+v", sl)
	}
	if v := s.NewGame(2); v.Error != errSlotTaken.Error() || !exists(left) {
		t.Fatalf("new game over it: %q", v.Error)
	}
	s.ReplaceGame(2)
	if v := fine(t, s.ConfirmReplaceGame(2)); v.Game != 2 {
		t.Fatalf("replaced: game %d", v.Game)
	}
	aside, _ := os.ReadDir(filepath.Join(root, replacedDir))
	if len(aside) != 1 || !exists(filepath.Join(root, replacedDir, aside[0].Name(), "backups", "roster1-x.json")) {
		t.Fatal("what the folder held was not kept")
	}
}

// Changing games is offered only at the Edge of Town with everything
// saved and nothing under way, and only in a library window; while a
// game is open, the slots cannot be acted on.
func TestChangingGamesIsRefusedMidGame(t *testing.T) {
	rand.Seed(2)
	s, _ := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(1))
	if v := s.ChangeGame(); v.Error != errLeaveAtEdge.Error() || v.Game != 1 {
		t.Fatalf("from the castle: %q", v.Error)
	}
	for _, f := range []func(int) View{s.NewGame, s.ContinueGame, s.ImportEarlierGame, s.ReplaceGame, s.ConfirmReplaceGame} {
		if v := f(2); v.Error == "" || v.Game != 1 || exists(s.slotDir(2)) {
			t.Fatalf("a slot was acted on with a game open: %q", v.Error)
		}
	}
	walkTo(t, s, "training")
	s.StartCreation("NEWCOMER")
	s.AnswerCreate(true)
	if v := s.ChangeGame(); v.Error == "" || v.Game != 1 {
		t.Fatal("changed games while creating a character")
	}
	s.SetPassword("", "")
	s.Back()
	s.CancelCreation()
	fillParty(t, s, false, engine.Neutral)
	fine(t, s.PracticeExpedition())
	if v := s.ChangeGame(); v.Error == "" || v.Game != 1 {
		t.Fatal("changed games during a practice")
	}
	s.AbandonPractice()
	s.unsaved = "disk full"
	if v := s.ChangeGame(); v.Error != errUnsavedLeave.Error() || v.Game != 1 {
		t.Fatalf("changed games unsaved: %q", v.Error)
	}
	s.unsaved = ""
	fine(t, s.ChangeGame())

	d, _ := newTestSession(t)
	defer d.shutdown()
	walkTo(t, d, "edge")
	if v := d.ChangeGame(); v.Error != errNotLibrary.Error() {
		t.Fatalf("a save folder opened directly: %q", v.Error)
	}
	if v := d.NewGame(1); v.Error != errNotLibrary.Error() {
		t.Fatalf("a save folder opened directly: %q", v.Error)
	}
}

// With no game open, every action of the window is refused or shows game
// selection: none fails, and none writes a game.
func TestNoGameOpenActsOnNothing(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	rv := reflect.ValueOf(s)
	for i := 0; i < rv.NumMethod(); i++ {
		m := rv.Type().Method(i)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s with no game open: %v", m.Name, r)
				}
			}()
			var args []reflect.Value
			for j := 1; j < m.Type.NumIn(); j++ {
				args = append(args, reflect.Zero(m.Type.In(j)))
			}
			out := rv.Method(i).Call(args)
			if v, ok := out[0].Interface().(View); ok && (v.Screen != "games" || v.Game != 0) {
				t.Errorf("%s with no game open shows %q, game %d", m.Name, v.Screen, v.Game)
			}
		}()
	}
	if exists(filepath.Join(root, gamesDir)) || exists(filepath.Join(root, saveFile)) {
		t.Fatal("a game was written")
	}
	fine(t, s.NewGame(1))
	walkTo(t, s, "edge")
	if v := fine(t, s.ChangeGame()); v.SaveDir != root {
		t.Fatalf("game selection names %s, not the save folder %s", v.SaveDir, root)
	}
}

// The window's settings belong to the window: kept in the library folder,
// the same in every game, and never in a slot.
func TestSettingsAreTheWindows(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(1))
	s.SetMinimapVisible(false)
	s.SetAppearance("light", "slate", "subdued")
	changeGame(t, s)
	if v := fine(t, s.NewGame(2)); v.Settings.MinimapVisible || v.Settings.Panels != "light" || v.Settings.Story != "slate" || v.Settings.Artwork != "subdued" {
		t.Fatalf("game 2 has its own settings: %+v", v.Settings)
	}
	changeGame(t, s)
	if v := fine(t, s.ContinueGame(1)); v.Settings.Panels != "light" || v.Settings.Story != "slate" || v.Settings.Artwork != "subdued" {
		t.Fatalf("back in game 1: %+v", v.Settings)
	}
	if !exists(filepath.Join(root, settingsFile)) || exists(filepath.Join(s.slotDir(1), settingsFile)) {
		t.Fatal("the settings are not the library's")
	}
}

// A second window on the same library is refused, as on one save folder.
func TestALibraryIsOneWindows(t *testing.T) {
	s, root := newLibrary(t)
	defer s.shutdown()
	other := NewLibrary(root)
	defer other.shutdown()
	if v := other.State(); !strings.Contains(v.Fatal, "Another Istary window") {
		t.Fatalf("second window: %q", v.Fatal)
	}
}

// A new game's first save is the engine's ordinary save of an empty game,
// in the same format as before slots.
func TestANewGameIsAnOrdinarySave(t *testing.T) {
	s, _ := newLibrary(t)
	defer s.shutdown()
	fine(t, s.NewGame(6))
	d, dir := newTestSession(t)
	defer d.shutdown()
	if err := d.ctrl.Save(); err != nil {
		t.Fatal(err)
	}
	if got, want := readFile(t, filepath.Join(s.slotDir(6), saveFile)), readFile(t, filepath.Join(dir, saveFile)); !bytes.Equal(got, want) {
		t.Fatalf("new game's save:\n%s\nan empty save before slots:\n%s", got, want)
	}
}
