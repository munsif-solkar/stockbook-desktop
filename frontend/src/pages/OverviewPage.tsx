import { useMemo } from "react";
import type { OverviewResult } from "../api";
import { api } from "../api";
import { CashflowPanel } from "../components/CashflowPanel";
import { RangeFilter } from "../components/RangeFilter";
import { filterQuery, readRangeFilters, toRangeInput } from "../filters";
import {
  formatDate,
  formatDateTime,
  formatMoney,
  formatNumber,
  formatPercent,
  shortDate,
} from "../format";
import { CURRENCY } from "../format";
import { useLoader } from "../hooks";
import { useRefresh } from "../refresh";
import { Link, useRouter } from "../router";

function emptyOverview(): OverviewResult | null {
  return null;
}

export function OverviewPage() {
  const { location } = useRouter();
  const { refreshKey } = useRefresh();
  const key = location.query.toString();
  const filters = useMemo(() => readRangeFilters(location.query), [key]);
  const state = useLoader(
    () => api.Overview(toRangeInput(filters, 1)),
    [key, refreshKey],
  );
  const data = state.data ?? emptyOverview();
  const currency = CURRENCY;
  const categories = data?.categories ?? [];

  if (!data) {
    return (
      <>
        <div className="page-head">
          <div>
            <h1 className="page-title">Overview</h1>
            <p className="page-sub">Sales, purchases, and profit</p>
          </div>
        </div>
        {state.error ? (
          <div className="form-error" role="alert">
            {state.error}
          </div>
        ) : (
          <div className="panel boot-panel">Loading your ledger…</div>
        )}
      </>
    );
  }

  const query = filterQuery(filters).toString();
  const margin = data.sales.revenue
    ? (data.sales.profit / data.sales.revenue) * 100
    : 0;
  const maxRevenue = Math.max(...data.trend.map((point) => point.revenue), 0);
  const rangeLabel = filters.allTime
    ? "All time"
    : filters.from && filters.to
      ? `${formatDate(filters.from)} – ${formatDate(filters.to)}`
      : filters.from
        ? `Since ${formatDate(filters.from)}`
        : filters.to
          ? `Through ${formatDate(filters.to)}`
          : "Last 30 days";

  return (
    <>
      <div className="page-head">
        <div>
          <h1 className="page-title">Overview</h1>
          <p className="page-sub">Sales, purchases, and profit · {rangeLabel}</p>
        </div>
      </div>

      <RangeFilter pathname="/overview" filters={filters} categories={categories} />

      {state.error ? (
        <div className="form-error" role="alert">
          {state.error}
        </div>
      ) : null}

      {data.lowStock.length > 0 ? (
        <div className="alert-bar" role="status">
          <span>
            <strong>{data.lowStock.length}</strong>{" "}
            {data.lowStock.length === 1 ? "product is" : "products are"} low or out
            of stock
            {data.lowStock.length <= 2
              ? `: ${data.lowStock.map((item) => item.name).join(", ")}`
              : `, including ${data.lowStock[0].name}`}
            .
          </span>
          <Link className="alert-link" to="/inventory?low=1">
            Review stock
          </Link>
        </div>
      ) : null}

      <div className="stat-strip">
        <div className="stat">
          <span className="stat-label">Sales</span>
          <div className="stat-value num">{formatMoney(data.sales.revenue, currency)}</div>
          <div className="stat-meta">
            {formatNumber(data.sales.count)} sale{data.sales.count === 1 ? "" : "s"}
          </div>
        </div>
        <div className="stat">
          <span className="stat-label">Purchases</span>
          <div className="stat-value num">
            {formatMoney(data.purchases.spend, currency)}
          </div>
          <div className="stat-meta">
            {formatNumber(data.purchases.units)} units purchased
          </div>
        </div>
        <div className="stat">
          <span className="stat-label">Gross profit</span>
          <div className={`stat-value num${data.sales.profit < 0 ? " negative" : ""}`}>
            {formatMoney(data.sales.profit, currency)}
          </div>
          <div className="stat-meta">
            {data.sales.revenue ? `${formatPercent(margin)} margin` : "No sales in range"}
          </div>
        </div>
        <div className="stat">
          <span className="stat-label">Units sold</span>
          <div className="stat-value num">{formatNumber(data.sales.units)}</div>
          <div className="stat-meta">
            Cost of goods {formatMoney(data.sales.cost, currency)}
          </div>
        </div>
      </div>

      <CashflowPanel
        cash={data.sales.cashRevenue}
        upi={data.sales.upiRevenue}
        cashCount={data.sales.cashCount}
        upiCount={data.sales.upiCount}
        currency={currency}
      />

      <div className="dash-grid">
        <section className="panel">
          <div className="section-head">
            <div>
              <h2 className="section-title">Sales trend</h2>
              <p className="section-sub">Revenue over the selected range</p>
            </div>
            <Link className="section-link" to={`/sales${query ? `?${query}` : ""}`}>
              View sales
            </Link>
          </div>
          <div className="chart-wrap">
            {data.trend.length > 0 ? (
              <>
                <div className="chart-meta">
                  <span className="chart-total num">
                    {formatMoney(data.sales.revenue, currency)}
                  </span>
                  <span className="chart-caption">{rangeLabel}</span>
                </div>
                <div className="chart" role="img" aria-label="Sales revenue over time">
                  {data.trend.map((point) => {
                    const height =
                      point.revenue > 0 ? Math.max(4, (point.revenue / maxRevenue) * 100) : 0;
                    return (
                      <div
                        className="chart-col"
                        key={point.date}
                        title={`${shortDate(point.date)} · ${formatMoney(
                          point.revenue,
                          currency,
                        )} · ${point.units} units`}
                      >
                        <div
                          className={point.revenue > 0 ? "chart-bar" : "chart-bar zero"}
                          style={{ height: point.revenue > 0 ? `${height}%` : "2px" }}
                        />
                      </div>
                    );
                  })}
                </div>
                <div className="chart-axis">
                  <span>{shortDate(data.trend[0].date)}</span>
                  <span>{shortDate(data.trend[data.trend.length - 1].date)}</span>
                </div>
              </>
            ) : (
              <div className="empty-state">No sales in this range.</div>
            )}
          </div>
        </section>

        <section className="panel">
          <div className="section-head">
            <div>
              <h2 className="section-title">Top items</h2>
              <p className="section-sub">By revenue in this range</p>
            </div>
            <Link className="section-link" to="/inventory">
              Inventory
            </Link>
          </div>
          {data.topProducts.length > 0 ? (
            <div className="rank-list">
              {data.topProducts.slice(0, 4).map((product, index) => (
                <div className="rank-item" key={product.id}>
                  <span className="rank-index num">{index + 1}</span>
                  <div className="rank-main">
                    <div className="rank-name">{product.name}</div>
                    <div className="rank-sub">
                      {product.category} · {formatNumber(product.units)} sold
                    </div>
                  </div>
                  <div>
                    <div className="rank-value num">
                      {formatMoney(product.revenue, currency)}
                    </div>
                    <div className="rank-sub num">
                      {formatMoney(product.profit, currency)} profit
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="empty-state">No item sales in this range.</div>
          )}
        </section>
      </div>

      <section className="panel panel-spaced">
        <div className="section-head">
          <div>
            <h2 className="section-title">Recent sales</h2>
            <p className="section-sub">Latest activity</p>
          </div>
          <Link className="section-link" to={`/sales${query ? `?${query}` : ""}`}>
            All sales
          </Link>
        </div>
        {data.recentSales.length > 0 ? (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Item</th>
                  <th className="right">Qty</th>
                  <th className="right">Sale</th>
                  <th className="right">Profit</th>
                </tr>
              </thead>
              <tbody>
                {data.recentSales.slice(0, 4).map((sale) => (
                  <tr key={sale.id}>
                    <td>
                      <div className="row-title">{sale.productName}</div>
                      <div className="row-sub">
                        {formatDateTime(sale.soldAt)} ·{" "}
                        {sale.paymentMode === "upi" ? "UPI" : "Cash"}
                      </div>
                    </td>
                    <td className="right num">{sale.quantity}</td>
                    <td className="right num">{formatMoney(sale.totalPrice, currency)}</td>
                    <td className={`right num${sale.profit < 0 ? " negative" : ""}`}>
                      {formatMoney(sale.profit, currency)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="empty-state">No sales recorded yet.</div>
        )}
      </section>
    </>
  );
}
