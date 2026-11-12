// Command Istary is the Wails desktop front end: Wizardry I's town,
// expeditions, Camp and combat, played through app.Controller.
package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// guiSaveDir is where the GUI keeps its games (games.go): $ISTARY_SAVE_DIR
// if set, otherwise ~/Library/Application Support/Istary/dev-saves, the
// folder that held the single save of earlier versions. It is never the
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
	// The save folder is opened when the window starts (OnStartup), not
	// here: Wails also runs this program to generate bindings.
	session := unopenedLibrary(guiSaveDir())

	err := wails.Run(&options.App{
		Title:     "Istary",
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
		// Let go of the save folder's lock on quitting (the system also
		// releases it if the app ends any other way).
		OnStartup: func(ctx context.Context) {
			session.setQuit(func() { runtime.Quit(ctx) })
			session.start()
		},
		// Closing the window or quitting asks first if the last save
		// failed (close.go).
		OnBeforeClose: func(ctx context.Context) bool {
			return session.beforeClose(func(message string) (string, error) {
				return runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
					Type:          runtime.WarningDialog,
					Title:         closeTitle,
					Message:       message,
					Buttons:       []string{closeKeepPlaying, closeSaveAgain, closeDiscard},
					DefaultButton: closeKeepPlaying,
					CancelButton:  closeKeepPlaying,
				})
			})
		},
		OnShutdown: func(context.Context) { session.shutdown() },
	})
	if err != nil {
		log.Fatal(err)
	}
}
