package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func validDataJSON(t *testing.T) []byte {
	t.Helper()
	return []byte(`{
		"format": "stockbook-backup",
		"version": 1,
		"exportedAt": "2026-11-14T00:00:00Z",
		"company": {"name": "Corner Shop", "email": "owner@example.com"},
		"products": [
			{"id": 1, "name": " Tea ", "sku": " TEA-1 ", "category": " Hot   Drinks ",
			 "purchasePrice": 100, "sellPrice": 150, "quantity": 7,
			 "lowStockThreshold": 2, "createdAt": "2026-01-01T00:00:00Z",
			 "updatedAt": "2026-01-02T00:00:00Z"},
			{"id": 2, "name": "Biscuits", "sku": null, "category": "Snacks",
			 "purchasePrice": 50, "sellPrice": 80, "quantity": 0}
		],
		"purchases": [
			{"id": 10, "productId": 1, "quantity": 10, "unitCost": 100,
			 "totalCost": 999999, "purchasedAt": "2026-01-03T00:00:00Z",
			 "note": " Opening stock ", "createdAt": "2026-01-03T00:00:00Z"}
		],
		"sales": [
			{"id": 20, "productId": 1, "quantity": 3, "unitPrice": 150,
			 "totalPrice": 1, "unitCost": 90, "totalCost": 1, "profit": -999,
			 "paymentMode": "UPI", "soldAt": "2026-01-04T00:00:00Z",
			 "createdAt": "2026-01-04T00:00:00Z"}
		]
	}`)
}

func TestParseDataFileNormalizesAndRecomputes(t *testing.T) {
	clean, err := parseDataFile(validDataJSON(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(clean.Products) != 2 || len(clean.Purchases) != 1 || len(clean.Sales) != 1 {
		t.Fatalf("counts = %d/%d/%d", len(clean.Products), len(clean.Purchases), len(clean.Sales))
	}

	tea := clean.Products[0]
	if tea.Name != "Tea" || tea.Category != "Hot Drinks" {
		t.Fatalf("product not normalised: %+v", tea)
	}
	if tea.SKU == nil || *tea.SKU != "TEA-1" {
		t.Fatalf("sku = %+v", tea.SKU)
	}
	if clean.Products[1].SKU != nil {
		t.Fatalf("blank sku should be nil: %+v", clean.Products[1].SKU)
	}

	purchase := clean.Purchases[0]
	if purchase.TotalCost != 1000 {
		t.Fatalf("purchase total = %d, want 1000", purchase.TotalCost)
	}
	if purchase.Note == nil || *purchase.Note != "Opening stock" {
		t.Fatalf("purchase note = %+v", purchase.Note)
	}

	sale := clean.Sales[0]
	if sale.PaymentMode != "upi" {
		t.Fatalf("payment mode = %q", sale.PaymentMode)
	}
	// Totals always follow quantity × price/cost; stale values in the file are
	// discarded so the imported ledger stays internally consistent.
	if sale.TotalPrice != 450 || sale.TotalCost != 270 || sale.Profit != 180 {
		t.Fatalf("sale totals = %+v", sale)
	}
}

func TestParseDataFileDropsAccountFields(t *testing.T) {
	clean, err := parseDataFile(validDataJSON(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := buildDataFile(clean)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, forbidden := range []string{"company", "email", "password", "accounts", "owner@example.com", "Corner Shop"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("export contains %q: %s", forbidden, text)
		}
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatal(err)
	}
	if string(envelope["format"]) != `"stockbook-backup"` {
		t.Fatalf("format = %s", envelope["format"])
	}
}

func TestParseDataFileRejectsBrokenFiles(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"invalid json", `{"format":`, "not valid JSON"},
		{"wrong format", `{"format":"other","version":1,"products":[],"purchases":[],"sales":[]}`, "Stockbook data"},
		{"newer version", `{"format":"stockbook-backup","version":2,"products":[],"purchases":[],"sales":[]}`, "newer version"},
		{"missing lists", `{"format":"stockbook-backup","version":1,"products":[]}`, "products, purchases, and sales"},
		{"wrong list type", `{"format":"stockbook-backup","version":1,"products":{},"purchases":[],"sales":[]}`, "not valid JSON"},
		{"zero product id", `{"format":"stockbook-backup","version":1,"products":[{"id":0,"name":"Tea","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[]}`, "identifier"},
		{"duplicate product id", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1},{"id":1,"name":"Coffee","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[]}`, "duplicate identifier"},
		{"duplicate sku", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","sku":"A","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1},{"id":2,"name":"Coffee","sku":"A","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[]}`, "repeats the SKU"},
		{"negative price", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","category":"Drinks","purchasePrice":-1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[]}`, "purchase price"},
		{"huge price", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","category":"Drinks","purchasePrice":1000000001,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[]}`, "purchase price"},
		{"negative quantity", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":-5}],"purchases":[],"sales":[]}`, "quantity"},
		{"duplicate sku case", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","sku":" Ab ","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1},{"id":2,"name":"Coffee","sku":"Ab","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[]}`, "repeats the SKU"},
		{"duplicate sku different case", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","sku":"AB","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1},{"id":2,"name":"Coffee","sku":"ab","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[]}`, "repeats the SKU"},
		{"orphan purchase", `{"format":"stockbook-backup","version":1,"products":[],"purchases":[{"productId":5,"quantity":1,"unitCost":10,"purchasedAt":"2026-01-01T00:00:00Z"}],"sales":[]}`, "not in the file"},
		{"orphan sale", `{"format":"stockbook-backup","version":1,"products":[],"purchases":[],"sales":[{"productId":5,"quantity":1,"unitPrice":10,"soldAt":"2026-01-01T00:00:00Z"}]}`, "not in the file"},
		{"bad sale quantity", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[{"productId":1,"quantity":0,"unitPrice":10,"soldAt":"2026-01-01T00:00:00Z"}]}`, "quantity"},
		{"bad payment mode", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[{"productId":1,"quantity":1,"unitPrice":10,"paymentMode":"card","soldAt":"2026-01-01T00:00:00Z"}]}`, "payment mode"},
		{"bad timestamp", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1,"createdAt":"yesterday"}],"purchases":[],"sales":[]}`, "created date"},
		{"long note", `{"format":"stockbook-backup","version":1,"products":[{"id":1,"name":"Tea","category":"Drinks","purchasePrice":1,"sellPrice":2,"quantity":1}],"purchases":[],"sales":[{"productId":1,"quantity":1,"unitPrice":10,"soldAt":"2026-01-01T00:00:00Z","note":"` + strings.Repeat("x", 241) + `"}]}`, "note is longer"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseDataFile([]byte(test.body))
			if err == nil {
				t.Fatalf("file was accepted: %s", test.body)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %q, want it to contain %q", err, test.want)
			}
		})
	}
}

func TestParseDataFileAcceptsEmptyInventory(t *testing.T) {
	clean, err := parseDataFile([]byte(
		`{"format":"stockbook-backup","version":1,"products":[],"purchases":[],"sales":[]}`,
	))
	if err != nil {
		t.Fatalf("empty inventory rejected: %v", err)
	}
	if len(clean.Products) != 0 || len(clean.Purchases) != 0 || len(clean.Sales) != 0 {
		t.Fatalf("clean = %+v", clean)
	}
}

func TestDataFileIsStableAcrossBackends(t *testing.T) {
	// Both backends emit the same envelope, so a local export can be handed
	// straight to the cloud import and vice versa.
	clean, err := parseDataFile(validDataJSON(t))
	if err != nil {
		t.Fatal(err)
	}
	first, err := buildDataFile(clean)
	if err != nil {
		t.Fatal(err)
	}
	reparsed, err := parseDataFile(first)
	if err != nil {
		t.Fatalf("re-parse built file: %v", err)
	}
	second, err := buildDataFile(reparsed)
	if err != nil {
		t.Fatal(err)
	}

	var a, b dataFile
	if err := json.Unmarshal(first, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second, &b); err != nil {
		t.Fatal(err)
	}
	a.ExportedAt = ""
	b.ExportedAt = ""
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	if string(left) != string(right) {
		t.Fatalf("export changed shape after round trip:\n%s\n%s", left, right)
	}
}
