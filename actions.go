package main

import (
	"time"
)

func mutationOutcome(outcome rpcOutcome, success string) MutationResult {
	if !outcome.OK {
		message := outcome.Error
		if message == "" {
			message = "Something went wrong. Please try again."
		}
		result := MutationResult{Error: message}
		if outcome.Field != "" {
			result.FieldErrors = map[string][]string{outcome.Field: {message}}
		}
		return result
	}
	return MutationResult{
		Success:    success,
		SaleID:     outcome.SaleID,
		PurchaseID: outcome.PurchaseID,
	}
}

func (a *App) cloudCreateItem(in ItemInput) (MutationResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return MutationResult{}, err
	}
	valid, fieldErrors := validateItem(in, false)
	if fieldErrors != nil {
		return MutationResult{FieldErrors: fieldErrors}, nil
	}

	var outcome rpcOutcome
	err := a.api.rpc(a.context(), "stockbook_create_item", map[string]any{
		"p_name":                valid.Name,
		"p_sku":                 stringOrEmpty(valid.SKU),
		"p_category":            valid.Category,
		"p_purchase_price":      valid.PurchasePrice,
		"p_sell_price":          valid.SellPrice,
		"p_low_stock_threshold": valid.LowStockThreshold,
	}, &outcome)
	if err != nil {
		return MutationResult{}, err
	}
	return mutationOutcome(outcome, ""), nil
}

func (a *App) cloudUpdateItem(in ItemInput) (MutationResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return MutationResult{}, err
	}
	valid, fieldErrors := validateItem(in, true)
	if fieldErrors != nil {
		return MutationResult{FieldErrors: fieldErrors}, nil
	}

	var outcome rpcOutcome
	err := a.api.rpc(a.context(), "stockbook_update_item", map[string]any{
		"p_product_id":          in.ProductID,
		"p_name":                valid.Name,
		"p_sku":                 stringOrEmpty(valid.SKU),
		"p_category":            valid.Category,
		"p_purchase_price":      valid.PurchasePrice,
		"p_sell_price":          valid.SellPrice,
		"p_low_stock_threshold": valid.LowStockThreshold,
	}, &outcome)
	if err != nil {
		return MutationResult{}, err
	}
	return mutationOutcome(outcome, ""), nil
}

func (a *App) cloudProduct(in IDInput) (ProductResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return ProductResult{}, err
	}
	id, ok := parseRecordID(in.ID)
	if !ok {
		return ProductResult{}, nil
	}
	product, err := a.product(id)
	if err != nil {
		return ProductResult{}, err
	}
	return ProductResult{Product: product}, nil
}

func (a *App) cloudProductOptions() ([]ProductOption, error) {
	if _, err := a.requireCompany(); err != nil {
		return nil, err
	}
	return a.productOptions()
}

func (a *App) cloudCategories() ([]string, error) {
	if _, err := a.requireCompany(); err != nil {
		return nil, err
	}
	return a.categories()
}

func (a *App) cloudCreatePurchase(in PurchaseInput) (MutationResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return MutationResult{}, err
	}
	valid, fieldErrors := validatePurchase(in)
	if fieldErrors != nil {
		return MutationResult{FieldErrors: fieldErrors}, nil
	}

	purchasedAt := nowISO()
	if valid.PurchasedAt != todayISO() {
		if parsed, ok := parseISODate(valid.PurchasedAt); ok {
			noon := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 12, 0, 0, 0, time.Local)
			purchasedAt = jsISO(noon)
		}
	}

	var outcome rpcOutcome
	err := a.api.rpc(a.context(), "stockbook_create_purchase", map[string]any{
		"p_product_id":   valid.ProductID,
		"p_quantity":     valid.Quantity,
		"p_unit_cost":    valid.UnitCost,
		"p_purchased_at": purchasedAt,
		"p_note":         stringOrEmpty(valid.Note),
	}, &outcome)
	if err != nil {
		return MutationResult{}, err
	}

	success := "Purchase recorded."
	if outcome.ProductName != "" {
		success = "Stock updated for " + outcome.ProductName + ". Purchase recorded."
	}
	return mutationOutcome(outcome, success), nil
}

func (a *App) cloudDeletePurchase(in IDInput) (MutationResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return MutationResult{}, err
	}
	id, ok := parseRecordID(in.ID)
	if !ok {
		return MutationResult{Error: "That purchase record could not be found."}, nil
	}

	var outcome rpcOutcome
	if err := a.api.rpc(a.context(), "stockbook_delete_purchase",
		map[string]any{"p_purchase_id": id}, &outcome); err != nil {
		return MutationResult{}, err
	}

	success := "Purchase deleted. Stock updated."
	if outcome.ProductName != "" {
		success = "Purchase for " + outcome.ProductName + " deleted. Stock updated."
	}
	return mutationOutcome(outcome, success), nil
}

func (a *App) cloudCreateSale(in SaleInput) (MutationResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return MutationResult{}, err
	}
	valid, fieldErrors := validateSale(in)
	if fieldErrors != nil {
		return MutationResult{FieldErrors: fieldErrors}, nil
	}

	var outcome rpcOutcome
	err := a.api.rpc(a.context(), "stockbook_create_sale", map[string]any{
		"p_product_id":   valid.ProductID,
		"p_quantity":     valid.Quantity,
		"p_unit_price":   valid.UnitPrice,
		"p_payment_mode": valid.PaymentMode,
		"p_note":         stringOrEmpty(valid.Note),
	}, &outcome)
	if err != nil {
		return MutationResult{}, err
	}
	return mutationOutcome(outcome, "Sale recorded. Stock updated."), nil
}

func (a *App) cloudDeleteSale(in IDInput) (MutationResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return MutationResult{}, err
	}
	id, ok := parseRecordID(in.ID)
	if !ok {
		return MutationResult{Error: "That sale record could not be found."}, nil
	}

	var outcome rpcOutcome
	if err := a.api.rpc(a.context(), "stockbook_delete_sale",
		map[string]any{"p_sale_id": id}, &outcome); err != nil {
		return MutationResult{}, err
	}

	success := "Sale deleted. Stock restored."
	if outcome.ProductName != "" {
		success = "Sale of " + outcome.ProductName + " deleted. Stock restored."
	}
	return mutationOutcome(outcome, success), nil
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
