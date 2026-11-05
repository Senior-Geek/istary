// Command Istary is the Wails desktop front end. This first prototype covers
// Wizardry I character creation only.
package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

// guiSaveDir is where the GUI keeps its roster: $ISTARY_SAVE_DIR if set,
// otherwise ~/Library/Application Support/Istary/dev-saves. It is never the
// terminal game's ~/.config/wizardry.
func guiSaveDir() string {
	if dir := os.Getenv("ISTARY_SAVE_DIR"); dir != "" {
		return dir
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "Istary", "dev-saves")
}

func main() {
	session := NewSession(guiSaveDir())

	err := wails.Run(&options.App{
		Title:     "Istary — Character Creation",
		Width:     980,
		Height:    720,
		MinWidth:  760,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 18, A: 1},
		// Wails v2 only reads the zoom setting from options.Mac; with Mac
		// unset it disables the green (zoom/fullscreen) button. An empty
		// mac.Options keeps zoom enabled, so the window can maximise and
		// enter native fullscreen.
		Mac:  &mac.Options{},
		Bind: []interface{}{session},
	})
	if err != nil {
		log.Fatal(err)
	}
}
