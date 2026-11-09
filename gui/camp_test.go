package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// Camp through the bridge (M6b-3B): CampAction runs the controller's Camp
// operations, which are the terminal's keys; every action is saved with
// its checkpoint on a real expedition, a restart puts Camp back on the
// screen it was on, and Disband and MALOR cannot be reached.

// campSession returns a session whose party (fighter, samurai, priest,
// bishop, mage, lord) has spells, gold, poison and mixed items (some
// cursed, equipped or unidentified), chosen by rng, out in Camp at the
// entrance on a real or practice expedition.
func campSession(t *testing.T, rng *rand.Rand, real bool) (*Session, string) {
	t.Helper()
	s, dir := newTestSession(t)
	items := s.scenario.Items
	var chars []*engine.Character
	for i, cls := range []engine.Class{engine.Fighter, engine.Samurai, engine.Priest, engine.Bishop, engine.Mage, engine.Lord} {
		ch := engine.NewCharacter(fmt.Sprintf("CAMPER %d", i+1), engine.Human, cls, engine.Neutral)
		ch.Level, ch.MaxHP, ch.HP = 13, 80, 20+rng.Intn(60)
		ch.Gold, ch.PoisonAmt = rng.Intn(500), rng.Intn(3)
		for l := 0; l < 7; l++ {
			ch.MageSpells[l], ch.MaxMageSpells[l] = rng.Intn(3), 3
			ch.PriestSpells[l], ch.MaxPriestSpells[l] = rng.Intn(3), 3
		}
		// Known spells as Load leaves them (finding #32), so a restart
		// changes nothing.
		ch.SpellKnown = [50]bool{}
		ch.MigrateSpellKnown()
		for n := 0; n < 3+rng.Intn(5); n++ {
			ch.AddItem(1+rng.Intn(len(items)-1), rng.Intn(2) == 0)
			if rng.Intn(5) == 0 {
				ch.Items[ch.ItemCount-1].Cursed = true
			}
			if rng.Intn(3) == 0 {
				ch.EquipItem(ch.ItemCount - 1)
			}
		}
		chars = append(chars, ch)
	}
	rosterWith(t, s, chars...)
	walkTo(t, s, "tavern")
	for i := range chars {
		if v := s.JoinParty(i); v.PendingJoin != "" {
			s.SubmitJoinPassword("")
		}
	}
	walkTo(t, s, "edge") // where the party sets out
	var v View
	if real {
		v = s.RealExpedition()
	} else {
		v = s.PracticeExpedition()
	}
	if v.Error != "" || v.Camp == nil || v.Camp.View.Screen != "menu" || len(v.Camp.Party) != 6 {
		t.Fatalf("not in Camp: %q %+v", v.Error, v.Camp)
	}
	return s, dir
}

// pickCamp chooses one of the panel's actions with arguments a player could
// give, reading only the panel. It leaves Camp only when leave is set.
func pickCamp(p *CampPanel, rng *rand.Rand, leave bool) CampRequest {
	v := p.View
	open := slices.DeleteFunc(slices.Clone(v.Actions), func(a string) bool { return a == app.ActLeaveCamp && !leave })
	r := CampRequest{Action: open[rng.Intn(len(open))], Slot: rng.Intn(len(p.Party))}
	var member CharacterView
	if v.Member >= 0 {
		member = p.Party[v.Member]
	}
	items := len(member.Items)
	switch r.Action {
	case app.ActReorderPick:
		for slices.Contains(v.Placed, r.Slot) {
			r.Slot = rng.Intn(len(p.Party))
		}
	case app.ActTradeWith:
		if len(p.Party) == 1 {
			r.Action = app.ActEndTrade
		}
		for r.Slot == v.Member && len(p.Party) > 1 {
			r.Slot = rng.Intn(len(p.Party))
		}
	case app.ActEquipChoose:
		if len(v.Choices) == 0 {
			r.Action = app.ActEquipSkip
		} else {
			r.N = 1 + rng.Intn(len(v.Choices))
		}
	case app.ActDropItem, app.ActTradeItem, app.ActUse:
		if items == 0 {
			r.Action = map[string]string{app.ActDropItem: app.ActEndDrop, app.ActTradeItem: app.ActEndTrade, app.ActUse: app.ActLeaveInspect}[r.Action]
		} else {
			r.N = 1 + rng.Intn(items)
		}
	case app.ActTradeGold:
		r.Gold = []int{0, member.Gold, member.Gold / 2, member.Gold + 1 + rng.Intn(50), rng.Intn(member.Gold + 1)}[rng.Intn(5)]
	case app.ActSpellBook:
		r.Priest = rng.Intn(2) == 0
	case app.ActCast:
		var castable []string
		for _, sp := range v.Spells {
			if sp.Castable {
				castable = append(castable, sp.Name)
			}
		}
		if len(castable) == 0 {
			r.Action = app.ActSpellBooks
		} else {
			r.Spell = castable[rng.Intn(len(castable))]
		}
	}
	return r
}

// terminalKeys are the terminal's keys for a request (as camp_ops.go
// presses them, written out independently).
func terminalKeys(screen string, r CampRequest) []app.Key {
	enter, esc := app.Key{Kind: app.KeyEnter}, app.Key{Kind: app.KeyEscape}
	digit := func(n int) app.Key { return app.Rune(rune('0' + n)) }
	switch r.Action {
	case app.ActCampInspect, app.ActReorderPick, app.ActTradeWith, app.ActCastOn:
		return []app.Key{digit(r.Slot + 1)}
	case app.ActEquipChoose, app.ActDropItem, app.ActTradeItem:
		return []app.Key{digit(r.N)}
	case app.ActUse:
		return []app.Key{app.Rune('U'), digit(r.N)}
	case app.ActReorder, app.ActSpellBooks:
		return []app.Key{app.Rune('R')}
	case app.ActEquipParty, app.ActEquip:
		return []app.Key{app.Rune('E')}
	case app.ActLeaveCamp, app.ActLeaveInspect, app.ActCloseBook, app.ActLeaveDumapic:
		return []app.Key{app.Rune('L')}
	case app.ActEquipSkip:
		return []app.Key{enter}
	case app.ActDrop:
		return []app.Key{app.Rune('D')}
	case app.ActEndDrop:
		return []app.Key{app.Rune('0')}
	case app.ActTrade:
		return []app.Key{app.Rune('T')}
	case app.ActTradeGold:
		var keys []app.Key
		if r.Gold > 0 {
			for _, d := range strconv.Itoa(r.Gold) {
				keys = append(keys, app.Rune(d))
			}
		}
		return append(keys, enter)
	case app.ActEndTrade:
		if screen == "tradeGold" {
			return []app.Key{esc}
		}
		return []app.Key{enter}
	case app.ActSpellBook:
		if r.Priest {
			return []app.Key{app.Rune('P')}
		}
		return []app.Key{app.Rune('M')}
	case app.ActCast:
		keys := []app.Key{app.Rune('S')}
		for _, c := range strings.ToLower(r.Spell) {
			keys = append(keys, app.Rune(c))
		}
		return append(keys, enter)
	case app.ActCancelCast:
		return []app.Key{esc}
	case app.ActIdentify:
		return []app.Key{app.Rune('I')}
	}
	panic("no keys for " + r.Action)
}

// campState is everything Camp can change, and the next random number.
func campState(g *engine.GameState) string {
	t := g.Town
	name := func(c *engine.Character) string {
		if c == nil {
			return "-"
		}
		return c.Name
	}
	var placed []string
	for _, m := range t.ReorderResult {
		placed = append(placed, name(m))
	}
	pending := "-"
	if t.PendingSpell != nil {
		pending = t.PendingSpell.Name
	}
	party, _ := json.Marshal(t.Party.Members)
	return fmt.Sprintf("%v %v %q %q %q | %s %s | %d %v %v %d | %v %s | light %d protect %d | L%d %d,%d %v | %s | next %d",
		g.Phase, t.InputMode, t.Message, t.Message2, t.InputBuf, name(t.EditChar), name(t.TradeTarget),
		t.EquipCategory, t.EquipChoices, t.EquipPartyMode, t.EquipPartyIdx, placed, pending,
		g.LightLevel, g.ProtectLevel, g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing, party, rand.Int63())
}

// goods are the party's gold and item indexes, sorted.
func goods(g *engine.GameState) (int, []int) {
	gold, items := 0, []int{}
	for _, m := range g.Town.Party.Members {
		gold += m.Gold
		for i := 0; i < m.ItemCount; i++ {
			items = append(items, m.Items[i].ItemIndex)
		}
	}
	sort.Ints(items)
	return gold, items
}

// slotsLeft is every member's spell slots left.
func slotsLeft(g *engine.GameState) map[string][14]int {
	out := map[string][14]int{}
	for _, m := range g.Town.Party.Members {
		var s [14]int
		copy(s[:7], m.MageSpells[:])
		copy(s[7:], m.PriestSpells[:])
		out[m.Name] = s
	}
	return out
}

// campInvariants fails if Camp created gold or items, lost them other than
// by dropping one, or gave back a spell slot.
func campInvariants(t *testing.T, after *engine.GameState, r CampRequest, slots map[string][14]int, gold int, items []int) {
	t.Helper()
	g, it := goods(after)
	if g != gold {
		t.Fatalf("%s: party gold %d, was %d", r.Action, g, gold)
	}
	lost := len(items) - len(it)
	if lost < 0 || lost > 1 || lost == 1 && r.Action != app.ActDropItem {
		t.Fatalf("%s: items %v, were %v", r.Action, it, items)
	}
	for name, now := range slotsLeft(after) {
		for i, n := range now {
			if n > slots[name][i] {
				t.Fatalf("%s: %s regained a spell slot: %v, was %v", r.Action, name, now, slots[name])
			}
		}
	}
}

// Each Camp action through the bridge is the terminal's keys: random
// actions on a real expedition, and their terminal keys through HandleKey
// on a copy of the same saved game, give the same game, the same random
// numbers and the same Camp panel. Every action is saved, gold and items
// are neither created nor lost (but by D)ROP), and no spell slot comes
// back.
func TestCampBridgeIsTheTerminalKeys(t *testing.T) {
	seen := map[string]bool{}
	for seed := int64(1); seed <= 10; seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir := campSession(t, rng, true)
		twin := reopened(t, s, dir).ctrl // the same saved game, played by keys
		rand.Seed(0)
		start := campState(s.ctrl.Game)
		rand.Seed(0)
		if start != campState(twin.Game) {
			t.Fatalf("seed %d: the copy differs at the start\n%s\n%s", seed, campState(s.ctrl.Game), campState(twin.Game))
		}
		for step := 0; step < 120; step++ {
			p := s.State().Camp
			if p == nil {
				seen["left camp"] = true
				break
			}
			screen := p.View.Screen
			r := pickCamp(p, rng, step > 100)
			gold, items := goods(s.ctrl.Game)
			slots := slotsLeft(s.ctrl.Game)

			rand.Seed(seed*1000 + int64(step))
			v := s.CampAction(r)
			got := campState(s.ctrl.Game)
			rand.Seed(seed*1000 + int64(step))
			for _, k := range terminalKeys(screen, r) {
				twin.HandleKey(k)
			}
			want := campState(twin.Game)
			switch {
			case v.Error != "" || v.Unsaved != "":
				t.Fatalf("seed %d step %d: %+v: %q %q", seed, step, r, v.Error, v.Unsaved)
			case got != want:
				t.Fatalf("seed %d step %d: %s %+v\n bridge %s\n keys   %s", seed, step, screen, r, got, want)
			}
			if twin.Game.Phase == engine.PhaseCamp && !reflect.DeepEqual(v.Camp, campPanel(twin)) {
				t.Fatalf("seed %d step %d: the panels differ", seed, step)
			}
			mustMatchDisk(t, s, dir, r.Action)
			campInvariants(t, s.ctrl.Game, r, slots, gold, items)
			seen[screen+" "+r.Action] = true
			if v.Camp != nil {
				seen["to "+v.Camp.View.Screen] = true
			}
		}
	}
	for _, k := range []string{"menu inspect", "menu reorder", "reorder reorder-pick", "menu equip-party", "equip equip-choose",
		"equip equip-skip", "inspect equip", "inspect drop", "drop drop-item", "drop end-drop", "inspect trade",
		"trade trade-with", "tradeGold trade-gold", "tradeItem trade-item", "tradeItem end-trade", "trade end-trade",
		"tradeGold end-trade", "inspect spell-books", "spellBooks spell-book", "spellList close-book", "spellBooks close-book",
		"inspect cast", "spellTarget cast-on", "spellTarget cancel-cast", "inspect identify", "inspect use",
		"dumapic leave-dumapic", "inspect leave", "menu leave-camp", "to dumapic", "left camp"} {
		if !seen[k] {
			t.Errorf("not covered: %s", k)
		}
	}
}

// copyDir copies the save folder (not its lock) to a new folder.
func copyDir(t *testing.T, dir string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "saves")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "roster1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "roster1.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return out
}

// Restarting Istary at any point in Camp, on any of its screens (equipping,
// trading, choosing a spell's target, reading DUMAPIC's result), brings
// back exactly that screen and its progress, and play goes on as if
// Istary had never closed: the same actions, played first without
// restarts and then from a copy of the same save with them, give the same
// panels and the same party. (One session at a time: the engine's save
// folder is the process's.)
func TestCampRestartFollowsUninterruptedPlay(t *testing.T) {
	type moment struct {
		camp     *CampPanel
		dungeon  *app.DungeonView
		standing []standing
	}
	restartedAt := map[string]bool{}
	for seed := int64(1); seed <= 12; seed++ { // 12: worked-out AC (#35) changed the walks
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir := campSession(t, rng, true)
		copied := copyDir(t, dir)
		var plays []CampRequest
		var want []moment
		for step := 0; step < 70; step++ {
			v := s.State()
			want = append(want, moment{v.Camp, v.Dungeon, partyStanding(s.ctrl.Game)})
			if v.Camp == nil {
				break
			}
			r := pickCamp(v.Camp, rng, step > 60)
			plays = append(plays, r)
			rand.Seed(seed*1000 + int64(step))
			if v := s.CampAction(r); v.Error != "" || v.Unsaved != "" {
				t.Fatalf("seed %d step %d: %+v: %q %q", seed, step, r, v.Error, v.Unsaved)
			}
		}
		s.shutdown()

		s = NewSession(copied)
		for step, w := range want {
			v := s.State()
			if !reflect.DeepEqual(v.Camp, w.camp) || !reflect.DeepEqual(v.Dungeon, w.dungeon) ||
				!slices.Equal(partyStanding(s.ctrl.Game), w.standing) {
				t.Fatalf("seed %d step %d: with restarts\n%+v\nuninterrupted\n%+v", seed, step, v.Camp, w.camp)
			}
			if step == len(plays) {
				break
			}
			rand.Seed(seed*1000 + int64(step))
			s.CampAction(plays[step])
			if s.real && rng.Intn(2) == 0 {
				if c := s.State().Camp; c != nil {
					restartedAt[c.View.Screen] = true
				}
				s.shutdown()
				s = NewSession(copied)
			}
		}
		s.shutdown()
	}
	for _, k := range []string{"menu", "reorder", "inspect", "equip", "drop", "trade", "tradeGold", "tradeItem",
		"spellBooks", "spellList", "spellTarget", "dumapic"} {
		if !restartedAt[k] {
			t.Errorf("never restarted at %s", k)
		}
	}
}

// Disband and MALOR cannot be reached through the bridge: CampAction
// refuses them with the reason, nothing changes (the save included), no
// random number is drawn, and no bound method offers them. A spell with no
// effect in Camp is refused too, unspent.
func TestCampRefusesDisbandAndMalor(t *testing.T) {
	rand.Seed(3)
	s, dir := campSession(t, rand.New(rand.NewSource(3)), true)
	mage := s.ctrl.Game.Town.Party.Members[4]
	mage.SpellKnown[engine.SpellIndex["MALOR"]] = true
	mage.SpellKnown[engine.SpellIndex["HALITO"]] = true
	mage.MageSpells[6], mage.MageSpells[0] = 2, 2
	s.CampAction(CampRequest{}) // save the change
	path := filepath.Join(dir, "roster1.json")

	refused := func(when string, r CampRequest, want string) {
		t.Helper()
		before, _ := os.ReadFile(path)
		rand.Seed(9)
		state := campState(s.ctrl.Game)
		rand.Seed(9)
		v := s.CampAction(r)
		after, _ := os.ReadFile(path)
		if !strings.Contains(v.Error, want) || campState(s.ctrl.Game) != state || !bytes.Equal(before, after) {
			t.Fatalf("%s: error %q, changed %v", when, v.Error, !bytes.Equal(before, after))
		}
	}
	v := s.State()
	disband := slices.IndexFunc(v.Camp.Unavailable, func(u Unavailable) bool { return u.Action == app.ActDisband })
	if slices.Contains(v.Camp.View.Actions, app.ActDisband) || disband < 0 || !strings.Contains(v.Camp.Unavailable[disband].Reason, "finding #26") {
		t.Fatalf("menu: %v %v", v.Camp.View.Actions, v.Camp.Unavailable)
	}
	refused("disband", CampRequest{Action: app.ActDisband}, "Disband is not available")
	if v := s.CampAction(CampRequest{Action: app.ActCampInspect, Slot: 4}); v.Error != "" {
		t.Fatal(v.Error)
	}
	refused("disband while inspecting", CampRequest{Action: app.ActDisband}, "Disband is not available")
	for _, name := range []string{"MALOR", "malor", " Malor ", "maLOR"} {
		refused("cast "+name, CampRequest{Action: app.ActCast, Spell: name}, "MALOR is not available")
	}
	refused("HALITO", CampRequest{Action: app.ActCast, Spell: "HALITO"}, "No effect in Camp")
	refused("unknown", CampRequest{Action: app.ActCast, Spell: "FROTZ"}, "does not know")
	for _, sp := range s.State().Camp.View.Spells {
		if sp.Name == "MALOR" && (sp.Castable || !strings.Contains(sp.Reason, "finding #25")) {
			t.Fatalf("MALOR listed as %+v", sp)
		}
	}
	// The dungeon's actions and the key-level methods do not lead there
	// either: no bound method takes keys or names Disband or MALOR.
	refusedDungeon := s.DungeonAction(app.ActDisband, 0)
	if refusedDungeon.Error == "" {
		t.Fatal("DungeonAction took disband")
	}
	st := reflect.TypeOf(s)
	for i := 0; i < st.NumMethod(); i++ {
		n := strings.ToLower(st.Method(i).Name)
		if strings.Contains(n, "disband") || strings.Contains(n, "malor") || strings.Contains(n, "key") {
			t.Fatalf("bound method %s", st.Method(i).Name)
		}
	}
	bindings, err := os.ReadFile(filepath.Join("frontend", "wailsjs", "go", "main", "Session.js"))
	if err != nil {
		t.Fatal(err)
	}
	if b := strings.ToLower(string(bindings)); strings.Contains(b, "disband") || strings.Contains(b, "malor") || !strings.Contains(b, "campaction") {
		t.Fatal("the generated bindings offer Disband or MALOR, or lack CampAction")
	}
}

// A practice expedition's Camp saves nothing: whatever is done there, the
// roster file is untouched, and when the practice ends the roster is as it
// was.
func TestPracticeCampKeepsNothing(t *testing.T) {
	for seed := int64(1); seed <= 4; seed++ {
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, dir := campSession(t, rng, false)
		path := filepath.Join(dir, "roster1.json")
		before, _ := os.ReadFile(path)
		roster := rosterBytes(s.ctrl.Game) // the characters, not the view: equipping is closed during practice
		for step := 0; step < 80; step++ {
			p := s.State().Camp
			if p == nil {
				break
			}
			if v := s.CampAction(pickCamp(p, rng, false)); v.Error != "" || v.Real {
				t.Fatalf("seed %d: %q", seed, v.Error)
			}
		}
		if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
			t.Fatalf("seed %d: practice Camp wrote the save", seed)
		}
		v := s.AbandonPractice()
		if !bytes.Equal(rosterBytes(s.ctrl.Game), roster) || v.Camp != nil {
			t.Fatalf("seed %d: the roster changed", seed)
		}
	}
}

// The panel shows what the game shows: unidentified items under their
// unknown names (on the sheets and the equip screen) until a bishop
// identifies them, the inspected character's known spells, and DUMAPIC's
// result only on its own screen.
func TestCampPanelShowsWhatTheGameShows(t *testing.T) {
	rand.Seed(5)
	s, _ := campSession(t, rand.New(rand.NewSource(5)), true)
	items := s.scenario.Items
	g := s.ctrl.Game
	bishop := g.Town.Party.Members[3]
	bishop.AddItem(1+rand.Intn(len(items)-1), false)
	for items[bishop.Items[bishop.ItemCount-1].ItemIndex].Name == items[bishop.Items[bishop.ItemCount-1].ItemIndex].NameUnknown {
		bishop.Items[bishop.ItemCount-1].ItemIndex = 1 + rand.Intn(len(items)-1)
	}
	mage := g.Town.Party.Members[4]
	mage.SpellKnown[engine.SpellIndex["DUMAPIC"]] = true
	mage.MageSpells[0] = 2

	sheetNames := func(v View, slot int) []string {
		var out []string
		for i, it := range v.Camp.Party[slot].Items {
			p := g.Town.Party.Members[slot].Items[i]
			want := items[p.ItemIndex].NameUnknown
			if p.Identified {
				want = items[p.ItemIndex].Name
			}
			if it.Name != want || it.Identified != p.Identified || it.Cursed != engine.KnownCursed(p) {
				t.Fatalf("slot %d item %d shown as %+v", slot, i, it)
			}
			out = append(out, it.Name)
		}
		return out
	}
	v := s.State()
	for slot := range v.Camp.Party {
		sheetNames(v, slot)
	}
	v = s.CampAction(CampRequest{Action: app.ActCampInspect, Slot: 3})
	v = s.CampAction(CampRequest{Action: app.ActEquip})
	for v.Camp.View.Screen == "equip" {
		for _, c := range v.Camp.View.Choices {
			if want := sheetNames(v, 3)[c.Item-1]; c.Name != want {
				t.Fatalf("equip choice %q, the sheet says %q", c.Name, want)
			}
		}
		v = s.CampAction(CampRequest{Action: app.ActEquipSkip})
	}
	v = s.CampAction(CampRequest{Action: app.ActIdentify})
	for i, n := range sheetNames(v, 3) {
		if n != items[bishop.Items[i].ItemIndex].Name {
			t.Fatalf("after identify: %q", n)
		}
	}
	// Spells: the inspected character's, as the controller lists them.
	if want, _ := s.ctrl.CampView(); !reflect.DeepEqual(v.Camp.View.Spells, want.Spells) || len(want.Spells) == 0 {
		t.Fatal("the panel's spells are not the controller's")
	}
	v = s.CampAction(CampRequest{Action: app.ActLeaveInspect})
	v = s.CampAction(CampRequest{Action: app.ActCampInspect, Slot: 4})
	noPosition := func(v View, when string) {
		raw, _ := json.Marshal(v)
		if v.Camp.View.Dumapic != nil || bytes.Contains(raw, []byte(`"dumapic"`)) || bytes.Contains(raw, []byte(`"east"`)) {
			t.Fatalf("%s: the view has a position", when)
		}
	}
	noPosition(v, "before DUMAPIC")
	v = s.CampAction(CampRequest{Action: app.ActCast, Spell: "dumapic"})
	if d := v.Camp.View.Dumapic; v.Camp.View.Screen != "dumapic" || d == nil || d.East != g.PlayerX || d.North != g.PlayerY || d.Level != g.MazeLevel+1 {
		t.Fatalf("DUMAPIC: %+v", v.Camp.View)
	}
	v = s.CampAction(CampRequest{Action: app.ActLeaveDumapic})
	noPosition(v, "after DUMAPIC")
}

// The front end's tests (frontend/src/lib/campViews.test.js) run its
// display helpers over real Camp panels. This records one panel for each
// distinct screen, set of actions and unavailable reasons in seeded
// practice expeditions. Run with ISTARY_UPDATE_FIXTURES=1 to rewrite the
// file after a deliberate change.
func TestCampPanelFixture(t *testing.T) {
	var panels []*CampPanel
	have := map[string]bool{}
	for seed := int64(1); seed <= 8; seed++ { // 8: stuck items (#34) changed the walk; DUMAPIC needs the 8th
		rand.Seed(seed)
		rng := rand.New(rand.NewSource(seed))
		s, _ := campSession(t, rng, false)
		if seed == 1 { // a party of one: nothing to reorder
			for len(s.practice.Game.Town.Party.Members) > 1 {
				s.practice.Game.Town.Party.Members = s.practice.Game.Town.Party.Members[:1]
			}
		}
		for step := 0; step < 120; step++ {
			p := s.State().Camp
			if p == nil {
				break
			}
			var unavailable []string
			for _, u := range p.Unavailable {
				unavailable = append(unavailable, u.Action)
			}
			kind := fmt.Sprint(p.View.Screen, p.View.Actions, unavailable, len(p.View.Choices) > 0, p.View.Spell != "")
			if !have[kind] {
				have[kind] = true
				panels = append(panels, p)
			}
			s.CampAction(pickCamp(p, rng, step > 110))
		}
	}
	got, _ := json.Marshal(panels)
	path := filepath.Join("frontend", "src", "lib", "testdata", "camp-views.json")
	if os.Getenv("ISTARY_UPDATE_FIXTURES") != "" {
		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with ISTARY_UPDATE_FIXTURES=1)", err)
	}
	if !bytes.Equal(append(got, '\n'), want) {
		t.Fatal("camp-views.json is out of date (run with ISTARY_UPDATE_FIXTURES=1)")
	}
}
