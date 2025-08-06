package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"log"
	"net/http"
	"os"
	rt "runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

const AppVersion = "1.0.2"

func main() {
	if rt.GOOS == "windows" {
		nullFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			log.Println("无法打开 null 文件:", err)
		} else {
			os.Stdout = nullFile
			os.Stderr = nullFile
		}
	}

	app := NewApp()

	appOptions := createAppOptions(app)

	if err := wails.Run(appOptions); err != nil {
		log.Fatalf("应用启动失败: %v", err)
	}
}

func createAppOptions(app *App) *options.App {
	return &options.App{
		Title:             "ShiroGal",
		Width:             1024,
		Height:            768,
		MinWidth:          800,
		MinHeight:         600,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		AlwaysOnTop:       false,
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: http.FileServer(http.FS(assets)),
		},

		Menu:       nil,
		Logger:     nil,
		LogLevel:   logger.DEBUG,
		OnStartup:  app.OnStartup,
		OnShutdown: app.OnShutdown,
		Bind: []interface{}{
			app,
		},
		WindowStartState: options.Normal,
		CSSDragProperty:  "--wails-draggable",
		CSSDragValue:     "drag",
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               "",
			ZoomFactor:                        1.0,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       false,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "ShiroGal",
				Message: "© 2025 ShiroGal - 版本 " + AppVersion,
			},
		},
	}
}
