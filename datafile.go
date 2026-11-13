package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Stockbook data files hold inventory only: items, purchases, and sales. They
// never contain accounts, email addresses, or passwords, so a file exported
// from one workspace can be imported into any other workspace on either
// backend.
const (
	dataFormat    = "stockbook-backup"
	dataVersion   = 1
	maxDataRows   = 500000
	maxMoney      = int64(1_000_000_000)
	maxQuantity   = int64(1_000_000_000)
	maxImportNote = 240
)

type dataProduct struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	SKU               *string `json:"sku"`
	Category          string  `json:"category"`
	PurchasePrice     int64   `json:"purchasePrice"`
	SellPrice         int64   `json:"sellPrice"`
	Quantity          int64   `json:"quantity"`
	LowStockThreshold *int64  `json:"lowStockThreshold,omitempty"`
	CreatedAt         string  `json:"createdAt,omitempty"`
	UpdatedAt         string  `json:"updatedAt,omitempty"`
}

type dataPurchase struct {
	ID          int64   `json:"id,omitempty"`
	ProductID   int64   `json:"productId"`
	Quantity    int64   `json:"quantity"`
	UnitCost    int64   `json:"unitCost"`
	TotalCost   int64   `json:"totalCost"`
	PurchasedAt string  `json:"purchasedAt"`
	Note        *string `json:"note"`
	CreatedAt   string  `json:"createdAt,omitempty"`
}

type dataSale struct {
	ID          int64   `json:"id,omitempty"`
	ProductID   int64   `json:"productId"`
	Quantity    int64   `json:"quantity"`
	UnitPrice   int64   `json:"unitPrice"`
	TotalPrice  int64   `json:"totalPrice"`
	UnitCost    *int64  `json:"unitCost,omitempty"`
	TotalCost   int64   `json:"totalCost"`
	Profit      int64   `json:"profit"`
	PaymentMode string  `json:"paymentMode"`
	SoldAt      string  `json:"soldAt"`
	Note        *string `json:"note"`
	CreatedAt   string  `json:"createdAt,omitempty"`
}

type dataFile struct {
	Format     string         `json:"format"`
	Version    int            `json:"version"`
	ExportedAt string         `json:"exportedAt,omitempty"`
	Products   []dataProduct  `json:"products"`
	Purchases  []dataPurchase `json:"purchases"`
	Sales      []dataSale     `json:"sales"`
}

type cleanProduct struct {
	ID                int64
	Name              string
	SKU               *string
	Category          string
	PurchasePrice     int64
	SellPrice         int64
	Quantity          int64
	LowStockThreshold int64
	CreatedAt         string
	UpdatedAt         string
}

type cleanPurchase struct {
	ID          int64
	ProductID   int64
	Quantity    int64
	UnitCost    int64
	TotalCost   int64
	PurchasedAt string
	Note        *string
	CreatedAt   string
}

type cleanSale struct {
	ID          int64
	ProductID   int64
	Quantity    int64
	UnitPrice   int64
	TotalPrice  int64
	UnitCost    int64
	TotalCost   int64
	Profit      int64
	PaymentMode string
	SoldAt      string
	Note        *string
	CreatedAt   string
}

type cleanData struct {
	Products  []cleanProduct
	Purchases []cleanPurchase
	Sales     []cleanSale
}

func invalidData(message string) error {
	return errors.New("not a Stockbook data file: " + message)
}

func parseTimestamp(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nowISO(), nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return "", errors.New("invalid timestamp")
	}
	return jsISO(parsed), nil
}

func trimText(value string, max int, fallback string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if fallback != "" {
			return fallback, nil
		}
		return "", errors.New("is empty")
	}
	if len([]rune(trimmed)) > max {
		return "", fmt.Errorf("is longer than %d characters", max)
	}
	return trimmed, nil
}

func cleanNote(value *string, where string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if len([]rune(trimmed)) > maxImportNote {
		return nil, fmt.Errorf("%s note is longer than %d characters", where, maxImportNote)
	}
	return &trimmed, nil
}

func checkRange(value, min, max int64, where, field string) error {
	if value < min || value > max {
		return fmt.Errorf("%s has an invalid %s", where, field)
	}
	return nil
}

// parseDataFile validates a JSON export and returns only clean, internally
// consistent records. Any problem is reported before the current workspace is
// touched, so a broken or hand-edited file can never corrupt the ledger.
func parseDataFile(data []byte) (*cleanData, error) {
	var file dataFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, invalidData("the file is not valid JSON")
	}
	if file.Format != dataFormat {
		return nil, invalidData("expected a Stockbook data export")
	}
	if file.Version < 1 {
		return nil, invalidData("the file has an unsupported version")
	}
	if file.Version > dataVersion {
		return nil, invalidData("the file was made by a newer version of Stockbook")
	}
	if file.Products == nil || file.Purchases == nil || file.Sales == nil {
		return nil, invalidData("expected products, purchases, and sales lists")
	}
	if len(file.Products)+len(file.Purchases)+len(file.Sales) > maxDataRows {
		return nil, invalidData("the file contains too many records")
	}

	clean := &cleanData{
		Products:  make([]cleanProduct, 0, len(file.Products)),
		Purchases: make([]cleanPurchase, 0, len(file.Purchases)),
		Sales:     make([]cleanSale, 0, len(file.Sales)),
	}
	productIDs := make(map[int64]struct{}, len(file.Products))
	skus := make(map[string]struct{}, len(file.Products))
	productPrices := make(map[int64]int64, len(file.Products))

	for index, raw := range file.Products {
		where := fmt.Sprintf("item %d", index+1)
		if raw.ID <= 0 {
			return nil, fmt.Errorf("%s has no identifier", where)
		}
		if _, duplicate := productIDs[raw.ID]; duplicate {
			return nil, fmt.Errorf("%s uses a duplicate identifier", where)
		}
		productIDs[raw.ID] = struct{}{}

		name, err := trimText(raw.Name, 100, "")
		if err != nil {
			return nil, fmt.Errorf("%s %s", where, err)
		}
		category, err := trimText(strings.Join(strings.Fields(raw.Category), " "), 50, "Uncategorised")
		if err != nil {
			return nil, fmt.Errorf("%s %s", where, err)
		}

		var sku *string
		if raw.SKU != nil {
			trimmed := strings.TrimSpace(*raw.SKU)
			if len([]rune(trimmed)) > 40 {
				return nil, fmt.Errorf("%s has a SKU longer than 40 characters", where)
			}
			if trimmed != "" {
				// Cloud workspaces keep SKUs unique case-insensitively, so
				// validate the same way before either backend is touched.
				key := strings.ToLower(trimmed)
				if _, duplicate := skus[key]; duplicate {
					return nil, fmt.Errorf("%s repeats the SKU %q", where, trimmed)
				}
				skus[key] = struct{}{}
				sku = &trimmed
			}
		}

		if err := checkRange(raw.PurchasePrice, 0, maxMoney, where, "purchase price"); err != nil {
			return nil, err
		}
		if err := checkRange(raw.SellPrice, 0, maxMoney, where, "selling price"); err != nil {
			return nil, err
		}
		if err := checkRange(raw.Quantity, 0, maxQuantity, where, "quantity"); err != nil {
			return nil, err
		}

		threshold := int64(5)
		if raw.LowStockThreshold != nil {
			threshold = *raw.LowStockThreshold
			if err := checkRange(threshold, 0, maxQuantity, where, "low-stock alert"); err != nil {
				return nil, err
			}
		}

		createdAt, err := parseTimestamp(raw.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s has an invalid created date", where)
		}
		updatedAt, err := parseTimestamp(raw.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s has an invalid updated date", where)
		}

		productPrices[raw.ID] = raw.PurchasePrice
		clean.Products = append(clean.Products, cleanProduct{
			ID:                raw.ID,
			Name:              name,
			SKU:               sku,
			Category:          category,
			PurchasePrice:     raw.PurchasePrice,
			SellPrice:         raw.SellPrice,
			Quantity:          raw.Quantity,
			LowStockThreshold: threshold,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		})
	}

	for index, raw := range file.Purchases {
		where := fmt.Sprintf("purchase %d", index+1)
		if _, ok := productIDs[raw.ProductID]; !ok {
			return nil, fmt.Errorf("%s refers to an item that is not in the file", where)
		}
		if err := checkRange(raw.Quantity, 1, maxQuantity, where, "quantity"); err != nil {
			return nil, err
		}
		if err := checkRange(raw.UnitCost, 0, maxMoney, where, "unit cost"); err != nil {
			return nil, err
		}
		purchasedAt, err := parseTimestamp(raw.PurchasedAt)
		if err != nil {
			return nil, fmt.Errorf("%s has an invalid purchase date", where)
		}
		createdAt, err := parseTimestamp(raw.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s has an invalid created date", where)
		}
		note, err := cleanNote(raw.Note, where)
		if err != nil {
			return nil, err
		}
		clean.Purchases = append(clean.Purchases, cleanPurchase{
			ID:          raw.ID,
			ProductID:   raw.ProductID,
			Quantity:    raw.Quantity,
			UnitCost:    raw.UnitCost,
			TotalCost:   raw.Quantity * raw.UnitCost,
			PurchasedAt: purchasedAt,
			Note:        note,
			CreatedAt:   createdAt,
		})
	}

	for index, raw := range file.Sales {
		where := fmt.Sprintf("sale %d", index+1)
		if _, ok := productIDs[raw.ProductID]; !ok {
			return nil, fmt.Errorf("%s refers to an item that is not in the file", where)
		}
		if err := checkRange(raw.Quantity, 1, maxQuantity, where, "quantity"); err != nil {
			return nil, err
		}
		if err := checkRange(raw.UnitPrice, 0, maxMoney, where, "unit price"); err != nil {
			return nil, err
		}

		unitCost := productPrices[raw.ProductID]
		if raw.UnitCost != nil {
			unitCost = *raw.UnitCost
			if err := checkRange(unitCost, 0, maxMoney, where, "unit cost"); err != nil {
				return nil, err
			}
		}
		paymentMode := strings.ToLower(strings.TrimSpace(raw.PaymentMode))
		if paymentMode == "" {
			paymentMode = "cash"
		}
		if paymentMode != "cash" && paymentMode != "upi" {
			return nil, fmt.Errorf("%s has an invalid payment mode", where)
		}
		soldAt, err := parseTimestamp(raw.SoldAt)
		if err != nil {
			return nil, fmt.Errorf("%s has an invalid sale date", where)
		}
		createdAt, err := parseTimestamp(raw.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s has an invalid created date", where)
		}
		note, err := cleanNote(raw.Note, where)
		if err != nil {
			return nil, err
		}

		totalPrice := raw.Quantity * raw.UnitPrice
		totalCost := raw.Quantity * unitCost
		clean.Sales = append(clean.Sales, cleanSale{
			ID:          raw.ID,
			ProductID:   raw.ProductID,
			Quantity:    raw.Quantity,
			UnitPrice:   raw.UnitPrice,
			TotalPrice:  totalPrice,
			UnitCost:    unitCost,
			TotalCost:   totalCost,
			Profit:      totalPrice - totalCost,
			PaymentMode: paymentMode,
			SoldAt:      soldAt,
			Note:        note,
			CreatedAt:   createdAt,
		})
	}

	return clean, nil
}

// buildDataFile turns validated records back into the one JSON shape shared by
// both backends.
func buildDataFile(clean *cleanData) ([]byte, error) {
	file := dataFile{
		Format:     dataFormat,
		Version:    dataVersion,
		ExportedAt: nowISO(),
		Products:   make([]dataProduct, 0, len(clean.Products)),
		Purchases:  make([]dataPurchase, 0, len(clean.Purchases)),
		Sales:      make([]dataSale, 0, len(clean.Sales)),
	}
	for _, product := range clean.Products {
		threshold := product.LowStockThreshold
		file.Products = append(file.Products, dataProduct{
			ID:                product.ID,
			Name:              product.Name,
			SKU:               product.SKU,
			Category:          product.Category,
			PurchasePrice:     product.PurchasePrice,
			SellPrice:         product.SellPrice,
			Quantity:          product.Quantity,
			LowStockThreshold: &threshold,
			CreatedAt:         product.CreatedAt,
			UpdatedAt:         product.UpdatedAt,
		})
	}
	for _, purchase := range clean.Purchases {
		file.Purchases = append(file.Purchases, dataPurchase{
			ID:          purchase.ID,
			ProductID:   purchase.ProductID,
			Quantity:    purchase.Quantity,
			UnitCost:    purchase.UnitCost,
			TotalCost:   purchase.TotalCost,
			PurchasedAt: purchase.PurchasedAt,
			Note:        purchase.Note,
			CreatedAt:   purchase.CreatedAt,
		})
	}
	for _, sale := range clean.Sales {
		unitCost := sale.UnitCost
		file.Sales = append(file.Sales, dataSale{
			ID:          sale.ID,
			ProductID:   sale.ProductID,
			Quantity:    sale.Quantity,
			UnitPrice:   sale.UnitPrice,
			TotalPrice:  sale.TotalPrice,
			UnitCost:    &unitCost,
			TotalCost:   sale.TotalCost,
			Profit:      sale.Profit,
			PaymentMode: sale.PaymentMode,
			SoldAt:      sale.SoldAt,
			Note:        sale.Note,
			CreatedAt:   sale.CreatedAt,
		})
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(file); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
