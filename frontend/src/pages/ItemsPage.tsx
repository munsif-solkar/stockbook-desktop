import { useMemo, useState, type FormEvent } from "react";
import { api } from "../api";
import { ItemForm } from "../components/ItemForm";
import { Pagination } from "../components/Pagination";
import { formQuery, queryHref, readPage, readParam } from "../filters";
import { CURRENCY, formatMoney, formatNumber, formatPercent } from "../format";
import { useLoader } from "../hooks";
import { useRefresh } from "../refresh";
import { Link, navigate, useRouter } from "../router";
import { NotFoundPage } from "./NotFoundPage";

export function ItemsPage() {
  const { location } = useRouter();
  const { refreshKey } = useRefresh();
  const key = location.query.toString();

  const search = readParam(location.query.get("q"), 80);
  const category = readParam(location.query.get("category"), 50);
  const page = readPage(location.query.get("page"));
  const showForm = readParam(location.query.get("new"), 1) === "1";
  const editRaw = readParam(location.query.get("edit"), 20);
  const editId = editRaw && /^\d+$/.test(editRaw) ? Number(editRaw) : null;
  const saved = readParam(location.query.get("saved"), 1) === "1";
  const currency = CURRENCY;

  const state = useLoader(
    () => api.Items({ search: search ?? "", category: category ?? "", page }),
    [key, refreshKey],
  );
  const productState = useLoader(
    () =>
      editId && !showForm
        ? api.Product(editId)
        : Promise.resolve({ product: null }),
    [editId, showForm, key, refreshKey],
  );

  const params = useMemo(() => {
    const next = new URLSearchParams();
    if (search) next.set("q", search);
    if (category) next.set("category", category);
    return next;
  }, [search, category]);

  if (editId && !showForm && !productState.loading && !productState.data?.product) {
    return <NotFoundPage />;
  }

  const data = state.data;
  const editItem = productState.data?.product ?? null;

  function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    navigate(queryHref("/items", formQuery(event.currentTarget)));
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1 className="page-title">Items</h1>
          <p className="page-sub">Your catalog: names, SKUs, and default prices.</p>
        </div>
        {!showForm && !editItem ? (
          <div className="page-actions">
            <Link className="btn btn-primary btn-sm" to="/items?new=1">
              <svg
                className="btn-icon"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                aria-hidden="true"
              >
                <path d="M12 5v14M5 12h14" />
              </svg>
              Add item
            </Link>
          </div>
        ) : null}
      </div>

      {saved ? (
        <div className="notice" role="status">
          <span>Item saved. It is at the top of the catalog — add stock from Inventory when it arrives.</span>
          <Link to="/items">Dismiss</Link>
        </div>
      ) : null}

      {showForm ? (
        <ItemForm
          mode="create"
          categories={data?.categories ?? []}
          onSaved={() => navigate("/items?saved=1")}
        />
      ) : editItem ? (
        <ItemForm
          mode="edit"
          item={editItem}
          categories={data?.categories ?? []}
          onSaved={() => navigate("/items?saved=1")}
        />
      ) : null}

      <form
        key={key}
        className="toolbar"
        onSubmit={applyFilters}
        aria-label="Item filters"
      >
        <div className="toolbar-fields grow">
          <label className="sr-only" htmlFor="items-search">
            Search items
          </label>
          <input
            className="input input-sm toolbar-search"
            id="items-search"
            name="q"
            type="search"
            defaultValue={search ?? ""}
            placeholder="Search name, SKU, or category"
          />
          <label className="sr-only" htmlFor="items-category">
            Category
          </label>
          <select
            className="select select-sm"
            id="items-category"
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
        </div>
        <div className="toolbar-fields">
          <button className="btn btn-secondary btn-sm" type="submit">
            Apply
          </button>
          <Link className="btn btn-ghost btn-sm" to="/items">
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
        <div className="panel boot-panel">Loading items…</div>
      ) : (
        <>
          <div className="stat-strip">
            <div className="stat">
              <span className="stat-label">Items</span>
              <div className="stat-value num">{formatNumber(data.summary.count)}</div>
              <div className="stat-meta">Matching this view</div>
            </div>
            <div className="stat">
              <span className="stat-label">Categories</span>
              <div className="stat-value num">{formatNumber(data.summary.categories)}</div>
              <div className="stat-meta">Across matching items</div>
            </div>
            <div className="stat">
              <span className="stat-label">Avg purchase price</span>
              <div className="stat-value num">
                {formatMoney(data.summary.avgCost, currency)}
              </div>
              <div className="stat-meta">Default cost per item</div>
            </div>
            <div className="stat">
              <span className="stat-label">Avg selling price</span>
              <div className="stat-value num">
                {formatMoney(data.summary.avgSell, currency)}
              </div>
              <div className="stat-meta">Default retail per item</div>
            </div>
          </div>

          <section className="panel">
            <div className="section-head">
              <div>
                <h2 className="section-title">Catalog</h2>
                <p className="section-sub">
                  Newest first. Items exist without stock; use Inventory to add
                  units and record purchases.
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
                    <th>Pricing</th>
                    <th className="right">Stock</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {data.items.length > 0 ? (
                    data.items.map((item) => {
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
                          <td className="right num">{formatNumber(item.quantity)}</td>
                          <td className="right">
                            <div className="row-actions">
                              <Link
                                className="btn btn-secondary btn-sm"
                                to={`/inventory?stock=${item.id}`}
                              >
                                Add stock
                              </Link>
                              <Link className="btn btn-ghost btn-sm" to={`/items?edit=${item.id}`}>
                                Edit
                              </Link>
                            </div>
                          </td>
                        </tr>
                      );
                    })
                  ) : (
                    <tr>
                      <td className="table-empty" colSpan={5}>
                        {search || category ? (
                          <>
                            No items match these filters. <Link to="/items">Clear filters</Link>.
                          </>
                        ) : (
                          <>
                            No items yet. <Link to="/items?new=1">Add your first item</Link>.
                          </>
                        )}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            <Pagination
              pathname="/items"
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
