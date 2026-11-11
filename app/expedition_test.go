package app

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
	"github.com/Senior-Geek/istary/scenarios/wiz1"
)

// pristine returns Wizardry I as loaded (each load is a fresh copy).
func pristine(t *testing.T) *data.Scenario {
	t.Helper()
	s, err := wiz1.Load()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// expeditionTrace describes everything an expedition can change: the maze
// state (mazeSnap), the rest of the checkpoint, every party member in full
// and the changed special squares.
func expeditionTrace(g *engine.GameState, orig *data.Scenario) string {
	var b strings.Builder
	b.WriteString(mazeSnap(g))
	fmt.Fprintf(&b, "fiz%d qp%v alarm%d search%d/%v tryget%v/%d back%v found%d camp%q input%v\n",
		g.Fizzles, g.QuickPlot, g.ChestAlarm, g.SearchMonster, g.SearchCell != nil, g.MazePendingTryGet,
		g.MazeTryGetItem, g.MazePendingBack2Shop, len(g.MazeInspectFound), g.Town.Message, g.Town.InputMode)
	for x := 0; x < 20; x++ {
		for y := 0; y < 20; y++ {
			if g.FightMap[x][y] {
				fmt.Fprintf(&b, "f%d,%d ", x, y)
			}
		}
	}
	for _, m := range g.Town.Party.Members {
		fmt.Fprintf(&b, "\n %s %d/%d %v xp%d g%d %v %v %v %d", m.Name, m.HP, m.MaxHP, m.Status, m.XP, m.Gold,
			m.MageSpells, m.PriestSpells, m.Items, m.PoisonAmt)
	}
	fmt.Fprintf(&b, "\n squares %v", engine.ChangedSquares(orig, g.Scenario))
	return b.String()
}

// partyTrace describes the party members in full.
func partyTrace(g *engine.GameState) string {
	var b strings.Builder
	for _, m := range g.Town.Party.Members {
		fmt.Fprintf(&b, "%s %d/%d %v xp%d g%d %v %v %v %d\n", m.Name, m.HP, m.MaxHP, m.Status, m.XP, m.Gold,
			m.MageSpells, m.PriestSpells, m.Items, m.PoisonAmt)
	}
	return b.String()
}

// reload saves the expedition at its checkpoint and starts again as after a
// restart: a fresh scenario, the roster loaded from disk, and Resume.
func reload(t *testing.T, c *Controller, orig *data.Scenario) *Controller {
	t.Helper()
	exp, err := c.Checkpoint(orig)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	c.Game.Expedition = exp
	if err := c.Game.Save(); err != nil {
		t.Fatal(err)
	}
	g := engine.New(pristine(t))
	g.Title, g.Phase = nil, engine.PhaseTown
	if err := g.Load(); err != nil {
		t.Fatal(err)
	}
	if g.Town.Party.Size() != 0 {
		t.Fatalf("the castle party holds %d expedition members", g.Town.Party.Size())
	}
	c2 := New(g)
	if err := c2.Resume(orig); err != nil {
		t.Fatalf("resume: %v", err)
	}
	return c2
}

// fightOut plays an encounter to its end with fixed choices: the front row
// fights the first living group, the back row parries, chests are left,
// friendly groups are left in peace.
func fightOut(t *testing.T, c *Controller) {
	t.Helper()
	g := c.Game
	for i := 0; i < 20000 && g.Phase == engine.PhaseCombat && g.Combat != nil; i++ {
		cs := g.Combat
		switch cs.Phase {
		case engine.CombatInit:
			c.StartCombat()
		case engine.CombatFriendly:
			c.LeaveInPeace()
		case engine.CombatChoose:
			group := 0
			for gi, gr := range cs.Groups {
				if gr.AliveCnt > 0 {
					group = gi
					break
				}
			}
			for slot, m := range g.Town.Party.Members {
				if m == nil || !CanGiveOrder(m) {
					continue
				}
				o := Order{Action: engine.ActionParry, Item: -1}
				if slot < FrontRowSize {
					o = Order{Action: engine.ActionFight, Group: group, Item: -1}
				}
				if c.SetOrder(m, o) != nil {
					c.SetOrder(m, Order{Action: engine.ActionParry, Item: -1})
				}
			}
			c.FinishOrders()
		case engine.CombatConfirm:
			c.Commit()
		case engine.CombatChest:
			c.LeaveChest()
		case engine.CombatChestResult:
			if done, _ := c.AdvanceChest(); done {
				c.EndChestResult()
			}
		case engine.CombatVictory, engine.CombatDefeat:
			c.LeaveCombat()
		default:
			if s, err := c.Advance(); err != nil || s.Done {
				c.EndRound()
			}
		}
	}
	if g.Phase == engine.PhaseCombat {
		t.Fatalf("combat never ended (phase %v, round %d, groups %d)", g.Combat.Phase, g.Combat.Round, len(g.Combat.Groups))
	}
}

// expeditionStep takes one action chosen by pick: answers the open prompt,
// or moves, turns, kicks, camps or inspects. It returns whether the action
// ended at a safe point outside combat.
func expeditionStep(t *testing.T, c *Controller, pick *rand.Rand) {
	t.Helper()
	g := c.Game
	switch {
	case g.Phase == engine.PhaseCombat:
		fightOut(t, c)
	case g.Phase == engine.PhaseCamp:
		c.LeaveCamp()
	default:
		switch c.mazePrompt() {
		case "messages":
			c.ContinueMessage()
		case "stairs":
			c.Answer(pick.Intn(10) == 0)
		case "search":
			c.Answer(pick.Intn(2) == 0)
		case "buttons":
			c.PressButton(1 + pick.Intn(g.ButtonCount))
		case "inspect":
			if pick.Intn(2) == 0 {
				c.PickUp(1)
			}
			c.LeaveInspect()
		default:
			switch r := pick.Intn(20); {
			case r < 12:
				if c.Forward() != nil {
					c.Kick()
				}
			case r < 15:
				c.TurnRight()
			case r < 17:
				c.TurnLeft()
			case r < 18:
				c.Camp()
			case r < 19:
				c.Inspect()
			default:
				c.Kick()
			}
		}
	}
}

// approach puts the party (out of Camp) next to a special square chosen by
// pick (from only, if given, as level, x, y), facing it through an open
// wall or a door, so the next Forward (or Kick) enters it. It is test setup, the same in both
// runs it is used in.
func approach(t *testing.T, c *Controller, pick *rand.Rand, only ...[3]int) {
	t.Helper()
	g := c.Game
	kinds := map[data.SquareType]bool{data.SqButtons: true, data.SqScnMsg: true, data.SqSpclEnctr: true,
		data.SqFizzle: true, data.SqOuchy: true, data.SqDark: true, data.SqSpinner: true, data.SqPit: true}
	type sq struct{ l, x, y int }
	var specials []sq
	for l, lv := range g.Scenario.Mazes.Levels {
		for y, row := range lv.Cells {
			for x, cell := range row {
				if kinds[cell.Type] {
					specials = append(specials, sq{l, x, y})
				}
			}
		}
	}
	if len(only) > 0 {
		specials = nil
		for _, o := range only {
			specials = append(specials, sq{o[0], o[1], o[2]})
		}
	}
	must(t, c.LeaveCamp())
	if c.mazePrompt() == "stairs" {
		must(t, c.Answer(false))
	}
	for try := 0; try < 50; try++ {
		s := specials[pick.Intn(len(specials))]
		for f := engine.North; f <= engine.West; f++ {
			g.MazeLevel, g.Facing = s.l, f
			g.PlayerX, g.PlayerY = (s.x-f.DX()+20)%20, (s.y-f.DY()+20)%20
			if g.WallAhead() != data.WallWall && g.CurrentCell().Type == data.SqNormal {
				g.InitFightMap()
				return
			}
		}
	}
	t.Fatal("no special square could be approached")
}

// target picks where a walk starts: for every sixth seed B1F (13,5), the
// search square whose Yes starts a special encounter (Murphy's ghost) and
// changes the square; for others a buttons square; otherwise any special
// square.
func target(s int64, g *engine.GameState) [][3]int {
	switch s % 6 {
	case 0:
		return [][3]int{{0, 13, 5}}
	case 4:
		var out [][3]int
		for l, lv := range g.Scenario.Mazes.Levels {
			for y, row := range lv.Cells {
				for x, cell := range row {
					if cell.Type == data.SqButtons {
						out = append(out, [3]int{l, x, y})
					}
				}
			}
		}
		return out
	}
	return nil
}

// explorersWithSaves is explorers, with saves in a temporary folder.
func explorersWithSaves(t *testing.T) *Controller {
	t.Helper()
	engine.SetSaveDir(t.TempDir())
	t.Cleanup(func() { engine.SetSaveDir("") })
	return explorers(t)
}

// Saving at a safe point and resuming after a restart changes nothing: for
// the same seed and the same actions, play that is saved, reloaded and
// resumed after every safe point follows the uninterrupted play exactly,
// state for state and random number for random number (no movement, square
// effect, encounter check or random draw is replayed or skipped).
func TestResumeFollowsUninterruptedPlay(t *testing.T) {
	orig := pristine(t)
	seen := map[string]int{}
	for s := int64(1); s <= 30; s++ {
		const steps = 300
		near := s%2 == 0 // half the walks start beside a special square
		// Uninterrupted.
		seed(s)
		pick := rand.New(rand.NewSource(s))
		c := explorersWithSaves(t)
		must(t, c.EnterMaze())
		if near {
			approach(t, c, pick, target(s, c.Game)...)
		}
		var want []string
		for i := 0; i < steps && c.Game.Phase != engine.PhaseTown; i++ {
			expeditionStep(t, c, pick)
			want = append(want, expeditionTrace(c.Game, orig))
		}

		// The same, saved and resumed after every safe point.
		seed(s)
		pick = rand.New(rand.NewSource(s))
		c = explorersWithSaves(t)
		must(t, c.EnterMaze())
		if near {
			approach(t, c, pick, target(s, c.Game)...)
		}
		for i := 0; i < len(want); i++ {
			expeditionStep(t, c, pick)
			if c.Game.Phase == engine.PhaseMaze || c.Game.Phase == engine.PhaseCamp {
				kind := c.mazePrompt()
				if c.Game.Phase == engine.PhaseCamp {
					kind = "camp"
				}
				seen["checkpoint "+kind]++
				c = reload(t, c, orig)
			}
			if got := expeditionTrace(c.Game, orig); got != want[i] {
				t.Fatalf("seed %d step %d: after save and resume\n%s\nuninterrupted\n%s", s, i, got, want[i])
			}
		}
		if len(engine.ChangedSquares(orig, c.Game.Scenario)) > 0 {
			seen["changed squares"]++
		}
		if c.Game.LightLevel > 0 {
			seen["light"]++
		}
	}
	t.Logf("covered: %v", seen)
	for _, k := range []string{"checkpoint ", "checkpoint camp", "checkpoint messages", "checkpoint stairs", "checkpoint inspect",
		"checkpoint buttons", "checkpoint search", "changed squares"} {
		if seen[k] == 0 {
			t.Errorf("not covered: %s", k)
		}
	}
}

// Every field of the checkpoint comes back, set by hand to values a walk
// rarely reaches: light, protection, anti-magic, quick plot, a pending
// alarm, a search prompt, a pending item, the fight zones and changed
// squares.
func TestResumeRestoresEveryField(t *testing.T) {
	orig := pristine(t)
	c := explorersWithSaves(t)
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(false))
	g := c.Game
	g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing = 3, 7, 12, engine.West
	g.InitFightMap()
	g.LightLevel, g.ProtectLevel, g.Fizzles, g.QuickPlot, g.ChestAlarm = 17, 4, 1, true, 1
	g.ViewportMsg, g.ViewportMsg2 = "DARK", "AREA"
	g.MazeMessages, g.MazeMsgScroll, g.MazeMsgWait = []string{"A", "B", "C", "D", "E", "F"}, 4, false
	g.MazeSearchYN, g.SearchCell, g.SearchMonster = true, g.CurrentCell(), 77
	g.MazePendingTryGet, g.MazeTryGetItem = true, 12
	cell := &g.Scenario.Mazes.Levels[2].Cells[4][9]
	cell.Count, cell.Type, cell.SpclMonster = cell.Count+3, data.SqNormal, 5
	before := expeditionTrace(g, orig)

	c2 := reload(t, c, orig)
	if got := expeditionTrace(c2.Game, orig); got != before {
		t.Fatalf("after resume\n%s\nwant\n%s", got, before)
	}
	if c2.Game.SearchCell != c2.Game.CurrentCell() {
		t.Fatal("the search prompt is not for the party's square")
	}
	if c2.Game.Expedition == nil || c2.Game.Expedition.Where.Level != 3 {
		t.Fatal("the loaded expedition is not kept")
	}
}

// An encounter interrupted by a restart is over: the party is back in the
// maze on the square where it began (already entered, its fight room
// cleared), with every committed loss and change kept and nothing awarded.
// Resuming again does not start it again; the square is not re-entered.
func TestResumeAfterInterruptedEncounter(t *testing.T) {
	orig := pristine(t)
	met := 0
	for s := int64(1); s <= 60 && met < 12; s++ {
		seed(s)
		pick := rand.New(rand.NewSource(s))
		c := explorersWithSaves(t)
		must(t, c.EnterMaze())
		for i := 0; i < 600 && c.Game.Phase != engine.PhaseCombat; i++ {
			expeditionStep(t, c, pick)
		}
		if c.Game.Phase != engine.PhaseCombat {
			continue
		}
		met++
		g := c.Game
		where := fmt.Sprintf("L%d (%d,%d) %v", g.MazeLevel, g.PlayerX, g.PlayerY, g.Facing)
		fight := fmt.Sprint(g.FightMap)
		// Play part of it: the opening and one round, if it gets there.
		c.StartCombat()
		if g.Combat.Phase == engine.CombatChoose {
			fightOut(t, c) // to the end, then put an encounter back to interrupt
			if g.Phase != engine.PhaseMaze {
				continue
			}
			g.Combat = engine.NewCombat(g)
			g.Phase = engine.PhaseCombat
			fight = fmt.Sprint(g.FightMap)
		}
		party := partyTrace(g)
		g.Combat.HasChest = true

		exp, err := c.Checkpoint(orig)
		must(t, err)
		if exp.Interrupted != InterruptedCombat {
			t.Fatalf("seed %d: interrupted %q", s, exp.Interrupted)
		}
		c2 := reload(t, c, orig)
		g2 := c2.Game
		if g2.Phase != engine.PhaseMaze || g2.Combat != nil || g2.MazeMessage != "" {
			t.Fatalf("seed %d: after resume phase %v combat %v message %q", s, g2.Phase, g2.Combat != nil, g2.MazeMessage)
		}
		if got := fmt.Sprintf("L%d (%d,%d) %v", g2.MazeLevel, g2.PlayerX, g2.PlayerY, g2.Facing); got != where {
			t.Fatalf("seed %d: resumed at %s, the encounter began at %s", s, got, where)
		}
		if fmt.Sprint(g2.FightMap) != fight {
			t.Fatalf("seed %d: fight zones changed", s)
		}
		if got := partyTrace(g2); got != party {
			t.Fatalf("seed %d: party after resume\n%s\nwant (losses kept, nothing awarded)\n%s", s, got, party)
		}
		// Resuming again (another restart) changes nothing.
		before := expeditionTrace(g2, orig)
		if after := expeditionTrace(reload(t, c2, orig).Game, orig); after != before {
			t.Fatalf("seed %d: a second restart changed the expedition", s)
		}
	}
	if met < 5 {
		t.Fatalf("only %d encounters met", met)
	}
}

// Changes outside the characters during an encounter are committed: a
// chest's teleporter moves the party and its alarm stays set, so a restart
// cannot undo them.
func TestInterruptedChestKeepsItsConsequences(t *testing.T) {
	orig := pristine(t)
	c := explorersWithSaves(t)
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	must(t, c.Answer(false))
	g := c.Game
	g.Combat = engine.NewCombat(g)
	g.Phase = engine.PhaseCombat
	g.Combat.Phase = engine.CombatChest
	// What the teleporter and the alarm leave (engine/combat.go applyTrap).
	g.PlayerX, g.PlayerY, g.ChestAlarm = 13, 6, 1
	exp, err := c.Checkpoint(orig)
	must(t, err)
	if exp.Interrupted != InterruptedChest {
		t.Fatalf("interrupted %q", exp.Interrupted)
	}
	g2 := reload(t, c, orig).Game
	if g2.PlayerX != 13 || g2.PlayerY != 6 || g2.ChestAlarm != 1 {
		t.Fatalf("resumed at (%d,%d) alarm %d", g2.PlayerX, g2.PlayerY, g2.ChestAlarm)
	}
}

// A saved expedition that does not match the roster or the maze is never
// resumed half-way or discarded: Resume says why and changes nothing;
// ReturnExpeditionToCastle then puts the members in the castle party,
// all or none.
func TestResumeRefusesInconsistentExpeditions(t *testing.T) {
	orig := pristine(t)
	c := explorersWithSaves(t)
	must(t, c.EnterMaze())
	must(t, c.LeaveCamp())
	good, err := c.Checkpoint(orig)
	must(t, err)

	for name, spoil := range map[string]func(e *engine.Expedition, g *engine.GameState){
		"missing member":  func(e *engine.Expedition, g *engine.GameState) { e.Party[2] = "NOBODY" },
		"listed twice":    func(e *engine.Expedition, g *engine.GameState) { e.Party[1] = e.Party[0] },
		"off the map":     func(e *engine.Expedition, g *engine.GameState) { e.Where.X = 20 },
		"negative floor":  func(e *engine.Expedition, g *engine.GameState) { e.Where.Level = -1 },
		"bad facing":      func(e *engine.Expedition, g *engine.GameState) { e.Where.Facing = 4 },
		"fight zones":     func(e *engine.Expedition, g *engine.GameState) { e.Where.FightMap = e.Where.FightMap[:3] },
		"unknown place":   func(e *engine.Expedition, g *engine.GameState) { e.Where.Phase = "castle" },
		"square outside":  func(e *engine.Expedition, g *engine.GameState) { e.Squares = []engine.SquareState{{Level: 12}} },
		"castle party":    func(e *engine.Expedition, g *engine.GameState) { g.Town.Party.Members = g.Town.Roster.Characters[:1] },
		"two of the same": func(e *engine.Expedition, g *engine.GameState) { g.Town.Roster.Add(member(e.Party[0], engine.Neutral)) },
	} {
		t.Run(name, func(t *testing.T) {
			g := engine.New(pristine(t))
			g.Title, g.Phase = nil, engine.PhaseTown
			for _, ch := range c.Game.Town.Roster.Characters {
				g.Town.Roster.Add(ch)
			}
			e := *good
			e.Party = append([]string(nil), good.Party...)
			e.Where.FightMap = append([]string(nil), good.Where.FightMap...)
			spoil(&e, g)
			g.Expedition = &e
			before := expeditionTrace(g, orig)
			c2 := New(g)
			var p *ExpeditionProblem
			if err := c2.Resume(orig); !errors.As(err, &p) {
				t.Fatalf("Resume = %v, want an ExpeditionProblem", err)
			}
			if expeditionTrace(g, orig) != before || g.Phase != engine.PhaseTown || g.Expedition != &e {
				t.Fatal("a refused resume changed the game")
			}
		})
	}

	// Back to the castle: all members join the castle party, or none.
	g := engine.New(pristine(t))
	g.Title, g.Phase = nil, engine.PhaseTown
	for _, ch := range c.Game.Town.Roster.Characters {
		g.Town.Roster.Add(ch)
	}
	g.Expedition = good
	g.Town.Party.Members = []*engine.Character{member("STRANGER", engine.Neutral)}
	c2 := New(g)
	if err := c2.ReturnExpeditionToCastle(); err == nil || g.Expedition == nil || g.Town.Party.Size() != 1 {
		t.Fatalf("no room: %v, party %d", err, g.Town.Party.Size())
	}
	g.Town.Party.Members = nil
	must(t, c2.ReturnExpeditionToCastle())
	if g.Expedition != nil || g.Town.Party.Size() != len(good.Party) {
		t.Fatalf("back in the castle: party %d", g.Town.Party.Size())
	}
}
