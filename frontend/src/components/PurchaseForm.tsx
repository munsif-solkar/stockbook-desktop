import { useState, type FormEvent } from "react";
import { api, type ProductOption } from "../api";
import { formatMoney, toMoneyInput } from "../format";
import { Link } from "../router";
import { SubmitButton } from "./SubmitButton";

export function PurchaseForm({
  products,
  currency,
  today,
  onSaved,
}: {
  products: ProductOption[];
  currency: string;
  today: string;
  onSaved: () => void;
}) {
  const [productId, setProductId] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [unitCost, setUnitCost] = useState("");
  const [purchasedAt, setPurchasedAt] = useState(today);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<
    Record<string, string[] | undefined>
  >({});
  const [pending, setPending] = useState(false);

  const selected = products.find((product) => String(product.id) === productId);
  const qty = Number(quantity);
  const cost = Number(unitCost);
  const total =
    Number.isFinite(qty) && Number.isFinite(cost) ? Math.round(qty * cost * 100) : 0;

  function chooseProduct(value: string) {
    setProductId(value);
    const product = products.find((item) => String(item.id) === value);
    if (product) setUnitCost(toMoneyInput(product.purchasePrice));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError("");
    setFieldErrors({});
    try {
      const result = await api.CreatePurchase({
        productId,
        quantity,
        unitCost,
        purchasedAt,
        note,
      });
      if (result.error) {
        setError(result.error);
        if (result.fieldErrors) setFieldErrors(result.fieldErrors);
        return;
      }
      if (result.fieldErrors) {
        setFieldErrors(result.fieldErrors);
        return;
      }
      onSaved();
    } catch (submitError) {
      setError(
        submitError instanceof Error ? submitError.message : String(submitError),
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="panel form-panel" onSubmit={submit} noValidate>
      <div className="section-head">
        <div>
          <h2 className="section-title">Record a purchase</h2>
          <p className="section-sub">
            Stock increases instantly and the latest unit cost is used for profit.
          </p>
        </div>
        <Link className="section-link" to="/purchase">
          Cancel
        </Link>
      </div>

      <div className="panel-body">
        {error ? (
          <div className="form-error" role="alert">
            {error}
          </div>
        ) : null}

        {products.length === 0 ? (
          <div className="empty-state" style={{ padding: "10px 0 18px" }}>
            Add an item first. <Link to="/items?new=1">Go to items</Link>.
          </div>
        ) : (
          <>
            <div className="form-grid">
              <div className="field full">
                <label htmlFor="purchase-product">Item</label>
                <select
                  className="select"
                  id="purchase-product"
                  name="productId"
                  value={productId}
                  onChange={(event) => chooseProduct(event.target.value)}
                  aria-invalid={Boolean(fieldErrors.productId)}
                  required
                >
                  <option value="" disabled>
                    Select an item
                  </option>
                  {products.map((product) => (
                    <option key={product.id} value={product.id}>
                      {product.name}
                      {product.sku ? ` · ${product.sku}` : ""} — {product.quantity} in
                      stock
                    </option>
                  ))}
                </select>
                {fieldErrors.productId ? (
                  <span className="error-text">{fieldErrors.productId[0]}</span>
                ) : selected ? (
                  <span className="hint">
                    Currently {selected.quantity} in stock · last cost{" "}
                    {formatMoney(selected.purchasePrice, currency)}
                  </span>
                ) : null}
              </div>

              <div className="field">
                <label htmlFor="purchase-quantity">Quantity received</label>
                <input
                  className="input"
                  id="purchase-quantity"
                  name="quantity"
                  type="number"
                  min="1"
                  step="1"
                  inputMode="numeric"
                  value={quantity}
                  onChange={(event) => setQuantity(event.target.value)}
                  aria-invalid={Boolean(fieldErrors.quantity)}
                  required
                />
                {fieldErrors.quantity ? (
                  <span className="error-text">{fieldErrors.quantity[0]}</span>
                ) : null}
              </div>

              <div className="field">
                <label htmlFor="purchase-cost">Unit cost</label>
                <input
                  className="input"
                  id="purchase-cost"
                  name="unitCost"
                  type="text"
                  inputMode="decimal"
                  placeholder="0.00"
                  value={unitCost}
                  onChange={(event) => setUnitCost(event.target.value)}
                  aria-invalid={Boolean(fieldErrors.unitCost)}
                  required
                />
                {fieldErrors.unitCost ? (
                  <span className="error-text">{fieldErrors.unitCost[0]}</span>
                ) : null}
              </div>

              <div className="field">
                <label htmlFor="purchase-date">Purchase date</label>
                <input
                  className="input"
                  id="purchase-date"
                  name="purchasedAt"
                  type="date"
                  value={purchasedAt}
                  onChange={(event) => setPurchasedAt(event.target.value)}
                  aria-invalid={Boolean(fieldErrors.purchasedAt)}
                  required
                />
                {fieldErrors.purchasedAt ? (
                  <span className="error-text">{fieldErrors.purchasedAt[0]}</span>
                ) : null}
              </div>

              <div className="field">
                <label htmlFor="purchase-note">Note (optional)</label>
                <input
                  className="input"
                  id="purchase-note"
                  name="note"
                  type="text"
                  maxLength={240}
                  value={note}
                  onChange={(event) => setNote(event.target.value)}
                  placeholder="e.g. Supplier invoice #204"
                />
              </div>
            </div>

            <div className="sale-summary">
              <span className="muted">
                {selected ? selected.name : "No product selected"}
                {qty > 0 ? ` × ${qty}` : ""}
              </span>
              <strong className="num">{formatMoney(total, currency)}</strong>
            </div>

            <div className="form-actions">
              <SubmitButton pending={pending} pendingLabel="Recording…">
                Record purchase
              </SubmitButton>
              <Link className="btn btn-secondary" to="/purchase">
                Cancel
              </Link>
            </div>
          </>
        )}
      </div>
    </form>
  );
}
