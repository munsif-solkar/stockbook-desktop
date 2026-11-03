import { useMemo, useState, type FormEvent } from "react";
import { api } from "../api";
import { Pagination } from "../components/Pagination";
import { StockUpdateButton } from "../components/StockUpdateButton";
import {
  formQuery,
  queryHref,
  readPage,
  readParam,
} from "../filters";
import {
  CURRENCY,
  formatMoney,
  formatNumber,
  formatPercent,
  todayISO,
} from "../format";
import { useLoader } from "../hooks";
import { useRefresh } from "../refresh";
import { Link, navigate, useRouter } from "../router";

export function InventoryPage() {
  const { location } = useRouter();
  const { refreshKey, refresh } = useRefresh();
  const key = location.query.toString();

  const search = readParam(location.query.get("q"), 80);
  const category = readParam(location.query.get("category"), 50);
  const lowStock = readParam(location.query.get("low"), 1) === "1";
  const page = readPage(location.query.get("page"));
  const stockRaw = readParam(location.query.get("stock"), 20);
  const stockId = stockRaw && /^\d+$/.test(stockRaw) ? Number(stockRaw) : null;
  const currency = CURRENCY;

  const state = useLoader(
    () =>
      api.Inventory({
        search: search ?? "",
        category: category ?? "",
        lowStock,
        page,
      }),
    [key, refreshKey],
  );
  const optionsState = useLoader(
    () => api.ProductOptions(),
    [key, refreshKey],
  );

  const params = useMemo(() => {
    const next = new URLSearchParams();
    if (search) next.set("q", search);
    if (category) next.set("category", category);
    if (lowStock) next.set("low", "1");
    return next;
  }, [search, category, lowStock]);

  const data = state.data;

  function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    navigate(queryHref("/inventory", formQuery(event.currentTarget)));
  }

  function clearStockParam() {
    if (!location.query.has("stock")) return;
    const next = new URLSearchParams(location.query);
    next.delete("stock");
    navigate(queryHref("/inventory", next), true);
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1 className="page-title">Inventory</h1>
          <p className="page-sub">Stock on hand, what it cost, and what it is worth.</p>
        </div>
        <div className="page-actions">
          <Link className="btn btn-secondary btn-sm" to="/items">
            Manage items
          </Link>
          <StockUpdateButton
            items={optionsState.data ?? []}
            currency={currency}
            today={todayISO()}
            autoOpen={Boolean(stockId)}
            defaultItemId={stockId}
            onChanged={refresh}
            onAutoOpenHandled={clearStockParam}
          />
        </div>
      </div>

      <form
        key={key}
        className="toolbar"
        onSubmit={applyFilters}
        aria-label="Inventory filters"
      >
        <div className="toolbar-fields grow">
          <label className="sr-only" htmlFor="inventory-search">
            Search items
          </label>
          <input
            className="input input-sm toolbar-search"
            id="inventory-search"
            name="q"
            type="search"
            defaultValue={search ?? ""}
            placeholder="Search name, SKU, or category"
          />
          <label className="sr-only" htmlFor="inventory-category">
            Category
          </label>
          <select
            className="select select-sm"
            id="inventory-category"
            name="category"
            defaultValue={category ?? ""}
          >
            <option value="">All categories</option>
            {(data?.categories ?? []).map((item) => (
              <option key={item} value={item}>
                {item}
              </option>
            ))}
          </select>
          <label className="toggle" htmlFor="inventory-low">
            <input
              id="inventory-low"
              name="low"
              type="checkbox"
              value="1"
              defaultChecked={lowStock}
            />
            Low stock
          </label>
        </div>
        <div className="toolbar-fields">
          <button className="btn btn-secondary btn-sm" type="submit">
            Apply
          </button>
          <Link className="btn btn-ghost btn-sm" to="/inventory">
            Reset
          </Link>
        </div>
      </form>

      {state.error ? (
        <div className="form-error" role="alert">
          {state.error}
        </div>
      ) : null}

      {!data ? (
        <div className="panel boot-panel">Loading inventory…</div>
      ) : (
        <>
          <div className="stat-strip">
            <div className="stat">
              <span className="stat-label">Items</span>
              <div className="stat-value num">{formatNumber(data.summary.count)}</div>
              <div className="stat-meta">Matching this view</div>
            </div>
            <div className="stat">
              <span className="stat-label">Units on hand</span>
              <div className="stat-value num">{formatNumber(data.summary.units)}</div>
              <div className="stat-meta">Across matching items</div>
            </div>
            <div className="stat">
              <span className="stat-label">Stock value</span>
              <div className="stat-value num">
                {formatMoney(data.summary.costValue, currency)}
              </div>
              <div className="stat-meta">At purchase cost</div>
            </div>
            <div className="stat">
              <span className="stat-label">Retail value</span>
              <div className="stat-value num">
                {formatMoney(data.summary.retailValue, currency)}
              </div>
              <div className="stat-meta">At selling price</div>
            </div>
          </div>

          <section className="panel">
            <div className="section-head">
              <div>
                <h2 className="section-title">Stock</h2>
                <p className="section-sub">
                  {lowStock
                    ? "Showing low and out-of-stock items"
                    : "Update stock to add units and record a purchase."}
                </p>
              </div>
              <span className="badge">{formatNumber(data.summary.count)} total</span>
            </div>

            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Item</th>
                    <th>Category</th>
                    <th className="right">Stock</th>
                    <th>Pricing</th>
                    <th>Status</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {data.products.length > 0 ? (
                    data.products.map((item) => {
                      const out = item.quantity === 0;
                      const low = item.quantity <= item.lowStockThreshold;
                      const margin =
                        item.sellPrice > 0
                          ? ((item.sellPrice - item.purchasePrice) / item.sellPrice) * 100
                          : null;
                      return (
                        <tr key={item.id}>
                          <td>
                            <div className="row-title">{item.name}</div>
                            <div className="row-sub">{item.sku ?? "No SKU"}</div>
                          </td>
                          <td>{item.category}</td>
                          <td className="right num">{formatNumber(item.quantity)}</td>
                          <td>
                            <div className="num">
                              {formatMoney(item.purchasePrice, currency)}
                              <span className="muted"> → </span>
                              {formatMoney(item.sellPrice, currency)}
                            </div>
                            <div className="row-sub num">
                              {margin === null
                                ? "No margin"
                                : `${formatPercent(margin)} margin`}
                            </div>
                          </td>
                          <td>
                            <span
                              className={`badge ${
                                out ? "badge-out" : low ? "badge-low" : "badge-ok"
                              }`}
                            >
                              {out ? "Out of stock" : low ? "Low stock" : "In stock"}
                            </span>
                          </td>
                          <td className="right">
                            <div className="row-actions">
                              <Link
                                className="btn btn-secondary btn-sm"
                                to={`/inventory?stock=${item.id}`}
                              >
                                Add stock
                              </Link>
                              <Link className="btn btn-ghost btn-sm" to={`/items?edit=${item.id}`}>
                                Edit item
                              </Link>
                            </div>
                          </td>
                        </tr>
                      );
                    })
                  ) : (
                    <tr>
                      <td className="table-empty" colSpan={6}>
                        {search || category || lowStock ? (
                          <>
                            No items match these filters.{" "}
                            <Link to="/inventory">Clear filters</Link>.
                          </>
                        ) : (
                          <>
                            No items yet.{" "}
                            <Link to="/items?new=1">Add items to your catalog</Link>, then
                            update stock here.
                          </>
                        )}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            <Pagination
              pathname="/inventory"
              params={params}
              page={data.page}
              totalPages={data.totalPages}
              total={data.summary.count}
              label={data.summary.count === 1 ? "item" : "items"}
            />
          </section>
        </>
      )}
    </>
  );
}
