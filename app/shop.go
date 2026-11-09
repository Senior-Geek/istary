package app

import (
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
)

// Boltac's Trading Post (M8-2), moved intact from the terminal (cmd/istary
// handleShopInput, shopBuyItem, fillShopBuyPage and handleShopReturn, and
// the castle menu's B): the terminal reaches it through HandleKey. The
// flow is traced from the SHOPS segment p-code; stock is the game's own
// (engine/stock.go). Nothing here draws a random number.

// castleShopKey is the castle menu's B: Boltac's, for a party.
func (c *Controller) castleShopKey() {
	town := c.Game.Town
	town.Message = ""
	if town.Party.Size() == 0 {
		town.Message = "YOU HAVE NO PARTY!"
	} else {
		town.Location = engine.Trading
		town.ShopStep = engine.ShopWho
		town.ShopChar = nil
		town.ShopCatalog = 0
	}
}

// shopKey handles a key at Boltac's with no prompt open: RETURN steps back,
// a letter or digit goes to the menu on screen.
func (c *Controller) shopKey(k Key) {
	town := c.Game.Town
	switch k.Kind {
	case KeyEnter, KeyEscape:
		shopReturn(town)
		return
	case KeyRune:
	default:
		return
	}
	town.Message = ""
	shopInput(c.Game, town, upper(k.Rune))
}

// shopPurchaseKey handles "PURCHASE WHICH ITEM ([RETURN] EXITS) ? >":
// 1-6 buys the item in that place on the page (p-code CLP 15, IC 1627).
func (c *Controller) shopPurchaseKey(k Key) {
	town := c.Game.Town
	if k.Kind == KeyRune && k.Rune >= '1' && k.Rune <= '6' {
		shopBuyItem(c.Game, town, int(k.Rune-'1'))
		town.InputMode = engine.InputNone
	}
	if k.Kind == KeyEnter || k.Kind == KeyEscape {
		town.InputMode = engine.InputNone
		town.Message = ""
	}
}

// fillShopBuyPage populates town.ShopPage with up to 6 item indices starting
// from town.ShopCatalog, skipping items with stock==0. Wraps around the item
// table (items 1..N, skipping item 0). From p-code CLP 13 (IC 1100-1306).
func fillShopBuyPage(game *engine.GameState, town *engine.TownState, items []data.Item, forward bool) {
	total := len(items)
	if total <= 1 {
		for i := range town.ShopPage {
			town.ShopPage[i] = -1
		}
		return
	}

	idx := town.ShopCatalog
	if forward {
		for slot := 0; slot < 6; slot++ {
			// Find next item with stock != 0
			found := false
			for tries := 0; tries < total; tries++ {
				idx++
				if idx >= total {
					idx = 1 // wrap to item 1 (skip item 0)
				}
				if game.Stock(idx) != 0 && !items[idx].Cursed {
					town.ShopPage[slot] = idx
					found = true
					break
				}
			}
			if !found {
				town.ShopPage[slot] = -1
			}
		}
	} else {
		// Fill backward: find 6 items going backward from ShopPage[0]
		idx = town.ShopPage[0]
		for slot := 5; slot >= 0; slot-- {
			found := false
			for tries := 0; tries < total; tries++ {
				idx--
				if idx < 1 {
					idx = total - 1 // wrap to last item
				}
				if game.Stock(idx) != 0 && !items[idx].Cursed {
					town.ShopPage[slot] = idx
					found = true
					break
				}
			}
			if !found {
				town.ShopPage[slot] = -1
			}
		}
	}
	// Update ShopCatalog to track the last displayed item (for forward scrolling)
	for i := 5; i >= 0; i-- {
		if town.ShopPage[i] >= 0 {
			town.ShopCatalog = town.ShopPage[i]
			break
		}
	}
}

// shopInput handles key presses at Boltac's Trading Post.
// Flow traced from SHOPS segment (seg 2) p-code:
//   - ShopWho: select party member (IC 4143-4188)
//   - ShopMain: B/S/U/I/L dispatch (IC 3664-4057)
//   - ShopBuy: P/F/B/S/L browse (CLP 12, IC 2168-2541)
//   - ShopSell/Uncurse/Identify: CLP 17 (IC 3424-3650)
func shopInput(game *engine.GameState, town *engine.TownState, ch rune) {
	items := game.Scenario.Items

	switch town.ShopStep {
	case engine.ShopWho:
		// Select party member by number (1-6) — p-code IC 4161-4183
		if ch >= '1' && ch <= '6' {
			idx := int(ch-'0') - 1
			if idx < len(town.Party.Members) && town.Party.Members[idx] != nil {
				town.ShopChar = town.Party.Members[idx]
				town.ShopStep = engine.ShopMain
				town.Message = ""
			}
		}

	case engine.ShopMain:
		// Main menu dispatch — p-code XJP at IC 4006
		// case 66('B')=Buy, 73('I')=Identify, 76('L')=Leave, 83('S')=Sell, 85('U')=Uncurse
		switch ch {
		case 'B':
			town.ShopStep = engine.ShopBuy
			town.ShopCatalog = 0
			fillShopBuyPage(game, town, items, true)
			town.Message = ""
		case 'S':
			if town.ShopChar.ItemCount == 0 {
				return // no items → stay on main menu (p-code XIT check)
			}
			town.ShopStep = engine.ShopSell
			town.Message = ""
		case 'U':
			if town.ShopChar.ItemCount == 0 {
				return
			}
			town.ShopStep = engine.ShopUncurse
			town.Message = ""
		case 'I':
			if town.ShopChar.ItemCount == 0 {
				return
			}
			town.ShopStep = engine.ShopIdentify
			town.Message = ""
		case 'L':
			// p-code XIT(8, 11) — back to "WHO WILL ENTER"
			town.ShopStep = engine.ShopWho
			town.ShopChar = nil
			town.Message = ""
		}

	case engine.ShopBuy:
		// Browse catalog — p-code CLP 12 (IC 2168-2541)
		// P=purchase, F=forward, B=back, S=start, L=leave
		switch ch {
		case 'F':
			fillShopBuyPage(game, town, items, true)
			town.Message = ""
		case 'B':
			fillShopBuyPage(game, town, items, false)
			town.Message = ""
		case 'S':
			town.ShopCatalog = 0
			fillShopBuyPage(game, town, items, true)
			town.Message = ""
		case 'P':
			town.InputMode = engine.InputShopPurchase
			town.Message = ""
		case 'L':
			town.ShopStep = engine.ShopMain
			town.InputMode = engine.InputNone
			town.Message = ""
		}

	case engine.ShopBuyConfirm:
		// Y/N confirmation for unusable item purchase — p-code IC 1899
		// Pascal SHOPS.TEXT line 358: "ITS YOUR MONEY"
		if ch == 'Y' {
			itemIdx := town.ShopCatalog
			c := town.ShopChar
			if itemIdx >= 0 && itemIdx < len(items) {
				item := items[itemIdx]
				c.Gold -= item.Price
				c.AddItem(itemIdx, true)
				if n := game.Stock(itemIdx); n > 0 {
					game.SetStock(itemIdx, n-1)
				}
				town.Message = "** ITS YOUR MONEY **"
			}
			town.ShopStep = engine.ShopBuy
		} else if ch == 'N' {
			town.Message = ""
			town.ShopStep = engine.ShopBuy
		}

	case engine.ShopSell:
		// p-code CLP 17 mode=0 (IC 2871-3422)
		if ch >= '1' && ch <= '8' {
			idx := int(ch-'0') - 1
			c := town.ShopChar
			if idx < c.ItemCount {
				poss := c.Items[idx]
				itemIdx := poss.ItemIndex
				if itemIdx >= 0 && itemIdx < len(items) {
					item := items[itemIdx]
					_, refusal := sellQuote(item, poss)
					if refusal != "" {
						town.Message = refusal
						town.ShopStep = engine.ShopMain // XIT back to main
						return
					}
					if poss.Equipped {
						// "If you attempt to sell an item that is equipped,
						// you will be asked if that's what you really want
						// to do" (Player's Guide p.31; decision B7).
						town.ShopSellItem = idx
						town.ShopStep = engine.ShopSellConfirm
						town.Message = sellEquippedQuestion
						return
					}
					shopSell(game, town, idx)
				}
			}
		}

	case engine.ShopSellConfirm:
		// Y sells the equipped item; N does not.
		switch ch {
		case 'Y':
			town.ShopStep = engine.ShopSell
			shopSell(game, town, town.ShopSellItem)
		case 'N':
			town.ShopStep = engine.ShopSell
			town.Message = ""
		}

	case engine.ShopUncurse:
		// p-code CLP 17 mode=1 (IC 2966-3422)
		if ch >= '1' && ch <= '8' {
			idx := int(ch-'0') - 1
			c := town.ShopChar
			if idx < c.ItemCount {
				poss := c.Items[idx]
				itemIdx := poss.ItemIndex
				if itemIdx >= 0 && itemIdx < len(items) {
					item := items[itemIdx]
					fee, refusal := uncurseQuote(c, item, poss)
					if refusal != "" {
						town.Message = refusal
						town.ShopStep = engine.ShopMain // XIT back to main
						return
					}
					// Subtract fee — p-code IC 3167-3179 (CXP proc=6)
					c.Gold -= fee
					// Remove item — p-code IC 3211-3308. Boltac takes it
					// even from the hand it is stuck to: the curse ends
					// (DropItem keeps a stuck item; finding #34).
					c.Items[idx].Equipped, c.Items[idx].Cursed = false, false
					c.DropItem(idx)
					c.RecalcAC(items) // #35
					// Increment stock — p-code IC 3337-3357
					if n := game.Stock(itemIdx); n > -1 {
						game.SetStock(itemIdx, n+1)
					}
					town.Message = "** ANYTHING ELSE, SIRE? **"
				}
			}
		}

	case engine.ShopIdentify:
		// p-code CLP 17 mode=2 (IC 3028-3422)
		if ch >= '1' && ch <= '8' {
			idx := int(ch-'0') - 1
			c := town.ShopChar
			if idx < c.ItemCount {
				poss := c.Items[idx]
				itemIdx := poss.ItemIndex
				if itemIdx >= 0 && itemIdx < len(items) {
					item := items[itemIdx]
					fee, refusal := identifyQuote(c, item, poss)
					if refusal != "" {
						town.Message = refusal
						town.ShopStep = engine.ShopMain // XIT back to main
						return
					}
					// Subtract fee — p-code IC 3167-3179
					c.Gold -= fee
					// Set identified flag — p-code IC 3189-3208
					c.Items[idx].Identified = true
					town.Message = "** ANYTHING ELSE, SIRE? **"
				}
			}
		}
	}
}

// sellEquippedQuestion is Boltac's question before buying an equipped
// item. The guide gives no words; these follow his question about an
// unusable item.
const sellEquippedQuestion = "EQUIPPED ITEM - CONFIRM SELL (Y/N) ? >"

// shopSell sells the customer's item idx (0-based): its price to the
// customer, the item to Boltac's stock, and the AC worked out again.
func shopSell(game *engine.GameState, town *engine.TownState, idx int) {
	c := town.ShopChar
	items := game.Scenario.Items
	if idx < 0 || idx >= c.ItemCount {
		return
	}
	poss := c.Items[idx]
	itemIdx := poss.ItemIndex
	if itemIdx < 0 || itemIdx >= len(items) {
		return
	}
	sellPrice, _ := sellQuote(items[itemIdx], poss)
	// Gold transfer: add sell price — p-code IC 3150-3162 (CXP proc=5)
	c.Gold += sellPrice
	// Remove item — p-code IC 3211-3308
	c.DropItem(idx)
	c.RecalcAC(items) // it may have been worn (#35)
	// Increment stock — p-code IC 3337-3357
	if n := game.Stock(itemIdx); n > -1 {
		game.SetStock(itemIdx, n+1)
	}
	town.Message = "** ANYTHING ELSE, SIRE? **"
}

// shopBuyItem executes a purchase from the buy catalog: the item in place
// pageSlot (0-based) on the page shown.
func shopBuyItem(game *engine.GameState, town *engine.TownState, pageSlot int) {
	shopBuy(game, town, town.ShopPage[pageSlot])
}

// shopBuy buys item for the customer, or refuses.
// From p-code CLP 15 (IC 1589-2166): validates stock, capacity, gold, class.
func shopBuy(game *engine.GameState, town *engine.TownState, itemIdx int) {
	items := game.Scenario.Items
	c := town.ShopChar
	if itemIdx < 0 || itemIdx >= len(items) {
		return
	}
	item := items[itemIdx]
	refusal, unusable := buyQuote(game, c, itemIdx)
	if refusal != "" {
		town.Message = refusal
		return
	}
	if unusable {
		// "UNUSABLE ITEM - CONFIRM BUY (Y/N) ? >" — p-code IC 1899
		town.Message = "UNUSABLE ITEM - CONFIRM BUY (Y/N) ? >"
		town.ShopCatalog = itemIdx
		town.ShopStep = engine.ShopBuyConfirm
		return
	}
	// Execute purchase — p-code IC 1989-2112
	c.Gold -= item.Price
	c.AddItem(itemIdx, true) // purchased items are identified — you know what you bought
	// Decrement stock — p-code IC 2072-2084
	if n := game.Stock(itemIdx); n > 0 {
		game.SetStock(itemIdx, n-1)
	}
	town.Message = "** JUST WHAT YOU NEEDED **"
}

// Boltac's answers. The checks below are the only place they are decided:
// the shop's keys and operations act on them, and ShopView reports them.

// buyQuote is Boltac's answer to c buying itemIdx now, in the game's order
// (stock, room, gold): a refusal, or "" with unusable set when he asks
// first because c's class cannot use it (p-code IC 1863-1880).
func buyQuote(game *engine.GameState, c *engine.Character, itemIdx int) (refusal string, unusable bool) {
	item := game.Scenario.Items[itemIdx]
	switch {
	case game.Stock(itemIdx) == 0:
		return "** YOU BOUGHT THE LAST ONE **", false
	case c.ItemCount >= 8:
		return "** YOU CANT CARRY ANYTHING MORE **", false
	case c.Gold < item.Price:
		return "** YOU CANNOT AFFORD IT **", false
	}
	return "", !usableBy(item, c)
}

// usableBy reports whether c's class can use item (its class bits).
func usableBy(item data.Item, c *engine.Character) bool {
	return item.ClassUse == 0 || item.ClassUse&(1<<uint(c.Class)) != 0
}

// sellQuote is what Boltac pays for poss: half the price, or 1 GP if it is
// not identified (p-code IC 2731-2768); he does not buy a cursed item
// (IC 2926-2962).
func sellQuote(item data.Item, poss engine.Possession) (price int, refusal string) {
	price = item.Price / 2
	if !poss.Identified {
		price = 1
	}
	if poss.Cursed {
		return price, "** WE DONT BUY CURSED ITEMS **"
	}
	return price, ""
}

// uncurseQuote is Boltac's fee to uncurse poss, half the price (IC
// 3088-3105), and why he will not: it is not cursed (IC 2966-2990), or c
// cannot pay.
func uncurseQuote(c *engine.Character, item data.Item, poss engine.Possession) (fee int, refusal string) {
	fee = item.Price / 2
	switch {
	case !poss.Cursed:
		return fee, "** THAT IS NOT A CURSED ITEM **"
	case c.Gold < fee:
		return fee, "** YOU CANT AFFORD THE FEE **"
	}
	return fee, ""
}

// identifyQuote is Boltac's fee to identify poss, half the price, and why he
// will not: it is identified already (IC 3028-3051), or c cannot pay.
func identifyQuote(c *engine.Character, item data.Item, poss engine.Possession) (fee int, refusal string) {
	fee = item.Price / 2
	switch {
	case poss.Identified:
		return fee, "** THAT HAS BEEN IDENTIFIED **"
	case c.Gold < fee:
		return fee, "** YOU CANT AFFORD THE FEE **"
	}
	return fee, ""
}

// shopReturn is RETURN (or Escape) at Boltac's with no prompt open.
func shopReturn(town *engine.TownState) {
	switch town.ShopStep {
	case engine.ShopWho:
		GoToCastle(town)
	case engine.ShopMain:
		town.ShopStep = engine.ShopWho
		town.ShopChar = nil
		town.Message = ""
	case engine.ShopSell, engine.ShopUncurse, engine.ShopIdentify:
		// RETURN at sell/uncurse/identify prompt — p-code IC 3620-3627: XIT(8,17)
		town.ShopStep = engine.ShopMain
		town.Message = ""
	case engine.ShopBuyConfirm:
		// RETURN at confirm prompt = cancel (treat as 'N')
		town.ShopStep = engine.ShopBuy
		town.Message = ""
	case engine.ShopSellConfirm:
		// RETURN = no: the item is kept, the sell list stays.
		town.ShopStep = engine.ShopSell
		town.Message = ""
	case engine.ShopBuy:
		if town.InputMode == engine.InputShopPurchase {
			// RETURN exits purchase sub-mode — p-code CLP 15 key==13
			town.InputMode = engine.InputNone
			town.Message = ""
		} else {
			town.ShopStep = engine.ShopMain
			town.Message = ""
		}
	}
}
