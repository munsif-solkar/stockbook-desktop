package main

import (
	"database/sql"
)

// exportLocalData reads the signed-in local workspace into the same inventory
// shape the cloud export uses. Account details are never included.
func (a *App) exportLocalData() (*cleanData, error) {
	company, err := a.requireCompany()
	if err != nil {
		return nil, err
	}

	clean := &cleanData{
		Products:  []cleanProduct{},
		Purchases: []cleanPurchase{},
		Sales:     []cleanSale{},
	}

	productRows, err := a.local.db.Query(
		`SELECT id, name, sku, category, purchase_price, sell_price, quantity,
		        low_stock_threshold, created_at, updated_at
		 FROM products
		 WHERE company_id = ?
		 ORDER BY id ASC`, company.ID)
	if err != nil {
		return nil, err
	}
	for productRows.Next() {
		var product cleanProduct
		var sku sql.NullString
		if err := productRows.Scan(
			&product.ID, &product.Name, &sku, &product.Category,
			&product.PurchasePrice, &product.SellPrice, &product.Quantity,
			&product.LowStockThreshold, &product.CreatedAt, &product.UpdatedAt,
		); err != nil {
			productRows.Close()
			return nil, err
		}
		if sku.Valid {
			value := sku.String
			product.SKU = &value
		}
		clean.Products = append(clean.Products, product)
	}
	if err := productRows.Err(); err != nil {
		productRows.Close()
		return nil, err
	}
	productRows.Close()

	purchaseRows, err := a.local.db.Query(
		`SELECT id, product_id, quantity, unit_cost, total_cost, purchased_at, note, created_at
		 FROM purchases
		 WHERE company_id = ?
		 ORDER BY id ASC`, company.ID)
	if err != nil {
		return nil, err
	}
	for purchaseRows.Next() {
		var purchase cleanPurchase
		var note sql.NullString
		if err := purchaseRows.Scan(
			&purchase.ID, &purchase.ProductID, &purchase.Quantity, &purchase.UnitCost,
			&purchase.TotalCost, &purchase.PurchasedAt, &note, &purchase.CreatedAt,
		); err != nil {
			purchaseRows.Close()
			return nil, err
		}
		if note.Valid && note.String != "" {
			value := note.String
			purchase.Note = &value
		}
		clean.Purchases = append(clean.Purchases, purchase)
	}
	if err := purchaseRows.Err(); err != nil {
		purchaseRows.Close()
		return nil, err
	}
	purchaseRows.Close()

	saleRows, err := a.local.db.Query(
		`SELECT id, product_id, quantity, unit_price, total_price, unit_cost,
		        total_cost, profit, payment_mode, sold_at, note, created_at
		 FROM sales
		 WHERE company_id = ?
		 ORDER BY id ASC`, company.ID)
	if err != nil {
		return nil, err
	}
	for saleRows.Next() {
		var sale cleanSale
		var note sql.NullString
		if err := saleRows.Scan(
			&sale.ID, &sale.ProductID, &sale.Quantity, &sale.UnitPrice,
			&sale.TotalPrice, &sale.UnitCost, &sale.TotalCost, &sale.Profit,
			&sale.PaymentMode, &sale.SoldAt, &note, &sale.CreatedAt,
		); err != nil {
			saleRows.Close()
			return nil, err
		}
		if note.Valid && note.String != "" {
			value := note.String
			sale.Note = &value
		}
		clean.Sales = append(clean.Sales, sale)
	}
	if err := saleRows.Err(); err != nil {
		saleRows.Close()
		return nil, err
	}
	saleRows.Close()

	return clean, nil
}

// importLocalData replaces the signed-in local workspace in one transaction.
// The records have already been validated, so a failure rolls back to the
// previous ledger instead of leaving a half-imported one.
func (a *App) importLocalData(clean *cleanData) error {
	company, err := a.requireCompany()
	if err != nil {
		return err
	}

	tx, err := a.local.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM sales WHERE company_id = ?", company.ID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM purchases WHERE company_id = ?", company.ID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM products WHERE company_id = ?", company.ID); err != nil {
		return err
	}

	newIDs := make(map[int64]int64, len(clean.Products))
	for _, product := range clean.Products {
		result, err := tx.Exec(
			`INSERT INTO products
			   (company_id, name, sku, category, purchase_price, sell_price, quantity,
			    low_stock_threshold, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			company.ID, product.Name, product.SKU, product.Category,
			product.PurchasePrice, product.SellPrice, product.Quantity,
			product.LowStockThreshold, product.CreatedAt, product.UpdatedAt,
		)
		if err != nil {
			return err
		}
		newID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		newIDs[product.ID] = newID
	}

	for _, purchase := range clean.Purchases {
		productID, ok := newIDs[purchase.ProductID]
		if !ok {
			return errLocalUnavailable
		}
		if _, err := tx.Exec(
			`INSERT INTO purchases
			   (company_id, product_id, quantity, unit_cost, total_cost,
			    purchased_at, note, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			company.ID, productID, purchase.Quantity, purchase.UnitCost,
			purchase.TotalCost, purchase.PurchasedAt, purchase.Note, purchase.CreatedAt,
		); err != nil {
			return err
		}
	}

	for _, sale := range clean.Sales {
		productID, ok := newIDs[sale.ProductID]
		if !ok {
			return errLocalUnavailable
		}
		if _, err := tx.Exec(
			`INSERT INTO sales
			   (company_id, product_id, quantity, unit_price, total_price, unit_cost,
			    total_cost, profit, payment_mode, sold_at, note, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			company.ID, productID, sale.Quantity, sale.UnitPrice, sale.TotalPrice,
			sale.UnitCost, sale.TotalCost, sale.Profit, sale.PaymentMode,
			sale.SoldAt, sale.Note, sale.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
