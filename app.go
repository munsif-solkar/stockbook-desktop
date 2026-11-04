package main

import (
	"context"
	"database/sql"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	db     *sql.DB
	dbPath string
	dbErr  error

	// opMu serialises maintenance operations that close or replace the
	// database (backup, restore, update).
	opMu sync.Mutex

	mu       sync.Mutex
	company  *Company
	token    string
	attempts map[string]*attemptEntry
}

func NewApp() *App {
	app := &App{attempts: map[string]*attemptEntry{}}
	db, path, err := openDatabase()
	app.db = db
	app.dbPath = path
	app.dbErr = err
	if err == nil && path != "" {
		// A leftover copy from an interrupted restore is no longer needed
		// once the active database opens cleanly.
		_ = os.Remove(path + ".replacing")
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
	if a.dbErr != nil {
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Stockbook",
			Message: "Stockbook could not open its local database.\n\n" + a.dbErr.Error(),
		})
		return
	}
	a.restoreSession()
}

func (a *App) shutdown(context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
}

// DataLocation is shown in the About area of the app and keeps the local-first
// storage location discoverable without exposing it in regular navigation.
func (a *App) DataLocation() string {
	if a.dbPath == "" {
		return "Unavailable"
	}
	return a.dbPath
}
