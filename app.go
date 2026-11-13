package main

import (
	"context"
	"os"
	"sync"
)

type App struct {
	ctx     context.Context
	dataDir string

	// backendMu guards the active backend and its initialization error. The
	// cloud client and the local SQLite store are both kept open once created
	// so switching between backends does not wait for files or connections.
	backendMu sync.RWMutex
	mode      backendMode
	api       *supabaseAPI
	local     *localStore
	initErr   error

	// opMu serialises maintenance operations that may replace a whole
	// workspace (export, import, update).
	opMu sync.Mutex

	mu         sync.Mutex
	company    *Company
	localToken string
	attempts   map[string]*attemptEntry
}

func NewApp() *App {
	app := &App{attempts: map[string]*attemptEntry{}}

	dir, err := dataDirectory()
	if err != nil {
		app.setInitErr(err)
		return app
	}
	app.dataDir = dir

	mode, modeErr := loadBackendMode(dir)
	app.mode = mode

	cloudConfig, configErr := cloudBackendConfig()
	api, apiErr := newSupabaseAPI(cloudConfig)
	if configErr != nil && mode == modeCloud {
		apiErr = configErr
	}
	if apiErr == nil {
		api.onTokens = app.writeSessionFile
		app.api = api
	} else if mode == modeCloud {
		app.setInitErr(apiErr)
	}

	if mode == modeLocal {
		store, localErr := openLocalStore()
		if localErr != nil {
			app.setInitErr(localErr)
		} else {
			app.local = store
		}
	}

	if modeErr != nil {
		app.setInitErr(modeErr)
	}

	// Windows keeps the previous executable as a .old file because it cannot
	// be deleted while it is running. Remove it after the next launch.
	if executable, err := os.Executable(); err == nil {
		_ = os.Remove(executable + ".old")
	}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.currentInitErr() != nil {
		// The sign-in screen shows the error and lets the user choose the
		// other backend, so no blocking native dialog is needed.
		return
	}
	a.restoreSession()
}

func (a *App) shutdown(context.Context) {
	a.backendMu.Lock()
	api := a.api
	local := a.local
	a.backendMu.Unlock()

	if api != nil {
		api.client.CloseIdleConnections()
	}
	if local != nil {
		local.close()
	}
}

// DataLocation is the folder that holds the local ledger, sessions, safety
// copies, and any exported files. It is shown on the Export/import data page.
func (a *App) DataLocation() string {
	if a.dataDir == "" {
		return "Unavailable"
	}
	return a.dataDir
}
