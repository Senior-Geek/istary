package main

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/Senior-Geek/istary/app"
	"github.com/Senior-Geek/istary/data"
	"github.com/Senior-Geek/istary/engine"
	"github.com/Senior-Geek/istary/render"
)

// The dungeon view's corridor carries exactly what the terminal's Classic
// view draws: changing a wall changes the drawing if and only if it changes
// the view. So the view reveals nothing the screen does not, and leaves out
// nothing it shows. (Without light a hidden door is a wall in both; the
// Classic one-in-six flicker is left out of the view, see app.DungeonView.)
func TestCorridorIsWhatTheClassicViewDraws(t *testing.T) {
	g := newGame(t, 1)
	c := app.New(g)
	g.Phase = engine.PhaseMaze
	pick := rand.New(rand.NewSource(1))
	walls := []data.WallType{data.WallOpen, data.WallWall, data.WallDoor, data.WallHidden}
	draw := func() render.MazeBitmap {
		var m render.MazeBitmap
		render.DrawMaze(&m, &g.Scenario.Mazes.Levels[g.MazeLevel], g.PlayerX, g.PlayerY, g.Facing, g.LightLevel, g.QuickPlot)
		return m
	}
	checked, changed := 0, 0
	for trial := 0; trial < 1500; trial++ {
		g.MazeLevel = pick.Intn(len(g.Scenario.Mazes.Levels))
		g.PlayerX, g.PlayerY, g.Facing = pick.Intn(20), pick.Intn(20), engine.Direction(pick.Intn(4))
		g.LightLevel = []int{0, 0, 30}[pick.Intn(3)]
		g.QuickPlot = pick.Intn(4) == 0
		level := &g.Scenario.Mazes.Levels[g.MazeLevel]
		saved := level.Cells
		level.Cells = make([][]data.MazeCell, len(saved))
		for y := range saved {
			level.Cells[y] = append([]data.MazeCell(nil), saved[y]...)
			if g.LightLevel == 0 {
				for x := range level.Cells[y] {
					cell := &level.Cells[y][x]
					for _, w := range []*data.WallType{&cell.N, &cell.E, &cell.S, &cell.W} {
						if *w == data.WallHidden {
							*w = data.WallWall
						}
					}
				}
			}
		}
		beforeView, beforeDraw := c.DungeonView(), draw()
		for p := 0; p < 60; p++ {
			// Perturb a wall near the corridor ahead (and just behind).
			right, fwd := pick.Intn(3)-1, pick.Intn(6)-1
			dx, dy := g.Facing.DX(), g.Facing.DY()
			x := ((g.PlayerX+fwd*dx+right*dy)%20 + 20) % 20
			y := ((g.PlayerY+fwd*dy-right*dx)%20 + 20) % 20
			cell := &level.Cells[y][x]
			side := []*data.WallType{&cell.N, &cell.E, &cell.S, &cell.W}[pick.Intn(4)]
			old := *side
			w := walls[pick.Intn(len(walls))]
			if g.LightLevel == 0 && w == data.WallHidden {
				w = data.WallWall
			}
			*side = w
			viewSame := reflect.DeepEqual(c.DungeonView(), beforeView)
			drawSame := draw() == beforeDraw
			if viewSame != drawSame {
				t.Fatalf("L%d (%d,%d) %v light %d: wall at (%d,%d) %s -> %s changes the drawing %v but the view %v",
					g.MazeLevel+1, g.PlayerX, g.PlayerY, g.Facing, g.LightLevel, x, y, old, w, !drawSame, !viewSame)
			}
			checked++
			if !drawSame {
				changed++
			}
			*side = old
		}
		level.Cells = saved
	}
	if changed < checked/20 {
		t.Fatalf("only %d of %d perturbations changed the drawing; the test is too weak", changed, checked)
	}
}
