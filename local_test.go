package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newLocalTestApp opens a throwaway SQLite ledger in a temporary folder so
// tests never touch the real Stockbook data directory.
func newLocalTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("STOCKBOOK_DB", filepath.Join(dir, "stockbook.db"))
	store, err := openLocalStore()
	if err != nil {
		t.Fatalf("open local store: %v", err)
	}
	t.Cleanup(store.close)
	return &App{
		dataDir:  dir,
		mode:     modeLocal,
		local:    store,
		attempts: map[string]*attemptEntry{},
	}
}

func registerLocal(t *testing.T, app *App, name, email, password string) *Company {
	t.Helper()
	result := app.localRegister(RegisterInput{
		CompanyName:     name,
		Email:           email,
		Password:        password,
		ConfirmPassword: password,
	})
	if result.Error != "" || result.Company == nil {
		t.Fatalf("register %s: %+v", email, result)
	}
	return result.Company
}

func createItem(t *testing.T, app *App, name, sku string) int64 {
	t.Helper()
	result, err := app.localCreateItem(ItemInput{
		Name:              name,
		SKU:               sku,
		Category:          "Beverages",
		LowStockThreshold: "2",
		PurchasePrice:     "1.00",
		SellPrice:         "1.50",
	})
	if err != nil || result.Error != "" || result.FieldErrors != nil {
		t.Fatalf("create item: %+v %v", result, err)
	}
	items, err := app.localItems(ItemsInput{})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	for _, item := range items.Items {
		if item.Name == name {
			return item.ID
		}
	}
	t.Fatalf("created item %q not found", name)
	return 0
}

func TestLocalWorkspaceLifecycle(t *testing.T) {
	app := newLocalTestApp(t)
	company := registerLocal(t, app, "Corner Shop", "owner@example.com", "secret123")
	if company.Name != "Corner Shop" {
		t.Fatalf("company = %+v", company)
	}

	// A local session is written beside the database and survives a restart.
	// Clearing only the in-memory copy simulates opening the app again.
	app.mu.Lock()
	app.company = nil
	app.localToken = ""
	app.mu.Unlock()
	app.restoreSession()
	restored := app.currentCompany()
	if restored == nil || restored.Email != "owner@example.com" {
		t.Fatalf("restored session = %+v", restored)
	}

	itemID := createItem(t, app, "Tea", "TEA-1")

	purchase, err := app.localCreatePurchase(PurchaseInput{
		ProductID:   idString(itemID),
		Quantity:    "10",
		UnitCost:    "1.00",
		PurchasedAt: todayISO(),
		Note:        "Opening stock",
	})
	if err != nil || purchase.Error != "" {
		t.Fatalf("create purchase: %+v %v", purchase, err)
	}

	sale, err := app.localCreateSale(SaleInput{
		ProductID:   idString(itemID),
		Quantity:    "3",
		UnitPrice:   "1.50",
		PaymentMode: "cash",
	})
	if err != nil || sale.Error != "" {
		t.Fatalf("create sale: %+v %v", sale, err)
	}

	items, err := app.localItems(ItemsInput{})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items.Items) != 1 {
		t.Fatalf("items = %+v", items.Items)
	}
	if items.Items[0].Quantity != 7 {
		t.Fatalf("quantity = %d, want 7", items.Items[0].Quantity)
	}

	sales, err := app.localSales(RangePageInput{AllTime: true})
	if err != nil {
		t.Fatalf("list sales: %v", err)
	}
	if len(sales.Rows) != 1 {
		t.Fatalf("sales = %+v", sales.Rows)
	}
	if sales.Rows[0].TotalPrice != 450 || sales.Rows[0].Profit != 150 {
		t.Fatalf("sale totals = %+v", sales.Rows[0])
	}

	purchases, err := app.localPurchases(RangePageInput{AllTime: true})
	if err != nil {
		t.Fatalf("list purchases: %v", err)
	}
	if len(purchases.Rows) != 1 || purchases.Rows[0].TotalCost != 1000 {
		t.Fatalf("purchases = %+v", purchases.Rows)
	}
}

func TestLocalStartupBackupWritesSnapshot(t *testing.T) {
	app := newLocalTestApp(t)
	registerLocal(t, app, "Corner Shop", "owner@example.com", "secret123")

	backupDatabase(app.local.db, app.local.path)

	entries, err := os.ReadDir(backupDirectory(app.local.path))
	if err != nil {
		t.Fatalf("read backups: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("backups = %d, want 1", len(entries))
	}
	info, err := os.Stat(filepath.Join(backupDirectory(app.local.path), entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("startup snapshot is empty")
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("backup permissions = %v, want owner-only", info.Mode().Perm())
	}
}

func TestLocalLoginFailureAndSuccess(t *testing.T) {
	app := newLocalTestApp(t)
	registerLocal(t, app, "Corner Shop", "owner@example.com", "secret123")
	app.destroyLocalSession()

	wrong := app.localLogin(LoginInput{Email: "owner@example.com", Password: "wrongpass1"})
	if wrong.Error == "" {
		t.Fatalf("wrong password was accepted: %+v", wrong)
	}

	right := app.localLogin(LoginInput{Email: "owner@example.com", Password: "secret123"})
	if right.Error != "" || right.Company == nil {
		t.Fatalf("login: %+v", right)
	}
	if right.Company.Email != "owner@example.com" {
		t.Fatalf("company = %+v", right.Company)
	}
}

func TestLocalExportImportAcrossAccounts(t *testing.T) {
	app := newLocalTestApp(t)
	registerLocal(t, app, "Corner Shop", "owner@example.com", "secret123")
	itemID := createItem(t, app, "Tea", "TEA-1")
	if r, err := app.localCreatePurchase(PurchaseInput{
		ProductID:   idString(itemID),
		Quantity:    "10",
		UnitCost:    "1.00",
		PurchasedAt: todayISO(),
	}); err != nil || r.Error != "" {
		t.Fatalf("purchase: %+v %v", r, err)
	}
	if r, err := app.localCreateSale(SaleInput{
		ProductID:   idString(itemID),
		Quantity:    "3",
		UnitPrice:   "1.50",
		PaymentMode: "upi",
	}); err != nil || r.Error != "" {
		t.Fatalf("sale: %+v %v", r, err)
	}

	exportPath := filepath.Join(app.dataDir, "export.json")
	if _, err := app.exportToFile(exportPath); err != nil {
		t.Fatalf("export: %v", err)
	}

	raw, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"company", "accounts", "email", "password"} {
		if _, present := envelope[forbidden]; present {
			t.Fatalf("export contains %q: %s", forbidden, raw)
		}
	}

	// A different account on the same computer imports the file. This is the
	// same shape a Stockbook Cloud account receives.
	app.destroyLocalSession()
	registerLocal(t, app, "Second Shop", "second@example.com", "secret456")

	safety, err := app.importFromFile(exportPath)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if safety == "" {
		t.Fatal("import did not write a safety copy")
	}
	if _, err := os.Stat(safety); err != nil {
		t.Fatalf("safety copy missing: %v", err)
	}

	items, err := app.localItems(ItemsInput{})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items.Items) != 1 || items.Items[0].Name != "Tea" || items.Items[0].Quantity != 7 {
		t.Fatalf("imported items = %+v", items.Items)
	}
	if items.Items[0].ID == itemID {
		t.Fatalf("import did not remap the item id: %d", items.Items[0].ID)
	}

	purchases, err := app.localPurchases(RangePageInput{AllTime: true})
	if err != nil {
		t.Fatalf("list purchases: %v", err)
	}
	sales, err := app.localSales(RangePageInput{AllTime: true})
	if err != nil {
		t.Fatalf("list sales: %v", err)
	}
	if len(purchases.Rows) != 1 || len(sales.Rows) != 1 {
		t.Fatalf("imported purchases=%d sales=%d", len(purchases.Rows), len(sales.Rows))
	}
	if sales.Rows[0].PaymentMode != "upi" || sales.Rows[0].TotalPrice != 450 {
		t.Fatalf("imported sale = %+v", sales.Rows[0])
	}
}

func TestLocalImportRollsBackOnFailure(t *testing.T) {
	app := newLocalTestApp(t)
	registerLocal(t, app, "Corner Shop", "owner@example.com", "secret123")
	itemID := createItem(t, app, "Tea", "TEA-1")

	before, err := app.localItems(ItemsInput{})
	if err != nil {
		t.Fatal(err)
	}

	// The product is inserted and then the purchase fails, so the whole
	// transaction must roll back and leave the original item untouched.
	bad := &cleanData{
		Products: []cleanProduct{{
			ID:                1,
			Name:              "Coffee",
			Category:          "Beverages",
			PurchasePrice:     200,
			SellPrice:         300,
			Quantity:          5,
			LowStockThreshold: 2,
			CreatedAt:         nowISO(),
			UpdatedAt:         nowISO(),
		}},
		Purchases: []cleanPurchase{{
			ProductID:   99,
			Quantity:    1,
			UnitCost:    200,
			TotalCost:   200,
			PurchasedAt: nowISO(),
			CreatedAt:   nowISO(),
		}},
	}
	if err := app.importLocalData(bad); err == nil {
		t.Fatal("broken import did not fail")
	}

	after, err := app.localItems(ItemsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Items) != len(before.Items) {
		t.Fatalf("items changed after failed import: before=%+v after=%+v", before.Items, after.Items)
	}
	if after.Items[0].ID != itemID || after.Items[0].Name != "Tea" {
		t.Fatalf("original item changed: %+v", after.Items[0])
	}
}

func TestLocalDispatchRoutes(t *testing.T) {
	app := newLocalTestApp(t)

	session := app.Session()
	if session.Error != "" || session.Authenticated {
		t.Fatalf("initial session: %+v", session)
	}

	auth := app.Register(RegisterInput{
		CompanyName:     "Corner Shop",
		Email:           "owner@example.com",
		Password:        "secret123",
		ConfirmPassword: "secret123",
	})
	if auth.Error != "" || auth.Company == nil {
		t.Fatalf("register through dispatcher: %+v", auth)
	}

	created, err := app.CreateItem(ItemInput{
		Name:              "Tea",
		SKU:               "TEA-1",
		Category:          "Beverages",
		LowStockThreshold: "2",
		PurchasePrice:     "1.00",
		SellPrice:         "1.50",
	})
	if err != nil || created.Error != "" {
		t.Fatalf("create item through dispatcher: %+v %v", created, err)
	}

	items, err := app.Items(ItemsInput{})
	if err != nil || len(items.Items) != 1 {
		t.Fatalf("items through dispatcher: %+v %v", items.Items, err)
	}

	if err := app.Logout(); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if app.currentCompany() != nil {
		t.Fatal("logout did not clear the local session")
	}
}

func TestSetBackendKeepsSeparateSessions(t *testing.T) {
	app := newLocalTestApp(t)
	registerLocal(t, app, "Corner Shop", "owner@example.com", "secret123")

	cloud := app.SetBackend("cloud")
	if cloud.Error != "" || !cloud.IsCloud {
		t.Fatalf("switch to cloud: %+v", cloud)
	}
	if app.currentMode() != modeCloud {
		t.Fatalf("mode = %q", app.currentMode())
	}
	// No cloud session exists in the temporary folder.
	if app.currentCompany() != nil {
		t.Fatalf("cloud workspace should be signed out: %+v", app.currentCompany())
	}

	local := app.SetBackend("local")
	if local.Error != "" || !local.IsLocal {
		t.Fatalf("switch to local: %+v", local)
	}
	company := app.currentCompany()
	if company == nil || company.Email != "owner@example.com" {
		t.Fatalf("local session was not restored: %+v", company)
	}

	raw, err := os.ReadFile(filepath.Join(app.dataDir, backendFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"mode": "local"`) {
		t.Fatalf("backend.json = %s", raw)
	}
	if strings.Contains(string(raw), "key") || strings.Contains(string(raw), "token") {
		t.Fatalf("backend.json contains secrets: %s", raw)
	}

	bad := app.SetBackend("whatever")
	if bad.Error == "" {
		t.Fatal("invalid backend was accepted")
	}
}

func TestLocalSessionFileIsPrivate(t *testing.T) {
	app := newLocalTestApp(t)
	registerLocal(t, app, "Corner Shop", "owner@example.com", "secret123")

	path := app.localSessionFilePath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("session file: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("session permissions = %v, want owner-only", info.Mode().Perm())
	}

	// Sessions expire; an expired token must not restore a workspace.
	if _, err := app.local.db.Exec(
		"UPDATE sessions SET expires_at = ?", jsISO(time.Now().Add(-time.Hour)),
	); err != nil {
		t.Fatal(err)
	}
	app.mu.Lock()
	app.company = nil
	app.localToken = ""
	app.mu.Unlock()
	app.restoreLocalSession()
	if app.currentCompany() != nil {
		t.Fatal("expired session restored a company")
	}
}
