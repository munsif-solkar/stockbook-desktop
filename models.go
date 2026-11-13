package main

// Company is the signed-in workspace account.
type Company struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type SessionResult struct {
	Authenticated bool     `json:"authenticated"`
	Company       *Company `json:"company"`
	Error         string   `json:"error,omitempty"`
}

type RegisterInput struct {
	CompanyName     string `json:"companyName"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResult struct {
	Company     *Company            `json:"company"`
	FieldErrors map[string][]string `json:"fieldErrors,omitempty"`
	Error       string              `json:"error,omitempty"`
}

type RangeFilters struct {
	From     *string `json:"from"`
	To       *string `json:"to"`
	AllTime  bool    `json:"allTime"`
	Category *string `json:"category"`
}

type RangePageInput struct {
	From     string `json:"from"`
	To       string `json:"to"`
	AllTime  bool   `json:"allTime"`
	Category string `json:"category"`
	Page     int    `json:"page"`
}

type ItemsInput struct {
	Search   string `json:"search"`
	Category string `json:"category"`
	Page     int    `json:"page"`
}

type InventoryInput struct {
	Search   string `json:"search"`
	Category string `json:"category"`
	LowStock bool   `json:"lowStock"`
	Page     int    `json:"page"`
}

type ProductOption struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	SKU               *string `json:"sku"`
	Category          string  `json:"category"`
	Quantity          int64   `json:"quantity"`
	PurchasePrice     int64   `json:"purchasePrice"`
	SellPrice         int64   `json:"sellPrice"`
	LowStockThreshold int64   `json:"lowStockThreshold"`
}

type InventorySummary struct {
	Count       int64 `json:"count"`
	Units       int64 `json:"units"`
	CostValue   int64 `json:"costValue"`
	RetailValue int64 `json:"retailValue"`
}

type ItemsSummary struct {
	Count      int64 `json:"count"`
	Categories int64 `json:"categories"`
	AvgCost    int64 `json:"avgCost"`
	AvgSell    int64 `json:"avgSell"`
}

type StockSummary struct {
	Count int64 `json:"count"`
	Units int64 `json:"units"`
	Spend int64 `json:"spend"`
}

type SalesSummary struct {
	Revenue     int64 `json:"revenue"`
	Cost        int64 `json:"cost"`
	Profit      int64 `json:"profit"`
	Units       int64 `json:"units"`
	CashRevenue int64 `json:"cashRevenue"`
	UPIRevenue  int64 `json:"upiRevenue"`
	CashCount   int64 `json:"cashCount"`
	UPICount    int64 `json:"upiCount"`
	Count       int64 `json:"count"`
}

type PurchaseSummary struct {
	Spend int64 `json:"spend"`
	Units int64 `json:"units"`
	Count int64 `json:"count"`
}

type InventoryTotalSummary struct {
	Units     int64 `json:"units"`
	CostValue int64 `json:"costValue"`
	Products  int64 `json:"products"`
}

type TrendRow struct {
	Day     string
	Revenue int64
	Units   int64
}

type TrendPoint struct {
	Date    string `json:"date"`
	Revenue int64  `json:"revenue"`
	Units   int64  `json:"units"`
}

type TopProduct struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Units    int64  `json:"units"`
	Revenue  int64  `json:"revenue"`
	Profit   int64  `json:"profit"`
}

type LowStockItem struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Quantity  int64  `json:"quantity"`
	Threshold int64  `json:"threshold"`
}

type RecentSale struct {
	ID          int64  `json:"id"`
	ProductName string `json:"productName"`
	Quantity    int64  `json:"quantity"`
	TotalPrice  int64  `json:"totalPrice"`
	Profit      int64  `json:"profit"`
	PaymentMode string `json:"paymentMode"`
	SoldAt      string `json:"soldAt"`
}

type OverviewResult struct {
	Filters     RangeFilters          `json:"filters"`
	Categories  []string              `json:"categories"`
	Sales       SalesSummary          `json:"sales"`
	Purchases   PurchaseSummary       `json:"purchases"`
	Inventory   InventoryTotalSummary `json:"inventory"`
	Trend       []TrendPoint          `json:"trend"`
	TopProducts []TopProduct          `json:"topProducts"`
	LowStock    []LowStockItem        `json:"lowStock"`
	RecentSales []RecentSale          `json:"recentSales"`
}

type ItemsResult struct {
	Items      []ProductOption `json:"items"`
	Summary    ItemsSummary    `json:"summary"`
	Page       int             `json:"page"`
	TotalPages int             `json:"totalPages"`
	PageSize   int             `json:"pageSize"`
	Categories []string        `json:"categories"`
}

type InventoryResult struct {
	Products   []ProductOption  `json:"products"`
	Summary    InventorySummary `json:"summary"`
	Page       int              `json:"page"`
	TotalPages int              `json:"totalPages"`
	PageSize   int              `json:"pageSize"`
	Categories []string         `json:"categories"`
}

type PurchaseRow struct {
	ID          int64   `json:"id"`
	ProductName string  `json:"productName"`
	SKU         *string `json:"sku"`
	Category    string  `json:"category"`
	Quantity    int64   `json:"quantity"`
	UnitCost    int64   `json:"unitCost"`
	TotalCost   int64   `json:"totalCost"`
	PurchasedAt string  `json:"purchasedAt"`
	Note        *string `json:"note"`
}

type PurchasesResult struct {
	Rows       []PurchaseRow   `json:"rows"`
	Summary    PurchaseSummary `json:"summary"`
	Page       int             `json:"page"`
	TotalPages int             `json:"totalPages"`
	PageSize   int             `json:"pageSize"`
	Categories []string        `json:"categories"`
	Filters    RangeFilters    `json:"filters"`
}

type SaleRow struct {
	ID          int64   `json:"id"`
	ProductName string  `json:"productName"`
	Category    string  `json:"category"`
	Quantity    int64   `json:"quantity"`
	UnitPrice   int64   `json:"unitPrice"`
	TotalPrice  int64   `json:"totalPrice"`
	UnitCost    int64   `json:"unitCost"`
	Profit      int64   `json:"profit"`
	PaymentMode string  `json:"paymentMode"`
	SoldAt      string  `json:"soldAt"`
	Note        *string `json:"note"`
}

type CategoryBreakdown struct {
	Category string `json:"category"`
	Revenue  int64  `json:"revenue"`
	Profit   int64  `json:"profit"`
	Units    int64  `json:"units"`
}

type SalesResult struct {
	Rows              []SaleRow           `json:"rows"`
	Summary           SalesSummary        `json:"summary"`
	CategoryBreakdown []CategoryBreakdown `json:"categoryBreakdown"`
	Page              int                 `json:"page"`
	TotalPages        int                 `json:"totalPages"`
	PageSize          int                 `json:"pageSize"`
	Categories        []string            `json:"categories"`
	Filters           RangeFilters        `json:"filters"`
}

type ItemInput struct {
	ProductID         int64  `json:"productId"`
	Name              string `json:"name"`
	SKU               string `json:"sku"`
	Category          string `json:"category"`
	LowStockThreshold string `json:"lowStockThreshold"`
	PurchasePrice     string `json:"purchasePrice"`
	SellPrice         string `json:"sellPrice"`
}

type PurchaseInput struct {
	ProductID   string `json:"productId"`
	Quantity    string `json:"quantity"`
	UnitCost    string `json:"unitCost"`
	PurchasedAt string `json:"purchasedAt"`
	Note        string `json:"note"`
}

type SaleInput struct {
	ProductID   string `json:"productId"`
	Quantity    string `json:"quantity"`
	UnitPrice   string `json:"unitPrice"`
	PaymentMode string `json:"paymentMode"`
	Note        string `json:"note"`
}

type IDInput struct {
	ID string `json:"id"`
}

type ProductResult struct {
	Product *ProductOption `json:"product"`
}

type MutationResult struct {
	Success     string              `json:"success,omitempty"`
	Error       string              `json:"error,omitempty"`
	FieldErrors map[string][]string `json:"fieldErrors,omitempty"`
	SaleID      int64               `json:"saleId,omitempty"`
	PurchaseID  int64               `json:"purchaseId,omitempty"`
}

// rpcOutcome is the shared shape returned by the stockbook_* write functions.
type rpcOutcome struct {
	OK          bool   `json:"ok"`
	Error       string `json:"error"`
	Field       string `json:"field"`
	SaleID      int64  `json:"saleId"`
	PurchaseID  int64  `json:"purchaseId"`
	ProductName string `json:"productName"`
}

type ExportResult struct {
	Canceled bool   `json:"canceled,omitempty"`
	Path     string `json:"path,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ImportResult struct {
	Canceled     bool   `json:"canceled,omitempty"`
	Imported     bool   `json:"imported,omitempty"`
	SafetyBackup string `json:"safetyBackup,omitempty"`
	Error        string `json:"error,omitempty"`
}

type UpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseName     string `json:"releaseName,omitempty"`
	ReleaseURL      string `json:"releaseUrl,omitempty"`
	PublishedAt     string `json:"publishedAt,omitempty"`
	Notes           string `json:"notes,omitempty"`
	AssetName       string `json:"assetName,omitempty"`
	AssetSize       int64  `json:"assetSize,omitempty"`
	Error           string `json:"error,omitempty"`
}

type UpdateApplyResult struct {
	Updated    bool   `json:"updated,omitempty"`
	Restarting bool   `json:"restarting,omitempty"`
	Error      string `json:"error,omitempty"`
}
