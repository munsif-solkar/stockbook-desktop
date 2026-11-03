package main

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("STOCKBOOK_DATA_DIR", dir)
	t.Setenv("STOCKBOOK_DB", filepath.Join(dir, "stockbook.db"))
	db, path, err := openDatabase()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &App{db: db, dbPath: path, attempts: map[string]*attemptEntry{}}
}

func TestAccountAndLedgerFlow(t *testing.T) {
	app := newTestApp(t)

	registered := app.Register(RegisterInput{
		CompanyName:     "Test Retail",
		Email:           "owner@example.com",
		Password:        "Password123",
		ConfirmPassword: "Password123",
	})
	if registered.Error != "" || registered.Company == nil {
		t.Fatalf("register failed: %+v", registered)
	}
	if got := app.Session(); !got.Authenticated || got.Company == nil {
		t.Fatalf("session not active after register: %+v", got)
	}

	sessionFile := filepath.Join(filepath.Dir(app.dbPath), "session")
	info, err := os.Stat(sessionFile)
	if err != nil {
		t.Fatalf("session file missing: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("session file mode = %o, want 600", info.Mode().Perm())
	}

	duplicate := app.Register(RegisterInput{
		CompanyName:     "Test Retail",
		Email:           "owner@example.com",
		Password:        "Password123",
		ConfirmPassword: "Password123",
	})
	if duplicate.FieldErrors["email"] == nil {
		t.Errorf("duplicate email should be rejected: %+v", duplicate)
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
		t.Errorf("unexpected item: %+v", item)
	}

	purchase, err := app.CreatePurchase(PurchaseInput{
		ProductID:   "1",
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
		ProductID:   "1",
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
		t.Errorf("payment split wrong: %+v", sales.Summary)
	}

	blocked, err := app.DeletePurchase(IDInput{ID: "1"})
	if err != nil || blocked.Error == "" {
		t.Fatalf("purchase delete should be blocked: %+v %v", blocked, err)
	}

	removedSale, err := app.DeleteSale(IDInput{ID: "1"})
	if err != nil || removedSale.Error != "" {
		t.Fatalf("delete sale: %+v %v", removedSale, err)
	}

	inventory, err = app.Inventory(InventoryInput{})
	if err != nil || inventory.Summary.Units != 10 {
		t.Fatalf("stock not restored: %+v %v", inventory, err)
	}

	removedPurchase, err := app.DeletePurchase(IDInput{ID: "1"})
	if err != nil || removedPurchase.Error != "" {
		t.Fatalf("delete purchase: %+v %v", removedPurchase, err)
	}

	inventory, err = app.Inventory(InventoryInput{})
	if err != nil || inventory.Summary.Units != 0 {
		t.Fatalf("stock not reduced: %+v %v", inventory, err)
	}

	if err := app.Logout(); err != nil {
		t.Fatal(err)
	}
	if got := app.Session(); got.Authenticated {
		t.Errorf("session still active after logout")
	}
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Errorf("session file should be removed")
	}

	login := app.Login(LoginInput{Email: "owner@example.com", Password: "Password123"})
	if login.Error != "" || login.Company == nil {
		t.Fatalf("login failed: %+v", login)
	}
	badLogin := app.Login(LoginInput{Email: "owner@example.com", Password: "wrong-password"})
	if badLogin.Error == "" {
		t.Errorf("wrong password should fail")
	}
}

func TestValidationAndOwnership(t *testing.T) {
	app := newTestApp(t)
	app.Register(RegisterInput{
		CompanyName:     "One",
		Email:           "one@example.com",
		Password:        "Password123",
		ConfirmPassword: "Password123",
	})

	invalid, err := app.CreateItem(ItemInput{
		Name:              "",
		Category:          "",
		LowStockThreshold: "-1",
		PurchasePrice:     "abc",
		SellPrice:         "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"name", "category", "lowStockThreshold", "purchasePrice"} {
		if invalid.FieldErrors[field] == nil {
			t.Errorf("missing validation for %s: %+v", field, invalid.FieldErrors)
		}
	}

	if _, err := app.CreateItem(ItemInput{
		Name:              "Rice",
		Category:          "Grocery",
		LowStockThreshold: "2",
		PurchasePrice:     "10",
		SellPrice:         "12",
	}); err != nil {
		t.Fatal(err)
	}

	overStock, _ := app.CreateSale(SaleInput{
		ProductID:   "1",
		Quantity:    "1",
		UnitPrice:   "12",
		PaymentMode: "cash",
	})
	if overStock.FieldErrors["quantity"] == nil {
		t.Errorf("sale without stock should fail: %+v", overStock)
	}
}
