package main

import (
	"context"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	api     *supabaseAPI
	dataDir string
	initErr error

	// opMu serialises maintenance operations that may replace a whole
	// workspace (backup, restore, update).
	opMu sync.Mutex

	mu       sync.Mutex
	company  *Company
	attempts map[string]*attemptEntry
}

func NewApp() *App {
	app := &App{attempts: map[string]*attemptEntry{}}

	dir, err := dataDirectory()
	if err != nil {
		app.initErr = err
		return app
	}
	app.dataDir = dir

	api, err := newSupabaseAPI(readBackendConfig(dir))
	if err != nil {
		app.initErr = err
		return app
	}
	app.api = api
	api.onTokens = app.writeSessionFile

	// Windows keeps the previous executable as a .old file because it cannot
	// be deleted while it is running. Remove it after the next launch.
	if executable, err := os.Executable(); err == nil {
		_ = os.Remove(executable + ".old")
	}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.initErr != nil {
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Stockbook",
			Message: "Stockbook could not start its Supabase connection.\n\n" + a.initErr.Error(),
		})
		return
	}
	a.restoreSession()
}

func (a *App) shutdown(context.Context) {
	if a.api != nil {
		a.api.client.CloseIdleConnections()
	}
}

// DataLocation is shown in the About area of the app and keeps the backend
// location discoverable without exposing anything secret.
func (a *App) DataLocation() string {
	if a.api == nil {
		return "Unavailable"
	}
	return a.api.baseURL
}
