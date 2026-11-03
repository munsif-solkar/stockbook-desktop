import { useMemo } from "react";
import { api } from "../api";
import { DeleteRecordButton } from "../components/DeleteRecordButton";
import { Pagination } from "../components/Pagination";
import { PurchaseForm } from "../components/PurchaseForm";
import { RangeFilter } from "../components/RangeFilter";
import { filterQuery, readPage, readParam, readRangeFilters, toRangeInput } from "../filters";
import {
  CURRENCY,
  formatDateTime,
  formatMoney,
  formatNumber,
  todayISO,
} from "../format";
import { useLoader } from "../hooks";
import { useRefresh } from "../refresh";
import { Link, navigate, useRouter } from "../router";

export function PurchasePage() {
  const { location } = useRouter();
  const { refreshKey, refresh } = useRefresh();
  const key = location.query.toString();
  const filters = useMemo(() => readRangeFilters(location.query), [key]);
  const page = readPage(location.query.get("page"));
  const showForm = readParam(location.query.get("new"), 1) === "1";
  const saved = readParam(location.query.get("saved"), 1) === "1";
  const currency = CURRENCY;

  const state = useLoader(
    () => api.Purchases(toRangeInput(filters, page)),
    [key, refreshKey],
  );
  const optionsState = useLoader(
    () => (showForm ? api.ProductOptions() : Promise.resolve([])),
    [key, refreshKey, showForm],
  );

  const params = useMemo(() => filterQuery(filters), [filters]);

  const data = state.data;
  const averageCost = data?.summary.units
    ? Math.round(data.summary.spend / data.summary.units)
    : 0;

  return (
    <>
      <div className="page-head">
        <div>
          <h1 className="page-title">Purchases</h1>
          <p className="page-sub">Stock received, what it cost, and what came in.</p>
        </div>
        <div className="page-actions">
          <Link className="btn btn-primary btn-sm" to="/purchase?new=1">
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
            Record purchase
          </Link>
        </div>
      </div>

      {saved ? (
        <div className="notice" role="status">
          <span>Purchase recorded. Stock and cost have been updated.</span>
          <Link to="/purchase">Dismiss</Link>
        </div>
      ) : null}

      {showForm ? (
        <PurchaseForm
          products={optionsState.data ?? []}
          currency={currency}
          today={todayISO()}
          onSaved={() => navigate("/purchase?saved=1")}
        />
      ) : null}

      <RangeFilter pathname="/purchase" filters={filters} categories={data?.categories ?? []} />

      {state.error ? (
        <div className="form-error" role="alert">
          {state.error}
        </div>
      ) : null}

      {!data ? (
        <div className="panel boot-panel">Loading purchases…</div>
      ) : (
        <>
          <div className="stat-strip">
            <div className="stat">
              <span className="stat-label">Purchase spend</span>
              <div className="stat-value num">
                {formatMoney(data.summary.spend, currency)}
              </div>
              <div className="stat-meta">
                {formatNumber(data.summary.count)} entr
                {data.summary.count === 1 ? "y" : "ies"}
              </div>
            </div>
            <div className="stat">
              <span className="stat-label">Units purchased</span>
              <div className="stat-value num">{formatNumber(data.summary.units)}</div>
              <div className="stat-meta">In the selected range</div>
            </div>
            <div className="stat">
              <span className="stat-label">Average unit cost</span>
              <div className="stat-value num">{formatMoney(averageCost, currency)}</div>
              <div className="stat-meta">Across all purchased units</div>
            </div>
            <div className="stat">
              <span className="stat-label">Average purchase</span>
              <div className="stat-value num">
                {formatMoney(
                  data.summary.count
                    ? Math.round(data.summary.spend / data.summary.count)
                    : 0,
                  currency,
                )}
              </div>
              <div className="stat-meta">Spend per purchase entry</div>
            </div>
          </div>

          <section className="panel panel-spaced">
            <div className="section-head">
              <div>
                <h2 className="section-title">Purchase history</h2>
                <p className="section-sub">Newest first</p>
              </div>
              <span className="badge">
                {formatNumber(data.summary.count)} entr
                {data.summary.count === 1 ? "y" : "ies"}
              </span>
            </div>

            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Date</th>
                    <th>Item</th>
                    <th className="right">Qty</th>
                    <th className="right">Unit cost</th>
                    <th className="right">Total</th>
                    <th className="hide-sm">Note</th>
                    <th className="right">
                      <span className="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {data.rows.length > 0 ? (
                    data.rows.map((purchase) => (
                      <tr key={purchase.id}>
                        <td className="nowrap">
                          {formatDateTime(purchase.purchasedAt)}
                        </td>
                        <td>
                          <div className="row-title">{purchase.productName}</div>
                          <div className="row-sub">
                            {purchase.sku ? `${purchase.sku} · ` : ""}
                            {purchase.category}
                          </div>
                        </td>
                        <td className="right num">{purchase.quantity}</td>
                        <td className="right num">
                          {formatMoney(purchase.unitCost, currency)}
                        </td>
                        <td className="right num">
                          {formatMoney(purchase.totalCost, currency)}
                        </td>
                        <td className="muted hide-sm">{purchase.note ?? "—"}</td>
                        <td className="right">
                          <DeleteRecordButton
                            kind="purchase"
                            id={purchase.id}
                            productName={purchase.productName}
                            quantity={purchase.quantity}
                            summary={`${formatDateTime(purchase.purchasedAt)} · ${formatMoney(
                              purchase.totalCost,
                              currency,
                            )}`}
                            onDeleted={refresh}
                          />
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td className="table-empty" colSpan={7}>
                        No purchases in this range.{" "}
                        <Link to="/purchase?new=1">Record a purchase</Link>.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            <Pagination
              pathname="/purchase"
              params={params}
              page={data.page}
              totalPages={data.totalPages}
              total={data.summary.count}
              label={data.summary.count === 1 ? "purchase" : "purchases"}
            />
          </section>
        </>
      )}
    </>
  );
}
