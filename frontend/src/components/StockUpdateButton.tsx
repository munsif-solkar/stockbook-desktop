import { useEffect, useRef, useState, type FormEvent } from "react";
import { api, type ProductOption } from "../api";
import { formatMoney, toMoneyInput } from "../format";
import { findBestItemMatch, matchesItemSearch } from "../itemSearch";
import { Link } from "../router";
import { SubmitButton } from "./SubmitButton";

export function StockUpdateButton({
  items,
  currency,
  today,
  autoOpen = false,
  defaultItemId,
  onChanged,
  onAutoOpenHandled,
}: {
  items: ProductOption[];
  currency: string;
  today: string;
  autoOpen?: boolean;
  defaultItemId?: number | null;
  onChanged: () => void;
  onAutoOpenHandled: () => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const initialItem = items.find((item) => item.id === defaultItemId);
  const [itemId, setItemId] = useState(initialItem ? String(initialItem.id) : "");
  const [quantity, setQuantity] = useState("1");
  const [unitCost, setUnitCost] = useState(
    initialItem ? toMoneyInput(initialItem.purchasePrice) : "",
  );
  const [purchasedAt, setPurchasedAt] = useState(today);
  const [note, setNote] = useState("");
  const [search, setSearch] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [fieldErrors, setFieldErrors] = useState<
    Record<string, string[] | undefined>
  >({});
  const [pending, setPending] = useState(false);

  const selected = items.find((item) => String(item.id) === itemId);
  const matchingItems = items.filter((item) => matchesItemSearch(item, search));
  const visibleItems =
    selected && !matchingItems.some((item) => item.id === selected.id)
      ? [selected, ...matchingItems]
      : matchingItems;
  const parsedQuantity = Number(quantity);
  const parsedCost = Number(unitCost);
  const validQuantity =
    Number.isInteger(parsedQuantity) && parsedQuantity > 0 && parsedQuantity <= 1_000_000;
  const total =
    validQuantity && Number.isFinite(parsedCost)
      ? Math.round(parsedQuantity * parsedCost * 100)
      : 0;
  const resultingStock =
    selected && validQuantity ? selected.quantity + parsedQuantity : null;

  function openDialog() {
    setError("");
    setSuccess("");
    setFieldErrors({});
    setSearch("");
    dialogRef.current?.showModal();
  }

  function closeDialog() {
    dialogRef.current?.close();
  }

  function chooseItem(value: string) {
    setItemId(value);
    const item = items.find((entry) => String(entry.id) === value);
    if (item) setUnitCost(toMoneyInput(item.purchasePrice));
  }

  function handleSearch(value: string) {
    setSearch(value);
    if (!value.trim()) return;

    const match = findBestItemMatch(items, value);
    if (match) {
      chooseItem(String(match.id));
    } else {
      setItemId("");
      setUnitCost("");
    }
  }

  useEffect(() => {
    if (!autoOpen) return;
    setSearch("");
    setError("");
    setSuccess("");
    setFieldErrors({});
    const next = items.find((item) => item.id === defaultItemId);
    if (next) {
      setItemId(String(next.id));
      setUnitCost(toMoneyInput(next.purchasePrice));
    }
    if (!dialogRef.current?.open) dialogRef.current?.showModal();
    // Opening is driven by the route; item data refreshes separately.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoOpen, defaultItemId, items.length]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError("");
    setSuccess("");
    setFieldErrors({});
    try {
      const result = await api.CreatePurchase({
        productId: itemId,
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
      if (!result.success) return;
      setSuccess(result.success);
      setQuantity("1");
      setNote("");
      onChanged();
    } catch (submitError) {
      setError(
        submitError instanceof Error ? submitError.message : String(submitError),
      );
    } finally {
      setPending(false);
    }
  }

  const disabled =
    !selected || !validQuantity || !unitCost.trim() || !Number.isFinite(parsedCost);

  return (
    <>
      <button className="btn btn-primary btn-sm" type="button" onClick={openDialog}>
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
        Update stock
      </button>

      <dialog
        ref={dialogRef}
        aria-labelledby="stock-update-title"
        onClose={() => {
          if (autoOpen) onAutoOpenHandled();
        }}
        onClick={(event) => {
          if (event.target === event.currentTarget) closeDialog();
        }}
      >
        <div className="dialog-card">
          <div className="dialog-head">
            <div>
              <h2 className="dialog-title" id="stock-update-title">
                Update stock
              </h2>
              <p className="dialog-sub">
                Add units and record what they cost. Every change is a purchase entry.
              </p>
            </div>
            <button
              className="dialog-close"
              type="button"
              onClick={closeDialog}
              aria-label="Close"
            >
              ×
            </button>
          </div>

          <form onSubmit={submit}>
            <div className="dialog-body">
              {error ? (
                <div className="form-error" role="alert">
                  {error}
                </div>
              ) : null}
              {success ? (
                <div className="notice" role="status">
                  <span>{success}</span>
                </div>
              ) : null}

              {items.length === 0 ? (
                <div className="empty-state" style={{ padding: "10px 0 16px" }}>
                  No items yet.{" "}
                  <Link to="/items?new=1" onClick={closeDialog}>
                    Add your first item
                  </Link>
                  .
                </div>
              ) : null}

              <div className="field">
                <label htmlFor="stock-search">Find an item</label>
                <input
                  className="input"
                  id="stock-search"
                  type="search"
                  placeholder="Search name, SKU, or category…"
                  value={search}
                  onChange={(event) => handleSearch(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter") event.preventDefault();
                  }}
                  autoComplete="off"
                  aria-controls="stock-item"
                />
                <select
                  className="select"
                  id="stock-item"
                  name="productId"
                  value={itemId}
                  onChange={(event) => chooseItem(event.target.value)}
                  disabled={items.length === 0}
                  aria-invalid={Boolean(fieldErrors.productId)}
                  aria-label="Item"
                  required
                >
                  <option value="" disabled>
                    {search.trim() && matchingItems.length === 0
                      ? "No matching items"
                      : "Select an item"}
                  </option>
                  {visibleItems.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name}
                      {item.sku ? ` · ${item.sku}` : ""} — {item.quantity} in stock
                    </option>
                  ))}
                </select>
                {fieldErrors.productId ? (
                  <span className="error-text">{fieldErrors.productId[0]}</span>
                ) : selected ? (
                  <span className="stock-hint">
                    Currently {selected.quantity} in stock
                    {resultingStock !== null ? ` · after this update ${resultingStock}` : ""}
                    {search.trim()
                      ? ` · ${matchingItems.length} match${
                          matchingItems.length === 1 ? "" : "es"
                        }`
                      : ""}
                  </span>
                ) : search.trim() ? (
                  <span className="stock-hint">No items match “{search.trim()}”.</span>
                ) : null}
              </div>

              <div className="form-grid">
                <div className="field">
                  <label htmlFor="stock-quantity">Quantity received</label>
                  <input
                    className="input"
                    id="stock-quantity"
                    name="quantity"
                    type="number"
                    min="1"
                    max="1000000"
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
                  <label htmlFor="stock-cost">Unit cost</label>
                  <input
                    className="input"
                    id="stock-cost"
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
                  ) : (
                    <span className="hint">Defaults to the item price. Editable.</span>
                  )}
                </div>

                <div className="field">
                  <label htmlFor="stock-date">Purchase date</label>
                  <input
                    className="input"
                    id="stock-date"
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
                  <label htmlFor="stock-note">Note (optional)</label>
                  <input
                    className="input"
                    id="stock-note"
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
                  {selected ? selected.name : "No item selected"}
                  {validQuantity ? ` × ${parsedQuantity}` : ""}
                </span>
                <strong className="num">{formatMoney(total, currency)}</strong>
              </div>
            </div>

            <div className="dialog-foot">
              <button className="btn btn-secondary" type="button" onClick={closeDialog}>
                Close
              </button>
              <SubmitButton
                pending={pending}
                pendingLabel="Updating…"
                disabled={disabled}
              >
                Add stock
              </SubmitButton>
            </div>
          </form>
        </div>
      </dialog>
    </>
  );
}
