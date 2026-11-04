export namespace main {
	
	export class Company {
	    id: number;
	    name: string;
	    email: string;
	
	    static createFrom(source: any = {}) {
	        return new Company(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.email = source["email"];
	    }
	}
	export class AuthResult {
	    company?: Company;
	    fieldErrors?: Record<string, Array<string>>;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.company = this.convertValues(source["company"], Company);
	        this.fieldErrors = source["fieldErrors"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupResult {
	    canceled?: boolean;
	    path?: string;
	    size?: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canceled = source["canceled"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.error = source["error"];
	    }
	}
	export class CategoryBreakdown {
	    category: string;
	    revenue: number;
	    profit: number;
	    units: number;
	
	    static createFrom(source: any = {}) {
	        return new CategoryBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.revenue = source["revenue"];
	        this.profit = source["profit"];
	        this.units = source["units"];
	    }
	}
	
	export class IDInput {
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new IDInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class InventoryInput {
	    search: string;
	    category: string;
	    lowStock: boolean;
	    page: number;
	
	    static createFrom(source: any = {}) {
	        return new InventoryInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.search = source["search"];
	        this.category = source["category"];
	        this.lowStock = source["lowStock"];
	        this.page = source["page"];
	    }
	}
	export class InventorySummary {
	    count: number;
	    units: number;
	    costValue: number;
	    retailValue: number;
	
	    static createFrom(source: any = {}) {
	        return new InventorySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.units = source["units"];
	        this.costValue = source["costValue"];
	        this.retailValue = source["retailValue"];
	    }
	}
	export class ProductOption {
	    id: number;
	    name: string;
	    sku?: string;
	    category: string;
	    quantity: number;
	    purchasePrice: number;
	    sellPrice: number;
	    lowStockThreshold: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sku = source["sku"];
	        this.category = source["category"];
	        this.quantity = source["quantity"];
	        this.purchasePrice = source["purchasePrice"];
	        this.sellPrice = source["sellPrice"];
	        this.lowStockThreshold = source["lowStockThreshold"];
	    }
	}
	export class InventoryResult {
	    products: ProductOption[];
	    summary: InventorySummary;
	    page: number;
	    totalPages: number;
	    pageSize: number;
	    categories: string[];
	
	    static createFrom(source: any = {}) {
	        return new InventoryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.products = this.convertValues(source["products"], ProductOption);
	        this.summary = this.convertValues(source["summary"], InventorySummary);
	        this.page = source["page"];
	        this.totalPages = source["totalPages"];
	        this.pageSize = source["pageSize"];
	        this.categories = source["categories"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class InventoryTotalSummary {
	    units: number;
	    costValue: number;
	    products: number;
	
	    static createFrom(source: any = {}) {
	        return new InventoryTotalSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.units = source["units"];
	        this.costValue = source["costValue"];
	        this.products = source["products"];
	    }
	}
	export class ItemInput {
	    productId: number;
	    name: string;
	    sku: string;
	    category: string;
	    lowStockThreshold: string;
	    purchasePrice: string;
	    sellPrice: string;
	
	    static createFrom(source: any = {}) {
	        return new ItemInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productId = source["productId"];
	        this.name = source["name"];
	        this.sku = source["sku"];
	        this.category = source["category"];
	        this.lowStockThreshold = source["lowStockThreshold"];
	        this.purchasePrice = source["purchasePrice"];
	        this.sellPrice = source["sellPrice"];
	    }
	}
	export class ItemsInput {
	    search: string;
	    category: string;
	    page: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.search = source["search"];
	        this.category = source["category"];
	        this.page = source["page"];
	    }
	}
	export class ItemsSummary {
	    count: number;
	    categories: number;
	    avgCost: number;
	    avgSell: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemsSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.categories = source["categories"];
	        this.avgCost = source["avgCost"];
	        this.avgSell = source["avgSell"];
	    }
	}
	export class ItemsResult {
	    items: ProductOption[];
	    summary: ItemsSummary;
	    page: number;
	    totalPages: number;
	    pageSize: number;
	    categories: string[];
	
	    static createFrom(source: any = {}) {
	        return new ItemsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ProductOption);
	        this.summary = this.convertValues(source["summary"], ItemsSummary);
	        this.page = source["page"];
	        this.totalPages = source["totalPages"];
	        this.pageSize = source["pageSize"];
	        this.categories = source["categories"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class LoginInput {
	    email: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = source["email"];
	        this.password = source["password"];
	    }
	}
	export class LowStockItem {
	    id: number;
	    name: string;
	    quantity: number;
	    threshold: number;
	
	    static createFrom(source: any = {}) {
	        return new LowStockItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.quantity = source["quantity"];
	        this.threshold = source["threshold"];
	    }
	}
	export class MutationResult {
	    success?: string;
	    error?: string;
	    fieldErrors?: Record<string, Array<string>>;
	    saleId?: number;
	    purchaseId?: number;
	
	    static createFrom(source: any = {}) {
	        return new MutationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.fieldErrors = source["fieldErrors"];
	        this.saleId = source["saleId"];
	        this.purchaseId = source["purchaseId"];
	    }
	}
	export class RecentSale {
	    id: number;
	    productName: string;
	    quantity: number;
	    totalPrice: number;
	    profit: number;
	    paymentMode: string;
	    soldAt: string;
	
	    static createFrom(source: any = {}) {
	        return new RecentSale(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.productName = source["productName"];
	        this.quantity = source["quantity"];
	        this.totalPrice = source["totalPrice"];
	        this.profit = source["profit"];
	        this.paymentMode = source["paymentMode"];
	        this.soldAt = source["soldAt"];
	    }
	}
	export class TopProduct {
	    id: number;
	    name: string;
	    category: string;
	    units: number;
	    revenue: number;
	    profit: number;
	
	    static createFrom(source: any = {}) {
	        return new TopProduct(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.units = source["units"];
	        this.revenue = source["revenue"];
	        this.profit = source["profit"];
	    }
	}
	export class TrendPoint {
	    date: string;
	    revenue: number;
	    units: number;
	
	    static createFrom(source: any = {}) {
	        return new TrendPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.revenue = source["revenue"];
	        this.units = source["units"];
	    }
	}
	export class PurchaseSummary {
	    spend: number;
	    units: number;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new PurchaseSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spend = source["spend"];
	        this.units = source["units"];
	        this.count = source["count"];
	    }
	}
	export class SalesSummary {
	    revenue: number;
	    cost: number;
	    profit: number;
	    units: number;
	    cashRevenue: number;
	    upiRevenue: number;
	    cashCount: number;
	    upiCount: number;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new SalesSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revenue = source["revenue"];
	        this.cost = source["cost"];
	        this.profit = source["profit"];
	        this.units = source["units"];
	        this.cashRevenue = source["cashRevenue"];
	        this.upiRevenue = source["upiRevenue"];
	        this.cashCount = source["cashCount"];
	        this.upiCount = source["upiCount"];
	        this.count = source["count"];
	    }
	}
	export class RangeFilters {
	    from?: string;
	    to?: string;
	    allTime: boolean;
	    category?: string;
	
	    static createFrom(source: any = {}) {
	        return new RangeFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.allTime = source["allTime"];
	        this.category = source["category"];
	    }
	}
	export class OverviewResult {
	    filters: RangeFilters;
	    categories: string[];
	    sales: SalesSummary;
	    purchases: PurchaseSummary;
	    inventory: InventoryTotalSummary;
	    trend: TrendPoint[];
	    topProducts: TopProduct[];
	    lowStock: LowStockItem[];
	    recentSales: RecentSale[];
	
	    static createFrom(source: any = {}) {
	        return new OverviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filters = this.convertValues(source["filters"], RangeFilters);
	        this.categories = source["categories"];
	        this.sales = this.convertValues(source["sales"], SalesSummary);
	        this.purchases = this.convertValues(source["purchases"], PurchaseSummary);
	        this.inventory = this.convertValues(source["inventory"], InventoryTotalSummary);
	        this.trend = this.convertValues(source["trend"], TrendPoint);
	        this.topProducts = this.convertValues(source["topProducts"], TopProduct);
	        this.lowStock = this.convertValues(source["lowStock"], LowStockItem);
	        this.recentSales = this.convertValues(source["recentSales"], RecentSale);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ProductResult {
	    product?: ProductOption;
	
	    static createFrom(source: any = {}) {
	        return new ProductResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product = this.convertValues(source["product"], ProductOption);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PurchaseInput {
	    productId: string;
	    quantity: string;
	    unitCost: string;
	    purchasedAt: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new PurchaseInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productId = source["productId"];
	        this.quantity = source["quantity"];
	        this.unitCost = source["unitCost"];
	        this.purchasedAt = source["purchasedAt"];
	        this.note = source["note"];
	    }
	}
	export class PurchaseRow {
	    id: number;
	    productName: string;
	    sku?: string;
	    category: string;
	    quantity: number;
	    unitCost: number;
	    totalCost: number;
	    purchasedAt: string;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new PurchaseRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.productName = source["productName"];
	        this.sku = source["sku"];
	        this.category = source["category"];
	        this.quantity = source["quantity"];
	        this.unitCost = source["unitCost"];
	        this.totalCost = source["totalCost"];
	        this.purchasedAt = source["purchasedAt"];
	        this.note = source["note"];
	    }
	}
	
	export class PurchasesResult {
	    rows: PurchaseRow[];
	    summary: PurchaseSummary;
	    page: number;
	    totalPages: number;
	    pageSize: number;
	    categories: string[];
	    filters: RangeFilters;
	
	    static createFrom(source: any = {}) {
	        return new PurchasesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], PurchaseRow);
	        this.summary = this.convertValues(source["summary"], PurchaseSummary);
	        this.page = source["page"];
	        this.totalPages = source["totalPages"];
	        this.pageSize = source["pageSize"];
	        this.categories = source["categories"];
	        this.filters = this.convertValues(source["filters"], RangeFilters);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RangePageInput {
	    from: string;
	    to: string;
	    allTime: boolean;
	    category: string;
	    page: number;
	
	    static createFrom(source: any = {}) {
	        return new RangePageInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.allTime = source["allTime"];
	        this.category = source["category"];
	        this.page = source["page"];
	    }
	}
	
	export class RegisterInput {
	    companyName: string;
	    email: string;
	    password: string;
	    confirmPassword: string;
	
	    static createFrom(source: any = {}) {
	        return new RegisterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.companyName = source["companyName"];
	        this.email = source["email"];
	        this.password = source["password"];
	        this.confirmPassword = source["confirmPassword"];
	    }
	}
	export class RestoreResult {
	    canceled?: boolean;
	    restored?: boolean;
	    safetyBackup?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RestoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canceled = source["canceled"];
	        this.restored = source["restored"];
	        this.safetyBackup = source["safetyBackup"];
	        this.error = source["error"];
	    }
	}
	export class SaleInput {
	    productId: string;
	    quantity: string;
	    unitPrice: string;
	    paymentMode: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new SaleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productId = source["productId"];
	        this.quantity = source["quantity"];
	        this.unitPrice = source["unitPrice"];
	        this.paymentMode = source["paymentMode"];
	        this.note = source["note"];
	    }
	}
	export class SaleRow {
	    id: number;
	    productName: string;
	    category: string;
	    quantity: number;
	    unitPrice: number;
	    totalPrice: number;
	    unitCost: number;
	    profit: number;
	    paymentMode: string;
	    soldAt: string;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new SaleRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.productName = source["productName"];
	        this.category = source["category"];
	        this.quantity = source["quantity"];
	        this.unitPrice = source["unitPrice"];
	        this.totalPrice = source["totalPrice"];
	        this.unitCost = source["unitCost"];
	        this.profit = source["profit"];
	        this.paymentMode = source["paymentMode"];
	        this.soldAt = source["soldAt"];
	        this.note = source["note"];
	    }
	}
	export class SalesResult {
	    rows: SaleRow[];
	    summary: SalesSummary;
	    categoryBreakdown: CategoryBreakdown[];
	    page: number;
	    totalPages: number;
	    pageSize: number;
	    categories: string[];
	    filters: RangeFilters;
	
	    static createFrom(source: any = {}) {
	        return new SalesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], SaleRow);
	        this.summary = this.convertValues(source["summary"], SalesSummary);
	        this.categoryBreakdown = this.convertValues(source["categoryBreakdown"], CategoryBreakdown);
	        this.page = source["page"];
	        this.totalPages = source["totalPages"];
	        this.pageSize = source["pageSize"];
	        this.categories = source["categories"];
	        this.filters = this.convertValues(source["filters"], RangeFilters);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SessionResult {
	    authenticated: boolean;
	    company?: Company;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.company = this.convertValues(source["company"], Company);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class UpdateApplyResult {
	    updated?: boolean;
	    restarting?: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updated = source["updated"];
	        this.restarting = source["restarting"];
	        this.error = source["error"];
	    }
	}
	export class UpdateInfo {
	    currentVersion: string;
	    latestVersion?: string;
	    updateAvailable: boolean;
	    releaseName?: string;
	    releaseUrl?: string;
	    publishedAt?: string;
	    notes?: string;
	    assetName?: string;
	    assetSize?: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.releaseName = source["releaseName"];
	        this.releaseUrl = source["releaseUrl"];
	        this.publishedAt = source["publishedAt"];
	        this.notes = source["notes"];
	        this.assetName = source["assetName"];
	        this.assetSize = source["assetSize"];
	        this.error = source["error"];
	    }
	}

}

