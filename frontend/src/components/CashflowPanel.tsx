import { formatMoney, formatNumber, formatPercent } from "../format";

export function CashflowPanel({
  cash,
  upi,
  cashCount,
  upiCount,
  currency,
}: {
  cash: number;
  upi: number;
  cashCount: number;
  upiCount: number;
  currency: string;
}) {
  const total = cash + upi;
  const cashShare = total > 0 ? (cash / total) * 100 : 0;
  const upiShare = total > 0 ? 100 - cashShare : 0;

  const saleLabel = (count: number) =>
    `${formatNumber(count)} sale${count === 1 ? "" : "s"}`;

  return (
    <section className="panel cashflow-panel" aria-labelledby="cashflow-title">
      <div className="section-head">
        <div>
          <h2 className="section-title" id="cashflow-title">
            Cashflow
          </h2>
          <p className="section-sub">Cash and UPI collected in this range</p>
        </div>
        <span className="badge badge-accent">
          {formatMoney(total, currency)} collected
        </span>
      </div>
      <div className="cashflow-body">
        <div className="cashflow-grid">
          <div className="cashflow-tile">
            <span className="pay-tag pay-cash">
              <span className="pay-swatch" aria-hidden="true" />
              Cash
            </span>
            <span className="cashflow-value num">{formatMoney(cash, currency)}</span>
            <span className="cashflow-meta">
              {saleLabel(cashCount)} · {formatPercent(cashShare, 0)} of revenue
            </span>
          </div>
          <div className="cashflow-tile">
            <span className="pay-tag pay-upi">
              <span className="pay-swatch" aria-hidden="true" />
              UPI
            </span>
            <span className="cashflow-value num">{formatMoney(upi, currency)}</span>
            <span className="cashflow-meta">
              {saleLabel(upiCount)} · {formatPercent(upiShare, 0)} of revenue
            </span>
          </div>
        </div>
        <div
          className={total > 0 ? "cashflow-bar" : "cashflow-bar empty"}
          aria-hidden="true"
        >
          {total > 0 ? (
            <>
              <span className="pay-cash" style={{ width: `${cashShare}%` }} />
              <span className="pay-upi" style={{ width: `${upiShare}%` }} />
            </>
          ) : null}
        </div>
      </div>
    </section>
  );
}
