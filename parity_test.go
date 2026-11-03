package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"
)

func parityApp(t *testing.T) *App {
	t.Helper()
	dir := os.Getenv("STOCKBOOK_TEST_DATA")
	if dir == "" {
		t.Skip("set STOCKBOOK_TEST_DATA to a copied Stockbook data directory")
	}
	if err := os.Setenv("STOCKBOOK_DATA_DIR", dir); err != nil {
		t.Fatal(err)
	}
	db, path, err := openDatabase()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	var company Company
	if err := db.QueryRow(
		"SELECT id, name, email FROM companies ORDER BY id LIMIT 1",
	).Scan(&company.ID, &company.Name, &company.Email); err != nil {
		if err == sql.ErrNoRows {
			t.Skip("copied data directory has no companies")
		}
		t.Fatal(err)
	}

	app := &App{db: db, dbPath: path, attempts: map[string]*attemptEntry{}, company: &company}
	return app
}

func TestParityReport(t *testing.T) {
	app := parityApp(t)
	company := *app.company

	if os.Getenv("STOCKBOOK_TEST_SEED") == "1" {
		if _, err := app.db.Exec("DELETE FROM sessions"); err != nil {
			t.Fatal(err)
		}
		if err := app.createSession(company.ID); err != nil {
			t.Fatal(err)
		}
		t.Logf("seeded desktop session for company %d", company.ID)
	}

	overview, err := app.Overview(RangePageInput{AllTime: true})
	if err != nil {
		t.Fatal(err)
	}
	items, err := app.Items(ItemsInput{})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := app.Inventory(InventoryInput{})
	if err != nil {
		t.Fatal(err)
	}
	purchases, err := app.Purchases(RangePageInput{AllTime: true})
	if err != nil {
		t.Fatal(err)
	}
	sales, err := app.Sales(RangePageInput{AllTime: true})
	if err != nil {
		t.Fatal(err)
	}

	report := map[string]any{
		"company":   company.Name,
		"overview":  overview,
		"items":     items,
		"inventory": inventory,
		"purchases": purchases,
		"sales":     sales,
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("\n%s", encoded)

	if sales.Summary.Revenue != 10000 {
		t.Errorf("sales revenue = %d, want 10000", sales.Summary.Revenue)
	}
	if sales.Summary.Profit != 5000 {
		t.Errorf("sales profit = %d, want 5000", sales.Summary.Profit)
	}
	if inventory.Summary.Units != 19 {
		t.Errorf("inventory units = %d, want 19", inventory.Summary.Units)
	}
	if purchases.Summary.Count != 1 || purchases.Summary.Spend != 100000 {
		t.Errorf("purchases = %+v, want 1 entry / 100000 cents", purchases.Summary)
	}
	if len(items.Items) != 1 || items.Items[0].Name != "JTP BASIC EARPHONE" {
		t.Errorf("unexpected items: %+v", items.Items)
	}
}

func TestSessionRestore(t *testing.T) {
	app := parityApp(t)
	if app.token == "" {
		t.Skip("no desktop session seeded")
	}
	app.company = nil
	app.restoreSession()
	if app.company == nil || app.company.ID == 0 {
		t.Error("session did not restore")
	}
}
