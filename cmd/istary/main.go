package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
	"github.com/Senior-Geek/istary/render"
	"github.com/Senior-Geek/istary/scenarios/wiz1"
	"github.com/Senior-Geek/istary/scenarios/wiz2"
	"github.com/Senior-Geek/istary/scenarios/wiz3"
)

var version = "1.2.2"
var buildDate string // set via ldflags: -ldflags "-X main.buildDate=15-APR-26"

func main() {
	scenarioName := "1"
	vpScale := 1.0 // --vpscale: viewport scale (1.0=100%, 1.5=150%, etc.)
	forceColor := false
	forceGreen := false

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--color" {
			forceColor = true
		} else if arg == "--green" {
			forceGreen = true
		} else if strings.HasPrefix(arg, "--vpscale=") {
			val := strings.TrimPrefix(arg, "--vpscale=")
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				if v < 1.0 {
					v = 1.0
				}
				vpScale = v
			}
		} else if arg == "--vpscale" && i+1 < len(os.Args) {
			i++
			if v, err := strconv.ParseFloat(os.Args[i], 64); err == nil {
				if v < 1.0 {
					v = 1.0
				}
				vpScale = v
			}
		} else if strings.HasPrefix(arg, "--scenario=") {
			scenarioName = strings.TrimPrefix(arg, "--scenario=")
		} else if arg == "--scenario" && i+1 < len(os.Args) {
			i++
			scenarioName = os.Args[i]
		}
	}

	scenario, err := loadScenario(scenarioName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", scenarioName, err)
		os.Exit(1)
	}

	game := engine.New(scenario)

	// Set version info for title screen display
	// Strip trailing ".0" for display (1.0.0 → 1.0, 1.1.0 → 1.1, 1.1.1 → 1.1.1)
	displayVersion := strings.TrimSuffix(version, ".0")
	game.Version = displayVersion
	if buildDate == "" {
		// Not set via ldflags — use current date
		buildDate = strings.ToUpper(time.Now().Format("02-Jan-06"))
	}
	game.BuildDate = buildDate

	game.Load() // restore roster/party from ~/.config/wizardry/ (Load works out every AC)

	// Detect sixel support BEFORE tcell takes over the terminal.
	// DetectSixel sends DA1 (ESC[c) and checks for attribute 4.
	render.SixelSupported = render.DetectSixel()

	screen, err := render.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Screen init failed: %v\n", err)
		os.Exit(1)
	}
	defer screen.Close()

	screen.VPScale = vpScale

	// Color mode: --green forces monochrome phosphor, --color forces color,
	// otherwise auto-detect based on terminal color support (256+ colors).
	if forceGreen {
		render.ColorMode = false
	} else if forceColor {
		render.ColorMode = true
	} else {
		render.ColorMode = screen.Colors() >= 256
	}
	if render.ColorMode {
		render.ApplyColorMode()
	}

	// Detect cell pixel size via ioctl TIOCGWINSZ — safe after tcell init,
	// no escape sequences, no stdin reads.
	if render.SixelSupported {
		render.CellWidth, render.CellHeight = render.DetectCellSize()
	}

	// Check terminal size — need at least 80x24 (40 logical cols × 2 scale)
	w, h := screen.Size()
	if w < 80 || h < 24 {
		screen.Close()
		fmt.Fprintf(os.Stderr, "Terminal too small: %dx%d. Minimum required: 80x24.\n", w, h)
		os.Exit(1)
	}

	redraw(screen, game)

	// Run the timed title text + animated art sequence in background (Wiz 1 only)
	// Wiz 2 has static title image (no animation), Wiz 3 has no title at all
	if game.Phase == engine.PhaseTitle && game.Scenario.TitleWT != nil {
		game.Title.Done = make(chan struct{})
		go startTitleSequence(screen, game)
	}

	// Use channel-based events for non-blocking timer support.
	// Combat messages auto-advance per p-code WIZARDRY.proc30 timed wait.
	quit := make(chan struct{})
	defer close(quit)
	eventCh := make(chan tcell.Event, 10)
	go func() {
		for {
			ev := screen.PollEvent()
			if ev == nil {
				return
			}
			eventCh <- ev
		}
	}()

	// Combat auto-advance poll timer.
	// Original Apple II PAUSE1 uses the T)IME delay value (global 7, range 1-5000)
	// as a busy-wait loop count. We map it to milliseconds: delay_ms = MazeDelay * 0.3.
	// Default MazeDelay=0 uses 1500ms. Poll every 100ms, advance when enough time elapsed.
	combatTicker := time.NewTicker(100 * time.Millisecond)
	defer combatTicker.Stop()
	var lastCombatAdvance time.Time

	// Story auto-advance timer: 6 seconds per page (Wiz 3 title sequence)
	// Original Apple II auto-advances with no input required
	storyTicker := time.NewTicker(6 * time.Second)
	defer storyTicker.Stop()

	// Inn healing timer: ~0.7 seconds per heal tick
	// Pascal HEALHP (CASTLE2.TEXT line 445): FOR PAUSEX := 1 TO 500 DO ;
	// On 1 MHz Apple II with p-code interpreter, this is roughly 0.5-1 second.
	innTicker := time.NewTicker(700 * time.Millisecond)
	defer innTicker.Stop()

	ctrl := app.New(game)
	ctrl.Beep = screen.Beep

	for {
		select {
		case ev := <-eventCh:
			switch ev := ev.(type) {
			case *tcell.EventResize:
				redraw(screen, game)

			case *tcell.EventKey:
				if ev.Key() == tcell.KeyCtrlC {
					return
				}

				if dispatchKey(ctrl, screen, ev) {
					return
				}
				ctrl.Save() // auto-save after every action
				redraw(screen, game)
			}

		case <-combatTicker.C:
			if combatTimerTick(game, time.Since(lastCombatAdvance)) {
				lastCombatAdvance = time.Now()
				redraw(screen, game)
			}

		case <-storyTicker.C:
			// Auto-advance Wiz 3 story pages every 6 seconds (no input required)
			if game.Phase == engine.PhaseTitle && game.Title != nil &&
				game.Title.Step == engine.TitleStory {
				title := game.Title
				title.StoryFrame++
				totalFrames := len(game.Scenario.TitleFrames)
				if totalFrames == 0 {
					totalFrames = len(game.Scenario.TitleStory)
				}
				if title.StoryFrame >= totalFrames {
					// No keypress through entire sequence → loop back to start
					// (p-code: cycling loop at offsets 855-871 in TITLELOA)
					title.StoryFrame = 0
				}
				redraw(screen, game)
			}

		case <-innTicker.C:
			if ctrl.InnTick() {
				redraw(screen, game)
			}
		}
	}
}


// combatTimerTick runs one poll of the combat auto-advance timer. sinceLast is
// the time since the last advance. It returns true when combat advanced (the
// caller then resets its clock and redraws).
func combatTimerTick(game *engine.GameState, sinceLast time.Duration) bool {
	if game.Phase != engine.PhaseCombat || game.Combat == nil {
		return false
	}
	combat := game.Combat

	// Compute delay based on combat phase.
	// PAUSE1 (CombatInit/CombatExecute): T)IME-controlled (MazeDelay 1-5000 → ms).
	// PAUSE2 (CombatChestResult/CombatVictory/CombatDefeat): fixed delay —
	//   Pascal PAUSE2 hardcodes 3000, independent of T)IME setting.
	delayMs := 1500 // default (PAUSE2 fixed, and PAUSE1 with no T)IME set)
	if combat.Phase == engine.CombatInit || combat.Phase == engine.CombatExecute {
		// PAUSE1: uses T)IME setting
		if game.MazeDelay > 0 {
			delayMs = game.MazeDelay * 3 / 10 // ~0.3ms per unit
			if delayMs < 50 {
				delayMs = 50 // minimum 50ms
			}
		}
	}
	if sinceLast < time.Duration(delayMs)*time.Millisecond {
		return false // not enough time yet
	}

	// The combat lifecycle is the controller's (app/combat_round.go); this
	// loop only decides when, by the PAUSE1/PAUSE2 delays above.
	ctrl := app.New(game)
	switch {
	case combat.Phase == engine.CombatInit:
		// Auto-advance after delay (simulates Apple II disk I/O time); a
		// monster surprise round is executed here.
		ctrl.StartCombat()
	case combat.Phase == engine.CombatExecute && !combat.HamanSelecting:
		// One message block per PAUSE1; after the last, the round ends.
		if step, err := ctrl.Advance(); err == nil && step.Done {
			ctrl.EndRound()
		}
	case combat.Phase == engine.CombatChestResult:
		advanceChestMessages(game)
	case combat.Phase == engine.CombatVictory || combat.Phase == engine.CombatDefeat:
		// PAUSE2 twice, then the controller leaves combat (victory: the
		// maze; defeat: the castle).
		if !combat.ChestPauseUsed {
			combat.ChestPauseUsed = true
		} else {
			ctrl.LeaveCombat()
		}
	default:
		return false
	}
	return true
}

func redraw(screen *render.Screen, game *engine.GameState) {
	switch game.Phase {
	case engine.PhaseTitle:
		screen.RenderTitle(game)
	case engine.PhaseTown:
		screen.RenderTown(game)
	case engine.PhaseCamp:
		screen.RenderCamp(game)
	case engine.PhaseMaze:
		if game.ShowMap {
			screen.RenderMap(game)
		} else {
			screen.RenderMaze(game)
		}
	case engine.PhaseCombat:
		screen.RenderCombat(game)
	case engine.PhaseUtilities:
		screen.RenderUtilities(game)
	case engine.PhaseCreation:
		screen.RenderCreation(game.Town.Creation)
	}
}

// startTitleSequence runs the timed text intro then wizard art.
// From p-code TITLELOA (SYSTEM.STARTUP seg 2):
//
//	GOTOXY(12,10) "PREPARE YOURSELF"  → CLP 5 delay(150)
//	GOTOXY(12,12) "FOR THE ULTIMATE"  → CLP 5 delay(150)
//	GOTOXY(12,14) "IN FANTASY GAMES"  → CLP 5 delay(500)
//	→ soft switches to Hi-Res → wizard art animation
//	→ falls through to OPTIONS (seg 3) = menu screen
//
// Any key during text or art skips immediately to the menu.
// Delays scaled up from p-code values (Apple II 1MHz made them feel longer).
func startTitleSequence(screen *render.Screen, game *engine.GameState) {
	if game.Title != nil && game.Title.Done != nil {
		defer close(game.Title.Done)
	}
	skipped := func() bool {
		return game.Title == nil || game.Title.Skipped
	}
	delay := func(d time.Duration) bool {
		time.Sleep(d)
		return skipped()
	}

	// Create WT animation engine if we have the data
	var anim *render.WTAnimation
	if game.Scenario.TitleWT != nil {
		anim = render.NewWTAnimation(game.Scenario.TitleWT)
	}

	// Phase 1: timed text intro
	// From Pascal TITLELOA: GOTOXY/WRITESTR with CHKEYPR delays
	delays := []time.Duration{
		800 * time.Millisecond,
		800 * time.Millisecond,
		1500 * time.Millisecond,
	}
	for i := 0; i < 3; i++ {
		if skipped() {
			return
		}
		game.Title.TextLine = i
		redraw(screen, game)
		if delay(delays[i]) {
			return
		}
	}

	if skipped() {
		return
	}
	game.Title.Step = engine.TitleArt

	// Store animation engine on the title state for the renderer to use
	game.Title.Anim = anim

	// Animation frame rate
	frameDur := 80 * time.Millisecond

	// LZFLAG alternates between XOR toggle and scroll on each loop
	lzflag := 0xFD

	for {
		if skipped() {
			return
		}

		if anim == nil {
			// Fallback: static bitmap reveal (no WT data)
			game.Title.AnimRow = 0
			redraw(screen, game)
			if delay(5 * time.Second) {
				return
			}
			continue
		}

		// ── Phase 2: Clear and draw base scene (section 0) ──
		// Pascal: LZDECOMP(WTBUFF[0], WTBUFF[8]); FILLCHAR($2000, 8192, 0)
		anim.DrawSection(8)
		anim.ClearHires()

		// ── Phase 3: Draw base fire scene + animate fire (20 frames) ──
		// Pascal: LZDECOMP(WTBUFF[0], WTBUFF[0]); FOR MP02:=1 TO 20 DO P070206
		anim.DrawSection(0)
		mp04 := 0
		mp05 := 4

		emitFrame := func() {
			if skipped() {
				return
			}
			if render.SixelSupported {
				anim.EmitSixelFrame()
				screen.MarkSixel()
			} else {
				screen.Clear()
				anim.EmitCanvasFrame(screen, render.BaseStyle)
				screen.Show()
			}
		}

		emitFrame()
		for i := 0; i < 20; i++ {
			if skipped() {
				return
			}
			time.Sleep(frameDur)
			mp04 = 1 + (mp04 % 3) // cycles 1,2,3,1,2,3...
			anim.DrawSection(mp04)
			emitFrame()
		}

		// ── Phase 4: Fire + magic waves (24 rounds of 2 fire + 1 wave) ──
		for i := 0; i < 24; i++ {
			if skipped() {
				return
			}
			for j := 0; j < 2; j++ {
				time.Sleep(frameDur)
				mp04 = 1 + (mp04 % 3)
				anim.DrawSection(mp04)
				emitFrame()
			}
			mp05 = 5 + ((mp05 - 4) % 3) // cycles 5,6,7,5,6,7...
			anim.DrawSection(mp05)
			emitFrame()
		}

		// ── Phase 5: Scene reset + smoke/dragon reveal (sections 9-23) ──
		anim.DrawSection(8) // reset scene
		mp04 = 1 + (mp04 % 3)
		anim.DrawSection(mp04)
		emitFrame()

		for sec := 9; sec <= 23; sec++ {
			if skipped() {
				return
			}
			for j := 0; j < 2; j++ {
				// Increasing delay: 5+sec iterations in original
				time.Sleep(frameDur + time.Duration(sec*4)*time.Millisecond)
				mp04 = 1 + (mp04 % 3)
				anim.DrawSection(mp04)
				emitFrame()
			}
			anim.DrawSection(sec)
			emitFrame()
		}

		// ── Phase 6: Title text reveal (sections 24-32) ──
		for sec := 24; sec <= 32; sec++ {
			if skipped() {
				return
			}
			for j := 0; j < 4; j++ {
				time.Sleep(frameDur)
				mp04 = 1 + (mp04 % 3)
				anim.DrawSection(mp04)
				emitFrame()
			}
			anim.DrawSection(sec)
			emitFrame()
		}

		// ── Phase 7: Hold with fire (40 frames) ──
		for i := 0; i < 40; i++ {
			if skipped() {
				return
			}
			time.Sleep(frameDur)
			mp04 = 1 + (mp04 % 3)
			anim.DrawSection(mp04)
			emitFrame()
		}

		// ── Phase 8: Scroll/flash effect (80 frames) ──
		for i := 0; i < 80; i++ {
			if skipped() {
				return
			}
			if lzflag == 0xFD {
				anim.XorToggle()
			} else {
				anim.ScrollLeft()
			}
			time.Sleep(60 * time.Millisecond)
			mp04 = 1 + (mp04 % 3)
			anim.DrawSection(mp04)
			emitFrame()
		}

		// ── Phase 9: Final hold (40 frames) ──
		for i := 0; i < 40; i++ {
			if skipped() {
				return
			}
			time.Sleep(frameDur)
			mp04 = 1 + (mp04 % 3)
			anim.DrawSection(mp04)
			emitFrame()
		}

		// Toggle LZFLAG for next loop iteration
		if lzflag == 0xFD {
			lzflag = 0xFE
		} else {
			lzflag = 0xFD
		}

		// Loop back: switch to text mode for PREPARE YOURSELF, then re-enter art
		if skipped() {
			return
		}
		game.Title.Step = engine.TitleText
		game.Title.TextLine = -1
		for i := 0; i < 3; i++ {
			if skipped() {
				return
			}
			game.Title.TextLine = i
			redraw(screen, game)
			if delay(delays[i]) {
				return
			}
		}
		if skipped() {
			return
		}
		game.Title.Step = engine.TitleArt
	}
}

// handleTitleInput processes keyboard input during the title screen.
// From p-code: any key during text or art → skip to menu (OPTIONS seg 3).
// At menu: S=start game, U=utilities (not yet), T=show title art again.
func handleTitleInput(screen *render.Screen, game *engine.GameState, ev *tcell.EventKey) {
	title := game.Title
	if title == nil {
		game.Phase = engine.PhaseTown
		return
	}

	switch title.Step {
	case engine.TitleStory:
		// Any keypress exits the story sequence → menu (p-code: CSP 4 XIT in proc 3)
		title.Step = engine.TitleMenu
		if render.SixelSupported {
			fmt.Fprintf(os.Stdout, "\x1b[2J")
			os.Stdout.Sync()
		}

	case engine.TitleText, engine.TitleArt:
		// Any key skips to the menu — from p-code: keypress exits title sequence
		title.Skipped = true
		// Wait for animation goroutine to exit so it can't write more sixel frames
		if title.Done != nil {
			<-title.Done
			title.Done = nil
		}
		title.Step = engine.TitleMenu
		// Force clear sixel layer — animation left residual sixel content
		if render.SixelSupported {
			fmt.Fprintf(os.Stdout, "\x1b[2J")
			os.Stdout.Sync()
		}

	case engine.TitleMenu:
		// From p-code OPTIONS (seg 3): wait for S, U, or T
		if ev.Key() != tcell.KeyRune {
			return
		}
		ch := ev.Rune()
		if ch >= 'a' && ch <= 'z' {
			ch -= 32
		}
		switch ch {
		case 'S': // S)TART GAME → enter castle
			game.Title = nil
			game.Phase = engine.PhaseTown
		case 'T': // T)ITLE PAGE → re-show title art/story
			if len(game.Scenario.TitleFrames) > 0 || len(game.Scenario.TitleStory) > 0 {
				// Wiz 3: restart the story sequence — clear screen first
				// to prevent menu text bleeding through sixel frames
				screen.Clear()
				screen.Show()
				title.Step = engine.TitleStory
				title.StoryFrame = 0
				return
			}
			if game.Scenario.Title == nil {
				return // no title art
			}
			if game.Scenario.TitleWT != nil {
				// Wiz 1: full text + animated art sequence
				title.Step = engine.TitleText
				title.TextLine = -1
				title.Skipped = false
				title.Done = make(chan struct{})
				go startTitleSequence(screen, game)
			} else {
				// Wiz 2: static title image (no animation)
				title.Step = engine.TitleArt
				title.AnimRow = 0
			}
		case 'U': // U)TILITIES — from p-code WIZBOOT offset 228: CXP seg=7 proc=1
			game.Util = engine.NewUtilState()
			game.Phase = engine.PhaseUtilities
		}
	}
}

// handleTownInput processes keyboard input during town phase.
// Returns true if the game should exit.
func handleTownInput(game *engine.GameState, ev *tcell.EventKey) bool {
	town := game.Town

	// If we're in a text input prompt, handle that first
	if town.InputMode != engine.InputNone {
		return handleTownPrompt(game, ev)
	}

	// [RETURN] = leave (context-dependent)
	if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyEscape {
		if town.Location == engine.Temple {
			handleTempleReturn(game, town)
			return false
		}
		if town.Location != engine.Castle {
			app.GoToCastle(town)
			return false
		}
	}

	if ev.Key() != tcell.KeyRune {
		return false
	}

	ch := ev.Rune()
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}
	town.Message = ""

	switch town.Location {
	case engine.Castle:
		// From p-code: A=Inn, G=Tavern, B=Boltac's, C=Temple, E=Edge
		// Inn/Boltac's/Temple require a party (from original game flow)
		switch ch {
		case 'G':
			town.Location = engine.Tavern
		// A (the Inn) and B (Boltac's) are handled by app.Controller
		// (app/inn.go, app/shop.go).
		case 'C':
			if town.Party.Size() == 0 {
				town.Message = "YOU HAVE NO PARTY!"
			} else {
				town.Location = engine.Temple
				town.TempleStep = engine.TempleWho
				town.TempleChar = nil
				town.TempleCost = 0
				town.TempleMessages = nil
				town.InputMode = engine.InputTempleHelp
				town.InputBuf = ""
			}
		case 'E':
			town.Location = engine.EdgeOfTown
		case 'Q':
			return true
		}

	// The Inn is handled by app.Controller (app/inn.go).

	// Boltac's Trading Post is handled by app.Controller (app/shop.go).

	case engine.Temple:
		handleTempleInput(game, town, ch)

	case engine.EdgeOfTown:
		// From p-code: M/T/C/L when party exists, T/C/L when no party
		switch ch {
		// M (enter the maze) is handled by app.Controller.EnterMaze.
		case 'T': // Training Grounds
			town.Location = engine.Training
		case 'C': // Castle
			app.GoToCastle(town)
		case 'L': // Leave game
			return true
		}

	}
	return false
}

// handleTownPrompt handles text input at a "WHO WILL JOIN ? >" style prompt.
func handleTownPrompt(game *engine.GameState, ev *tcell.EventKey) bool {
	town := game.Town

	// Temple name input — "WHO ARE YOU HELPING ? >"
	if town.InputMode == engine.InputTempleHelp {
		return handleTempleNameInput(game, ev)
	}


	// Set new password — from ROLLER proc 3 (IC 4480-4816)
	if town.InputMode == engine.InputSetPassword {
		return handleSetPassword(game, ev)
	}

	// Class change selection — from Pascal ROLLER.TEXT CHGCLASS (lines 599-638)
	if town.InputMode == engine.InputClassChange {
		return handleClassChange(game, ev)
	}

	// Rite of Passage ceremony — waiting for RETURN
	if town.InputMode == engine.InputRiteCeremony {
		if ev.Key() == tcell.KeyEnter {
			c := town.EditChar
			good, neut, evil := engine.RiteAlignOptions(c.Class)
			if !good && !neut && !evil {
				// Lord → forced Good, Ninja → forced Evil
				if c.Class == engine.Lord {
					c.Alignment = engine.Good
				} else {
					c.Alignment = engine.Evil
				}
				engine.RiteApply(c)
				town.Message = c.Name + " IS NOW A LEGACY!"
				town.InputMode = engine.InputCharEdit
				game.Save()
			} else {
				town.RiteAlignGood = good
				town.RiteAlignNeut = neut
				town.RiteAlignEvil = evil
				town.InputMode = engine.InputRiteAlign
			}
		}
		return false
	}

	// Rite of Passage alignment choice
	if town.InputMode == engine.InputRiteAlign {
		if ev.Key() == tcell.KeyRune {
			ch := ev.Rune()
			if ch >= 'a' && ch <= 'z' {
				ch -= 32
			}
			c := town.EditChar
			picked := false
			switch ch {
			case 'A':
				if town.RiteAlignGood {
					c.Alignment = engine.Good
					picked = true
				}
			case 'B':
				if town.RiteAlignNeut {
					c.Alignment = engine.Neutral
					picked = true
				}
			case 'C':
				if town.RiteAlignEvil {
					c.Alignment = engine.Evil
					picked = true
				}
			}
			if picked {
				engine.RiteApply(c)
				town.Message = c.Name + " IS NOW A LEGACY!"
				town.InputMode = engine.InputCharEdit
				game.Save()
			}
		}
		return false
	}

	// Character edit mode — from ROLLER segment p-code
	if town.InputMode == engine.InputCharEdit {
		return handleCharEdit(game, ev)
	}

	// Confirm reroll — p-code proc 6: "ARE YOU SURE YOU WANT TO REROLL (Y/N) ?"
	if town.InputMode == engine.InputConfirmReroll {
		if ev.Key() == tcell.KeyRune {
			ch := ev.Rune()
			if ch >= 'a' && ch <= 'z' {
				ch -= 32
			}
			if ch == 'Y' {
				c := town.EditChar
				name := c.Name
				town.Roster.Remove(name)
				town.Creation = engine.NewCreationState()
				town.Creation.Name = name
				town.Creation.Step = engine.StepPassword
				town.Creation.Reroll = true
				game.Phase = engine.PhaseCreation
				town.EditChar = nil
				town.InputMode = engine.InputNone
			} else if ch == 'N' {
				town.InputMode = engine.InputCharEdit
				town.Message = ""
			}
		}
		return false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		town.InputMode = engine.InputNone
		town.InputBuf = ""
		town.Message = ""
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(town.InputBuf) > 0 {
			town.InputBuf = town.InputBuf[:len(town.InputBuf)-1]
		}
	case tcell.KeyEnter:
		town.InputMode = engine.InputNone
		town.InputBuf = ""
	case tcell.KeyRune:
		ch := ev.Rune()
		if len(town.InputBuf) < 15 {
			if ch >= 'a' && ch <= 'z' {
				ch -= 32
			}
			if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == ' ' || ch == '-' || ch == '\'' {
				town.InputBuf += string(ch)
			}
		}
	}
	return false
}

// handleTempleNameInput handles text input at "WHO ARE YOU HELPING ? >".
// From SHOPS segment p-code proc 28 (IC 76-436):
//   Read name, search roster, check status, display result.
func handleTempleNameInput(game *engine.GameState, ev *tcell.EventKey) bool {
	town := game.Town

	switch ev.Key() {
	case tcell.KeyEscape:
		// ESC exits temple
		town.InputMode = engine.InputNone
		town.InputBuf = ""
		app.GoToCastle(town)
		return false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(town.InputBuf) > 0 {
			town.InputBuf = town.InputBuf[:len(town.InputBuf)-1]
		}
	case tcell.KeyEnter:
		name := strings.TrimSpace(town.InputBuf)
		if name == "" {
			// Empty name = exit temple (from p-code: empty string → EXIT)
			town.InputMode = engine.InputNone
			town.InputBuf = ""
			app.GoToCastle(town)
			return false
		}

		// Search roster for character by name (case-insensitive)
		// From p-code proc 28 (IC 205-299): loop through roster comparing names
		var found *engine.Character
		for _, c := range town.Roster.Characters {
			if c == nil {
				continue
			}
			if strings.EqualFold(c.Name, name) {
				found = c
				break
			}
		}

		if found == nil {
			// Character not found — silent return to name prompt
			// (p-code just loops back to "WHO ARE YOU HELPING")
			town.InputBuf = ""
			town.Message = ""
			return false
		}

		// Check if character is lost — from p-code proc 28 (IC 374-396)
		if found.Status == engine.Lost {
			town.Message = found.Name + " IS LOST"
			town.InputBuf = ""
			return false
		}

		// Check if character is OK — from p-code proc 28 (IC 397-416)
		if found.Status == engine.OK {
			town.Message = found.Name + " IS OK"
			town.InputBuf = ""
			return false
		}

		// Character needs healing — check if they need temple services
		// Statuses that temple can handle: Paralyzed, Stoned, Dead, Ashed
		cost := engine.TempleDonation(found.Status, found.Level)
		if cost == 0 {
			// Asleep/Afraid — not handled by temple (will be OK after resting)
			town.Message = found.Name + " IS OK"
			town.InputBuf = ""
			return false
		}

		// Set up donation display
		town.TempleChar = found
		town.TempleCost = cost
		town.TempleStep = engine.TempleTithe
		town.InputMode = engine.InputNone
		town.InputBuf = ""
		town.Message = ""
	case tcell.KeyRune:
		ch := ev.Rune()
		if len(town.InputBuf) < 15 {
			town.InputBuf += string(ch)
		}
	}
	return false
}

// handleTempleInput handles key presses at the temple based on current TempleStep.
// From SHOPS segment p-code procs 27/23/25.
func handleTempleInput(game *engine.GameState, town *engine.TownState, ch rune) {
	switch town.TempleStep {
	case engine.TempleWho:
		// Text input mode — handled by handleTempleNameInput via InputTempleHelp
		// If we get here, the user pressed a key while in TempleWho without InputMode set
		town.InputMode = engine.InputTempleHelp
		town.InputBuf = string(ch)

	case engine.TempleTithe:
		// "WHO WILL TITHE" — select party member (1-6)
		// From p-code proc 27 (IC 561-597): WIZARDRY.proc15 reads party member selection
		if ch >= '1' && ch <= '6' {
			idx := int(ch-'0') - 1
			if idx < len(town.Party.Members) && town.Party.Members[idx] != nil {
				payer := town.Party.Members[idx]
				// Check if payer can afford the donation
				// From p-code proc 27 (IC 599-647): WIZARDRY.proc11 subtracts gold,
				// if result > 0 (not enough): "CHEAP APOSTATES! OUT!"
				if payer.Gold < town.TempleCost {
					town.TempleStep = engine.TempleResult
					town.TempleMessages = []string{"CHEAP APOSTATES! OUT!"}
					town.Message = ""
					return
				}

				// Deduct gold from payer
				payer.Gold -= town.TempleCost

				// Advance to ritual display
				// From p-code proc 23 (IC 796-1052): MURMUR-CHANT-PRAY-INVOKE
				town.TempleStep = engine.TempleRitual
				town.Message = ""
			}
		}

	case engine.TempleRitual:
		// During ritual display, any keypress performs the heal and shows result
		templePerformHeal(game, town)
	}
}

// templePerformHeal executes the healing attempt and transitions to result.
func templePerformHeal(game *engine.GameState, town *engine.TownState) {
	msgs, _ := engine.TempleHeal(town.TempleChar)
	town.TempleMessages = msgs
	town.TempleStep = engine.TempleResult
	town.Message = ""
	// Auto-save after healing (original writes character back to disk)
	if game != nil {
		_ = game.Save()
	}
}

// handleTempleReturn handles RETURN/ESC at the temple.
func handleTempleReturn(game *engine.GameState, town *engine.TownState) {
	switch town.TempleStep {
	case engine.TempleWho:
		// RETURN at name prompt without InputMode — exit temple
		app.GoToCastle(town)
	case engine.TempleTithe:
		// RETURN at tithe selection — back to name prompt
		town.TempleStep = engine.TempleWho
		town.TempleChar = nil
		town.TempleCost = 0
		town.InputMode = engine.InputTempleHelp
		town.InputBuf = ""
		town.Message = ""
	case engine.TempleRitual:
		// RETURN during ritual — perform heal and show result
		templePerformHeal(game, town)
	case engine.TempleResult:
		// RETURN at result — back to name prompt
		town.TempleStep = engine.TempleWho
		town.TempleChar = nil
		town.TempleCost = 0
		town.TempleMessages = nil
		town.InputMode = engine.InputTempleHelp
		town.InputBuf = ""
		town.Message = ""
	}
}

// handleSetPassword handles the two-step password change at Training Grounds.
// From p-code ROLLER proc 3 (IC 4480-4816):
//   Step 0: "ENTER NEW PASSWORD ([RET] FOR NONE)" → type password
//   Step 1: "ENTER AGAIN TO BE SURE" → confirm; match → save, mismatch → "NOT THE SAME"
func handleSetPassword(game *engine.GameState, ev *tcell.EventKey) bool {
	town := game.Town
	switch ev.Key() {
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(town.InputBuf) > 0 {
			town.InputBuf = town.InputBuf[:len(town.InputBuf)-1]
		}
	case tcell.KeyEnter:
		if town.PasswordStep == 0 {
			if len(town.InputBuf) > 15 {
				town.InputBuf = town.InputBuf[:15]
			}
			town.PasswordFirst = town.InputBuf
			town.PasswordStep = 1
			town.InputBuf = ""
		} else {
			if town.InputBuf == town.PasswordFirst {
				town.EditChar.Password = town.PasswordFirst
				town.Message = "PASSWORD CHANGED - "
			} else {
				town.Message = "THEY ARE NOT THE SAME - YOUR PASSWORD HAS NOT BEEN CHANGED!"
			}
			town.InputMode = engine.InputCharEdit
			town.InputBuf = ""
			town.PasswordStep = 0
			town.PasswordFirst = ""
		}
	case tcell.KeyEscape:
		town.InputMode = engine.InputCharEdit
		town.InputBuf = ""
		town.PasswordStep = 0
		town.PasswordFirst = ""
	case tcell.KeyRune:
		ch := ev.Rune()
		if ch >= 'a' && ch <= 'z' {
			ch -= 32
		}
		if len(town.InputBuf) < 15 {
			town.InputBuf += string(ch)
		}
	}
	return false
}

// handleClassChange handles the class change selection screen.
// From Pascal ROLLER.TEXT CHGCLASS (lines 599-638):
//   REPEAT/UNTIL loop accepts A-H or RET. Validates CHG2LST[class] and class != current.
//   On valid selection: reset stats to race base, set new class, level=1, XP=0,
//   age character, grant starting spell, zero all spell slots, unequip non-cursed items.
func handleClassChange(game *engine.GameState, ev *tcell.EventKey) bool {
	town := game.Town
	c := town.EditChar
	if c == nil {
		town.InputMode = engine.InputTrainingName
		return false
	}

	// RET cancels — Pascal line 606-607: EXIT(CHGCLASS)
	if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyEscape {
		town.InputMode = engine.InputCharEdit
		town.Message = ""
		return false
	}

	if ev.Key() != tcell.KeyRune {
		return false
	}

	ch := ev.Rune()
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}

	// Pascal lines 604-605: accept 'A' through 'H'
	if ch < 'A' || ch > 'H' {
		return false
	}

	// Pascal lines 608-613: convert letter to class index
	// A=Fighter(0), B=Mage(1), ..., H=Ninja(7)
	classIdx := engine.Class(ch - 'A')

	// Pascal line 614: UNTIL CHG2LST[CLASSX] AND NOT(CLASSX = CHARREC.CLASS)
	if !town.ClassChangeList[classIdx] || classIdx == c.Class {
		return false
	}

	// Pascal lines 616-618: SETBASE — reset stats to race base values
	base := engine.RaceBaseStats[c.Race]
	c.Strength = base[0]
	c.IQ = base[1]
	c.Piety = base[2]
	c.Vitality = base[3]
	c.Agility = base[4]
	c.Luck = base[5]

	// Pascal line 619: set new class
	c.Class = classIdx
	// Pascal line 620: level = 1
	c.Level = 1
	// Pascal lines 621-623: XP = 0
	c.XP = 0
	// Pascal line 624: AGE + 52*(RANDOM MOD 3) + 252
	// (ages character ~5-7 years — 252 weeks = ~4.8 years base + 0-2 more years)
	c.Age += 52*(rand.Intn(3)) + 252

	// Pascal lines 625-628: grant starting spell
	// SPELLSKN[3] = KATINO for Mage, SPELLSKN[23] = DIOS for Priest
	if classIdx == engine.Mage {
		c.SpellKnown[engine.SpellIndex["KATINO"]] = true
	} else if classIdx == engine.Priest {
		c.SpellKnown[engine.SpellIndex["DIOS"]] = true
	}

	// Pascal lines 629-633: zero all spell slots
	for i := 0; i < 7; i++ {
		c.MageSpells[i] = 0
		c.PriestSpells[i] = 0
		c.MaxMageSpells[i] = 0
		c.MaxPriestSpells[i] = 0
	}

	// Pascal lines 634-636: unequip all non-cursed items
	for i := 0; i < c.ItemCount; i++ {
		if c.Items[i].Equipped && !c.Items[i].Cursed {
			c.Items[i].Equipped = false
		}
	}

	// Recalculate AC (only cursed equipped items remain)
	c.RecalcAC(game.Scenario.Items)

	// Recalculate spell slots for new class (SetSpells uses class+level)
	engine.SetSpells(c)

	// Copy current slots to match max (fully rested after class change)
	for i := 0; i < 7; i++ {
		c.MageSpells[i] = c.MaxMageSpells[i]
		c.PriestSpells[i] = c.MaxPriestSpells[i]
	}

	// Pascal line 637: PUTCHARC — save character
	game.Save()

	town.InputMode = engine.InputCharEdit
	town.Message = ""
	return false
}

func handleCharEdit(game *engine.GameState, ev *tcell.EventKey) bool {
	town := game.Town
	c := town.EditChar
	if c == nil {
		town.InputMode = engine.InputTrainingName
		town.InputBuf = ""
		return false
	}

	if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyEscape {
		app.New(game).LeaveCharacterEdit()
		return false
	}

	if ev.Key() != tcell.KeyRune {
		return false
	}

	ch := ev.Rune()
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}

	switch ch {
	case 'I': // Inspect — enter full character sheet
		town.InputMode = engine.InputInspect
		town.Message = ""
		return false
	case 'D': // Delete — confirm first (p-code proc 5: "ARE YOU SURE YOU WANT TO DELETE (Y/N) ?")
		app.New(game).BeginDeleteCharacter()
	case 'R':
		if game.Scenario.ScenarioNum == 3 {
			// Rite of Passage — Wiz 3 only. From Pascal ROLLER.TEXT RITEPASS.
			msg := engine.RiteCanPerform(c)
			if msg != "" {
				town.Message = msg
			} else {
				town.InputMode = engine.InputRiteCeremony
				town.Message = ""
			}
		} else {
			// Reroll — confirm first (p-code proc 6: "ARE YOU SURE YOU WANT TO REROLL (Y/N) ?")
			town.InputMode = engine.InputConfirmReroll
			town.Message = ""
		}
	case 'C': // Change class — from Pascal ROLLER.TEXT CHGCLASS (lines 574-639)
		avail := engine.CharClassQualifies(c)
		// Exclude current class (Pascal line 590: NOT (CLASSX = CHARREC.CLASS))
		avail[c.Class] = false
		hasAny := false
		for _, v := range avail {
			if v {
				hasAny = true
				break
			}
		}
		if !hasAny {
			town.Message = "NO CLASSES AVAILABLE"
		} else {
			town.ClassChangeList = avail
			town.InputMode = engine.InputClassChange
			town.Message = ""
		}
	case 'S': // Set new password — from p-code ROLLER proc 3 (IC 4480-4816)
		town.InputMode = engine.InputSetPassword
		town.InputBuf = ""
		town.Message = ""
	}
	return false
}

// handleCombatInput processes keyboard input during combat phase.
// Combat flow traced from p-code COMBAT (seg 4), CUTIL (seg 6).
func handleCombatInput(game *engine.GameState, ev *tcell.EventKey) {
	combat := game.Combat
	if combat == nil {
		game.Phase = engine.PhaseMaze
		return
	}

	switch combat.Phase {
	case engine.CombatInit:
		// Timed auto-advance — no keypress needed.
		// The ticker handles the transition after ~1.5s.
		return

	case engine.CombatFriendly:
		// From p-code CINIT proc 1: F)ight or L)eave
		if ev.Key() == tcell.KeyRune {
			ch := strings.ToUpper(string(ev.Rune()))
			if ch == "F" {
				// Fight — transition to normal combat
				app.New(game).FightFriendly()
			} else if ch == "L" {
				// Leave in peace — exit combat
				app.New(game).LeaveInPeace()
			}
		}

	case engine.CombatChoose:
		handleCombatChoose(game, ev)

	case engine.CombatConfirm:
		if ev.Key() == tcell.KeyEnter {
			// Execute the round (once) and start playback
			app.New(game).Commit()
		}
		if ev.Key() == tcell.KeyRune {
			ch := ev.Rune()
			if ch == 'b' || ch == 'B' {
				// Go back to redo options (clears every order)
				app.New(game).ReturnToOrders()
			}
		}

	case engine.CombatExecute:
		// HAMAN/MAHAMAN interactive boon selection (Wiz 2/3)
		if combat.HamanSelecting {
			if ev.Key() == tcell.KeyRune {
				ch := ev.Rune()
				if ch >= '1' && ch <= '3' {
					app.New(game).ChooseBoon(int(ch - '1'))
				}
			}
			return
		}
		// Original game: combat messages auto-advance on timer only (PAUSE1).
		// No keypress skips or speeds them up. Timer handler at combatTicker does the advancing.

	case engine.CombatChest:
		handleChestInput(game, combat, ev)

	case engine.CombatChestResult:
		// Keypress also advances (speed through)
		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyRune {
			advanceChestMessages(game)
		}

	case engine.CombatVictory:
		// Auto-advances via ticker (PAUSE2 from Pascal source, not keypress).
		// Keypress also advances immediately.
		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyRune {
			app.New(game).LeaveCombat() // back to the maze
		}

	case engine.CombatDefeat:
		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyRune {
			app.New(game).LeaveCombat() // back to the castle (party is dead)
		}
	}
}

// advanceChestMessages paces a chest result: the controller shows the next
// block; after the last one, one more PAUSE2 (Pascal), then the controller
// ends the result (rewards, or back to the chest menu).
func advanceChestMessages(game *engine.GameState) {
	combat := game.Combat
	if combat == nil {
		return
	}
	ctrl := app.New(game)
	if done, err := ctrl.AdvanceChest(); err != nil || !done {
		return
	}
	if !combat.ChestPauseUsed {
		combat.ChestPauseUsed = true
		return
	}
	combat.ChestPauseUsed = false
	ctrl.EndChestResult()
}

// handleChestInput handles input during the chest interaction phase.
// From p-code REWARDS proc 16 (IC 3028-3216): O)PEN, C)ALFO, L)EAVE, I)NSPECT, D)ISARM.
func handleChestInput(game *engine.GameState, combat *engine.CombatState, ev *tcell.EventKey) {
	party := game.Town.Party.Members

	// Sub-phases that need a member number (1-6)
	if combat.ChestSubPhase != engine.ChestMenu {
		if ev.Key() == tcell.KeyEscape {
			combat.ChestSubPhase = engine.ChestMenu
			return
		}
		if ev.Key() == tcell.KeyRune {
			ch := ev.Rune()
			if ch >= '1' && ch <= '6' {
				idx := int(ch-'0') - 1
				if idx < len(party) && party[idx] != nil {
					// The controller checks the member can act (app/chest.go).
					ctrl := app.New(game)
					switch combat.ChestSubPhase {
					case engine.ChestWhoOpen:
						ctrl.OpenChest(party[idx])
					case engine.ChestWhoCalfo:
						ctrl.CastCalfo(party[idx])
					case engine.ChestWhoInspect:
						ctrl.InspectChest(party[idx])
					case engine.ChestWhoDisarm:
						ctrl.DisarmChest(party[idx])
					}
				}
			}
		}
		return
	}

	// Main chest menu: O/C/L/I/D
	if ev.Key() != tcell.KeyRune {
		return
	}
	ch := ev.Rune()
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}

	switch ch {
	case 'O':
		combat.ChestSubPhase = engine.ChestWhoOpen
	case 'C':
		combat.ChestSubPhase = engine.ChestWhoCalfo
	case 'L':
		app.New(game).LeaveChest()
	case 'I':
		combat.ChestSubPhase = engine.ChestWhoInspect
	case 'D':
		combat.ChestSubPhase = engine.ChestWhoDisarm
	}
}

// handleCombatChoose handles input during action selection for each party member.
// From p-code CUTIL segment: F)IGHT, S)PELL, P)ARRY, R)UN, U)SE, D)ISPELL
func handleCombatChoose(game *engine.GameState, ev *tcell.EventKey) {
	combat := game.Combat
	party := game.Town.Party.Members

	ctrl := app.New(game)
	if combat.CurrentActor >= len(party) {
		ctrl.FinishOrders()
		return
	}

	member := party[combat.CurrentActor]
	if !app.CanGiveOrder(member) {
		combat.CurrentActor = findNextActor(game, combat.CurrentActor)
		if combat.CurrentActor >= len(party) {
			ctrl.FinishOrders()
		}
		return
	}

	// USE item selection — pick item by number (1-8)
	if combat.SelectingUseItem {
		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyEscape {
			combat.SelectingUseItem = false
			return
		}
		idx := int(ev.Rune()) - '1'
		if idx >= 0 && idx < len(combat.UsableItems) {
			combat.SelectingUseItem = false
			// The controller aims the item's spell (group spells at the
			// first living group, others at the user).
			if ctrl.SetOrder(member, app.Order{Action: engine.ActionUse, Item: combat.UsableItems[idx]}) == nil {
				advanceActor(game)
			}
		}
		return
	}

	// Spell input mode
	if combat.InputtingSpell {
		handleSpellInput(game, ev)
		return
	}

	// Spell group selection — "CAST SPELL ON GROUP #?"
	if combat.SelectingSpellGroup {
		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyEscape {
			combat.SelectingSpellGroup = false
			combat.PendingSpellName = ""
			return
		}
		if ev.Key() == tcell.KeyRune {
			digit := ev.Rune()
			if digit >= '1' && digit <= '4' {
				groupIdx := int(digit - '1')
				if ctrl.SetOrder(member, app.Order{Action: engine.ActionSpell, Spell: combat.PendingSpellName, Group: groupIdx}) == nil {
					combat.SelectingSpellGroup = false
					combat.PendingSpellName = ""
					advanceActor(game)
				}
			}
		}
		return
	}

	// Spell person selection — " CAST SPELL ON PERSON # ?"
	if combat.SelectingSpellTarget {
		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyEscape {
			combat.SelectingSpellTarget = false
			combat.PendingSpellName = ""
			return
		}
		if ev.Key() == tcell.KeyRune {
			digit := ev.Rune()
			if digit >= '1' && digit <= '6' {
				idx := int(digit - '1')
				if idx < len(party) && party[idx] != nil &&
					ctrl.SetOrder(member, app.Order{Action: engine.ActionSpell, Spell: combat.PendingSpellName, Ally: party[idx]}) == nil {
					combat.SelectingSpellTarget = false
					combat.PendingSpellName = ""
					advanceActor(game)
				}
			}
		}
		return
	}

	// Group selection mode — from p-code CUTIL IC 55-200
	// Accept '1'-'4' for group number, RETURN to cancel
	if combat.SelectingGroup {
		if ev.Key() == tcell.KeyEnter {
			// Cancel — return to options (set -999 = unset, like p-code)
			combat.SelectingGroup = false
			return
		}
		if ev.Key() == tcell.KeyRune {
			digit := ev.Rune()
			if digit >= '1' && digit <= '4' {
				groupIdx := int(digit-'1')
				if ctrl.SetOrder(member, app.Order{Action: combat.GroupAction, Group: groupIdx}) == nil {
					combat.SelectingGroup = false
					advanceActor(game)
				}
				// Invalid group — ignore, wait for valid input
			}
		}
		return
	}

	if ev.Key() != tcell.KeyRune {
		return
	}

	ch := ev.Rune()
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}

	// Which orders are available, and why not, comes from the controller
	// (app/combat.go); this switch only runs the prompts.
	menu, err := ctrl.CombatOrderMenu(member)
	if err != nil {
		return
	}
	available := func(a engine.CombatAction) bool {
		for _, o := range menu {
			if o.Action == a {
				return o.Available
			}
		}
		return false
	}
	// askGroup gives a group order at once when one group is alive, or asks
	// which group (from p-code CUTIL IC 0-200).
	askGroup := func(a engine.CombatAction, prompt string) {
		if combat.AliveGroupCount() > 1 {
			combat.SelectingGroup = true
			combat.GroupAction = a
			combat.GroupPrompt = prompt
		} else if ctrl.SetOrder(member, app.Order{Action: a, Group: combat.FirstAliveGroup()}) == nil {
			advanceActor(game)
		}
	}
	order := func(a engine.CombatAction) {
		if ctrl.SetOrder(member, app.Order{Action: a, Item: -1}) == nil {
			advanceActor(game)
		}
	}

	switch ch {
	case 'F': // Fight — only the front row (positions 1-3) can melee
		if available(engine.ActionFight) {
			askGroup(engine.ActionFight, "FIGHT AGAINST GROUP# ?")
		}

	case 'S': // Spell
		if available(engine.ActionSpell) {
			combat.InputtingSpell = true
			combat.SpellInput = ""
		}

	case 'P': // Parry
		order(engine.ActionParry)

	case 'R': // Run
		order(engine.ActionRun)

	case 'U': // Use item — Pascal COMBAT2.TEXT lines 51-173
		combat.UsableItems = ctrl.CombatUsableItems(member)
		if len(combat.UsableItems) == 0 {
			order(engine.ActionUse) // nothing usable: recorded as Parry
		} else {
			combat.SelectingUseItem = true
		}

	case 'D': // Dispel undead
		if available(engine.ActionDispel) {
			askGroup(engine.ActionDispel, "DISPELL WHICH GROUP# ?")
		}

	}
}

// handleSpellInput handles typing a spell name during combat.
// From p-code CUTIL: "SPELL NAME ? >" prompt, hash lookup.
func handleSpellInput(game *engine.GameState, ev *tcell.EventKey) {
	combat := game.Combat

	switch ev.Key() {
	case tcell.KeyEscape:
		combat.InputtingSpell = false
		combat.SpellInput = ""
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(combat.SpellInput) > 0 {
			combat.SpellInput = combat.SpellInput[:len(combat.SpellInput)-1]
		}
	case tcell.KeyEnter:
		if len(combat.SpellInput) == 0 {
			combat.InputtingSpell = false
			return
		}
		member := game.Town.Party.Members[combat.CurrentActor]
		ctrl := app.New(game)
		target, err := ctrl.CombatSpellTarget(member, combat.SpellInput)
		var refusal *app.Refusal
		if errors.As(err, &refusal) {
			// "** UNKNOWN SPELL **" or "** CANT CAST **"
			combat.Messages = []string{refusal.Message}
			combat.MessageIndex = 0
			combat.SpellInput = ""
			return
		} else if err != nil {
			return
		}

		combat.InputtingSpell = false

		// Determine target based on spell type — from p-code CUTIL procs 27/28
		switch target {
		case app.TargetGroup:
			// Attack spell targeting a group — ask which group if multiple
			if combat.AliveGroupCount() > 1 {
				combat.SelectingSpellGroup = true
				combat.PendingSpellName = combat.SpellInput
				combat.SpellInput = ""
				return
			}
			// Only 1 group — auto-select
			ctrl.SetOrder(member, app.Order{Action: engine.ActionSpell, Spell: combat.SpellInput, Group: combat.FirstAliveGroup()})
		case app.TargetMember:
			// Heal/buff targeting a person — ask who
			combat.SelectingSpellTarget = true
			combat.PendingSpellName = combat.SpellInput
			combat.SpellInput = ""
			return
		default:
			ctrl.SetOrder(member, app.Order{Action: engine.ActionSpell, Spell: combat.SpellInput})
		}

		combat.SpellInput = ""
		advanceActor(game)

	case tcell.KeyRune:
		ch := ev.Rune()
		if len(combat.SpellInput) < 12 {
			if ch >= 'a' && ch <= 'z' {
				ch -= 32
			}
			if ch >= 'A' && ch <= 'Z' {
				combat.SpellInput += string(ch)
			}
		}
	}
}

// findNextActor returns the index of the next alive party member after `from`.
func findNextActor(game *engine.GameState, from int) int {
	party := game.Town.Party.Members
	for i := from + 1; i < len(party); i++ {
		if app.CanGiveOrder(party[i]) {
			return i
		}
	}
	return len(party) // past end = all done
}

// advanceActor moves to the next party member or to confirm phase.
func advanceActor(game *engine.GameState) {
	combat := game.Combat
	combat.CurrentActor = findNextActor(game, combat.CurrentActor)
	if combat.CurrentActor >= len(game.Town.Party.Members) {
		// Every member who can act has an order: confirmation.
		app.New(game).FinishOrders()
	}
}

// handleUtilInput processes keyboard input during the utilities phase.
func handleUtilInput(game *engine.GameState, ev *tcell.EventKey) {
	util := game.Util
	if util == nil {
		game.Phase = engine.PhaseTitle
		return
	}

	switch util.Step {
	case engine.UtilMenu:
		if ev.Key() == tcell.KeyEscape {
			game.Util = nil
			game.Phase = engine.PhaseTitle
			if game.Title == nil {
				game.Title = &engine.TitleState{Step: engine.TitleMenu}
			} else {
				game.Title.Step = engine.TitleMenu
			}
			return
		}
		if ev.Key() != tcell.KeyRune {
			return
		}
		ch := ev.Rune()
		if ch >= 'a' && ch <= 'z' {
			ch -= 32
		}
		switch ch {
		case 'B':
			util.Step = engine.UtilBackup
		case 'C':
			util.Step = engine.UtilRename
			util.SelectedChar = -1
		case 'I':
			util.Step = engine.UtilImport
			util.InputBuf = ""
			util.Message = ""
		case 'T':
			util.TransferSources = engine.AvailableTransferScenarios(game)
			util.Step = engine.UtilTransfer
		case 'L':
			game.Util = nil
			game.Phase = engine.PhaseTitle
			if game.Title == nil {
				game.Title = &engine.TitleState{Step: engine.TitleMenu}
			} else {
				game.Title.Step = engine.TitleMenu
			}
		}

	case engine.UtilBackup:
		if ev.Key() == tcell.KeyEscape {
			util.Step = engine.UtilMenu
			return
		}
		if ev.Key() != tcell.KeyRune {
			return
		}
		ch := ev.Rune()
		if ch == 't' || ch == 'T' {
			util.Step = engine.UtilBackupTo
			util.InputBuf = ""
			util.Message = ""
		} else if ch == 'f' || ch == 'F' {
			util.Step = engine.UtilBackupFrom
			util.InputBuf = ""
			util.Message = ""
		}

	case engine.UtilBackupTo:
		handleUtilTextInput(util, ev, func(path string) {
			if err := engine.BackupRoster(game, path); err != nil {
				util.Message = fmt.Sprintf("ERROR: %s", err)
			} else {
				util.Message = "BACKUP MADE"
			}
		}, func() { util.Step = engine.UtilBackup })

	case engine.UtilBackupFrom:
		handleUtilTextInput(util, ev, func(path string) {
			if err := engine.RestoreRoster(game, path); err != nil {
				util.Message = fmt.Sprintf("ERROR: %s", err)
			} else {
				util.Message = "BACKUP RECOVERED"
			}
		}, func() { util.Step = engine.UtilBackup })

	case engine.UtilRename:
		if ev.Key() == tcell.KeyEscape {
			util.Step = engine.UtilMenu
			return
		}
		if ev.Key() == tcell.KeyRune {
			ch := ev.Rune()
			idx := int(ch-'1')
			if idx >= 0 && idx < len(game.Town.Roster.Characters) && game.Town.Roster.Characters[idx] != nil {
				util.SelectedChar = idx
				util.Step = engine.UtilRenameNew
				util.InputBuf = ""
				util.Message = ""
			}
		}

	case engine.UtilRenameNew:
		handleUtilTextInput(util, ev, func(newName string) {
			if err := app.New(game).RenameCharacter(util.SelectedChar, newName); err != nil {
				util.Message = strings.ToUpper(err.Error())
				return
			}
			util.Message = "NAME CHANGED!"
			game.Save()
		}, func() { util.Step = engine.UtilRename })

	case engine.UtilImport:
		handleUtilTextInput(util, ev, func(path string) {
			messages, err := engine.ImportCharactersFromDSK(game, path)
			if err != nil {
				util.Message = fmt.Sprintf("ERROR: %s", err)
				return
			}
			util.Messages = messages
			util.Step = engine.UtilImportResult
			game.Save()
		}, func() { util.Step = engine.UtilMenu })

	case engine.UtilImportResult:
		// Any key returns to menu
		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyRune || ev.Key() == tcell.KeyEscape {
			util.Step = engine.UtilMenu
			util.Messages = nil
		}

	case engine.UtilTransfer:
		if ev.Key() == tcell.KeyEscape {
			util.Step = engine.UtilMenu
			return
		}
		if ev.Key() == tcell.KeyRune {
			idx := int(ev.Rune()) - '1'
			if idx >= 0 && idx < len(util.TransferSources) {
				src := util.TransferSources[idx]
				msgs, err := engine.TransferCharacters(game, src)
				if err != nil {
					util.Messages = []string{err.Error()}
				} else {
					util.Messages = msgs
				}
				util.Step = engine.UtilTransferResult
				game.Save()
			}
		}

	case engine.UtilTransferResult:
		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyRune || ev.Key() == tcell.KeyEscape {
			util.Step = engine.UtilMenu
			util.Messages = nil
		}
	}
}

// handleUtilTextInput handles text entry for utilities screens.
func handleUtilTextInput(util *engine.UtilState, ev *tcell.EventKey, onEnter func(string), onEsc func()) {
	switch ev.Key() {
	case tcell.KeyEscape:
		onEsc()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(util.InputBuf) > 0 {
			util.InputBuf = util.InputBuf[:len(util.InputBuf)-1]
		}
	case tcell.KeyEnter:
		input := strings.TrimSpace(util.InputBuf)
		if input == "" {
			onEsc()
			return
		}
		onEnter(input)
	case tcell.KeyRune:
		if len(util.InputBuf) < 60 {
			util.InputBuf += string(ev.Rune())
		}
	}
}

func clamp(val *int, max int) {
	if *val > max {
		*val = max
	}
	if *val < 0 {
		*val = 0
	}
}

func loadScenario(name string) (*data.Scenario, error) {
	switch name {
	case "1":
		return wiz1.Load()
	case "2":
		return wiz2.Load()
	case "3":
		return wiz3.Load()
	default:
		return nil, fmt.Errorf("unknown scenario: %s (use --scenario=1, --scenario=2, or --scenario=3)", name)
	}
}
