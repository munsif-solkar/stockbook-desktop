import { useState, type FormEvent } from "react";
import { api, type ItemInput, type ProductOption } from "../api";
import { toMoneyInput } from "../format";
import { Link } from "../router";
import { SubmitButton } from "./SubmitButton";

type EditableItem = {
  id: number;
  name: string;
  sku: string | null;
  category: string;
  purchasePrice: number;
  sellPrice: number;
  lowStockThreshold: number;
};

export function ItemForm({
  mode,
  item,
  categories,
  onSaved,
}: {
  mode: "create" | "edit";
  item?: EditableItem | ProductOption;
  categories: string[];
  onSaved: () => void;
}) {
  const [name, setName] = useState(item?.name ?? "");
  const [sku, setSku] = useState(item?.sku ?? "");
  const [category, setCategory] = useState(item?.category ?? "");
  const [threshold, setThreshold] = useState(
    item ? String(item.lowStockThreshold) : "5",
  );
  const [purchasePrice, setPurchasePrice] = useState(
    item ? toMoneyInput(item.purchasePrice) : "0.00",
  );
  const [sellPrice, setSellPrice] = useState(
    item ? toMoneyInput(item.sellPrice) : "0.00",
  );
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<
    Record<string, string[] | undefined>
  >({});
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError("");
    setFieldErrors({});
    const input: ItemInput = {
      productId: mode === "edit" && item ? item.id : 0,
      name,
      sku,
      category,
      lowStockThreshold: threshold,
      purchasePrice,
      sellPrice,
    };
    try {
      const result =
        mode === "create"
          ? await api.CreateItem(input)
          : await api.UpdateItem(input);
      if (result.error) {
        setError(result.error);
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
          <h2 className="section-title">
            {mode === "create" ? "Add item" : `Edit ${item?.name ?? "item"}`}
          </h2>
          <p className="section-sub">
            {mode === "create"
              ? "Items define your catalog. Add stock from Inventory when it arrives."
              : "Update catalog details. Stock is changed only through Inventory."}
          </p>
        </div>
        <Link className="section-link" to="/items">
          Cancel
        </Link>
      </div>

      <div className="panel-body">
        {error ? (
          <div className="form-error" role="alert">
            {error}
          </div>
        ) : null}

        <div className="form-grid">
          <div className="field full">
            <label htmlFor="item-name">Item name</label>
            <input
              className="input"
              id="item-name"
              name="name"
              type="text"
              maxLength={100}
              value={name}
              onChange={(event) => setName(event.target.value)}
              aria-invalid={Boolean(fieldErrors.name)}
              required
            />
            {fieldErrors.name ? (
              <span className="error-text">{fieldErrors.name[0]}</span>
            ) : null}
          </div>

          <div className="field">
            <label htmlFor="item-sku">SKU (optional)</label>
            <input
              className="input"
              id="item-sku"
              name="sku"
              type="text"
              maxLength={40}
              value={sku}
              onChange={(event) => setSku(event.target.value)}
              aria-invalid={Boolean(fieldErrors.sku)}
              placeholder="e.g. TEA-001"
            />
            {fieldErrors.sku ? (
              <span className="error-text">{fieldErrors.sku[0]}</span>
            ) : null}
          </div>

          <div className="field">
            <label htmlFor="item-category">Category</label>
            <input
              className="input"
              id="item-category"
              name="category"
              type="text"
              maxLength={50}
              list="item-category-options"
              value={category}
              onChange={(event) => setCategory(event.target.value)}
              aria-invalid={Boolean(fieldErrors.category)}
              placeholder="e.g. Beverages"
              required
            />
            <datalist id="item-category-options">
              {categories.map((value) => (
                <option key={value} value={value} />
              ))}
            </datalist>
            {fieldErrors.category ? (
              <span className="error-text">{fieldErrors.category[0]}</span>
            ) : null}
          </div>

          <div className="field">
            <label htmlFor="item-threshold">Low-stock alert at</label>
            <input
              className="input"
              id="item-threshold"
              name="lowStockThreshold"
              type="number"
              min="0"
              step="1"
              inputMode="numeric"
              value={threshold}
              onChange={(event) => setThreshold(event.target.value)}
              aria-invalid={Boolean(fieldErrors.lowStockThreshold)}
              required
            />
            {fieldErrors.lowStockThreshold ? (
              <span className="error-text">{fieldErrors.lowStockThreshold[0]}</span>
            ) : (
              <span className="hint">Inventory flags this item below this level.</span>
            )}
          </div>

          <div className="field">
            <label htmlFor="item-purchase-price">Default purchase price</label>
            <input
              className="input"
              id="item-purchase-price"
              name="purchasePrice"
              type="text"
              inputMode="decimal"
              placeholder="0.00"
              value={purchasePrice}
              onChange={(event) => setPurchasePrice(event.target.value)}
              aria-invalid={Boolean(fieldErrors.purchasePrice)}
              required
            />
            {fieldErrors.purchasePrice ? (
              <span className="error-text">{fieldErrors.purchasePrice[0]}</span>
            ) : (
              <span className="hint">Pre-fills the cost when adding stock.</span>
            )}
          </div>

          <div className="field">
            <label htmlFor="item-sell-price">Default selling price</label>
            <input
              className="input"
              id="item-sell-price"
              name="sellPrice"
              type="text"
              inputMode="decimal"
              placeholder="0.00"
              value={sellPrice}
              onChange={(event) => setSellPrice(event.target.value)}
              aria-invalid={Boolean(fieldErrors.sellPrice)}
              required
            />
            {fieldErrors.sellPrice ? (
              <span className="error-text">{fieldErrors.sellPrice[0]}</span>
            ) : (
              <span className="hint">Pre-fills the price when selling.</span>
            )}
          </div>
        </div>

        <div className="form-actions">
          <SubmitButton
            pending={pending}
            pendingLabel={mode === "create" ? "Adding…" : "Saving…"}
          >
            {mode === "create" ? "Add item" : "Save changes"}
          </SubmitButton>
          <Link className="btn btn-secondary" to="/items">
            Cancel
          </Link>
        </div>
      </div>
    </form>
  );
}
