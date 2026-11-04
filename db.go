package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS companies (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL COLLATE NOCASE UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_company_idx ON sessions(company_id);
CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  sku TEXT,
  category TEXT NOT NULL,
  purchase_price INTEGER NOT NULL CHECK (purchase_price >= 0),
  sell_price INTEGER NOT NULL CHECK (sell_price >= 0),
  quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  low_stock_threshold INTEGER NOT NULL DEFAULT 5 CHECK (low_stock_threshold >= 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS products_company_sku_idx
  ON products(company_id, sku) WHERE sku IS NOT NULL AND sku <> '';
CREATE INDEX IF NOT EXISTS products_company_name_idx ON products(company_id, name);
CREATE INDEX IF NOT EXISTS products_company_category_idx ON products(company_id, category);

CREATE TABLE IF NOT EXISTS purchases (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_cost INTEGER NOT NULL CHECK (unit_cost >= 0),
  total_cost INTEGER NOT NULL CHECK (total_cost >= 0),
  purchased_at TEXT NOT NULL,
  note TEXT,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS purchases_company_date_idx
  ON purchases(company_id, purchased_at);
CREATE INDEX IF NOT EXISTS purchases_company_product_idx
  ON purchases(company_id, product_id);

CREATE TABLE IF NOT EXISTS sales (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
  total_price INTEGER NOT NULL CHECK (total_price >= 0),
  unit_cost INTEGER NOT NULL CHECK (unit_cost >= 0),
  total_cost INTEGER NOT NULL CHECK (total_cost >= 0),
  profit INTEGER NOT NULL,
  payment_mode TEXT NOT NULL DEFAULT 'cash' CHECK (payment_mode IN ('cash', 'upi')),
  sold_at TEXT NOT NULL,
  note TEXT,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS sales_company_date_idx ON sales(company_id, sold_at);
CREATE INDEX IF NOT EXISTS sales_company_product_idx ON sales(company_id, product_id);
`

func dataDirectory() (string, error) {
	if configured := os.Getenv("STOCKBOOK_DATA_DIR"); configured != "" {
		return filepath.Abs(configured)
	}
	return defaultDataDirectory()
}

func databasePath() (string, error) {
	if configured := os.Getenv("STOCKBOOK_DB"); configured != "" {
		return filepath.Abs(configured)
	}
	dir, err := dataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "stockbook.db"), nil
}

// copyFile copies src to dst with owner-only permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// importLegacyDatabase copies a stockbook.db that lives beside the old web app
// into the desktop data directory the first time the desktop app starts.
func importLegacyDatabase(target string) {
	if _, err := os.Stat(target); err == nil {
		return
	}
	cwd, _ := os.Getwd()
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(cwd, "data", "stockbook.db"),
		filepath.Join(exeDir, "data", "stockbook.db"),
		filepath.Join(exeDir, "..", "data", "stockbook.db"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		if err := copyFile(candidate, target); err != nil {
			return
		}
		// Keep the write-ahead log so the most recent committed data survives.
		for _, suffix := range []string{"-wal", "-shm"} {
			if _, err := os.Stat(candidate + suffix); err == nil {
				_ = copyFile(candidate+suffix, target+suffix)
			}
		}
		return
	}
}

const backupRetention = 10

func backupDirectory(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), "backups")
}

// backupDatabase writes a consistent copy of the database next to it. Startup
// snapshots mean a bad upgrade, a mistaken deletion, or a corrupted file never
// takes the only copy with it. Any failure is non-fatal: opening the ledger is
// more important than making a backup.
func backupDatabase(db *sql.DB, dbPath string) {
	var companies int
	if err := db.QueryRow("SELECT COUNT(*) FROM companies").Scan(&companies); err != nil || companies == 0 {
		return
	}

	dir := backupDirectory(dbPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}

	name := "stockbook-" + time.Now().Format("20060102-150405.000") + ".db"
	target := filepath.Join(dir, name)
	if _, err := db.Exec("VACUUM INTO ?", target); err != nil {
		return
	}
	_ = os.Chmod(target, 0o600)
	pruneBackups(dir)
}

func pruneBackups(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "stockbook-") || !strings.HasSuffix(name, ".db") {
			continue
		}
		names = append(names, name)
	}

	// Timestamped names sort oldest first.
	sort.Strings(names)
	for len(names) > backupRetention {
		_ = os.Remove(filepath.Join(dir, names[0]))
		names = names[1:]
	}
}

func openDatabase() (*sql.DB, string, error) {
	path, err := databasePath()
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, "", err
	}

	// If a restore was interrupted between moving the old database aside and
	// installing the new one, put the old database back before opening.
	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		if _, oldErr := os.Stat(path + ".replacing"); oldErr == nil {
			_ = os.Rename(path+".replacing", path)
		}
	}

	importLegacyDatabase(path)

	// Pass the native file name straight to the driver. Building a file: URI
	// here used to produce file://C:%5C... on Windows, which SQLite treated as
	// an empty path (reported as the misleading "out of memory" CANTOPEN
	// error). A plain path is accepted on every platform.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, "", err
	}
	// A single connection keeps writes serialised and PRAGMAs consistent.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		// FULL fsyncs every commit, so a power cut cannot lose an
		// acknowledged purchase or sale. The write volume here is tiny.
		"PRAGMA synchronous = FULL",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, "", fmt.Errorf("configure database %s: %w", path, err)
		}
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("create schema: %w", err)
	}

	// Snapshot on startup before any migration runs. Backups live beside the
	// database, so they survive replacing the executable or installing a new
	// build. Failures never stop the app from opening.
	backupDatabase(db, path)

	needsPaymentMode, err := columnMissing(db, "sales", "payment_mode")
	if err != nil {
		db.Close()
		return nil, "", err
	}
	if needsPaymentMode {
		if _, err := db.Exec(`ALTER TABLE sales ADD COLUMN payment_mode TEXT NOT NULL DEFAULT 'cash' CHECK (payment_mode IN ('cash', 'upi'))`); err != nil {
			db.Close()
			return nil, "", fmt.Errorf("add payment mode: %w", err)
		}
	}

	if _, err := db.Exec("DELETE FROM sessions WHERE expires_at <= ?", jsISO(time.Now())); err != nil {
		db.Close()
		return nil, "", err
	}

	if err := os.Chmod(path, 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Some filesystems do not support chmod; the app still runs.
		_ = err
	}
	return db, path, nil
}

func columnMissing(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var cid int64
		var name, dataType string
		var notNull, pk int64
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			found = true
		}
	}
	return !found, rows.Err()
}
