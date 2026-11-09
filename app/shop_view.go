package app

import "github.com/Senior-Geek/istary/engine"

// ShopView describes Boltac's Trading Post for a front end: where the
// visit is, Boltac's last words, the customer, the catalog and the
// customer's items, each with what Boltac would answer now. Every price,
// fee and refusal comes from the checks the operations act on (buyQuote,
// sellQuote, uncurseQuote, identifyQuote), so a front end decides nothing,
// except that a curse the player has not learned of is not told in advance
// (ShopOwned.Cursed).
// It changes nothing and draws no random number.
type ShopView struct {
	// Step is "who" (choosing the customer), "main", "buy" (the catalog),
	// "confirm" (Boltac asks about an unusable item), "sell", "uncurse"
	// or "identify" (the customer's items), "confirmSell" (Boltac asks
	// before buying an equipped item).
	Step    string `json:"step"`
	Prompt  bool   `json:"prompt"`  // the terminal's PURCHASE WHICH ITEM prompt is open
	Message string `json:"message"` // Boltac's last words
	// Customer is the customer's party slot (0-based), or -1 at "who".
	Customer int    `json:"customer"`
	Name     string `json:"name"`
	Class    string `json:"class"` // the customer's class, e.g. "MAGE"
	Gold     int    `json:"gold"`
	// Confirm is the item Boltac asks about at "confirm", or -1.
	Confirm int `json:"confirm"`
	// SellConfirm is the customer's item (1-based, as Sell takes) Boltac
	// asks about at "confirmSell": it is equipped. 0 otherwise.
	SellConfirm int `json:"sellConfirm"`
	// Catalog is every item Boltac sells now, in his order; Items the
	// customer's possessions, by number. Both are empty at "who".
	Catalog []ShopOffer `json:"catalog"`
	Items   []ShopOwned `json:"items"`
	// Room is how many more items the customer can carry (8 at most).
	Room int `json:"room"`
}

// ShopOffer is one item in Boltac's catalog.
type ShopOffer struct {
	Item  int    `json:"item"` // index in the scenario; what Buy takes
	Name  string `json:"name"`
	Price int    `json:"price"`
	Stock int    `json:"stock"` // -1 unlimited, or how many are left
	// Usable says whether the customer's class can use it; if not, Boltac
	// asks before selling it (not a refusal).
	Usable bool `json:"usable"`
	// Refusal is Boltac's answer if the customer tried to buy it now
	// (too little gold, no room...), or "".
	Refusal string `json:"refusal"`
}

// ShopOwned is one of the customer's items, with Boltac's terms for it.
type ShopOwned struct {
	N          int    `json:"n"` // 1-based, as Sell, Uncurse and ShopIdentify take
	Item       int    `json:"item"`
	Name       string `json:"name"` // its unknown name until identified
	Identified bool   `json:"identified"`
	Equipped   bool   `json:"equipped"`
	// Cursed: stuck to the customer, as far as the player knows
	// (engine.KnownCursed): never before it is identified or equipped.
	Cursed bool `json:"cursed"`
	// Price is what Boltac pays (1 GP if not identified); SellRefusal why
	// he will not buy it, or ""; SellAsks that he asks first (it is
	// equipped; selling takes it off).
	Price       int    `json:"price"`
	SellRefusal string `json:"sellRefusal"`
	SellAsks    bool   `json:"sellAsks"`
	// Fee is Boltac's fee to uncurse or identify it; UncurseRefusal and
	// IdentifyRefusal why he will not, or "".
	Fee             int    `json:"fee"`
	UncurseRefusal  string `json:"uncurseRefusal"`
	IdentifyRefusal string `json:"identifyRefusal"`
}

var shopStepNames = map[engine.ShopStep]string{
	engine.ShopWho: "who", engine.ShopMain: "main", engine.ShopBuy: "buy", engine.ShopBuyConfirm: "confirm",
	engine.ShopSell: "sell", engine.ShopUncurse: "uncurse", engine.ShopIdentify: "identify",
	engine.ShopSellConfirm: "confirmSell",
}

// ShopView describes Boltac's now; ErrWrongState away from it.
func (c *Controller) ShopView() (ShopView, error) {
	g := c.Game
	if !c.atShop() {
		return ShopView{}, ErrWrongState
	}
	town := g.Town
	v := ShopView{
		Step:     shopStepNames[town.ShopStep],
		Prompt:   town.InputMode == engine.InputShopPurchase,
		Message:  town.Message,
		Customer: -1,
		Confirm:  -1,
	}
	ch := town.ShopChar
	if ch == nil || town.ShopStep == engine.ShopWho {
		return v, nil
	}
	items := g.Scenario.Items
	v.Customer = PartySlot(town, ch)
	v.Name, v.Class, v.Gold = ch.Name, ch.Class.String(), ch.Gold
	v.Room = max(0, 8-ch.ItemCount)
	if town.ShopStep == engine.ShopBuyConfirm {
		v.Confirm = town.ShopCatalog
	}
	if town.ShopStep == engine.ShopSellConfirm {
		v.SellConfirm = town.ShopSellItem + 1
	}
	for i := 1; i < len(items); i++ {
		if !inCatalog(g, i) {
			continue
		}
		refusal, _ := buyQuote(g, ch, i)
		v.Catalog = append(v.Catalog, ShopOffer{
			Item: i, Name: items[i].Name, Price: items[i].Price, Stock: g.Stock(i),
			Usable: usableBy(items[i], ch), Refusal: refusal,
		})
	}
	for i := 0; i < ch.ItemCount; i++ {
		p := ch.Items[i]
		if p.ItemIndex < 0 || p.ItemIndex >= len(items) {
			continue
		}
		item := items[p.ItemIndex]
		o := ShopOwned{N: i + 1, Item: p.ItemIndex, Name: item.Name, Identified: p.Identified, Equipped: p.Equipped, Cursed: engine.KnownCursed(p)}
		if !p.Identified && item.NameUnknown != "" {
			o.Name = item.NameUnknown
		}
		o.Price, o.SellRefusal = sellQuote(item, p)
		o.SellAsks = o.SellRefusal == "" && p.Equipped
		o.Fee, o.UncurseRefusal = uncurseQuote(ch, item, p)
		_, o.IdentifyRefusal = identifyQuote(ch, item, p)
		if p.Cursed && !o.Cursed {
			// A curse the player has not learned of (only an imported item
			// can carry one unseen) is not told in advance: Boltac's answer
			// to trying says it, as in the game.
			o.SellRefusal, o.UncurseRefusal = "", ""
		}
		v.Items = append(v.Items, o)
	}
	return v, nil
}
