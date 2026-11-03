import { useMemo } from "react";
import { api } from "../api";
import { CashflowPanel } from "../components/CashflowPanel";
import { DeleteRecordButton } from "../components/DeleteRecordButton";
import { Pagination } from "../components/Pagination";
import { RangeFilter } from "../components/RangeFilter";
import { filterQuery, readPage, readRangeFilters, toRangeInput } from "../filters";
import {
  CURRENCY,
  formatDateTime,
  formatMoney,
  formatNumber,
  formatPercent,
} from "../format";
import { useLoader } from "../hooks";
import { useRefresh } from "../refresh";
import { Link, useRouter } from "../router";

export function SalesPage() {
  const { location } = useRouter();
  const { refreshKey, refresh } = useRefresh();
  const key = location.query.toString();
  const filters = useMemo(() => readRangeFilters(location.query), [key]);
  const page = readPage(location.query.get("page"));
  const currency = CURRENCY;

  const state = useLoader(
    () => api.Sales(toRangeInput(filters, page)),
    [key, refreshKey],
  );

  const params = useMemo(() => filterQuery(filters), [filters]);
  const data = state.data;

  const margin = data?.summary.revenue
    ? (data.summary.profit / data.summary.revenue) * 100
    : 0;
  const averageSale = data?.summary.count
    ? Math.round(data.summary.revenue / data.summary.count)
    : 0;
  const unitsPerSale = data?.summary.count
    ? data.summary.units / data.summary.count
    : 0;
  const topCategory = data?.categoryBreakdown[0]?.category ?? "—";
  const maxCategoryRevenue = Math.max(
    ...(data?.categoryBreakdown.map((category) => category.revenue) ?? []),
    0,
  );

  return (
    <>
      <div className="page-head">
        <div>
          <h1 className="page-title">Sales</h1>
          <p className="page-sub">Every sale, its cost basis, and the profit it made.</p>
        </div>
      </div>

      <RangeFilter pathname="/sales" filters={filters} categories={data?.categories ?? []} />

      {state.error ? (
        <div className="form-error" role="alert">
          {state.error}
        </div>
      ) : null}

      {!data ? (
        <div className="panel boot-panel">Loading sales…</div>
      ) : (
        <>
          <div className="stat-strip">
            <div className="stat">
              <span className="stat-label">Revenue</span>
              <div className="stat-value num">
                {formatMoney(data.summary.revenue, currency)}
              </div>
              <div className="stat-meta">
                {formatNumber(data.summary.count)} sale
                {data.summary.count === 1 ? "" : "s"}
              </div>
            </div>
            <div className="stat">
              <span className="stat-label">Gross profit</span>
              <div className={`stat-value num${data.summary.profit < 0 ? " negative" : ""}`}>
                {formatMoney(data.summary.profit, currency)}
              </div>
              <div className="stat-meta">
                {data.summary.revenue ? `${formatPercent(margin)} margin` : "No sales in range"}
              </div>
            </div>
            <div className="stat">
              <span className="stat-label">Cost of goods</span>
              <div className="stat-value num">{formatMoney(data.summary.cost, currency)}</div>
              <div className="stat-meta">At cost when sold</div>
            </div>
            <div className="stat">
              <span className="stat-label">Units sold</span>
              <div className="stat-value num">{formatNumber(data.summary.units)}</div>
              <div className="stat-meta">{unitsPerSale.toFixed(1)} units per sale</div>
            </div>
          </div>

          <CashflowPanel
            cash={data.summary.cashRevenue}
            upi={data.summary.upiRevenue}
            cashCount={data.summary.cashCount}
            upiCount={data.summary.upiCount}
            currency={currency}
          />

          <section className="panel">
            <div className="section-head">
              <div>
                <h2 className="section-title">Category performance</h2>
                <p className="section-sub">Revenue and profit in this range</p>
              </div>
              <div className="meta-chips">
                <span className="chip">
                  Avg sale{" "}
                  <strong className="num">{formatMoney(averageSale, currency)}</strong>
                </span>
                <span className="chip">
                  Margin{" "}
                  <strong className="num">
                    {data.summary.revenue ? formatPercent(margin) : "—"}
                  </strong>
                </span>
                <span className="chip">
                  Units/sale{" "}
                  <strong className="num">
                    {data.summary.count ? unitsPerSale.toFixed(1) : "—"}
                  </strong>
                </span>
                <span className="chip">
                  Top <strong>{topCategory}</strong>
                </span>
              </div>
            </div>
            {data.categoryBreakdown.length > 0 ? (
              <div className="category-grid">
                {data.categoryBreakdown.map((category) => (
                  <div className="category-item" key={category.category}>
                    <div className="category-top">
                      <span className="category-name">{category.category}</span>
                      <span className="category-value num">
                        {formatMoney(category.revenue, currency)}
                      </span>
                    </div>
                    <div className="category-meta num">
                      {formatMoney(category.profit, currency)} profit ·{" "}
                      {formatNumber(category.units)} unit
                      {category.units === 1 ? "" : "s"}
                    </div>
                    <div className="category-bar" aria-hidden="true">
                      <span
                        style={{
                          width: `${
                            maxCategoryRevenue > 0
                              ? Math.max(4, (category.revenue / maxCategoryRevenue) * 100)
                              : 0
                          }%`,
                        }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty-state">No sales in this range.</div>
            )}
          </section>

          <section className="panel panel-spaced">
            <div className="section-head">
              <div>
                <h2 className="section-title">Sales history</h2>
                <p className="section-sub">Newest first · cost captured at sale time</p>
              </div>
              <span className="badge">
                {formatNumber(data.summary.count)} sale
                {data.summary.count === 1 ? "" : "s"}
              </span>
            </div>

            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Date</th>
                    <th>Item</th>
                    <th className="right">Qty</th>
                    <th className="right">Unit price</th>
                    <th className="right hide-sm">Cost</th>
                    <th className="right">Total</th>
                    <th className="right">Profit</th>
                    <th className="right">
                      <span className="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {data.rows.length > 0 ? (
                    data.rows.map((sale) => (
                      <tr key={sale.id}>
                        <td className="nowrap">{formatDateTime(sale.soldAt)}</td>
                        <td>
                          <div className="row-title">{sale.productName}</div>
                          <div className="row-sub">
                            {sale.paymentMode === "upi" ? "UPI" : "Cash"} · {sale.category}
                            {sale.note ? ` · ${sale.note}` : ""}
                          </div>
                        </td>
                        <td className="right num">{sale.quantity}</td>
                        <td className="right num">
                          {formatMoney(sale.unitPrice, currency)}
                        </td>
                        <td className="right num hide-sm">
                          {formatMoney(sale.unitCost * sale.quantity, currency)}
                        </td>
                        <td className="right num">
                          {formatMoney(sale.totalPrice, currency)}
                        </td>
                        <td className={`right num${sale.profit < 0 ? " negative" : ""}`}>
                          {formatMoney(sale.profit, currency)}
                        </td>
                        <td className="right">
                          <DeleteRecordButton
                            kind="sale"
                            id={sale.id}
                            productName={sale.productName}
                            quantity={sale.quantity}
                            summary={`${formatDateTime(sale.soldAt)} · ${formatMoney(
                              sale.totalPrice,
                              currency,
                            )}`}
                            onDeleted={refresh}
                          />
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td className="table-empty" colSpan={8}>
                        No sales in this range. Use <strong>New sell</strong> to record one.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            <Pagination
              pathname="/sales"
              params={params}
              page={data.page}
              totalPages={data.totalPages}
              total={data.summary.count}
              label={data.summary.count === 1 ? "sale" : "sales"}
            />
          </section>
        </>
      )}
    </>
  );
}
