package main

import (
	"strings"
	"time"
)

func cleanLimited(value string, max int) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	runes := []rune(trimmed)
	if len(runes) > max {
		trimmed = string(runes[:max])
	}
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeFilters(in RangePageInput) RangeFilters {
	category := cleanLimited(in.Category, 50)

	if in.AllTime {
		return RangeFilters{AllTime: true, Category: category}
	}

	var from *string
	var to *string
	if value := strings.TrimSpace(in.From); isISODate(value) {
		from = &value
	}
	if value := strings.TrimSpace(in.To); isISODate(value) {
		to = &value
	}

	if from == nil && to == nil {
		today := todayISO()
		start := addDaysISO(today, -29)
		return RangeFilters{From: &start, To: &today, Category: category}
	}

	if from != nil && to != nil && *from > *to {
		from, to = to, from
	}

	return RangeFilters{From: from, To: to, Category: category}
}

// filterArgs maps a normalized date range to the arguments expected by the
// stockbook_* SQL functions. Timestamps are sent as the same JS-style UTC
// strings the desktop app used to store locally.
func filterArgs(filters RangeFilters) map[string]any {
	args := map[string]any{
		"p_all_time": filters.AllTime,
		"p_from":     nil,
		"p_to":       nil,
		"p_category": nil,
	}
	if filters.From != nil {
		args["p_from"] = startOfLocalDay(*filters.From)
	}
	if filters.To != nil {
		args["p_to"] = endOfLocalDay(*filters.To)
	}
	if filters.Category != nil {
		args["p_category"] = *filters.Category
	}
	return args
}

func buildTrend(rows []TrendRow, filters RangeFilters) []TrendPoint {
	if len(rows) == 0 {
		return []TrendPoint{}
	}

	if filters.From != nil && filters.To != nil {
		start, errStart := time.Parse("2006-01-02", *filters.From)
		end, errEnd := time.Parse("2006-01-02", *filters.To)
		if errStart == nil && errEnd == nil {
			span := int(end.Sub(start).Hours() / 24)
			if span <= 45 {
				byDay := map[string]TrendRow{}
				for _, row := range rows {
					byDay[row.Day] = row
				}
				points := make([]TrendPoint, 0, span+1)
				for i := 0; i <= span; i++ {
					date := addDaysISO(*filters.From, i)
					row := byDay[date]
					points = append(points, TrendPoint{Date: date, Revenue: row.Revenue, Units: row.Units})
				}
				return points
			}
		}
	}

	firstDate, _ := time.Parse("2006-01-02", rows[0].Day)
	lastDate, _ := time.Parse("2006-01-02", rows[len(rows)-1].Day)
	span := int(lastDate.Sub(firstDate).Hours() / 24)

	if span > 400 {
		keys := []string{}
		byMonth := map[string]*TrendPoint{}
		for _, row := range rows {
			key := row.Day[:7]
			existing, ok := byMonth[key]
			if ok {
				existing.Revenue += row.Revenue
				existing.Units += row.Units
			} else {
				byMonth[key] = &TrendPoint{Date: key + "-01", Revenue: row.Revenue, Units: row.Units}
				keys = append(keys, key)
			}
		}
		points := make([]TrendPoint, 0, len(keys))
		for _, key := range keys {
			points = append(points, *byMonth[key])
		}
		return points
	}

	keys := []string{}
	byWeek := map[string]*TrendPoint{}
	for _, row := range rows {
		date, ok := parseISODate(row.Day)
		if !ok {
			continue
		}
		utcDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		mondayOffset := (int(utcDate.Weekday()) + 6) % 7
		utcDate = utcDate.AddDate(0, 0, -mondayOffset)
		key := utcDate.Format("2006-01-02")
		existing, ok := byWeek[key]
		if ok {
			existing.Revenue += row.Revenue
			existing.Units += row.Units
		} else {
			byWeek[key] = &TrendPoint{Date: key, Revenue: row.Revenue, Units: row.Units}
			keys = append(keys, key)
		}
	}
	points := make([]TrendPoint, 0, len(keys))
	for _, key := range keys {
		points = append(points, *byWeek[key])
	}
	return points
}

func (a *App) categories() ([]string, error) {
	categories := []string{}
	if err := a.api.rpc(a.context(), "stockbook_categories", map[string]any{}, &categories); err != nil {
		return nil, err
	}
	if categories == nil {
		categories = []string{}
	}
	return categories, nil
}

func (a *App) productOptions() ([]ProductOption, error) {
	products := []ProductOption{}
	if err := a.api.rpc(a.context(), "stockbook_product_options", map[string]any{}, &products); err != nil {
		return nil, err
	}
	if products == nil {
		products = []ProductOption{}
	}
	return products, nil
}

func (a *App) product(productID int64) (*ProductOption, error) {
	var product *ProductOption
	if err := a.api.rpc(a.context(), "stockbook_product",
		map[string]any{"p_product_id": productID}, &product); err != nil {
		return nil, err
	}
	return product, nil
}

func (a *App) Overview(in RangePageInput) (OverviewResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return OverviewResult{}, err
	}

	filters := normalizeFilters(in)
	args := filterArgs(filters)
	_, offsetSeconds := time.Now().Zone()
	args["p_tz_offset"] = offsetSeconds / 60

	var result OverviewResult
	if err := a.api.rpc(a.context(), "stockbook_overview", args, &result); err != nil {
		return OverviewResult{}, err
	}

	result.Filters = filters
	trendRows := make([]TrendRow, 0, len(result.Trend))
	for _, point := range result.Trend {
		trendRows = append(trendRows, TrendRow{Day: point.Date, Revenue: point.Revenue, Units: point.Units})
	}
	result.Trend = buildTrend(trendRows, filters)

	if result.Categories == nil {
		result.Categories = []string{}
	}
	if result.TopProducts == nil {
		result.TopProducts = []TopProduct{}
	}
	if result.LowStock == nil {
		result.LowStock = []LowStockItem{}
	}
	if result.RecentSales == nil {
		result.RecentSales = []RecentSale{}
	}
	return result, nil
}

func (a *App) Items(in ItemsInput) (ItemsResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return ItemsResult{}, err
	}

	search := cleanLimited(in.Search, 80)
	category := cleanLimited(in.Category, 50)

	args := map[string]any{
		"p_search":    nil,
		"p_category":  nil,
		"p_page":      in.Page,
		"p_page_size": 50,
	}
	if search != nil {
		args["p_search"] = *search
	}
	if category != nil {
		args["p_category"] = *category
	}

	var result ItemsResult
	if err := a.api.rpc(a.context(), "stockbook_items", args, &result); err != nil {
		return ItemsResult{}, err
	}
	if result.Items == nil {
		result.Items = []ProductOption{}
	}
	if result.Categories == nil {
		result.Categories = []string{}
	}
	return result, nil
}

func (a *App) Inventory(in InventoryInput) (InventoryResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return InventoryResult{}, err
	}

	search := cleanLimited(in.Search, 80)
	category := cleanLimited(in.Category, 50)

	args := map[string]any{
		"p_search":    nil,
		"p_category":  nil,
		"p_low_stock": in.LowStock,
		"p_page":      in.Page,
		"p_page_size": 50,
	}
	if search != nil {
		args["p_search"] = *search
	}
	if category != nil {
		args["p_category"] = *category
	}

	var result InventoryResult
	if err := a.api.rpc(a.context(), "stockbook_inventory", args, &result); err != nil {
		return InventoryResult{}, err
	}
	if result.Products == nil {
		result.Products = []ProductOption{}
	}
	if result.Categories == nil {
		result.Categories = []string{}
	}
	return result, nil
}

func (a *App) Purchases(in RangePageInput) (PurchasesResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return PurchasesResult{}, err
	}

	filters := normalizeFilters(in)
	args := filterArgs(filters)
	args["p_page"] = in.Page
	args["p_page_size"] = 25

	var result PurchasesResult
	if err := a.api.rpc(a.context(), "stockbook_purchases", args, &result); err != nil {
		return PurchasesResult{}, err
	}
	result.Filters = filters
	if result.Rows == nil {
		result.Rows = []PurchaseRow{}
	}
	if result.Categories == nil {
		result.Categories = []string{}
	}
	return result, nil
}

func (a *App) Sales(in RangePageInput) (SalesResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return SalesResult{}, err
	}

	filters := normalizeFilters(in)
	args := filterArgs(filters)
	args["p_page"] = in.Page
	args["p_page_size"] = 25

	var result SalesResult
	if err := a.api.rpc(a.context(), "stockbook_sales", args, &result); err != nil {
		return SalesResult{}, err
	}
	result.Filters = filters
	if result.Rows == nil {
		result.Rows = []SaleRow{}
	}
	if result.CategoryBreakdown == nil {
		result.CategoryBreakdown = []CategoryBreakdown{}
	}
	if result.Categories == nil {
		result.Categories = []string{}
	}
	return result, nil
}
