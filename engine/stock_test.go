package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

// A new game's stock is the scenario's initial stock, item by item, for
// every scenario; making a game draws no random number.
func TestNewGameStockIsTheScenarios(t *testing.T) {
	for _, num := range []int{1, 2, 3} {
		s := loadScenario(t, num)
		seed(41)
		g := New(s)
		next := rand.Int63()
		seed(41)
		if next != rand.Int63() {
			t.Errorf("scenario %d: New drew a random number", num)
		}
		for i, item := range s.Items {
			if g.Stock(i) != item.Stock {
				t.Fatalf("scenario %d item %d: stock %d, the scenario says %d", num, i, g.Stock(i), item.Stock)
			}
		}
		if g.Stock(-1) != 0 || g.Stock(len(s.Items)) != 0 {
			t.Errorf("scenario %d: an unknown item has stock", num)
		}
		g.SetStock(-1, 5)
		g.SetStock(len(s.Items), 5) // ignored, no panic
	}
}

// Games on one scenario keep their own stock: changing one changes neither
// the other nor the scenario.
func TestGamesDoNotShareStock(t *testing.T) {
	s := loadScenario(t, 1)
	initial := make([]int, len(s.Items))
	for i, item := range s.Items {
		initial[i] = item.Stock
	}
	a, b := New(s), New(s)
	a.SetStock(18, 0)
	a.SetStock(24, 3)
	b.SetStock(28, 26)
	for i := range s.Items {
		if s.Items[i].Stock != initial[i] {
			t.Fatalf("item %d: the scenario's stock changed to %d", i, s.Items[i].Stock)
		}
	}
	if a.Stock(18) != 0 || a.Stock(24) != 3 || a.Stock(28) != 27 {
		t.Errorf("game a: %d %d %d", a.Stock(18), a.Stock(24), a.Stock(28))
	}
	if b.Stock(18) != 2 || b.Stock(24) != 0 || b.Stock(28) != 26 {
		t.Errorf("game b: %d %d %d", b.Stock(18), b.Stock(24), b.Stock(28))
	}
	if c := New(s); !reflect.DeepEqual(c.shopStock, initial) {
		t.Error("a third game does not start from the scenario's stock")
	}
}
