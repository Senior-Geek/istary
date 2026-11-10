package engine

import "testing"

// Every square of every level converts to L:R:C and back, both ways, and
// positions off the maze have no canonical form.
func TestPlaceRoundTrip(t *testing.T) {
	levels := len(loadScenario(t, 1).Mazes.Levels)
	if levels != 10 {
		t.Fatalf("%d levels", levels)
	}
	n := 0
	for level := 0; level < levels; level++ {
		for x := 0; x < MazeSize; x++ {
			for y := 0; y < MazeSize; y++ {
				p, ok := PlaceOf(levels, level, x, y)
				if !ok || !p.Valid(levels) {
					t.Fatalf("lvl%d x%d y%d: %v %v", level, x, y, p, ok)
				}
				if p.Level != level+1 || p.Row != 19-y || p.Col != x {
					t.Fatalf("lvl%d x%d y%d -> %v", level, x, y, p)
				}
				if l2, x2, y2 := p.Internal(); l2 != level || x2 != x || y2 != y {
					t.Fatalf("%v -> lvl%d x%d y%d", p, l2, x2, y2)
				}
				n++
			}
		}
	}
	for l := 1; l <= levels; l++ {
		for r := 0; r < MazeSize; r++ {
			for c := 0; c < MazeSize; c++ {
				p := Place{Level: l, Row: r, Col: c}
				il, ix, iy := p.Internal()
				if q, ok := PlaceOf(levels, il, ix, iy); !ok || q != p {
					t.Fatalf("%v -> %v %v", p, q, ok)
				}
			}
		}
	}
	if n != 4000 {
		t.Fatalf("%d squares", n)
	}
	for _, bad := range [][3]int{{-1, 0, 0}, {levels, 0, 0}, {levels + 1, 5, 5}, {0, 20, 0}, {0, 0, 20}, {0, -1, 0}, {0, 0, -1}} {
		if p, ok := PlaceOf(levels, bad[0], bad[1], bad[2]); ok {
			t.Errorf("lvl%d x%d y%d gave %v", bad[0], bad[1], bad[2], p)
		}
	}
	for _, bad := range []Place{{Level: 0, Row: 3}, {Level: 11}, {Level: 1, Row: 20}, {Level: 1, Col: -1}} {
		if bad.Valid(levels) {
			t.Errorf("%v valid", bad)
		}
	}
	if CastlePlace.String() != "L0" || !CastlePlace.Valid(levels) {
		t.Errorf("castle %v", CastlePlace)
	}
	if got := (Place{Level: 3, Row: 12, Col: 8}).String(); got != "L3:R12:C8" {
		t.Errorf("%s", got)
	}
}

// The entrance is the one square of the first level with stairs to the
// castle, L1:R19:C0, in every scenario, and Step wraps at all four edges.
func TestEntranceAndStep(t *testing.T) {
	for _, num := range []int{1, 2, 3} {
		got, err := MazeEntrance(&loadScenario(t, num).Mazes)
		if err != nil || got != Entrance {
			t.Errorf("scenario %d: %v %v", num, got, err)
		}
	}
	if Entrance.String() != "L1:R19:C0" || EntranceFacing.Letter() != "N" {
		t.Errorf("%v %v", Entrance, EntranceFacing)
	}
	for _, c := range []struct {
		from Place
		d    Direction
		want Place
	}{
		{Place{1, 0, 10}, North, Place{1, 19, 10}},
		{Place{1, 19, 10}, South, Place{1, 0, 10}},
		{Place{2, 2, 19}, East, Place{2, 2, 0}},
		{Place{2, 2, 0}, West, Place{2, 2, 19}},
		{Place{1, 5, 5}, North, Place{1, 4, 5}},
		{Place{1, 5, 5}, East, Place{1, 5, 6}},
	} {
		if got := c.from.Step(c.d); got != c.want {
			t.Errorf("%v %v: %v", c.from, c.d, got)
		}
	}
	// Step agrees with the engine's own step (MoveForward's arithmetic).
	for x := 0; x < MazeSize; x++ {
		for y := 0; y < MazeSize; y++ {
			for d := North; d <= West; d++ {
				p, _ := PlaceOf(10, 0, x, y)
				nx, ny := Shift(x, y, d, 0, 1)
				want, _ := PlaceOf(10, 0, nx, ny)
				if got := p.Step(d); got != want {
					t.Fatalf("%v %v: %v, engine %v", p, d, got, want)
				}
			}
		}
	}
}
