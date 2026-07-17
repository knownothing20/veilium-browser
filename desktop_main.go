package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knownothing20/veilium-browser/internal/desktop"
	"github.com/knownothing20/veilium-browser/internal/profile"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dataRoot, err := desktopDataRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	store, err := profile.Open(filepath.Join(dataRoot, "profiles.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	service, err := desktop.NewService(store, dataRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	app := NewDesktopApp(service)
	if err := wails.Run(&options.App{Title: "Veilium Browser", Width: 1440, Height: 900, MinWidth: 1120, MinHeight: 720, AssetServer: &assetserver.Options{Assets: assets}, BackgroundColour: &options.RGBA{R: 245, G: 247, B: 250, A: 1}, OnStartup: app.startup, OnShutdown: app.shutdown, Bind: []interface{}{app}}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func desktopDataRoot() (string, error) {
	if override := os.Getenv("VEILIUM_DATA_DIR"); override != "" {
		return filepath.Abs(override)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(configDir, "Veilium"), nil
}
