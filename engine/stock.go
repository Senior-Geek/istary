package engine

import "github.com/Senior-Geek/istary/data"

// Boltac's stock (M8-1). The scenario's items define each item's initial
// stock: -1 unlimited, 0 not stocked, n that many. Buying and selling change
// the stock of the game being played, never the scenario, so games on one
// scenario (the GUI's saved game and its practice games, a reload) never
// share it. Stock is not saved (finding #12): every new game, and so every
// restart and reload, starts from the scenario's values.

// initialStock copies the scenario's initial stock, by item index.
func initialStock(items []data.Item) []int {
	stock := make([]int, len(items))
	for i, item := range items {
		stock[i] = item.Stock
	}
	return stock
}

// Stock returns item's stock in this game: -1 unlimited, 0 none, or how
// many are left. An item the game does not know has none.
func (g *GameState) Stock(item int) int {
	if item < 0 || item >= len(g.shopStock) {
		return 0
	}
	return g.shopStock[item]
}

// SetStock sets item's stock in this game.
func (g *GameState) SetStock(item, n int) {
	if item >= 0 && item < len(g.shopStock) {
		g.shopStock[item] = n
	}
}
