package main

// Boltac's Trading Post (M8-3). The Trading Post screen calls one method,
// ShopAction, with one of the requests below; it runs the controller's
// Boltac's operation (app/shop_ops.go), which presses exactly the
// terminal's keys. View.Shop is the controller's ShopView: every price,
// fee, limit and refusal comes from it, so the screen decides nothing.
//
// Saving follows the town's convention (the terminal saves after every
// key), narrowed to what changes the save: a completed transaction (a
// purchase, a sale, an uncursing, an identification) is saved at once, in
// one atomic write of the roster, so gold and item are saved together. An
// action that changes no character (choosing the customer, a refusal,
// Boltac's question about an unusable item, an answer of no) writes
// nothing, and neither does looking. Boltac's stock is not saved (finding
// #12): it starts again from the scenario on every load.
//
// A save that fails does not undo the transaction: it has happened in this
// window and stays (as every action does, session.go save), the View says
// it is not saved (Unsaved, and the receipt's Saved), and the next save
// writes it with whatever follows. Shop actions are refused away from
// Boltac's, and so during practice and expeditions. Nothing here draws a
// random number.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/engine"
)

// Shop actions (ShopRequest.Action).
const (
	ShopActEnter         = "enter"          // Slot: the party member who enters (0-based), in place of any customer
	ShopActLeaveCustomer = "leave-customer" // back to "WHO WILL ENTER"
	ShopActBuy           = "buy"            // Item: a ShopOffer's Item
	ShopActConfirm       = "confirm"        // Yes: buy the unusable item anyway
	ShopActConfirmSale   = "confirm-sale"   // Yes: sell the equipped item (B7)
	ShopActSell          = "sell"           // N: a ShopOwned's N
	ShopActUncurse       = "uncurse"        // N
	ShopActIdentify      = "identify"       // N
)

// ShopRequest is a Boltac's action chosen in the front end.
type ShopRequest struct {
	Action string `json:"action"`
	Slot   int    `json:"slot"`
	Item   int    `json:"item"`
	N      int    `json:"n"`
	Yes    bool   `json:"yes"`
}

// ShopPanel is Boltac's for the front end: the controller's ShopView, and
// the last transaction made there.
type ShopPanel struct {
	View app.ShopView `json:"view"`
	// Last is the transaction the last shop action completed, until the
	// next one (or the player leaves); nil when it completed none.
	Last *ShopReceipt `json:"last"`
}

// ShopReceipt is a completed transaction and whether it is saved.
type ShopReceipt struct {
	Action   string `json:"action"`   // buy, sell, uncurse or identify
	Customer string `json:"customer"` // the customer's name
	Item     string `json:"item"`     // as the customer knows it (identify: its true name)
	Gold     int    `json:"gold"`     // the customer's gold change: negative paid, positive received
	// Saved says the save folder holds the transaction; false, it exists
	// only in this window until a save succeeds (View.Unsaved says why).
	// Durable says the folder was synced too (View.Warning when not).
	Saved   bool `json:"saved"`
	Durable bool `json:"durable"`
}

var (
	errNoCustomer   = errors.New("Choose who will enter Boltac's first.")
	errAnswerBoltac = errors.New("Answer Boltac's question first.")
	errNoSuchOffer  = errors.New("Boltac does not sell that item now.")
	errNoSuchItem   = errors.New("The customer has no such item.")
	errNotAsked     = errors.New("Boltac is not asking that.")
)

// ShopAction performs one of Boltac's actions.
func (s *Session) ShopAction(r ShopRequest) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.view(nil)
	}
	s.notice, s.receipt = "", nil
	if err := s.shopRefusal(); err != nil {
		return s.view(err)
	}
	c := s.ctrl
	before, _ := c.ShopView()
	roster := rosterBytes(c.Game)
	err := shopOp(c, before, r)
	if bytes.Equal(roster, rosterBytes(c.Game)) {
		return s.view(err) // nothing to save
	}
	saveErr := s.save()
	s.receipt = receipt(c, before, r)
	s.receipt.Saved = saveErr == nil
	s.receipt.Durable = saveErr == nil && s.uncertain == ""
	s.notice = receiptNotice(s.receipt)
	return s.view(errors.Join(err, saveErr))
}

// shopRefusal is why no shop action can be taken now, or nil.
func (s *Session) shopRefusal() error {
	if err := s.navigationRefusal(); err != nil {
		return err // an expedition, practice, a character being created
	}
	if s.ctrl.Game.Town.Location != engine.Trading {
		return notAt(engine.Trading)
	}
	return nil
}

// shopOp runs r on c, which is at Boltac's (v is its ShopView now).
func shopOp(c *app.Controller, v app.ShopView, r ShopRequest) error {
	switch r.Action {
	case ShopActEnter:
		// Another customer: the one at the counter leaves first, as RETURN
		// does, unless Boltac is asking something.
		switch {
		case asking(v):
			return errAnswerBoltac
		case v.Step == "who":
		case r.Slot == v.Customer:
			return nil // already at the counter
		case !isMember(c.Game, r.Slot):
			return app.ErrNoSuchPartyMember
		default:
			if err := c.ShopLeaveCustomer(); err != nil {
				return err
			}
		}
		return c.ShopEnter(r.Slot)
	case ShopActLeaveCustomer:
		if v.Step == "who" {
			return errNoCustomer
		}
		return c.ShopLeaveCustomer()
	case ShopActConfirm:
		if v.Step != "confirm" {
			return errNotAsked
		}
		return c.ConfirmUnusable(r.Yes)
	case ShopActConfirmSale:
		if v.Step != "confirmSell" {
			return errNotAsked
		}
		return c.ConfirmSale(r.Yes)
	case ShopActBuy, ShopActSell, ShopActUncurse, ShopActIdentify:
	default:
		return app.ErrInvalidChoice
	}
	switch {
	case v.Step == "who":
		return errNoCustomer
	case asking(v):
		return errAnswerBoltac
	}
	var err error
	switch r.Action {
	case ShopActBuy:
		err = c.Buy(r.Item)
	case ShopActSell:
		err = c.Sell(r.N)
	case ShopActUncurse:
		err = c.Uncurse(r.N)
	case ShopActIdentify:
		err = c.ShopIdentify(r.N)
	}
	if errors.Is(err, app.ErrInvalidChoice) {
		if r.Action == ShopActBuy {
			return errNoSuchOffer
		}
		return errNoSuchItem
	}
	return err
}

// asking reports whether Boltac waits for an answer: about an unusable
// item, or before buying an equipped one.
func asking(v app.ShopView) bool {
	return v.Step == "confirm" || v.Step == "confirmSell"
}

// isMember reports whether slot (0-based) holds a party member.
func isMember(g *engine.GameState, slot int) bool {
	m := g.Town.Party.Members
	return slot >= 0 && slot < len(m) && m[slot] != nil
}

// rosterBytes is the roster as saved, to tell whether an action changed it.
func rosterBytes(g *engine.GameState) []byte {
	raw, err := json.Marshal(g.Town.Roster.Characters)
	if err != nil {
		return nil // never equal to another: the change is saved
	}
	return raw
}

// receipt describes the transaction r completed; v is the ShopView before it.
func receipt(c *app.Controller, v app.ShopView, r ShopRequest) *ShopReceipt {
	out := &ShopReceipt{Action: r.Action, Customer: v.Name}
	if ch := c.Game.Town.ShopChar; ch != nil {
		out.Gold = ch.Gold - v.Gold
	}
	items := c.Game.Scenario.Items
	switch r.Action {
	case ShopActBuy:
		if r.Item >= 0 && r.Item < len(items) {
			out.Item = items[r.Item].Name
		}
	case ShopActConfirm:
		out.Action = ShopActBuy
		if v.Confirm >= 0 && v.Confirm < len(items) {
			out.Item = items[v.Confirm].Name
		}
	case ShopActSell, ShopActUncurse:
		if r.N >= 1 && r.N <= len(v.Items) {
			out.Item = v.Items[r.N-1].Name
		}
	case ShopActConfirmSale:
		out.Action = ShopActSell
		if n := v.SellConfirm; n >= 1 && n <= len(v.Items) {
			out.Item = v.Items[n-1].Name
		}
	case ShopActIdentify:
		if r.N >= 1 && r.N <= len(v.Items) {
			out.Item = items[v.Items[r.N-1].Item].Name
		}
	}
	return out
}

// receiptNotice says what was done, and whether it is saved.
func receiptNotice(r *ShopReceipt) string {
	var did string
	switch r.Action {
	case ShopActBuy:
		did = fmt.Sprintf("%s bought %s for %d GP.", r.Customer, r.Item, -r.Gold)
	case ShopActSell:
		did = fmt.Sprintf("%s sold %s for %d GP.", r.Customer, r.Item, r.Gold)
	case ShopActUncurse:
		did = fmt.Sprintf("Boltac uncursed %s's %s and kept it, for %d GP.", r.Customer, r.Item, -r.Gold)
	case ShopActIdentify:
		did = fmt.Sprintf("Boltac identified %s's item, for %d GP: it is %s.", r.Customer, -r.Gold, r.Item)
	}
	return did + savedWords(r.Saved, r.Durable)
}

// shopPanel is Boltac's for the View, or nil away from it.
func (s *Session) shopPanel() *ShopPanel {
	if s.active() != nil {
		return nil
	}
	v, err := s.ctrl.ShopView()
	if err != nil {
		return nil
	}
	return &ShopPanel{View: v, Last: s.receipt}
}
