package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestDatabasePathWithSpecialCharacters guards the direct-path database
// opening. Building a file: URI from a native path broke Windows (drive
// letters turned into a URI authority), so paths are handed to the driver
// unchanged and must work even when they contain spaces or punctuation.
func TestDatabasePathWithSpecialCharacters(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Stock book #1 (local)", "stockbook.db")
	t.Setenv("STOCKBOOK_DATA_DIR", "")
	t.Setenv("STOCKBOOK_DB", target)

	db, path, err := openDatabase()
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if path != target {
		t.Errorf("database path = %q, want %q", path, target)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal mode = %q, want wal", journalMode)
	}

	if _, err := db.Exec("INSERT INTO companies (name, email, password_hash, created_at) VALUES (?, ?, ?, ?)",
		"Special Path Co", "special@example.com", "hash", nowISO()); err != nil {
		t.Fatalf("insert company: %v", err)
	}
}

func TestStartupBackupAndRetention(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "stockbook.db")
	t.Setenv("STOCKBOOK_DATA_DIR", "")
	t.Setenv("STOCKBOOK_DB", target)

	db, _, err := openDatabase()
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if _, err := db.Exec("INSERT INTO companies (name, email, password_hash, created_at) VALUES (?, ?, ?, ?)",
		"Backup Co", "backup@example.com", "hash", nowISO()); err != nil {
		t.Fatalf("insert company: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	// Reopening an existing ledger snapshots it before touching anything.
	db, _, err = openDatabase()
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close reopened database: %v", err)
	}

	backups, err := os.ReadDir(backupDirectory(target))
	if err != nil {
		t.Fatalf("read backups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backups))
	}

	backup, err := sql.Open("sqlite", filepath.Join(backupDirectory(target), backups[0].Name()))
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer backup.Close()
	var name string
	if err := backup.QueryRow("SELECT name FROM companies").Scan(&name); err != nil {
		t.Fatalf("read backup company: %v", err)
	}
	if name != "Backup Co" {
		t.Errorf("backup company = %q, want Backup Co", name)
	}

	for i := 0; i < 12; i++ {
		path := filepath.Join(backupDirectory(target), fmt.Sprintf("stockbook-20250101-0000%02d.000.db", i))
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pruneBackups(backupDirectory(target))

	backups, err = os.ReadDir(backupDirectory(target))
	if err != nil {
		t.Fatalf("read pruned backups: %v", err)
	}
	if len(backups) != backupRetention {
		t.Errorf("backup count after prune = %d, want %d", len(backups), backupRetention)
	}
}
