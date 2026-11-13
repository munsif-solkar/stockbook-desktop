package main

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

func escapeLike(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return replacer.Replace(value)
}

func clampPage(page int, totalPages int) int {
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	return page
}

func totalPagesFor(count int64, pageSize int) int {
	pages := int((count + int64(pageSize) - 1) / int64(pageSize))
	if pages < 1 {
		return 1
	}
	return pages
}

type whereClause struct {
	SQL    string
	Params []any
}

func salesWhere(companyID int64, filters RangeFilters) whereClause {
	clauses := []string{"s.company_id = ?"}
	params := []any{companyID}
	if filters.From != nil {
		clauses = append(clauses, "s.sold_at >= ?")
		params = append(params, startOfLocalDay(*filters.From))
	}
	if filters.To != nil {
		clauses = append(clauses, "s.sold_at <= ?")
		params = append(params, endOfLocalDay(*filters.To))
	}
	if filters.Category != nil {
		clauses = append(clauses, "p.category = ? COLLATE NOCASE")
		params = append(params, *filters.Category)
	}
	return whereClause{SQL: strings.Join(clauses, " AND "), Params: params}
}

func purchaseWhere(companyID int64, filters RangeFilters) whereClause {
	clauses := []string{"pu.company_id = ?"}
	params := []any{companyID}
	if filters.From != nil {
		clauses = append(clauses, "pu.purchased_at >= ?")
		params = append(params, startOfLocalDay(*filters.From))
	}
	if filters.To != nil {
		clauses = append(clauses, "pu.purchased_at <= ?")
		params = append(params, endOfLocalDay(*filters.To))
	}
	if filters.Category != nil {
		clauses = append(clauses, "p.category = ? COLLATE NOCASE")
		params = append(params, *filters.Category)
	}
	return whereClause{SQL: strings.Join(clauses, " AND "), Params: params}
}

func (a *App) localCategories(companyID int64) ([]string, error) {
	rows, err := a.local.db.Query(
		`SELECT DISTINCT category FROM products
		 WHERE company_id = ?
		 ORDER BY category COLLATE NOCASE ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []string{}
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func scanProductOptions(rows *sql.Rows) ([]ProductOption, error) {
	products := []ProductOption{}
	for rows.Next() {
		var product ProductOption
		var sku sql.NullString
		if err := rows.Scan(
			&product.ID, &product.Name, &sku, &product.Category, &product.Quantity,
			&product.PurchasePrice, &product.SellPrice, &product.LowStockThreshold,
		); err != nil {
			return nil, err
		}
		if sku.Valid {
			value := sku.String
			product.SKU = &value
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

func (a *App) localProductOptions(companyID int64) ([]ProductOption, error) {
	rows, err := a.local.db.Query(
		`SELECT id, name, sku, category, quantity,
		        purchase_price, sell_price, low_stock_threshold
		 FROM products
		 WHERE company_id = ?
		 ORDER BY name COLLATE NOCASE ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProductOptions(rows)
}

func (a *App) localProduct(companyID, productID int64) (*ProductOption, error) {
	rows, err := a.local.db.Query(
		`SELECT id, name, sku, category, quantity,
		        purchase_price, sell_price, low_stock_threshold
		 FROM products
		 WHERE id = ? AND company_id = ?
		 LIMIT 1`, productID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products, err := scanProductOptions(rows)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, nil
	}
	return &products[0], nil
}

func (a *App) localOverview(in RangePageInput) (OverviewResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return OverviewResult{}, err
	}
	filters := normalizeFilters(in)
	result := OverviewResult{Filters: filters}

	categories, err := a.localCategories(company.ID)
	if err != nil {
		return OverviewResult{}, err
	}
	result.Categories = categories

	salesFilter := salesWhere(company.ID, filters)
	salesQuery := fmt.Sprintf(
		`SELECT COALESCE(SUM(s.total_price), 0),
		        COALESCE(SUM(s.total_cost), 0),
		        COALESCE(SUM(s.profit), 0),
		        COALESCE(SUM(s.quantity), 0),
		        COALESCE(SUM(CASE WHEN s.payment_mode = 'cash' THEN s.total_price ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN s.payment_mode = 'upi' THEN s.total_price ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN s.payment_mode = 'cash' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN s.payment_mode = 'upi' THEN 1 ELSE 0 END), 0),
		        COUNT(*)
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE %s`, salesFilter.SQL)
	if err := a.local.db.QueryRow(salesQuery, salesFilter.Params...).Scan(
		&result.Sales.Revenue, &result.Sales.Cost, &result.Sales.Profit, &result.Sales.Units,
		&result.Sales.CashRevenue, &result.Sales.UPIRevenue,
		&result.Sales.CashCount, &result.Sales.UPICount, &result.Sales.Count,
	); err != nil {
		return OverviewResult{}, err
	}

	purchaseFilter := purchaseWhere(company.ID, filters)
	purchaseQuery := fmt.Sprintf(
		`SELECT COALESCE(SUM(pu.total_cost), 0),
		        COALESCE(SUM(pu.quantity), 0),
		        COUNT(*)
		 FROM purchases pu
		 JOIN products p ON p.id = pu.product_id
		 WHERE %s`, purchaseFilter.SQL)
	if err := a.local.db.QueryRow(purchaseQuery, purchaseFilter.Params...).Scan(
		&result.Purchases.Spend, &result.Purchases.Units, &result.Purchases.Count,
	); err != nil {
		return OverviewResult{}, err
	}

	if err := a.local.db.QueryRow(
		`SELECT COALESCE(SUM(quantity), 0),
		        COALESCE(SUM(quantity * purchase_price), 0),
		        COUNT(*)
		 FROM products WHERE company_id = ?`, company.ID,
	).Scan(&result.Inventory.Units, &result.Inventory.CostValue, &result.Inventory.Products); err != nil {
		return OverviewResult{}, err
	}

	trendQuery := fmt.Sprintf(
		`SELECT date(s.sold_at, 'localtime') AS day,
		        COALESCE(SUM(s.total_price), 0),
		        COALESCE(SUM(s.quantity), 0)
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE %s
		 GROUP BY day
		 ORDER BY day ASC`, salesFilter.SQL)
	trendRows := []TrendRow{}
	rows, err := a.local.db.Query(trendQuery, salesFilter.Params...)
	if err != nil {
		return OverviewResult{}, err
	}
	for rows.Next() {
		var row TrendRow
		if err := rows.Scan(&row.Day, &row.Revenue, &row.Units); err != nil {
			rows.Close()
			return OverviewResult{}, err
		}
		trendRows = append(trendRows, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return OverviewResult{}, err
	}
	rows.Close()
	result.Trend = buildTrend(trendRows, filters)

	topQuery := fmt.Sprintf(
		`SELECT p.id, p.name, p.category,
		        COALESCE(SUM(s.quantity), 0) AS units,
		        COALESCE(SUM(s.total_price), 0) AS revenue,
		        COALESCE(SUM(s.profit), 0) AS profit
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE %s
		 GROUP BY p.id
		 ORDER BY revenue DESC, units DESC
		 LIMIT 5`, salesFilter.SQL)
	topRows, err := a.local.db.Query(topQuery, salesFilter.Params...)
	if err != nil {
		return OverviewResult{}, err
	}
	result.TopProducts = []TopProduct{}
	for topRows.Next() {
		var product TopProduct
		if err := topRows.Scan(&product.ID, &product.Name, &product.Category, &product.Units, &product.Revenue, &product.Profit); err != nil {
			topRows.Close()
			return OverviewResult{}, err
		}
		result.TopProducts = append(result.TopProducts, product)
	}
	if err := topRows.Err(); err != nil {
		topRows.Close()
		return OverviewResult{}, err
	}
	topRows.Close()

	lowRows, err := a.local.db.Query(
		`SELECT id, name, quantity, low_stock_threshold
		 FROM products
		 WHERE company_id = ? AND quantity <= low_stock_threshold
		 ORDER BY quantity ASC, name COLLATE NOCASE ASC
		 LIMIT 6`, company.ID)
	if err != nil {
		return OverviewResult{}, err
	}
	result.LowStock = []LowStockItem{}
	for lowRows.Next() {
		var item LowStockItem
		if err := lowRows.Scan(&item.ID, &item.Name, &item.Quantity, &item.Threshold); err != nil {
			lowRows.Close()
			return OverviewResult{}, err
		}
		result.LowStock = append(result.LowStock, item)
	}
	if err := lowRows.Err(); err != nil {
		lowRows.Close()
		return OverviewResult{}, err
	}
	lowRows.Close()

	recentQuery := fmt.Sprintf(
		`SELECT s.id, p.name, s.quantity, s.total_price, s.profit, s.payment_mode, s.sold_at
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE %s
		 ORDER BY s.sold_at DESC, s.id DESC
		 LIMIT 6`, salesFilter.SQL)
	recentRows, err := a.local.db.Query(recentQuery, salesFilter.Params...)
	if err != nil {
		return OverviewResult{}, err
	}
	result.RecentSales = []RecentSale{}
	for recentRows.Next() {
		var sale RecentSale
		if err := recentRows.Scan(&sale.ID, &sale.ProductName, &sale.Quantity, &sale.TotalPrice, &sale.Profit, &sale.PaymentMode, &sale.SoldAt); err != nil {
			recentRows.Close()
			return OverviewResult{}, err
		}
		result.RecentSales = append(result.RecentSales, sale)
	}
	if err := recentRows.Err(); err != nil {
		recentRows.Close()
		return OverviewResult{}, err
	}
	recentRows.Close()

	return result, nil
}

func (a *App) localItems(in ItemsInput) (ItemsResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return ItemsResult{}, err
	}
	search := cleanLimited(in.Search, 80)
	category := cleanLimited(in.Category, 50)

	clauses := []string{"company_id = ?"}
	params := []any{company.ID}
	if search != nil {
		query := "%" + escapeLike(*search) + "%"
		clauses = append(clauses, `(name LIKE ? ESCAPE '\' OR COALESCE(sku, '') LIKE ? ESCAPE '\' OR category LIKE ? ESCAPE '\')`)
		params = append(params, query, query, query)
	}
	if category != nil {
		clauses = append(clauses, "category = ? COLLATE NOCASE")
		params = append(params, *category)
	}
	where := strings.Join(clauses, " AND ")

	var summary ItemsSummary
	var avgCost, avgSell float64
	if err := a.local.db.QueryRow(fmt.Sprintf(
		`SELECT COUNT(*), COUNT(DISTINCT category),
		        COALESCE(ROUND(AVG(purchase_price)), 0),
		        COALESCE(ROUND(AVG(sell_price)), 0)
		 FROM products WHERE %s`, where), params...,
	).Scan(&summary.Count, &summary.Categories, &avgCost, &avgSell); err != nil {
		return ItemsResult{}, err
	}
	summary.AvgCost = int64(avgCost)
	summary.AvgSell = int64(avgSell)

	const pageSize = 50
	totalPages := totalPagesFor(summary.Count, pageSize)
	page := clampPage(in.Page, totalPages)

	rows, err := a.local.db.Query(fmt.Sprintf(
		`SELECT id, name, sku, category, quantity,
		        purchase_price, sell_price, low_stock_threshold
		 FROM products
		 WHERE %s
		 ORDER BY created_at DESC, id DESC
		 LIMIT ? OFFSET ?`, where), append(params, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return ItemsResult{}, err
	}
	items, err := scanProductOptions(rows)
	rows.Close()
	if err != nil {
		return ItemsResult{}, err
	}

	categories, err := a.localCategories(company.ID)
	if err != nil {
		return ItemsResult{}, err
	}

	return ItemsResult{
		Items: items, Summary: summary, Page: page, TotalPages: totalPages,
		PageSize: pageSize, Categories: categories,
	}, nil
}

func (a *App) localInventory(in InventoryInput) (InventoryResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return InventoryResult{}, err
	}
	search := cleanLimited(in.Search, 80)
	category := cleanLimited(in.Category, 50)

	clauses := []string{"company_id = ?"}
	params := []any{company.ID}
	if search != nil {
		query := "%" + escapeLike(*search) + "%"
		clauses = append(clauses, `(name LIKE ? ESCAPE '\' OR COALESCE(sku, '') LIKE ? ESCAPE '\' OR category LIKE ? ESCAPE '\')`)
		params = append(params, query, query, query)
	}
	if category != nil {
		clauses = append(clauses, "category = ? COLLATE NOCASE")
		params = append(params, *category)
	}
	if in.LowStock {
		clauses = append(clauses, "quantity <= low_stock_threshold")
	}
	where := strings.Join(clauses, " AND ")

	var summary InventorySummary
	if err := a.local.db.QueryRow(fmt.Sprintf(
		`SELECT COUNT(*), COALESCE(SUM(quantity), 0),
		        COALESCE(SUM(quantity * purchase_price), 0),
		        COALESCE(SUM(quantity * sell_price), 0)
		 FROM products WHERE %s`, where), params...,
	).Scan(&summary.Count, &summary.Units, &summary.CostValue, &summary.RetailValue); err != nil {
		return InventoryResult{}, err
	}

	const pageSize = 50
	totalPages := totalPagesFor(summary.Count, pageSize)
	page := clampPage(in.Page, totalPages)

	rows, err := a.local.db.Query(fmt.Sprintf(
		`SELECT id, name, sku, category, quantity,
		        purchase_price, sell_price, low_stock_threshold
		 FROM products
		 WHERE %s
		 ORDER BY CASE
		            WHEN quantity = 0 THEN 0
		            WHEN quantity <= low_stock_threshold THEN 1
		            ELSE 2
		          END,
		          name COLLATE NOCASE ASC
		 LIMIT ? OFFSET ?`, where), append(params, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return InventoryResult{}, err
	}
	products, err := scanProductOptions(rows)
	rows.Close()
	if err != nil {
		return InventoryResult{}, err
	}

	categories, err := a.localCategories(company.ID)
	if err != nil {
		return InventoryResult{}, err
	}

	return InventoryResult{
		Products: products, Summary: summary, Page: page, TotalPages: totalPages,
		PageSize: pageSize, Categories: categories,
	}, nil
}

func (a *App) localPurchases(in RangePageInput) (PurchasesResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return PurchasesResult{}, err
	}
	filters := normalizeFilters(in)
	filter := purchaseWhere(company.ID, filters)
	result := PurchasesResult{Filters: filters}

	if err := a.local.db.QueryRow(fmt.Sprintf(
		`SELECT COALESCE(SUM(pu.total_cost), 0),
		        COALESCE(SUM(pu.quantity), 0),
		        COUNT(*)
		 FROM purchases pu
		 JOIN products p ON p.id = pu.product_id
		 WHERE %s`, filter.SQL), filter.Params...,
	).Scan(&result.Summary.Spend, &result.Summary.Units, &result.Summary.Count); err != nil {
		return PurchasesResult{}, err
	}

	const pageSize = 25
	totalPages := totalPagesFor(result.Summary.Count, pageSize)
	page := clampPage(in.Page, totalPages)

	rows, err := a.local.db.Query(fmt.Sprintf(
		`SELECT pu.id, p.name, p.sku, p.category,
		        pu.quantity, pu.unit_cost, pu.total_cost,
		        pu.purchased_at, pu.note
		 FROM purchases pu
		 JOIN products p ON p.id = pu.product_id
		 WHERE %s
		 ORDER BY pu.purchased_at DESC, pu.id DESC
		 LIMIT ? OFFSET ?`, filter.SQL), append(filter.Params, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return PurchasesResult{}, err
	}
	result.Rows = []PurchaseRow{}
	for rows.Next() {
		var row PurchaseRow
		var sku, note sql.NullString
		if err := rows.Scan(&row.ID, &row.ProductName, &sku, &row.Category, &row.Quantity, &row.UnitCost, &row.TotalCost, &row.PurchasedAt, &note); err != nil {
			rows.Close()
			return PurchasesResult{}, err
		}
		if sku.Valid {
			value := sku.String
			row.SKU = &value
		}
		if note.Valid {
			value := note.String
			row.Note = &value
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PurchasesResult{}, err
	}
	rows.Close()

	categories, err := a.localCategories(company.ID)
	if err != nil {
		return PurchasesResult{}, err
	}
	result.Categories = categories
	result.Page = page
	result.TotalPages = totalPages
	result.PageSize = pageSize
	return result, nil
}

func (a *App) localSales(in RangePageInput) (SalesResult, error) {
	company, err := a.requireCompany()
	if err != nil {
		return SalesResult{}, err
	}
	filters := normalizeFilters(in)
	filter := salesWhere(company.ID, filters)
	result := SalesResult{Filters: filters}

	if err := a.local.db.QueryRow(fmt.Sprintf(
		`SELECT COALESCE(SUM(s.total_price), 0),
		        COALESCE(SUM(s.total_cost), 0),
		        COALESCE(SUM(s.profit), 0),
		        COALESCE(SUM(s.quantity), 0),
		        COALESCE(SUM(CASE WHEN s.payment_mode = 'cash' THEN s.total_price ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN s.payment_mode = 'upi' THEN s.total_price ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN s.payment_mode = 'cash' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN s.payment_mode = 'upi' THEN 1 ELSE 0 END), 0),
		        COUNT(*)
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE %s`, filter.SQL), filter.Params...,
	).Scan(
		&result.Summary.Revenue, &result.Summary.Cost, &result.Summary.Profit, &result.Summary.Units,
		&result.Summary.CashRevenue, &result.Summary.UPIRevenue,
		&result.Summary.CashCount, &result.Summary.UPICount, &result.Summary.Count,
	); err != nil {
		return SalesResult{}, err
	}

	const pageSize = 25
	totalPages := totalPagesFor(result.Summary.Count, pageSize)
	page := clampPage(in.Page, totalPages)

	rows, err := a.local.db.Query(fmt.Sprintf(
		`SELECT s.id, p.name, p.category,
		        s.quantity, s.unit_price, s.total_price,
		        s.unit_cost, s.profit, s.payment_mode,
		        s.sold_at, s.note
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE %s
		 ORDER BY s.sold_at DESC, s.id DESC
		 LIMIT ? OFFSET ?`, filter.SQL), append(filter.Params, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return SalesResult{}, err
	}
	result.Rows = []SaleRow{}
	for rows.Next() {
		var row SaleRow
		var note sql.NullString
		if err := rows.Scan(&row.ID, &row.ProductName, &row.Category, &row.Quantity, &row.UnitPrice, &row.TotalPrice, &row.UnitCost, &row.Profit, &row.PaymentMode, &row.SoldAt, &note); err != nil {
			rows.Close()
			return SalesResult{}, err
		}
		if note.Valid {
			value := note.String
			row.Note = &value
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return SalesResult{}, err
	}
	rows.Close()

	categoryRows, err := a.local.db.Query(fmt.Sprintf(
		`SELECT p.category,
		        COALESCE(SUM(s.total_price), 0) AS revenue,
		        COALESCE(SUM(s.profit), 0) AS profit,
		        COALESCE(SUM(s.quantity), 0) AS units
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE %s
		 GROUP BY p.id
		 ORDER BY revenue DESC`, filter.SQL), filter.Params...)
	if err != nil {
		return SalesResult{}, err
	}
	keys := []string{}
	merged := map[string]*CategoryBreakdown{}
	for categoryRows.Next() {
		var row CategoryBreakdown
		if err := categoryRows.Scan(&row.Category, &row.Revenue, &row.Profit, &row.Units); err != nil {
			categoryRows.Close()
			return SalesResult{}, err
		}
		key := strings.ToLower(row.Category)
		if existing, ok := merged[key]; ok {
			existing.Revenue += row.Revenue
			existing.Profit += row.Profit
			existing.Units += row.Units
		} else {
			copy := row
			merged[key] = &copy
			keys = append(keys, key)
		}
	}
	if err := categoryRows.Err(); err != nil {
		categoryRows.Close()
		return SalesResult{}, err
	}
	categoryRows.Close()

	breakdown := make([]CategoryBreakdown, 0, len(keys))
	for _, key := range keys {
		breakdown = append(breakdown, *merged[key])
	}
	sort.SliceStable(breakdown, func(i, j int) bool {
		return breakdown[i].Revenue > breakdown[j].Revenue
	})
	if len(breakdown) > 6 {
		breakdown = breakdown[:6]
	}
	result.CategoryBreakdown = breakdown

	categories, err := a.localCategories(company.ID)
	if err != nil {
		return SalesResult{}, err
	}
	result.Categories = categories
	result.Page = page
	result.TotalPages = totalPages
	result.PageSize = pageSize
	return result, nil
}
