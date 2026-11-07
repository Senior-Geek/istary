package main

import (
	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
	"github.com/Senior-Geek/istary/render"
	"github.com/gdamore/tcell/v2"
)

// keyFromTcell translates a terminal key event into a front-end-neutral key.
func keyFromTcell(ev *tcell.EventKey) app.Key {
	switch ev.Key() {
	case tcell.KeyRune:
		return app.Rune(ev.Rune())
	case tcell.KeyEnter:
		return app.Key{Kind: app.KeyEnter}
	case tcell.KeyEscape:
		return app.Key{Kind: app.KeyEscape}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return app.Key{Kind: app.KeyBackspace}
	case tcell.KeyUp:
		return app.Key{Kind: app.KeyUp}
	case tcell.KeyDown:
		return app.Key{Kind: app.KeyDown}
	case tcell.KeyLeft:
		return app.Key{Kind: app.KeyLeft}
	case tcell.KeyRight:
		return app.Key{Kind: app.KeyRight}
	}
	return app.Key{Kind: app.KeyOther}
}

// dispatchKey is the single entry point for terminal key presses, used by
// the main loop and by tests. Migrated workflows go through the controller;
// everything else still uses the legacy tcell handlers. It returns true when
// the player chose to quit.
func dispatchKey(ctrl *app.Controller, screen *render.Screen, ev *tcell.EventKey) bool {
	if ctrl.HandleKey(keyFromTcell(ev)) {
		return false
	}
	return legacyDispatch(screen, ctrl.Game, ev)
}

// legacyDispatch routes keys for phases not yet moved into the controller.
func legacyDispatch(screen *render.Screen, game *engine.GameState, ev *tcell.EventKey) bool {
	switch game.Phase {
	case engine.PhaseTitle:
		handleTitleInput(screen, game, ev)
	case engine.PhaseTown:
		return handleTownInput(game, ev)
	// Camp is handled by app.Controller (camp.go).
	case engine.PhaseCombat:
		handleCombatInput(game, ev)
	case engine.PhaseUtilities:
		handleUtilInput(game, ev)
	}
	return false
}
