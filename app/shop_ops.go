package app

import "github.com/Senior-Geek/istary/engine"

// Boltac's operations (M8-2), for front ends without keys. Each one is the
// terminal's keys at Boltac's, pressed for the player, so the two cannot
// differ; arguments are checked first, and a bad one changes nothing. A
// refusal is returned as *Refusal with Boltac's words, which are also in
// Town.Message, as the keys leave them. Buying an item the customer's
// class cannot use is not a refusal: Boltac asks first (ShopView.Step
// "confirm"), and ConfirmUnusable answers. Nothing here draws a random
// number.
//
// The one place an operation is not a key: Buy takes the item itself,
// where the terminal pages through the catalog to it; the page shown and
// the catalog cursor are the terminal's, and Buy leaves them as they are
// (save that, as the keys do, an unusable item becomes the cursor).

// shopRefusals are Boltac's refusals: the answers that change nothing.
var shopRefusals = map[string]bool{
	"** YOU BOUGHT THE LAST ONE **":      true,
	"** YOU CANT CARRY ANYTHING MORE **": true,
	"** YOU CANNOT AFFORD IT **":         true,
	"** WE DONT BUY CURSED ITEMS **":     true,
	"** THAT IS NOT A CURSED ITEM **":    true,
	"** YOU CANT AFFORD THE FEE **":      true,
	"** THAT HAS BEEN IDENTIFIED **":     true,
}

// shopAnswer returns Boltac's last answer as a *Refusal if it refused.
func (c *Controller) shopAnswer() error {
	if msg := c.Game.Town.Message; shopRefusals[msg] {
		return &Refusal{Message: msg}
	}
	return nil
}

// atShop reports that the player is at Boltac's.
func (c *Controller) atShop() bool {
	g := c.Game
	return g.Phase == engine.PhaseTown && g.Town.Location == engine.Trading
}

// shopping reports that a customer is at Boltac's and nothing waits for an
// answer: the main menu, the catalog, or a list.
func (c *Controller) shopping() bool {
	town := c.Game.Town
	return c.atShop() && town.ShopChar != nil && town.ShopStep != engine.ShopWho && town.ShopStep != engine.ShopBuyConfirm
}

// toShopStep opens Boltac's menu on screen for step (ShopBuy, ShopSell,
// ShopUncurse or ShopIdentify) as the keys do: RETURN back to the main
// menu, then the menu's letter. A screen already open stays open.
func (c *Controller) toShopStep(step engine.ShopStep, letter rune) {
	town := c.Game.Town
	if town.InputMode == engine.InputShopPurchase {
		c.press(Key{Kind: KeyEnter})
	}
	if town.ShopStep == step {
		return
	}
	for town.ShopStep != engine.ShopMain {
		c.press(Key{Kind: KeyEnter})
	}
	c.press(Rune(letter))
}

// ShopEnter chooses the party member in slot (0-based) as Boltac's
// customer ("WHO WILL ENTER").
func (c *Controller) ShopEnter(slot int) error {
	town := c.Game.Town
	if !c.atShop() || town.ShopStep != engine.ShopWho || town.InputMode != engine.InputNone {
		return ErrWrongState
	}
	if slot < 0 || slot >= MaxPartySize || slot >= len(town.Party.Members) || town.Party.Members[slot] == nil {
		return ErrNoSuchPartyMember
	}
	c.press(Rune('1' + rune(slot)))
	return nil
}

// ShopLeaveCustomer ends the customer's visit: back to "WHO WILL ENTER",
// by RETURN as the keys do (a question waiting is answered no).
func (c *Controller) ShopLeaveCustomer() error {
	town := c.Game.Town
	if !c.atShop() || town.ShopChar == nil {
		return ErrWrongState
	}
	for town.ShopStep != engine.ShopWho {
		c.press(Key{Kind: KeyEnter})
	}
	return nil
}

// LeaveShop leaves Boltac's for the castle from wherever the player is in
// it, by RETURN as the keys do: an open prompt is closed, a question
// waiting is answered no, the customer leaves.
func (c *Controller) LeaveShop() error {
	if !c.atShop() {
		return ErrWrongState
	}
	for c.Game.Town.Location == engine.Trading {
		c.press(Key{Kind: KeyEnter})
	}
	return nil
}

// Buy buys item (its index in the scenario) for the customer from the
// catalog: the keys P and the item's place on the page. Boltac may refuse,
// or ask first about an item the customer's class cannot use.
func (c *Controller) Buy(item int) error {
	g := c.Game
	if !c.shopping() {
		return ErrWrongState
	}
	if !inCatalog(g, item) {
		return ErrInvalidChoice
	}
	c.toShopStep(engine.ShopBuy, 'B')
	g.Town.Message = "" // as the key P does
	shopBuy(g, g.Town, item)
	return c.shopAnswer()
}

// ConfirmUnusable answers Boltac's "UNUSABLE ITEM - CONFIRM BUY": Y buys it
// anyway, N does not.
func (c *Controller) ConfirmUnusable(yes bool) error {
	town := c.Game.Town
	if !c.atShop() || town.ShopStep != engine.ShopBuyConfirm {
		return ErrWrongState
	}
	if yes {
		c.press(Rune('Y'))
	} else {
		c.press(Rune('N'))
	}
	return nil
}

// Sell sells the customer's item n (1-based): S and n.
func (c *Controller) Sell(n int) error {
	return c.shopItem(engine.ShopSell, 'S', n)
}

// Uncurse has Boltac uncurse the customer's item n (1-based), which he
// takes: U and n.
func (c *Controller) Uncurse(n int) error {
	return c.shopItem(engine.ShopUncurse, 'U', n)
}

// ShopIdentify has Boltac identify the customer's item n (1-based): I and
// n.
func (c *Controller) ShopIdentify(n int) error {
	return c.shopItem(engine.ShopIdentify, 'I', n)
}

func (c *Controller) shopItem(step engine.ShopStep, letter rune, n int) error {
	if !c.shopping() {
		return ErrWrongState
	}
	if ch := c.Game.Town.ShopChar; n < 1 || n > ch.ItemCount {
		return ErrInvalidChoice
	}
	c.toShopStep(step, letter)
	c.press(Rune('0' + rune(n)))
	return c.shopAnswer()
}

// inCatalog reports whether Boltac lists item: one he has in stock and not
// of a cursed type (item 0 is never sold), as the catalog pages show.
func inCatalog(g *engine.GameState, item int) bool {
	items := g.Scenario.Items
	return item >= 1 && item < len(items) && g.Stock(item) != 0 && !items[item].Cursed
}
