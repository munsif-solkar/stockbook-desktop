import { useRef, useState, type FormEvent } from "react";
import { api, type ProductOption } from "../api";
import { formatMoney, toMoneyInput } from "../format";
import { findBestItemMatch, matchesItemSearch } from "../itemSearch";
import { Link } from "../router";
import { SubmitButton } from "./SubmitButton";

export function NewSellButton({
  currency,
  onChanged,
}: {
  currency: string;
  onChanged: () => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const searchRef = useRef("");
  const [products, setProducts] = useState<ProductOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [productId, setProductId] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [unitPrice, setUnitPrice] = useState("");
  const [paymentMode, setPaymentMode] = useState("cash");
  const [note, setNote] = useState("");
  const [search, setSearch] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [dismissedSuccess, setDismissedSuccess] = useState(true);
  const [fieldErrors, setFieldErrors] = useState<
    Record<string, string[] | undefined>
  >({});
  const [pending, setPending] = useState(false);

  const selected = products.find((product) => String(product.id) === productId);
  const matchingItems = products.filter((product) =>
    matchesItemSearch(product, search),
  );
  const visibleItems =
    selected && !matchingItems.some((product) => product.id === selected.id)
      ? [selected, ...matchingItems]
      : matchingItems;
  const parsedQuantity = Number(quantity);
  const parsedPrice = Number(unitPrice);
  const total =
    Number.isFinite(parsedQuantity) && Number.isFinite(parsedPrice)
      ? Math.round(parsedQuantity * parsedPrice * 100)
      : 0;
  const overStock =
    selected && Number.isFinite(parsedQuantity) && parsedQuantity > selected.quantity;

  async function loadProducts() {
    setLoading(true);
    setLoadError("");
    try {
      const data = await api.ProductOptions();
      setProducts(data);

      const term = searchRef.current.trim();
      if (term) {
        const matches = data.filter((product) => matchesItemSearch(product, term));
        const inStock = matches.filter((product) => product.quantity > 0);
        const match = findBestItemMatch(inStock.length > 0 ? inStock : matches, term);
        if (match) {
          setProductId(String(match.id));
          setUnitPrice(toMoneyInput(match.sellPrice));
        } else {
          setProductId("");
          setUnitPrice("");
        }
        return;
      }

      const current = data.find(
        (product) => String(product.id) === productId && product.quantity > 0,
      );
      const nextProduct = current ?? data.find((product) => product.quantity > 0);
      if (nextProduct) {
        if (!current) {
          setProductId(String(nextProduct.id));
          setUnitPrice(toMoneyInput(nextProduct.sellPrice));
        }
      } else {
        setProductId("");
        setUnitPrice("");
      }
    } catch (loadFailure) {
      setLoadError(
        loadFailure instanceof Error
          ? loadFailure.message
          : "Could not load items. Check your connection and try again.",
      );
    } finally {
      setLoading(false);
    }
  }

  function openDialog() {
    searchRef.current = "";
    setSearch("");
    setPaymentMode("cash");
    setError("");
    setSuccess("");
    setDismissedSuccess(true);
    setFieldErrors({});
    dialogRef.current?.showModal();
    void loadProducts();
  }

  function closeDialog() {
    setDismissedSuccess(true);
    dialogRef.current?.close();
  }

  function chooseProduct(value: string) {
    setProductId(value);
    const product = products.find((item) => String(item.id) === value);
    if (product) {
      setUnitPrice(toMoneyInput(product.sellPrice));
      setQuantity((current) => {
        const requested = Number(current);
        if (!Number.isFinite(requested) || requested < 1) return "1";
        return String(Math.min(requested, Math.max(1, product.quantity)));
      });
    }
  }

  function handleSearch(value: string) {
    searchRef.current = value;
    setSearch(value);
    if (!value.trim()) return;

    const matches = products.filter((product) => matchesItemSearch(product, value));
    const inStock = matches.filter((product) => product.quantity > 0);
    const match = findBestItemMatch(inStock.length > 0 ? inStock : matches, value);

    if (match) {
      chooseProduct(String(match.id));
    } else {
      setProductId("");
      setUnitPrice("");
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError("");
    setFieldErrors({});
    try {
      const result = await api.CreateSale({
        productId,
        quantity,
        unitPrice,
        paymentMode,
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
      setDismissedSuccess(false);
      setQuantity("1");
      setPaymentMode("cash");
      setNote("");
      searchRef.current = "";
      setSearch("");
      onChanged();
      await loadProducts();
    } catch (submitError) {
      setError(
        submitError instanceof Error ? submitError.message : String(submitError),
      );
    } finally {
      setPending(false);
    }
  }

  const showSuccess = Boolean(success && !dismissedSuccess);
  const disabled =
    !selected ||
    selected.quantity === 0 ||
    !quantity ||
    !unitPrice ||
    Boolean(overStock) ||
    parsedQuantity < 1;

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
        New sell
      </button>

      <dialog
        ref={dialogRef}
        aria-labelledby="new-sell-title"
        onClick={(event) => {
          if (event.target === event.currentTarget) closeDialog();
        }}
      >
        <div className="dialog-card">
          <div className="dialog-head">
            <div>
              <h2 className="dialog-title" id="new-sell-title">
                New sale
              </h2>
              <p className="dialog-sub">Record a sale and update stock instantly.</p>
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
              {error && !showSuccess ? (
                <div className="form-error" role="alert">
                  {error}
                </div>
              ) : null}
              {showSuccess ? (
                <div className="notice" role="status">
                  <span>{success}</span>
                </div>
              ) : null}
              {loadError ? (
                <div className="form-error" role="alert">
                  {loadError}
                </div>
              ) : null}

              {products.length === 0 && !loading ? (
                <div className="empty-state" style={{ padding: "10px 0 16px" }}>
                  No items yet.{" "}
                  <Link to="/items?new=1" onClick={closeDialog}>
                    Add your first item
                  </Link>
                  .
                </div>
              ) : null}

              <div className="field">
                <label htmlFor="sell-search">Find an item</label>
                <input
                  className="input"
                  id="sell-search"
                  type="search"
                  placeholder="Search name, SKU, or category…"
                  value={search}
                  onChange={(event) => handleSearch(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter") event.preventDefault();
                  }}
                  autoComplete="off"
                  aria-controls="sell-product"
                />
                <select
                  className="select"
                  id="sell-product"
                  name="productId"
                  value={productId}
                  onChange={(event) => chooseProduct(event.target.value)}
                  disabled={loading || products.length === 0}
                  aria-label="Item"
                  required
                >
                  <option value="" disabled>
                    {loading
                      ? "Loading items…"
                      : search.trim() && matchingItems.length === 0
                        ? "No matching items"
                        : "Select an item"}
                  </option>
                  {visibleItems.map((product) => (
                    <option
                      key={product.id}
                      value={product.id}
                      disabled={product.quantity === 0}
                    >
                      {product.name}
                      {product.sku ? ` · ${product.sku}` : ""} — {product.quantity} in
                      stock — {formatMoney(product.sellPrice, currency)}
                    </option>
                  ))}
                </select>
                {fieldErrors.productId ? (
                  <span className="error-text">{fieldErrors.productId[0]}</span>
                ) : selected ? (
                  <span className={overStock ? "stock-hint warn" : "stock-hint"}>
                    {selected.quantity === 0
                      ? "Out of stock."
                      : `${selected.quantity} available · ${selected.category}`}
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
                  <label htmlFor="sell-quantity">Quantity</label>
                  <input
                    className="input"
                    id="sell-quantity"
                    name="quantity"
                    type="number"
                    min="1"
                    max={selected?.quantity ?? undefined}
                    step="1"
                    inputMode="numeric"
                    value={quantity}
                    onChange={(event) => setQuantity(event.target.value)}
                    required
                  />
                  {fieldErrors.quantity ? (
                    <span className="error-text">{fieldErrors.quantity[0]}</span>
                  ) : null}
                </div>
                <div className="field">
                  <label htmlFor="sell-price">Selling price (each)</label>
                  <input
                    className="input"
                    id="sell-price"
                    name="unitPrice"
                    type="text"
                    inputMode="decimal"
                    placeholder="0.00"
                    value={unitPrice}
                    onChange={(event) => setUnitPrice(event.target.value)}
                    required
                  />
                  {fieldErrors.unitPrice ? (
                    <span className="error-text">{fieldErrors.unitPrice[0]}</span>
                  ) : (
                    <span className="hint">Editable — discount allowed.</span>
                  )}
                </div>
              </div>

              <div className="sale-summary">
                <span className="muted">
                  {selected ? selected.name : "No product selected"}
                  {parsedQuantity > 0 ? ` × ${parsedQuantity}` : ""}
                </span>
                <strong className="num">{formatMoney(total, currency)}</strong>
              </div>

              <div className="form-grid">
                <div className="field">
                  <label htmlFor="sell-payment">Payment mode</label>
                  <select
                    className="select"
                    id="sell-payment"
                    name="paymentMode"
                    value={paymentMode}
                    onChange={(event) => setPaymentMode(event.target.value)}
                    required
                  >
                    <option value="cash">Cash</option>
                    <option value="upi">UPI</option>
                  </select>
                  {fieldErrors.paymentMode ? (
                    <span className="error-text">{fieldErrors.paymentMode[0]}</span>
                  ) : null}
                </div>
                <div className="field">
                  <label htmlFor="sell-note">Note (optional)</label>
                  <input
                    className="input"
                    id="sell-note"
                    name="note"
                    type="text"
                    maxLength={240}
                    value={note}
                    onChange={(event) => setNote(event.target.value)}
                    placeholder="e.g. Walk-in customer"
                  />
                </div>
              </div>
            </div>

            <div className="dialog-foot">
              <button className="btn btn-secondary" type="button" onClick={closeDialog}>
                Close
              </button>
              <SubmitButton
                pending={pending}
                pendingLabel="Recording…"
                disabled={disabled}
              >
                Record sale
              </SubmitButton>
            </div>
          </form>
        </div>
      </dialog>
    </>
  );
}
