package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"devclip/internal/platform"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Single-instance guard: prevent multiple copies of DevClip running.
	isWails := false
	for _, arg := range os.Args {
		if strings.Contains(strings.ToLower(arg), "wails") {
			isWails = true
			break
		}
	}
	if !isWails {
		release, err := platform.EnsureSingleInstance()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(0)
		}
		defer release()
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "DevClip",
		Width:         360,
		Height:        480,
		Frameless:     true,
		StartHidden:   true,
		AlwaysOnTop:   true,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Acrylic,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// Minimize to tray instead of quitting when user clicks close.
			wailsRuntime.WindowHide(ctx)
			return true
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
