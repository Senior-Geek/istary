package app

// The maze: one key press in the dungeon (M6a). This is the terminal's maze
// handler moved here unchanged, so every front end takes a step the same
// way: the move, then the square's effect, then healing and poison, then the
// encounter check, in that order and with the same random numbers.

import (
	"fmt"
	"math/rand"

	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

func (c *Controller) mazeKey(k Key) {
	game := c.Game
	// Map view: arrow keys pan, C re-centers, any other key dismisses
	if game.ShowMap {
		switch k.Kind {
		case KeyUp:
			game.MapScrollY--
			return
		case KeyDown:
			game.MapScrollY++
			return
		case KeyLeft:
			game.MapScrollX--
			return
		case KeyRight:
			game.MapScrollX++
			return
		case KeyRune:
			ch := k.Rune
			if ch == 'c' || ch == 'C' {
				game.MapScrollX = 0
				game.MapScrollY = 0
				return
			}
		}
		// Any other key: dismiss map and reset scroll
		game.ShowMap = false
		game.MapScrollX = 0
		game.MapScrollY = 0
		return
	}

	// T)IME delay input — from RUNNER proc 5 (IC 3936):
	// "NEW DELAY (1-5000) >" — reads numeric string, stores in global 7
	if game.MazeDelayInput {
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch >= '0' && ch <= '9' && len(game.MazeDelayBuf) < 4 {
				game.MazeDelayBuf += string(ch)
			}
		}
		if k.Kind == KeyBackspace {
			if len(game.MazeDelayBuf) > 0 {
				game.MazeDelayBuf = game.MazeDelayBuf[:len(game.MazeDelayBuf)-1]
			}
		}
		if k.Kind == KeyEnter {
			val := 0
			for _, c := range game.MazeDelayBuf {
				val = val*10 + int(c-'0')
			}
			if val > 0 && val <= 5000 {
				game.MazeDelay = val
			}
			game.MazeDelayInput = false
			game.MazeDelayBuf = ""
			game.MazeMessage = ""
			game.MazeMessage2 = ""
		}
		if k.Kind == KeyEscape {
			game.MazeDelayInput = false
			game.MazeDelayBuf = ""
			game.MazeMessage = ""
			game.MazeMessage2 = ""
		}
		return
	}

	// I)NSPECT dungeon screen — from SPECIALS proc 38:
	// P)ICK UP (if characters found) or L)EAVE
	if game.MazeInspecting {
		c.mazeInspectKey(k)
		return
	}

	// Handle SCNMSG pagination — only [RET] advances (matches "[RET] FOR MORE" prompt)
	// All other keys are consumed while messages are showing (prevents key repeat bleed)
	if len(game.MazeMessages) > 0 {
		if k.Kind == KeyEnter {
			if game.MazeMsgWait {
				// More pages to show — advance scroll
				game.MazeMsgScroll += 4
				if game.MazeMsgScroll >= len(game.MazeMessages) {
					game.MazeMessages = nil
					game.MazeMsgScroll = 0
					game.MazeMsgWait = false
				} else {
					game.MazeMsgWait = game.MazeMsgScroll+4 < len(game.MazeMessages)
				}
			} else {
				// Final page shown — Enter dismisses
				game.MazeMessages = nil
				game.MazeMsgScroll = 0
				// If GETYN pending, show the search prompt now
				if game.MazeSearchYN {
					game.MazeMessage = "SEARCH (Y/N) ?"
				}
				// BCK2SHOP (AUX2=8): warp party back to castle after message.
				// Pascal BCK2SHOP: MAZELEV := 0; XGOTO := XNEWMAZE.
				if game.MazePendingBack2Shop {
					game.MazePendingBack2Shop = false
					game.PartyHome()
					game.MazeMessage = ""
					game.Phase = engine.PhaseTown
					game.Town.Location = engine.Castle
					game.Town.InputMode = engine.InputNone
				}
				// TRYGET (AUX2=2): give item to first eligible party member.
				// Pascal GOTITEM (SPECIALS2.TEXT P010315 w/ WC034 fix): iterate party,
				// per char show "ALREADY HAS ONE" / "IS FULL" / "GOT ITEM" and stop on
				// first successful give. EQINDEX stores the raw item index (0-99).
				if game.MazePendingTryGet {
					itemIdx := game.MazeTryGetItem
					game.MazePendingTryGet = false
					game.MazeTryGetItem = 0
					for _, m := range game.Town.Party.Members {
						if m == nil {
							continue
						}
						alreadyHas := false
						for i := 0; i < m.ItemCount; i++ {
							if m.Items[i].ItemIndex == itemIdx {
								alreadyHas = true
								break
							}
						}
						if alreadyHas {
							game.MazeMessage = m.Name + " ALREADY HAS ONE"
							continue
						}
						if m.ItemCount >= 8 {
							game.MazeMessage = m.Name + " IS FULL"
							continue
						}
						m.Items[m.ItemCount] = engine.Possession{
							ItemIndex:  itemIdx,
							Equipped:   false,
							Identified: false,
						}
						m.ItemCount++
						game.MazeMessage = m.Name + " GOT ITEM"
						break
					}
				}
			}
		}
		return // consume ALL keys while messages are showing
	}

	// Handle "SEARCH (Y/N) ?" prompt — Pascal GETYN (SPECIALS2.TEXT P010319)
	// AUX2=4: Y branches on sign of AUX0 — positive triggers combat with AUX0
	// as monster index, negative calls TRYGET with ABS(AUX0) as quest item ID.
	// N exits SPECIALS with no effect.
	if game.MazeSearchYN {
		if k.Kind == KeyRune {
			ch := k.Rune
			if ch == 'y' || ch == 'Y' {
				game.MazeSearchYN = false
				game.MazeMessage = ""
				cell := game.SearchCell
				aux0 := game.SearchMonster
				if cell != nil {
					if aux0 > 0 {
						// Combat with monster AUX0 (e.g., Murphy's Ghost = 77)
						cell.SpclMonster = aux0
						game.MazeMessage = "AN ENCOUNTER"
						game.Combat = engine.NewCombat(game)
						game.Combat.EncounterType = 0
						game.Phase = engine.PhaseCombat
					} else if aux0 < 0 {
						// TRYGET: give ABS(AUX0) as raw item index.
						// SPCMISC pre-processing already applied AUX0 += 1000 for values
						// ≤ -1000, so ABS(aux0) is the raw item index (0-99). Matches
						// Pascal GOTITEM (SPECIALS2.TEXT P010315 w/ WC034 messages).
						itemIdx := -aux0
						for _, m := range game.Town.Party.Members {
							if m == nil {
								continue
							}
							alreadyHas := false
							for i := 0; i < m.ItemCount; i++ {
								if m.Items[i].ItemIndex == itemIdx {
									alreadyHas = true
									break
								}
							}
							if alreadyHas {
								game.MazeMessage = m.Name + " ALREADY HAS ONE"
								continue
							}
							if m.ItemCount >= 8 {
								game.MazeMessage = m.Name + " IS FULL"
								continue
							}
							m.Items[m.ItemCount] = engine.Possession{
								ItemIndex:  itemIdx,
								Equipped:   false,
								Identified: false,
							}
							m.ItemCount++
							game.MazeMessage = m.Name + " GOT ITEM"
							break
						}
					}
				}
				game.SearchCell = nil
			} else if ch == 'n' || ch == 'N' {
				game.MazeSearchYN = false
				game.MazeMessage = ""
				game.SearchCell = nil
			}
		}
		return
	}

	if k.Kind != KeyRune {
		return
	}
	ch := k.Rune
	if ch >= 'a' && ch <= 'z' {
		ch -= 32
	}

	// If waiting for button press — from p-code RUNNER IC 3047
	if game.WaitButton {
		if k.Kind == KeyEnter {
			// Leave buttons alone
			game.WaitButton = false
			game.ButtonCount = 0
			game.MazeMessage = ""
			game.MazeMessage2 = ""
		} else if ch >= 'A' && ch <= 'A'+rune(game.ButtonCount-1) {
			buttonIdx := int(ch - 'A')
			game.WaitButton = false
			game.ButtonCount = 0
			// BUTTONS effect — Pascal RUNNER2.TEXT lines 330-354
			// EXITRUN(MINBUT + button_index): teleport to level, keep X,Y
			cell := game.CurrentCell()
			if cell != nil && len(cell.Aux) > 2 {
				minBut := cell.Aux[2] // AUX2 = MINBUT
				if cell.Aux[0] > 0 {
					// AUX0 > 0: randomize position
					game.PlayerX = rand.Intn(20)
					game.PlayerY = rand.Intn(20)
				}
				newLevel := minBut + buttonIdx // 1-based level
				game.MazeLevel = newLevel - 1  // convert to 0-based
				game.MazeMessage = ""
				game.MazeMessage2 = ""
			}
		}
		return
	}

	// If waiting for Y/N at stairs prompt
	if game.MazeMessage2 == "TAKE THEM (Y/N) ?" {
		if ch == 'Y' {
			cell := game.CurrentCell()
			if cell != nil && cell.Type == data.SqStairs {
				if cell.DestLevel == 0 {
					// Stairs up to castle — Pascal does NOT cure poison on return
					game.PartyHome()
					game.Phase = engine.PhaseTown
					game.Town.Location = engine.Castle
					game.Town.Message = ""
				} else {
					game.MazeLevel = cell.DestLevel - 1 // convert 1-based to 0-based
					game.PlayerX = cell.DestX
					game.PlayerY = cell.DestY
					game.InitFightMap()
				}
			}
			game.MazeMessage = ""
			game.MazeMessage2 = ""
		} else if ch == 'N' {
			game.MazeMessage = ""
			game.MazeMessage2 = ""
		}
		return
	}

	// TRYGET (AUX2=2): single-line message path — give item on first keypress after message.
	// Multi-line path fires in the MazeMessages dismissal handler above.
	if game.MazePendingTryGet {
		eqIdx := game.MazeTryGetItem + 1000
		game.MazePendingTryGet = false
		game.MazeTryGetItem = 0
		for _, m := range game.Town.Party.Members {
			if m == nil || m.ItemCount >= 8 {
				continue
			}
			alreadyHas := false
			for i := 0; i < m.ItemCount; i++ {
				if m.Items[i].ItemIndex == eqIdx {
					alreadyHas = true
					break
				}
			}
			if !alreadyHas {
				m.Items[m.ItemCount] = engine.Possession{
					ItemIndex:  eqIdx,
					Equipped:   false,
					Identified: false,
				}
				m.ItemCount++
				game.MazeMessage = m.Name + " GOT ITEM"
				return // consume key — player must press again to dismiss "GOT ITEM"
			}
		}
	}
	game.MazeMessage = ""
	game.MazeMessage2 = ""
	game.MazeMessages = nil
	game.MazeMsgScroll = 0
	game.MazeMsgWait = false
	game.MazeSearchYN = false
	game.SearchCell = nil
	game.SearchMonster = 0
	game.MazePendingBack2Shop = false
	game.MazePendingTryGet = false
	game.MazeTryGetItem = 0
	game.ViewportMsg = ""
	game.ViewportMsg2 = ""
	moved := false // track whether player actually moved to a new square

	switch ch {
	// Original Wizardry maze controls from p-code RUNNER segment
	case 'F', 'W': // Forward
		if game.MoveForward() {
			moved = true
		} else {
			game.ViewportMsg = "OUCH!"
			c.beep()
		}
	case 'L', 'A': // Turn left
		game.TurnLeft()
	case 'R', 'D': // Turn right
		game.TurnRight()
	case 'K': // Pascal KICK: moves through anything ≠ WALL
		if game.KickDoor() {
			game.PlayerX = (game.PlayerX + game.Facing.DX() + 20) % 20
			game.PlayerY = (game.PlayerY + game.Facing.DY() + 20) % 20
			moved = true
		} else {
			game.ViewportMsg = "OUCH!"
			c.beep()
		}
	case 'C': // Camp — from p-code RUNNER: sets global[10]=12, exits to CAMP segment
		game.Phase = engine.PhaseCamp
	case 'S': // S)TATUS — from p-code RUNNER (CIP 11): redraws party status
		// In the original, this refreshed the party display area.
		// Our renderer redraws every frame, so just clear any stale messages.
		game.MazeMessage = ""
		game.MazeMessage2 = ""
	case 'Q': // Q)UICK — from p-code RUNNER proc 6 (IC 4070): toggle quick plot mode
		game.QuickPlot = !game.QuickPlot
		if game.QuickPlot {
			game.MazeMessage = "QUICK PLOT ON"
		} else {
			game.MazeMessage = "QUICK PLOT OFF"
		}
	case 'T': // T)IME — from p-code RUNNER proc 5 (IC 3936):
		// Prompts "NEW DELAY (1-5000) >" and sets animation delay value.
		// Controls wireframe drawing speed on the Apple II.
		game.MazeMessage = "NEW DELAY (1-5000) >"
		game.MazeMessage2 = ""
		game.MazeDelayInput = true
		game.MazeDelayBuf = ""
	case 'I': // I)NSPECT — from p-code SPECIALS proc 38: search the area
		// Full separate screen: "LOOKING...", then "FOUND:" + search results:
		// the characters lost in the party's area (EXPLROOM, LOOKLOST), by
		// GameState.FindStranded (R-2).
		game.MazeInspecting = true
		game.MazeInspectFound = game.FindStranded()
		game.MazePicking, game.MazePickBuf = false, ""
		game.MazeMessage = ""
	case 'M': // Map view
		game.ShowMap = true
		game.MapScrollX = 0
		game.MapScrollY = 0
	}

	// Pascal DRAWMAZE: light decrements on every screen redraw (turn, kick, forward)
	if game.LightLevel > 0 {
		game.LightLevel--
	}

	// Only check special squares, apply poison, and roll encounters after actual movement
	if moved {
		checkSquare(game)

		// Pascal UPDATEHP (RUNNER2.TEXT lines 398-436):
		// Fires with 25% probability per step: (RANDOM MOD 4) = 2
		// Net change: HPLEFT += HEALPTS - POISNAMT (regeneration offsets poison)
		if rand.Intn(4) == 2 {
			for _, m := range game.Town.Party.Members {
				if m != nil && m.Status == engine.OK {
					healPts := m.GetHealPts(game.Scenario.Items)
					if m.PoisonAmt > 0 || healPts > 0 {
						m.HP += healPts - m.PoisonAmt
						if m.HP <= 0 {
							m.HP = 0
							m.Status = engine.Dead
						} else if m.HP > m.MaxHP {
							m.HP = m.MaxHP
						}
					}
				}
			}
		}

		// R-1 (decision D-C): a step after which no one can go on (a pit, a
		// chute, an OUCH square, SOLID ROCK, poison) leaves the party
		// stranded on the square it ended on, after the chute or teleporter
		// that moved it. Nothing more happens: no encounter.
		if game.Phase == engine.PhaseMaze && !engine.PartyCanContinue(game.Town.Party.Members) {
			game.StrandParty()
			return
		}

		// Pascal RUNMAIN (RUNNER2.TEXT lines 616-624): encounter triggers
		// 1. RANDOM MOD 99 = 35 (1/99 chance)
		// 2. CHSTALRM = 1 (alarm trap)
		// 3. FIGHTMAP[x][y] = true (fight zone)
		if game.Phase == engine.PhaseMaze {
			triggerEncounter := rand.Intn(99) == 35
			if game.ChestAlarm == 1 {
				triggerEncounter = true
			}
			if game.FightMap[game.PlayerX][game.PlayerY] {
				triggerEncounter = true
			}
			if triggerEncounter {
				combat := engine.NewCombat(game)
				// Pascal ENCOUNTR (RUNNER2.TEXT lines 134-143): set ATTK012
				level := &game.Scenario.Mazes.Levels[game.MazeLevel]
				cell := level.Cells[game.PlayerY][game.PlayerX]
				if game.ChestAlarm == 1 {
					combat.EncounterType = 2
					combat.Surprised = 2 // alarm: monsters surprise party
					game.ChestAlarm = 0
				} else if cell.Encounter {
					// Fight-zone square: check FIGHTMAP
					if game.FightMap[game.PlayerX][game.PlayerY] {
						combat.EncounterType = 2 // first encounter here
					} else {
						combat.EncounterType = 1 // already cleared
					}
				} else {
					combat.EncounterType = 0 // random encounter
				}
				game.Combat = combat
				game.Phase = engine.PhaseCombat
				// Clear fight room after combat triggers
				game.ClearFightRoom(game.PlayerX, game.PlayerY)
			}
		}
	}

	return
}

func checkSquare(game *engine.GameState) {
	cell := game.CurrentCell()
	if cell == nil {
		return
	}
	// Pascal SPECSQAR: FIZZLES := 0 at start, set to 1 only for fizzle squares
	game.Fizzles = 0
	// Square effects from RUNNER segment p-code
	switch cell.Type {
	case data.SqStairs:
		if cell.DestLevel < game.MazeLevel+1 {
			game.MazeMessage = "STAIRS GOING UP."
		} else {
			game.MazeMessage = "STAIRS GOING DOWN."
		}
		game.MazeMessage2 = "TAKE THEM (Y/N) ?"
	case data.SqDark:
		// Pascal VERYDARK (RUNNER2.TEXT lines 214-221):
		// GOTOXY(2,5) "IT'S VERY" / GOTOXY(2,6) "DARK HERE"
		game.ViewportMsg = "IT'S VERY"
		game.ViewportMsg2 = "DARK HERE"
		game.LightLevel = 0
	case data.SqChute:
		// From p-code proc 22 (IC 2206): "A CHUTE!" + damage + move down
		game.MazeMessage = "A CHUTE!"
		hazardDamage(game)
		if cell.DestLevel > 0 {
			game.MazeLevel = cell.DestLevel - 1
			game.PlayerX = cell.DestX
			game.PlayerY = cell.DestY
			game.InitFightMap()
		}
	case data.SqPit:
		// From p-code proc 18 (IC 2643): "A PIT!" + damage
		game.MazeMessage = "A PIT!"
		hazardDamage(game)
	case data.SqEncounter, data.SqEncounter2:
		game.Combat = engine.NewCombat(game)
		game.Combat.EncounterType = 2 // fixed encounter → Reward2 (chest)
		game.Phase = engine.PhaseCombat
		return
	case data.SqButtons:
		// From p-code RUNNER IC 2967-3047
		game.MazeMessage = "THERE ARE BUTTONS ON THE WALL"
		if len(cell.Aux) > 1 && cell.Aux[1] > 0 {
			lastBtn := 'A' + rune(cell.Aux[1]-1)
			game.MazeMessage2 = fmt.Sprintf("MARKED A-%c. PRESS ONE OR [RET]", lastBtn)
			game.ButtonCount = cell.Aux[1]
			game.WaitButton = true
		}
	case data.SqSpclEnctr:
		// Pascal CHENCOUN (RUNNER2.TEXT lines 290-319) + SPCMISC (SPECIALS2.TEXT):
		// If FIGHTMAP set and AUX0 != 0: trigger encounter (CHENCOUN).
		// Otherwise: fall through to SPCMISC message/event handler using AUX2.
		if cell.Count == 0 {
			break
		}
		if cell.Encounter && game.FightMap[game.PlayerX][game.PlayerY] {
			// CHENCOUN: trigger combat encounter
			game.MazeMessage = "AN ENCOUNTER"
			cell.SpclMonster = cell.Count
			game.Combat = engine.NewCombat(game)
			game.Combat.EncounterType = 2
			game.Phase = engine.PhaseCombat
			return
		}
		// SPCMISC fallthrough: handle via AUX2 dispatch (messages, events)
		// Pascal SPECIALS2.TEXT lines 657-722
		aux2 := cell.Aux2
		if aux2 == 0 {
			break
		}
		// AUX2=1: message with counter depletion
		// AUX2=8: back-to-shop with counter depletion
		// AUX2=4: GETYN — see AUX0 transform below (Pascal SPCMISC)
		// Pascal SPECIALS2.TEXT lines 626-645: AUX0 transforms only apply when AUX2 in {1,4,8}
		aux0 := cell.Count
		if aux2 == 1 || aux2 == 8 {
			if cell.Count > 0 {
				cell.Count--
				if cell.Count == 0 {
					cell.Type = data.SqNormal // convert to normal when depleted
				}
			}
		} else if aux2 == 4 {
			// Pascal: IF AUX0 < 0 THEN
			//   IF AUX0 > -1000 THEN MAZEFLOR.AUX0 := 0  (one-shot trigger consumed)
			//   ELSE AUX0 := AUX0 + 1000                  (quest-item encoding)
			if aux0 < 0 {
				if aux0 > -1000 {
					cell.Count = 0
				} else {
					aux0 = aux0 + 1000
				}
			}
		}
		// Pascal: IF NOT((AUX2 > 12) OR (AUX2 = 5) OR (AUX2 = 6)) THEN DOMSG(AUX1, ...)
		// AUX2=5 (ITM2PASS) and AUX2=6 (CHKALIGN) skip the main message; any message
		// comes from within BOUNCEBK after the item/align check fails.
		showMsg := aux2 != 5 && aux2 != 6 && aux2 <= 12
		if showMsg {
			// Display message from AUX1 line index (Pascal DOMSG)
			block := game.Scenario.MessageBlock(cell.Aux1)
			if block != nil {
				if len(block) == 1 {
					game.MazeMessage = block[0]
				} else if len(block) > 1 {
					game.MazeMessages = block
					game.MazeMsgScroll = 0
					game.MazeMsgWait = len(block) > 4
				}
			}
		}
		switch aux2 {
		case 2:
			// TRYGET (SPECIALS2.TEXT P010316): after message, give item AUX0 to first
			// party member. Pascal GOTITEM stores EQINDEX := ITEMX directly.
			game.MazePendingTryGet = true
			game.MazeTryGetItem = aux0
		case 4:
			// GETYN (SPECIALS2.TEXT P010319): after message, show "SEARCH (Y/N) ?"
			// On Y: positive AUX0 → combat with monster AUX0; negative AUX0 → TRYGET
			// with ABS(AUX0) as item index. aux0 already has the -1000 transform applied.
			game.MazeSearchYN = true
			game.SearchCell = cell
			game.SearchMonster = aux0
		case 5:
			// ITM2PASS (SPECIALS2.TEXT P01031B): pass if party holds item AUX0, else BOUNCEBK.
			// Pascal checks POSS.EQINDEX == AUX0 directly — item indices are raw (0-99).
			itemTarget := cell.Count
			partyHasItem := false
			for _, m := range game.Town.Party.Members {
				if m == nil {
					continue
				}
				for i := 0; i < m.ItemCount; i++ {
					if m.Items[i].ItemIndex == itemTarget {
						partyHasItem = true
					}
				}
			}
			if !partyHasItem {
				// BOUNCEBK: reverse last movement direction, then show message at AUX1.
				// Pascal BOUNCEBK: CASE DIRECTIO OF 0: MAZEY-=1; 1: MAZEX-=1; ...
				rev := game.Facing.Reverse()
				game.PlayerX = ((game.PlayerX+rev.DX())%20 + 20) % 20
				game.PlayerY = ((game.PlayerY+rev.DY())%20 + 20) % 20
				// BOUNCEBK shows DOMSG(AUX1, FALSE) if AUX1 >= 0.
				if cell.Aux1 >= 0 {
					block := game.Scenario.MessageBlock(cell.Aux1)
					if block != nil {
						if len(block) == 1 {
							game.MazeMessage = block[0]
						} else if len(block) > 1 {
							game.MazeMessages = block
							game.MazeMsgScroll = 0
							game.MazeMsgWait = len(block) > 4
						}
					}
				}
			}
		case 8:
			// BCK2SHOP: after message, warp party back to castle.
			// Pascal BCK2SHOP: MAZELEV := 0; XGOTO := XNEWMAZE (SPECIALS2 line 353-358).
			game.MazePendingBack2Shop = true
		case 9:
			// LOOKOUT (SPECIALS2.TEXT P01032A): fill FIGHTMAP in radius AUX0 around player.
			// For SqSpclEnctr, AUX0 = cell.Count (radius).
			radius := cell.Count
			for x2 := -radius; x2 <= radius; x2++ {
				for y2 := -radius; y2 <= radius; y2++ {
					fx := ((game.PlayerX+x2)%20 + 20) % 20
					fy := ((game.PlayerY+y2)%20 + 20) % 20
					game.FightMap[fx][fy] = true
				}
			}
			game.FightMap[game.PlayerX][game.PlayerY] = false
		}
	case data.SqScnMsg:
		// Pascal SPCMISC (SPECIALS2.TEXT): dispatches on AUX2 sub-type
		aux2 := 0
		if len(cell.Aux) > 2 {
			aux2 = cell.Aux[2]
		}
		switch aux2 {
		case 9:
			// LOOKOUT (SPECIALS2.TEXT P010324): set FIGHTMAP in square radius
			// AUX0 = radius; excludes center square
			radius := 0
			if len(cell.Aux) > 0 {
				radius = cell.Aux[0]
			}
			for x2 := -radius; x2 <= radius; x2++ {
				for y2 := -radius; y2 <= radius; y2++ {
					fx := ((game.PlayerX+x2)%20 + 20) % 20
					fy := ((game.PlayerY+y2)%20 + 20) % 20
					game.FightMap[fx][fy] = true
				}
			}
			game.FightMap[game.PlayerX][game.PlayerY] = false
		default:
			// Message display — Pascal DOMSG: AUX1 is a starting LINE number
			msgLine := cell.MsgIndex
			if len(cell.Aux) > 1 && cell.Aux[1] > 0 {
				msgLine = cell.Aux[1]
			}
			block := game.Scenario.MessageBlock(msgLine)
			if block != nil {
				if len(block) == 1 {
					game.MazeMessage = block[0]
				} else if len(block) > 1 {
					game.MazeMessages = block
					game.MazeMsgScroll = 0
					game.MazeMsgWait = len(block) > 4 // need pagination if > 4 lines
				}
			}
		}
	case data.SqFizzle:
		game.Fizzles = 1
		game.MazeMessage = "SPELLS FIZZLE HERE"
	case data.SqTransfer:
		// From p-code proc 23 (IC 3229): teleport to destination
		game.MazeMessage = "TELEPORTER"
		if cell.DestLevel > 0 {
			game.MazeLevel = cell.DestLevel - 1
			game.PlayerX = cell.DestX
			game.PlayerY = cell.DestY
			game.InitFightMap()
		}
	case data.SqOuchy:
		// From p-code proc 17 (IC 2669): "OUCH!" + damage
		game.MazeMessage = "OUCH!"
		ouchyDamage(game, cell)
	case data.SqRockwater:
		// Istary's SOLID ROCK square: everyone still able to act dies (kept
		// for now, decision Q3 of R-2). This is not the original's rule. In
		// the reconstructed Pascal, SPECSQAR ROCKWATE sets MAZELEV := -99 and
		// goes to NEWMAZE, which turns any level below 0 into level 1 and
		// goes to the entrance's Camp (XEQUIP6): the party is put back on
		// the first level, at the same X and Y, in Camp, unharmed (RUNNER2,
		// UTILITIE; docs/design/r2/RESCUE.md). No scenario of Wizardry I-III
		// has such a square. Bodies left here are found, and picked up, from
		// any square joined to it by open walls (rescue.go).
		game.MazeMessage = "SOLID ROCK!"
		for _, m := range game.Town.Party.Members {
			if m != nil && m.IsAlive() {
				m.HP = 0
				m.Status = engine.Dead
			}
		}
		// The party is now stranded on this square, as after any step that
		// leaves no one able to go on (mazeKey, R-1); it is no longer carried
		// to the castle.
		return
	case data.SqSpinner:
		// Pascal SPECSQAR SPINNER: randomize direction
		game.Facing = engine.Direction(rand.Intn(4))
	}
}

// hazardDamage applies falling damage to party members (chutes, pits).
// Pascal ROCKWATR (RUNNER2.TEXT lines 232-261):
//
//	Agility save: (RANDOM MOD 25) + MAZELEV > ATTRIB[AGILITY]
//	Damage: AUX0 (base) + sum(AUX2 dice of AUX1 sides)
//
// Uses the current cell's damage data.
func hazardDamage(game *engine.GameState) {
	cell := game.CurrentCell()
	if cell == nil {
		return
	}
	level := game.MazeLevel + 1
	for _, m := range game.Town.Party.Members {
		if m == nil || m.IsDead() {
			continue
		}
		// Agility save
		if rand.Intn(25)+level > m.Agility {
			// Roll damage from cell's AUX data
			dmg := cell.BaseDamage
			for i := 0; i < cell.NumDice; i++ {
				if cell.DieSize > 0 {
					dmg += 1 + rand.Intn(cell.DieSize)
				}
			}
			if dmg < 1 {
				dmg = 1
			}
			m.HP -= dmg
			if m.HP <= 0 {
				m.HP = 0
				m.Status = engine.Dead
			}
		}
	}
}

// ouchyDamage applies damage from ouchy squares using the cell's dice.
// From p-code proc 17 (IC 2669).
func ouchyDamage(game *engine.GameState, cell *data.MazeCell) {
	if cell.NumDice <= 0 || cell.DieSize <= 0 {
		return
	}
	for _, m := range game.Town.Party.Members {
		if m == nil || m.IsDead() {
			continue
		}
		dmg := cell.BaseDamage
		for d := 0; d < cell.NumDice; d++ {
			dmg += rand.Intn(cell.DieSize) + 1
		}
		m.HP -= dmg
		if m.HP <= 0 {
			m.HP = 0
			m.Status = engine.Dead
		}
	}
}
