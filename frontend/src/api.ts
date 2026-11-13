export type Company = {
  id: number;
  name: string;
  email: string;
};

export type SessionResult = {
  authenticated: boolean;
  company: Company | null;
  error?: string;
};

export type AuthResult = {
  company: Company | null;
  fieldErrors?: FieldErrors;
  error?: string;
};

export type FieldErrors = Record<string, string[] | undefined>;

export type RangeFilters = {
  from: string | null;
  to: string | null;
  allTime: boolean;
  category: string | null;
};

export type RangePageInput = {
  from: string;
  to: string;
  allTime: boolean;
  category: string;
  page: number;
};

export type ItemsInput = {
  search: string;
  category: string;
  page: number;
};

export type InventoryInput = {
  search: string;
  category: string;
  lowStock: boolean;
  page: number;
};

export type ProductOption = {
  id: number;
  name: string;
  sku: string | null;
  category: string;
  quantity: number;
  purchasePrice: number;
  sellPrice: number;
  lowStockThreshold: number;
};

export type InventorySummary = {
  count: number;
  units: number;
  costValue: number;
  retailValue: number;
};

export type ItemsSummary = {
  count: number;
  categories: number;
  avgCost: number;
  avgSell: number;
};

export type SalesSummary = {
  revenue: number;
  cost: number;
  profit: number;
  units: number;
  cashRevenue: number;
  upiRevenue: number;
  cashCount: number;
  upiCount: number;
  count: number;
};

export type PurchaseSummary = {
  spend: number;
  units: number;
  count: number;
};

export type TrendPoint = { date: string; revenue: number; units: number };
export type TopProduct = {
  id: number;
  name: string;
  category: string;
  units: number;
  revenue: number;
  profit: number;
};
export type LowStockItem = {
  id: number;
  name: string;
  quantity: number;
  threshold: number;
};
export type RecentSale = {
  id: number;
  productName: string;
  quantity: number;
  totalPrice: number;
  profit: number;
  paymentMode: string;
  soldAt: string;
};

export type OverviewResult = {
  filters: RangeFilters;
  categories: string[];
  sales: SalesSummary;
  purchases: PurchaseSummary;
  inventory: { units: number; costValue: number; products: number };
  trend: TrendPoint[];
  topProducts: TopProduct[];
  lowStock: LowStockItem[];
  recentSales: RecentSale[];
};

export type ItemsResult = {
  items: ProductOption[];
  summary: ItemsSummary;
  page: number;
  totalPages: number;
  pageSize: number;
  categories: string[];
};

export type InventoryResult = {
  products: ProductOption[];
  summary: InventorySummary;
  page: number;
  totalPages: number;
  pageSize: number;
  categories: string[];
};

export type PurchaseRow = {
  id: number;
  productName: string;
  sku: string | null;
  category: string;
  quantity: number;
  unitCost: number;
  totalCost: number;
  purchasedAt: string;
  note: string | null;
};

export type PurchasesResult = {
  rows: PurchaseRow[];
  summary: PurchaseSummary;
  page: number;
  totalPages: number;
  pageSize: number;
  categories: string[];
  filters: RangeFilters;
};

export type SaleRow = {
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
  note: string | null;
};

export type CategoryBreakdown = {
  category: string;
  revenue: number;
  profit: number;
  units: number;
};

export type SalesResult = {
  rows: SaleRow[];
  summary: SalesSummary;
  categoryBreakdown: CategoryBreakdown[];
  page: number;
  totalPages: number;
  pageSize: number;
  categories: string[];
  filters: RangeFilters;
};

export type ItemInput = {
  productId: number;
  name: string;
  sku: string;
  category: string;
  lowStockThreshold: string;
  purchasePrice: string;
  sellPrice: string;
};

export type PurchaseInput = {
  productId: string;
  quantity: string;
  unitCost: string;
  purchasedAt: string;
  note: string;
};

export type SaleInput = {
  productId: string;
  quantity: string;
  unitPrice: string;
  paymentMode: string;
  note: string;
};

export type MutationResult = {
  success?: string;
  error?: string;
  fieldErrors?: FieldErrors;
  saleId?: number;
  purchaseId?: number;
};

export type ProductResult = { product: ProductOption | null };

export type BackupResult = {
  canceled?: boolean;
  path?: string;
  size?: number;
  error?: string;
};

export type RestoreResult = {
  canceled?: boolean;
  restored?: boolean;
  safetyBackup?: string;
  error?: string;
};

export type UpdateInfo = {
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
};

export type UpdateApplyResult = {
  updated?: boolean;
  restarting?: boolean;
  error?: string;
};


type Bridge = Record<string, (...args: unknown[]) => Promise<unknown>>;

function bridge(): Bridge | null {
  const value = (window as unknown as { go?: { main?: { App?: Bridge } } }).go;
  return value?.main?.App ?? null;
}

function errorMessage(error: unknown): string {
  if (typeof error === "string") return error;
  if (error && typeof error === "object" && "message" in error) {
    return String((error as { message: unknown }).message);
  }
  return "Something went wrong. Please try again.";
}

async function invoke<T>(method: string, ...args: unknown[]): Promise<T> {
  const api = bridge();
  if (!api || typeof api[method] !== "function") {
    throw new Error(
      "Stockbook's desktop bridge is unavailable. Open the desktop app to reach your data.",
    );
  }
  try {
    return (await api[method](...args)) as T;
  } catch (error) {
    const message = errorMessage(error);
    if (message.toLowerCase().includes("unauthorized")) {
      window.dispatchEvent(new Event("stockbook:unauthorized"));
      throw new Error("Your session has expired. Sign in again.");
    }
    throw new Error(message);
  }
}

export const api = {
  Session: () => invoke<SessionResult>("Session"),
  Register: (input: {
    companyName: string;
    email: string;
    password: string;
    confirmPassword: string;
  }) => invoke<AuthResult>("Register", input),
  Login: (input: { email: string; password: string }) =>
    invoke<AuthResult>("Login", input),
  Logout: () => invoke<void>("Logout"),
  Overview: (input: RangePageInput) => invoke<OverviewResult>("Overview", input),
  Items: (input: ItemsInput) => invoke<ItemsResult>("Items", input),
  Inventory: (input: InventoryInput) =>
    invoke<InventoryResult>("Inventory", input),
  Purchases: (input: RangePageInput) =>
    invoke<PurchasesResult>("Purchases", input),
  Sales: (input: RangePageInput) => invoke<SalesResult>("Sales", input),
  Product: (id: number) => invoke<ProductResult>("Product", { id: String(id) }),
  ProductOptions: () => invoke<ProductOption[]>("ProductOptions"),
  CreateItem: (input: ItemInput) => invoke<MutationResult>("CreateItem", input),
  UpdateItem: (input: ItemInput) => invoke<MutationResult>("UpdateItem", input),
  CreatePurchase: (input: PurchaseInput) =>
    invoke<MutationResult>("CreatePurchase", input),
  DeletePurchase: (id: number) =>
    invoke<MutationResult>("DeletePurchase", { id: String(id) }),
  CreateSale: (input: SaleInput) => invoke<MutationResult>("CreateSale", input),
  DeleteSale: (id: number) =>
    invoke<MutationResult>("DeleteSale", { id: String(id) }),
  DataLocation: () => invoke<string>("DataLocation"),
  Version: () => invoke<string>("Version"),
  BackupDatabase: () => invoke<BackupResult>("BackupDatabase"),
  RestoreDatabase: () => invoke<RestoreResult>("RestoreDatabase"),
  CheckForUpdates: () => invoke<UpdateInfo>("CheckForUpdates"),
  ApplyUpdate: () => invoke<UpdateApplyResult>("ApplyUpdate"),
};
