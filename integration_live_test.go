package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func idString(id int64) string {
	return strconv.FormatInt(id, 10)
}

// TestLiveSupabaseFlow exercises the real Supabase backend. It is skipped
// unless STOCKBOOK_TEST_SUPABASE=1 is set, because it creates an auth user and
// writes rows to the configured project.
//
//	STOCKBOOK_TEST_SUPABASE=1 STOCKBOOK_TEST_EMAIL=you+stockbook-test@gmail.com \
//	  go test -run Live
func TestLiveSupabaseFlow(t *testing.T) {
	if os.Getenv("STOCKBOOK_TEST_SUPABASE") != "1" {
		t.Skip("set STOCKBOOK_TEST_SUPABASE=1 to run against a live Supabase project")
	}

	email := os.Getenv("STOCKBOOK_TEST_EMAIL")
	if email == "" {
		t.Fatal("set STOCKBOOK_TEST_EMAIL to an email address you control; the test registers this account")
	}

	dir := t.TempDir()
	t.Setenv("STOCKBOOK_DATA_DIR", dir)

	app := NewApp()
	if app.initErr != nil {
		t.Fatalf("start app: %v", app.initErr)
	}

	registered := app.Register(RegisterInput{
		CompanyName:     "Test Retail",
		Email:           email,
		Password:        "Password123",
		ConfirmPassword: "Password123",
	})
	if registered.Error != "" || registered.Company == nil {
		t.Fatalf("register: %+v", registered)
	}
	if !app.Session().Authenticated {
		t.Fatal("session should be active after register")
	}

	created, err := app.CreateItem(ItemInput{
		Name:              "Tea",
		SKU:               "TEA-1",
		Category:          "Beverages",
		LowStockThreshold: "5",
		PurchasePrice:     "50.00",
		SellPrice:         "80.00",
	})
	if err != nil || created.Error != "" || created.FieldErrors != nil {
		t.Fatalf("create item: %+v %v", created, err)
	}

	items, err := app.Items(ItemsInput{})
	if err != nil || len(items.Items) != 1 {
		t.Fatalf("items list: %+v %v", items, err)
	}
	item := items.Items[0]
	if item.Name != "Tea" || item.SKU == nil || *item.SKU != "TEA-1" {
		t.Fatalf("unexpected item: %+v", item)
	}

	purchase, err := app.CreatePurchase(PurchaseInput{
		ProductID:   idString(item.ID),
		Quantity:    "10",
		UnitCost:    "50.00",
		PurchasedAt: todayISO(),
	})
	if err != nil || purchase.Error != "" {
		t.Fatalf("create purchase: %+v %v", purchase, err)
	}

	inventory, err := app.Inventory(InventoryInput{})
	if err != nil || inventory.Summary.Units != 10 {
		t.Fatalf("inventory after purchase: %+v %v", inventory, err)
	}

	sale, err := app.CreateSale(SaleInput{
		ProductID:   idString(item.ID),
		Quantity:    "2",
		UnitPrice:   "80.00",
		PaymentMode: "upi",
	})
	if err != nil || sale.Error != "" {
		t.Fatalf("create sale: %+v %v", sale, err)
	}

	sales, err := app.Sales(RangePageInput{AllTime: true})
	if err != nil || sales.Summary.Revenue != 16000 || sales.Summary.Profit != 6000 {
		t.Fatalf("sales summary: %+v %v", sales.Summary, err)
	}
	if sales.Summary.UPIRevenue != 16000 || sales.Summary.CashRevenue != 0 {
		t.Fatalf("payment split wrong: %+v", sales.Summary)
	}

	overview, err := app.Overview(RangePageInput{AllTime: true})
	if err != nil || overview.Sales.Revenue != 16000 {
		t.Fatalf("overview: %+v %v", overview, err)
	}

	// A second app instance must pick up the saved Supabase session.
	reopened := NewApp()
	reopened.restoreSession()
	if !reopened.Session().Authenticated {
		t.Fatal("saved session should be restored")
	}

	blocked, err := app.DeletePurchase(IDInput{ID: idString(purchase.PurchaseID)})
	if err != nil || blocked.Error == "" {
		t.Fatalf("purchase delete should be blocked: %+v %v", blocked, err)
	}

	removedSale, err := app.DeleteSale(IDInput{ID: idString(sale.SaleID)})
	if err != nil || removedSale.Error != "" {
		t.Fatalf("delete sale: %+v %v", removedSale, err)
	}

	inventory, err = app.Inventory(InventoryInput{})
	if err != nil || inventory.Summary.Units != 10 {
		t.Fatalf("stock not restored: %+v %v", inventory, err)
	}

	removedPurchase, err := app.DeletePurchase(IDInput{ID: idString(purchase.PurchaseID)})
	if err != nil || removedPurchase.Error != "" {
		t.Fatalf("delete purchase: %+v %v", removedPurchase, err)
	}

	inventory, err = app.Inventory(InventoryInput{})
	if err != nil || inventory.Summary.Units != 0 {
		t.Fatalf("stock not reduced: %+v %v", inventory, err)
	}

	backupPath := filepath.Join(dir, "backup.json")
	backupSize, err := app.backupToFile(backupPath)
	if err != nil || backupSize == 0 {
		t.Fatalf("backup: %d %v", backupSize, err)
	}

	if _, err := app.CreateItem(ItemInput{
		Name:              "Coffee",
		Category:          "Beverages",
		LowStockThreshold: "5",
		PurchasePrice:     "100.00",
		SellPrice:         "150.00",
	}); err != nil {
		t.Fatalf("create second item: %v", err)
	}

	safety, err := app.restoreFromFile(backupPath)
	if err != nil || safety == "" {
		t.Fatalf("restore: %q %v", safety, err)
	}
	if app.Session().Authenticated {
		t.Fatal("restore should sign out")
	}

	// The session file must be gone after a restore.
	if _, err := os.Stat(filepath.Join(dir, sessionFileName)); !os.IsNotExist(err) {
		t.Fatalf("session file should be removed, got %v", err)
	}
}
