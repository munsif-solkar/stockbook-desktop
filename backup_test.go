package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExportToFileStripsAccountFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v1/rpc/stockbook_export" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"format": "stockbook-backup",
			"version": 1,
			"exportedAt": "2026-11-14T00:00:00Z",
			"company": {"name": "Corner Shop", "email": "owner@example.com"},
			"products": [{"id": 1, "name": "Tea", "sku": "TEA-1", "category": "Beverages",
				"purchasePrice": 100, "sellPrice": 150, "quantity": 5,
				"lowStockThreshold": 2, "createdAt": "2026-01-01T00:00:00Z",
				"updatedAt": "2026-01-01T00:00:00Z"}],
			"purchases": [], "sales": []
		}`))
	}))
	defer server.Close()

	app := &App{
		dataDir: t.TempDir(),
		api: &supabaseAPI{
			baseURL:     server.URL,
			anonKey:     "test-key",
			client:      server.Client(),
			accessToken: "test-token",
			expiresAt:   time.Now().Add(time.Hour),
		},
	}

	target := filepath.Join(app.dataDir, "export.json")
	size, err := app.exportToFile(target)
	if err != nil || size == 0 {
		t.Fatalf("export: %d %v", size, err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"company", "email", "password", "password_hash", "accounts"} {
		if _, present := payload[forbidden]; present {
			t.Fatalf("export contains %q: %s", forbidden, data)
		}
	}
	if strings.Contains(string(data), "owner@example.com") {
		t.Fatalf("export contains the account email: %s", data)
	}
	var products []map[string]any
	if err := json.Unmarshal(payload["products"], &products); err != nil {
		t.Fatal(err)
	}
	if len(products) != 1 || products[0]["name"] != "Tea" {
		t.Fatalf("products = %+v", products)
	}
}
