package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// BackupDatabase asks for a destination and writes a consistent snapshot of
// the entire local database there. The snapshot is an ordinary SQLite file,
// so it contains every account, item, purchase, and sale.
func (a *App) BackupDatabase() (BackupResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return BackupResult{}, err
	}
	if a.ctx == nil {
		return BackupResult{Error: "The app window is not ready yet. Try again in a moment."}, nil
	}

	a.opMu.Lock()
	defer a.opMu.Unlock()

	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:                "Save Stockbook backup",
		DefaultFilename:      "stockbook-backup-" + time.Now().Format("2006-01-02-150405") + ".db",
		CanCreateDirectories: true,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Stockbook backup (*.db)", Pattern: "*.db"},
		},
	})
	if err != nil {
		return BackupResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return BackupResult{Canceled: true}, nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".db") {
		path += ".db"
	}

	size, err := a.backupToFile(path)
	if err != nil {
		return BackupResult{Error: friendlyBackupError(err)}, nil
	}
	return BackupResult{Path: path, Size: size}, nil
}

func friendlyBackupError(err error) string {
	if errors.Is(err, os.ErrPermission) {
		return "Stockbook could not write to that folder. Choose another location."
	}
	return "Could not create the backup: " + err.Error()
}

// backupToFile writes a consistent copy of the live database to target and
// returns its size. It is also used by tests.
func (a *App) backupToFile(target string) (int64, error) {
	if a.db == nil {
		return 0, errors.New("database unavailable")
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, err
	}

	// VACUUM INTO refuses to overwrite, so snapshot to a unique temporary
	// file in the destination folder and move it into place afterwards.
	temp, err := os.CreateTemp(dir, ".stockbook-backup-*.db")
	if err != nil {
		return 0, err
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return 0, err
	}
	if err := os.Remove(tempPath); err != nil {
		return 0, err
	}
	defer os.Remove(tempPath)

	if _, err := a.db.Exec("VACUUM INTO ?", tempPath); err != nil {
		return 0, err
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		return 0, err
	}
	if err := replaceFile(tempPath, target); err != nil {
		return 0, err
	}

	info, err := os.Stat(target)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// replaceFile moves source over target. Windows will not rename over an
// existing file, so the destination is removed after the source is ready.
func replaceFile(source, target string) error {
	if runtime.GOOS == "windows" {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return os.Rename(source, target)
}

// RestoreDatabase asks for a Stockbook backup and replaces the local database
// with it. The current database is snapshotted first, so a bad restore never
// destroys the only copy.
func (a *App) RestoreDatabase() (RestoreResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return RestoreResult{}, err
	}
	if a.ctx == nil {
		return RestoreResult{Error: "The app window is not ready yet. Try again in a moment."}, nil
	}

	a.opMu.Lock()
	defer a.opMu.Unlock()

	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Choose a Stockbook backup",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Stockbook backup (*.db;*.sqlite;*.sqlite3)", Pattern: "*.db;*.sqlite;*.sqlite3"},
		},
	})
	if err != nil {
		return RestoreResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return RestoreResult{Canceled: true}, nil
	}

	safety, err := a.restoreFromFile(path)
	if err != nil {
		return RestoreResult{Error: friendlyRestoreError(err)}, nil
	}
	return RestoreResult{Restored: true, SafetyBackup: safety}, nil
}

func friendlyRestoreError(err error) string {
	switch {
	case errors.Is(err, os.ErrPermission):
		return "Stockbook could not replace its database. Close any other running copy of Stockbook and try again."
	case strings.Contains(err.Error(), "not a Stockbook backup"):
		return err.Error()
	default:
		return "Could not restore the backup: " + err.Error()
	}
}

// restoreFromFile validates source, snapshots the current database, swaps the
// files atomically, and reopens the ledger. It returns the safety snapshot
// path. The caller must hold opMu.
func (a *App) restoreFromFile(source string) (string, error) {
	if a.db == nil {
		return "", errors.New("database unavailable")
	}
	if sameFile(source, a.dbPath) {
		return "", errors.New("That is the database Stockbook is already using.")
	}

	// Work on a copy so validating a backup never changes the file the user
	// selected (SQLite can create journal files during a read).
	temp, err := os.CreateTemp(filepath.Dir(a.dbPath), ".stockbook-restore-*.db")
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	defer os.Remove(tempPath)

	if err := copyFile(source, tempPath); err != nil {
		return "", err
	}
	if err := validateBackup(tempPath); err != nil {
		return "", err
	}

	safety, err := a.writeSafetySnapshot()
	if err != nil {
		return "", fmt.Errorf("create safety snapshot: %w", err)
	}

	// Close the live connection before touching the files. WAL and SHM files
	// must go too, otherwise stale pages could be replayed into the restored
	// database.
	if err := a.db.Close(); err != nil {
		return "", err
	}
	a.db = nil
	_ = os.Remove(a.dbPath + "-wal")
	_ = os.Remove(a.dbPath + "-shm")

	replacing := a.dbPath + ".replacing"
	_ = os.Remove(replacing)
	if err := os.Rename(a.dbPath, replacing); err != nil {
		// The old database is untouched; simply reopen it.
		if reopened, _, openErr := openDatabase(); openErr == nil {
			a.db = reopened
		}
		return "", err
	}
	if err := os.Rename(tempPath, a.dbPath); err != nil {
		_ = os.Rename(replacing, a.dbPath)
		if reopened, _, openErr := openDatabase(); openErr == nil {
			a.db = reopened
		}
		return "", err
	}

	reopened, _, err := openDatabase()
	if err != nil {
		// Roll back to the known-good database and keep the safety snapshot.
		_ = os.Remove(a.dbPath)
		_ = os.Rename(replacing, a.dbPath)
		if old, _, openErr := openDatabase(); openErr == nil {
			a.db = old
		}
		return "", fmt.Errorf("reopen restored database: %w", err)
	}
	a.db = reopened
	_ = os.Remove(replacing)

	// A restored database has its own accounts and sessions. Force a fresh
	// sign-in rather than trusting the token from the previous database.
	_ = os.Remove(a.sessionFilePath())
	a.mu.Lock()
	a.token = ""
	a.company = nil
	a.mu.Unlock()

	return safety, nil
}

// writeSafetySnapshot keeps a pre-restore copy outside the normal rolling
// backup retention, so a mistaken restore can always be undone.
func (a *App) writeSafetySnapshot() (string, error) {
	dir := backupDirectory(a.dbPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(dir, "restore-safety-*.db")
	if err != nil {
		return "", err
	}
	target := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(target)
		return "", err
	}
	if err := os.Remove(target); err != nil {
		return "", err
	}
	if _, err := a.db.Exec("VACUUM INTO ?", target); err != nil {
		_ = os.Remove(target)
		return "", err
	}
	_ = os.Chmod(target, 0o600)
	return target, nil
}

func validateBackup(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return errors.New("not a Stockbook backup: the file could not be opened")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	var integrity string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		return errors.New("not a Stockbook backup: the file is not a valid SQLite database")
	}
	if integrity != "ok" {
		return errors.New("not a Stockbook backup: the database failed its integrity check")
	}

	for _, table := range []string{"companies", "sessions", "products", "purchases", "sales"} {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table,
		).Scan(&name)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("not a Stockbook backup: the %s table is missing", table)
		}
		if err != nil {
			return err
		}
	}

	var companies int
	if err := db.QueryRow("SELECT COUNT(*) FROM companies").Scan(&companies); err != nil {
		return errors.New("not a Stockbook backup: the accounts table could not be read")
	}
	if companies == 0 {
		return errors.New("not a Stockbook backup: it contains no accounts")
	}

	// Touch the tables the app reads on every screen so a superficially valid
	// SQLite file with the wrong columns is rejected before it replaces data.
	if _, err := db.Exec(`SELECT id, company_id, name, category, quantity, created_at FROM products LIMIT 1`); err != nil {
		return errors.New("not a Stockbook backup: the items table has an unexpected structure")
	}
	if _, err := db.Exec(`SELECT id, company_id, product_id, quantity, total_price, sold_at FROM sales LIMIT 1`); err != nil {
		return errors.New("not a Stockbook backup: the sales table has an unexpected structure")
	}
	if _, err := db.Exec(`SELECT id, company_id, product_id, quantity, total_cost, purchased_at FROM purchases LIMIT 1`); err != nil {
		return errors.New("not a Stockbook backup: the purchases table has an unexpected structure")
	}
	return nil
}

func sameFile(a, b string) bool {
	left := filepath.Clean(a)
	right := filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
