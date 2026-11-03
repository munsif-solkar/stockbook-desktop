package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type businessError struct {
	message string
	field   string
}

func (e *businessError) Error() string { return e.message }

func (a *App) CreateItem(in ItemInput) (MutationResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return MutationResult{}, err
	}
	valid, fieldErrors := validateItem(in, false)
	if fieldErrors != nil {
		return MutationResult{FieldErrors: fieldErrors}, nil
	}

	now := nowISO()
	_, err = a.db.Exec(
		`INSERT INTO products
		   (company_id, name, sku, category, purchase_price, sell_price, quantity,
		    low_stock_threshold, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?)`,
		company.ID, valid.Name, valid.SKU, valid.Category, valid.PurchasePrice,
		valid.SellPrice, valid.LowStockThreshold, now, now,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return MutationResult{FieldErrors: map[string][]string{
				"sku": {"An item with this SKU already exists."},
			}}, nil
		}
		return MutationResult{Error: "Could not add this item. Please try again."}, nil
	}

	return MutationResult{}, nil
}

func (a *App) UpdateItem(in ItemInput) (MutationResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return MutationResult{}, err
	}
	valid, fieldErrors := validateItem(in, true)
	if fieldErrors != nil {
		return MutationResult{FieldErrors: fieldErrors}, nil
	}

	now := nowISO()
	result, err := a.db.Exec(
		`UPDATE products
		 SET name = ?, sku = ?, category = ?, purchase_price = ?, sell_price = ?,
		     low_stock_threshold = ?, updated_at = ?
		 WHERE id = ? AND company_id = ?`,
		valid.Name, valid.SKU, valid.Category, valid.PurchasePrice, valid.SellPrice,
		valid.LowStockThreshold, now, in.ProductID, company.ID,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return MutationResult{FieldErrors: map[string][]string{
				"sku": {"An item with this SKU already exists."},
			}}, nil
		}
		return MutationResult{Error: "Could not update this item. Please try again."}, nil
	}
	changed, err := result.RowsAffected()
	if err != nil || changed == 0 {
		return MutationResult{Error: "Item not found."}, nil
	}

	return MutationResult{}, nil
}

func (a *App) Product(in IDInput) (ProductResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return ProductResult{}, err
	}
	id, ok := parseRecordID(in.ID)
	if !ok {
		return ProductResult{}, nil
	}
	product, err := a.product(company.ID, id)
	if err != nil {
		return ProductResult{}, err
	}
	return ProductResult{Product: product}, nil
}

func (a *App) ProductOptions() ([]ProductOption, error) {
	company, err := a.requireCompany()
	if err != nil {
		return nil, err
	}
	return a.productOptions(company.ID)
}

func (a *App) Categories() ([]string, error) {
	company, err := a.requireCompany()
	if err != nil {
		return nil, err
	}
	return a.categories(company.ID)
}

func (a *App) recordPurchase(companyID int64, input validPurchase) (int64, string, error) {
	tx, err := a.db.Begin()
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()

	var productName string
	if err := tx.QueryRow(
		"SELECT name FROM products WHERE id = ? AND company_id = ?",
		input.ProductID, companyID,
	).Scan(&productName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", &businessError{message: "Select an item from your catalog.", field: "productId"}
		}
		return 0, "", err
	}

	now := time.Now()
	nowString := jsISO(now)
	purchasedAt := nowString
	if input.PurchasedAt != todayISO() {
		if parsed, ok := parseISODate(input.PurchasedAt); ok {
			noon := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 12, 0, 0, 0, time.Local)
			purchasedAt = jsISO(noon)
		}
	}
	totalCost := input.Quantity * input.UnitCost

	result, err := tx.Exec(
		`INSERT INTO purchases
		   (company_id, product_id, quantity, unit_cost, total_cost, purchased_at, note, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		companyID, input.ProductID, input.Quantity, input.UnitCost, totalCost,
		purchasedAt, input.Note, nowString,
	)
	if err != nil {
		return 0, "", err
	}
	purchaseID, err := result.LastInsertId()
	if err != nil {
		return 0, "", err
	}

	if _, err := tx.Exec(
		`UPDATE products
		 SET quantity = quantity + ?, purchase_price = ?, updated_at = ?
		 WHERE id = ? AND company_id = ?`,
		input.Quantity, input.UnitCost, nowString, input.ProductID, companyID,
	); err != nil {
		return 0, "", err
	}

	if err := tx.Commit(); err != nil {
		return 0, "", err
	}
	return purchaseID, productName, nil
}

func purchaseMutationError(err error) MutationResult {
	var business *businessError
	if errors.As(err, &business) {
		result := MutationResult{Error: business.message}
		if business.field != "" {
			result.FieldErrors = map[string][]string{business.field: {business.message}}
		}
		return result
	}
	return MutationResult{Error: "Could not record this purchase. Please try again."}
}

func (a *App) CreatePurchase(in PurchaseInput) (MutationResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return MutationResult{}, err
	}
	valid, fieldErrors := validatePurchase(in)
	if fieldErrors != nil {
		return MutationResult{FieldErrors: fieldErrors}, nil
	}

	purchaseID, productName, err := a.recordPurchase(company.ID, valid)
	if err != nil {
		return purchaseMutationError(err), nil
	}
	return MutationResult{
		Success:    "Stock updated for " + productName + ". Purchase recorded.",
		PurchaseID: purchaseID,
	}, nil
}

func (a *App) DeletePurchase(in IDInput) (MutationResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return MutationResult{}, err
	}
	id, ok := parseRecordID(in.ID)
	if !ok {
		return MutationResult{Error: "That purchase record could not be found."}, nil
	}

	tx, err := a.db.Begin()
	if err != nil {
		return MutationResult{Error: "Could not delete this purchase. Please try again."}, nil
	}
	defer tx.Rollback()

	var purchaseID, productID, quantity, stock int64
	var productName string
	err = tx.QueryRow(
		`SELECT pu.id, pu.product_id, pu.quantity, p.name, p.quantity
		 FROM purchases pu
		 JOIN products p ON p.id = pu.product_id
		 WHERE pu.id = ? AND pu.company_id = ? AND p.company_id = ?`,
		id, company.ID, company.ID,
	).Scan(&purchaseID, &productID, &quantity, &productName, &stock)
	if errors.Is(err, sql.ErrNoRows) {
		return MutationResult{Error: "That purchase record no longer exists."}, nil
	}
	if err != nil {
		return MutationResult{Error: "Could not delete this purchase. Please try again."}, nil
	}

	if stock < quantity {
		plural := "s"
		if quantity == 1 {
			plural = ""
		}
		return MutationResult{Error: fmt.Sprintf(
			"Cannot delete this purchase: it added %d unit%s of %s, but only %d remain in stock. Sell or adjust stock before deleting it.",
			quantity, plural, productName, stock,
		)}, nil
	}

	if _, err := tx.Exec("DELETE FROM purchases WHERE id = ? AND company_id = ?", purchaseID, company.ID); err != nil {
		return MutationResult{Error: "Could not delete this purchase. Please try again."}, nil
	}

	var latestCost int64
	latestErr := tx.QueryRow(
		`SELECT unit_cost
		 FROM purchases
		 WHERE product_id = ? AND company_id = ?
		 ORDER BY purchased_at DESC, id DESC
		 LIMIT 1`, productID, company.ID,
	).Scan(&latestCost)

	now := nowISO()
	if latestErr == nil {
		if _, err := tx.Exec(
			`UPDATE products
			 SET quantity = quantity - ?, purchase_price = ?, updated_at = ?
			 WHERE id = ? AND company_id = ?`,
			quantity, latestCost, now, productID, company.ID,
		); err != nil {
			return MutationResult{Error: "Could not delete this purchase. Please try again."}, nil
		}
	} else if errors.Is(latestErr, sql.ErrNoRows) {
		if _, err := tx.Exec(
			`UPDATE products
			 SET quantity = quantity - ?, updated_at = ?
			 WHERE id = ? AND company_id = ?`,
			quantity, now, productID, company.ID,
		); err != nil {
			return MutationResult{Error: "Could not delete this purchase. Please try again."}, nil
		}
	} else {
		return MutationResult{Error: "Could not delete this purchase. Please try again."}, nil
	}

	if err := tx.Commit(); err != nil {
		return MutationResult{Error: "Could not delete this purchase. Please try again."}, nil
	}
	return MutationResult{Success: "Purchase for " + productName + " deleted. Stock updated."}, nil
}

func (a *App) CreateSale(in SaleInput) (MutationResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return MutationResult{}, err
	}
	valid, fieldErrors := validateSale(in)
	if fieldErrors != nil {
		return MutationResult{FieldErrors: fieldErrors}, nil
	}

	tx, err := a.db.Begin()
	if err != nil {
		return MutationResult{Error: "Could not record this sale. Please try again."}, nil
	}
	defer tx.Rollback()

	var productName string
	var stock, purchasePrice int64
	err = tx.QueryRow(
		`SELECT name, quantity, purchase_price
		 FROM products WHERE id = ? AND company_id = ?`,
		valid.ProductID, company.ID,
	).Scan(&productName, &stock, &purchasePrice)
	if errors.Is(err, sql.ErrNoRows) {
		return MutationResult{Error: "That product is no longer in your inventory."}, nil
	}
	if err != nil {
		return MutationResult{Error: "Could not record this sale. Please try again."}, nil
	}

	if stock < valid.Quantity {
		plural := "s"
		if stock == 1 {
			plural = ""
		}
		message := fmt.Sprintf("Only %d unit%s of %s in stock.", stock, plural, productName)
		return MutationResult{
			Error:       message,
			FieldErrors: map[string][]string{"quantity": {message}},
		}, nil
	}

	totalPrice := valid.Quantity * valid.UnitPrice
	totalCost := valid.Quantity * purchasePrice
	profit := totalPrice - totalCost
	now := nowISO()

	result, err := tx.Exec(
		`INSERT INTO sales
		   (company_id, product_id, quantity, unit_price, total_price, unit_cost,
		    total_cost, profit, payment_mode, sold_at, note, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		company.ID, valid.ProductID, valid.Quantity, valid.UnitPrice, totalPrice,
		purchasePrice, totalCost, profit, valid.PaymentMode, now, valid.Note, now,
	)
	if err != nil {
		return MutationResult{Error: "Could not record this sale. Please try again."}, nil
	}
	saleID, err := result.LastInsertId()
	if err != nil {
		return MutationResult{Error: "Could not record this sale. Please try again."}, nil
	}

	if _, err := tx.Exec(
		`UPDATE products
		 SET quantity = quantity - ?, updated_at = ?
		 WHERE id = ? AND company_id = ?`,
		valid.Quantity, now, valid.ProductID, company.ID,
	); err != nil {
		return MutationResult{Error: "Could not record this sale. Please try again."}, nil
	}

	if err := tx.Commit(); err != nil {
		return MutationResult{Error: "Could not record this sale. Please try again."}, nil
	}
	return MutationResult{Success: "Sale recorded. Stock updated.", SaleID: saleID}, nil
}

func (a *App) DeleteSale(in IDInput) (MutationResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return MutationResult{}, err
	}
	id, ok := parseRecordID(in.ID)
	if !ok {
		return MutationResult{Error: "That sale record could not be found."}, nil
	}

	tx, err := a.db.Begin()
	if err != nil {
		return MutationResult{Error: "Could not delete this sale. Please try again."}, nil
	}
	defer tx.Rollback()

	var saleID, productID, quantity int64
	var productName string
	err = tx.QueryRow(
		`SELECT s.id, s.product_id, s.quantity, p.name
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE s.id = ? AND s.company_id = ? AND p.company_id = ?`,
		id, company.ID, company.ID,
	).Scan(&saleID, &productID, &quantity, &productName)
	if errors.Is(err, sql.ErrNoRows) {
		return MutationResult{Error: "That sale record no longer exists."}, nil
	}
	if err != nil {
		return MutationResult{Error: "Could not delete this sale. Please try again."}, nil
	}

	if _, err := tx.Exec("DELETE FROM sales WHERE id = ? AND company_id = ?", saleID, company.ID); err != nil {
		return MutationResult{Error: "Could not delete this sale. Please try again."}, nil
	}
	if _, err := tx.Exec(
		`UPDATE products
		 SET quantity = quantity + ?, updated_at = ?
		 WHERE id = ? AND company_id = ?`,
		quantity, nowISO(), productID, company.ID,
	); err != nil {
		return MutationResult{Error: "Could not delete this sale. Please try again."}, nil
	}

	if err := tx.Commit(); err != nil {
		return MutationResult{Error: "Could not delete this sale. Please try again."}, nil
	}
	return MutationResult{Success: "Sale of " + productName + " deleted. Stock restored."}, nil
}
