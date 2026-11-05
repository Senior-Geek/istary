package engine

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestScenariosLoad(t *testing.T) {
	tests := []struct {
		num      int
		game     string
		levels   int
		monsters int
		items    int
	}{
		{1, "PROVING GROUNDS OF THE MAD OVERLORD!", 10, 101, 101},
		{2, "THE KNIGHT OF DIAMONDS", 6, 78, 130},
		{3, "THE LEGACY OF LLYLGAMYN", 6, 78, 104},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("wiz%d", tt.num), func(t *testing.T) {
			s := loadScenario(t, tt.num)
			if s.ScenarioNum != tt.num {
				t.Errorf("ScenarioNum = %d, want %d", s.ScenarioNum, tt.num)
			}
			if s.Game != tt.game {
				t.Errorf("Game = %q, want %q", s.Game, tt.game)
			}
			if got := len(s.Mazes.Levels); got != tt.levels {
				t.Errorf("levels = %d, want %d", got, tt.levels)
			}
			if got := len(s.Monsters); got != tt.monsters {
				t.Errorf("monsters = %d, want %d", got, tt.monsters)
			}
			if got := len(s.Items); got != tt.items {
				t.Errorf("items = %d, want %d", got, tt.items)
			}
			// Movement wraps coordinates mod 20, so every level must be 20x20.
			for i, lvl := range s.Mazes.Levels {
				if len(lvl.Cells) != 20 {
					t.Fatalf("level %d has %d rows, want 20", i, len(lvl.Cells))
				}
				for y, row := range lvl.Cells {
					if len(row) != 20 {
						t.Fatalf("level %d row %d has %d cells, want 20", i, y, len(row))
					}
				}
			}
			if len(s.Messages) == 0 {
				t.Error("no scenario messages loaded")
			}
			if len(s.MessagesByLine) != len(s.Messages) {
				t.Errorf("MessagesByLine has %d entries, want %d", len(s.MessagesByLine), len(s.Messages))
			}
			if scenarioKey(s.Game) != fmt.Sprint(tt.num) {
				t.Errorf("scenarioKey(%q) = %q, want %d", s.Game, scenarioKey(s.Game), tt.num)
			}
		})
	}
}

// Each scenario opens with a different title sequence.
func TestScenarioTitleSequence(t *testing.T) {
	tests := []struct {
		num  int
		step TitleStep
	}{
		{1, TitleText},  // animated WT reveal after the text intro
		{2, TitleArt},   // static title bitmap
		{3, TitleStory}, // multi-frame story
	}
	for _, tt := range tests {
		s := loadScenario(t, tt.num)
		g := New(s)
		if g.Phase != PhaseTitle {
			t.Errorf("wiz%d: Phase = %v, want PhaseTitle", tt.num, g.Phase)
		}
		if g.Title.Step != tt.step {
			t.Errorf("wiz%d: title step = %v, want %v", tt.num, g.Title.Step, tt.step)
		}
	}
	if s := loadScenario(t, 1); s.TitleWT == nil || s.MonsterPics == nil {
		t.Error("wiz1 should embed the WT animation and monster pictures")
	}
	if s := loadScenario(t, 3); len(s.TitleFrames) == 0 && len(s.TitleStory) == 0 {
		t.Error("wiz3 should have title story frames or pages")
	}
}

// A digest of each fully loaded scenario. Any change to embedded game data or
// the JSON loaders shows up here.
func TestScenarioDataDigest(t *testing.T) {
	var out strings.Builder
	for num := 1; num <= 3; num++ {
		s := loadScenario(t, num)
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&out, "wiz%d %x\n", num, sha256.Sum256(raw))
	}
	checkGolden(t, "scenario_digest.golden", []byte(out.String()))
}

func TestScenarioKeyDefaultsToOne(t *testing.T) {
	if got := scenarioKey("SOMETHING ELSE"); got != "1" {
		t.Errorf("scenarioKey(unknown) = %q, want 1", got)
	}
}
