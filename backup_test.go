package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func registerTestAccount(t *testing.T, app *App, email string) {
	t.Helper()
	result := app.Register(RegisterInput{
		CompanyName:     "Backup Test",
		Email:           email,
		Password:        "Password123",
		ConfirmPassword: "Password123",
	})
	if result.Error != "" || result.Company == nil {
		t.Fatalf("register: %+v", result)
	}
}

func TestBackupAndRestore(t *testing.T) {
	app := newTestApp(t)
	registerTestAccount(t, app, "backup@example.com")

	if result, err := app.CreateItem(ItemInput{
		Name:              "Original item",
		Category:          "Test",
		LowStockThreshold: "2",
		PurchasePrice:     "10",
		SellPrice:         "15",
	}); err != nil || result.Error != "" {
		t.Fatalf("create original item: %+v %v", result, err)
	}

	backupDir := t.TempDir()
	backupPath := filepath.Join(backupDir, "stockbook-backup.db")
	size, err := app.backupToFile(backupPath)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if size == 0 {
		t.Fatal("backup is empty")
	}
	if err := validateBackup(backupPath); err != nil {
		t.Fatalf("backup should be valid: %v", err)
	}

	// Change the live database after the backup.
	if result, err := app.CreateItem(ItemInput{
		Name:              "Added after backup",
		Category:          "Test",
		LowStockThreshold: "2",
		PurchasePrice:     "12",
		SellPrice:         "18",
	}); err != nil || result.Error != "" {
		t.Fatalf("create later item: %+v %v", result, err)
	}

	safety, err := app.restoreFromFile(backupPath)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := os.Stat(safety); err != nil {
		t.Fatalf("safety snapshot missing: %v", err)
	}
	if !strings.HasPrefix(filepath.Base(safety), "restore-safety-") {
		t.Errorf("safety snapshot name = %q", filepath.Base(safety))
	}

	if app.Session().Authenticated {
		t.Error("restore should sign the user out")
	}
	if _, err := os.Stat(app.sessionFilePath()); !os.IsNotExist(err) {
		t.Error("session file should be removed after restore")
	}

	login := app.Login(LoginInput{Email: "backup@example.com", Password: "Password123"})
	if login.Error != "" || login.Company == nil {
		t.Fatalf("login after restore: %+v", login)
	}

	items, err := app.Items(ItemsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items.Items) != 1 || items.Items[0].Name != "Original item" {
		t.Fatalf("items after restore = %+v, want only Original item", items.Items)
	}

	// The safety snapshot must contain the state that was replaced.
	safeDB, err := sql.Open("sqlite", safety)
	if err != nil {
		t.Fatal(err)
	}
	defer safeDB.Close()
	var found int
	if err := safeDB.QueryRow("SELECT COUNT(*) FROM products WHERE name = ?", "Added after backup").Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found != 1 {
		t.Errorf("safety snapshot does not contain the replaced data")
	}
}

func TestValidateBackupRejectsInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-backup.db")
	if err := os.WriteFile(path, []byte("this is not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateBackup(path); err == nil {
		t.Fatal("invalid backup should be rejected")
	}
}

func TestRestoreRefusesLiveDatabase(t *testing.T) {
	app := newTestApp(t)
	registerTestAccount(t, app, "same-file@example.com")

	_, err := app.restoreFromFile(app.dbPath)
	if err == nil || !strings.Contains(err.Error(), "already using") {
		t.Fatalf("restoring the live database should fail, got %v", err)
	}
}
